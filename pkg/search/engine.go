package search

type Engine interface {
	Search(options Options) (*Response, error)
	Name() string
}

type Options struct {
	Query  string
	Format string
	Limit  int // if available
	Offset int // if available
}

type Response struct {
	Results []Result `json:"results"`
}

type Result struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}
