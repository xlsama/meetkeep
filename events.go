package main

import (
	"github.com/egoist/mygo"

	"github.com/xlsama/meetkeep/internal/config"
	"github.com/xlsama/meetkeep/internal/processor"
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
	// processing tells how far the processing of a meeting got.
	processing = mygo.NewEvent[processor.Progress]("processing")
	// notesChanged carries a meeting's notes as they are written.
	notesChanged = mygo.NewEvent[NotesState]("notes")
	// libraryChanged tells that the meetings changed.
	libraryChanged = mygo.NewEvent[string]("library")
	// settingsChanged carries the settings after they were saved.
	settingsChanged = mygo.NewEvent[config.Settings]("settings")
	// updateReady names a newer version of the app that is in place, and
	// runs once the app starts again.
	updateReady = mygo.NewEvent[string]("update")
	// navigate asks the main window to show a route.
	navigate = mygo.NewEvent[string]("navigate")
)

func (a *app) Status(s recorder.Status) {
	_ = statusChanged.Broadcast(s)
	a.windows.follow(s)
	a.sounds.follow(s)
	a.notes.follow(s)
	if s.Phase == recorder.Idle || s.Phase == recorder.Recording && s.ElapsedMs < 1000 {
		_ = libraryChanged.Broadcast("")
	}
}

func (a *app) Utterance(u recorder.LiveUtterance) { _ = utteranceChanged.Broadcast(u) }
func (a *app) Levels(l recorder.Levels)           { _ = levelsChanged.Broadcast(l) }
func (a *app) Fault(f recorder.Fault) {
	_ = faulted.Broadcast(f)
	a.sounds.fault()
}
