package main

import (
	"context"
	"encoding/json"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/xlsama/meetkeep/internal/export"
	"github.com/xlsama/meetkeep/internal/llm"
	"github.com/xlsama/meetkeep/internal/notes"
	"github.com/xlsama/meetkeep/internal/recorder"
	"github.com/xlsama/meetkeep/internal/store"
	"github.com/xlsama/meetkeep/internal/trace"
)

// NotesState is what is known of a meeting's notes.
type NotesState struct {
	MeetingID string `json:"meetingId"`
	// Notes are nil while the meeting has none.
	Notes *notes.Notes `json:"notes"`
	// Doc is the document the user made of the notes by editing them, as
	// SaveNotesDoc took it, or "".
	Doc string `json:"doc"`
	// Replaced tells that the notes written before these can be restored.
	Replaced bool `json:"replaced"`
	// Error tells why the last attempt to write them failed.
	Error string `json:"error"`
	// Working is true while the model writes them.
	Working bool `json:"working"`
}

const (
	// notesEvery is how often the notes of a meeting in progress are
	// brought up to date, as long as enough was said since.
	notesEvery = 45 * time.Second
	// notesAtLatest is how long anything said waits to be noted.
	notesAtLatest = 2 * time.Minute
	notesFirst    = 120
	notesEnough   = 60
)

// noteTaker writes the notes of meetings: an outline while one is recorded,
// the final notes once it is processed.
type noteTaker struct {
	app *app

	mu      sync.Mutex
	meeting string
	stop    context.CancelFunc
	working map[string]bool
}

func (n *noteTaker) model() (*llm.Client, error) {
	s := n.app.current().LLM
	return llm.New(s.BaseURL, s.Key, s.Model)
}

func (n *noteTaker) lines(ctx context.Context, id string) ([]notes.Line, error) {
	utterances, err := n.app.store.Utterances(ctx, id)
	if err != nil {
		return nil, err
	}
	speakers, err := n.app.store.Speakers(ctx, id)
	if err != nil {
		return nil, err
	}
	utterances = store.Shown(utterances)
	lines := make([]notes.Line, len(utterances))
	for i, u := range utterances {
		lines[i] = notes.Line{Seq: u.Seq, StartMs: u.StartMs, EndMs: u.EndMs, Speaker: export.Name(u, speakers), Text: u.Text}
	}
	return lines, nil
}

func (n *noteTaker) state(ctx context.Context, id string) (NotesState, error) {
	kept, err := n.app.store.Notes(ctx, id)
	if err != nil {
		return NotesState{}, err
	}
	n.mu.Lock()
	state := NotesState{MeetingID: id, Doc: kept.Doc, Replaced: kept.Replaced, Error: kept.Error, Working: n.working[id]}
	n.mu.Unlock()
	if kept.Data != "" {
		var saved notes.Notes
		if err := json.Unmarshal([]byte(kept.Data), &saved); err != nil {
			return NotesState{}, err
		}
		state.Notes = &saved
	}
	return state, nil
}

func (n *noteTaker) announce(id string) {
	if state, err := n.state(context.Background(), id); err == nil {
		_ = notesChanged.Broadcast(state)
	}
}

func (n *noteTaker) busy(id string, working bool) {
	n.mu.Lock()
	if n.working == nil {
		n.working = map[string]bool{}
	}
	n.working[id] = working
	n.mu.Unlock()
	n.announce(id)
}

// save stores notes. Final notes that take the place of others keep them,
// so that the user can have them back.
func (n *noteTaker) save(ctx context.Context, id string, written notes.Notes, err error) {
	if err == nil {
		var data []byte
		if data, err = json.Marshal(written); err == nil {
			before, _ := n.state(ctx, id)
			if written.Final && before.Notes != nil && before.Notes.Final {
				err = n.app.store.ReplaceNotes(ctx, id, string(data), written.Text())
			} else {
				err = n.app.store.SaveNotes(ctx, id, string(data), written.Text())
			}
		}
	}
	if err != nil && ctx.Err() == nil {
		_ = n.app.store.SetNotesError(context.Background(), id, err.Error())
	}
}

// follow keeps an outline of the meeting being recorded.
func (n *noteTaker) follow(s recorder.Status) {
	n.mu.Lock()
	defer n.mu.Unlock()
	id := ""
	if s.Meeting != nil && (s.Phase == recorder.Recording || s.Phase == recorder.Paused) {
		id = s.Meeting.ID
	}
	if id == n.meeting {
		return
	}
	if n.stop != nil {
		n.stop()
		n.stop = nil
	}
	n.meeting = id
	if id != "" {
		ctx, cancel := context.WithCancel(context.Background())
		n.stop = cancel
		go n.outline(ctx, id)
	}
}

func (n *noteTaker) outline(ctx context.Context, id string) {
	current := notes.Empty()
	noted := map[int]bool{}
	last := time.Now()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		model, err := n.model()
		if err != nil {
			continue
		}
		lines, err := n.lines(ctx, id)
		if err != nil {
			continue
		}
		fresh := 0
		for _, l := range lines {
			if !noted[l.Seq] {
				fresh += utf8.RuneCountInString(l.Text)
			}
		}
		waited := time.Since(last)
		enough := notesEnough
		if len(current.Topics) == 0 {
			enough = notesFirst
		}
		if fresh == 0 || !(fresh >= enough && waited >= notesEvery || waited >= notesAtLatest) {
			continue
		}

		n.busy(id, true)
		request, cancel := context.WithTimeout(ctx, 2*time.Minute)
		updated, err := notes.Update(request, model, current, lines, n.app.current().ASR.Hotwords)
		cancel()
		if err == nil {
			current = updated
			for _, l := range lines {
				noted[l.Seq] = true
			}
		}
		last = time.Now()
		if ctx.Err() == nil {
			n.save(ctx, id, current, err)
		}
		n.busy(id, false)
	}
}

// finalize writes the notes of a meeting that was processed.
func (n *noteTaker) finalize(ctx context.Context, id string) (trace.Detail, error) {
	model, err := n.model()
	if err != nil {
		return trace.Detail{"skipped": err.Error()}, nil
	}
	meeting, err := n.app.store.Meeting(ctx, id)
	if err != nil {
		return nil, err
	}
	lines, err := n.lines(ctx, id)
	if err != nil {
		return nil, err
	}
	detail := trace.Detail{"model": model.Model, "lines": len(lines)}
	if len(lines) == 0 {
		return detail, nil
	}
	n.busy(id, true)
	defer n.busy(id, false)
	written, err := notes.Finalize(ctx, model, meeting.Title, lines, n.app.current().ASR.Hotwords)
	used := model.Used()
	detail["requests"], detail["promptTokens"], detail["completionTokens"] = used.Requests, used.PromptTokens, used.CompletionTokens
	n.save(ctx, id, written, err)
	if err != nil {
		return detail, err
	}
	detail["topics"], detail["todos"] = len(written.Topics), len(written.Todos)
	n.app.windows.notesReady(meeting, written)
	return detail, nil
}

// rewrite writes the final notes of a processed meeting again.
func (n *noteTaker) rewrite(id string) {
	_ = trace.Begin(n.app.store, id, "notes").Step("notes", func() (trace.Detail, error) {
		return n.finalize(context.Background(), id)
	})
}
