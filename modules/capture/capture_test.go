package capture_test

import (
	"context"
	"encoding/binary"
	"math"
	"os"
	"testing"
	"time"

	"meetkeep-poc/modules/capture"
)

func TestSystemAudioIsHeard(t *testing.T) {
	if os.Getenv("MEETKEEP_TONE_PID") == "" {
		t.Skip("MEETKEEP_TONE_PID is not set: no tone is playing")
	}
	recorder, err := capture.NewRecorder()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()

	frames, peak := 0, 0.0
	for frame, err := range recorder.Frames(ctx, capture.System) {
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			t.Fatal(err)
		}
		if len(frame) != 3200 {
			t.Fatalf("a frame holds %d bytes, want 3200", len(frame))
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
