package search

import (
	"context"
	"net/http"
	"time"
)

// Engine is a single search backend.
type Engine interface {
	Search(ctx context.Context, opts Options) (*Response, error)
	Name() string
	Code() string // short identifier: ddg, wiki, hn, reddit
}

type Options struct {
	Query  string
	Limit  int
	Offset int
}

type Response struct {
	Results []Result `json:"results"`
}

type Result struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
	Source  string `json:"source"` // populated for multi-engine results
}

// Shared HTTP client with a hard ceiling; per-request context handles the
// finer-grained timeout.
var client = &http.Client{Timeout: 15 * time.Second}

const userAgent = "atlas.websearch/1.0 (+https://github.com/fezcode/atlas.websearch)"

// Registry returns every built-in engine in display order.
func Registry() []Engine {
	return []Engine{
		NewDuckDuckGoEngine(),
		NewWikipediaEngine(),
		NewHackerNewsEngine(),
		NewRedditEngine(),
	}
}

// ByCode looks up an engine by its short code (ddg/wiki/hn/reddit). Falls
// back to DuckDuckGo when the code is unknown.
func ByCode(code string) Engine {
	for _, e := range Registry() {
		if e.Code() == code {
			return e
		}
	}
	return NewDuckDuckGoEngine()
}
