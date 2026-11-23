// Command meetkeep reads the meetings MeetKeep keeps, for agents and
// scripts: it prints text to read, or JSON with --json. It only reads; what
// changes the meetings it asks the running app to do.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/xlsama/meetkeep/internal/config"
	"github.com/xlsama/meetkeep/internal/store"
	"github.com/xlsama/meetkeep/skills"
)

type command struct {
	name    string
	args    string
	summary string
	run     func(ctx context.Context, e *env, args []string) error
}

var commands []command

func init() {
	commands = []command{
		{"list", "[--date DAY | --from TIME --to TIME] [--person NAME] [--limit N]", "List meetings, newest first", list},
		{"show", "<meeting>", "Notes of a meeting: conclusions, todos, open questions, topics, timeline", show},
		{"transcript", "<meeting> [--from 12:00 --to 20:00 | --around 12:30 | --chapter N] [--speaker NAME]", "Transcript of a meeting, or of a stretch of it", transcript},
		{"search", "<text> [--in title,notes,transcript] [--date DAY | --from TIME --to TIME]", "Find text in titles, notes and transcripts", find},
		{"todos", "[--date DAY | --from TIME --to TIME] [--owner NAME] [--all]", "Todos (action items) across meetings", todos},
		{"people", "", "People who spoke in meetings", people},
		{"import", "<file> [--wait]", "Hand an audio or video file to the running MeetKeep app, which makes a meeting of it", importFile},
		{"wait", "<meeting> [--timeout 10m]", "Wait until a meeting is processed", wait},
		{"status", "[<meeting>]", "How far processing a meeting got; without one, the meetings that are not done", status},
		{"steps", "<meeting>", "Time and memory each processing step of a meeting took", steps},
		{"skill", "show", "Print the skill that tells an agent how to use this command", skill},
	}
}

// env is what a command works with.
type env struct {
	out   io.Writer
	now   time.Time
	json  bool
	store *store.Store
}

type usageError string

func (u usageError) Error() string { return string(u) }

func usage(format string, args ...any) error { return usageError(fmt.Sprintf(format, args...)) }

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr, time.Now()))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer, now time.Time) int {
	if len(args) == 0 || slices.Contains([]string{"help", "--help", "-h"}, args[0]) {
		help(stdout)
		return 0
	}
	err := usage("unknown command %q", args[0])
	for _, c := range commands {
		if c.name == args[0] {
			e := &env{out: stdout, now: now}
			err = c.run(ctx, e, args[1:])
			if e.store != nil {
				_ = e.store.Close()
			}
		}
	}
	var wrong usageError
	switch {
	case err == nil || errors.Is(err, flag.ErrHelp):
		return 0
	case slices.Contains(args, "--json") || slices.Contains(args, "-json"):
		_ = json.NewEncoder(stderr).Encode(map[string]string{"error": err.Error()})
	case errors.As(err, &wrong):
		fmt.Fprintf(stderr, "meetkeep: %s\nRun meetkeep help for usage.\n", err)
	default:
		fmt.Fprintf(stderr, "meetkeep: %s\n", err)
	}
	if errors.As(err, &wrong) {
		return 2
	}
	return 1
}

func help(w io.Writer) {
	fmt.Fprint(w, "meetkeep reads the meetings MeetKeep keeps and hands it recordings to process.\nIt prints text; --json prints JSON instead.\n\n")
	for _, c := range commands {
		fmt.Fprintf(w, "  meetkeep %s\n      %s\n", strings.TrimSpace(c.name+" "+c.args), c.summary)
	}
	fmt.Fprint(w, `
<meeting> is an ID, the start of one, or latest (the newest meeting).
Dates and times: today, yesterday, 2026-11-20, 2026-11-20T15:00, -7d (the start of the day 7 days ago), -3h, -30m.
A --to that names a day includes that day.
Moments of a meeting: 12:30, 1:02:03, or seconds.
Run meetkeep <command> --help for the flags of a command.
`)
}

// flags returns the flags every command of that name takes.
func (e *env) flags(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.BoolVar(&e.json, "json", false, "print JSON")
	return fs
}

