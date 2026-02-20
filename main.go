package main

import (
	"flag"
	"fmt"
	"os"

	"atlas.websearch/pkg/search"
	"atlas.websearch/pkg/ui"
)

var Version = "dev"

func main() {
	queryFlag := flag.String("q", "", "Search query")
	limit := flag.Int("l", 10, "Result limit")
	engineType := flag.String("e", "ddg", "Engine to use (ddg, wiki, hn, reddit)")
	version := flag.Bool("version", false, "Show version")
	flag.BoolVar(version, "v", false, "Show version")
	flag.Parse()

	if *version {
		fmt.Printf("atlas.websearch v%s\n", Version)
		return
	}

	query := *queryFlag
	// If -q is not provided, take only the first non-flag argument
	if query == "" && flag.NArg() > 0 {
		query = flag.Arg(0)
	}

	if query == "" {
		fmt.Println("Usage: atlas-websearch [\"query\"] or -q <query> [-e ddg|wiki|hn|reddit]")
		os.Exit(1)
	}

	var engine search.Engine
	switch *engineType {
	case "wiki":
		engine = search.NewWikipediaEngine()
	case "hn":
		engine = search.NewHackerNewsEngine()
	case "reddit":
		engine = search.NewRedditEngine()
	default:
		engine = search.NewDuckDuckGoEngine()
	}

	opts := search.Options{
		Query: query,
		Limit: *limit,
	}

	fmt.Printf("Searching for '%s' using %s...\n", query, engine.Name())
	resp, err := engine.Search(opts)
	if err != nil {
		fmt.Printf("Error searching: %v\n", err)
		os.Exit(1)
	}

	if err := ui.RenderResults(resp, engine.Name(), query); err != nil {
		fmt.Printf("Error rendering UI: %v\n", err)
		os.Exit(1)
	}
}
