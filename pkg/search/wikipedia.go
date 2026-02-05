package search

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type WikipediaEngine struct{}

func NewWikipediaEngine() *WikipediaEngine {
	return &WikipediaEngine{}
}

func (e *WikipediaEngine) Name() string {
	return "Wikipedia"
}

func (e *WikipediaEngine) Search(options Options) (*Response, error) {
	baseURL := "https://en.wikipedia.org/w/api.php"
	params := url.Values{}
	params.Add("action", "opensearch")
	params.Add("search", options.Query)
	params.Add("limit", fmt.Sprintf("%d", options.Limit))
	params.Add("namespace", "0")
	params.Add("format", "json")

	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s?%s", baseURL, params.Encode()), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "AtlasWebSearch/1.0 (https://github.com/user/atlas-websearch)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	if len(raw) < 4 {
		return &Response{}, nil
	}

	titles := raw[1].([]interface{})
	descriptions := raw[2].([]interface{})
	urls := raw[3].([]interface{})

	results := []Result{}
	for i := 0; i < len(titles); i++ {
		results = append(results, Result{
			Title:   titles[i].(string),
			URL:     urls[i].(string),
			Snippet: descriptions[i].(string),
		})
	}

	return &Response{Results: results}, nil
}
