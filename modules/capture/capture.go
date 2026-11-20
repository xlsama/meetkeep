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

// ErrDenied means the user has not allowed the app to record the source.
var ErrDenied = native.NewError("denied")

// Recorder records audio.
//
//mygo:native
type Recorder interface {
	// Frames yields the audio of source as 16 kHz mono 16-bit little-endian
	// PCM, 100 ms a frame, until ctx is canceled.
	Frames(ctx context.Context, source Source) iter.Seq2[[]byte, error]
}
