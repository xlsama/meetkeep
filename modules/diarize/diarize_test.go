package diarize_test

import (
	"os"
	"path/filepath"
	"testing"

	"meetkeep-poc/modules/diarize"
)

func TestSeparateFourSpeakers(t *testing.T) {
	dir := os.Getenv("MEETKEEP_DIARIZE_FIXTURES")
	if dir == "" {
		t.Skip("MEETKEEP_DIARIZE_FIXTURES is not set: no models to run")
	}
	engine, err := diarize.NewEngine()
	if err != nil {
		t.Fatal(err)
	}
	segments, err := engine.Separate(t.Context(), filepath.Join(dir, "0-four-speakers-zh.wav"), diarize.Models{
		Segmentation: filepath.Join(dir, "segmentation.onnx"),
		Embedding:    filepath.Join(dir, "embedding.onnx"),
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	speakers := map[int]int{}
	for i, s := range segments {
		if s.EndMs <= s.StartMs {
			t.Errorf("segment %d is empty: %+v", i, s)
		}
		if i > 0 && s.StartMs < segments[i-1].StartMs {
			t.Errorf("segment %d starts before the one before it", i)
		}
		speakers[s.Speaker] += s.EndMs - s.StartMs
	}
	t.Logf("%d segments, speaking time by speaker: %v", len(segments), speakers)
	if len(speakers) != 4 {
		t.Errorf("found %d speakers, want 4", len(speakers))
	}
}
