package main

import (
	"context"

	"github.com/xlsama/meetkeep/internal/polish"
	"github.com/xlsama/meetkeep/internal/store"
	"github.com/xlsama/meetkeep/internal/trace"
)

// polish returns what makes utterances easier to read, to give to
// SetPolished, and tells what it took. It returns no edits when polishing
// is off or there is no model to ask.
func (a *app) polish(ctx context.Context, utterances []store.Utterance) ([]store.Utterance, trace.Detail, error) {
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
	result, err := polish.Polish(ctx, model, lines, settings.ASR.Hotwords)
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
		"refused": result.Refused, "failed": result.Failed, "failure": result.Failure, "terms": result.Terms,
		"requests": used.Requests, "promptTokens": used.PromptTokens, "completionTokens": used.CompletionTokens,
	}
	if err != nil {
		return nil, detail, err
	}
	return edits, detail, nil
}
