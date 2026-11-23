package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/xlsama/meetkeep/internal/export"
	"github.com/xlsama/meetkeep/internal/library"
	"github.com/xlsama/meetkeep/internal/store"
)

func find(ctx context.Context, e *env, args []string) error {
	fs := e.flags("search")
	var when between
	when.flags(fs)
	in := fs.String("in", "title,notes,transcript", "where to look: title, notes, transcript, separated by commas")
	limit := fs.Int("limit", 8, "list at most `n` transcript lines of a meeting, 0 for all")
	rest, err := e.parse(fs, args, "<text>")
	if err != nil {
		return err
	}
	places := map[string]bool{}
	for place := range strings.SplitSeq(*in, ",") {
		if place != "title" && place != "notes" && place != "transcript" {
			return usage("--in is title, notes, transcript, separated by commas")
		}
		places[place] = true
	}
	within, err := when.span(e.now)
	if err != nil {
		return err
	}
	s, err := e.open(ctx)
	if err != nil {
		return err
	}
	hits, err := library.Search(ctx, s, rest[0])
	if err != nil {
		return err
	}
	type hit struct {
		summary
		TitleMatch bool                 `json:"titleMatch"`
		Notes      []library.SearchNote `json:"notes"`
		Lines      []library.SearchLine `json:"lines"`
		meeting    store.Meeting
	}
	found := []hit{}
	for _, h := range hits {
		item := hit{summary: summarize(h.Meeting), TitleMatch: h.TitleMatch && places["title"], Notes: []library.SearchNote{}, Lines: []library.SearchLine{}, meeting: h.Meeting}
		if places["notes"] {
			item.Notes = h.Notes
		}
		if places["transcript"] {
			item.Lines = h.Lines
		}
		if within.holds(h.Meeting.StartedAt) && (item.TitleMatch || len(item.Notes)+len(item.Lines) > 0) {
			found = append(found, item)
		}
	}
	if e.json {
		return e.print(map[string]any{"hits": found})
	}
	if len(found) == 0 {
		fmt.Fprintln(e.out, "(nothing found)")
	}
	for i, h := range found {
		if i > 0 && len(found[i-1].Notes)+len(found[i-1].Lines) > 0 {
			fmt.Fprintln(e.out)
		}
		fmt.Fprintf(e.out, "%s  %s  %s\n", h.ID, day(h.meeting.StartedAt), h.Title)
		for _, note := range h.Notes {
			if note.AtMs > 0 {
				fmt.Fprintf(e.out, "    notes [%s] %s\n", export.Clock(note.AtMs), note.Text)
			} else {
				fmt.Fprintf(e.out, "    notes %s\n", note.Text)
			}
		}
		for n, line := range h.Lines {
			if *limit > 0 && n == *limit {
				fmt.Fprintf(e.out, "    %d more lines; --limit 0 lists them all\n", len(h.Lines)-n)
				break
			}
			fmt.Fprintf(e.out, "    [%s] %s: %s\n", export.Clock(line.StartMs), line.Speaker, line.Text)
		}
	}
	return nil
}

func todos(ctx context.Context, e *env, args []string) error {
	fs := e.flags("todos")
	var when between
	when.flags(fs)
	owner := fs.String("owner", "", "only the todos of this `name`, such as 我")
	all := fs.Bool("all", false, "the todos checked off too")
	if _, err := e.parse(fs, args); err != nil {
		return err
	}
	if when == (between{}) {
		when.from = "-30d"
	}
	within, err := when.span(e.now)
	if err != nil {
		return err
	}
	s, err := e.open(ctx)
	if err != nil {
		return err
	}
	meetings, err := s.Meetings(ctx)
	if err != nil {
		return err
	}
	type todo struct {
		Meeting summary `json:"meeting"`
		export.Task
		At string `json:"at,omitempty"`
	}
	found := []todo{}
	for _, m := range meetings {
		if !within.holds(m.StartedAt) {
			continue
		}
		kept, err := s.Notes(ctx, m.ID)
		if err != nil {
			return err
		}
		for _, task := range export.Tasks(kept) {
			if task.Done && !*all || !strings.Contains(strings.ToLower(task.Owner), strings.ToLower(*owner)) {
				continue
			}
			item := todo{Meeting: summarize(m), Task: task}
			if task.AtMs > 0 {
				item.At = export.Clock(task.AtMs)
			}
			found = append(found, item)
		}
	}
	if e.json {
		return e.print(map[string]any{"todos": found})
	}
	if len(found) == 0 {
		fmt.Fprintln(e.out, "(no todos)")
	}
	last := ""
	for _, t := range found {
		if t.Meeting.ID != last {
			if last != "" {
				fmt.Fprintln(e.out)
			}
			last = t.Meeting.ID
			fmt.Fprintf(e.out, "%s  %s  %s\n", t.Meeting.ID, strings.Replace(t.Meeting.StartedAt[:16], "T", " ", 1), t.Meeting.Title)
		}
		mark := "[ ]"
		if t.Done {
			mark = "[x]"
		}
		fmt.Fprintf(e.out, "    %s %s", mark, t.Text)
		var label []string
		for _, part := range []string{t.Owner, t.Due} {
			if part != "" {
				label = append(label, part)
			}
		}
		if len(label) > 0 {
			fmt.Fprintf(e.out, " (%s)", strings.Join(label, " · "))
		}
		if t.At != "" {
			fmt.Fprintf(e.out, " [%s]", t.At)
		}
		fmt.Fprintln(e.out)
	}
	return nil
}
