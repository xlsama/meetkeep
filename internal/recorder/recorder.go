// Package recorder records a meeting: it captures the audio, cuts it into
// utterances, transcribes them and stores what was said.
package recorder

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xlsama/meetkeep/internal/asr"
	"github.com/xlsama/meetkeep/internal/config"
	"github.com/xlsama/meetkeep/internal/store"
	"github.com/xlsama/meetkeep/modules/capture"
	"github.com/xlsama/meetkeep/modules/speech"
)

type Phase string

const (
	Idle      Phase = "idle"
	Recording Phase = "recording"
	Paused    Phase = "paused"
	Stopping  Phase = "stopping"
)

// Status is what the recorder does now. ElapsedMs was measured at At, in
// Unix milliseconds, and runs on while Phase is recording.
type Status struct {
	Phase     Phase          `json:"phase"`
	Meeting   *store.Meeting `json:"meeting"`
	ElapsedMs int64          `json:"elapsedMs"`
	At        int64          `json:"at"`
}

type UtteranceState string

const (
	Speaking     UtteranceState = "speaking"
	Transcribing UtteranceState = "transcribing"
	Transcribed  UtteranceState = "done"
	Dropped      UtteranceState = "dropped"
	Failed       UtteranceState = "failed"
)

// LiveUtterance is an utterance of the meeting being recorded, as it goes
// from being spoken to being text.
type LiveUtterance struct {
	MeetingID string         `json:"meetingId"`
	Seq       int            `json:"seq"`
	Track     store.Track    `json:"track"`
	StartMs   int64          `json:"startMs"`
	EndMs     int64          `json:"endMs"`
	Text      string         `json:"text"`
	State     UtteranceState `json:"state"`
	Error     string         `json:"error,omitempty"`
}

// Levels are how loud the tracks are, from 0 to 1.
type Levels struct {
	Mic    float64 `json:"mic"`
	System float64 `json:"system"`
}

// Fault is something that went wrong while recording goes on.
type Fault struct {
	Track   store.Track `json:"track"`
	Message string      `json:"message"`
}

// Options say what to record.
type Options struct {
	Title      string          `json:"title"`
	Microphone bool            `json:"microphone"`
	System     bool            `json:"system"`
	Apps       []config.AppRef `json:"apps"`
}

// Snapshot is the recorder's status with the utterances that are not
// stored yet.
type Snapshot struct {
	Status  Status          `json:"status"`
	Pending []LiveUtterance `json:"pending"`
}

// Sink hears what happens while recording. Its methods are called from
// several goroutines.
type Sink interface {
	Status(Status)
	Utterance(LiveUtterance)
	Levels(Levels)
	Fault(Fault)
}

type Recorder struct {
	Capture  capture.Recorder
	Speech   speech.Engine
	Store    *store.Store
	Settings func() config.Settings
	Sink     Sink
	// Transcriber returns what transcribes with the settings.
	Transcriber func(config.ASRSettings) (asr.Transcriber, error)

	mu      sync.Mutex
	session *session
}

var (
	ErrBusy       = errors.New("已经在录制了")
	ErrIdle       = errors.New("没有在录制")
	ErrNoSource   = errors.New("至少选择一个声音来源")
	errMicDenied  = errors.New("没有麦克风权限，请在系统设置的「隐私与安全性」里允许 MeetKeep 使用麦克风")
	errTapDenied  = errors.New("没有录制系统声音的权限，请在系统设置的「隐私与安全性」里允许 MeetKeep 录制系统音频")
	startTimeout  = 2 * time.Minute
	silentTimeout = 1500 * time.Millisecond
)

type session struct {
	r           *Recorder
	meeting     store.Meeting
	transcriber asr.Transcriber
	cancel      context.CancelFunc
	tracks      []*track
	capturing   sync.WaitGroup
	transcribes sync.WaitGroup
	slots       chan struct{}
	seq         atomic.Int64
	paused      atomic.Bool

	mu      sync.Mutex
	phase   Phase
	elapsed time.Duration
	resumed time.Time
	pending map[int]LiveUtterance
}

func (r *Recorder) Status() Status {
	r.mu.Lock()
	s := r.session
	r.mu.Unlock()
	if s == nil {
		return Status{Phase: Idle, At: time.Now().UnixMilli()}
	}
	return s.status()
}

