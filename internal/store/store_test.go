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
	pending, err := s.Recover(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 2 || pending[0] != "a" {
		t.Errorf("%v wait to be processed, want both meetings, the oldest first", pending)
	}
	meetings, err := s.Meetings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(meetings) != 2 || meetings[0].ID != "b" || meetings[1].ID != "a" {
		t.Fatalf("got %+v, want the newest first", meetings)
	}
	if got := meetings[0]; got.DurationMs != 5000 || got.Status != Processing || !got.System || got.Apps[0].Name != "Zoom" {
		t.Errorf("got %+v", got)
	}
	if got := meetings[1]; got.Status != Processing || got.DurationMs != 2400 {
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

func TestTranscriptNotesAndBoard(t *testing.T) {
	ctx := context.Background()
	s := open(t)
	if err := s.CreateMeeting(ctx, Meeting{ID: "a", Title: "上线前对齐会", Status: Processing, Apps: []config.AppRef{}}); err != nil {
		t.Fatal(err)
	}
	if err := s.AddUtterance(ctx, "a", Utterance{Seq: 1, Track: System, Text: "会中的一句"}); err != nil {
		t.Fatal(err)
	}
	speakers := []Speaker{{ID: 1, Name: "我", Self: true}, {ID: 2, Name: "说话人 1"}, {ID: 3, Name: "说话人 2"}}
	final := []Utterance{
		{Seq: 1, Track: System, StartMs: 0, EndMs: 2000, Text: "价格页我来，100% 周四给。", Speaker: 2},
		{Seq: 2, Track: Mic, StartMs: 2500, EndMs: 3500, Text: "好。", Speaker: 1},
		{Seq: 3, Track: System, StartMs: 4000, EndMs: 7000, Text: "首页文案呢？", Speaker: 3, Uncertain: true},
	}
	if err := s.ReplaceTranscript(ctx, "a", final, speakers, "audio.opus"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetStatus(ctx, "a", Done, ""); err != nil {
		t.Fatal(err)
	}
	m, _ := s.Meeting(ctx, "a")
	if m.Status != Done || m.Audio != "audio.opus" {
		t.Errorf("got %+v", m)
	}
	stored, _ := s.Utterances(ctx, "a")
	if len(stored) != 3 || stored[0].Speaker != 2 || !stored[2].Uncertain {
		t.Fatalf("stored %+v", stored)
	}

	if err := s.RenameSpeaker(ctx, "a", 2, "周然"); err != nil {
		t.Fatal(err)
	}
	if err := s.MergeSpeakers(ctx, "a", 3, 2); err != nil {
		t.Fatal(err)
	}
	who, _ := s.Speakers(ctx, "a")
	if len(who) != 2 || who[1].Name != "周然" || who[1].DurationMs != 5000 || who[0].DurationMs != 1000 {
		t.Errorf("speakers are %+v", who)
	}

	lines, err := s.SearchLines(ctx, "100%")
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || lines[0].Speaker != "周然" || lines[0].MeetingID != "a" {
		t.Errorf("found %+v", lines)
	}
	if lines, _ := s.SearchLines(ctx, "%"); len(lines) != 1 {
		t.Errorf("a percent sign matched %d lines, want the one that holds it", len(lines))
	}
	if titled, _ := s.SearchTitles(ctx, "对齐"); len(titled) != 1 {
		t.Errorf("found %d meetings by title", len(titled))
	}

	if data, failure, err := s.Notes(ctx, "a"); data != "" || failure != "" || err != nil {
		t.Errorf("a meeting without notes has %q, %q, %v", data, failure, err)
	}
	if err := s.SaveNotes(ctx, "a", `{"headline":"3 月 14 号上线"}`, "3 月 14 号上线\n价格页"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetNotesError(ctx, "a", "offline"); err != nil {
		t.Fatal(err)
	}
	if data, failure, _ := s.Notes(ctx, "a"); data == "" || failure != "offline" {
		t.Errorf("a failure dropped the notes: %q, %q", data, failure)
	}
	if found, _ := s.SearchNotes(ctx, "价格页"); found["a"] == "" {
		t.Errorf("the notes were not found: %v", found)
	}

	if scene, _ := s.Board(ctx, "a"); scene != "" {
		t.Errorf("a meeting without a board has %q", scene)
	}
	for _, scene := range []string{`{"elements":[]}`, `{"elements":[1]}`} {
		if err := s.SaveBoard(ctx, "a", scene); err != nil {
			t.Fatal(err)
		}
	}
	if scene, _ := s.Board(ctx, "a"); scene != `{"elements":[1]}` {
		t.Errorf("the board is %q", scene)
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
