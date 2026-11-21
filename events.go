package main

import (
	"github.com/egoist/mygo"

	"github.com/xlsama/meetkeep/internal/config"
	"github.com/xlsama/meetkeep/internal/recorder"
)

var (
	// statusChanged tells what the recorder does now.
	statusChanged = mygo.NewEvent[recorder.Status]("status")
	// utteranceChanged follows an utterance of the meeting being recorded.
	utteranceChanged = mygo.NewEvent[recorder.LiveUtterance]("utterance")
	// levelsChanged tells how loud the tracks are, ten times a second.
	levelsChanged = mygo.NewEvent[recorder.Levels]("levels")
	// faulted tells of something that went wrong while recording goes on.
	faulted = mygo.NewEvent[recorder.Fault]("fault")
	// libraryChanged tells that the meetings changed.
	libraryChanged = mygo.NewEvent[string]("library")
	// settingsChanged carries the settings after they were saved.
	settingsChanged = mygo.NewEvent[config.Settings]("settings")
	// navigate asks the main window to show a route.
	navigate = mygo.NewEvent[string]("navigate")
)

func (a *app) Status(s recorder.Status) {
	_ = statusChanged.Broadcast(s)
	a.windows.follow(s)
	if s.Phase == recorder.Idle || s.Phase == recorder.Recording && s.ElapsedMs < 1000 {
		_ = libraryChanged.Broadcast("")
	}
}

func (a *app) Utterance(u recorder.LiveUtterance) { _ = utteranceChanged.Broadcast(u) }
func (a *app) Levels(l recorder.Levels)           { _ = levelsChanged.Broadcast(l) }
func (a *app) Fault(f recorder.Fault)             { _ = faulted.Broadcast(f) }
