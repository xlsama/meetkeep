package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"github.com/egoist/mygo"

	"github.com/xlsama/meetkeep/internal/asr"
	"github.com/xlsama/meetkeep/internal/config"
	"github.com/xlsama/meetkeep/internal/recorder"
	"github.com/xlsama/meetkeep/internal/store"
	"github.com/xlsama/meetkeep/modules/capture"
	"github.com/xlsama/meetkeep/modules/speech"
)

type app struct {
	mu       sync.Mutex
	settings config.Settings

	store    *store.Store
	capture  capture.Recorder
	recorder *recorder.Recorder
	windows  windows
}

func main() {
	a := &app{}
	a.windows.app = a
	mygo.Bind(&Config{a}, &Sources{a}, &Recorder{a}, &Library{a}, &Windows{a})
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
	if !mygo.App.RequestSingleInstanceLock() {
		mygo.App.Exit(0)
		return nil
	}
	mygo.App.OnSecondInstance(func([]string, string) { a.windows.showMain("") })

	settings, err := config.Load()
	if err != nil {
		return fmt.Errorf("设置文件 %s 读不了：%w", config.Path(), err)
	}
	a.settings = settings
	if err := os.MkdirAll(config.Dir(), 0o700); err != nil {
		return err
	}
	ctx := context.Background()
	if a.store, err = store.Open(ctx, filepath.Join(config.Dir(), "meetkeep.db")); err != nil {
		return err
	}
	if err := a.store.Recover(ctx); err != nil {
		return err
	}
	if a.capture, err = capture.NewRecorder(); err != nil {
		return err
	}
	engine, err := speech.NewEngine()
	if err != nil {
		return err
	}
	a.recorder = &recorder.Recorder{
		Capture:  a.capture,
		Speech:   engine,
		Store:    a.store,
		Settings: a.current,
		Sink:     a,
		Transcriber: func(s config.ASRSettings) (asr.Transcriber, error) {
			return asr.NewMAI(s.Endpoint, s.Key, s.Hotwords)
		},
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
	return a.windows.start()
}

func (a *app) current() config.Settings {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.settings
}

func (a *app) update(change func(*config.Settings)) error {
	a.mu.Lock()
	next := a.settings
	change(&next)
	if reflect.DeepEqual(next, a.settings) {
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
	return nil
}
