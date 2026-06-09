package cli

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/ogatalars/codegraph-go/internal/indexer"
	"github.com/ogatalars/codegraph-go/internal/query"
)

// REPL runs an interactive command loop using the reader shared with Wizard.
func REPL(q *query.Engine, idx *indexer.Indexer, root string, r *bufio.Reader) {
	fmt.Printf("Ready. Root: %s\nType 'help' for commands.\n\n", root)

	for {
		fmt.Print("codegraph> ")
		line, err := r.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		cmd := parts[0]
		args := parts[1:]

		switch cmd {
		case "help":
			printHelp()
		case "exit", "quit":
			return
		case "status":
			handleStatus(q)
		case "index":
			handleIndex(idx, root)
		case "search":
			handleSearch(q, args)
		case "node":
			handleNode(q, args)
		case "callers":
			handleCallers(q, args)
		case "callees":
			handleCallees(q, args)
		case "trace":
			handleTrace(q, args)
		case "files":
			handleFiles(q, args)
		default:
			fmt.Printf("unknown command: %s (type 'help')\n", cmd)
		}
	}
}

func printHelp() {
	fmt.Print(`
Commands:
  search <name> [kind]          search symbols by name
  node <fqn>                    full detail for a symbol
  callers <fqn> [depth]         who calls this symbol
  callees <fqn> [depth]         what this symbol calls
  trace <from> <to>             call path from A to B
  files <path>                  list files under path
  index                         re-index project
  status                        index stats
  exit                          quit
`)
}

func handleStatus(q *query.Engine) {
	st, err := q.Status()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("files: %d  symbols: %d  edges: %d\n", st.Files, st.Symbols, st.Edges)
}

func handleIndex(idx *indexer.Indexer, root string) {
	fmt.Printf("indexing %s...\n", root)
	if err := idx.Index(root); err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("done")
}

func handleSearch(q *query.Engine, args []string) {
	if len(args) == 0 {
		fmt.Println("usage: search <name> [kind]")
		return
	}
	kind := ""
	if len(args) > 1 {
		kind = args[1]
	}
	results, err := q.Search(args[0], kind, 20)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	for _, r := range results {
		fmt.Printf("  %s  (%s)  %s:%d\n", r.FQN, r.Kind, r.File, r.Line)
	}
}

func handleNode(q *query.Engine, args []string) {
	if len(args) == 0 {
		fmt.Println("usage: node <fqn>")
		return
	}
	n, err := q.Node(args[0])
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	if n == nil {
		fmt.Println("not found")
		return
	}
	fmt.Printf("fqn:  %s\nkind: %s\nfile: %s:%d\nsig:  %s\n", n.FQN, n.Kind, n.File, n.Line, n.Signature)
	if n.Docstring != "" {
		fmt.Println("doc: ", n.Docstring)
	}
}

func handleCallers(q *query.Engine, args []string) {
	if len(args) == 0 {
		fmt.Println("usage: callers <fqn> [depth]")
		return
	}
	results, err := q.Callers(args[0], 1)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	for _, e := range results {
		fmt.Printf("  %s → %s\n", e.FromFQN, e.ToFQN)
	}
}

func handleCallees(q *query.Engine, args []string) {
	if len(args) == 0 {
		fmt.Println("usage: callees <fqn> [depth]")
		return
	}
	results, err := q.Callees(args[0], 1)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	for _, e := range results {
		fmt.Printf("  %s → %s\n", e.FromFQN, e.ToFQN)
	}
}

func handleTrace(q *query.Engine, args []string) {
	if len(args) < 2 {
		fmt.Println("usage: trace <from> <to>")
		return
	}
	path, err := q.Trace(args[0], args[1])
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	for i, fqn := range path {
		fmt.Printf("  [%d] %s\n", i, fqn)
	}
}

func handleFiles(q *query.Engine, args []string) {
	prefix := "."
	if len(args) > 0 {
		prefix = args[0]
	}
	files, err := q.Files(prefix)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	for _, f := range files {
		fmt.Println(" ", f)
	}
}
