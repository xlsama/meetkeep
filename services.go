package main

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/egoist/mygo"

	"github.com/xlsama/meetkeep/internal/asr"
	"github.com/xlsama/meetkeep/internal/audio"
	"github.com/xlsama/meetkeep/internal/config"
	"github.com/xlsama/meetkeep/internal/llm"
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

// Stop ends the meeting and returns it.
func (r *Recorder) Stop() (store.Meeting, error) {
	return r.app.recorder.Stop(45 * time.Second)
}

// Snapshot returns what the recorder does now, with the utterances that
// are still being spoken or transcribed.
func (r *Recorder) Snapshot() recorder.Snapshot { return r.app.recorder.Snapshot() }

// Library keeps the meetings.
type Library struct{ app *app }

// MeetingDetail is a meeting with what was said in it.
type MeetingDetail struct {
	Meeting    store.Meeting     `json:"meeting"`
	Utterances []store.Utterance `json:"utterances"`
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
	return MeetingDetail{Meeting: meeting, Utterances: utterances}, err
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

// CloseMenu closes the panel of the menu bar item.
func (w *Windows) CloseMenu() { w.app.windows.closeMenu() }

// Quit ends the app.
func (w *Windows) Quit() { mygo.App.Quit() }
