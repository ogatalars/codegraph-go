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

func (e *Engine) Search(pattern string, kind string, limit int) ([]SymbolResult, error) {
	rows, err := e.store.SearchSymbols(pattern, kind, limit)
	if err != nil {
		return nil, err
	}
	result := make([]SymbolResult, len(rows))
	for i, r := range rows {
		result[i] = SymbolResult{
			FQN:       r.FQN,
			Kind:      r.Kind,
			File:      r.FilePath,
			Line:      r.Line,
			Signature: r.Signature,
			Docstring: r.Docstring,
		}
	}
	return result, nil
}

func (e *Engine) Node(fqn string) (*SymbolResult, error) {
	r, err := e.store.GetSymbol(fqn)
	if err != nil || r == nil {
		return nil, err
	}
	return &SymbolResult{
		FQN:       r.FQN,
		Kind:      r.Kind,
		File:      r.FilePath,
		Line:      r.Line,
		Signature: r.Signature,
		Docstring: r.Docstring,
	}, nil
}

func (e *Engine) Files(pathPrefix string) ([]string, error) {
	rows, err := e.store.GetFiles(pathPrefix)
	if err != nil {
		return nil, err
	}
	result := make([]string, len(rows))
	for i, r := range rows {
		result[i] = r.Path
	}
	return result, nil
}

func (e *Engine) Status() (*StatusResult, error) {
	files, symbols, edges, err := e.store.GetStatus()
	if err != nil {
		return nil, err
	}
	return &StatusResult{Files: files, Symbols: symbols, Edges: edges}, nil
}
