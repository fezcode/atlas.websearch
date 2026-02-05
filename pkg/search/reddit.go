package search

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type RedditEngine struct{}

func NewRedditEngine() *RedditEngine {
	return &RedditEngine{}
}

func (e *RedditEngine) Name() string {
	return "Reddit"
}

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

func (e *RedditEngine) Search(options Options) (*Response, error) {
	baseURL := "https://www.reddit.com/search.json"
	params := url.Values{}
	params.Add("q", options.Query)
	params.Add("limit", fmt.Sprintf("%d", options.Limit))
	params.Add("sort", "relevance")

	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s?%s", baseURL, params.Encode()), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "AtlasWebSearch/1.0 (CLI Tool)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reddit api returned status %d", resp.StatusCode)
	}

	var rResp redditResponse
	if err := json.NewDecoder(resp.Body).Decode(&rResp); err != nil {
		return nil, err
	}

	results := []Result{}
	for _, child := range rResp.Data.Children {
		data := child.Data

		meta := fmt.Sprintf("r/%s | By u/%s | %d pts | %d comments",
			data.Subreddit, data.Author, data.Score, data.NumComments)

		snippet := meta
		if data.Selftext != "" {
			text := strings.ReplaceAll(data.Selftext, "\n", " ")
			if len(text) > 200 {
				text = text[:200] + "..."
			}
			snippet = text + "\n" + meta
		}

		link := data.URL
		if !strings.HasPrefix(link, "http") {
			link = "https://www.reddit.com" + data.Permalink
		}

		results = append(results, Result{
			Title:   data.Title,
			URL:     link,
			Snippet: snippet,
		})
	}

	return &Response{Results: results}, nil
}
