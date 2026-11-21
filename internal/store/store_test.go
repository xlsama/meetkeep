package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/xlsama/meetkeep/internal/config"
)

func open(t *testing.T) *Store {
	t.Helper()
	s, err := Open(context.Background(), filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestMeetingsAndUtterances(t *testing.T) {
	ctx := context.Background()
	s := open(t)
	first := Meeting{ID: "a", Title: "周会", StartedAt: 1000, Status: Recording, Directory: "/tmp/a", Microphone: true, Apps: []config.AppRef{}}
	second := Meeting{ID: "b", Title: "访谈", StartedAt: 2000, Status: Recording, Directory: "/tmp/b", System: true, Apps: []config.AppRef{{ID: "us.zoom.xos", Name: "Zoom"}}}
	for _, m := range []Meeting{first, second} {
		if err := s.CreateMeeting(ctx, m); err != nil {
			t.Fatal(err)
		}
	}
	for _, u := range []Utterance{
		{Seq: 2, Track: System, StartMs: 900, EndMs: 2400, Text: "好的"},
		{Seq: 1, Track: Mic, StartMs: 0, EndMs: 800, Text: "开始吧"},
	} {
		if err := s.AddUtterance(ctx, "a", u); err != nil {
			t.Fatal(err)
		}
	}

	if err := s.FinishMeeting(ctx, "b", 5000); err != nil {
		t.Fatal(err)
	}
	if err := s.Recover(ctx); err != nil {
		t.Fatal(err)
	}
	meetings, err := s.Meetings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(meetings) != 2 || meetings[0].ID != "b" || meetings[1].ID != "a" {
		t.Fatalf("got %+v, want the newest first", meetings)
	}
	if got := meetings[0]; got.DurationMs != 5000 || got.Status != Done || !got.System || got.Apps[0].Name != "Zoom" {
		t.Errorf("got %+v", got)
	}
	if got := meetings[1]; got.Status != Done || got.DurationMs != 2400 {
		t.Errorf("an interrupted meeting was recovered as %+v", got)
	}

	utterances, err := s.Utterances(ctx, "a")
	if err != nil {
		t.Fatal(err)
	}
	if len(utterances) != 2 || utterances[0].Text != "开始吧" || utterances[1].Track != System {
		t.Errorf("got %+v, want them in the order they were said", utterances)
	}

	if err := s.Rename(ctx, "a", "周会 · 复盘"); err != nil {
		t.Fatal(err)
	}
	if m, _ := s.Meeting(ctx, "a"); m.Title != "周会 · 复盘" {
		t.Errorf("renamed to %q", m.Title)
	}
	if err := s.Delete(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if left, _ := s.Utterances(ctx, "a"); len(left) != 0 {
		t.Errorf("%d utterances outlived their meeting", len(left))
	}
	if _, err := s.Meeting(ctx, "a"); !errors.Is(err, ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
	if err := s.Rename(ctx, "a", "x"); !errors.Is(err, ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

func TestReopenKeepsData(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CreateMeeting(ctx, Meeting{ID: "a", Title: "周会", Status: Done, Apps: []config.AppRef{}}); err != nil {
		t.Fatal(err)
	}
	s.Close()
	s, err = Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if meetings, _ := s.Meetings(ctx); len(meetings) != 1 {
		t.Errorf("got %d meetings after reopening", len(meetings))
	}
}
