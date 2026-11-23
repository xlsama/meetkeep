// Package export writes a meeting's transcript as a document.
package export

import (
	"fmt"
	"strings"
	"time"

	"github.com/xlsama/meetkeep/internal/store"
)

type Format string

const (
	Markdown Format = "markdown"
	Text     Format = "text"
	SRT      Format = "srt"
)

// Extension is the file name extension of a format, with its dot.
func (f Format) Extension() string {
	switch f {
	case Text:
		return ".txt"
	case SRT:
		return ".srt"
	}
	return ".md"
}

// Name returns who says an utterance: the speaker it was given, or, before
// the meeting is processed, the side its track stands for.
func Name(u store.Utterance, speakers []store.Speaker) string {
	for _, s := range speakers {
		if s.ID == u.Speaker {
			return s.Name
		}
	}
	if u.Track == store.Mic {
		return "我"
	}
	return "对方"
}

// Clock writes a moment of a meeting as minutes and seconds, with hours
// once it has them.
func Clock(ms int64) string {
	s := ms / 1000
	if s >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", s/3600, s/60%60, s%60)
	}
	return fmt.Sprintf("%02d:%02d", s/60, s%60)
}

func stamp(ms int64) string {
	return fmt.Sprintf("%02d:%02d:%02d,%03d", ms/3600000, ms/60000%60, ms/1000%60, ms%1000)
}

// Render writes the transcript in format.
func Render(m store.Meeting, utterances []store.Utterance, speakers []store.Speaker, format Format) string {
	var b strings.Builder
	switch format {
	case SRT:
		for i, u := range utterances {
			fmt.Fprintf(&b, "%d\n%s --> %s\n%s：%s\n\n", i+1, stamp(u.StartMs), stamp(u.EndMs), Name(u, speakers), u.Text)
		}
	case Text:
		fmt.Fprintf(&b, "%s\n%s\n\n", m.Title, heading(m))
		for _, u := range utterances {
			fmt.Fprintf(&b, "[%s] %s：%s\n", Clock(u.StartMs), Name(u, speakers), u.Text)
		}
	default:
		fmt.Fprintf(&b, "# %s\n\n%s\n", m.Title, heading(m))
		last := ""
		for _, u := range utterances {
			if name := Name(u, speakers); name != last {
				fmt.Fprintf(&b, "\n**%s** %s\n\n", name, Clock(u.StartMs))
				last = name
			}
			b.WriteString(u.Text)
			b.WriteString("\n")
		}
	}
	return b.String()
}

func heading(m store.Meeting) string {
	started := time.UnixMilli(m.StartedAt).Format("2006-01-02 15:04")
	return fmt.Sprintf("%s · %d 分钟", started, max(1, (m.DurationMs+30000)/60000))
}
