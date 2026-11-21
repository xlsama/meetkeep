package recorder

import (
	"context"
	"errors"
	"iter"
	"path/filepath"
	"slices"
	"sync/atomic"
	"time"

	"github.com/xlsama/meetkeep/internal/audio"
	"github.com/xlsama/meetkeep/internal/store"
	"github.com/xlsama/meetkeep/modules/capture"
	"github.com/xlsama/meetkeep/modules/speech"
)

const frameBytes = audio.CaptureRate / 10 * 2

// track is one source of a meeting. Its goroutine alone touches what
// follows the level.
type track struct {
	s     *session
	name  store.Track
	ready chan error
	level atomic.Int64

	recording speech.Recording
	detector  speech.Detector
	decimator audio.Decimator
	segmenter audio.Segmenter
	windows   []int16
	current   LiveUtterance
	started   bool
	paused    bool
	faulted   bool
}

func (s *session) open(name store.Track) (*track, error) {
	recording, err := s.r.Speech.Record(filepath.Join(s.meeting.Directory, string(name)+".opus"))
	if err != nil {
		return nil, err
	}
	detector, err := s.r.Speech.Detector()
	if err != nil {
		recording.Close()
		return nil, err
	}
	return &track{s: s, name: name, ready: make(chan error, 1), recording: recording, detector: detector}, nil
}

func (t *track) close() {
	t.detector.Close()
	t.recording.Close()
}

func (t *track) explain(err error) error {
	if !errors.Is(err, capture.ErrDenied) {
		return err
	}
	if t.name == store.Mic {
		return errMicDenied
	}
	return errTapDenied
}

func (t *track) fault(err error) {
	if t.faulted {
		return
	}
	t.faulted = true
	t.s.r.Sink.Fault(Fault{Track: t.name, Message: t.explain(err).Error()})
}

func (s *session) capture(ctx context.Context, t *track, frames iter.Seq2[[]byte, error]) {
	defer s.capturing.Done()
	type result struct {
		frame []byte
		err   error
	}
	in := make(chan result, 64)
	go func() {
		defer close(in)
		for frame, err := range frames {
			select {
			case in <- result{frame, err}:
			case <-ctx.Done():
				return
			}
			if err != nil {
				return
			}
		}
	}()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	silence := make([]byte, frameBytes)
	last := time.Now()
	defer t.finish()
	for {
		select {
		case <-ctx.Done():
			return
		case r, ok := <-in:
			if !ok {
				return
			}
			if r.err != nil {
				if ctx.Err() != nil {
					return
				}
				if !t.started {
					t.ready <- r.err
				} else {
					t.fault(r.err)
				}
				return
			}
			if !t.started {
				t.started = true
				t.ready <- nil
			}
			last = time.Now()
			t.process(r.frame)
		case <-ticker.C:
			if t.started && time.Since(last) > 300*time.Millisecond {
				t.process(silence)
			}
		}
	}
}

func (t *track) process(frame []byte) {
	if t.s.paused.Load() {
		if !t.paused {
			t.paused = true
			t.finish()
		}
		t.level.Store(0)
		return
	}
	t.paused = false
	if err := t.recording.Write(frame); err != nil {
		t.fault(err)
	}
	samples := audio.Samples(frame)
	t.level.Store(int64(audio.Level(samples) * 1000))
	low := t.decimator.Push(samples)
	t.windows = append(t.windows, low...)
	probabilities, err := t.detector.Feed(audio.Bytes(low))
	if err != nil {
		t.fault(err)
		return
	}
	for _, p := range probabilities {
		window := slices.Clone(t.windows[:audio.Window])
		t.windows = t.windows[audio.Window:]
		t.handle(t.segmenter.Push(window, p))
	}
}

func (t *track) finish() {
	t.handle(t.segmenter.Finish())
}

func (t *track) handle(events []audio.Event) {
	for _, e := range events {
		switch e.Kind {
		case audio.Began:
			t.current = LiveUtterance{
				MeetingID: t.s.meeting.ID,
				Seq:       int(t.s.seq.Add(1)),
				Track:     t.name,
				StartMs:   e.StartMs,
				EndMs:     e.StartMs,
				State:     Speaking,
			}
			t.s.emit(t.current)
		case audio.Dropped:
			t.current.State = Dropped
			t.s.emit(t.current)
		case audio.Ended:
			u := t.current
			u.StartMs, u.EndMs, u.State = e.StartMs, e.EndMs, Transcribing
			t.s.emit(u)
			t.s.transcribes.Add(1)
			go t.s.transcribe(u, e.PCM)
		}
	}
}
