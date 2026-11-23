package main

import (
	"embed"
	"sync"
	"time"

	"github.com/egoist/mygo/audio"

	"github.com/xlsama/meetkeep/internal/recorder"
)

//go:embed assets/sounds/*.wav
var soundFiles embed.FS

// cue names a sound the app makes.
type cue string

const (
	cueStart  cue = "start"
	cueStop   cue = "stop"
	cuePause  cue = "pause"
	cueResume cue = "resume"
	cueNotice cue = "notice"
	cueReady  cue = "ready"
	cueFault  cue = "fault"
)

var cues = []cue{cueStart, cueStop, cuePause, cueResume, cueNotice, cueReady, cueFault}

// faultEvery is how often at most a recording that goes wrong is heard.
const faultEvery = time.Minute

// sounds plays the cues from the app's own process, which the recording of
// the system's audio leaves out.
type sounds struct {
	app *app

	mu      sync.Mutex
	loaded  map[cue]*audio.Sound
	phase   recorder.Phase
	faulted time.Time
}

func loadCue(c cue) (*audio.Sound, error) {
	data, err := soundFiles.ReadFile("assets/sounds/" + string(c) + ".wav")
	if err != nil {
		return nil, err
	}
	return audio.DecodeSound(data)
}

func (s *sounds) play(c cue) {
	if !s.app.current().Recording.Sounds {
		return
	}
	s.mu.Lock()
	sound, ok := s.loaded[c]
	if !ok {
		sound, _ = loadCue(c)
		if s.loaded == nil {
			s.loaded = map[cue]*audio.Sound{}
		}
		s.loaded[c] = sound
	}
	s.mu.Unlock()
	if sound != nil {
		sound.Play()
	}
}

// cueBetween names the sound of the recorder going from one phase to
// another, or none.
func cueBetween(from, to recorder.Phase) cue {
	switch {
	case from == recorder.Idle && to == recorder.Recording:
		return cueStart
	case from == recorder.Recording && to == recorder.Paused:
		return cuePause
	case from == recorder.Paused && to == recorder.Recording:
		return cueResume
	case (from == recorder.Recording || from == recorder.Paused) && to == recorder.Stopping:
		return cueStop
	}
	return ""
}

// follow sounds what the recorder began to do.
func (s *sounds) follow(status recorder.Status) {
	s.mu.Lock()
	from := s.phase
	if from == "" {
		from = recorder.Idle
	}
	s.phase = status.Phase
	if status.Phase == recorder.Idle {
		s.faulted = time.Time{}
	}
	s.mu.Unlock()
	if c := cueBetween(from, status.Phase); c != "" {
		s.play(c)
	}
}

// fault sounds that a recording goes wrong, once for the faults that come
// close together.
func (s *sounds) fault() {
	s.mu.Lock()
	now := time.Now()
	heard := now.Sub(s.faulted) < faultEvery
	if !heard {
		s.faulted = now
	}
	s.mu.Unlock()
	if !heard {
		s.play(cueFault)
	}
}
