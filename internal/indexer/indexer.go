package indexer

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/tools/go/packages"

	"github.com/ogatalars/codegraph-go/internal/store"
)

type Indexer struct {
	store *store.Store
}

func New(s *store.Store) *Indexer {
	return &Indexer{store: s}
}

func (idx *Indexer) Index(root string) error {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}

	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedImports,
		Dir: absRoot,
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return fmt.Errorf("load packages: %w", err)
	}

	now := time.Now().Unix()

	for _, pkg := range pkgs {
		for _, e := range pkg.Errors {
			fmt.Fprintf(os.Stderr, "warn: %s: %v\n", pkg.ID, e)
		}

		fileDataMap := ExtractPackage(pkg)

		for filePath, data := range fileDataMap {
			relPath, _ := filepath.Rel(absRoot, filePath)
			if relPath == "" {
				relPath = filePath
			}

			fileID, err := idx.store.UpsertFile(relPath, data.PkgName, now)
			if err != nil {
				return fmt.Errorf("upsert file %s: %w", relPath, err)
			}

			if err := idx.store.DeleteFileData(fileID); err != nil {
				return err
			}

			for _, sym := range data.Symbols {
				_ = idx.store.InsertSymbol(store.InsertSymbolParams{
					FileID:    fileID,
					Name:      sym.Name,
					FQN:       sym.FQN,
					Kind:      sym.Kind,
					Line:      sym.Line,
					Col:       sym.Col,
					Signature: sym.Signature,
					Docstring: sym.Docstring,
				})
			}

			for _, edge := range data.Edges {
				_ = idx.store.InsertEdge(store.InsertEdgeParams{
					FromFQN: edge.FromFQN,
					ToFQN:   edge.ToFQN,
					Kind:    edge.Kind,
					FileID:  fileID,
					Line:    edge.Line,
				})
			}
		}
	}

	return nil
}
