package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xlsama/meetkeep/internal/config"
	"github.com/xlsama/meetkeep/internal/control"
	"github.com/xlsama/meetkeep/internal/store"
)

var now = time.Date(2026, 11, 23, 15, 0, 0, 0, time.Local)

func at(day, hour int) int64 {
	return time.Date(2026, 11, day, hour, 0, 0, 0, time.Local).UnixMilli()
}

const weekly = `{"final":true,"headline":"周五上线","gist":"对齐上线前的事","conclusions":[{"text":"周五上线","seq":3,"atMs":70000}],
"topics":[{"title":"上线日期","points":[{"text":"测试周四收尾","seq":2,"atMs":12000}]}],
"todos":[{"text":"写回滚方案","owner":"陈昊","due":"周四","seq":3,"atMs":70000},{"text":"通知客户","owner":"我","seq":4,"atMs":130000}],
"open":[],"chapters":[{"title":"定日期","text":"陈昊问能不能定。","seq":1,"atMs":0},{"title":"分工","text":"各自领事。","seq":3,"atMs":70000}]}`

// fill fills the database of a test with two meetings: a processed
// one, 周会, of today, and one still recording, 访谈, of three days ago.
func fill(t *testing.T) {
	t.Helper()
	// The app's socket is there too, and its path has to be short, which
	// that of t.TempDir is not.
	home, err := os.MkdirTemp("", "mk")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(home) })
	t.Setenv("MEETKEEP_HOME", home)
	ctx := context.Background()
	s, err := store.Open(ctx, filepath.Join(config.Dir(), "meetkeep.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(s.CreateMeeting(ctx, store.Meeting{ID: "20261123-100000-aaaa", Title: "周会", StartedAt: at(23, 10), DurationMs: 30 * 60000, Status: store.Done, Apps: []config.AppRef{}}))
	must(s.CreateMeeting(ctx, store.Meeting{ID: "20261120-090000-bbbb", Title: "访谈", StartedAt: at(20, 9), Status: store.Recording, Apps: []config.AppRef{}}))
	must(s.ReplaceTranscript(ctx, "20261123-100000-aaaa", []store.Utterance{
		{Seq: 1, Track: store.System, StartMs: 1000, EndMs: 5000, Text: "呃今天能不能把上线日期定下来", Polished: "今天能不能把上线日期定下来？", Speaker: 2},
		{Seq: 2, Track: store.System, StartMs: 12000, EndMs: 15000, Text: "测试周四收尾。", Speaker: 2},
		{Seq: 5, Track: store.System, StartMs: 16000, EndMs: 17000, Text: "嗯。", Speaker: 3, Filler: true},
		{Seq: 3, Track: store.Mic, StartMs: 70000, EndMs: 75000, Text: "那就周五上线，回滚方案陈昊写。", Speaker: 1},
		{Seq: 4, Track: store.System, StartMs: 130000, EndMs: 133000, Text: "客户那边你通知。", Speaker: 3, Uncertain: true},
	}, []store.Speaker{{ID: 1, Name: "我", Self: true}, {ID: 2, Name: "陈昊"}, {ID: 3, Name: "说话人 2"}}, "meeting.ogg"))
	must(s.SaveNotes(ctx, "20261123-100000-aaaa", weekly, "周五上线"))
	must(s.AddUtterance(ctx, "20261120-090000-bbbb", store.Utterance{Seq: 1, Track: store.Mic, StartMs: 0, EndMs: 900, Text: "聊聊上线流程"}))
	must(s.AddStep(ctx, "20261123-100000-aaaa", store.Step{Run: at(23, 11), Kind: "process", Name: "transcribe", DurationMs: 3300, MemoryPeak: 120 << 20, Detail: `{"sentences":4}`}))
}

// meetkeep runs the command line and returns what it prints and its exit
// code.
func meetkeep(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), args, &stdout, &stderr, now)
	return stdout.String(), stderr.String(), code
}

