package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"

	"github.com/xlsama/meetkeep/internal/asr"
	"github.com/xlsama/meetkeep/internal/config"
	"github.com/xlsama/meetkeep/internal/control"
	"github.com/xlsama/meetkeep/internal/processor"
	"github.com/xlsama/meetkeep/internal/recorder"
	"github.com/xlsama/meetkeep/internal/store"
	"github.com/xlsama/meetkeep/modules/capture"
	"github.com/xlsama/meetkeep/modules/speech"
)

type app struct {
	mu       sync.Mutex
	settings config.Settings

	store     *store.Store
	capture   capture.Recorder
	recorder  *recorder.Recorder
	processor *processor.Processor
	notes     noteTaker
	windows   windows
	sounds    sounds
}

func main() {
	a := &app{}
	a.windows.app = a
	a.notes.app = a
	a.sounds.app = a
	mygo.Bind(&Config{a}, &Sources{a}, &Recorder{a}, &Library{a}, &People{a}, &Windows{a})
	mygo.App.SetPath(mygo.PathUserData, filepath.Join(config.Dir(), "app"))
	mygo.App.OnWindowAllClosed(func() {})
	mygo.App.WhenReady(func() {
		if err := a.start(); err != nil {
			mygo.Dialog.Error("MeetKeep 无法启动", err.Error())
			mygo.App.Exit(1)
		}
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

func (a *app) start() error {
	if !a.alone() {
		mygo.App.Exit(0)
		return nil
	}
	mygo.App.OnSecondInstance(func([]string, string) { a.windows.showMain("") })

	settings, err := config.Load()
	if err != nil {
		return fmt.Errorf("设置文件 %s 读不了：%w", config.Path(), err)
	}
	a.settings = settings
	userData, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(userData, 0o700); err != nil {
		return err
	}
	ctx := context.Background()
	if a.store, err = store.Open(ctx, filepath.Join(config.Dir(), "meetkeep.db")); err != nil {
		return err
	}
	interrupted, err := a.store.Recover(ctx)
	if err != nil {
		return err
	}
	if a.capture, err = capture.NewRecorder(); err != nil {
		return err
	}
	engine, err := speech.NewEngine()
	if err != nil {
		return err
	}
	transcriber := func(s config.ASRSettings) (asr.Transcriber, error) {
		return asr.NewMAI(s.Endpoint, s.Key, s.Hotwords)
	}
	a.recorder = &recorder.Recorder{
		Capture:     a.capture,
		Speech:      engine,
		Store:       a.store,
		Settings:    a.current,
		Sink:        a,
		Transcriber: transcriber,
	}
	resources, err := mygo.App.Path(mygo.PathResources)
	if err != nil {
		return err
	}
	a.processor = &processor.Processor{
		Speech:      engine,
		Store:       a.store,
		Settings:    a.current,
		Transcriber: transcriber,
		Models: speech.Models{
			Segmentation: filepath.Join(resources, "models", "segmentation.onnx"),
			Embedding:    filepath.Join(resources, "models", "embedding.onnx"),
		},
		Recognition: filepath.Join(resources, "models", "recognition.onnx"),
		Polish:      a.polish,
		Summarize:   a.notes.finalize,
		Progress:    func(p processor.Progress) { _ = processing.Broadcast(p) },
		Changed:     func(id string) { _ = libraryChanged.Broadcast(id) },
	}
	for _, id := range interrupted {
		a.processor.Process(id)
	}
	if err := mygo.Protocol.HandleFunc("recording", a.serveRecording); err != nil {
		return err
	}
	if _, err := control.Listen(a.control); err != nil {
		log.Print(err)
	}

	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) {
		a.windows.quitting = true
		if a.recorder.Status().Phase != recorder.Idle {
			_, _ = a.recorder.Stop(3 * time.Second)
		}
	})
	mygo.App.OnActivate(func(visible bool) {
		if !visible {
			a.windows.showMain("")
		}
	})
	if err := a.windows.start(); err != nil {
		return err
	}
	if err := a.conform(settings.General); err != nil {
		log.Print(err)
	}
	return nil
}

// serveRecording serves what a meeting is played from:
// recording://localhost/<meeting>/audio, and /video for the picture of an
// imported video.
func (a *app) serveRecording(w http.ResponseWriter, r *http.Request) {
	id, name, _ := strings.Cut(strings.TrimPrefix(r.URL.Path, "/"), "/")
	meeting, err := a.store.Meeting(r.Context(), id)
	switch {
	case err == nil && name == "audio" && meeting.Audio != "":
		w.Header().Set("Content-Type", processor.Playable[filepath.Ext(meeting.Audio)])
		http.ServeFile(w, r, filepath.Join(meeting.Directory, filepath.Base(meeting.Audio)))
	case err == nil && name == "video" && meeting.Video != "":
		kinds := map[string]string{".mp4": "video/mp4", ".m4v": "video/mp4", ".mov": "video/quicktime", ".webm": "video/webm"}
		w.Header().Set("Content-Type", kinds[filepath.Ext(meeting.Video)])
		http.ServeFile(w, r, filepath.Join(meeting.Directory, filepath.Base(meeting.Video)))
	default:
		http.NotFound(w, r)
	}
}

// control does what the command line asks for.
func (a *app) control(ctx context.Context, request control.Request) (control.Answer, error) {
	if request.Progress != "" {
		progress, working := a.processor.State(request.Progress)
		return control.Answer{Working: working, Step: string(progress.Step), Percent: progress.Percent}, nil
	}
	if !media[strings.ToLower(filepath.Ext(request.Import))] {
		return control.Answer{}, errors.New("only audio and video files can be imported")
	}
	meeting, err := (&Library{a}).ImportFile(ctx, request.Import)
	return control.Answer{Meeting: meeting}, err
}

// media are the kinds of files a meeting is made of.
var media = map[string]bool{
	".aac": true, ".aif": true, ".aiff": true, ".caf": true, ".flac": true, ".m4a": true, ".m4v": true, ".mkv": true,
	".mov": true, ".mp3": true, ".mp4": true, ".oga": true, ".ogg": true, ".opus": true, ".wav": true, ".webm": true,
}

// alone reports whether this is the only instance of the app. A development
// build waits for the one it replaces to quit.
func (a *app) alone() bool {
	for attempt := 0; ; attempt++ {
		if mygo.App.RequestSingleInstanceLock() {
			return true
		}
		if !mygo.IsDev() || attempt == 20 {
			return false
		}
		time.Sleep(150 * time.Millisecond)
	}
}

func (a *app) current() config.Settings {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.settings
}

func (a *app) update(change func(*config.Settings)) error {
	a.mu.Lock()
	old := a.settings
	next := old
	change(&next)
	if reflect.DeepEqual(next, old) {
		a.mu.Unlock()
		return nil
	}
	if err := config.Save(next); err != nil {
		a.mu.Unlock()
		return err
	}
	a.settings = next
	a.mu.Unlock()
	_ = settingsChanged.Broadcast(next)
	if next.General != old.General {
		return a.conform(next.General)
	}
	return nil
}

// conform has the app be in the menu bar and open at login as the settings
// say. A development build leaves the login items alone.
func (a *app) conform(general config.GeneralSettings) error {
	if err := a.windows.showTray(general.MenuBar); err != nil {
		return err
	}
	if mygo.IsDev() || mygo.App.OpenAtLogin() == general.OpenAtLogin {
		return nil
	}
	return mygo.App.SetOpenAtLogin(general.OpenAtLogin)
}
