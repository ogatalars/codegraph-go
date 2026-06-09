package indexer

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

// FileData holds extracted symbols and edges for a single source file.
type FileData struct {
	PkgName string
	Symbols []Symbol
	Edges   []Edge
}

// ExtractPackage extracts all symbols and edges from every file in pkg.
func ExtractPackage(pkg *packages.Package) map[string]*FileData {
	result := make(map[string]*FileData)
	if pkg.Fset == nil || pkg.TypesInfo == nil {
		return result
	}

	pkgPath := pkg.ID
	if pkg.Types != nil {
		pkgPath = pkg.Types.Path()
	}

	for i, file := range pkg.Syntax {
		if i >= len(pkg.GoFiles) {
			continue
		}
		filePath := pkg.GoFiles[i]

		data := &FileData{PkgName: pkg.Name}

		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				data.Symbols = append(data.Symbols, extractFunc(pkg.Fset, d, pkgPath))
			case *ast.GenDecl:
				data.Symbols = append(data.Symbols, extractGenDecl(pkg.Fset, d, pkgPath)...)
			}
		}

		ranges := buildFuncRanges(pkg.Fset, file, pkgPath)
		extractCallEdges(pkg.Fset, file, pkg.TypesInfo, ranges, &data.Edges)

		result[filePath] = data
	}
	return result
}

func extractFunc(fset *token.FileSet, d *ast.FuncDecl, pkgPath string) Symbol {
	pos := fset.Position(d.Name.Pos())
	kind := "func"
	name := d.Name.Name
	fqnStr := pkgPath + "." + name

	if d.Recv != nil && len(d.Recv.List) > 0 {
		kind = "method"
		recv := receiverName(d.Recv.List[0].Type)
		fqnStr = fmt.Sprintf("%s.%s.%s", pkgPath, recv, name)
	}

	return Symbol{
		Name:      name,
		FQN:       fqnStr,
		Kind:      kind,
		Line:      pos.Line,
		Col:       pos.Column,
		Signature: funcSignature(d),
		Docstring: commentText(d.Doc),
	}
}

func extractGenDecl(fset *token.FileSet, d *ast.GenDecl, pkgPath string) []Symbol {
	var syms []Symbol
	for _, spec := range d.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			pos := fset.Position(s.Name.Pos())
			kind := "type"
			switch s.Type.(type) {
			case *ast.StructType:
				kind = "struct"
			case *ast.InterfaceType:
				kind = "interface"
			}
			syms = append(syms, Symbol{
				Name:      s.Name.Name,
				FQN:       pkgPath + "." + s.Name.Name,
				Kind:      kind,
				Line:      pos.Line,
				Col:       pos.Column,
				Docstring: commentText(d.Doc),
			})
		case *ast.ValueSpec:
			kind := "var"
			if d.Tok.String() == "const" {
				kind = "const"
			}
			for _, name := range s.Names {
				pos := fset.Position(name.Pos())
				syms = append(syms, Symbol{
					Name: name.Name,
					FQN:  pkgPath + "." + name.Name,
					Kind: kind,
					Line: pos.Line,
					Col:  pos.Column,
				})
			}
		}
	}
	return syms
}

// funcRange maps a named function's body span to its FQN for edge attribution.
type funcRange struct {
	fqn   string
	start token.Pos
	end   token.Pos
}

func buildFuncRanges(fset *token.FileSet, file *ast.File, pkgPath string) []funcRange {
	var ranges []funcRange
	for _, decl := range file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			continue
		}
		fqnStr := pkgPath + "." + fd.Name.Name
		if fd.Recv != nil && len(fd.Recv.List) > 0 {
			recv := receiverName(fd.Recv.List[0].Type)
			fqnStr = fmt.Sprintf("%s.%s.%s", pkgPath, recv, fd.Name.Name)
		}
		ranges = append(ranges, funcRange{
			fqn:   fqnStr,
			start: fd.Body.Lbrace,
			end:   fd.Body.Rbrace,
		})
	}
	return ranges
}

func findEnclosing(pos token.Pos, ranges []funcRange) string {
	for _, r := range ranges {
		if pos > r.start && pos < r.end {
			return r.fqn
		}
	}
	return ""
}

func extractCallEdges(fset *token.FileSet, file *ast.File, info *types.Info, ranges []funcRange, edges *[]Edge) {
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		fromFQN := findEnclosing(call.Pos(), ranges)
		if fromFQN == "" {
			return true
		}
		toFQN := resolveCallFQN(call, info)
		if toFQN == "" {
			return true
		}
		pos := fset.Position(call.Pos())
		*edges = append(*edges, Edge{
			FromFQN: fromFQN,
			ToFQN:   toFQN,
			Kind:    "call",
			Line:    pos.Line,
		})
		return true
	})
}

func resolveCallFQN(call *ast.CallExpr, info *types.Info) string {
	if info == nil {
		return ""
	}
	var ident *ast.Ident
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		ident = fn
	case *ast.SelectorExpr:
		ident = fn.Sel
	default:
		return ""
	}
	obj := info.Uses[ident]
	if obj == nil {
		return ""
	}
	pkg := obj.Pkg()
	if pkg == nil {
		return "" // built-in (len, make, etc.)
	}
	return pkg.Path() + "." + obj.Name()
}

func receiverName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return "(*" + receiverName(t.X) + ")"
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		return receiverName(t.X)
	case *ast.IndexListExpr:
		return receiverName(t.X)
	}
	return "unknown"
}

func funcSignature(d *ast.FuncDecl) string {
	var b strings.Builder
	b.WriteString("func ")
	if d.Recv != nil && len(d.Recv.List) > 0 {
		b.WriteString("(")
		b.WriteString(receiverName(d.Recv.List[0].Type))
		b.WriteString(") ")
	}
	b.WriteString(d.Name.Name)
	b.WriteString("(")
	if d.Type.Params != nil {
		for i, field := range d.Type.Params.List {
			if i > 0 {
				b.WriteString(", ")
			}
			for j, name := range field.Names {
				if j > 0 {
					b.WriteString(", ")
				}
				b.WriteString(name.Name)
			}
			if len(field.Names) > 0 {
				b.WriteString(" ")
			}
			b.WriteString(exprString(field.Type))
		}
	}
	b.WriteString(")")
	if d.Type.Results != nil && len(d.Type.Results.List) > 0 {
		results := d.Type.Results.List
		if len(results) == 1 && len(results[0].Names) == 0 {
			b.WriteString(" ")
			b.WriteString(exprString(results[0].Type))
		} else {
			b.WriteString(" (")
			for i, field := range results {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString(exprString(field.Type))
			}
			b.WriteString(")")
		}
	}
	return b.String()
}

func exprString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + exprString(t.X)
	case *ast.SelectorExpr:
		return exprString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + exprString(t.Elt)
		}
		return "[...]" + exprString(t.Elt)
	case *ast.MapType:
		return "map[" + exprString(t.Key) + "]" + exprString(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.Ellipsis:
		return "..." + exprString(t.Elt)
	case *ast.ChanType:
		return "chan " + exprString(t.Value)
	case *ast.FuncType:
		return "func(...)"
	}
	return "_"
}

func commentText(cg *ast.CommentGroup) string {
	if cg == nil {
		return ""
	}
	var lines []string
	for _, c := range cg.List {
		text := strings.TrimPrefix(c.Text, "//")
		text = strings.TrimPrefix(text, "/*")
		text = strings.TrimSuffix(text, "*/")
		text = strings.TrimSpace(text)
		if text != "" {
			lines = append(lines, text)
		}
	}
	return strings.Join(lines, "\n")
}
