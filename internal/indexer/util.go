package indexer

import (
	"regexp"
	"sort"
	"strings"
)

func splitLines(content []byte) []string {
	return strings.Split(string(content), "\n")
}

var skipDirs = map[string]bool{
	"vendor":       true,
	"node_modules": true,
	"testdata":     true,
	".git":         true,
	"dist":         true,
	"build":        true,
	".next":        true,
	"__pycache__":  true,
	".venv":        true,
	"venv":         true,
}

var callRe = regexp.MustCompile(`\b(\w+)\s*\(`)

// intraFileCallEdges scans each func/method symbol's body for calls to other
// symbols defined in the same file. Body range heuristic: from symbol's line
// to the next symbol's line. No brace/indent tracking — false positives possible
// but acceptable for AI navigation use cases.
func intraFileCallEdges(relPath string, lines []string, symbols []Symbol, skipName func(string) bool) []Edge {
	type funcSym struct {
		fqn  string
		name string
		line int
	}

	var funcs []funcSym
	byName := make(map[string]string) // name -> FQN, callable symbols only

	for _, sym := range symbols {
		if sym.Kind == "func" || sym.Kind == "method" {
			funcs = append(funcs, funcSym{sym.FQN, sym.Name, sym.Line})
			byName[sym.Name] = sym.FQN
		}
	}

	if len(funcs) == 0 {
		return nil
	}

	sort.Slice(funcs, func(i, j int) bool { return funcs[i].line < funcs[j].line })

	seen := make(map[string]bool)
	var edges []Edge

	for i, fn := range funcs {
		endLine := len(lines)
		if i+1 < len(funcs) {
			endLine = funcs[i+1].line - 1
		}

		for lineIdx := fn.line; lineIdx < endLine && lineIdx < len(lines); lineIdx++ {
			for _, m := range callRe.FindAllStringSubmatch(lines[lineIdx], -1) {
				name := m[1]
				toFQN, ok := byName[name]
				if !ok || toFQN == fn.fqn || skipName(name) {
					continue
				}
				key := fn.fqn + ">" + toFQN
				if seen[key] {
					continue
				}
				seen[key] = true
				edges = append(edges, Edge{
					FromFQN: fn.fqn,
					ToFQN:   toFQN,
					Kind:    "call",
					Line:    lineIdx + 1,
				})
			}
		}
	}

	return edges
}
