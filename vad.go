package main

import "math"

const (
	vadFrame      = sampleRate / 50
	frameMs       = 20
	preRoll       = 15
	onsetFrames   = 3
	endSilence    = 25
	softLimit     = 10 * 50
	softSilence   = 12
	hardLimit     = 15 * 50
	minSpeech     = 15
	trailingKept  = 10
	onsetMargin   = 12.0
	releaseMargin = 8.0
)

type segmentEvent struct {
	kind    segmentKind
	startMs int64
	endMs   int64
	pcm     []int16
}

type segmentKind int

const (
	segmentBegan segmentKind = iota
	segmentEnded
	segmentDropped
)

type segmenter struct {
	ceiling float64
	rest    []int16
	frames  int64
	floor   float64
	history [][]int16
	onset   int

	active  bool
	start   int64
	pcm     []int16
	length  int
	speech  int
	silence int
}

func newSegmenter(ceiling float64) *segmenter { return &segmenter{ceiling: ceiling, floor: -60} }

func (s *segmenter) push(samples []int16) []segmentEvent {
	var events []segmentEvent
	s.rest = append(s.rest, samples...)
	for len(s.rest) >= vadFrame {
		frame := s.rest[:vadFrame:vadFrame]
		s.rest = s.rest[vadFrame:]
		events = s.step(frame, events)
		s.frames++
	}
	return events
}

func (s *segmenter) step(frame []int16, events []segmentEvent) []segmentEvent {
	db := decibels(frame)
	threshold := math.Max(s.floor+onsetMargin, s.ceiling)
	if s.active {
		threshold = math.Max(s.floor+releaseMargin, s.ceiling-4)
	}
	voiced := db > threshold

	if !s.active {
		if db < s.floor {
			s.floor = math.Max(db, -80)
		} else if !voiced {
			s.floor = math.Min(s.floor+0.05, -30)
		}
		s.history = append(s.history, frame)
		if len(s.history) > preRoll {
			s.history = s.history[1:]
		}
		if !voiced {
			s.onset = 0
			return events
		}
		if s.onset++; s.onset < onsetFrames {
			return events
		}
		s.active, s.onset = true, 0
		s.start = s.frames - int64(len(s.history)) + 1
		s.pcm = s.pcm[:0]
		for _, f := range s.history {
			s.pcm = append(s.pcm, f...)
		}
		s.length, s.speech, s.silence = len(s.history), onsetFrames, 0
		s.history = s.history[:0]
		return append(events, segmentEvent{kind: segmentBegan, startMs: s.start * frameMs})
	}

	s.pcm = append(s.pcm, frame...)
	s.length++
	if voiced {
		s.speech++
		s.silence = 0
	} else {
		s.silence++
	}
	switch {
	case s.silence >= endSilence,
		s.length >= softLimit && s.silence >= softSilence,
		s.length >= hardLimit:
		return append(events, s.close())
	}
	return events
}

func (s *segmenter) finish() []segmentEvent {
	if !s.active {
		return nil
	}
	return []segmentEvent{s.close()}
}

func (s *segmenter) close() segmentEvent {
	trim := max(s.silence-trailingKept, 0)
	pcm := append([]int16(nil), s.pcm[:len(s.pcm)-trim*vadFrame]...)
	event := segmentEvent{
		kind:    segmentEnded,
		startMs: s.start * frameMs,
		endMs:   (s.start + int64(s.length-trim)) * frameMs,
		pcm:     pcm,
	}
	if s.speech < minSpeech {
		event.kind, event.pcm = segmentDropped, nil
	}
	s.active = false
	return event
}

func decibels(frame []int16) float64 {
	var sum float64
	for _, v := range frame {
		x := float64(v) / 32768
		sum += x * x
	}
	rms := math.Sqrt(sum / float64(len(frame)))
	if rms <= 0 {
		return -100
	}
	return 20 * math.Log10(rms)
}

func level(frame []int16) float64 {
	return math.Max(0, math.Min(1, (decibels(frame)+50)/50))
}
