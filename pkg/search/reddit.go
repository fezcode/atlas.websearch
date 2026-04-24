package search

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type RedditEngine struct{}

func NewRedditEngine() *RedditEngine { return &RedditEngine{} }

func (e *RedditEngine) Name() string { return "Reddit" }
func (e *RedditEngine) Code() string { return "reddit" }

type redditResponse struct {
	Data struct {
		Children []struct {
			Data struct {
				Title       string `json:"title"`
				URL         string `json:"url"`
				Selftext    string `json:"selftext"`
				Subreddit   string `json:"subreddit"`
				Author      string `json:"author"`
				Score       int    `json:"score"`
				NumComments int    `json:"num_comments"`
				Permalink   string `json:"permalink"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

func (e *RedditEngine) Search(ctx context.Context, opts Options) (*Response, error) {
	params := url.Values{}
	params.Add("q", opts.Query)
	params.Add("limit", fmt.Sprintf("%d", opts.Limit))
	params.Add("sort", "relevance")

	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.reddit.com/search.json?"+params.Encode(), nil)
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
		return nil, fmt.Errorf("reddit: status %d", resp.StatusCode)
	}

	var rr redditResponse
	if err := json.NewDecoder(resp.Body).Decode(&rr); err != nil {
		return nil, err
	}

	out := make([]Result, 0, len(rr.Data.Children))
	for _, child := range rr.Data.Children {
		d := child.Data
		meta := fmt.Sprintf("r/%s · u/%s · %d pts · %d comments",
			d.Subreddit, d.Author, d.Score, d.NumComments)

		snippet := meta
		if d.Selftext != "" {
			text := strings.ReplaceAll(d.Selftext, "\n", " ")
			if len(text) > 220 {
				text = text[:220] + "…"
			}
			snippet = text + " — " + meta
		}
		link := d.URL
		if !strings.HasPrefix(link, "http") {
			link = "https://www.reddit.com" + d.Permalink
		}
		out = append(out, Result{Title: d.Title, URL: link, Snippet: snippet})
	}
	return &Response{Results: out}, nil
}
