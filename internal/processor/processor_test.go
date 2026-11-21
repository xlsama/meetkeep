package processor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xlsama/meetkeep/internal/asr"
	"github.com/xlsama/meetkeep/internal/config"
	"github.com/xlsama/meetkeep/internal/store"
	"github.com/xlsama/meetkeep/modules/speech"
)

func TestAssign(t *testing.T) {
	segments := []speech.Segment{
		{StartMs: 0, EndMs: 4000, Speaker: 0},
		{StartMs: 4000, EndMs: 6000, Speaker: 1},
		{StartMs: 9000, EndMs: 12000, Speaker: 0},
	}
	phrases := []asr.Phrase{
		{StartMs: 500, EndMs: 3500},
		{StartMs: 3000, EndMs: 5000},
		{StartMs: 4200, EndMs: 5900},
		{StartMs: 6500, EndMs: 7000},
		{StartMs: 8200, EndMs: 8800},
	}
	speakers, uncertain := assign(phrases, segments)
	for i, want := range []struct {
		speaker   int
		uncertain bool
	}{{0, false}, {0, true}, {1, false}, {1, true}, {0, true}} {
		if speakers[i] != want.speaker || uncertain[i] != want.uncertain {
			t.Errorf("phrase %d: speaker %d, uncertain %v; want %d, %v", i, speakers[i], uncertain[i], want.speaker, want.uncertain)
		}
	}
}

type fakeEngine struct {
	speech.Engine
	mixed    []string
	segments []speech.Segment
	counts   []int
}

func (e *fakeEngine) Mix(_ context.Context, inputs []string, output string) error {
	e.mixed = inputs
	return os.WriteFile(output, []byte("mixed"), 0o644)
}

func (e *fakeEngine) Import(_ context.Context, input, output string) (int, error) {
	if filepath.Ext(input) == ".txt" {
		return 0, errors.New("no audio")
	}
	return 90000, os.WriteFile(output, []byte("imported"), 0o644)
}

func (e *fakeEngine) Separate(_ context.Context, _ string, _ speech.Models, count int) ([]speech.Segment, error) {
	e.counts = append(e.counts, count)
	if count == 1 {
		return []speech.Segment{{StartMs: 0, EndMs: 60000, Speaker: 0}}, nil
	}
	return e.segments, nil
}

type fakeTranscriber struct{ fail bool }

func (fakeTranscriber) Transcribe(context.Context, []int16) (string, error) { return "", nil }
func (f fakeTranscriber) TranscribeFile(_ context.Context, path string) ([]asr.Phrase, error) {
	if f.fail {
		return nil, errors.New("offline")
	}
	if filepath.Base(path) == "mic.opus" {
		return []asr.Phrase{{StartMs: 3000, EndMs: 4000, Text: "我来排。"}}, nil
	}
	return []asr.Phrase{
		{StartMs: 500, EndMs: 2500, Text: "今天把日期定下来。"},
		{StartMs: 5000, EndMs: 7000, Text: "十四号可以。"},
		{StartMs: 7800, EndMs: 9000, Text: "价格页我来。"},
	}, nil
}

type harness struct {
	p          *Processor
	db         *store.Store
	engine     *fakeEngine
	changed    chan string
	dir        string
	summarized []string
}

