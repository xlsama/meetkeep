// Package processor turns the recording of a meeting into its record: it
// transcribes the whole of it again, tells the speakers apart and writes
// the file the meeting is played from.
package processor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/xlsama/meetkeep/internal/asr"
	"github.com/xlsama/meetkeep/internal/config"
	"github.com/xlsama/meetkeep/internal/store"
	"github.com/xlsama/meetkeep/internal/trace"
	"github.com/xlsama/meetkeep/modules/speech"
)

type Step string

const (
	// Queued is a meeting that waits for the ones before it: they are
	// processed one at a time.
	Queued       Step = "queued"
	Importing    Step = "import"
	Transcribing Step = "transcribe"
	Separating   Step = "speakers"
	Summarizing  Step = "summary"
)

// Progress is how far the processing of a meeting got.
type Progress struct {
	MeetingID string `json:"meetingId"`
	Step      Step   `json:"step"`
	Percent   int    `json:"percent"`
}

// audioFile is the file a meeting is played from when it has several tracks,
// and the audio of an imported file.
const audioFile = "audio.opus"

// Original names, without its extension, the imported audio file kept
// beside what was made of it, to play the meeting from.
const Original = "original"

// Playable are the kinds of audio a web view plays as they are, with their
// media types.
var Playable = map[string]string{
	".mp3": "audio/mpeg", ".m4a": "audio/mp4", ".aac": "audio/aac", ".wav": "audio/wav",
	".flac": "audio/flac", ".ogg": "audio/ogg", ".opus": "audio/ogg",
}

// original returns the name of the imported audio file kept in directory,
// or "".
func original(directory string) string {
	for extension := range Playable {
		if _, err := os.Stat(filepath.Join(directory, Original+extension)); err == nil {
			return Original + extension
		}
	}
	return ""
}

type Processor struct {
	Speech      speech.Engine
	Store       *store.Store
	Settings    func() config.Settings
	Transcriber func(config.ASRSettings) (asr.Transcriber, error)
	Models      speech.Models
	// Recognition is the file of the model that tells voices apart across
	// meetings. Without it nobody is recognized.
	Recognition string
	// Summarize writes the notes of a meeting whose record is ready and
	// tells what it took. What it fails at does not fail the meeting.
	Summarize func(ctx context.Context, id string) (trace.Detail, error)
	// Polish returns what makes the transcript of a meeting easier to
	// read, to give to the store, and tells what it took. It needs only
	// what was said, so it runs while the speakers are told apart. What it
	// fails at does not fail the meeting.
	Polish func(ctx context.Context, id string, utterances []store.Utterance) ([]store.Utterance, trace.Detail, error)
	// Progress hears how far a meeting got.
	Progress func(Progress)
	// Changed hears that a meeting's record changed.
	Changed func(id string)

	once    sync.Once
	jobs    chan job
	mu      sync.Mutex
	running map[string]*state
}

type job struct {
	id string
	// source is the media file to import first, or "".
	source string
	// speakers alone tells the speakers apart again, count of them.
	speakers bool
	count    int
	record   *trace.Run
}

type state struct {
	progress Progress
	cancel   context.CancelFunc
	canceled bool
}

func (p *Processor) start() {
	p.once.Do(func() {
		p.jobs = make(chan job, 256)
		p.running = map[string]*state{}
		go func() {
			for j := range p.jobs {
				p.run(j)
			}
		}()
	})
}

func (p *Processor) enqueue(j job) {
	p.start()
	kind := "process"
	switch {
	case j.speakers:
		kind = "speakers"
	case j.source != "":
		kind = "import"
	}
	j.record = trace.Begin(p.Store, j.id, kind)
	progress := Progress{MeetingID: j.id, Step: Queued}
	p.mu.Lock()
	p.running[j.id] = &state{progress: progress}
	p.mu.Unlock()
	p.Progress(progress)
	p.jobs <- j
}

// Process queues a meeting whose recording ended.
func (p *Processor) Process(id string) { p.enqueue(job{id: id}) }

// Import queues a meeting made from a media file.
func (p *Processor) Import(id, source string) { p.enqueue(job{id: id, source: source}) }

// Separate queues telling the speakers of a processed meeting apart again,
// count of them, or as many as are found when count is 0.
func (p *Processor) Separate(id string, count int) {
	p.enqueue(job{id: id, speakers: true, count: count})
}

// Cancel gives up processing a meeting.
func (p *Processor) Cancel(id string) {
	p.start()
	p.mu.Lock()
	defer p.mu.Unlock()
	if s := p.running[id]; s != nil {
		s.canceled = true
		if s.cancel != nil {
			s.cancel()
		}
	}
}

