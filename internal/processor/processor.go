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
	"sort"
	"sync"

	"github.com/xlsama/meetkeep/internal/asr"
	"github.com/xlsama/meetkeep/internal/config"
	"github.com/xlsama/meetkeep/internal/store"
	"github.com/xlsama/meetkeep/modules/speech"
)

type Step string

const (
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
	// Summarize writes the notes of a meeting whose record is ready. What
	// it fails at does not fail the meeting.
	Summarize func(ctx context.Context, id string)
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
	p.mu.Lock()
	p.running[j.id] = &state{progress: Progress{MeetingID: j.id, Step: Transcribing}}
	p.mu.Unlock()
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
		if j.speakers {
			err = p.separate(ctx, j)
		} else {
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

type track struct {
	name store.Track
	path string
}

func tracks(m store.Meeting) []track {
	var found []track
	for _, name := range []store.Track{store.Mic, store.System} {
		path := filepath.Join(m.Directory, string(name)+".opus")
		if info, err := os.Stat(path); err == nil && info.Size() > 0 {
			found = append(found, track{name, path})
		}
	}
	if len(found) == 0 {
		path := filepath.Join(m.Directory, audioFile)
		if _, err := os.Stat(path); err == nil {
			found = append(found, track{store.File, path})
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

func (p *Processor) process(ctx context.Context, j job) error {
	meeting, err := p.Store.Meeting(ctx, j.id)
	if err != nil {
		return err
	}
	if j.source != "" {
		p.report(j.id, Importing, 0)
		ms, err := p.Speech.Import(ctx, j.source, filepath.Join(meeting.Directory, audioFile))
		if err != nil {
			return fmt.Errorf("读不了这个文件：%w", err)
		}
		if err := p.Store.SetDuration(ctx, j.id, int64(ms)); err != nil {
			return err
		}
		p.Changed(j.id)
	}
	found := tracks(meeting)
	if len(found) == 0 {
		return errors.New("找不到这场会议的录音")
	}
	transcriber, err := p.Transcriber(p.Settings().ASR)
	if err != nil {
		return err
	}

	p.report(j.id, Transcribing, 5)
	var utterances []store.Utterance
	for i, t := range found {
		phrases, err := transcriber.TranscribeFile(ctx, t.path)
		if err != nil {
			return fmt.Errorf("重新转写失败：%w", err)
		}
		for _, phrase := range phrases {
			utterances = append(utterances, store.Utterance{
				Track: t.name, StartMs: phrase.StartMs, EndMs: phrase.EndMs, Text: phrase.Text,
			})
		}
		p.report(j.id, Transcribing, 5+55*(i+1)/len(found))
	}
	sort.SliceStable(utterances, func(a, b int) bool { return utterances[a].StartMs < utterances[b].StartMs })
	for i := range utterances {
		utterances[i].Seq = i + 1
	}

	p.report(j.id, Separating, 60)
	audio := filepath.Base(found[0].path)
	if kept := original(meeting.Directory); kept != "" && found[0].name == store.File {
		audio = kept
	}
	if len(found) > 1 {
		paths := make([]string, len(found))
		for i, t := range found {
			paths[i] = t.path
		}
		if err := p.Speech.Mix(ctx, paths, filepath.Join(meeting.Directory, audioFile)); err != nil {
			return err
		}
		audio = audioFile
	}
	p.report(j.id, Separating, 70)
	speakers, err := p.label(ctx, j.id, found, utterances, 0)
	if err != nil {
		return err
	}
	if err := p.Store.ReplaceTranscript(ctx, j.id, utterances, speakers, audio); err != nil {
		return err
	}
	if p.Summarize != nil {
		p.Changed(j.id)
		p.report(j.id, Summarizing, 90)
		p.Summarize(ctx, j.id)
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
	p.report(j.id, Separating, 60)
	speakers, err := p.label(ctx, j.id, found, utterances, j.count)
	if err != nil {
		return err
	}
	return p.Store.ReplaceTranscript(ctx, j.id, utterances, speakers, meeting.Audio)
}

// label sets who says each utterance and returns the speakers: the user,
// when the microphone is a track of its own, and those told apart in the
// other track.
func (p *Processor) label(ctx context.Context, id string, found []track, utterances []store.Utterance, count int) ([]store.Speaker, error) {
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
	segments, err := p.Speech.Separate(ctx, mixed.path, p.Models, count)
	if err != nil {
		return nil, fmt.Errorf("区分说话人失败：%w", err)
	}
	who, uncertain := assign(phrases, segments)
	p.report(id, Separating, 80)
	sentences := make([]store.Utterance, len(index))
	for k, i := range index {
		sentences[k] = utterances[i]
	}
	heard, err := p.listen(ctx, mixed.path, sentences, who, uncertain)
	if err != nil {
		return nil, fmt.Errorf("区分说话人失败：%w", err)
	}
	known, names, err := p.Store.Voices(ctx)
	if err != nil {
		return nil, err
	}
	people := map[int][]float32{}
	for person, samples := range known {
		people[person] = mean(samples)
	}
	recognizedAs := match(heard.voices, people)

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
