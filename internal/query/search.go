package query

import (
	"github.com/ogatalars/codegraph-go/internal/store"
)

type Engine struct {
	store *store.Store
}

func New(s *store.Store) *Engine {
	return &Engine{store: s}
}

type SymbolResult struct {
	FQN       string
	Kind      string
	File      string
	Line      int
	Signature string
	Docstring string
}

type StatusResult struct {
	Files   int
	Symbols int
	Edges   int
}

// Search finds symbols by name (LIKE pattern).
func (e *Engine) Search(pattern string, kind string, limit int) ([]SymbolResult, error) {
	// TODO: implement
	return nil, nil
}

// Node returns full detail for a single FQN.
func (e *Engine) Node(fqn string) (*SymbolResult, error) {
	// TODO: implement
	return nil, nil
}

// Files lists files under a path prefix.
func (e *Engine) Files(pathPrefix string) ([]string, error) {
	// TODO: implement
	return nil, nil
}

// Status returns index counts.
func (e *Engine) Status() (*StatusResult, error) {
	// TODO: implement
	return nil, nil
}
