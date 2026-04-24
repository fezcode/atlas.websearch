package search

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type WikipediaEngine struct{}

func NewWikipediaEngine() *WikipediaEngine { return &WikipediaEngine{} }

func (e *WikipediaEngine) Name() string { return "Wikipedia" }
func (e *WikipediaEngine) Code() string { return "wiki" }

func (e *WikipediaEngine) Search(ctx context.Context, opts Options) (*Response, error) {
	params := url.Values{}
	params.Add("action", "opensearch")
	params.Add("search", opts.Query)
	params.Add("limit", fmt.Sprintf("%d", opts.Limit))
	params.Add("namespace", "0")
	params.Add("format", "json")

	req, err := http.NewRequestWithContext(ctx, "GET", "https://en.wikipedia.org/w/api.php?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wiki: status %d", resp.StatusCode)
	}

	var raw []any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	if len(raw) < 4 {
		return &Response{}, nil
	}

	titles, _ := raw[1].([]any)
	descriptions, _ := raw[2].([]any)
	urls, _ := raw[3].([]any)

	out := make([]Result, 0, len(titles))
	for i := range titles {
		title, _ := titles[i].(string)
		desc := ""
		if i < len(descriptions) {
			desc, _ = descriptions[i].(string)
		}
		link := ""
		if i < len(urls) {
			link, _ = urls[i].(string)
		}
		out = append(out, Result{Title: title, URL: link, Snippet: desc})
	}
	return &Response{Results: out}, nil
}
