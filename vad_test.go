package main

import (
	"math"
	"testing"
)

func tone(ms int, amplitude float64) []int16 {
	out := make([]int16, sampleRate*ms/1000)
	for i := range out {
		out[i] = int16(amplitude * 32767 * math.Sin(2*math.Pi*220*float64(i)/sampleRate))
	}
	return out
}

func collect(s *segmenter, chunks ...[]int16) []segmentEvent {
	var events []segmentEvent
	for _, c := range chunks {
		for len(c) > 0 {
			n := min(len(c), sampleRate/10)
			events = append(events, s.push(c[:n])...)
			c = c[n:]
		}
	}
	return events
}

func TestSegmenterCutsOnSilence(t *testing.T) {
	events := collect(newSegmenter(-60), tone(500, 0), tone(2000, 0.3), tone(1500, 0), tone(1000, 0.3), tone(1500, 0))
	var kinds []segmentKind
	for _, e := range events {
		kinds = append(kinds, e.kind)
	}
	want := []segmentKind{segmentBegan, segmentEnded, segmentBegan, segmentEnded}
	if len(kinds) != len(want) {
		t.Fatalf("got %v, want %v", kinds, want)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Fatalf("got %v, want %v", kinds, want)
		}
	}
	first := events[1]
	if first.startMs > 500 || first.startMs < 200 {
		t.Errorf("first segment starts at %dms, want within the pre-roll before 500ms", first.startMs)
	}
	if d := first.endMs - first.startMs; d < 2000 || d > 2600 {
		t.Errorf("first segment lasts %dms, want about 2000ms plus padding", d)
	}
	if got := int64(len(first.pcm)) * 1000 / sampleRate; got != first.endMs-first.startMs {
		t.Errorf("pcm holds %dms, segment spans %dms", got, first.endMs-first.startMs)
	}
}

func TestSegmenterFinishesOpenSegment(t *testing.T) {
	s := newSegmenter(-60)
	events := collect(s, tone(500, 0), tone(2000, 0.3))
	if len(events) != 1 || events[0].kind != segmentBegan {
		t.Fatalf("got %+v, want a segment that began", events)
	}
	events = s.finish()
	if len(events) != 1 || events[0].kind != segmentEnded {
		t.Fatalf("got %+v, want the open segment ended", events)
	}
	if s.finish() != nil {
		t.Error("finish ended a segment twice")
	}
}

func TestSegmenterDropsBlips(t *testing.T) {
	events := collect(newSegmenter(-60), tone(500, 0), tone(120, 0.3), tone(1500, 0))
	if len(events) != 2 || events[1].kind != segmentDropped {
		t.Fatalf("got %+v, want a segment that began and was dropped", events)
	}
}

func TestSegmenterSplitsLongSpeech(t *testing.T) {
	events := collect(newSegmenter(-60), tone(500, 0), tone(45000, 0.3), tone(1500, 0))
	ended := 0
	for _, e := range events {
		if e.kind == segmentEnded {
			ended++
			if d := e.endMs - e.startMs; d > 15000 {
				t.Errorf("segment lasts %dms, over the 15s limit", d)
			}
		}
	}
	if ended != 3 {
		t.Errorf("got %d segments, want 3", ended)
	}
}
