package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/ogatalars/codegraph-go/internal/cli"
	"github.com/ogatalars/codegraph-go/internal/indexer"
	"github.com/ogatalars/codegraph-go/internal/store"
)

func main() {
	mcpMode := flag.Bool("mcp", false, "run as MCP server (stdio)")
	root := flag.String("root", "", "project root to index (MCP mode)")
	flag.Parse()

	dbPath := filepath.Join(os.TempDir(), "codegraph.db")

	s, err := store.Open(dbPath)
	if err != nil {
		log.Fatal("open store:", err)
	}
	defer s.Close()

	idx := indexer.New(s)

	if *mcpMode {
		if *root == "" {
			fmt.Fprintln(os.Stderr, "error: --root required in --mcp mode")
			os.Exit(1)
		}
		startMCP(s, idx, *root)
		return
	}

	// Interactive CLI wizard
	cfg, err := cli.Wizard()
	if err != nil {
		log.Fatal(err)
	}

	if cfg.Mode == "mcp" {
		fmt.Printf(`
Add to your MCP config (~/.claude/claude_desktop_config.json):

  "codegraph": {
    "command": "%s",
    "args": ["--mcp", "--root", "%s"]
  }

Then restart Claude Code.
`, os.Args[0], cfg.Root)
		return
	}

	// Index on start
	fmt.Printf("indexing %s...\n", cfg.Root)
	if err := idx.Index(cfg.Root); err != nil {
		log.Fatal("index:", err)
	}

	// TODO: replace with query.New(s) once query pkg has Status
	_ = s

	// cli.REPL(q, idx, cfg.Root)
	fmt.Println("index done. REPL not yet implemented.")
}

func startMCP(s *store.Store, idx *indexer.Indexer, root string) {
	// TODO: implement MCP server start
	_ = s
	_ = idx
	_ = root
	fmt.Fprintln(os.Stderr, "MCP server not yet implemented")
	os.Exit(1)
}
