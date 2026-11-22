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

	if err := s.Remember(ctx, "a", 2); !errors.Is(err, ErrNoVoice) {
		t.Errorf("a speaker without a voiceprint was remembered: %v", err)
	}
	voiced := []Speaker{{ID: 1, Name: "我", Self: true}, {ID: 2, Name: "周然", Voice: []float32{0.6, 0.8}, SampleStartMs: 100, SampleEndMs: 1900}}
	if err := s.ReplaceTranscript(ctx, "a", final[:2], voiced, "audio.opus"); err != nil {
		t.Fatal(err)
	}
	if err := s.Remember(ctx, "a", 2); err != nil {
		t.Fatal(err)
	}
	people, _ := s.People(ctx)
	if len(people) != 1 || people[0].Name != "周然" || people[0].Samples != 1 || people[0].Meetings != 1 || people[0].SampleMeeting != "a" || people[0].SampleEndMs != 1900 {
		t.Fatalf("people are %+v", people)
	}
	voices, names, _ := s.Voices(ctx)
	if got := voices[people[0].ID]; len(got) != 1 || got[0][1] != 0.8 || names[people[0].ID] != "周然" {
		t.Errorf("voices are %v, names %v", voices, names)
	}
	if who, _ = s.Speakers(ctx, "a"); !who[1].Remembered || who[1].PersonID != people[0].ID || who[0].Remembered {
		t.Errorf("speakers are %+v", who)
	}
	if err := s.RenamePerson(ctx, people[0].ID, "周老师"); err != nil {
		t.Fatal(err)
	}
	if who, _ = s.Speakers(ctx, "a"); who[1].Name != "周老师" {
		t.Errorf("the person's new name did not reach the meeting: %+v", who)
	}
	again := []Speaker{{ID: 1, Name: "周老师", PersonID: people[0].ID, Voice: []float32{0.6, 0.8}}}
	if err := s.ReplaceTranscript(ctx, "a", final[:1], again, "audio.opus"); err != nil {
		t.Fatal(err)
	}
	if who, _ = s.Speakers(ctx, "a"); len(who) != 1 || !who[0].Remembered {
		t.Errorf("processing the meeting again lost the sample: %+v", who)
	}
	if err := s.Forget(ctx, "a", 1); err != nil {
		t.Fatal(err)
	}
	if people, _ = s.People(ctx); len(people) != 0 {
		t.Errorf("a person without samples is kept: %+v", people)
	}
	if who, _ = s.Speakers(ctx, "a"); who[0].PersonID != 0 || who[0].Name != "周老师" {
		t.Errorf("forgetting the voice changed the speaker to %+v", who[0])
	}
	if err := s.ReplaceTranscript(ctx, "a", final, speakers, "audio.opus"); err != nil {
		t.Fatal(err)
	}
	if err := s.RenameSpeaker(ctx, "a", 2, "周然"); err != nil {
		t.Fatal(err)
	}
	if n, _, _ := s.Self(ctx); n != 0 {
		t.Errorf("the user was heard in %d meetings", n)
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

	if kept, err := s.Notes(ctx, "a"); kept.Data != "" || kept.Error != "" || err != nil {
		t.Errorf("a meeting without notes has %+v, %v", kept, err)
	}
	if err := s.SaveNotes(ctx, "a", `{"headline":"3 月 14 号上线"}`, "3 月 14 号上线\n价格页"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetNotesError(ctx, "a", "offline"); err != nil {
		t.Fatal(err)
	}
	if kept, _ := s.Notes(ctx, "a"); kept.Data == "" || kept.Error != "offline" || kept.Replaced {
		t.Errorf("a failure dropped the notes: %+v", kept)
	}
	if found, _ := s.SearchNotes(ctx, "价格页"); found["a"].Data == "" {
		t.Errorf("the notes were not found: %v", found)
	}
	if err := s.SaveNotesDoc(ctx, "a", `{"type":"doc"}`, "我改过的纪要\n首页文案我来写"); err != nil {
		t.Fatal(err)
	}
	if found, _ := s.SearchNotes(ctx, "首页文案"); found["a"].Doc == "" {
		t.Errorf("the edited notes were not found: %v", found)
	}
	if found, _ := s.SearchNotes(ctx, "价格页"); len(found) != 0 {
		t.Errorf("words the user removed are still found: %v", found)
	}
	if err := s.ReplaceNotes(ctx, "a", `{"headline":"重新写的"}`, "重新写的"); err != nil {
		t.Fatal(err)
	}
	if kept, _ := s.Notes(ctx, "a"); kept.Doc != "" || kept.Data != `{"headline":"重新写的"}` || !kept.Replaced {
		t.Errorf("replaced as %+v", kept)
	}
	if err := s.RestoreNotes(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if kept, _ := s.Notes(ctx, "a"); kept.Doc != `{"type":"doc"}` || kept.Text != "我改过的纪要\n首页文案我来写" || !kept.Replaced {
		t.Errorf("restored as %+v", kept)
	}
	if err := s.SaveNotesDoc(ctx, "none", "{}", ""); !errors.Is(err, ErrNotFound) {
		t.Errorf("a document was saved for a meeting without notes: %v", err)
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

func TestStepsGoWithTheirMeeting(t *testing.T) {
	ctx := context.Background()
	s := open(t)
	if err := s.CreateMeeting(ctx, Meeting{ID: "a", Title: "周会", StartedAt: 1000, Status: Processing, Directory: "/tmp/a", Apps: []config.AppRef{}}); err != nil {
		t.Fatal(err)
	}
	first := Step{Run: 5000, Kind: "import", Name: "transcribe", StartedAt: 5200, DurationMs: 30100, MemoryPeak: 334 << 20, Detail: `{"sentences":848}`}
	failed := Step{Run: 5000, Kind: "import", Name: "speakers", StartedAt: 35300, DurationMs: 12, Error: "区分说话人失败", Detail: "{}"}
	for _, step := range []Step{first, failed} {
		if err := s.AddStep(ctx, "a", step); err != nil {
			t.Fatal(err)
		}
	}
	steps, err := s.Steps(ctx, "a")
	if err != nil || len(steps) != 2 || steps[0] != first || steps[1] != failed {
		t.Fatalf("the steps read back as %+v, %v", steps, err)
	}
	if err := s.Delete(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if steps, _ := s.Steps(ctx, "a"); len(steps) != 0 {
		t.Errorf("the steps of a deleted meeting are still there: %+v", steps)
	}
}