func TestCommands(t *testing.T) {
	fill(t)
	for _, c := range []struct {
		args []string
		want []string
		lack []string
	}{
		{
			args: []string{"list"},
			want: []string{
				"20261123-100000-aaaa  2026-11-23 10:00  30 min  周会\n    周五上线\n    speakers: 我, 陈昊, 1 unnamed\n",
				"20261120-090000-bbbb  2026-11-20 09:00  1 min  访谈  [recording]\n",
			},
		},
		{args: []string{"list", "--date", "today"}, want: []string{"周会"}, lack: []string{"访谈"}},
		{args: []string{"list", "--from", "-5d", "--to", "-1d"}, want: []string{"访谈"}, lack: []string{"周会"}},
		{args: []string{"list", "--from", "-3h"}, lack: []string{"周会", "访谈"}, want: []string{"(no meetings)"}},
		{args: []string{"list", "--person", "陈"}, want: []string{"周会"}, lack: []string{"访谈"}},
		{args: []string{"list", "--limit", "1"}, want: []string{"1 of 2 meetings; --limit 0 lists them all"}, lack: []string{"访谈"}},
		{
			args: []string{"show", "latest"},
			want: []string{
				"# 周会\n20261123-100000-aaaa · 2026-11-23 10:00 · 30 min\nspeakers: 我 (1 min), 陈昊 (1 min), 说话人 2 (1 min)\n\n## 周五上线",
				"- [ ] 写回滚方案（陈昊 · 周四） [01:10]", "### Timeline\n\n1. [00:00] 定日期: 陈昊问能不能定。\n2. [01:10] 分工: 各自领事。\n",
			},
		},
		{args: []string{"show", "20261120"}, want: []string{"· recording\n", "(no notes yet; meetkeep transcript has what was said)"}},
		{
			args: []string{"transcript", "latest"},
			want: []string{
				"周会 · 2026-11-23 10:00 · 30 min\n\n[00:01] 陈昊: 今天能不能把上线日期定下来？测试周四收尾。\n[01:10] 我: 那就周五上线，回滚方案陈昊写。\n[02:10] 说话人 2 (uncertain): 客户那边你通知。\n",
			},
			lack: []string{"嗯。"},
		},
		{args: []string{"transcript", "--raw", "latest"}, want: []string{"呃今天能不能把上线日期定下来", "[00:16] 说话人 2: 嗯。"}},
		{args: []string{"transcript", "latest", "--from", "0:10", "--to", "1:30"}, want: []string{"[00:12] 陈昊: 测试周四收尾。", "周五上线"}, lack: []string{"定下来", "客户"}},
		{args: []string{"transcript", "latest", "--around", "70", "--window", "5s"}, want: []string{"周五上线"}, lack: []string{"测试", "客户"}},
		{args: []string{"transcript", "latest", "--chapter", "1"}, want: []string{"定下来", "测试"}, lack: []string{"周五上线"}},
		{args: []string{"transcript", "latest", "--chapter", "2", "--speaker", "我"}, want: []string{"周五上线"}, lack: []string{"客户"}},
		{args: []string{"transcript", "latest", "--format", "srt"}, want: []string{"1\n00:00:01,000 --> 00:00:05,000\n陈昊：呃今天能不能把上线日期定下来\n"}},
		{args: []string{"transcript", "20261120", "--to", "10"}, want: []string{"· recording, the transcript is the draft kept during the meeting\n\n[00:00] 我: 聊聊上线流程\n"}},
		{
			args: []string{"search", "上线"},
			want: []string{
				"20261123-100000-aaaa  2026-11-23 10:00  周会\n    notes 周五上线\n",
				"    notes [01:10] 周五上线\n    notes 上线日期\n", "    [01:10] 我: 那就周五上线，回滚方案陈昊写。\n",
				"20261120-090000-bbbb  2026-11-20 09:00  访谈\n    [00:00] 我: 聊聊上线流程\n",
			},
		},
		{args: []string{"search", "上线", "--in", "transcript", "--date", "2026-11-20"}, want: []string{"访谈"}, lack: []string{"周会", "notes"}},
		{args: []string{"search", "周", "--in", "title"}, want: []string{"周会"}, lack: []string{"notes", "收尾"}},
		{args: []string{"search", "没人说过"}, want: []string{"(nothing found)"}},
		{
			args: []string{"todos"},
			want: []string{"20261123-100000-aaaa  2026-11-23 10:00  周会\n    [ ] 通知客户 (我) [02:10]\n    [ ] 写回滚方案 (陈昊 · 周四) [01:10]\n"},
		},
		{args: []string{"todos", "--owner", "我"}, want: []string{"通知客户"}, lack: []string{"回滚"}},
		{args: []string{"todos", "--date", "yesterday"}, want: []string{"(no todos)"}},
		{args: []string{"people"}, want: []string{"我  meetings: 1  last: 2026-11-23\n", "陈昊  meetings: 1  last: 2026-11-23\n"}, lack: []string{"说话人"}},
		{args: []string{"status"}, want: []string{"2 meetings, the newest: 20261123-100000-aaaa  2026-11-23 10:00  周会\n20261120-090000-bbbb  2026-11-20 09:00  访谈  [recording]\n"}},
		{args: []string{"status", "latest"}, want: []string{"20261123-100000-aaaa  2026-11-23 10:00  30 min  周会\ndone, with notes\n"}},
		{args: []string{"status", "20261120"}, want: []string{"访谈\nrecording\n"}},
		{args: []string{"wait", "latest"}, want: []string{"周会\ndone, with notes\n"}},
		{args: []string{"steps", "latest"}, want: []string{"2026-11-23 11:00:00  process\n    transcribe  3.3s  120 MB  {\"sentences\":4}\n"}},
		{args: []string{"skill", "show"}, want: []string{"name: meetkeep"}},
		{args: []string{"help"}, want: []string{"meetkeep transcript <meeting>"}},
		{args: []string{"transcript", "--help"}, want: []string{"Usage: meetkeep transcript <meeting>", "-chapter n"}},
	} {
		stdout, stderr, code := meetkeep(t, c.args...)
		if code != 0 {
			t.Errorf("meetkeep %s exits %d: %s", strings.Join(c.args, " "), code, stderr)
		}
		for _, want := range c.want {
			if !strings.Contains(stdout, want) {
				t.Errorf("meetkeep %s lacks %q:\n%s", strings.Join(c.args, " "), want, stdout)
			}
		}
		for _, lack := range c.lack {
			if strings.Contains(stdout, lack) {
				t.Errorf("meetkeep %s has %q:\n%s", strings.Join(c.args, " "), lack, stdout)
			}
		}
	}
}

