package capture_test

import (
	"context"
	"encoding/binary"
	"math"
	"os"
	"testing"
	"time"

	"github.com/xlsama/meetkeep/modules/capture"
)

func recorder(t *testing.T) capture.Recorder {
	t.Helper()
	r, err := capture.NewRecorder()
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestListsWhatCanBeRecorded(t *testing.T) {
	r := recorder(t)
	apps, err := r.Apps()
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, a := range apps {
		if a.ID == "" || a.Name == "" {
			t.Errorf("an app has no id or name: %+v", a)
		}
		if seen[a.ID] {
			t.Errorf("%s is listed twice", a.ID)
		}
		seen[a.ID] = true
	}
	t.Logf("%d apps", len(apps))
	if _, err := r.Headphones(); err != nil {
		t.Errorf("headphones: %v", err)
	}
}

func TestSystemAudioIsHeard(t *testing.T) {
	if os.Getenv("MEETKEEP_TONE_PID") == "" {
		t.Skip("MEETKEEP_TONE_PID is not set: no tone is playing")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()

	frames, peak := 0, 0.0
	for frame, err := range recorder(t).Frames(ctx, capture.System, capture.Options{}) {
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			t.Fatal(err)
		}
		if len(frame) != 9600 {
			t.Fatalf("a frame holds %d bytes, want 9600", len(frame))
		}
		var sum float64
		for i := 0; i < len(frame); i += 2 {
			v := float64(int16(binary.LittleEndian.Uint16(frame[i:]))) / 32768
			sum += v * v
		}
		peak = max(peak, math.Sqrt(sum/float64(len(frame)/2)))
		frames++
	}
	t.Logf("%d frames, peak rms %.4f", frames, peak)
	if frames < 20 {
		t.Errorf("got %d frames in 3 s, want about 30", frames)
	}
	if peak < 0.01 {
		t.Errorf("peak rms %.4f: the tone was not captured", peak)
	}
}