func newHarness(t *testing.T, transcriber fakeTranscriber) *harness {
	t.Helper()
	db, err := store.Open(context.Background(), filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	h := &harness{db: db, changed: make(chan string, 16), dir: t.TempDir()}
	h.engine = &fakeEngine{segments: []speech.Segment{
		{StartMs: 0, EndMs: 3000, Speaker: 0},
		{StartMs: 4800, EndMs: 7200, Speaker: 1},
		{StartMs: 7500, EndMs: 9500, Speaker: 0},
	}}
	h.p = &Processor{
		Speech:      h.engine,
		Store:       db,
		Settings:    func() config.Settings { return config.Settings{} },
		Transcriber: func(config.ASRSettings) (asr.Transcriber, error) { return transcriber, nil },
		Progress:    func(Progress) {},
		Summarize:   func(_ context.Context, id string) { h.summarized = append(h.summarized, id) },
		Changed:     func(id string) { h.changed <- id },
	}
	return h
}

func (h *harness) meeting(t *testing.T, id string, files ...string) {
	t.Helper()
	dir := filepath.Join(h.dir, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("opus"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	m := store.Meeting{ID: id, Title: id, Status: store.Processing, Directory: dir, Apps: []config.AppRef{}}
	if err := h.db.CreateMeeting(context.Background(), m); err != nil {
		t.Fatal(err)
	}
	live := store.Utterance{Seq: 1, Track: store.System, StartMs: 0, EndMs: 900, Text: "会中转写的一句"}
	if err := h.db.AddUtterance(context.Background(), id, live); err != nil {
		t.Fatal(err)
	}
}

func (h *harness) wait(t *testing.T, id string) store.Meeting {
	t.Helper()
	for {
		select {
		case got := <-h.changed:
			m, err := h.db.Meeting(context.Background(), id)
			if err != nil {
				t.Fatal(err)
			}
			if got == id && m.Status != store.Processing {
				return m
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for %s", id)
		}
	}
}

func TestProcessesBothTracks(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t, fakeTranscriber{})
	h.meeting(t, "a", "mic.opus", "system.opus")
	h.p.Process("a")
	m := h.wait(t, "a")
	if m.Status != store.Done || m.Audio != audioFile {
		t.Fatalf("processed as %+v", m)
	}
	if len(h.engine.mixed) != 2 {
		t.Errorf("mixed %v, want both tracks", h.engine.mixed)
	}
	utterances, _ := h.db.Utterances(ctx, "a")
	speakers, _ := h.db.Speakers(ctx, "a")
	if len(speakers) != 3 || !speakers[0].Self || speakers[0].Name != "我" || speakers[1].Name != "说话人 1" {
		t.Fatalf("speakers are %+v", speakers)
	}
	want := []struct {
		track   store.Track
		speaker int
		text    string
	}{
		{store.System, 2, "今天把日期定下来。"},
		{store.Mic, 1, "我来排。"},
		{store.System, 3, "十四号可以。"},
		{store.System, 2, "价格页我来。"},
	}
	if len(utterances) != len(want) {
		t.Fatalf("stored %+v", utterances)
	}
	for i, w := range want {
		u := utterances[i]
		if u.Seq != i+1 || u.Track != w.track || u.Speaker != w.speaker || u.Text != w.text || u.Uncertain {
			t.Errorf("utterance %d is %+v, want %+v", i, u, w)
		}
	}
	if speakers[1].DurationMs != 3200 {
		t.Errorf("the first speaker talks for %d ms, want 3200", speakers[1].DurationMs)
	}
	if _, ok := h.p.State("a"); ok {
		t.Error("the meeting is still being processed")
	}
	if len(h.summarized) != 1 || h.summarized[0] != "a" {
		t.Errorf("notes were asked for %v", h.summarized)
	}

	if err := h.db.SetStatus(ctx, "a", store.Processing, ""); err != nil {
		t.Fatal(err)
	}
	h.p.Separate("a", 1)
	h.wait(t, "a")
	speakers, _ = h.db.Speakers(ctx, "a")
	if len(speakers) != 2 || speakers[1].Name != "对方" {
		t.Errorf("told apart again as %+v, want the user and one other", speakers)
	}
	if got := h.engine.counts; len(got) != 2 || got[1] != 1 {
		t.Errorf("asked for %v speakers", got)
	}
}

func TestProcessesOneTrackAndImports(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t, fakeTranscriber{})
	h.meeting(t, "room", "mic.opus")
	h.p.Process("room")
	if m := h.wait(t, "room"); m.Status != store.Done || m.Audio != "mic.opus" {
		t.Fatalf("processed as %+v", m)
	}
	speakers, _ := h.db.Speakers(ctx, "room")
	if len(speakers) != 1 || speakers[0].Self || speakers[0].Name != "说话人 1" {
		t.Errorf("speakers of a room are %+v", speakers)
	}

	h.meeting(t, "file")
	h.p.Import("file", "/tmp/talk.mp4")
	m := h.wait(t, "file")
	if m.Status != store.Done || m.Audio != audioFile || m.DurationMs != 90000 {
		t.Fatalf("imported as %+v", m)
	}
	utterances, _ := h.db.Utterances(ctx, "file")
	if len(utterances) != 3 || utterances[0].Track != store.File || utterances[1].Speaker != 2 {
		t.Errorf("stored %+v", utterances)
	}

	h.meeting(t, "bad")
	h.p.Import("bad", "/tmp/notes.txt")
	if m := h.wait(t, "bad"); m.Status != store.Failed || m.Error == "" {
		t.Errorf("a text file was imported as %+v", m)
	}
}

func TestFailureKeepsTheLiveTranscript(t *testing.T) {
	h := newHarness(t, fakeTranscriber{fail: true})
	h.meeting(t, "a", "system.opus")
	h.p.Process("a")
	m := h.wait(t, "a")
	if m.Status != store.Failed || m.Error == "" {
		t.Fatalf("processed as %+v", m)
	}
	utterances, _ := h.db.Utterances(context.Background(), "a")
	if len(utterances) != 1 || utterances[0].Text != "会中转写的一句" {
		t.Errorf("the live transcript is now %+v", utterances)
	}
}
