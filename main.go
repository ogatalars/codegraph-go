package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/ogatalars/codegraph-go/internal/cli"
	"github.com/ogatalars/codegraph-go/internal/indexer"
	"github.com/ogatalars/codegraph-go/internal/query"
	"github.com/ogatalars/codegraph-go/internal/store"
)

func main() {
	mcpMode := flag.Bool("mcp", false, "run as MCP server (stdio)")
	root := flag.String("root", "", "project root (required in --mcp mode)")
	flag.Parse()

	dbPath := filepath.Join(os.TempDir(), "codegraph.db")

	s, err := store.Open(dbPath)
	if err != nil {
		log.Fatal("open store:", err)
	}
	defer s.Close()

	idx := indexer.New(s)
	q := query.New(s)

	if *mcpMode {
		if *root == "" {
			fmt.Fprintln(os.Stderr, "error: --root required in --mcp mode")
			os.Exit(1)
		}
		startMCP(s, idx, q, *root)
		return
	}

	cfg, stdin, err := cli.Wizard()
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

	fmt.Printf("indexing %s...\n", cfg.Root)
	if err := idx.Index(cfg.Root); err != nil {
		log.Fatal("index:", err)
	}

	cli.REPL(q, idx, cfg.Root, stdin)
}

func startMCP(s *store.Store, idx *indexer.Indexer, q *query.Engine, root string) {
	// TODO: implement MCP server
	_ = s
	_ = idx
	_ = q
	_ = root
	fmt.Fprintln(os.Stderr, "MCP server not yet implemented")
	os.Exit(1)
}
