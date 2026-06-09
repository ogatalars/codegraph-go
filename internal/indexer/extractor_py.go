package indexer

import (
	"regexp"
	"strings"
)

func init() {
	registerExtractor(&pyExtractor{})
}

type pyExtractor struct{}

func (p *pyExtractor) Extensions() []string {
	return []string{".py"}
}

var (
	pyFunc  = regexp.MustCompile(`(?m)^(?:    )*(?:async\s+)?def\s+(\w+)\s*\(`)
	pyClass = regexp.MustCompile(`(?m)^class\s+(\w+)`)
)

func (p *pyExtractor) Extract(relPath string, content []byte) ([]Symbol, []Edge, error) {
	full := string(content)
	lines := strings.Split(full, "\n")
	var symbols []Symbol

	add := func(name, kind string, lineIdx int) {
		if name == "" {
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

	for _, re := range []struct {
		r    *regexp.Regexp
		kind string
	}{
		{pyFunc, "func"},
		{pyClass, "class"},
	} {
		for _, m := range re.r.FindAllStringSubmatchIndex(full, -1) {
			if len(m) < 4 || m[2] < 0 {
				continue
			}
			name := full[m[2]:m[3]]
			line := strings.Count(full[:m[0]], "\n")
			add(name, re.kind, line)
		}
	}

	_ = lines
	return symbols, nil, nil
}
