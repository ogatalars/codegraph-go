# SDD — codegraph-go

## Objetivo

CLI + MCP server em Go que indexa workspaces **multi-linguagem** (Go, TypeScript, JavaScript, Python) em SQLite e expõe tools de code intelligence para Claude (e qualquer AI via MCP). Zero CGo, zero Node.js, binary estático. Mesmo binário, dois modos de uso.

---

## Escopo v1

**Inclui:**
- Parse de arquivos `.go` com `go/ast` + `go/types` (symbols + call edges cross-package)
- Parse de `.ts`/`.tsx`/`.js`/`.jsx` via regex (symbols; sem call edges)
- Parse de `.py` via regex (symbols; sem call edges)
- Interface `LangExtractor` plugável — fácil adicionar linguagens
- Índice SQLite com símbolos e arestas do grafo
- CLI interativo com prompt de setup (modo, projeto)
- MCP server com 8 tools (mesmo binário, flag `--mcp`)
- Re-index manual (sem file watcher)

**Fora do escopo v1:**
- File watcher automático
- Call edges para TS/JS/Python (só Go tem)
- Suporte a múltiplos workspaces simultâneos

---

## Suporte Multi-Linguagem

### Estratégia por linguagem

| Linguagem | Extrator | Call edges | Deps extras | CGo |
|---|---|---|---|---|
| Go | `go/packages` + `go/ast` | Sim (type-resolved, cross-pkg) | nenhuma | Não |
| TypeScript / JavaScript | regex | Não (v1) | nenhuma | Não |
| Python | regex | Não (v1) | nenhuma | Não |

Decisão: tree-sitter foi descartado (CGo). esbuild foi descartado (API interna instável). Regex cobre 80% dos casos relevantes para AI navigation com zero deps.

### Interface plugável

```go
type LangExtractor interface {
    Extensions() []string  // ex: []string{".ts", ".tsx"}
    Extract(relPath string, content []byte) ([]Symbol, []Edge, error)
}
```

Extractors registrados globalmente no `indexer`. Para adicionar linguagem nova: implementar interface e registrar.

### FQN por linguagem

- **Go**: `<import-path>.<Symbol>` / `<import-path>.(*Receiver).<Method>` (resolvido por `go/types`)
- **TS/JS/Python**: `<rel-file-path>::<SymbolName>` (ex: `src/utils/api.ts::fetchUser`)

### Símbolos extraídos por regex

**TypeScript / JavaScript:**
- `function name(` / `async function name(`
- `export function` / `export default function`
- `const name = (` / `const name = async (` (arrow functions)
- `class Name` / `abstract class Name`
- `interface Name`
- `type Name =`

**Python:**
- `def name(` / `async def name(`
- `class Name:`

---

## Arquitetura

Mesmo binário, dois entry points:

```
codegraph [flags] [command]

  sem flags      → modo CLI interativo
  --mcp          → modo MCP server (stdio, para Claude Code)
```

```
┌──────────────────────────────────────────────────────┐
│                     main.go                           │
│   if --mcp → startMCPServer()                        │
│   else     → startCLI()  (prompt interativo)         │
└──────────┬───────────────────────┬───────────────────┘
           │ MCP (JSON-RPC/stdio)  │ CLI (cobra/bubbletea)
           ▼                       ▼
┌──────────────────────────────────────────────────────┐
│                   Query Layer                         │
│          (callers, callees, trace, search…)           │
└──────────────────────┬───────────────────────────────┘
                       │
┌──────────────────────▼───────────────────────────────┐
│                  SQLite Store                         │
│              modernc.org/sqlite                       │
└──────────────────────┬───────────────────────────────┘
                       │
┌──────────────────────▼───────────────────────────────┐
│                    Indexer                            │
│              go/ast + go/packages                     │
└──────────────────────────────────────────────────────┘
```

---

## Fluxo de Inicialização (CLI interativo)

Quando rodado sem `--mcp`, o binário apresenta um wizard de setup:

