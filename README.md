# codegraph-go

Code intelligence tool for Go projects. Indexes a Go workspace into SQLite and exposes symbol search, call graph traversal, and trace queries — via an interactive CLI or as an MCP server for AI assistants (Claude Code, etc.).

Zero CGo. Single static binary.

---

## Install

```bash
go install github.com/ogatalars/codegraph-go@latest
```

Or build from source:

```bash
git clone git@github.com:ogatalars/codegraph-go.git
cd codegraph-go
go build -o codegraph .
```

Requires `go` in PATH (needed to load and type-check packages).

---

## CLI Usage

Run without flags to enter interactive mode:

```bash
./codegraph
```

The wizard asks two questions:

```
codegraph-go — code intelligence for AI

Mode [cli/mcp] (default: cli): cli
Project root (default: ./): /path/to/your/project
indexing /path/to/your/project...
Ready. Root: /path/to/your/project
Type 'help' for commands.

codegraph>
```

### Commands

```
search <name> [kind]       search symbols by name (substring match)
                           kind: func | method | struct | interface | type | var | const

node <fqn>                 full detail for a symbol

callers <fqn> [depth]      who calls this symbol

callees <fqn> [depth]      what this symbol calls

trace <from> <to>          call path from symbol A to symbol B (BFS, max depth 10)

files <path>               list indexed files under a path prefix

index                      re-index the project

status                     show index stats (files, symbols, edges)

help                       show this list

exit                       quit
```

### Examples

```
codegraph> status
files: 8  symbols: 47  edges: 112

codegraph> search Index
  github.com/ogatalars/codegraph-go/internal/indexer.(*Indexer).Index  (method)  internal/indexer/indexer.go:24

codegraph> node github.com/ogatalars/codegraph-go/internal/indexer.(*Indexer).Index
fqn:  github.com/ogatalars/codegraph-go/internal/indexer.(*Indexer).Index
kind: method
file: internal/indexer/indexer.go:24
sig:  func (*Indexer) Index(root string) error

codegraph> callers github.com/ogatalars/codegraph-go/internal/indexer.(*Indexer).Index
  main.main → github.com/ogatalars/codegraph-go/internal/indexer.(*Indexer).Index

codegraph> callees github.com/ogatalars/codegraph-go/internal/indexer.(*Indexer).Index
  github.com/ogatalars/codegraph-go/internal/indexer.(*Indexer).Index → golang.org/x/tools/go/packages.Load
  github.com/ogatalars/codegraph-go/internal/indexer.(*Indexer).Index → github.com/ogatalars/codegraph-go/internal/indexer.ExtractPackage
  ...

codegraph> trace main.main github.com/ogatalars/codegraph-go/internal/store.(*Store).InsertSymbol
  [0] main.main
  [1] github.com/ogatalars/codegraph-go/internal/indexer.(*Indexer).Index
  [2] github.com/ogatalars/codegraph-go/internal/store.(*Store).InsertSymbol
```

### FQN format

Fully qualified names follow the pattern:

```
<import-path>.<Symbol>              # func, type, var, const
<import-path>.(*ReceiverType).<Method>   # pointer receiver method
<import-path>.(ReceiverType).<Method>    # value receiver method
```

Examples:
```
github.com/myorg/myapp/internal/auth.Authenticate
github.com/myorg/myapp/internal/auth.(*Handler).ServeHTTP
```

---

## MCP Server (Claude Code / AI assistants)

Select `mcp` in the wizard to get the config snippet:

```bash
./codegraph
# Mode: mcp
# Project root: /path/to/your/project
```

Output:

```
Add to your MCP config (~/.claude/claude_desktop_config.json):

  "codegraph": {
    "command": "/path/to/codegraph",
    "args": ["--mcp", "--root", "/path/to/your/project"]
  }

Then restart Claude Code.
```

Or pass flags directly:

```bash
./codegraph --mcp --root /path/to/your/project
```

> MCP server implementation is in progress (v1 CLI is complete).

---

## How it works

1. **Index** — uses `golang.org/x/tools/go/packages` to load packages with full type info. Walks AST to extract symbols and resolves call targets cross-package via `types.Info`.
2. **Store** — SQLite (pure Go, `modernc.org/sqlite`). Three tables: `files`, `symbols`, `edges`.
3. **Query** — direct SQL for search/callers/callees; BFS over edge queries for `trace`.

### What gets indexed

| AST node | Symbol kind |
|---|---|
| `FuncDecl` (no receiver) | `func` |
| `FuncDecl` (with receiver) | `method` |
| `TypeSpec` + `StructType` | `struct` |
| `TypeSpec` + `InterfaceType` | `interface` |
| `TypeSpec` (other) | `type` |
| `ValueSpec` in `var` block | `var` |
| `ValueSpec` in `const` block | `const` |

Call edges are resolved to their canonical import path using `types.Info.Uses`, so cross-package calls (e.g. `http.ListenAndServe`) are correctly attributed.

---

## Limitations (v1)

- Go only (no TypeScript, Python, etc.)
- No file watcher — re-run `index` after code changes
- Generics: type parameters appear in symbols but instantiation edges are not tracked
- Interface `implements` edges not yet detected (call edges only)
- Closures attributed to their enclosing named function

---

## Roadmap

- [ ] MCP server (8 tools: search, node, callers, callees, trace, files, status, index)
- [ ] File watcher for incremental re-index
- [ ] `implements` edge detection
- [ ] `codegraph_context` composite tool (search + node + callers + callees in one call)
- [ ] `codegraph_impact` (what breaks if symbol X changes)
