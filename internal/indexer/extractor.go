package indexer

import (
	"go/ast"
	"go/token"
)

// extractSymbols walks an AST file and returns all top-level symbols.
func extractSymbols(fset *token.FileSet, file *ast.File, fileID int64, pkgPath string) []Symbol {
	var symbols []Symbol
	// TODO: implement
	return symbols
}

// extractEdges walks an AST file and returns all call/embed/implements edges.
func extractEdges(fset *token.FileSet, file *ast.File, fileID int64, pkgPath string) []Edge {
	var edges []Edge
	// TODO: implement
	return edges
}
