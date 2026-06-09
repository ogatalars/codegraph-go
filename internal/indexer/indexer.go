package indexer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

	now := time.Now().Unix()

	if err := idx.indexGoPackages(absRoot, now); err != nil {
		return err
	}
	if err := idx.indexOtherFiles(absRoot, now); err != nil {
		return err
	}
	return nil
}

func (idx *Indexer) indexGoPackages(absRoot string, now int64) error {
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
			if err := idx.insertFileData(relPath, data.PkgName, now, data.Symbols, data.Edges); err != nil {
				return err
			}
		}
	}
	return nil
}

func (idx *Indexer) indexOtherFiles(absRoot string, now int64) error {
	return filepath.Walk(absRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if skipDirs[info.Name()] || strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		ext := fileExt(path)
		if ext == ".go" {
			return nil // handled by indexGoPackages
		}

		ex := extractorFor(path)
		if ex == nil {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		relPath, _ := filepath.Rel(absRoot, path)
		if relPath == "" {
			relPath = path
		}

		symbols, edges, err := ex.Extract(relPath, content)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: extract %s: %v\n", relPath, err)
			return nil
		}

		return idx.insertFileData(relPath, ext[1:], now, symbols, edges)
	})
}

func (idx *Indexer) insertFileData(relPath, pkg string, now int64, symbols []Symbol, edges []Edge) error {
	fileID, err := idx.store.UpsertFile(relPath, pkg, now)
	if err != nil {
		return fmt.Errorf("upsert file %s: %w", relPath, err)
	}

	if err := idx.store.DeleteFileData(fileID); err != nil {
		return err
	}

	for _, sym := range symbols {
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

	for _, edge := range edges {
		_ = idx.store.InsertEdge(store.InsertEdgeParams{
			FromFQN: edge.FromFQN,
			ToFQN:   edge.ToFQN,
			Kind:    edge.Kind,
			FileID:  fileID,
			Line:    edge.Line,
		})
	}
	return nil
}
