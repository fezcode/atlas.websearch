package search

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type HackerNewsEngine struct{}

func NewHackerNewsEngine() *HackerNewsEngine { return &HackerNewsEngine{} }

func (e *HackerNewsEngine) Name() string { return "Hacker News" }
func (e *HackerNewsEngine) Code() string { return "hn" }

type hnResponse struct {
	Hits []struct {
		Title       string `json:"title"`
		URL         string `json:"url"`
		StoryText   string `json:"story_text"`
		Author      string `json:"author"`
		Points      int    `json:"points"`
		NumComments int    `json:"num_comments"`
		ObjectID    string `json:"objectID"`
	} `json:"hits"`
}

func (e *HackerNewsEngine) Search(ctx context.Context, opts Options) (*Response, error) {
	params := url.Values{}
	params.Add("query", opts.Query)
	params.Add("tags", "story")
	params.Add("hitsPerPage", fmt.Sprintf("%d", opts.Limit))

	req, err := http.NewRequestWithContext(ctx, "GET", "https://hn.algolia.com/api/v1/search?"+params.Encode(), nil)
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
		return nil, fmt.Errorf("hn: status %d", resp.StatusCode)
	}

	var hr hnResponse
	if err := json.NewDecoder(resp.Body).Decode(&hr); err != nil {
		return nil, err
	}

	out := make([]Result, 0, len(hr.Hits))
	for _, hit := range hr.Hits {
		title := hit.Title
		if title == "" {
			continue
		}
		link := hit.URL
		if link == "" {
			link = fmt.Sprintf("https://news.ycombinator.com/item?id=%s", hit.ObjectID)
		}
		snippet := fmt.Sprintf("by %s · %d pts · %d comments", hit.Author, hit.Points, hit.NumComments)
		if hit.StoryText != "" {
			text := hit.StoryText
			if len(text) > 220 {
				text = text[:220] + "…"
			}
			snippet = text + " — " + snippet
		}
		out = append(out, Result{Title: title, URL: link, Snippet: snippet})
	}
	return &Response{Results: out}, nil
}
