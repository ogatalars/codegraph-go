# codegraph-go

When an AI agent needs to understand a codebase, it usually reads files one by one — burning tokens to answer questions like "what calls this function?" or "where is this type defined?". codegraph-go solves this by pre-indexing your project into a SQLite knowledge graph and exposing structured queries so agents get precise answers in a single call instead of reading dozens of files.

Inspired by [colbymchenry/codegraph](https://github.com/colbymchenry/codegraph) (TypeScript MCP server) — rewritten in Go, extended to multiple languages, and designed to run as both an interactive CLI for humans and an MCP server for AI agents.

Zero CGo. Single static binary. No Node.js required.

---

## The problem

AI agents exploring an unfamiliar codebase typically do this:

```
read file A → read file B → read file C → grep for symbol → read file D ...
```

Each step consumes tokens. For a 500-file TypeScript project, finding "what calls `authenticate()`" might require reading 20+ files. The agent pays the full token cost even for files where the symbol doesn't appear.

## How codegraph-go helps

Index once, query forever:

```
codegraph> callers src/auth/index.ts::authenticate
  src/middleware/auth.ts::verifyRequest → src/auth/index.ts::authenticate
  src/api/users.ts::createUser → src/auth/index.ts::authenticate

codegraph> trace src/api/users.ts::createUser src/db/index.ts::query
  [0] src/api/users.ts::createUser
  [1] src/auth/index.ts::authenticate
  [2] src/db/index.ts::query
```

The agent gets the full call chain in one query instead of reading the entire codebase. When used as an MCP server, Claude and other AI assistants can call these tools directly — no file reading needed for navigation questions.

### Token savings

| Task | Without codegraph-go | With codegraph-go |
|---|---|---|
| "What calls function X?" | Read 10-50 files | 1 MCP tool call |
| "Trace the path from A to B" | Read + grep across codebase | 1 `trace` query |
| "Where is type Foo defined?" | grep + read | 1 `search` + `node` query |
| "What does this function call?" | Read function + follow imports | 1 `callees` query |

---

## Supported languages

| Language | Symbols | Call edges |
|---|---|---|
| Go | func, method, struct, interface, type, var, const | Yes — cross-package, fully type-resolved via `go/types` |
| TypeScript / JavaScript | func, arrow fn, class, interface, type, enum, method | No (v1) |
| Python | def, class | No (v1) |

Go gets full call graph resolution because we use `golang.org/x/tools/go/packages` with complete type information — so `http.HandleFunc(...)` correctly resolves to `net/http.HandleFunc`, not just a local guess. TypeScript and Python use regex-based extraction with no external dependencies.

---

## Install

```bash
go install github.com/ogatalars/codegraph-go/cmd/codegraph@latest
```

Or build from source:

```bash
git clone git@github.com:ogatalars/codegraph-go.git
cd codegraph-go
go build -o codegraph ./cmd/codegraph/
```

Requires `go` in PATH (needed to load and type-check Go packages).

---

## CLI Usage

Run without flags to enter interactive mode:

```bash
./codegraph
```

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
                           kind: func | method | struct | interface | type | var | const | class

node <fqn>                 full detail for a symbol (signature, file, line, docstring)

callers <fqn>              all symbols that call this one

callees <fqn>              all symbols this one calls

trace <from> <to>          shortest call path from A to B (BFS, max depth 10)

files <path>               list indexed files under a path prefix

index                      re-index the project

status                     index stats: file count, symbol count, edge count

help                       show this list

exit                       quit
```

### FQN format

**Go** — resolved via `go/types`, works across packages:
```
<import-path>.<Symbol>
<import-path>.(*ReceiverType).<Method>

github.com/myorg/myapp/internal/auth.Authenticate
github.com/myorg/myapp/internal/auth.(*Handler).ServeHTTP
```

**TypeScript / JavaScript / Python** — file-relative:
```
<rel-file-path>::<SymbolName>

src/modules/auth/index.ts::authenticate
scripts/migrate.py::run_migration
```

### Example session on a TypeScript project

```
codegraph> status
files: 934  symbols: 2765  edges: 205

codegraph> search authenticate
  src/auth/index.ts::authenticate  (func)  src/auth/index.ts:14
  src/auth/index.ts::authenticateWithToken  (func)  src/auth/index.ts:31

codegraph> node src/auth/index.ts::authenticate
fqn:  src/auth/index.ts::authenticate
kind: func
file: src/auth/index.ts:14

codegraph> search Handler
  src/api/users.ts::UserHandler  (class)  src/api/users.ts:8
  src/api/orders.ts::OrderHandler  (class)  src/api/orders.ts:11
  src/middleware/auth.ts::withAuthHandler  (func)  src/middleware/auth.ts:5
```

### Example session on a Go project

```
codegraph> status
files: 48  symbols: 312  edges: 891

codegraph> search Index
  github.com/myorg/myapp/internal/indexer.(*Indexer).Index  (method)  internal/indexer/indexer.go:24

codegraph> callees github.com/myorg/myapp/internal/indexer.(*Indexer).Index
  → golang.org/x/tools/go/packages.Load
  → github.com/myorg/myapp/internal/indexer.ExtractPackage
  → github.com/myorg/myapp/internal/store.(*Store).UpsertFile

codegraph> trace github.com/myorg/myapp/cmd/server.main github.com/myorg/myapp/internal/store.(*Store).Query
  [0] github.com/myorg/myapp/cmd/server.main
  [1] github.com/myorg/myapp/internal/http.(*Server).Handle
  [2] github.com/myorg/myapp/internal/store.(*Store).Query
```

---

## MCP Server (Claude Code / AI assistants)

When used as an MCP server, codegraph-go exposes its tools directly to AI assistants. Claude can call `codegraph_search`, `codegraph_callers`, `codegraph_trace` etc. without reading any files — it just gets the answer.

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

Or start the MCP server directly:

```bash
./codegraph --mcp --root /path/to/your/project
```

> MCP server implementation is in progress — CLI is complete and functional.

---

## Architecture

```
cmd/codegraph/main.go       entry point — wizard branches to CLI or MCP mode

internal/indexer/           parsing layer
  indexer.go                orchestrates indexing: go/packages for Go, filepath.Walk for others
  extractor.go              go/ast + types.Info → symbols and call edges for Go
  extractor_ts.go           regex extractor for .ts/.tsx/.js/.jsx
  extractor_py.go           regex extractor for .py
  lang.go                   LangExtractor interface + registry

internal/store/             SQLite layer (modernc.org/sqlite, pure Go)
  store.go                  open/migrate, CRUD, query methods
  schema.go                 CREATE TABLE: files, symbols, edges

internal/query/             query logic
  search.go                 search, node, files, status
  trace.go                  callers, callees, BFS trace

internal/cli/               interactive terminal
  wizard.go                 mode + root prompt
  repl.go                   command loop

internal/mcp/               MCP server (in progress)
  register.go               registers tools with mark3labs/mcp-go
```

Skipped during indexing: `vendor`, `node_modules`, `testdata`, `dist`, `build`, `.next`, `__pycache__`, `.venv`, hidden directories.

---

## Limitations (v1)

- Call edges for Go only — TS/JS/Python symbols are indexed but no call graph
- No file watcher — re-run `index` after code changes
- Go generics: type parameters appear in symbols but instantiation edges not tracked
- Interface `implements` edges not yet detected (call edges only)
- Closures attributed to their enclosing named function

---

## Roadmap

- [ ] MCP server with 8 tools (search, node, callers, callees, trace, files, status, index)
- [ ] File watcher for automatic incremental re-index
- [ ] Call edge heuristics for TypeScript
- [ ] `implements` edge detection for Go interfaces
- [ ] `codegraph_context` — composite tool: search + node + callers + callees in one call
- [ ] `codegraph_impact` — what symbols break if X changes
