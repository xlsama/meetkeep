package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/xlsama/meetkeep/internal/control"
	"github.com/xlsama/meetkeep/internal/store"
)

// poll is how often wait looks whether a meeting is processed.
var poll = 2 * time.Second

// progress is where a meeting is on its way from being recorded to being
// read.
type progress struct {
	Meeting summary `json:"meeting"`
	// State says it in words.
	State string `json:"state"`
	// Step and Percent tell how far the app got with a meeting it is
	// processing, and Stalled that no app is running to process it.
	Step    string `json:"step,omitempty"`
	Percent int    `json:"percent,omitempty"`
	Stalled bool   `json:"stalled,omitempty"`
	// Notes tells that the meeting has notes, and NotesError why not.
	Notes      bool   `json:"notes"`
	NotesError string `json:"notesError,omitempty"`
	startedAt  int64
}

// follow tells where a meeting is; it asks the app about one that is
// being processed.
func follow(ctx context.Context, s *store.Store, m store.Meeting) (progress, error) {
	p := progress{Meeting: summarize(m), startedAt: m.StartedAt}
	switch m.Status {
	case store.Recording:
		p.State = "recording"
	case store.Failed:
		p.State = "failed: " + m.Error
	case store.Processing:
		answer, err := control.Send(ctx, control.Request{Progress: m.ID})
		switch {
		case errors.Is(err, control.ErrNotRunning):
			p.State, p.Stalled = "stalled: MeetKeep is not running; it goes on processing once it is opened", true
		case err != nil:
			return p, err
		case answer.Working && answer.Step == "queued":
			p.State = "queued"
		case answer.Working:
			p.Step, p.Percent = answer.Step, answer.Percent
			p.State = fmt.Sprintf("processing: %s %d%%", answer.Step, answer.Percent)
		default:
			p.State = "processing"
		}
	default:
		kept, err := s.Notes(ctx, m.ID)
		if err != nil {
			return p, err
		}
		p.Notes, p.NotesError = kept.Data != "" || kept.Doc != "", kept.Error
		switch {
		case p.NotesError != "":
			p.State = "done, the notes failed: " + p.NotesError
		case p.Notes:
			p.State = "done, with notes"
		default:
			p.State = "done, without notes"
		}
	}
	return p, nil
}

func (e *env) report(p progress) error {
	if e.json {
		return e.print(p)
	}
	_, err := fmt.Fprintf(e.out, "%s  %s  %d min  %s\n%s\n", p.Meeting.ID, day(p.startedAt), p.Meeting.Minutes, p.Meeting.Title, p.State)
	return err
}

func status(ctx context.Context, e *env, args []string) error {
	rest, err := e.parse(e.flags("status"), args, "[<meeting>]")
	if err != nil {
		return err
	}
	s, err := e.open(ctx)
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		m, err := e.meeting(ctx, s, rest[0])
		if err != nil {
			return err
		}
		p, err := follow(ctx, s, m)
		if err != nil {
			return err
		}
		return e.report(p)
	}
	meetings, err := s.Meetings(ctx)
	if err != nil {
		return err
	}
	unfinished := []progress{}
	for _, m := range meetings {
		if m.Status != store.Done {
			p, err := follow(ctx, s, m)
			if err != nil {
				return err
			}
			unfinished = append(unfinished, p)
		}
	}
	if e.json {
		var latest *summary
		if len(meetings) > 0 {
			first := summarize(meetings[0])
			latest = &first
		}
		return e.print(map[string]any{"meetings": len(meetings), "latest": latest, "unfinished": unfinished})
	}
	if len(meetings) == 0 {
		fmt.Fprintln(e.out, "(no meetings)")
		return nil
	}
	fmt.Fprintf(e.out, "%d meetings, the newest: %s  %s  %s\n", len(meetings), meetings[0].ID, day(meetings[0].StartedAt), meetings[0].Title)
	for _, p := range unfinished {
		fmt.Fprintf(e.out, "%s  %s  %s  [%s]\n", p.Meeting.ID, day(p.startedAt), p.Meeting.Title, p.State)
	}
	return nil
}

// await waits until the meeting is processed, for as long as timeout, and
// reports it.
func (e *env) await(ctx context.Context, s *store.Store, id string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		m, err := s.Meeting(ctx, id)
		if err != nil {
			return err
		}
		p, err := follow(ctx, s, m)
		switch {
		case err != nil:
			return err
		case m.Status == store.Done:
			return e.report(p)
		case m.Status == store.Failed || p.Stalled:
			return errors.New(p.State)
		case !time.Now().Before(deadline):
			return fmt.Errorf("not done after %s (%s): run meetkeep wait %s again", timeout, p.State, id)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(min(poll, time.Until(deadline))):
		}
	}
}

func wait(ctx context.Context, e *env, args []string) error {
	fs := e.flags("wait")
	timeout := fs.Duration("timeout", 10*time.Minute, "how long to wait at most")
	rest, err := e.parse(fs, args, "<meeting>")
	if err != nil {
		return err
	}
	s, err := e.open(ctx)
	if err != nil {
		return err
	}
	m, err := e.meeting(ctx, s, rest[0])
	if err != nil {
		return err
	}
	return e.await(ctx, s, m.ID, *timeout)
}

func importFile(ctx context.Context, e *env, args []string) error {
	fs := e.flags("import")
	waits := fs.Bool("wait", false, "wait until the meeting is processed")
	timeout := fs.Duration("timeout", 10*time.Minute, "how long --wait waits at most")
	rest, err := e.parse(fs, args, "<file>")
	if err != nil {
		return err
	}
	path, err := filepath.Abs(rest[0])
	if err != nil {
		return err
	}
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		return fmt.Errorf("no file %s", path)
	}
	answer, err := control.Send(ctx, control.Request{Import: path})
	if err != nil {
		return err
	}
	if !*waits {
		if e.json {
			return e.print(map[string]any{"meeting": summarize(answer.Meeting)})
		}
		_, err := fmt.Fprintf(e.out, "%s  processing %s\nIt takes minutes: meetkeep wait %s waits until it is done, meetkeep status %s tells how far it got.\n", answer.Meeting.ID, filepath.Base(path), answer.Meeting.ID, answer.Meeting.ID)
		return err
	}
	s, err := e.open(ctx)
	if err != nil {
		return err
	}
	return e.await(ctx, s, answer.Meeting.ID, *timeout)
}
