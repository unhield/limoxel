package symbol

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
)

// parseGoFile parses a single Go source file into structured symbols, documentation entries, and relationships.
func parseGoFile(absPath, relPath string) ([]*Symbol, []*DocEntry, []*SymbolRelationship, []*discovery.Diagnostic) {
	var diagnostics []*discovery.Diagnostic

	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, absPath, nil, parser.ParseComments)
	if err != nil {
		diagnostics = append(diagnostics, discovery.NewDiagnostic(
			discovery.SeverityWarning,
			"GO_SYNTAX_ERROR",
			err.Error(),
			relPath,
			false,
		))
		if astFile == nil {
			return nil, nil, nil, diagnostics
		}
	}

	cleanRel := filepath.ToSlash(filepath.Clean(relPath))
	pkgPath := filepath.ToSlash(filepath.Dir(cleanRel))
	if pkgPath == "" {
		pkgPath = "."
	}
	pkgName := ""
	if astFile.Name != nil {
		pkgName = astFile.Name.Name
	}

	var (
		symbols       []*Symbol
		docs          []*DocEntry
		relationships []*SymbolRelationship
	)

	// 1. Package Declaration Symbol & Documentation
	pkgPos := fset.Position(astFile.Package)
	pkgPosModel := NewSourcePosition(cleanRel, pkgPos.Line, pkgPos.Column, pkgPos.Offset)

	var pkgDoc *DocEntry
	if astFile.Doc != nil {
		docText := strings.TrimSpace(astFile.Doc.Text())
		if docText != "" {
			docPos := fset.Position(astFile.Doc.Pos())
			pkgDoc = NewDocEntry(
				fmt.Sprintf("doc:pkg:%s", pkgPath),
				pkgPath,
				DocKindPackage,
				docText,
				astFile.Doc.Text(),
				NewSourcePosition(cleanRel, docPos.Line, docPos.Column, docPos.Offset),
			)
			docs = append(docs, pkgDoc)
		}
	}

	pkgSymbol := NewSymbol(
		pkgPath,
		SymbolKindPackage,
		pkgName,
		pkgName,
		pkgPath,
		cleanRel,
		"",
		false,
		"package "+pkgName,
		"",
		false,
		nil,
		nil,
		pkgPosModel,
		pkgDoc,
	)
	symbols = append(symbols, pkgSymbol)

	// 2. Extract TODO / FIXME comments from all file comment groups
	for _, cg := range astFile.Comments {
		for _, c := range cg.List {
			text := c.Text
			cPos := fset.Position(c.Pos())
			posModel := NewSourcePosition(cleanRel, cPos.Line, cPos.Column, cPos.Offset)

			if idx := strings.Index(text, "TODO"); idx != -1 {
				todoContent := strings.TrimSpace(text[idx:])
				docs = append(docs, NewDocEntry(
					fmt.Sprintf("todo:%s:%d", cleanRel, cPos.Line),
					"",
					DocKindTODO,
					todoContent,
					text,
					posModel,
				))
			}
			if idx := strings.Index(text, "FIXME"); idx != -1 {
				fixmeContent := strings.TrimSpace(text[idx:])
				docs = append(docs, NewDocEntry(
					fmt.Sprintf("fixme:%s:%d", cleanRel, cPos.Line),
					"",
					DocKindFIXME,
					fixmeContent,
					text,
					posModel,
				))
			}
		}
	}

	// 3. Inspect AST Declarations
	for _, decl := range astFile.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			sym, dEntry, rels := extractFuncDecl(fset, d, cleanRel, pkgName, pkgPath)
			if sym != nil {
				symbols = append(symbols, sym)
			}
			if dEntry != nil {
				docs = append(docs, dEntry)
			}
			relationships = append(relationships, rels...)

		case *ast.GenDecl:
			syms, dEntries, rels := extractGenDecl(fset, d, cleanRel, pkgName, pkgPath)
			symbols = append(symbols, syms...)
			docs = append(docs, dEntries...)
			relationships = append(relationships, rels...)
		}
	}

	return symbols, docs, relationships, diagnostics
}