func TestJSON(t *testing.T) {
	fill(t)
	for _, args := range [][]string{
		{"list"}, {"show", "latest"}, {"transcript", "latest"}, {"search", "上线"}, {"todos"}, {"people"}, {"status"}, {"status", "latest"}, {"wait", "latest"},
		{"steps", "latest"},
	} {
		stdout, stderr, code := meetkeep(t, append(args, "--json")...)
		var got map[string]any
		if code != 0 || json.Unmarshal([]byte(stdout), &got) != nil {
			t.Errorf("meetkeep %s --json exits %d with %q and prints:\n%s", strings.Join(args, " "), code, stderr, stdout)
		}
	}

	stdout, _, _ := meetkeep(t, "todos", "--all", "--json")
	var listed struct {
		Todos []struct {
			Meeting struct{ ID, StartedAt string }
			Text    string
			Owner   string
			Done    bool
			At      string
		}
	}
	if err := json.Unmarshal([]byte(stdout), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Todos) != 2 || listed.Todos[1].Owner != "陈昊" || listed.Todos[1].At != "01:10" || listed.Todos[1].Meeting.ID != "20261123-100000-aaaa" {
		t.Errorf("got %+v", listed.Todos)
	}
	if got := listed.Todos[0].Meeting.StartedAt; got != time.UnixMilli(at(23, 10)).Format(time.RFC3339) {
		t.Errorf("the meeting starts at %s", got)
	}

	stdout, _, _ = meetkeep(t, "transcript", "latest", "--speaker", "陈", "--json")
	var said struct {
		Utterances []struct {
			Seq     int
			At      string
			Speaker string
			Text    string
		}
	}
	if err := json.Unmarshal([]byte(stdout), &said); err != nil {
		t.Fatal(err)
	}
	if len(said.Utterances) != 2 || said.Utterances[1].At != "00:12" || said.Utterances[0].Text != "今天能不能把上线日期定下来？" {
		t.Errorf("got %+v", said.Utterances)
	}
}

