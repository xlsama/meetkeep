package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/egoist/mygo"

	"github.com/xlsama/meetkeep/internal/asr"
	"github.com/xlsama/meetkeep/internal/audio"
	"github.com/xlsama/meetkeep/internal/config"
	"github.com/xlsama/meetkeep/internal/export"
	"github.com/xlsama/meetkeep/internal/llm"
	"github.com/xlsama/meetkeep/internal/notes"
	"github.com/xlsama/meetkeep/internal/processor"
	"github.com/xlsama/meetkeep/internal/recorder"
	"github.com/xlsama/meetkeep/internal/store"
	"github.com/xlsama/meetkeep/modules/capture"
)

// Config reads and changes the settings.
type Config struct{ app *app }

// Get returns the settings.
func (c *Config) Get() config.Settings { return c.app.current() }

// Save stores the settings.
func (c *Config) Save(s config.Settings) error {
	s.ASR.Endpoint = strings.TrimSpace(s.ASR.Endpoint)
	s.ASR.Key = strings.TrimSpace(s.ASR.Key)
	s.LLM.BaseURL = strings.TrimSpace(s.LLM.BaseURL)
	s.LLM.Key = strings.TrimSpace(s.LLM.Key)
	s.LLM.Model = strings.TrimSpace(s.LLM.Model)
	if s.ASR.Hotwords == nil {
		s.ASR.Hotwords = []string{}
	}
	if s.Sources.Apps == nil {
		s.Sources.Apps = []config.AppRef{}
	}
	if s.Recording.IgnoredApps == nil {
		s.Recording.IgnoredApps = []config.AppRef{}
	}
	return c.app.update(func(current *config.Settings) { *current = s })
}

// TestTranscription transcribes a second of silence with the settings.
func (c *Config) TestTranscription(ctx context.Context, s config.ASRSettings) error {
	mai, err := asr.NewMAI(strings.TrimSpace(s.Endpoint), strings.TrimSpace(s.Key), nil)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	_, err = mai.Transcribe(ctx, make([]int16, audio.SampleRate))
	return err
}

// TestModel asks the model for a word with the settings.
func (c *Config) TestModel(ctx context.Context, s config.LLMSettings) error {
	client, err := llm.New(strings.TrimSpace(s.BaseURL), strings.TrimSpace(s.Key), strings.TrimSpace(s.Model))
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	_, err = client.Complete(ctx, []llm.Message{{Role: "user", Content: "ping"}}, 1)
	return err
}

// Path returns where the settings file is, with ~ for the home directory.
func (c *Config) Path() string { return config.Contract(config.Path()) }

// OpenFile opens the settings file in the editor of the system.
func (c *Config) OpenFile() error {
	if _, err := os.Stat(config.Path()); errors.Is(err, os.ErrNotExist) {
		if err := config.Save(c.app.current()); err != nil {
			return err
		}
	}
	return mygo.Shell.OpenPath(config.Path())
}

// ChooseDirectory asks where recordings go and returns the choice, or ""
// when the user canceled.
func (c *Config) ChooseDirectory(ctx context.Context) (string, error) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent:            mygo.CallerWindow(ctx),
		DefaultPath:       config.Expand(c.app.current().Recording.Directory),
		Directory:         true,
		CreateDirectories: true,
	})
	if err != nil || len(paths) == 0 {
		return "", err
	}
	return config.Contract(paths[0]), nil
}

// Sources tells what can be recorded.
type Sources struct{ app *app }

// SourceApp is an app whose sound can be recorded alone.
type SourceApp struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Icon is a PNG, or empty.
	Icon    []byte `json:"icon"`
	Playing bool   `json:"playing"`
}

// Apps lists the apps whose sound can be recorded alone, those that play
// now first.
func (s *Sources) Apps() ([]SourceApp, error) {
	apps, err := s.app.capture.Apps()
	if err != nil {
		return nil, err
	}
	out := make([]SourceApp, len(apps))
	for i, a := range apps {
		out[i] = SourceApp{ID: a.ID, Name: a.Name, Icon: a.Icon, Playing: a.Playing}
	}
	return out, nil
}