func extractFuncDecl(
	fset *token.FileSet,
	fd *ast.FuncDecl,
	filePath, pkgName, pkgPath string,
) (*Symbol, *DocEntry, []*SymbolRelationship) {
	if fd.Name == nil {
		return nil, nil, nil
	}

	fnName := fd.Name.Name
	fnPos := fset.Position(fd.Pos())
	posModel := NewSourcePosition(filePath, fnPos.Line, fnPos.Column, fnPos.Offset)

	kind := SymbolKindFunction
	var (
		recvType      string
		isPointerRecv bool
		relationships []*SymbolRelationship
	)

	// Check if method
	if fd.Recv != nil && len(fd.Recv.List) > 0 {
		kind = SymbolKindMethod
		field := fd.Recv.List[0]
		recvType = extractExprString(field.Type)
		isPointerRecv = strings.HasPrefix(recvType, "*")
	}

	// Extract attached doc comment
	var docEntry *DocEntry
	if fd.Doc != nil {
		docText := strings.TrimSpace(fd.Doc.Text())
		if docText != "" {
			dPos := fset.Position(fd.Doc.Pos())
			dKind := DocKindFunction
			if kind == SymbolKindMethod {
				dKind = DocKindMethod
			}
			docEntry = NewDocEntry(
				fmt.Sprintf("doc:%s:%s", filePath, fnName),
				"",
				dKind,
				docText,
				fd.Doc.Text(),
				NewSourcePosition(filePath, dPos.Line, dPos.Column, dPos.Offset),
			)
		}
	}

	// Extract generics (type parameters)
	var generics []string
	if fd.Type.TypeParams != nil {
		for _, f := range fd.Type.TypeParams.List {
			constraint := extractExprString(f.Type)
			for _, id := range f.Names {
				gStr := id.Name
				if constraint != "" {
					gStr += " " + constraint
				}
				generics = append(generics, gStr)
			}
		}
	}

	sig := extractFuncSignature(fd.Type)

	sym := NewSymbol(
		"",
		kind,
		fnName,
		pkgName,
		pkgPath,
		filePath,
		recvType,
		isPointerRecv,
		sig,
		"",
		false,
		generics,
		nil,
		posModel,
		docEntry,
	)

	// Build relationships
	if kind == SymbolKindMethod {
		baseRecv := strings.TrimPrefix(recvType, "*")
		var recvSymID string
		if pkgPath != "" && pkgPath != "." {
			recvSymID = fmt.Sprintf("%s.%s", pkgPath, baseRecv)
		} else {
			recvSymID = baseRecv
		}
		relationships = append(relationships, NewSymbolRelationship(
			sym.ID(),
			recvSymID,
			RelMethodReceiver,
			"go_method_receiver_binding",
			posModel,
		))
	} else {
		var ownerID string
		if pkgPath != "" && pkgPath != "." {
			ownerID = pkgPath
		} else {
			ownerID = "."
		}
		relationships = append(relationships, NewSymbolRelationship(
			ownerID,
			sym.ID(),
			RelFunctionOwnership,
			"package_function_declaration",
			posModel,
		))
	}

	// Build generic constraint relationships
	if fd.Type.TypeParams != nil {
		for _, f := range fd.Type.TypeParams.List {
			constraint := extractExprString(f.Type)
			if constraint != "" && constraint != "any" && constraint != "comparable" {
				relationships = append(relationships, NewSymbolRelationship(
					sym.ID(),
					constraint,
					RelGenericConstraint,
					"type_parameter_constraint",
					posModel,
				))
			}
		}
	}

	return sym, docEntry, relationships
}

func extractGenDecl(
	fset *token.FileSet,
	gd *ast.GenDecl,
	filePath, pkgName, pkgPath string,
) ([]*Symbol, []*DocEntry, []*SymbolRelationship) {
	var (
		symbols       []*Symbol
		docs          []*DocEntry
		relationships []*SymbolRelationship
	)

	for _, spec := range gd.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			sym, dEntry, rels := extractTypeSpec(fset, gd, s, filePath, pkgName, pkgPath)
			if sym != nil {
				symbols = append(symbols, sym)
			}
			if dEntry != nil {
				docs = append(docs, dEntry)
			}
			relationships = append(relationships, rels...)

		case *ast.ValueSpec:
			syms, dEntries := extractValueSpec(fset, gd, s, filePath, pkgName, pkgPath)
			symbols = append(symbols, syms...)
			docs = append(docs, dEntries...)
		}
	}

	return symbols, docs, relationships
}

