package recorder

import (
	"context"
	"errors"
	"iter"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/xlsama/meetkeep/internal/asr"
	"github.com/xlsama/meetkeep/internal/audio"
	"github.com/xlsama/meetkeep/internal/config"
	"github.com/xlsama/meetkeep/internal/store"
	"github.com/xlsama/meetkeep/modules/capture"
	"github.com/xlsama/meetkeep/modules/speech"
)

// fakeCapture plays a script of loud and silent frames for each source,
// then silence, faster than real time.
type fakeCapture struct {
	script map[capture.Source][]bool
	deny   capture.Source
}

func (f *fakeCapture) Apps() ([]capture.App, error)      { return nil, nil }
func (f *fakeCapture) Listening() ([]capture.App, error) { return nil, nil }
func (f *fakeCapture) Microphone() (string, error)       { return "fake", nil }
func (f *fakeCapture) Headphones() (bool, error)         { return true, nil }

func (f *fakeCapture) Frames(ctx context.Context, source capture.Source, _ capture.Options) iter.Seq2[[]byte, error] {
	return func(yield func([]byte, error) bool) {
		if source == f.deny {
			yield(nil, capture.ErrDenied)
			return
		}
		loud := make([]int16, frameBytes/2)
		for i := range loud {
			loud[i] = 8000
		}
		script := f.script[source]
		for i := 0; ctx.Err() == nil; i++ {
			frame := make([]byte, frameBytes)
			if i < len(script) && script[i] {
				frame = audio.Bytes(loud)
			}
			if !yield(frame, nil) {
				return
			}
			time.Sleep(time.Millisecond)
		}
	}
}

type fakeEngine struct{ speech.Engine }

func (fakeEngine) Detector() (speech.Detector, error) { return &fakeDetector{}, nil }
func (fakeEngine) Record(path string) (speech.Recording, error) {
	f, err := os.Create(path)
	return fakeRecording{f}, err
}

// fakeDetector calls a window speech when it is loud.
type fakeDetector struct{ rest []int16 }

func (d *fakeDetector) Close() error { return nil }
func (d *fakeDetector) Feed(pcm []byte) ([]float32, error) {
	d.rest = append(d.rest, audio.Samples(pcm)...)
	var out []float32
	for ; len(d.rest) >= audio.Window; d.rest = d.rest[audio.Window:] {
		out = append(out, float32(audio.Level(d.rest[:audio.Window])))
	}
	return out, nil
}

type fakeRecording struct{ *os.File }

func (r fakeRecording) Write(pcm []byte) error {
	_, err := r.File.Write(pcm)
	return err
}

type fakeTranscriber struct{}

func (fakeTranscriber) Transcribe(_ context.Context, pcm []int16) (string, error) {
	return "你好", nil
}

func (fakeTranscriber) TranscribeFile(context.Context, string) ([]asr.Phrase, error) {
	return nil, nil
}

type sink struct {
	mu         sync.Mutex
	statuses   []Status
	utterances []LiveUtterance
	faults     []Fault
}

func (s *sink) Status(v Status) { s.mu.Lock(); s.statuses = append(s.statuses, v); s.mu.Unlock() }
func (s *sink) Levels(Levels)   {}
func (s *sink) Fault(v Fault)   { s.mu.Lock(); s.faults = append(s.faults, v); s.mu.Unlock() }
func (s *sink) Utterance(v LiveUtterance) {
	s.mu.Lock()
	s.utterances = append(s.utterances, v)
	s.mu.Unlock()
}

func (s *sink) count(state UtteranceState) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, u := range s.utterances {
		if u.State == state {
			n++
		}
	}
	return n
}

func newRecorder(t *testing.T, c *fakeCapture) (*Recorder, *sink) {
	t.Helper()
	db, err := store.Open(context.Background(), filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	directory := t.TempDir()
	heard := &sink{}
	return &Recorder{
		Capture: c,
		Speech:  fakeEngine{},
		Store:   db,
		Sink:    heard,
		Settings: func() config.Settings {
			return config.Settings{Recording: config.RecordingSettings{Directory: directory}}
		},
		Transcriber: func(config.ASRSettings) (asr.Transcriber, error) { return fakeTranscriber{}, nil },
	}, heard
}

func wait(t *testing.T, what string, done func() bool) {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
		if done() {
			return
		}
	}
	t.Fatalf("timed out waiting for %s", what)
}

func talk(silent, loud, rest int) []bool {
	script := make([]bool, silent+loud+rest)
	for i := silent; i < silent+loud; i++ {
		script[i] = true
	}
	return script
}