// Microphone returns the name of the microphone that is recorded.
func (s *Sources) Microphone() (string, error) { return s.app.capture.Microphone() }

// Meter sends how loud the microphone is, from 0 to 1, ten times a second
// until the page closes levels. It asks for the microphone's permission.
func (s *Sources) Meter(ctx context.Context, levels *mygo.Channel[float64]) error {
	shown := 0.0
	for frame, err := range s.app.capture.Frames(ctx, capture.Microphone, capture.Options{}) {
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		shown = max(audio.Level(audio.Samples(frame)), shown*0.6)
		if levels.Send(shown) != nil {
			return nil
		}
	}
	return nil
}

// Recorder records meetings.
type Recorder struct{ app *app }

// StartOptions say what to record and what to show meanwhile.
type StartOptions struct {
	recorder.Options
	ShowPanel bool `json:"showPanel"`
}

// Start begins a meeting and remembers the choices for the next one.
func (r *Recorder) Start(ctx context.Context, o StartOptions) (store.Meeting, error) {
	if o.Apps == nil {
		o.Apps = []config.AppRef{}
	}
	if err := r.app.update(func(s *config.Settings) {
		s.Sources = config.SourceSettings{Microphone: o.Microphone, System: o.System, Apps: o.Apps}
		s.Recording.ShowPanel = o.ShowPanel
	}); err != nil {
		return store.Meeting{}, err
	}
	return r.app.recorder.Start(ctx, o.Options)
}

// Pause stops taking in audio until Resume.
func (r *Recorder) Pause() error { return r.app.recorder.Pause() }

// Resume goes on after Pause.
func (r *Recorder) Resume() error { return r.app.recorder.Resume() }

// Stop ends the meeting and returns it. The meeting is then processed.
func (r *Recorder) Stop() (store.Meeting, error) {
	meeting, err := r.app.recorder.Stop(45 * time.Second)
	if err == nil {
		r.app.processor.Process(meeting.ID)
	}
	return meeting, err
}

// Snapshot returns what the recorder does now, with the utterances that
// are still being spoken or transcribed.
func (r *Recorder) Snapshot() recorder.Snapshot { return r.app.recorder.Snapshot() }

// Library keeps the meetings.
type Library struct{ app *app }

// MeetingDetail is a meeting with what was said in it and by whom.
type MeetingDetail struct {
	Meeting    store.Meeting     `json:"meeting"`
	Utterances []store.Utterance `json:"utterances"`
	Speakers   []store.Speaker   `json:"speakers"`
	// Progress is how far processing the meeting got, while its status is
	// processing.
	Progress *processor.Progress `json:"progress"`
}

// List returns the meetings, the newest first.
func (l *Library) List(ctx context.Context) ([]store.Meeting, error) {
	return l.app.store.Meetings(ctx)
}

// Get returns a meeting with what was said in it.
func (l *Library) Get(ctx context.Context, id string) (MeetingDetail, error) {
	meeting, err := l.app.store.Meeting(ctx, id)
	if err != nil {
		return MeetingDetail{}, err
	}
	utterances, err := l.app.store.Utterances(ctx, id)
	if err != nil {
		return MeetingDetail{}, err
	}
	speakers, err := l.app.store.Speakers(ctx, id)
	detail := MeetingDetail{Meeting: meeting, Utterances: utterances, Speakers: speakers}
	if progress, ok := l.app.processor.State(id); ok {
		detail.Progress = &progress
	}
	return detail, err
}

// RenameSpeaker gives a speaker of a meeting another name.
func (l *Library) RenameSpeaker(ctx context.Context, id string, speaker int, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("名字不能为空")
	}
	speakers, err := l.app.store.Speakers(ctx, id)
	if err != nil {
		return err
	}
	remembered := false
	for _, s := range speakers {
		remembered = remembered || s.ID == speaker && s.Remembered
	}
	if remembered {
		if err := l.app.store.Forget(ctx, id, speaker); err != nil {
			return err
		}
	}
	if err := l.app.store.RenameSpeaker(ctx, id, speaker, name); err != nil {
		return err
	}
	if remembered {
		if err := l.app.store.Remember(ctx, id, speaker); err != nil {
			return err
		}
	}
	return libraryChanged.Broadcast(id)
}

