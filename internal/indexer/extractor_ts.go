package indexer

import (
	"regexp"
	"strings"
)

func init() {
	registerExtractor(&tsExtractor{})
}

type tsExtractor struct{}

func (t *tsExtractor) Extensions() []string {
	return []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"}
}

var (
	tsFunc      = regexp.MustCompile(`(?m)^(?:export\s+(?:default\s+)?)?(?:async\s+)?function\s+(\w+)\s*[\(<]`)
	tsArrow     = regexp.MustCompile(`(?m)^(?:export\s+)?(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s*)?\(`)
	tsClass     = regexp.MustCompile(`(?m)^(?:export\s+)?(?:abstract\s+)?class\s+(\w+)`)
	tsInterface = regexp.MustCompile(`(?m)^(?:export\s+)?interface\s+(\w+)`)
	tsType      = regexp.MustCompile(`(?m)^(?:export\s+)?type\s+(\w+)\s*[=<]`)
	tsMethod    = regexp.MustCompile(`(?m)^\s{2,}(?:(?:public|private|protected|static|async|readonly|override)\s+)*(\w+)\s*\(`)
	tsEnum      = regexp.MustCompile(`(?m)^(?:export\s+)?(?:const\s+)?enum\s+(\w+)`)
)

func (t *tsExtractor) Extract(relPath string, content []byte) ([]Symbol, []Edge, error) {
	lines := splitLines(content)
	var symbols []Symbol

	add := func(name, kind string, lineIdx int) {
		if name == "" || isKeyword(name) {
			return
		}
		symbols = append(symbols, Symbol{
			Name: name,
			FQN:  relPath + "::" + name,
			Kind: kind,
			Line: lineIdx + 1,
			Col:  1,
		})
	}

	extractAll(lines, tsFunc, "func", add)
	extractAll(lines, tsArrow, "func", add)
	extractAll(lines, tsClass, "class", add)
	extractAll(lines, tsInterface, "interface", add)
	extractAll(lines, tsType, "type", add)
	extractAll(lines, tsEnum, "type", add)
	extractAll(lines, tsMethod, "method", add)

	return symbols, nil, nil
}

// tsKeywords that look like identifiers but aren't symbols
var tsKeywordSet = map[string]bool{
	"if": true, "for": true, "while": true, "switch": true, "return": true,
	"new": true, "delete": true, "typeof": true, "instanceof": true,
	"constructor": true, "super": true, "this": true, "catch": true,
}

func isKeyword(name string) bool {
	return tsKeywordSet[name]
}

func extractAll(lines []string, re *regexp.Regexp, kind string, add func(string, string, int)) {
	full := strings.Join(lines, "\n")
	matches := re.FindAllStringIndex(full, -1)
	submatches := re.FindAllStringSubmatchIndex(full, -1)
	_ = matches
	for _, m := range submatches {
		if len(m) < 4 || m[2] < 0 {
			continue
		}
		name := full[m[2]:m[3]]
		line := strings.Count(full[:m[0]], "\n")
		add(name, kind, line)
	}
}
