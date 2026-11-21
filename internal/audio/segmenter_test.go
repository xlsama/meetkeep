package audio

import "testing"

func feed(s *Segmenter, probs ...float32) []Event {
	var events []Event
	for _, p := range probs {
		events = append(events, s.Push(make([]int16, Window), p)...)
	}
	return events
}

func repeat(p float32, n int) []float32 {
	out := make([]float32, n)
	for i := range out {
		out[i] = p
	}
	return out
}

func kinds(events []Event) []Kind {
	out := make([]Kind, len(events))
	for i, e := range events {
		out[i] = e.Kind
	}
	return out
}

func equal(a, b []Kind) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestCutsOnSilence(t *testing.T) {
	s := &Segmenter{}
	var probs []float32
	probs = append(probs, repeat(0.02, 20)...)
	probs = append(probs, repeat(0.9, 60)...)
	probs = append(probs, repeat(0.02, 40)...)
	probs = append(probs, repeat(0.9, 30)...)
	probs = append(probs, repeat(0.02, 40)...)
	events := feed(s, probs...)
	if want := []Kind{Began, Ended, Began, Ended}; !equal(kinds(events), want) {
		t.Fatalf("got %v, want %v", kinds(events), want)
	}
	first := events[1]
	if first.StartMs != (20+onsetRun-preRoll)*windowMs {
		t.Errorf("starts at %dms, want the pre-roll before the speech", first.StartMs)
	}
	if got := int64(len(first.PCM)) * 1000 / SampleRate; got != first.EndMs-first.StartMs {
		t.Errorf("pcm holds %dms, the utterance spans %dms", got, first.EndMs-first.StartMs)
	}
	if d := first.EndMs - first.StartMs; d < 60*windowMs || d > 80*windowMs {
		t.Errorf("lasts %dms, want about %dms plus padding", d, 60*windowMs)
	}
}

func TestDropsBlips(t *testing.T) {
	events := feed(&Segmenter{}, append(append(repeat(0.02, 20), repeat(0.9, 4)...), repeat(0.02, 40)...)...)
	if want := []Kind{Began, Dropped}; !equal(kinds(events), want) {
		t.Fatalf("got %v, want %v", kinds(events), want)
	}
}

func TestSplitsLongSpeechAtTheQuietestMoment(t *testing.T) {
	s := &Segmenter{}
	probs := append(repeat(0.02, 20), repeat(0.9, hardLimit+100)...)
	quiet := 20 + hardLimit - 30
	probs[quiet] = 0.4
	events := feed(s, probs...)
	if want := []Kind{Began, Ended, Began}; !equal(kinds(events), want) {
		t.Fatalf("got %v, want %v", kinds(events), want)
	}
	if events[1].EndMs != int64(quiet)*windowMs {
		t.Errorf("cut at %dms, want the quiet window at %dms", events[1].EndMs, quiet*windowMs)
	}
	if events[2].StartMs != events[1].EndMs {
		t.Errorf("the next utterance starts at %dms, want %dms", events[2].StartMs, events[1].EndMs)
	}
	if !s.Active() {
		t.Error("the rest of the speech is not in progress")
	}
}

func TestFinishEndsTheOpenUtterance(t *testing.T) {
	s := &Segmenter{}
	feed(s, append(repeat(0.02, 20), repeat(0.9, 40)...)...)
	events := s.Finish()
	if len(events) != 1 || events[0].Kind != Ended {
		t.Fatalf("got %+v, want the open utterance ended", events)
	}
	if s.Finish() != nil {
		t.Error("finish ended an utterance twice")
	}
}
