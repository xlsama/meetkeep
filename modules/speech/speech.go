// Package speech is a native module that runs the speech models: it tells
// when someone speaks, and who speaks when in a recording.
package speech

import (
	"context"
	"io"
)

// Models are the ONNX files Separate runs.
type Models struct {
	// Segmentation is pyannote segmentation 3.0.
	Segmentation string
	// Embedding is CAM++ (3D-Speaker campplus, 192 dimensions).
	Embedding string
}

// Segment is a stretch of audio in which one speaker talks.
type Segment struct {
	StartMs int
	EndMs   int
	// Speaker counts from 0, the one who talks the most.
	Speaker int
}

// Engine runs the models.
//
//mygo:native
type Engine interface {
	// Detector returns a voice activity detector for one stream of audio.
	Detector() (Detector, error)
	// Record creates an Ogg Opus file at path and returns what writes it.
	Record(path string) (Recording, error)
	// Separate returns who speaks when in the 16 kHz mono 16-bit WAV file at
	// path. speakers is how many people talk, or 0 to estimate it.
	Separate(ctx context.Context, path string, models Models, speakers int) ([]Segment, error)
}

// Recording is an audio file being written.
type Recording interface {
	// Close ends the file.
	io.Closer
	// Write appends 48 kHz mono 16-bit little-endian PCM. What it is given
	// is on disk within a second.
	Write(pcm []byte) error
}

// Detector tells when someone speaks in a stream of audio.
type Detector interface {
	io.Closer
	// Feed takes 16 kHz mono 16-bit little-endian PCM and returns the
	// probability of speech in each 32 ms window it completes.
	Feed(pcm []byte) ([]float32, error)
}