func (r *Recorder) Snapshot() Snapshot {
	r.mu.Lock()
	s := r.session
	r.mu.Unlock()
	if s == nil {
		return Snapshot{Status: Status{Phase: Idle, At: time.Now().UnixMilli()}, Pending: []LiveUtterance{}}
	}
	s.mu.Lock()
	pending := make([]LiveUtterance, 0, len(s.pending))
	for _, u := range s.pending {
		pending = append(pending, u)
	}
	s.mu.Unlock()
	return Snapshot{Status: s.status(), Pending: pending}
}

func (s *session) status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	meeting := s.meeting
	elapsed := s.elapsed
	if s.phase == Recording {
		elapsed += time.Since(s.resumed)
	}
	return Status{Phase: s.phase, Meeting: &meeting, ElapsedMs: elapsed.Milliseconds(), At: time.Now().UnixMilli()}
}

func newID(now time.Time) string {
	var suffix [2]byte
	_, _ = rand.Read(suffix[:])
	return fmt.Sprintf("%s-%x", now.Format("20060102-150405"), suffix)
}

// Start begins a meeting. It returns once every source delivers audio, or
// the first of them fails.
func (r *Recorder) Start(ctx context.Context, o Options) (store.Meeting, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.session != nil {
		return store.Meeting{}, ErrBusy
	}
	if !o.Microphone && !o.System {
		return store.Meeting{}, ErrNoSource
	}
	settings := r.Settings()
	transcriber, err := r.Transcriber(settings.ASR)
	if err != nil {
		return store.Meeting{}, err
	}

	now := time.Now()
	id := newID(now)
	directory := filepath.Join(config.Expand(settings.Recording.Directory), id)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return store.Meeting{}, err
	}
	title := strings.TrimSpace(o.Title)
	if title == "" {
		title = now.Format("1月2日 15:04") + " 的会议"
	}
	if o.Apps == nil || !o.System {
		o.Apps = []config.AppRef{}
	}
	apps := make([]string, len(o.Apps))
	for i, a := range o.Apps {
		apps[i] = a.ID
	}
	s := &session{
		r: r,
		meeting: store.Meeting{
			ID: id, Title: title, StartedAt: now.UnixMilli(), Status: store.Recording, Directory: directory,
			Microphone: o.Microphone, System: o.System, Apps: o.Apps,
		},
		transcriber: transcriber,
		slots:       make(chan struct{}, 4),
		phase:       Recording,
		pending:     map[int]LiveUtterance{},
	}

	echo := false
	if o.Microphone && o.System {
		switch settings.Recording.EchoCancellation {
		case config.EchoOn:
			echo = true
		case config.EchoAuto:
			private, err := r.Capture.Headphones()
			echo = err == nil && !private
		}
	}

	capturing, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	fail := func(err error) (store.Meeting, error) {
		cancel()
		s.capturing.Wait()
		for _, t := range s.tracks {
			t.close()
		}
		os.RemoveAll(directory)
		return store.Meeting{}, err
	}
	sources := map[store.Track]capture.Source{}
	if o.Microphone {
		sources[store.Mic] = capture.Microphone
	}
	if o.System {
		sources[store.System] = capture.System
	}
	for _, name := range []store.Track{store.Mic, store.System} {
		source, ok := sources[name]
		if !ok {
			continue
		}
		t, err := s.open(name)
		if err != nil {
			return fail(err)
		}
		s.tracks = append(s.tracks, t)
		frames := r.Capture.Frames(capturing, source, capture.Options{Apps: apps, EchoCancellation: echo})
		s.capturing.Add(1)
		go s.capture(capturing, t, frames)
	}
	for _, t := range s.tracks {
		wait := startTimeout
		if t.name == store.System {
			wait = silentTimeout
		}
		select {
		case err := <-t.ready:
			if err != nil {
				return fail(t.explain(err))
			}
		case <-time.After(wait):
			if t.name == store.Mic {
				return fail(errors.New("麦克风没有声音进来"))
			}
		case <-ctx.Done():
			return fail(ctx.Err())
		}
	}

	if err := r.Store.CreateMeeting(ctx, s.meeting); err != nil {
		return fail(err)
	}
	s.resumed = time.Now()
	r.session = s
	s.capturing.Add(1)
	go s.meter(capturing)
	r.Sink.Status(s.status())
	return s.meeting, nil
}