// State returns how far a meeting got, if it is queued or being processed.
func (p *Processor) State(id string) (Progress, bool) {
	p.start()
	p.mu.Lock()
	defer p.mu.Unlock()
	s := p.running[id]
	if s == nil {
		return Progress{}, false
	}
	return s.progress, true
}

func (p *Processor) report(id string, step Step, percent int) {
	progress := Progress{MeetingID: id, Step: step, Percent: percent}
	p.mu.Lock()
	if s := p.running[id]; s != nil {
		s.progress = progress
	}
	p.mu.Unlock()
	p.Progress(progress)
}

func (p *Processor) run(j job) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p.mu.Lock()
	s := p.running[j.id]
	canceled := s.canceled
	s.cancel = cancel
	p.mu.Unlock()

	var err error
	if !canceled {
		j.record.Waited()
		switch {
		case j.speakers:
			p.report(j.id, Separating, 60)
			err = p.separate(ctx, j)
		case j.source != "":
			p.report(j.id, Importing, 0)
			err = p.process(ctx, j)
		default:
			p.report(j.id, Transcribing, 0)
			err = p.process(ctx, j)
		}
	}
	p.mu.Lock()
	canceled = s.canceled
	delete(p.running, j.id)
	p.mu.Unlock()
	if canceled {
		return
	}
	status, failure := store.Done, ""
	if err != nil {
		status, failure = store.Failed, err.Error()
	}
	_ = p.Store.SetStatus(context.Background(), j.id, status, failure)
	p.Changed(j.id)
}

