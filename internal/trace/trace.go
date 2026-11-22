// Package trace records what each step of processing a meeting took.
package trace

import (
	"context"
	"encoding/json"
	"time"

	"github.com/xlsama/meetkeep/internal/memory"
	"github.com/xlsama/meetkeep/internal/store"
)

// Detail is what a step worked on and made.
type Detail map[string]any

// Run is one processing of a meeting.
type Run struct {
	store   *store.Store
	meeting string
	kind    string
	asked   time.Time
}

// Begin starts the record of a processing asked for now.
func Begin(s *store.Store, meeting, kind string) *Run {
	return &Run{store: s, meeting: meeting, kind: kind, asked: time.Now()}
}

// Waited records, as the step queue, the time since the processing was
// asked for.
func (r *Run) Waited() { r.add("queue", r.asked, 0, nil, nil) }

// Step does work and records what it took, its detail and what it failed
// at, which it returns.
func (r *Run) Step(name string, work func() (Detail, error)) error {
	started := time.Now()
	peak := memory.Watch()
	detail, err := work()
	r.add(name, started, peak(), detail, err)
	return err
}

func (r *Run) add(name string, started time.Time, peak int64, detail Detail, err error) {
	step := store.Step{
		Run: r.asked.UnixMilli(), Kind: r.kind, Name: name, StartedAt: started.UnixMilli(),
		DurationMs: time.Since(started).Milliseconds(), MemoryPeak: peak, Detail: "{}",
	}
	if data, e := json.Marshal(detail); e == nil && len(detail) > 0 {
		step.Detail = string(data)
	}
	if err != nil {
		step.Error = err.Error()
	}
	_ = r.store.AddStep(context.Background(), r.meeting, step)
}