func TestErrors(t *testing.T) {
	fill(t)
	for _, c := range []struct {
		args []string
		code int
		want string
	}{
		{[]string{"show", "2026"}, 1, `"2026" matches 2 meetings: 20261123-100000-aaaa, 20261120-090000-bbbb`},
		{[]string{"show", "nope"}, 1, `no meeting "nope"`},
		{[]string{"show"}, 2, "show needs <meeting>"},
		{[]string{"people", "陈昊"}, 2, `people takes no "陈昊"`},
		{[]string{"record"}, 2, `unknown command "record"`},
		{[]string{"list", "--bogus"}, 2, "-bogus"},
		{[]string{"list", "--from", "上周"}, 2, `cannot read the time "上周"`},
		{[]string{"list", "--date", "today", "--to", "now"}, 2, "--date goes without --from and --to"},
		{[]string{"transcript", "latest", "--from", "a:b"}, 2, `cannot read the moment "a:b"`},
		{[]string{"transcript", "latest", "--chapter", "3"}, 1, "has 2 chapters, no chapter 3"},
		{[]string{"transcript", "latest", "--format", "docx"}, 2, "--format is text, markdown or srt"},
		{[]string{"search", "上线", "--in", "board"}, 2, "--in is title, notes, transcript"},
		{[]string{"skill"}, 2, "skill needs show"},
		{[]string{"status", "latest", "more"}, 2, `status takes no "more"`},
		{[]string{"import"}, 2, "import needs <file>"},
		{[]string{"import", "/nowhere/a.wav"}, 1, "no file /nowhere/a.wav"},
		{[]string{"wait", "20261120", "--timeout", "30ms"}, 1, "not done after 30ms (recording): run meetkeep wait 20261120-090000-bbbb again"},
	} {
		stdout, stderr, code := meetkeep(t, c.args...)
		if code != c.code || !strings.Contains(stderr, c.want) || stdout != "" {
			t.Errorf("meetkeep %s exits %d and says %q (and prints %q), want %d and %q", strings.Join(c.args, " "), code, stderr, stdout, c.code, c.want)
		}
	}
	_, stderr, code := meetkeep(t, "show", "nope", "--json")
	var failure struct{ Error string }
	if json.Unmarshal([]byte(stderr), &failure) != nil || code != 1 || !strings.Contains(failure.Error, `no meeting "nope"`) {
		t.Errorf("exits %d and says %q", code, stderr)
	}

	t.Setenv("MEETKEEP_HOME", t.TempDir())
	if _, stderr, code := meetkeep(t, "list"); code != 1 || !strings.Contains(stderr, "no meetings yet") {
		t.Errorf("without a database it exits %d and says %q", code, stderr)
	}
}

