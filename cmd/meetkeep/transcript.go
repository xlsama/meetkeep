package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/xlsama/meetkeep/internal/export"
	"github.com/xlsama/meetkeep/internal/notes"
	"github.com/xlsama/meetkeep/internal/store"
)

// paragraph is how long one speaker goes on before the transcript marks
// the time again.
const paragraph = time.Minute

func transcript(ctx context.Context, e *env, args []string) error {
	fs := e.flags("transcript")
	from := fs.String("from", "", "from this `moment` of the meeting, such as 12:00")
	to := fs.String("to", "", "up to this `moment` of the meeting")
	around := fs.String("around", "", "--window before and after this `moment`")
	window := fs.Duration("window", 2*time.Minute, "how far --around reaches each way")
	chapter := fs.Int("chapter", 0, "only chapter `n` of the timeline that meetkeep show prints, counting from 1")
	speaker := fs.String("speaker", "", "only what this `name` says")
	format := fs.String("format", "text", "text, markdown or srt")
	raw := fs.Bool("raw", false, "the words as they were said, fillers included, not the polished text")
	rest, err := e.parse(fs, args, "<meeting>")
	if err != nil {
		return err
	}
	if !map[string]bool{"text": true, "markdown": true, "srt": true}[*format] {
		return usage("--format is text, markdown or srt")
	}
	s, err := e.open(ctx)
	if err != nil {
		return err
	}
	m, err := e.meeting(ctx, s, rest[0])
	if err != nil {
		return err
	}
	var start, end int64 = 0, -1
	switch {
	case *chapter != 0:
		kept, err := s.Notes(ctx, m.ID)
		if err != nil {
			return err
		}
		var written notes.Notes
		_ = json.Unmarshal([]byte(kept.Data), &written)
		if *chapter < 1 || *chapter > len(written.Chapters) {
			return fmt.Errorf("the timeline of this meeting has %d chapters, no chapter %d", len(written.Chapters), *chapter)
		}
		start = written.Chapters[*chapter-1].AtMs
		if *chapter < len(written.Chapters) {
			end = written.Chapters[*chapter].AtMs
		}
	case *around != "":
		at, err := offset(*around)
		if err != nil {
			return err
		}
		start, end = max(0, at-window.Milliseconds()), at+window.Milliseconds()
	default:
		if *from != "" {
			if start, err = offset(*from); err != nil {
				return err
			}
		}
		if *to != "" {
			if end, err = offset(*to); err != nil {
				return err
			}
		}
	}
	utterances, err := s.Utterances(ctx, m.ID)
	if err != nil {
		return err
	}
	speakers, err := s.Speakers(ctx, m.ID)
	if err != nil {
		return err
	}
	if !*raw && *format != "srt" {
		utterances = store.Shown(utterances)
	}
	said := utterances[:0:0]
	for _, u := range utterances {
		name := strings.ToLower(export.Name(u, speakers))
		if u.StartMs >= start && (end < 0 || u.StartMs < end) && strings.Contains(name, strings.ToLower(*speaker)) {
			said = append(said, u)
		}
	}
	if e.json {
		type line struct {
			Seq       int    `json:"seq"`
			At        string `json:"at"`
			StartMs   int64  `json:"startMs"`
			EndMs     int64  `json:"endMs"`
			Speaker   string `json:"speaker"`
			Uncertain bool   `json:"uncertain,omitempty"`
			Text      string `json:"text"`
		}
		lines := make([]line, len(said))
		for i, u := range said {
			lines[i] = line{u.Seq, export.Clock(u.StartMs), u.StartMs, u.EndMs, export.Name(u, speakers), u.Uncertain, u.Text}
		}
		return e.print(map[string]any{"meeting": summarize(m), "utterances": lines})
	}
	if *format != "text" {
		_, err := fmt.Fprint(e.out, export.Render(m, said, speakers, export.Format(*format)))
		return err
	}
	fmt.Fprintf(e.out, "%s · %s · %d min", m.Title, day(m.StartedAt), minutes(m.DurationMs))
	if m.Status != store.Done {
		fmt.Fprintf(e.out, " · %s, the transcript is the draft kept during the meeting", state(m))
	}
	fmt.Fprintln(e.out)
	if len(said) == 0 {
		fmt.Fprintln(e.out, "\n(nothing said)")
	}
	last, since := "", int64(0)
	for _, u := range said {
		name := export.Name(u, speakers)
		if u.Uncertain {
			name += " (uncertain)"
		}
		if name != last || u.StartMs-since >= paragraph.Milliseconds() {
			fmt.Fprintf(e.out, "\n[%s] %s: ", export.Clock(u.StartMs), name)
			last, since = name, u.StartMs
		}
		fmt.Fprint(e.out, u.Text)
	}
	fmt.Fprintln(e.out)
	return nil
}
