package main

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/egoist/mygo"

	"github.com/xlsama/meetkeep/internal/recorder"
)

const (
	updateFirst = 15 * time.Second
	updateEvery = 4 * time.Hour
)

// updater keeps the app up to date: a newer release is downloaded and put
// in place while the app runs, and runs once the app starts again.
type updater struct {
	mu    sync.Mutex
	ready string
}

// version returns the newer version that is in place, or "".
func (u *updater) version() string {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.ready
}

// watch looks for a newer release soon after the app started, and then a
// few times a day. A build that cannot update itself, as a development
// build, looks for none.
func (u *updater) watch() {
	if !mygo.Updater.Enabled() {
		return
	}
	for wait := updateFirst; ; wait = updateEvery {
		time.Sleep(wait)
		u.fetch()
	}
}

func (u *updater) fetch() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()
	found, err := mygo.Updater.Check(ctx)
	if err != nil || found == nil || found.Version == u.version() {
		return
	}
	if err := found.Install(ctx, func(int64, int64) {}); err != nil {
		log.Print(err)
		return
	}
	u.mu.Lock()
	u.ready = found.Version
	u.mu.Unlock()
	_ = updateReady.Broadcast(found.Version)
}

// Updates tells of a newer version of the app, and starts it.
type Updates struct{ app *app }

// Ready returns the newer version that is in place and runs once the app
// starts again, or "".
func (u *Updates) Ready() string { return u.app.updates.version() }

// Restart starts the app again, as the version that is ready. A recording
// is not cut short for it.
func (u *Updates) Restart() error {
	if u.app.recorder.Status().Phase != recorder.Idle {
		return errors.New("录制结束后再更新")
	}
	mygo.App.Relaunch()
	return nil
}