// TestImport imports a file as the app does it: the test stands in for
// the app, and processes the meeting a while after it is asked for.
func TestImport(t *testing.T) {
	fill(t)
	poll = 10 * time.Millisecond
	ctx := context.Background()
	recording := filepath.Join(t.TempDir(), "访谈录音.wav")
	if err := os.WriteFile(recording, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, stderr, code := meetkeep(t, "import", recording); code != 1 || !strings.Contains(stderr, "MeetKeep is not running") {
		t.Errorf("with no app it exits %d and says %q", code, stderr)
	}

	s, err := store.Open(ctx, filepath.Join(config.Dir(), "meetkeep.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	const id = "20261123-150001-cccc"
	var imported atomic.Int32
	listener, err := control.Listen(func(ctx context.Context, r control.Request) (control.Answer, error) {
		switch {
		case r.Progress != "":
			return control.Answer{Working: true, Step: "transcribe", Percent: 40}, nil
		case filepath.Ext(r.Import) != ".wav":
			return control.Answer{}, errors.New("only audio and video files can be imported")
		}
		m := store.Meeting{
			ID: fmt.Sprintf("20261123-15000%d-cccc", imported.Add(1)), Title: "访谈录音", StartedAt: at(23, 15),
			Status: store.Processing, Apps: []config.AppRef{},
		}
		time.AfterFunc(150*time.Millisecond, func() {
			_ = s.SaveNotes(ctx, m.ID, weekly, "周五上线")
			_ = s.SetStatus(ctx, m.ID, store.Done, "")
		})
		return control.Answer{Meeting: m}, s.CreateMeeting(ctx, m)
	})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	if _, stderr, code := meetkeep(t, "import", filepath.Join(config.Dir(), "meetkeep.db")); code != 1 || !strings.Contains(stderr, "only audio and video files can be imported") {
		t.Errorf("for a file that is no recording it exits %d and says %q", code, stderr)
	}
	stdout, stderr, code := meetkeep(t, "import", recording)
	if code != 0 || !strings.HasPrefix(stdout, id+"  processing 访谈录音.wav\n") {
		t.Fatalf("it exits %d, says %q and prints:\n%s", code, stderr, stdout)
	}
	if stdout, _, _ := meetkeep(t, "status", id); !strings.Contains(stdout, "访谈录音\nprocessing: transcribe 40%\n") {
		t.Errorf("while the app works the status is:\n%s", stdout)
	}
	if _, stderr, code := meetkeep(t, "wait", id, "--timeout", "20ms"); code != 1 || !strings.Contains(stderr, "not done after 20ms (processing: transcribe 40%)") {
		t.Errorf("a short wait exits %d and says %q", code, stderr)
	}
	if stdout, stderr, code := meetkeep(t, "wait", id); code != 0 || !strings.Contains(stdout, "访谈录音\ndone, with notes\n") {
		t.Errorf("the wait exits %d, says %q and prints:\n%s", code, stderr, stdout)
	}
	if stdout, _, code := meetkeep(t, "show", id); code != 0 || !strings.Contains(stdout, "## 周五上线") {
		t.Errorf("the notes of the imported meeting read:\n%s", stdout)
	}

	stdout, stderr, code = meetkeep(t, "import", recording, "--wait", "--json")
	var done struct {
		Meeting struct{ ID, Status string }
		Notes   bool
	}
	if json.Unmarshal([]byte(stdout), &done) != nil || code != 0 || done.Meeting.ID != "20261123-150002-cccc" || done.Meeting.Status != "done" || !done.Notes {
		t.Errorf("import --wait exits %d, says %q and prints:\n%s", code, stderr, stdout)
	}

	listener.Close()
	if err := s.SetStatus(ctx, id, store.Processing, ""); err != nil {
		t.Fatal(err)
	}
	if stdout, _, _ := meetkeep(t, "status", id); !strings.Contains(stdout, "stalled: MeetKeep is not running") {
		t.Errorf("once the app quit the status is:\n%s", stdout)
	}
	if _, stderr, code := meetkeep(t, "wait", id); code != 1 || !strings.Contains(stderr, "stalled") {
		t.Errorf("once the app quit the wait exits %d and says %q", code, stderr)
	}
}

func TestMoment(t *testing.T) {
	for text, want := range map[string]string{
		"now": "2026-11-23T15:00", "today": "2026-11-23T00:00", "yesterday": "2026-11-22T00:00",
		"-7d": "2026-11-16T00:00", "-3h": "2026-11-23T12:00", "-30m": "2026-11-23T14:30",
		"2026-10-20": "2026-10-20T00:00", "2026-10-20T15:04": "2026-10-20T15:04", "2026-10-20 15:04": "2026-10-20T15:04",
		"1795417200000": time.UnixMilli(1795417200000).Format("2006-01-02T15:04"),
	} {
		got, _, err := moment(text, now)
		if err != nil || got.Format("2006-01-02T15:04") != want {
			t.Errorf("%s is %v (%v), want %s", text, got, err, want)
		}
	}
	for _, text := range []string{"", "-", "-d", "7d", "tomorrow", "2026-13-01", "12:00"} {
		if got, _, err := moment(text, now); err == nil {
			t.Errorf("%q is read as %v", text, got)
		}
	}
	for text, want := range map[string]int64{"90": 90000, "12:30": 750000, "1:02:03": 3723000, "0": 0} {
		if got, err := offset(text); err != nil || got != want {
			t.Errorf("%s is %d (%v), want %d", text, got, err, want)
		}
	}
}
