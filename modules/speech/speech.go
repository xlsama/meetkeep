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

// Span is a stretch of audio.
type Span struct {
	StartMs int
	EndMs   int
}

// Voiceprint is what tells a voice from others: a unit vector, near those
// of the same voice. It is empty for a span too short to tell a voice from.
type Voiceprint struct {
	Vector []float32
}

// Engine runs the models.
//
//mygo:native
type Engine interface {
	// Detector returns a voice activity detector for one stream of audio.
	Detector() (Detector, error)
	// Record creates an Ogg Opus file at path and returns what writes it.
	Record(path string) (Recording, error)
	// Mix writes the sum of the Ogg Opus files to output, as Ogg Opus.
	Mix(ctx context.Context, inputs []string, output string) error
	// Import decodes the audio of the media file at input and writes it to
	// output as Ogg Opus. It returns how long the audio lasts, in
	// milliseconds.
	Import(ctx context.Context, input string, output string) (int, error)
	// Open reads the sound of the file at path: an Ogg Opus file, or any
	// media file Import reads.
	Open(ctx context.Context, path string) (Sound, error)
}

// Sound is the sound of a file, held as 16 kHz mono: what the models and
// the services that transcribe hear. A file is read once, however many of
// them hear it, and they may hear it at the same time.
type Sound interface {
	// Close lets go of the sound.
	io.Closer
	// Duration returns how long the sound lasts, in milliseconds.
	Duration() (int, error)
	// Read returns a stretch of the sound as 16 kHz mono 16-bit
	// little-endian PCM.
	Read(startMs int, endMs int) ([]byte, error)
	// Clip returns a stretch of the sound as an Ogg Opus file.
	Clip(startMs int, endMs int) ([]byte, error)
	// Hear tells where the speech of the sound is. model is pyannote
	// segmentation 3.0, which hears those far from the microphone too.
	Hear(ctx context.Context, model string) (Hearing, error)
	// Embed returns the voiceprint of each span of the sound. model is
	// ReDimNet2 b6 as ONNX: a 16 kHz waveform in, 192 dimensions out.
	Embed(ctx context.Context, model string, spans []Span) ([]Voiceprint, error)
	// Separate returns who speaks when in the sound. speakers is how many
	// people talk, or 0 to estimate it.
	Separate(ctx context.Context, models Models, speakers int) ([]Segment, error)
}

// Recording is an audio file being written.
type Recording interface {
	// Close ends the file.
	io.Closer
	// Write appends 48 kHz mono 16-bit little-endian PCM. What it is given
	// is on disk within a second.
	Write(pcm []byte) error
}

// Hearing tells where the speech of a recording is, by the 32 ms.
type Hearing struct {
	// Speech is how likely someone speaks in each 32 ms of the recording.
	Speech []float32
	// Turns are the 32 ms at which another speaker starts to talk.
	Turns []int
}

// Detector tells when someone speaks in a stream of audio.
type Detector interface {
	io.Closer
	// Feed takes 16 kHz mono 16-bit little-endian PCM and returns the
	// probability of speech in each 32 ms window it completes.
	Feed(pcm []byte) ([]float32, error)
}
