# codegraph-go

Code intelligence tool for multi-language projects. Indexes Go, TypeScript, JavaScript, and Python workspaces into SQLite and exposes symbol search, call graph traversal, and trace queries — via an interactive CLI or as an MCP server for AI assistants (Claude Code, etc.).

Zero CGo. Single static binary.

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
                           kind: func | method | struct | interface | type | var | const | class

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
files: 934  symbols: 2765  edges: 205

codegraph> search fetchUser
  src/modules/user/api.ts::fetchUser  (func)  src/modules/user/api.ts:12
  src/modules/user/api.ts::fetchUserById  (func)  src/modules/user/api.ts:28

codegraph> node src/modules/user/api.ts::fetchUser
fqn:  src/modules/user/api.ts::fetchUser
kind: func
file: src/modules/user/api.ts:12

codegraph> search HandleRequest
  github.com/myorg/myapp/internal/http.(*Server).HandleRequest  (method)  internal/http/server.go:42

codegraph> callers github.com/myorg/myapp/internal/http.(*Server).HandleRequest
  github.com/myorg/myapp/cmd/server.main → github.com/myorg/myapp/internal/http.(*Server).HandleRequest

codegraph> trace github.com/myorg/myapp/cmd/server.main github.com/myorg/myapp/internal/store.(*Store).Query
  [0] github.com/myorg/myapp/cmd/server.main
  [1] github.com/myorg/myapp/internal/http.(*Server).HandleRequest
  [2] github.com/myorg/myapp/internal/store.(*Store).Query
```

### FQN format

FQN format depends on the language:

**Go** — resolved via `go/types`, cross-package:
```
<import-path>.<Symbol>
<import-path>.(*ReceiverType).<Method>
<import-path>.(ReceiverType).<Method>

github.com/myorg/myapp/internal/auth.Authenticate
github.com/myorg/myapp/internal/auth.(*Handler).ServeHTTP
```

**TypeScript / JavaScript / Python** — file-relative:
```
<rel-file-path>::<SymbolName>

src/modules/auth/index.ts::authenticate
src/utils/api.ts::fetchUser
scripts/migrate.py::run_migration
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

> MCP server implementation is in progress (CLI is complete).

---

## How it works

1. **Index** — two-phase walk:
   - Go files: `golang.org/x/tools/go/packages` with full type info. Call edges resolved cross-package via `types.Info`.
   - TS/JS/Python files: regex-based symbol extraction. No CGo, no external tools required.
2. **Store** — SQLite (pure Go, `modernc.org/sqlite`). Three tables: `files`, `symbols`, `edges`.
3. **Query** — direct SQL for search/callers/callees; BFS over edge queries for `trace`.

### What gets indexed

| Language | Symbols | Call edges |
|---|---|---|
| Go | func, method, struct, interface, type, var, const | Yes (cross-package, type-resolved) |
| TypeScript / JavaScript | func, arrow fn, class, interface, type, enum, method | No (v1) |
| Python | def, class | No (v1) |

Skipped directories: `vendor`, `node_modules`, `testdata`, `dist`, `build`, `.next`, `__pycache__`, `.venv`.

---

## Limitations (v1)

- Go call edges only — TS/JS/Python symbols indexed but no call graph
- No file watcher — re-run `index` after code changes
- Generics: type parameters appear in symbols but instantiation edges not tracked
- Interface `implements` edges not yet detected
- Closures attributed to their enclosing named function

---

## Roadmap

- [ ] MCP server (8 tools: search, node, callers, callees, trace, files, status, index)
- [ ] File watcher for incremental re-index
- [ ] Call edges for TypeScript via regex heuristic
- [ ] `implements` edge detection for Go
- [ ] `codegraph_context` composite tool (search + node + callers + callees in one call)
- [ ] `codegraph_impact` (what breaks if symbol X changes)