func extractTypeSpec(
	fset *token.FileSet,
	gd *ast.GenDecl,
	ts *ast.TypeSpec,
	filePath, pkgName, pkgPath string,
) (*Symbol, *DocEntry, []*SymbolRelationship) {
	if ts.Name == nil {
		return nil, nil, nil
	}

	name := ts.Name.Name
	pos := fset.Position(ts.Pos())
	posModel := NewSourcePosition(filePath, pos.Line, pos.Column, pos.Offset)

	var (
		kind          = SymbolKindType
		isAlias       = ts.Assign != token.NoPos
		fields        []string
		generics      []string
		relationships []*SymbolRelationship
		typeDef       = extractExprString(ts.Type)
	)

	if isAlias {
		kind = SymbolKindAlias
	}

	// Attached doc comment
	var docEntry *DocEntry
	docGroup := ts.Doc
	if docGroup == nil {
		docGroup = gd.Doc
	}
	if docGroup != nil {
		docText := strings.TrimSpace(docGroup.Text())
		if docText != "" {
			dPos := fset.Position(docGroup.Pos())
			dKind := DocKindGeneral
			docEntry = NewDocEntry(
				fmt.Sprintf("doc:%s:%s", filePath, name),
				"",
				dKind,
				docText,
				docGroup.Text(),
				NewSourcePosition(filePath, dPos.Line, dPos.Column, dPos.Offset),
			)
		}
	}

	// Extract generics
	if ts.TypeParams != nil {
		for _, f := range ts.TypeParams.List {
			constraint := extractExprString(f.Type)
			for _, id := range f.Names {
				gStr := id.Name
				if constraint != "" {
					gStr += " " + constraint
				}
				generics = append(generics, gStr)
			}
		}
	}

	var parentID string
	if pkgPath != "" && pkgPath != "." {
		parentID = fmt.Sprintf("%s.%s", pkgPath, name)
	} else {
		parentID = name
	}

	// Inspect specific type structures
	switch t := ts.Type.(type) {
	case *ast.StructType:
		kind = SymbolKindStruct
		if docEntry != nil {
			docEntry.kind = DocKindStruct
		}
		if t.Fields != nil {
			for _, f := range t.Fields.List {
				fType := extractExprString(f.Type)
				if len(f.Names) == 0 {
					// Embedded field
					fields = append(fields, fType)
					relationships = append(relationships, NewSymbolRelationship(
						parentID,
						fType,
						RelStructEmbedding,
						"struct_embedded_field",
						posModel,
					))
				} else {
					for _, id := range f.Names {
						fields = append(fields, id.Name+": "+fType)
					}
				}
			}
		}

	case *ast.InterfaceType:
		kind = SymbolKindInterface
		if docEntry != nil {
			docEntry.kind = DocKindInterface
		}
		if t.Methods != nil {
			for _, m := range t.Methods.List {
				mType := extractExprString(m.Type)
				if len(m.Names) == 0 {
					// Embedded interface
					fields = append(fields, "embedded: "+mType)
					relationships = append(relationships, NewSymbolRelationship(
						parentID,
						mType,
						RelStructEmbedding,
						"interface_embedded_interface",
						posModel,
					))
				} else {
					for _, id := range m.Names {
						fields = append(fields, id.Name+" "+mType)
					}
				}
			}
		}
	}

	sym := NewSymbol(
		"",
		kind,
		name,
		pkgName,
		pkgPath,
		filePath,
		"",
		false,
		typeDef,
		typeDef,
		isAlias,
		generics,
		fields,
		posModel,
		docEntry,
	)

	// Record type alias relationship
	if isAlias && typeDef != "" {
		relationships = append(relationships, NewSymbolRelationship(
			sym.ID(),
			typeDef,
			RelTypeAlias,
			"type_alias_declaration",
			posModel,
		))
	}

	// Record generic constraint relationships
	if ts.TypeParams != nil {
		for _, f := range ts.TypeParams.List {
			constraint := extractExprString(f.Type)
			if constraint != "" && constraint != "any" && constraint != "comparable" {
				relationships = append(relationships, NewSymbolRelationship(
					sym.ID(),
					constraint,
					RelGenericConstraint,
					"type_parameter_constraint",
					posModel,
				))
			}
		}
	}

	return sym, docEntry, relationships
}

