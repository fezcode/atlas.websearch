package search

import (
	"context"
	"sort"
	"sync"
	"time"
)

// Outcome captures the result of a single engine call: the results, its
// wall-clock latency, and any error.
type Outcome struct {
	Engine   Engine
	Response *Response
	Latency  time.Duration
	Err      error
}

// RunAll fans Search out across the provided engines concurrently. Each
// engine shares the parent context; partial failures are reported per-engine
// rather than bubbling up.
func RunAll(ctx context.Context, engines []Engine, opts Options) []Outcome {
	outcomes := make([]Outcome, len(engines))
	var wg sync.WaitGroup
	for i, e := range engines {
		wg.Add(1)
		go func(i int, e Engine) {
			defer wg.Done()
			start := time.Now()
			resp, err := e.Search(ctx, opts)
			outcomes[i] = Outcome{
				Engine:   e,
				Response: resp,
				Latency:  time.Since(start),
				Err:      err,
			}
		}(i, e)
	}
	wg.Wait()
	return outcomes
}

// Merge flattens successful outcomes into a single result list, tagging each
// result with its source engine's display name and interleaving top hits
// round-robin so no engine dominates the top of the list.
func Merge(outcomes []Outcome) []Result {
	buckets := make([][]Result, 0, len(outcomes))
	names := make([]string, 0, len(outcomes))
	for _, o := range outcomes {
		if o.Err != nil || o.Response == nil || len(o.Response.Results) == 0 {
			continue
		}
		buckets = append(buckets, o.Response.Results)
		names = append(names, o.Engine.Name())
	}
	var out []Result
	for depth := 0; ; depth++ {
		done := true
		for i, b := range buckets {
			if depth < len(b) {
				r := b[depth]
				r.Source = names[i]
				out = append(out, r)
				done = false
			}
		}
		if done {
			break
		}
	}
	return out
}

// SortOutcomes sorts outcomes by the engine display order.
func SortOutcomes(outcomes []Outcome) {
	sort.SliceStable(outcomes, func(i, j int) bool {
		return outcomes[i].Engine.Name() < outcomes[j].Engine.Name()
	})
}