func (r *Recorder) active() (*session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.session == nil {
		return nil, ErrIdle
	}
	return r.session, nil
}

// Rename gives the meeting being recorded another title, if it is the one
// id names.
func (r *Recorder) Rename(id, title string) {
	r.mu.Lock()
	s := r.session
	r.mu.Unlock()
	if s == nil {
		return
	}
	s.mu.Lock()
	renamed := s.meeting.ID == id
	if renamed {
		s.meeting.Title = title
	}
	s.mu.Unlock()
	if renamed {
		r.Sink.Status(s.status())
	}
}

func (r *Recorder) Pause() error {
	s, err := r.active()
	if err != nil {
		return err
	}
	s.mu.Lock()
	changed := s.phase == Recording
	if changed {
		s.elapsed += time.Since(s.resumed)
		s.phase = Paused
		s.paused.Store(true)
	}
	s.mu.Unlock()
	if changed {
		r.Sink.Status(s.status())
	}
	return nil
}

func (r *Recorder) Resume() error {
	s, err := r.active()
	if err != nil {
		return err
	}
	s.mu.Lock()
	changed := s.phase == Paused
	if changed {
		s.resumed = time.Now()
		s.phase = Recording
		s.paused.Store(false)
	}
	s.mu.Unlock()
	if changed {
		r.Sink.Status(s.status())
	}
	return nil
}

// Stop ends the meeting. It waits up to grace for the utterances still
// being transcribed.
func (r *Recorder) Stop(grace time.Duration) (store.Meeting, error) {
	s, err := r.active()
	if err != nil {
		return store.Meeting{}, err
	}
	s.mu.Lock()
	if s.phase == Stopping {
		s.mu.Unlock()
		return store.Meeting{}, ErrIdle
	}
	if s.phase == Recording {
		s.elapsed += time.Since(s.resumed)
	}
	s.phase = Stopping
	s.mu.Unlock()
	r.Sink.Status(s.status())

	s.cancel()
	s.capturing.Wait()
	done := make(chan struct{})
	go func() {
		s.transcribes.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(grace):
	}
	for _, t := range s.tracks {
		t.close()
	}

	s.mu.Lock()
	s.meeting.Status = store.Done
	s.meeting.DurationMs = s.elapsed.Milliseconds()
	meeting := s.meeting
	s.mu.Unlock()
	err = r.Store.FinishMeeting(context.Background(), meeting.ID, meeting.DurationMs)

	r.mu.Lock()
	r.session = nil
	r.mu.Unlock()
	r.Sink.Status(Status{Phase: Idle, At: time.Now().UnixMilli()})
	return meeting, err
}

func (s *session) meter(ctx context.Context) {
	defer s.capturing.Done()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	var shown Levels
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		var now Levels
		if !s.paused.Load() {
			for _, t := range s.tracks {
				level := float64(t.level.Load()) / 1000
				if t.name == store.Mic {
					now.Mic = level
				} else {
					now.System = level
				}
			}
		}
		shown = Levels{Mic: max(now.Mic, shown.Mic*0.6), System: max(now.System, shown.System*0.6)}
		s.r.Sink.Levels(shown)
	}
}

func (s *session) emit(u LiveUtterance) {
	s.mu.Lock()
	if u.State == Transcribed || u.State == Dropped {
		delete(s.pending, u.Seq)
	} else {
		s.pending[u.Seq] = u
	}
	s.mu.Unlock()
	s.r.Sink.Utterance(u)
}

func (s *session) transcribe(u LiveUtterance, pcm []int16) {
	defer s.transcribes.Done()
	s.slots <- struct{}{}
	defer func() { <-s.slots }()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	text, err := s.transcriber.Transcribe(ctx, pcm)
	if err != nil {
		time.Sleep(time.Second)
		text, err = s.transcriber.Transcribe(ctx, pcm)
	}
	switch {
	case err != nil:
		u.State, u.Error = Failed, err.Error()
	case text == "":
		u.State = Dropped
	default:
		u.State, u.Text = Transcribed, text
		stored := store.Utterance{Seq: u.Seq, Track: u.Track, StartMs: u.StartMs, EndMs: u.EndMs, Text: text}
		if err := s.r.Store.AddUtterance(ctx, u.MeetingID, stored); err != nil {
			s.r.Sink.Fault(Fault{Track: u.Track, Message: err.Error()})
		}
	}
	s.emit(u)
}