func TestRecordsBothTracks(t *testing.T) {
	ctx := context.Background()
	r, heard := newRecorder(t, &fakeCapture{script: map[capture.Source][]bool{
		capture.Microphone: talk(5, 15, 15),
		capture.System:     talk(25, 15, 15),
	}})
	meeting, err := r.Start(ctx, Options{Title: " 周会 ", Microphone: true, System: true})
	if err != nil {
		t.Fatal(err)
	}
	if meeting.Title != "周会" || meeting.Status != store.Recording {
		t.Errorf("started %+v", meeting)
	}
	if _, err := r.Start(ctx, Options{Microphone: true}); !errors.Is(err, ErrBusy) {
		t.Errorf("a second start got %v, want ErrBusy", err)
	}
	wait(t, "both utterances", func() bool { return heard.count(Transcribed) == 2 })

	if err := r.Pause(); err != nil {
		t.Fatal(err)
	}
	if got := r.Status().Phase; got != Paused {
		t.Errorf("phase is %q after pausing", got)
	}
	if err := r.Resume(); err != nil {
		t.Fatal(err)
	}
	done, err := r.Stop(time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if done.Status != store.Processing || done.DurationMs <= 0 {
		t.Errorf("stopped as %+v", done)
	}
	if got := r.Status().Phase; got != Idle {
		t.Errorf("phase is %q after stopping", got)
	}
	if _, err := r.Stop(time.Second); !errors.Is(err, ErrIdle) {
		t.Errorf("a second stop got %v, want ErrIdle", err)
	}

	utterances, err := r.Store.Utterances(ctx, meeting.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(utterances) != 2 || utterances[0].Track != store.Mic || utterances[1].Track != store.System {
		t.Fatalf("stored %+v, want the microphone's utterance then the system's", utterances)
	}
	for _, u := range utterances {
		if d := u.EndMs - u.StartMs; u.Text != "你好" || d < 1200 || d > 2400 {
			t.Errorf("stored %+v, want about 1.5 s of speech", u)
		}
	}
	for _, name := range []string{"mic.opus", "system.opus"} {
		if info, err := os.Stat(filepath.Join(meeting.Directory, name)); err != nil || info.Size() == 0 {
			t.Errorf("%s was not written: %v", name, err)
		}
	}
	if len(heard.faults) != 0 {
		t.Errorf("faults: %+v", heard.faults)
	}
	if n := len(r.Snapshot().Pending); n != 0 {
		t.Errorf("%d utterances are pending after stopping", n)
	}
}

func TestStopFlushesTheOpenUtterance(t *testing.T) {
	r, heard := newRecorder(t, &fakeCapture{script: map[capture.Source][]bool{
		capture.Microphone: talk(5, 100000, 0),
	}})
	meeting, err := r.Start(context.Background(), Options{Microphone: true})
	if err != nil {
		t.Fatal(err)
	}
	wait(t, "speech to begin", func() bool { return heard.count(Speaking) == 1 })
	if pending := r.Snapshot().Pending; len(pending) != 1 || pending[0].State != Speaking {
		t.Errorf("pending is %+v", pending)
	}
	time.Sleep(50 * time.Millisecond)
	if _, err := r.Stop(time.Second); err != nil {
		t.Fatal(err)
	}
	if utterances, _ := r.Store.Utterances(context.Background(), meeting.ID); len(utterances) != 1 {
		t.Errorf("stored %d utterances, want the one cut short by stopping", len(utterances))
	}
}

func TestStartFailsWhenASourceIsDenied(t *testing.T) {
	r, _ := newRecorder(t, &fakeCapture{deny: capture.Microphone})
	_, err := r.Start(context.Background(), Options{Microphone: true, System: true})
	if !errors.Is(err, errMicDenied) {
		t.Fatalf("got %v, want the microphone's permission error", err)
	}
	if got := r.Status().Phase; got != Idle {
		t.Errorf("phase is %q", got)
	}
	if meetings, _ := r.Store.Meetings(context.Background()); len(meetings) != 0 {
		t.Errorf("a meeting was stored: %+v", meetings)
	}
	entries, _ := os.ReadDir(r.Settings().Recording.Directory)
	if len(entries) != 0 {
		t.Errorf("the meeting's directory was left behind")
	}
	if _, err := r.Start(context.Background(), Options{}); !errors.Is(err, ErrNoSource) {
		t.Errorf("got %v, want ErrNoSource", err)
	}
}
