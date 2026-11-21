// Package diarize is a native module that tells who speaks when in a
// recording.
package diarize

import "context"

// Models are the ONNX files the engine runs.
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

// Engine separates speakers.
//
//mygo:native
type Engine interface {
	// Separate returns who speaks when in the 16 kHz mono 16-bit WAV file at
	// path. speakers is how many people talk, or 0 to estimate it.
	Separate(ctx context.Context, path string, models Models, speakers int) ([]Segment, error)
}