// RememberSpeaker keeps the voice of a meeting's speaker under the
// speaker's name, so that later meetings recognize it, or forgets the
// sample when remember is false.
func (l *Library) RememberSpeaker(ctx context.Context, id string, speaker int, remember bool) error {
	var err error
	if remember {
		err = l.app.store.Remember(ctx, id, speaker)
	} else {
		err = l.app.store.Forget(ctx, id, speaker)
	}
	if err != nil {
		return err
	}
	return libraryChanged.Broadcast(id)
}

// MergeSpeakers gives what one speaker of a meeting says to another.
func (l *Library) MergeSpeakers(ctx context.Context, id string, from, into int) error {
	if from == into {
		return nil
	}
	if err := l.app.store.MergeSpeakers(ctx, id, from, into); err != nil {
		return err
	}
	return libraryChanged.Broadcast(id)
}

func (l *Library) again(ctx context.Context, id string, queue func()) error {
	if s := l.app.recorder.Status(); s.Meeting != nil && s.Meeting.ID == id {
		return errors.New("这场会议还在录制")
	}
	if _, busy := l.app.processor.State(id); busy {
		return errors.New("这场会议正在整理")
	}
	if err := l.app.store.SetStatus(ctx, id, store.Processing, ""); err != nil {
		return err
	}
	queue()
	return libraryChanged.Broadcast(id)
}

// Reidentify tells the speakers of a meeting apart again: count of them, or
// as many as are found when count is 0.
func (l *Library) Reidentify(ctx context.Context, id string, count int) error {
	return l.again(ctx, id, func() { l.app.processor.Separate(id, max(count, 0)) })
}

// Reprocess transcribes a meeting's recording again and tells its speakers
// apart.
func (l *Library) Reprocess(ctx context.Context, id string) error {
	return l.again(ctx, id, func() { l.app.processor.Process(id) })
}

// Notes returns what is known of a meeting's notes.
func (l *Library) Notes(ctx context.Context, id string) (NotesState, error) {
	return l.app.notes.state(ctx, id)
}

// SaveNotesDoc stores the document the user made of a meeting's notes by
// editing them, and text, its words line by line, to search.
func (l *Library) SaveNotesDoc(ctx context.Context, id, doc, text string) error {
	return l.app.store.SaveNotesDoc(ctx, id, doc, text)
}

// RestoreNotes brings back the notes that writing them again replaced,
// with what the user had made of them.
func (l *Library) RestoreNotes(ctx context.Context, id string) error {
	if err := l.app.store.RestoreNotes(ctx, id); err != nil {
		return err
	}
	l.app.notes.announce(id)
	return nil
}

// RegenerateNotes writes the final notes of a processed meeting again.
// The notes they replace can be restored.
func (l *Library) RegenerateNotes(ctx context.Context, id string) error {
	meeting, err := l.app.store.Meeting(ctx, id)
	if err != nil {
		return err
	}
	if meeting.Status != store.Done {
		return errors.New("这场会议还没有整理好")
	}
	if _, err := l.app.notes.model(); err != nil {
		return err
	}
	go l.app.notes.finalize(context.Background(), id)
	return nil
}

// Board returns the scene of a meeting's whiteboard as SaveBoard took it,
// or "" when it has none.
func (l *Library) Board(ctx context.Context, id string) (string, error) {
	return l.app.store.Board(ctx, id)
}

// SaveBoard stores the scene of a meeting's whiteboard.
func (l *Library) SaveBoard(ctx context.Context, id, scene string) error {
	return l.app.store.SaveBoard(ctx, id, scene)
}

// SearchLine is an utterance that contains what was searched for.
type SearchLine struct {
	Seq     int   `json:"seq"`
	StartMs int64 `json:"startMs"`
	// Speaker is the name of who says it and SpeakerID their ID in the
	// meeting, 0 before the meeting is processed. Self marks the user.
	Speaker   string `json:"speaker"`
	SpeakerID int    `json:"speakerId"`
	Self      bool   `json:"self"`
	Text      string `json:"text"`
}