func extractValueSpec(
	fset *token.FileSet,
	gd *ast.GenDecl,
	vs *ast.ValueSpec,
	filePath, pkgName, pkgPath string,
) ([]*Symbol, []*DocEntry) {
	var (
		symbols []*Symbol
		docs    []*DocEntry
	)

	kind := SymbolKindVariable
	if gd.Tok == token.CONST {
		kind = SymbolKindConstant
	}

	typeStr := ""
	if vs.Type != nil {
		typeStr = extractExprString(vs.Type)
	}

	// Attached doc comment
	var docEntry *DocEntry
	docGroup := vs.Doc
	if docGroup == nil {
		docGroup = gd.Doc
	}
	if docGroup != nil {
		docText := strings.TrimSpace(docGroup.Text())
		if docText != "" {
			dPos := fset.Position(docGroup.Pos())
			docEntry = NewDocEntry(
				fmt.Sprintf("doc:%s:val", filePath),
				"",
				DocKindGeneral,
				docText,
				docGroup.Text(),
				NewSourcePosition(filePath, dPos.Line, dPos.Column, dPos.Offset),
			)
			docs = append(docs, docEntry)
		}
	}

	for _, id := range vs.Names {
		name := id.Name
		pos := fset.Position(id.Pos())
		posModel := NewSourcePosition(filePath, pos.Line, pos.Column, pos.Offset)

		sym := NewSymbol(
			"",
			kind,
			name,
			pkgName,
			pkgPath,
			filePath,
			"",
			false,
			typeStr,
			typeStr,
			false,
			nil,
			nil,
			posModel,
			docEntry,
		)
		symbols = append(symbols, sym)
	}

	return symbols, docs
}

func extractExprString(expr ast.Expr) string {
	if expr == nil {
		return ""
	}
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return "*" + extractExprString(e.X)
	case *ast.SelectorExpr:
		return extractExprString(e.X) + "." + e.Sel.Name
	case *ast.ArrayType:
		lenStr := ""
		if e.Len != nil {
			lenStr = extractExprString(e.Len)
		}
		return "[" + lenStr + "]" + extractExprString(e.Elt)
	case *ast.MapType:
		return "map[" + extractExprString(e.Key) + "]" + extractExprString(e.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.StructType:
		return "struct{}"
	case *ast.FuncType:
		return extractFuncSignature(e)
	case *ast.IndexExpr:
		return extractExprString(e.X) + "[" + extractExprString(e.Index) + "]"
	case *ast.IndexListExpr:
		var indices []string
		for _, idx := range e.Indices {
			indices = append(indices, extractExprString(idx))
		}
		return extractExprString(e.X) + "[" + strings.Join(indices, ", ") + "]"
	case *ast.BasicLit:
		return e.Value
	}
	return ""
}

func extractFuncSignature(ft *ast.FuncType) string {
	if ft == nil {
		return "func()"
	}

	var params []string
	if ft.Params != nil {
		for _, p := range ft.Params.List {
			pType := extractExprString(p.Type)
			if len(p.Names) == 0 {
				params = append(params, pType)
			} else {
				for _, n := range p.Names {
					params = append(params, n.Name+" "+pType)
				}
			}
		}
	}

	var results []string
	if ft.Results != nil {
		for _, r := range ft.Results.List {
			rType := extractExprString(r.Type)
			if len(r.Names) == 0 {
				results = append(results, rType)
			} else {
				for _, n := range r.Names {
					results = append(results, n.Name+" "+rType)
				}
			}
		}
	}

	resStr := ""
	if len(results) == 1 && !strings.Contains(results[0], " ") {
		resStr = " " + results[0]
	} else if len(results) > 0 {
		resStr = " (" + strings.Join(results, ", ") + ")"
	}

	return fmt.Sprintf("func(%s)%s", strings.Join(params, ", "), resStr)
}