```
$ codegraph

? Mode:
  ▸ CLI  (query manual no terminal)
    MCP  (iniciar como MCP server — adicione ao claude_desktop_config.json)

? Project root:  [./]  (caminho do workspace Go a indexar)

? Index now? [Y/n]

→ Indexing /path/to/project... 142 files, 3841 symbols, 12034 edges (4.2s)
→ Ready. Type 'help' for commands.

codegraph> _
```

Se modo MCP for selecionado no wizard, imprime instrução de config e encerra:

```
Add to your MCP config (~/.claude/claude_desktop_config.json):

  "codegraph": {
    "command": "/usr/local/bin/codegraph",
    "args": ["--mcp", "--root", "/path/to/project"]
  }

Then restart Claude Code.
```

### Comandos CLI disponíveis

```
codegraph> search HandleRequest
codegraph> node pkg/http.HandleRequest
codegraph> callers pkg/http.HandleRequest
codegraph> callees pkg/http.HandleRequest
codegraph> trace pkg/main.main pkg/http.HandleRequest
codegraph> files ./internal/auth
codegraph> index          (re-indexa o projeto atual)
codegraph> status
codegraph> help
codegraph> exit
```

---

## Schema SQLite

### `files`
```sql
CREATE TABLE files (
    id      INTEGER PRIMARY KEY,
    path    TEXT UNIQUE NOT NULL,  -- relativo ao workspace root
    pkg     TEXT NOT NULL,         -- package name
    indexed_at INTEGER NOT NULL    -- unix timestamp
);
```

### `symbols`
```sql
CREATE TABLE symbols (
    id      INTEGER PRIMARY KEY,
    file_id INTEGER NOT NULL REFERENCES files(id),
    name    TEXT NOT NULL,         -- nome simples: "HandleRequest"
    fqn     TEXT UNIQUE NOT NULL,  -- fully qualified: "pkg/http.HandleRequest"
    kind    TEXT NOT NULL,         -- func | method | type | var | const | interface | struct
    line    INTEGER NOT NULL,
    col     INTEGER NOT NULL,
    signature TEXT,                -- "func(w http.ResponseWriter, r *http.Request) error"
    docstring TEXT                 -- comentário acima da declaração
);

CREATE INDEX idx_symbols_name ON symbols(name);
CREATE INDEX idx_symbols_fqn  ON symbols(fqn);
CREATE INDEX idx_symbols_file ON symbols(file_id);
```

### `edges`
```sql
CREATE TABLE edges (
    id       INTEGER PRIMARY KEY,
    from_fqn TEXT NOT NULL,   -- quem chama
    to_fqn   TEXT NOT NULL,   -- quem é chamado
    kind     TEXT NOT NULL,   -- call | implements | embeds | references
    file_id  INTEGER NOT NULL REFERENCES files(id),
    line     INTEGER NOT NULL
);

CREATE INDEX idx_edges_from ON edges(from_fqn);
CREATE INDEX idx_edges_to   ON edges(to_fqn);
```

---

## Indexer

### Fluxo

```
workspace root
    └── walk .go files (excluindo vendor/, testdata/, _test.go opcional)
            └── per file:
                    1. parse AST (go/parser.ParseFile)
                    2. type-check package (go/types.Checker)  ← resolve calls entre pkgs
                    3. extract symbols  → INSERT INTO symbols
                    4. extract edges    → INSERT INTO edges
                    5── INSERT INTO files
```

### O que extraímos

**Símbolos:**
- `FuncDecl` → kind=`func` ou `method` (se tem receiver)
- `TypeSpec` → kind=`type`, sub-kind: `struct`, `interface`
- `ValueSpec` → kind=`var` ou `const`

**Arestas:**
- `CallExpr` dentro de func body → edge kind=`call`
- `InterfaceType` com métodos correspondendo a struct → edge kind=`implements`
- Struct com campo do tipo de outra struct → edge kind=`embeds`

### Type-checking

`go/types` resolve chamadas cross-package: `http.HandleFunc(...)` vira edge `from=pkg/main.main → to=net/http.HandleFunc`. Sem isso, só temos arestas intra-package.

---

## MCP Tools (8)

### `codegraph_status`
Retorna: total de arquivos, símbolos, edges, timestamp do último index.