// SearchNote is a line of a meeting's notes that contains what was
// searched for.
type SearchNote struct {
	Text string `json:"text"`
	AtMs int64  `json:"atMs"`
}

// SearchHit is a meeting whose title, transcript or notes contain what was
// searched for.
type SearchHit struct {
	Meeting    store.Meeting `json:"meeting"`
	TitleMatch bool          `json:"titleMatch"`
	Lines      []SearchLine  `json:"lines"`
	Notes      []SearchNote  `json:"notes"`
}

func matchingNotes(kept store.StoredNotes, query string) []SearchNote {
	query = strings.ToLower(query)
	var found []SearchNote
	if kept.Doc != "" {
		for line := range strings.SplitSeq(kept.Text, "\n") {
			if strings.Contains(strings.ToLower(line), query) {
				found = append(found, SearchNote{Text: strings.TrimSpace(line)})
			}
		}
		return found
	}
	var saved notes.Notes
	if json.Unmarshal([]byte(kept.Data), &saved) != nil {
		return nil
	}
	add := func(text string, at int64) {
		if strings.Contains(strings.ToLower(text), query) {
			found = append(found, SearchNote{Text: text, AtMs: at})
		}
	}
	add(saved.Headline, 0)
	for _, topic := range saved.Topics {
		add(topic.Title, topic.AtMs)
		for _, p := range topic.Points {
			add(p.Text, p.AtMs)
		}
	}
	for _, todo := range saved.Todos {
		add(todo.Text, todo.AtMs)
	}
	for _, p := range saved.Open {
		add(p.Text, p.AtMs)
	}
	return found
}

// Search finds the meetings whose titles or transcripts contain query, the
// newest first.
func (l *Library) Search(ctx context.Context, query string) ([]SearchHit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []SearchHit{}, nil
	}
	titled, err := l.app.store.SearchTitles(ctx, query)
	if err != nil {
		return nil, err
	}
	lines, err := l.app.store.SearchLines(ctx, query)
	if err != nil {
		return nil, err
	}
	noted, err := l.app.store.SearchNotes(ctx, query)
	if err != nil {
		return nil, err
	}
	hits := map[string]*SearchHit{}
	for _, m := range titled {
		hits[m.ID] = &SearchHit{Meeting: m, TitleMatch: true, Lines: []SearchLine{}, Notes: []SearchNote{}}
	}
	hitOf := func(id string) (*SearchHit, error) {
		if hit := hits[id]; hit != nil {
			return hit, nil
		}
		m, err := l.app.store.Meeting(ctx, id)
		if err != nil {
			return nil, err
		}
		hits[id] = &SearchHit{Meeting: m, Lines: []SearchLine{}, Notes: []SearchNote{}}
		return hits[id], nil
	}
	for id, kept := range noted {
		if found := matchingNotes(kept, query); len(found) > 0 {
			hit, err := hitOf(id)
			if err != nil {
				return nil, err
			}
			hit.Notes = found
		}
	}
	for _, line := range lines {
		hit, err := hitOf(line.MeetingID)
		if err != nil {
			return nil, err
		}
		u := line.Utterance
		name := line.Speaker
		if name == "" {
			name = export.Name(u, nil)
		}
		hit.Lines = append(hit.Lines, SearchLine{
			Seq: u.Seq, StartMs: u.StartMs, Speaker: name, SpeakerID: u.Speaker, Self: line.Self, Text: u.Text,
		})
	}
	found := make([]SearchHit, 0, len(hits))
	for _, hit := range hits {
		found = append(found, *hit)
	}
	sort.Slice(found, func(a, b int) bool { return found[a].Meeting.StartedAt > found[b].Meeting.StartedAt })
	return found, nil
}

// Export asks where to save a meeting's transcript in format and returns
// where it went, or "" when the user canceled.
func (l *Library) Export(ctx context.Context, id string, format export.Format) (string, error) {
	detail, err := l.Get(ctx, id)
	if err != nil {
		return "", err
	}
	path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
		Parent:      mygo.CallerWindow(ctx),
		DefaultPath: fileName(detail.Meeting.Title) + format.Extension(),
	})
	if err != nil || path == "" {
		return "", err
	}
	text := export.Render(detail.Meeting, detail.Utterances, detail.Speakers, format)
	return path, os.WriteFile(path, []byte(text), 0o644)
}

