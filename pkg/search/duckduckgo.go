package search

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type DuckDuckGoEngine struct{}

func NewDuckDuckGoEngine() *DuckDuckGoEngine {
	return &DuckDuckGoEngine{}
}

func (e *DuckDuckGoEngine) Name() string {
	return "DuckDuckGo"
}

type ddgRelatedTopic struct {
	Text     string            `json:"Text"`
	FirstURL string            `json:"FirstURL"`
	Topics   []ddgRelatedTopic `json:"Topics"` // Nested topics
}

type ddgResponse struct {
	Abstract       string     `json:"Abstract"`
	AbstractURL    string     `json:"AbstractURL"`
	AbstractSource string     `json:"AbstractSource"`
	Results        []struct { // Official / Direct results
		Text     string `json:"Text"`
		FirstURL string `json:"FirstURL"`
	} `json:"Results"`
	RelatedTopics []ddgRelatedTopic `json:"RelatedTopics"`
}

func (e *DuckDuckGoEngine) Search(options Options) (*Response, error) {
	baseURL := "https://api.duckduckgo.com/"
	params := url.Values{}
	params.Add("q", options.Query)
	params.Add("format", "json")
	params.Add("no_html", "1")
	params.Add("skip_disambig", "1")

	resp, err := http.Get(fmt.Sprintf("%s?%s", baseURL, params.Encode()))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ddgResp ddgResponse
	if err := json.NewDecoder(resp.Body).Decode(&ddgResp); err != nil {
		return nil, err
	}

	results := []Result{}

	// 1. Add Abstract (The primary summary)
	if ddgResp.AbstractURL != "" {
		results = append(results, Result{
			Title:   ddgResp.AbstractSource + " (Summary)",
			URL:     ddgResp.AbstractURL,
			Snippet: ddgResp.Abstract,
		})
	}

	// 2. Add Direct Results (Often Official Sites)
	for _, r := range ddgResp.Results {
		results = append(results, Result{
			Title:   r.Text,
			URL:     r.FirstURL,
			Snippet: "Official/Direct Result",
		})
	}

	// 3. Add Related Topics (Recursive)
	results = append(results, e.parseTopics(ddgResp.RelatedTopics)...)

	// Filter and Limit
	uniqueResults := []Result{}
	seen := make(map[string]bool)
	for _, r := range results {
		if r.URL == "" || seen[r.URL] {
			continue
		}
		// Filter out internal DDG links that aren't useful
		if strings.Contains(r.URL, "duckduckgo.com/") &&
			(strings.Contains(r.URL, "/c/") || strings.HasSuffix(r.URL, "topic")) {
			continue
		}
		seen[r.URL] = true
		uniqueResults = append(uniqueResults, r)
		if options.Limit > 0 && len(uniqueResults) >= options.Limit {
			break
		}
	}

	return &Response{Results: uniqueResults}, nil
}

func (e *DuckDuckGoEngine) parseTopics(topics []ddgRelatedTopic) []Result {
	results := []Result{}
	for _, t := range topics {
		if t.FirstURL != "" {
			results = append(results, Result{
				Title:   t.Text,
				URL:     t.FirstURL,
				Snippet: t.Text,
			})
		}
		if len(t.Topics) > 0 {
			results = append(results, e.parseTopics(t.Topics)...)
		}
	}
	return results
}