// size returns how many bytes the file at path holds, or 0.
func size(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

type track struct {
	name store.Track
	path string
	// sound is what the track holds, once it was opened.
	sound speech.Sound
}

func tracks(m store.Meeting) []track {
	var found []track
	for _, name := range []store.Track{store.Mic, store.System} {
		path := filepath.Join(m.Directory, string(name)+".opus")
		if info, err := os.Stat(path); err == nil && info.Size() > 0 {
			found = append(found, track{name: name, path: path})
		}
	}
	if len(found) == 0 {
		path := filepath.Join(m.Directory, audioFile)
		if _, err := os.Stat(path); err == nil {
			found = append(found, track{name: store.File, path: path})
		}
	}
	return found
}

// several returns the track in which several people may talk: what the
// computer plays when the user's own microphone is another track.
func several(found []track) track {
	for _, t := range found {
		if t.name != store.Mic {
			return t
		}
	}
	return found[0]
}

// beside starts a step that goes on beside the others and returns what
// tells how it ended.
func beside(record *trace.Run, name string, work func() (trace.Detail, error)) <-chan error {
	ended := make(chan error, 1)
	go func() { ended <- record.Step(name, work) }()
	return ended
}

// open reads the sound of each of the tracks, at once.
func (p *Processor) open(ctx context.Context, j job, found []track) error {
	return j.record.Step("open", func() (trace.Detail, error) {
		failures := make([]error, len(found))
		var wait sync.WaitGroup
		for i := range found {
			wait.Go(func() { found[i].sound, failures[i] = p.Speech.Open(ctx, found[i].path) })
		}
		wait.Wait()
		detail := trace.Detail{"tracks": len(found)}
		if err := errors.Join(failures...); err != nil {
			shut(found)
			return detail, fmt.Errorf("读取录音失败：%w", err)
		}
		longest := 0
		for _, t := range found {
			if ms, err := t.sound.Duration(); err == nil {
				longest = max(longest, ms)
			}
		}
		detail["audioMs"] = longest
		return detail, nil
	})
}

func shut(found []track) {
	for _, t := range found {
		if t.sound != nil {
			_ = t.sound.Close()
		}
	}
}

// tell says who speaks when in a sound, count of them, or as many as are
// found when count is 0.
func (p *Processor) tell(ctx context.Context, j job, sound speech.Sound) ([]speech.Segment, error) {
	var segments []speech.Segment
	err := j.record.Step("speakers", func() (trace.Detail, error) {
		var err error
		if segments, err = sound.Separate(ctx, p.Models, j.count); err != nil {
			return nil, fmt.Errorf("识别说话人失败：%w", err)
		}
		told := map[int]bool{}
		for _, s := range segments {
			told[s.Speaker] = true
		}
		return trace.Detail{"segments": len(segments), "speakers": len(told)}, nil
	})
	return segments, err
}

func (p *Processor) process(ctx context.Context, j job) error {
	meeting, err := p.Store.Meeting(ctx, j.id)
	if err != nil {
		return err
	}
	// What goes on beside the steps ends with them.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// written tells how writing the file the meeting is played from ended:
	// it is written while the meeting is transcribed.
	var found []track
	var written <-chan error
	played := filepath.Join(meeting.Directory, audioFile)
	if j.source != "" {
		var sound speech.Sound
		err := j.record.Step("import", func() (trace.Detail, error) {
			detail := trace.Detail{"format": strings.TrimPrefix(strings.ToLower(filepath.Ext(j.source)), "."), "bytes": size(j.source)}
			var err error
			if sound, err = p.Speech.Open(ctx, j.source); err != nil {
				return detail, fmt.Errorf("无法读取这个文件：%w", err)
			}
			ms, err := sound.Duration()
			if err != nil {
				return detail, err
			}
			detail["audioMs"] = ms
			return detail, p.Store.SetDuration(ctx, j.id, int64(ms))
		})
		if err != nil {
			return err
		}
		found = []track{{name: store.File, path: played, sound: sound}}
		defer shut(found)
		p.Changed(j.id)
		written = beside(j.record, "encode", func() (trace.Detail, error) {
			part := played + ".part"
			_, err := p.Speech.Import(ctx, j.source, part)
			if err == nil {
				err = os.Rename(part, played)
			}
			if err != nil {
				_ = os.Remove(part)
				return nil, fmt.Errorf("无法读取这个文件：%w", err)
			}
			return trace.Detail{"bytes": size(played)}, nil
		})
	} else {
		if found = tracks(meeting); len(found) == 0 {
			return errors.New("找不到这场会议的录音")
		}
		if err := p.open(ctx, j, found); err != nil {
			return err
		}
		defer shut(found)
	}
	transcriber, err := p.Transcriber(p.Settings().ASR)
	if err != nil {
		return err
	}

	audio := filepath.Base(found[0].path)
	if kept := original(meeting.Directory); kept != "" && found[0].name == store.File {
		audio = kept
	}
	if len(found) > 1 {
		paths := make([]string, len(found))
		for i, t := range found {
			paths[i] = t.path
		}
		written = beside(j.record, "mix", func() (trace.Detail, error) {
			err := p.Speech.Mix(ctx, paths, played)
			return trace.Detail{"tracks": len(paths), "bytes": size(played)}, err
		})
		audio = audioFile
	}

	// Who speaks when needs only the sound, so it is told while the tracks
	// are transcribed.
	type telling struct {
		segments []speech.Segment
		err      error
	}
	told := make(chan telling, 1)
	go func() {
		segments, err := p.tell(ctx, j, several(found).sound)
		told <- telling{segments, err}
	}()

	p.report(j.id, Transcribing, 5)
	heard := make([][]store.Utterance, len(found))
	transcribed := make(chan error, len(found))
	for i, t := range found {
		go func() {
			transcribed <- j.record.Step("transcribe", func() (trace.Detail, error) {
				detail := trace.Detail{"track": t.name}
				if ms, err := t.sound.Duration(); err == nil {
					detail["audioMs"] = ms
				}
				phrases, err := transcriber.TranscribeSound(ctx, t.sound)
				if err != nil {
					return detail, fmt.Errorf("重新转写失败：%w", err)
				}
				detail["sentences"] = len(phrases)
				for _, phrase := range phrases {
					marks := make([]store.Mark, len(phrase.Marks))
					for i, m := range phrase.Marks {
						marks[i] = store.Mark{At: m.At, Ms: m.Ms}
					}
					heard[i] = append(heard[i], store.Utterance{
						Track: t.name, StartMs: phrase.StartMs, EndMs: phrase.EndMs, Text: phrase.Text, Marks: marks,
					})
				}
				return detail, nil
			})
		}()
	}
	for i := range found {
		if err := <-transcribed; err != nil {
			return err
		}
		p.report(j.id, Transcribing, 5+55*(i+1)/len(found))
	}
	utterances := slices.Concat(heard...)
	sort.SliceStable(utterances, func(a, b int) bool { return utterances[a].StartMs < utterances[b].StartMs })
	for i := range utterances {
		utterances[i].Seq = i + 1
	}

	var polished chan []store.Utterance
	if p.Polish != nil {
		polished = make(chan []store.Utterance, 1)
		said := slices.Clone(utterances)
		go func() {
			var edits []store.Utterance
			_ = j.record.Step("polish", func() (detail trace.Detail, err error) {
				edits, detail, err = p.Polish(ctx, j.id, said)
				return detail, err
			})
			polished <- edits
		}()
	}

	p.report(j.id, Separating, 60)
	speakers, err := p.label(ctx, j, found, utterances, func() ([]speech.Segment, error) {
		t := <-told
		return t.segments, t.err
	})
	if err != nil {
		return err
	}
	if written != nil {
		if err := <-written; err != nil {
			return err
		}
	}
	if err := p.Store.ReplaceTranscript(ctx, j.id, utterances, speakers, audio); err != nil {
		return err
	}
	if polished != nil {
		p.Changed(j.id)
		p.report(j.id, Summarizing, 84)
		if edits := <-polished; edits != nil {
			_ = p.Store.SetPolished(ctx, j.id, edits)
		}
	}
	if p.Summarize != nil {
		p.Changed(j.id)
		p.report(j.id, Summarizing, 90)
		_ = j.record.Step("notes", func() (trace.Detail, error) { return p.Summarize(ctx, j.id) })
	}
	return nil
}

func (p *Processor) separate(ctx context.Context, j job) error {
	meeting, err := p.Store.Meeting(ctx, j.id)
	if err != nil {
		return err
	}
	found := tracks(meeting)
	if len(found) == 0 {
		return errors.New("找不到这场会议的录音")
	}
	utterances, err := p.Store.Utterances(ctx, j.id)
	if err != nil {
		return err
	}
	mixed := several(found)
	for i := range found {
		if found[i].name == mixed.name {
			if err := p.open(ctx, j, found[i:i+1]); err != nil {
				return err
			}
			defer shut(found[i : i+1])
			mixed = found[i]
		}
	}
	speakers, err := p.label(ctx, j, found, utterances, func() ([]speech.Segment, error) {
		return p.tell(ctx, j, mixed.sound)
	})
	if err != nil {
		return err
	}
	return p.Store.ReplaceTranscript(ctx, j.id, utterances, speakers, meeting.Audio)
}

// label sets who says each utterance and returns the speakers: the user,
// when the microphone is a track of its own, and those told apart in the
// other track, as told says who speaks when in it.
func (p *Processor) label(ctx context.Context, j job, found []track, utterances []store.Utterance, told func() ([]speech.Segment, error)) ([]store.Speaker, error) {
	var speakers []store.Speaker
	self := 0
	if len(found) > 1 {
		self = 1
		speakers = append(speakers, store.Speaker{ID: self, Name: "我", Self: true})
	}
	mixed := several(found)
	var phrases []asr.Phrase
	var index []int
	for i, u := range utterances {
		if u.Track != mixed.name {
			utterances[i].Speaker, utterances[i].Uncertain = self, false
			continue
		}
		phrases = append(phrases, asr.Phrase{StartMs: u.StartMs, EndMs: u.EndMs})
		index = append(index, i)
	}
	if len(phrases) == 0 {
		return speakers, nil
	}
	segments, err := told()
	if err != nil {
		return nil, err
	}
	who, uncertain := assign(phrases, segments)
	p.report(j.id, Separating, 80)
	sentences := make([]store.Utterance, len(index))
	for k, i := range index {
		sentences[k] = utterances[i]
	}
	var heard voiced
	var names map[int]string
	var recognizedAs map[int]int
	err = j.record.Step("voices", func() (trace.Detail, error) {
		var err error
		if heard, err = p.listen(ctx, mixed.sound, sentences, who, uncertain); err != nil {
			return nil, fmt.Errorf("识别说话人失败：%w", err)
		}
		known, named, err := p.Store.Voices(ctx)
		if err != nil {
			return nil, err
		}
		people := map[int][]float32{}
		for person, samples := range known {
			people[person] = mean(samples)
		}
		names, recognizedAs = named, match(heard.voices, people)
		return trace.Detail{"spans": heard.spans, "voices": len(heard.voices), "recognized": len(recognizedAs)}, nil
	})
	if err != nil {
		return nil, err
	}

	used := map[int]int{}
	var order []int
	for _, speaker := range who {
		if _, ok := used[speaker]; !ok {
			used[speaker] = 0
			order = append(order, speaker)
		}
	}
	sort.Ints(order)
	for n, speaker := range order {
		used[speaker] = self + n + 1
		name := fmt.Sprintf("说话人 %d", n+1)
		if len(order) == 1 && self != 0 {
			name = "对方"
		}
		sample := heard.samples[speaker]
		found := store.Speaker{
			ID: used[speaker], Name: name, Voice: heard.voices[speaker],
			SampleStartMs: int64(sample.StartMs), SampleEndMs: int64(sample.EndMs),
		}
		if person, ok := recognizedAs[speaker]; ok {
			found.PersonID, found.Name = person, names[person]
		}
		speakers = append(speakers, found)
	}
	for k, i := range index {
		utterances[i].Speaker = used[who[k]]
		utterances[i].Uncertain = uncertain[k] && len(order) > 1
	}
	return speakers, nil
}
