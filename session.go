package main

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"meetkeep-poc/modules/capture"

	"github.com/egoist/mygo"
)

type State string

const (
	StateSpeaking     State = "speaking"
	StateTranscribing State = "transcribing"
	StateDone         State = "done"
	StateDropped      State = "dropped"
	StateFailed       State = "failed"
)

type Utterance struct {
	ID      string         `json:"id"`
	Source  capture.Source `json:"source"`
	StartMs int64          `json:"startMs"`
	EndMs   int64          `json:"endMs"`
	Text    string         `json:"text"`
	State   State          `json:"state"`
}

type Levels struct {
	Mic    float64 `json:"mic"`
	System float64 `json:"system"`
}

type Fault struct {
	Source  capture.Source `json:"source"`
	Message string         `json:"message"`
}

type Options struct {
	Mic    bool `json:"mic"`
	System bool `json:"system"`
}

var (
	UtteranceChanged = mygo.NewEvent[Utterance]("utterance")
	LevelsChanged    = mygo.NewEvent[Levels]("levels")
	Faulted          = mygo.NewEvent[Fault]("fault")
)

type Session struct {
	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

func newSession() *Session { return &Session{} }

// Start records the chosen sources and transcribes what is said.
func (s *Session) Start(opts Options) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		return errors.New("already recording")
	}
	if !opts.Mic && !opts.System {
		return errors.New("choose a source to record")
	}
	asr, err := newAzureMAI()
	if err != nil {
		return err
	}
	recorder, err := capture.NewRecorder()
	if err != nil {
		return fmt.Errorf("audio capture: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	began := time.Now()
	var mic, system atomic.Uint64
	var tracks sync.WaitGroup
	if opts.Mic {
		tracks.Go(func() { track(ctx, recorder, asr, capture.Microphone, began, &mic) })
	}
	if opts.System {
		tracks.Go(func() { track(ctx, recorder, asr, capture.System, began, &system) })
	}
	go meter(ctx, &mic, &system)

	done := make(chan struct{})
	go func() {
		tracks.Wait()
		close(done)
	}()
	s.cancel, s.done = cancel, done
	return nil
}

// Stop ends the recording.
func (s *Session) Stop() {
	s.mu.Lock()
	cancel, done := s.cancel, s.done
	s.cancel, s.done = nil, nil
	s.mu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	<-done
	LevelsChanged.Broadcast(Levels{})
}

func meter(ctx context.Context, mic, system *atomic.Uint64) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	var shown Levels
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			shown.Mic = max(math.Float64frombits(mic.Swap(0)), shown.Mic*0.6)
			shown.System = max(math.Float64frombits(system.Swap(0)), shown.System*0.6)
			LevelsChanged.Broadcast(shown)
		}
	}
}

func track(ctx context.Context, recorder capture.Recorder, asr Transcriber, source capture.Source, began time.Time, peak *atomic.Uint64) {
	frames := make(chan []int16, 64)
	go func() {
		defer close(frames)
		for frame, err := range recorder.Frames(ctx, source) {
			if err != nil {
				if ctx.Err() == nil {
					log.Printf("%s: %v", source, err)
					Faulted.Broadcast(Fault{Source: source, Message: err.Error()})
				}
				return
			}
			samples := make([]int16, len(frame)/2)
			for i := range samples {
				samples[i] = int16(binary.LittleEndian.Uint16(frame[i*2:]))
			}
			select {
			case frames <- samples:
			case <-ctx.Done():
				return
			}
		}
	}()

	ceiling := -60.0
	if source == capture.Microphone {
		ceiling = -45
	}
	seg := newSegmenter(ceiling)
	silence := make([]int16, sampleRate/10)
	idle := time.NewTicker(100 * time.Millisecond)
	defer idle.Stop()

	var offset int64 = -1
	var last time.Time
	var count int
	var current string
	var pending sync.WaitGroup
	defer pending.Wait()

	handle := func(events []segmentEvent) {
		for _, event := range events {
			switch event.kind {
			case segmentBegan:
				count++
				current = fmt.Sprintf("%s-%d", source, count)
				UtteranceChanged.Broadcast(Utterance{
					ID: current, Source: source, StartMs: offset + event.startMs, State: StateSpeaking,
				})
			case segmentDropped:
				UtteranceChanged.Broadcast(Utterance{ID: current, Source: source, State: StateDropped})
			case segmentEnded:
				u := Utterance{
					ID: current, Source: source,
					StartMs: offset + event.startMs, EndMs: offset + event.endMs,
					State: StateTranscribing,
				}
				UtteranceChanged.Broadcast(u)
				pending.Go(func() { transcribe(asr, u, event.pcm) })
			}
		}
	}
	feed := func(samples []int16) {
		if v := level(samples); v > math.Float64frombits(peak.Load()) {
			peak.Store(math.Float64bits(v))
		}
		handle(seg.push(samples))
	}

	for {
		select {
		case samples, ok := <-frames:
			if !ok {
				handle(seg.finish())
				return
			}
			if offset < 0 {
				offset = time.Since(began).Milliseconds()
			}
			last = time.Now()
			feed(samples)
		case <-idle.C:
			if offset >= 0 && seg.active && time.Since(last) > 300*time.Millisecond {
				feed(silence)
			}
		}
	}
}

func transcribe(asr Transcriber, u Utterance, pcm []int16) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	sent := time.Now()
	text, err := asr.Transcribe(ctx, pcm)
	switch {
	case err != nil:
		log.Printf("%s: %v", u.ID, err)
		u.State, u.Text = StateFailed, err.Error()
	case text == "":
		u.State = StateDropped
	default:
		u.State, u.Text = StateDone, text
	}
	log.Printf("%s %.1fs audio, %dms: %s", u.ID, float64(len(pcm))/sampleRate, time.Since(sent).Milliseconds(), u.Text)
	UtteranceChanged.Broadcast(u)
}