// parse reads the flags wherever they stand among args and returns the
// rest, which have to be as many as words names; a word in brackets may
// be left out.
func (e *env) parse(fs *flag.FlagSet, args []string, words ...string) ([]string, error) {
	var rest []string
	for {
		if err := fs.Parse(args); errors.Is(err, flag.ErrHelp) {
			for _, c := range commands {
				if c.name == fs.Name() {
					fmt.Fprintf(e.out, "Usage: meetkeep %s\n%s\n\n", strings.TrimSpace(c.name+" "+c.args), c.summary)
				}
			}
			fs.SetOutput(e.out)
			fs.PrintDefaults()
			return nil, err
		} else if err != nil {
			return nil, usageError(err.Error())
		}
		if fs.NArg() == 0 {
			break
		}
		rest, args = append(rest, fs.Arg(0)), fs.Args()[1:]
	}
	required := 0
	for _, word := range words {
		if !strings.HasPrefix(word, "[") {
			required++
		}
	}
	switch {
	case len(rest) > len(words):
		return nil, usage("%s takes no %q", fs.Name(), rest[len(words)])
	case len(rest) < required:
		return nil, usage("%s needs %s", fs.Name(), strings.Join(words, " "))
	}
	return rest, nil
}

// open opens the app's database, to read.
func (e *env) open(ctx context.Context) (*store.Store, error) {
	path := filepath.Join(config.Dir(), "meetkeep.db")
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("no meetings yet: %s does not exist", config.Contract(path))
	}
	s, err := store.OpenReadOnly(ctx, path)
	e.store = s
	return s, err
}

// meeting finds the meeting ref names: its ID, the start of it, or latest.
func (e *env) meeting(ctx context.Context, s *store.Store, ref string) (store.Meeting, error) {
	meetings, err := s.Meetings(ctx)
	if err != nil {
		return store.Meeting{}, err
	}
	var found []store.Meeting
	for _, m := range meetings {
		if m.ID == ref || ref == "latest" {
			return m, nil
		}
		if strings.HasPrefix(m.ID, ref) {
			found = append(found, m)
		}
	}
	switch len(found) {
	case 0:
		return store.Meeting{}, fmt.Errorf("no meeting %q: meetkeep list shows the IDs", ref)
	case 1:
		return found[0], nil
	}
	ids := make([]string, len(found))
	for i, m := range found {
		ids[i] = m.ID
	}
	return store.Meeting{}, fmt.Errorf("%q matches %d meetings: %s", ref, len(found), strings.Join(ids, ", "))
}

// print writes v as a line of JSON.
func (e *env) print(v any) error {
	encoder := json.NewEncoder(e.out)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(v)
}

// summary is a meeting as the JSON of every command has it.
type summary struct {
	ID        string              `json:"id"`
	Title     string              `json:"title"`
	StartedAt string              `json:"startedAt"`
	Minutes   int64               `json:"minutes"`
	Status    store.MeetingStatus `json:"status"`
	Error     string              `json:"error,omitempty"`
}

func summarize(m store.Meeting) summary {
	return summary{
		ID: m.ID, Title: m.Title, StartedAt: time.UnixMilli(m.StartedAt).Format(time.RFC3339),
		Minutes: minutes(m.DurationMs), Status: m.Status, Error: m.Error,
	}
}

func minutes(ms int64) int64 { return max(1, (ms+30000)/60000) }

func day(ms int64) string { return time.UnixMilli(ms).Format("2006-01-02 15:04") }

// state says in words what a meeting that is not done is at.
func state(m store.Meeting) string {
	if m.Status == store.Failed {
		return "failed: " + m.Error
	}
	return string(m.Status)
}

func skill(_ context.Context, e *env, args []string) error {
	if rest, err := e.parse(e.flags("skill"), args, "show"); err != nil {
		return err
	} else if rest[0] != "show" {
		return usage("skill needs show")
	}
	_, err := io.WriteString(e.out, skills.MeetKeep)
	return err
}
