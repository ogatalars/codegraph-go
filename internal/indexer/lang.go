package indexer

// LangExtractor extracts symbols and call edges from a single source file.
// Implement this interface to add support for a new language.
type LangExtractor interface {
	Extensions() []string
	Extract(relPath string, content []byte) ([]Symbol, []Edge, error)
}

var registry []LangExtractor

func registerExtractor(e LangExtractor) {
	registry = append(registry, e)
}

func extractorFor(path string) LangExtractor {
	ext := fileExt(path)
	for _, e := range registry {
		for _, x := range e.Extensions() {
			if x == ext {
				return e
			}
		}
	}
	return nil
}

func fileExt(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			return path[i:]
		}
		if path[i] == '/' {
			break
		}
	}
	return ""
}