### `codegraph_index`
Dispara re-indexação do workspace. Params: `{ root: string }`.

### `codegraph_search`
Busca símbolo por nome (LIKE). Params: `{ query: string, kind?: string, limit?: int }`.
Retorna: lista de `{ fqn, kind, file, line, signature }`.

### `codegraph_node`
Detalhe completo de um símbolo. Params: `{ fqn: string }`.
Retorna: símbolo + docstring + arquivo + linha.

### `codegraph_callers`
Quem chama este símbolo. Params: `{ fqn: string, depth?: int }`.
Retorna: lista de edges `from_fqn → to_fqn`.

### `codegraph_callees`
O que este símbolo chama. Params: `{ fqn: string, depth?: int }`.
Retorna: lista de edges.

### `codegraph_trace`
Caminho de A até B no grafo (BFS). Params: `{ from: string, to: string }`.
Retorna: lista de FQNs representando o path.

### `codegraph_files`
Lista arquivos de um package ou diretório. Params: `{ path: string }`.
Retorna: lista de arquivos com contagem de símbolos.

---

## Estrutura de Pacotes

```
codegraph-go/
├── main.go                  -- entry point: branch --mcp vs CLI
├── internal/
│   ├── indexer/
│   │   ├── indexer.go       -- walk + orquestra parse por package
│   │   ├── extractor.go     -- ast → symbols e edges
│   │   └── types.go         -- structs internas (Symbol, Edge)
│   ├── store/
│   │   ├── store.go         -- open/migrate SQLite, métodos CRUD
│   │   └── schema.go        -- SQL de criação das tabelas
│   ├── query/
│   │   ├── search.go        -- search, node, files
│   │   └── trace.go         -- callers, callees, BFS trace
│   ├── mcp/
│   │   └── register.go      -- registra tools no mcp-go server
│   └── cli/
│       ├── wizard.go        -- prompt interativo de setup (mode + root)
│       └── repl.go          -- loop de comandos CLI
└── go.mod
```

---

## Dependências

```
modernc.org/sqlite                  -- SQLite pure Go, zero CGo
mark3labs/mcp-go                    -- MCP server protocol
golang.org/x/tools/go/packages     -- type-check cross-package
github.com/charmbracelet/bubbletea -- wizard CLI interativo (opcional, pode usar fmt+bufio simples)
```

> Se quiser manter zero deps extras no CLI, o wizard pode ser implementado com `fmt.Scan` + `bufio` nativo — suficiente para v1.

---

## Decisões de Design

| Decisão | Escolha | Alternativa descartada | Motivo |
|---|---|---|---|
| Parser | `go/ast` nativo | tree-sitter | Zero CGo, type info grátis |
| Type resolution | `golang.org/x/tools/go/packages` | `go/types` direto | Resolve imports cross-pkg automaticamente |
| SQLite driver | `modernc.org/sqlite` | `mattn/go-sqlite3` | Pure Go, binary estático |
| Transport MCP | stdio | HTTP | Mais simples, padrão do Claude Code |
| File watcher | Nenhum (v1) | fsnotify | Reduz escopo; re-index é suficiente |
| Interface | CLI + MCP, mesmo binário | só MCP | CLI útil pra humanos; MCP para AI. Core compartilhado, custo ~100 linhas extras |
| CLI wizard | `fmt`/`bufio` nativo | bubbletea | Zero deps extras; suficiente para perguntas simples de setup |

---

## Limitações Conhecidas

- `go/packages` requer `go` toolchain no PATH (já presente em qualquer dev machine)
- Interfaces `implements` são detectadas estruturalmente, mas só dentro do workspace (não stdlib)
- `_test.go` indexados por padrão (configurável)
- Sem suporte a generics completo: type params aparecem nos símbolos mas edges de instanciação não são mapeadas

---

## Próximos Passos (v2)

- File watcher com `fsnotify` + re-index incremental por arquivo
- `codegraph_context` (compõe search + node + callers + callees em 1 call)
- `codegraph_impact` (quais símbolos quebram se X mudar)
- Suporte a `.ts`/`.py` via tree-sitter (opcional)
