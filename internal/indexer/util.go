package indexer

import "strings"

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
