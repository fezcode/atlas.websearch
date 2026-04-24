package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"atlas.websearch/pkg/ui"
)

var Version = "dev"

func printHelp() {
	fmt.Println("Atlas Websearch — phosphor-CRT TUI for interactive web search.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  atlas.websearch [query]")
	fmt.Println("  atlas.websearch -q <query> [-e ddg|wiki|hn|reddit|all] [-l N]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -q string   Search query (alternative to positional argument)")
	fmt.Println("  -e string   Engine: ddg, wiki, hn, reddit, all (default \"ddg\")")
	fmt.Println("  -l int      Per-engine result limit (default 10)")
	fmt.Println("  -v          Show version")
	fmt.Println("  -h          Show this help")
	fmt.Println()
	fmt.Println("Inside the UI:")
	fmt.Println("  / search    ↵ open    e cycle engine    r re-run    q quit")
}

func main() {
	// Early-exit flags (match atlas.stats convention).
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-v", "--version":
			fmt.Printf("atlas.websearch v%s\n", Version)
			return
		case "-h", "--help", "help":
			printHelp()
			return
		}
	}

	queryFlag := flag.String("q", "", "Search query")
	engineFlag := flag.String("e", "ddg", "Engine: ddg, wiki, hn, reddit, all")
	limitFlag := flag.Int("l", 10, "Per-engine result limit")
	flag.Usage = printHelp
	flag.Parse()

	query := *queryFlag
	if query == "" && flag.NArg() > 0 {
		query = strings.Join(flag.Args(), " ")
	}

	cfg := ui.Config{
		Version:      Version,
		InitialQuery: query,
		EngineCode:   *engineFlag,
		Limit:        *limitFlag,
	}

	if err := ui.Start(cfg); err != nil {
		fmt.Printf("Error starting UI: %v\n", err)
		os.Exit(1)
	}
}
