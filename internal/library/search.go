// Package library looks through the meetings that are kept.
package library

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/xlsama/meetkeep/internal/export"
	"github.com/xlsama/meetkeep/internal/notes"
	"github.com/xlsama/meetkeep/internal/store"
)

// SearchLine is an utterance that contains what was searched for.
type SearchLine struct {
	Seq     int   `json:"seq"`
	StartMs int64 `json:"startMs"`
	// Speaker is the name of who says it and SpeakerID their ID in the
	// meeting, 0 before the meeting is processed. Self marks the user.
	Speaker   string `json:"speaker"`
	SpeakerID int    `json:"speakerId"`
	Self      bool   `json:"self"`
	Text      string `json:"text"`
}

// SearchNote is a line of a meeting's notes that contains what was
// searched for.
type SearchNote struct {
	Text string `json:"text"`
	AtMs int64  `json:"atMs"`
}

// SearchHit is a meeting whose title, transcript or notes contain what was
// searched for.
type SearchHit struct {
	Meeting    store.Meeting `json:"meeting"`
	TitleMatch bool          `json:"titleMatch"`
	Lines      []SearchLine  `json:"lines"`
	Notes      []SearchNote  `json:"notes"`
}

func matching(kept store.StoredNotes, query string) []SearchNote {
	query = strings.ToLower(query)
	var found []SearchNote
	if kept.Doc != "" {
		for line := range strings.SplitSeq(kept.Text, "\n") {
			if strings.Contains(strings.ToLower(line), query) {
				found = append(found, SearchNote{Text: strings.TrimSpace(line)})
			}
		}
		return found
	}
	var saved notes.Notes
	if json.Unmarshal([]byte(kept.Data), &saved) != nil {
		return nil
	}
	add := func(text string, at int64) {
		if strings.Contains(strings.ToLower(text), query) {
			found = append(found, SearchNote{Text: text, AtMs: at})
		}
	}
	add(saved.Headline, 0)
	add(saved.Gist, 0)
	for _, p := range saved.Conclusions {
		add(p.Text, p.AtMs)
	}
	for _, topic := range saved.Topics {
		add(topic.Title, topic.AtMs)
		for _, p := range topic.Points {
			add(p.Text, p.AtMs)
		}
	}
	for _, todo := range saved.Todos {
		add(todo.Text, todo.AtMs)
	}
	for _, chapter := range saved.Chapters {
		add(chapter.Title, chapter.AtMs)
		add(chapter.Text, chapter.AtMs)
	}
	for _, p := range saved.Open {
		add(p.Text, p.AtMs)
	}
	return found
}

// Search returns the meetings whose titles, transcripts or notes contain query,
// the newest first.
func Search(ctx context.Context, s *store.Store, query string) ([]SearchHit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []SearchHit{}, nil
	}
	titled, err := s.SearchTitles(ctx, query)
	if err != nil {
		return nil, err
	}
	lines, err := s.SearchLines(ctx, query)
	if err != nil {
		return nil, err
	}
	noted, err := s.SearchNotes(ctx, query)
	if err != nil {
		return nil, err
	}
	hits := map[string]*SearchHit{}
	for _, m := range titled {
		hits[m.ID] = &SearchHit{Meeting: m, TitleMatch: true, Lines: []SearchLine{}, Notes: []SearchNote{}}
	}
	hitOf := func(id string) (*SearchHit, error) {
		if hit := hits[id]; hit != nil {
			return hit, nil
		}
		m, err := s.Meeting(ctx, id)
		if err != nil {
			return nil, err
		}
		hits[id] = &SearchHit{Meeting: m, Lines: []SearchLine{}, Notes: []SearchNote{}}
		return hits[id], nil
	}
	for id, kept := range noted {
		if found := matching(kept, query); len(found) > 0 {
			hit, err := hitOf(id)
			if err != nil {
				return nil, err
			}
			hit.Notes = found
		}
	}
	for _, line := range lines {
		hit, err := hitOf(line.MeetingID)
		if err != nil {
			return nil, err
		}
		u := line.Utterance
		name := line.Speaker
		if name == "" {
			name = export.Name(u, nil)
		}
		hit.Lines = append(hit.Lines, SearchLine{
			Seq: u.Seq, StartMs: u.StartMs, Speaker: name, SpeakerID: u.Speaker, Self: line.Self, Text: u.Text,
		})
	}
	found := make([]SearchHit, 0, len(hits))
	for _, hit := range hits {
		found = append(found, *hit)
	}
	sort.Slice(found, func(a, b int) bool { return found[a].Meeting.StartedAt > found[b].Meeting.StartedAt })
	return found, nil
}