// SaveRecording asks where to save a copy of a meeting's recording and
// returns where it went, or "" when the user canceled.
func (l *Library) SaveRecording(ctx context.Context, id string) (string, error) {
	meeting, err := l.app.store.Meeting(ctx, id)
	if err != nil {
		return "", err
	}
	if meeting.Audio == "" {
		return "", errors.New("录音还在整理")
	}
	return mygo.Dialog.SaveCopy(filepath.Join(meeting.Directory, meeting.Audio), fileName(meeting.Title)+filepath.Ext(meeting.Audio))
}

// playable are the kinds of video a web view plays as they are.
var playable = map[string]bool{".mp4": true, ".m4v": true, ".mov": true, ".webm": true}

// keepVideo keeps the file at path in directory when it is a video the app
// can play, and returns the name it has there, or "". A link to the file
// takes no room; where the file system makes none, the file is copied.
func keepVideo(path, directory string) string {
	extension := strings.ToLower(filepath.Ext(path))
	if !playable[extension] {
		return ""
	}
	name := "video" + extension
	kept := filepath.Join(directory, name)
	if os.Link(path, kept) == nil {
		return name
	}
	source, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer source.Close()
	target, err := os.Create(kept)
	if err != nil {
		return ""
	}
	if _, err := io.Copy(target, source); err != nil {
		target.Close()
		os.Remove(kept)
		return ""
	}
	if target.Close() != nil {
		return ""
	}
	return name
}

// keepAudio links the audio file at path into directory as the file the
// meeting is played from, when the app can play it as it is. Where the
// file system makes no link, the meeting is played from what processing
// makes of the file.
func keepAudio(path, directory string) {
	if extension := strings.ToLower(filepath.Ext(path)); processor.Playable[extension] != "" {
		_ = os.Link(path, filepath.Join(directory, processor.Original+extension))
	}
}

var unsafeInFileNames = strings.NewReplacer("/", "-", "\\", "-", ":", "-", "*", "-", "?", "-", "\"", "-", "<", "-", ">", "-", "|", "-")

func fileName(title string) string {
	return strings.TrimSpace(unsafeInFileNames.Replace(title))
}

// Import asks for an audio or video file and makes a meeting of it, which
// is then processed. It returns nil when the user canceled.
func (l *Library) Import(ctx context.Context) (*store.Meeting, error) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent:  mygo.CallerWindow(ctx),
		Filters: []mygo.FileFilter{{Name: "录音或视频", MIMETypes: []string{"audio/*", "video/*"}}},
	})
	if err != nil || len(paths) == 0 {
		return nil, err
	}
	meeting, err := l.ImportFile(ctx, paths[0])
	if err != nil {
		return nil, err
	}
	return &meeting, nil
}

// ImportFile makes a meeting of the audio or video file at path, which is
// then processed.
func (l *Library) ImportFile(ctx context.Context, path string) (store.Meeting, error) {
	info, err := os.Stat(path)
	if err != nil {
		return store.Meeting{}, err
	}
	id := store.NewID(time.Now())
	directory := filepath.Join(config.Expand(l.app.current().Recording.Directory), id)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return store.Meeting{}, err
	}
	base := filepath.Base(path)
	meeting := store.Meeting{
		ID:        id,
		Title:     strings.TrimSuffix(base, filepath.Ext(base)),
		StartedAt: info.ModTime().UnixMilli(),
		Status:    store.Processing,
		Directory: directory,
		Apps:      []config.AppRef{},
		Video:     keepVideo(path, directory),
	}
	keepAudio(path, directory)
	if err := l.app.store.CreateMeeting(ctx, meeting); err != nil {
		return store.Meeting{}, err
	}
	l.app.processor.Import(id, path)
	return meeting, libraryChanged.Broadcast(id)
}

// Rename gives a meeting another title.
func (l *Library) Rename(ctx context.Context, id, title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return errors.New("标题不能为空")
	}
	if err := l.app.store.Rename(ctx, id, title); err != nil {
		return err
	}
	l.app.recorder.Rename(id, title)
	return libraryChanged.Broadcast(id)
}

