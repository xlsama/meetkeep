package audio

import (
	"math"
	"testing"
)

func tone(hz float64, n int) []int16 {
	out := make([]int16, n)
	for i := range out {
		out[i] = int16(16000 * math.Sin(2*math.Pi*hz*float64(i)/CaptureRate))
	}
	return out
}

func rms(samples []int16) float64 {
	var sum float64
	for _, s := range samples {
		sum += float64(s) * float64(s)
	}
	return math.Sqrt(sum / float64(len(samples)))
}

func TestDecimatorKeepsSpeechAndRemovesWhatWouldAlias(t *testing.T) {
	for _, c := range []struct {
		hz   float64
		keep bool
	}{{1000, true}, {5000, true}, {12000, false}, {20000, false}} {
		var d Decimator
		var out []int16
		in := tone(c.hz, CaptureRate)
		for at := 0; at < len(in); at += 4800 {
			out = append(out, d.Push(in[at:at+4800])...)
		}
		if want := SampleRate - len(lowpass)/decimation - 1; len(out) < want {
			t.Fatalf("%v Hz: got %d samples, want at least %d", c.hz, len(out), want)
		}
		ratio := rms(out[100:]) / rms(in)
		if c.keep && ratio < 0.9 {
			t.Errorf("%v Hz is kept at %.2f of its level", c.hz, ratio)
		}
		if !c.keep && ratio > 0.02 {
			t.Errorf("%v Hz leaks at %.3f of its level", c.hz, ratio)
		}
	}
}

func TestLevel(t *testing.T) {
	if got := Level(make([]int16, 480)); got != 0 {
		t.Errorf("silence is at %v", got)
	}
	if got := Level(tone(1000, 4800)); got < 0.8 || got > 0.9 {
		t.Errorf("a tone at -9 dBFS is at %v", got)
	}
}

func TestBytesRoundTrip(t *testing.T) {
	in := []int16{0, 1, -1, math.MaxInt16, math.MinInt16}
	out := Samples(Bytes(in))
	for i := range in {
		if in[i] != out[i] {
			t.Fatalf("got %v, want %v", out, in)
		}
	}
}

func TestQuietest(t *testing.T) {
	samples := make([]int16, 10*SampleRate)
	for i := range samples {
		samples[i] = 4000
	}
	for i := 6200 * 16; i < 6800*16; i++ {
		samples[i] = 3
	}
	for i := 2000 * 16; i < 2300*16; i++ {
		samples[i] = 0
	}
	if got := Quietest(samples, 400); got != 6200 {
		t.Errorf("the quietest pause starts at %d ms, want 6200: the start of the only pause long enough", got)
	}
	if got := Quietest(make([]int16, 10*SampleRate), 400); got != 4800 {
		t.Errorf("silence is cut at %d ms, want its middle", got)
	}
	if got := Quietest(samples[:100], 400); got != 0 {
		t.Errorf("a sound shorter than a pause is cut at %d", got)
	}
}
