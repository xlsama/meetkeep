package main

import (
	"testing"

	"github.com/xlsama/meetkeep/internal/recorder"
)

func TestCueBetween(t *testing.T) {
	tests := []struct {
		from, to recorder.Phase
		want     cue
	}{
		{recorder.Idle, recorder.Recording, cueStart},
		{recorder.Recording, recorder.Recording, ""},
		{recorder.Recording, recorder.Paused, cuePause},
		{recorder.Paused, recorder.Recording, cueResume},
		{recorder.Recording, recorder.Stopping, cueStop},
		{recorder.Paused, recorder.Stopping, cueStop},
		{recorder.Stopping, recorder.Idle, ""},
		{recorder.Idle, recorder.Idle, ""},
	}
	for _, tt := range tests {
		if got := cueBetween(tt.from, tt.to); got != tt.want {
			t.Errorf("cueBetween(%q, %q) = %q, want %q", tt.from, tt.to, got, tt.want)
		}
	}
}

func TestCuesDecode(t *testing.T) {
	for _, c := range cues {
		sound, err := loadCue(c)
		if err != nil {
			t.Errorf("%s: %v", c, err)
			continue
		}
		if d := sound.Duration(); d <= 0 || d.Seconds() > 2 {
			t.Errorf("%s lasts %v", c, d)
		}
	}
}
