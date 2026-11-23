package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xlsama/meetkeep/internal/export"
	"github.com/xlsama/meetkeep/internal/notes"
	"github.com/xlsama/meetkeep/internal/store"
)

// unnamed tells the name the app gives a speaker nobody has named.
func unnamed(name string) bool {
	number, found := strings.CutPrefix(name, "说话人 ")
	_, err := strconv.Atoi(number)
	return found && err == nil
}

// attendees names who talks in a meeting, and counts those without a name.
func attendees(speakers []store.Speaker) string {
	var names []string
	others := 0
	for _, s := range speakers {
		if unnamed(s.Name) {
			others++
		} else {
			names = append(names, s.Name)
		}
	}
	if others > 0 {
		names = append(names, fmt.Sprintf("%d unnamed", others))
	}
	return strings.Join(names, ", ")
}

func list(ctx context.Context, e *env, args []string) error {
	fs := e.flags("list")
	var when between
	when.flags(fs)
	person := fs.String("person", "", "only the meetings this `name` speaks in")
	limit := fs.Int("limit", 30, "list at most `n` meetings, 0 for all")
	if _, err := e.parse(fs, args); err != nil {
		return err
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
	glances, err := s.Glances(ctx)
	if err != nil {
		return err
	}
	type listed struct {
		summary
		Headline string   `json:"headline"`
		Topics   []string `json:"topics"`
		Speakers []string `json:"speakers"`
		who      string
	}
	found := []listed{}
	for _, m := range meetings {
		glance := glances[m.ID]
		spoke := *person == ""
		item := listed{summary: summarize(m), Topics: []string{}, Speakers: []string{}, who: attendees(glance.Speakers)}
		for _, speaker := range glance.Speakers {
			item.Speakers = append(item.Speakers, speaker.Name)
			spoke = spoke || strings.Contains(strings.ToLower(speaker.Name), strings.ToLower(*person))
		}
		if !spoke || !within.holds(m.StartedAt) {
			continue
		}
		var written notes.Notes
		if json.Unmarshal([]byte(glance.Notes), &written) == nil {
			item.Headline = written.Headline
			for _, topic := range written.Topics {
				item.Topics = append(item.Topics, topic.Title)
			}
		}
		found = append(found, item)
	}
	total := len(found)
	if *limit > 0 && total > *limit {
		found = found[:*limit]
	}
	if e.json {
		return e.print(map[string]any{"total": total, "meetings": found})
	}
	if total == 0 {
		fmt.Fprintln(e.out, "(no meetings)")
	}
	for _, m := range found {
		fmt.Fprintf(e.out, "%s  %s  %d min  %s", m.ID, strings.Replace(m.StartedAt[:16], "T", " ", 1), m.Minutes, m.Title)
		if m.Status != store.Done {
			fmt.Fprintf(e.out, "  [%s]", state(store.Meeting{Status: m.Status, Error: m.Error}))
		}
		fmt.Fprintln(e.out)
		if m.Headline != "" {
			fmt.Fprintf(e.out, "    %s\n", m.Headline)
		}
		if m.who != "" {
			fmt.Fprintf(e.out, "    speakers: %s\n", m.who)
		}
	}
	if total > len(found) {
		fmt.Fprintf(e.out, "%d of %d meetings; --limit 0 lists them all\n", len(found), total)
	}
	return nil
}

func show(ctx context.Context, e *env, args []string) error {
	rest, err := e.parse(e.flags("show"), args, "<meeting>")
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
	speakers, err := s.Speakers(ctx, m.ID)
	if err != nil {
		return err
	}
	kept, err := s.Notes(ctx, m.ID)
	if err != nil {
		return err
	}
	markdown := export.Notes(kept)
	var written notes.Notes
	_ = json.Unmarshal([]byte(kept.Data), &written)
	if len(written.Chapters) > 0 {
		markdown += "\n\n### Timeline\n"
		for i, chapter := range written.Chapters {
			markdown += fmt.Sprintf("\n%d. [%s] %s", i+1, export.Clock(chapter.AtMs), chapter.Title)
			if chapter.Text != "" {
				markdown += ": " + chapter.Text
			}
		}
		markdown = strings.TrimLeft(markdown, "\n")
	}
	if e.json {
		type speaker struct {
			ID      int    `json:"id"`
			Name    string `json:"name"`
			Self    bool   `json:"self"`
			Minutes int64  `json:"minutes"`
		}
		who := make([]speaker, len(speakers))
		for i, sp := range speakers {
			who[i] = speaker{sp.ID, sp.Name, sp.Self, minutes(sp.DurationMs)}
		}
		var data any
		if kept.Data != "" {
			data = json.RawMessage(kept.Data)
		}
		return e.print(map[string]any{
			"meeting": summarize(m), "speakers": who, "markdown": markdown, "notes": data,
			"final": written.Final, "edited": kept.Doc != "", "notesError": kept.Error,
		})
	}
	fmt.Fprintf(e.out, "# %s\n%s · %s · %d min", m.Title, m.ID, day(m.StartedAt), minutes(m.DurationMs))
	if m.Status != store.Done {
		fmt.Fprintf(e.out, " · %s", state(m))
	}
	fmt.Fprintln(e.out)
	if len(speakers) > 0 {
		who := make([]string, len(speakers))
		for i, sp := range speakers {
			who[i] = fmt.Sprintf("%s (%d min)", sp.Name, minutes(sp.DurationMs))
		}
		fmt.Fprintf(e.out, "speakers: %s\n", strings.Join(who, ", "))
	}
	fmt.Fprintln(e.out)
	switch {
	case markdown == "" && kept.Error != "":
		fmt.Fprintf(e.out, "(no notes: %s)\n", kept.Error)
	case markdown == "":
		fmt.Fprintln(e.out, "(no notes yet; meetkeep transcript has what was said)")
	default:
		if !written.Final && kept.Doc == "" {
			fmt.Fprint(e.out, "(the outline kept during the meeting; the final notes replace it once the meeting is processed)\n\n")
		}
		fmt.Fprintln(e.out, markdown)
	}
	return nil
}

func people(ctx context.Context, e *env, args []string) error {
	if _, err := e.parse(e.flags("people"), args); err != nil {
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
	glances, err := s.Glances(ctx)
	if err != nil {
		return err
	}
	type person struct {
		Name     string `json:"name"`
		Self     bool   `json:"self"`
		Meetings int    `json:"meetings"`
		LastAt   string `json:"lastAt"`
		// Remembered tells that the app knows the voice and names it by itself.
		Remembered bool `json:"remembered"`
	}
	byName := map[string]*person{}
	found := []*person{}
	for _, m := range meetings {
		for _, speaker := range glances[m.ID].Speakers {
			if unnamed(speaker.Name) {
				continue
			}
			p := byName[speaker.Name]
			if p == nil {
				p = &person{Name: speaker.Name, LastAt: time.UnixMilli(m.StartedAt).Format(time.RFC3339)}
				byName[speaker.Name] = p
				found = append(found, p)
			}
			p.Meetings++
			p.Self = p.Self || speaker.Self
			p.Remembered = p.Remembered || speaker.PersonID != 0
		}
	}
	sort.SliceStable(found, func(a, b int) bool { return found[a].Meetings > found[b].Meetings })
	if e.json {
		return e.print(map[string]any{"people": found})
	}
	if len(found) == 0 {
		fmt.Fprintln(e.out, "(no named speakers)")
	}
	for _, p := range found {
		fmt.Fprintf(e.out, "%s  meetings: %d  last: %s", p.Name, p.Meetings, p.LastAt[:10])
		if p.Remembered {
			fmt.Fprint(e.out, "  voice remembered")
		}
		fmt.Fprintln(e.out)
	}
	return nil
}

func steps(ctx context.Context, e *env, args []string) error {
	rest, err := e.parse(e.flags("steps"), args, "<meeting>")
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
	taken, err := s.Steps(ctx, m.ID)
	if err != nil {
		return err
	}
	if e.json {
		return e.print(map[string]any{"meeting": summarize(m), "steps": taken})
	}
	if len(taken) == 0 {
		fmt.Fprintln(e.out, "(never processed)")
	}
	var run int64
	for _, step := range taken {
		if step.Run != run {
			run = step.Run
			fmt.Fprintf(e.out, "%s  %s\n", time.UnixMilli(run).Format("2006-01-02 15:04:05"), step.Kind)
		}
		fmt.Fprintf(e.out, "    %s  %.1fs", step.Name, float64(step.DurationMs)/1000)
		if step.MemoryPeak > 0 {
			fmt.Fprintf(e.out, "  %d MB", step.MemoryPeak>>20)
		}
		if step.Detail != "" && step.Detail != "{}" {
			fmt.Fprintf(e.out, "  %s", step.Detail)
		}
		if step.Error != "" {
			fmt.Fprintf(e.out, "  failed: %s", step.Error)
		}
		fmt.Fprintln(e.out)
	}
	return nil
}
