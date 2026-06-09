package indexer

import (
	"github.com/ogatalars/codegraph-go/internal/store"
)

type Indexer struct {
	store *store.Store
}

func New(s *store.Store) *Indexer {
	return &Indexer{store: s}
}

// Index walks root, parses all .go files, and populates the store.
func (idx *Indexer) Index(root string) error {
	// TODO: implement
	return nil
}