// Delete removes a meeting and moves its recordings to the trash.
func (l *Library) Delete(ctx context.Context, id string) error {
	if s := l.app.recorder.Status(); s.Meeting != nil && s.Meeting.ID == id {
		return errors.New("这场会议还在录制")
	}
	meeting, err := l.app.store.Meeting(ctx, id)
	if err != nil {
		return err
	}
	l.app.processor.Cancel(id)
	if err := l.app.store.Delete(ctx, id); err != nil {
		return err
	}
	if _, err := os.Stat(meeting.Directory); err == nil {
		_ = mygo.Shell.TrashItem(meeting.Directory)
	}
	return libraryChanged.Broadcast(id)
}

// Reveal shows a meeting's recordings in the file manager.
func (l *Library) Reveal(ctx context.Context, id string) error {
	meeting, err := l.app.store.Meeting(ctx, id)
	if err != nil {
		return err
	}
	mygo.Shell.ShowItemInFolder(meeting.Directory)
	return nil
}

// People keeps the voices that are remembered.
type People struct{ app *app }

// Known are the people whose voices are remembered, and the user.
type Known struct {
	// SelfMeetings is in how many meetings the user's microphone was
	// recorded, SelfLastAt when the newest of them began, in Unix
	// milliseconds.
	SelfMeetings int            `json:"selfMeetings"`
	SelfLastAt   int64          `json:"selfLastAt"`
	People       []store.Person `json:"people"`
}

// List returns the people whose voices are remembered.
func (p *People) List(ctx context.Context) (Known, error) {
	people, err := p.app.store.People(ctx)
	if err != nil {
		return Known{}, err
	}
	meetings, lastAt, err := p.app.store.Self(ctx)
	return Known{SelfMeetings: meetings, SelfLastAt: lastAt, People: people}, err
}

// Rename gives a person another name, in every meeting they talk in.
func (p *People) Rename(ctx context.Context, id int, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("名字不能为空")
	}
	if err := p.app.store.RenamePerson(ctx, id, name); err != nil {
		return err
	}
	return libraryChanged.Broadcast("")
}

// Delete forgets a person's voice. The meetings they talk in keep their
// name.
func (p *People) Delete(ctx context.Context, id int) error {
	if err := p.app.store.DeletePerson(ctx, id); err != nil {
		return err
	}
	return libraryChanged.Broadcast("")
}

// Windows shows and hides the app's windows.
type Windows struct{ app *app }

// ShowMain brings the main window to the front, at route unless it is "".
func (w *Windows) ShowMain(route string) {
	w.app.windows.closeMenu()
	w.app.windows.showMain(route)
}

// ShowPanel shows the side panel of the meeting being recorded.
func (w *Windows) ShowPanel() {
	w.app.windows.closeMenu()
	w.app.windows.setMode(panelOpen)
}

// CollapsePanel shrinks the side panel to a strip.
func (w *Windows) CollapsePanel() { w.app.windows.setMode(panelMini) }

// HidePanel hides the side panel and its strip.
func (w *Windows) HidePanel() { w.app.windows.setMode(panelHidden) }

// ResizePanel makes the side panel width wide.
func (w *Windows) ResizePanel(width int) { w.app.windows.resizePanel(width) }

// CloseNotice closes the notice that a call began.
func (w *Windows) CloseNotice() { w.app.windows.closeNotice() }

// IgnoreApp stops announcing the calls of an app, and closes the notice
// of one.
func (w *Windows) IgnoreApp(app config.AppRef) error {
	w.app.windows.closeNotice()
	return w.app.update(func(s *config.Settings) {
		if !ignored(*s, app.ID) {
			s.Recording.IgnoredApps = append(slices.Clone(s.Recording.IgnoredApps), app)
		}
	})
}

// CloseToast closes the notice that a meeting's notes are ready.
func (w *Windows) CloseToast() { w.app.windows.closeToast() }

// CloseMenu closes the panel of the menu bar item.
func (w *Windows) CloseMenu() { w.app.windows.closeMenu() }

// Quit ends the app.
func (w *Windows) Quit() { mygo.App.Quit() }
