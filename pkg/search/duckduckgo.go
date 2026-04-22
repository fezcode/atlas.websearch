package search

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

type DuckDuckGoEngine struct{}

func NewDuckDuckGoEngine() *DuckDuckGoEngine {
	return &DuckDuckGoEngine{}
}

func (e *DuckDuckGoEngine) Name() string {
	return "DuckDuckGo"
}

func (e *DuckDuckGoEngine) Search(options Options) (*Response, error) {
	params := url.Values{}
	params.Set("q", options.Query)

	req, err := http.NewRequest("POST", "https://lite.duckduckgo.com/lite/", strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search returned status %d", resp.StatusCode)
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, err
	}

	return &Response{Results: extractDDGResults(doc, options.Limit)}, nil
}

func extractDDGResults(n *html.Node, limit int) []Result {
	var links []struct{ text, href string }
	var snippets []string

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			switch node.Data {
			case "a":
				if ddgHasClass(node, "result-link") {
					href := ddgAttr(node, "href")
					realURL := ddgDecodeRedirect(href)
					text := ddgTextContent(node)
					if realURL != "" && text != "" {
						links = append(links, struct{ text, href string }{text, realURL})
					}
				}
			case "td":
				if ddgHasClass(node, "result-snippet") {
					snippet := strings.TrimSpace(ddgTextContent(node))
					snippets = append(snippets, snippet)
				}
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)

	var results []Result
	for i, link := range links {
		snippet := ""
		if i < len(snippets) {
			snippet = snippets[i]
		}
		results = append(results, Result{
			Title:   link.text,
			URL:     link.href,
			Snippet: snippet,
		})
		if limit > 0 && len(results) >= limit {
			break
		}
	}
	return results
}

func ddgDecodeRedirect(href string) string {
	if !strings.Contains(href, "uddg=") {
		if strings.HasPrefix(href, "http") {
			return href
		}
		return ""
	}
	if strings.HasPrefix(href, "//") {
		href = "https:" + href
	}
	u, err := url.Parse(href)
	if err != nil {
		return ""
	}
	decoded, err := url.QueryUnescape(u.Query().Get("uddg"))
	if err != nil || decoded == "" {
		return ""
	}
	return decoded
}

func ddgHasClass(n *html.Node, class string) bool {
	for _, a := range n.Attr {
		if a.Key == "class" {
			for _, c := range strings.Fields(a.Val) {
				if c == class {
					return true
				}
			}
		}
	}
	return false
}

func ddgAttr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func ddgTextContent(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			sb.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.TrimSpace(sb.String())
}
