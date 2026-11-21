// Package capture is a native module that records the microphone and the
// system's audio output.
package capture

import (
	"context"
	"iter"

	"github.com/egoist/mygo/native"
)

// Source is where audio is recorded from.
type Source string

const (
	Microphone Source = "mic"
	System     Source = "system"
)

// App is an app whose sound can be recorded alone.
type App struct {
	// ID names the app across launches: its bundle identifier on macOS, the
	// name of its executable on Windows.
	ID   string
	Name string
	// Icon is a PNG, or empty.
	Icon []byte
	// Playing reports whether the app makes sound now.
	Playing bool
}

// Options say what Frames records.
type Options struct {
	// Apps limits the system's audio to these apps, by ID. Empty records
	// everything the system plays but this app.
	Apps []string
	// EchoCancellation removes what the speakers play from the microphone.
	EchoCancellation bool
}

// ErrDenied means the user has not allowed the app to record the source.
var ErrDenied = native.NewError("denied")

// Recorder records audio.
//
//mygo:native
type Recorder interface {
	// Apps lists the apps whose sound can be recorded alone.
	Apps() ([]App, error)
	// Listening lists the apps that record the microphone now, as those in
	// a call do.
	Listening() ([]App, error)
	// Microphone returns the name of the microphone Frames records.
	Microphone() (string, error)
	// Headphones reports whether the output is heard by the user alone, so
	// that the microphone does not pick it up.
	Headphones() (bool, error)
	// Frames yields the audio of source as 48 kHz mono 16-bit little-endian
	// PCM, 100 ms a frame, until ctx is canceled.
	Frames(ctx context.Context, source Source, options Options) iter.Seq2[[]byte, error]
}
