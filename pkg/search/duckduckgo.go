package search

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

type DuckDuckGoEngine struct{}

func NewDuckDuckGoEngine() *DuckDuckGoEngine { return &DuckDuckGoEngine{} }

func (e *DuckDuckGoEngine) Name() string { return "DuckDuckGo" }
func (e *DuckDuckGoEngine) Code() string { return "ddg" }

func (e *DuckDuckGoEngine) Search(ctx context.Context, opts Options) (*Response, error) {
	params := url.Values{}
	params.Set("q", opts.Query)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://lite.duckduckgo.com/lite/", strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ddg: status %d", resp.StatusCode)
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, err
	}
	return &Response{Results: extractDDGResults(doc, opts.Limit)}, nil
}

func extractDDGResults(n *html.Node, limit int) []Result {
	type anchor struct{ text, href string }
	var links []anchor
	var snippets []string

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			switch node.Data {
			case "a":
				if ddgHasClass(node, "result-link") {
					href := ddgAttr(node, "href")
					real := ddgDecodeRedirect(href)
					text := ddgTextContent(node)
					if real != "" && text != "" {
						links = append(links, anchor{text, real})
					}
				}
			case "td":
				if ddgHasClass(node, "result-snippet") {
					snippets = append(snippets, strings.TrimSpace(ddgTextContent(node)))
				}
			}
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)

	out := make([]Result, 0, len(links))
	for i, a := range links {
		snip := ""
		if i < len(snippets) {
			snip = snippets[i]
		}
		out = append(out, Result{Title: a.text, URL: a.href, Snippet: snip})
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
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
