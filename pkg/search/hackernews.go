package search

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type HackerNewsEngine struct{}

func NewHackerNewsEngine() *HackerNewsEngine {
	return &HackerNewsEngine{}
}

func (e *HackerNewsEngine) Name() string {
	return "Hacker News"
}

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

func (e *HackerNewsEngine) Search(options Options) (*Response, error) {
	baseURL := "https://hn.algolia.com/api/v1/search"
	params := url.Values{}
	params.Add("query", options.Query)
	params.Add("tags", "story") // Only search for stories
	params.Add("hitsPerPage", fmt.Sprintf("%d", options.Limit))

	resp, err := http.Get(fmt.Sprintf("%s?%s", baseURL, params.Encode()))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var hResp hnResponse
	if err := json.NewDecoder(resp.Body).Decode(&hResp); err != nil {
		return nil, err
	}

	results := []Result{}
	for _, hit := range hResp.Hits {
		title := hit.Title
		if title == "" {
			continue
		}

		link := hit.URL
		if link == "" {
			// If no external URL, link to the HN discussion
			link = fmt.Sprintf("https://news.ycombinator.com/item?id=%s", hit.ObjectID)
		}

		snippet := fmt.Sprintf("By %s | %d points | %d comments", hit.Author, hit.Points, hit.NumComments)
		if hit.StoryText != "" {
			snippet = hit.StoryText[:min(len(hit.StoryText), 200)] + "..."
		}

		results = append(results, Result{
			Title:   title,
			URL:     link,
			Snippet: snippet,
		})
	}

	return &Response{Results: results}, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
