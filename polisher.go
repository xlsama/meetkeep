package main

import (
	"context"

	"github.com/xlsama/meetkeep/internal/polish"
	"github.com/xlsama/meetkeep/internal/store"
	"github.com/xlsama/meetkeep/internal/trace"
)

// polish returns what makes the utterances of a meeting easier to read, to
// give to SetPolished, and tells what it took. The meeting is read with the
// words earlier meetings and the user set right, and those it sets right
// are kept for the next, as are those it leaves for the user to say. It returns no edits when polishing is off or there
// is no model to ask.
func (a *app) polish(ctx context.Context, id string, utterances []store.Utterance) ([]store.Utterance, trace.Detail, error) {
	settings := a.current()
	if !settings.ASR.Polish {
		return nil, trace.Detail{"skipped": "已关闭"}, nil
	}
	model, err := a.notes.model()
	if err != nil {
		return nil, trace.Detail{"skipped": err.Error()}, nil
	}
	lines := make([]polish.Line, len(utterances))
	for i, u := range utterances {
		lines[i] = polish.Line{Seq: u.Seq, Text: u.Text}
	}
	kept, _ := a.store.Terms(ctx)
	learned := make([]polish.Learned, len(kept))
	for i, t := range kept {
		learned[i] = polish.Learned{
			Heard: t.Heard, Meant: t.Meant, Meetings: t.Meetings,
			Confirmed: t.State == store.TermConfirmed, Dropped: t.State == store.TermDropped,
		}
	}
	result, err := polish.Polish(ctx, model, lines, polish.Recall(learned, settings.ASR.Hotwords))
	if err == nil {
		// What the lines alone told is left to the next meeting to find.
		var found []store.Term
		for _, t := range result.Terms {
			if !t.Local {
				found = append(found, store.Term{Heard: t.Heard, Meant: t.Meant})
			}
		}
		_ = a.store.Learn(ctx, id, found)
		guesses := make([]store.Guess, len(result.Guesses))
		for i, g := range result.Guesses {
			guesses[i] = store.Guess{Seq: g.Seq, Heard: g.Heard, Meant: g.Meant}
		}
		_ = a.store.SetGuesses(ctx, id, guesses)
	}
	used := model.Used()
	fillers := 0
	edits := make([]store.Utterance, len(result.Edits))
	for i, e := range result.Edits {
		edits[i] = store.Utterance{Seq: e.Seq, Polished: e.Text, Filler: e.Filler}
		if e.Filler {
			fillers++
		}
	}
	detail := trace.Detail{
		"model": model.Model, "lines": len(lines), "changed": len(edits) - fillers, "fillers": fillers,
		"refused": result.Refused, "failed": result.Failed, "failure": result.Failure, "unread": result.Unread, "terms": result.Terms, "readMs": result.Read.Milliseconds(),
		"reviewed": result.Reviewed, "revised": result.Revised, "reviewMs": result.Review.Milliseconds(), "guesses": result.Guesses,
		"requests": used.Requests, "promptTokens": used.PromptTokens, "cachedTokens": used.CachedTokens, "completionTokens": used.CompletionTokens,
	}
	if err != nil {
		return nil, detail, err
	}
	return edits, detail, nil
}
