package xref

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
)

var goBuiltins = map[string]bool{
	"append": true, "cap": true, "clear": true, "close": true, "complex": true,
	"copy": true, "delete": true, "imag": true, "len": true, "make": true,
	"max": true, "min": true, "new": true, "panic": true, "print": true,
	"println": true, "real": true, "recover": true, "error": true,
	"bool": true, "byte": true, "complex64": true, "complex128": true,
	"float32": true, "float64": true, "int": true, "int8": true,
	"int16": true, "int32": true, "int64": true, "rune": true,
	"string": true, "uint": true, "uint8": true, "uint16": true,
	"uint32": true, "uint64": true, "uintptr": true, "nil": true,
	"true": true, "false": true, "iota": true,
}

// parseGoFileXRef extracts cross-references and call graph edges from a single Go source file.
func parseGoFileXRef(
	absPath string,
	relPath string,
	symModel *symbol.SymbolModel,
) ([]*Reference, []*CallEdge, []*discovery.Diagnostic) {
	var (
		references  []*Reference
		callEdges   []*CallEdge
		diagnostics []*discovery.Diagnostic
	)

	cleanRel := filepath.ToSlash(filepath.Clean(relPath))
	fset := token.NewFileSet()

	astFile, err := parser.ParseFile(fset, absPath, nil, parser.AllErrors)
	if err != nil {
		diag := discovery.NewDiagnostic(
			discovery.SeverityWarning,
			"XREF_SYNTAX_WARNING",
			fmt.Sprintf("xref: syntax errors while parsing %s: %v", cleanRel, err),
			cleanRel,
			false,
		)
		diagnostics = append(diagnostics, diag)
		if astFile == nil {
			return nil, nil, diagnostics
		}
	}

	cleanDir := filepath.ToSlash(filepath.Dir(cleanRel))
	pkgPath := cleanDir
	if pkgPath == "." || pkgPath == "" {
		pkgPath = "."
	}

	// Map imported packages: alias/name -> importPath
	importMap := make(map[string]string)
	for _, imp := range astFile.Imports {
		if imp.Path == nil {
			continue
		}
		rawPath := strings.Trim(imp.Path.Value, `"`)
		cleanImpPath := filepath.ToSlash(filepath.Clean(rawPath))

		alias := ""
		if imp.Name != nil {
			alias = imp.Name.Name
		} else {
			parts := strings.Split(cleanImpPath, "/")
			alias = parts[len(parts)-1]
		}
		if alias != "" && alias != "_" {
			importMap[alias] = cleanImpPath
		}
	}

	symDB := symModel.Symbols()

	// Walk top-level declarations
	for _, decl := range astFile.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			funcRefs, funcCalls := analyzeFuncDecl(fset, d, cleanRel, pkgPath, importMap, symDB)
			references = append(references, funcRefs...)
			callEdges = append(callEdges, funcCalls...)

		case *ast.GenDecl:
			genRefs := analyzeGenDecl(fset, d, cleanRel, pkgPath, importMap, symDB)
			references = append(references, genRefs...)
		}
	}

	return references, callEdges, diagnostics
}

func analyzeFuncDecl(
	fset *token.FileSet,
	fd *ast.FuncDecl,
	cleanRel string,
	pkgPath string,
	importMap map[string]string,
	symDB *symbol.SymbolDatabase,
) ([]*Reference, []*CallEdge) {
	if fd == nil || fd.Name == nil {
		return nil, nil
	}

	var (
		refs  []*Reference
		calls []*CallEdge
	)

	// Determine enclosing caller symbol ID
	var callerID string
	var recvType string
	if fd.Recv != nil && len(fd.Recv.List) > 0 {
		recvType = extractReceiverType(fd.Recv.List[0].Type)
		baseRecv := strings.TrimPrefix(recvType, "*")
		if pkgPath != "" && pkgPath != "." {
			callerID = fmt.Sprintf("%s.(%s).%s", pkgPath, recvType, fd.Name.Name)
		} else {
			callerID = fmt.Sprintf("(%s).%s", recvType, fd.Name.Name)
		}
		// Record reference to receiver type
		var recvTargetID string
		if pkgPath != "" && pkgPath != "." {
			recvTargetID = fmt.Sprintf("%s.%s", pkgPath, baseRecv)
		} else {
			recvTargetID = baseRecv
		}
		pos := fset.Position(fd.Recv.Pos())
		refs = append(refs, NewReference(
			callerID,
			recvTargetID,
			RefType,
			cleanRel,
			symbol.NewSourcePosition(cleanRel, pos.Line, pos.Column, pos.Offset),
			StateResolved,
			"method_receiver_type_binding",
		))
	} else {
		if pkgPath != "" && pkgPath != "." {
			callerID = fmt.Sprintf("%s.%s", pkgPath, fd.Name.Name)
		} else {
			callerID = fd.Name.Name
		}
	}

	if fd.Body == nil {
		return refs, calls
	}

	// Inspect AST inside function body
	ast.Inspect(fd.Body, func(node ast.Node) bool {
		if node == nil {
			return true
		}

		switch n := node.(type) {
		case *ast.CallExpr:
			callRef, callEdge := resolveCallExpr(fset, n, callerID, cleanRel, pkgPath, recvType, importMap, symDB)
			if callRef != nil {
				refs = append(refs, callRef)
			}
			if callEdge != nil {
				calls = append(calls, callEdge)
			}

		case *ast.CompositeLit:
			if n.Type != nil {
				litRef := resolveTypeRef(fset, n.Type, callerID, cleanRel, pkgPath, importMap, symDB, RefStruct, "struct_composite_literal")
				if litRef != nil {
					refs = append(refs, litRef)
				}
			}

		case *ast.TypeAssertExpr:
			if n.Type != nil {
				assertRef := resolveTypeRef(fset, n.Type, callerID, cleanRel, pkgPath, importMap, symDB, RefInterface, "type_assertion")
				if assertRef != nil {
					refs = append(refs, assertRef)
				}
			}
		}

		return true
	})

	return refs, calls
}

func analyzeGenDecl(
	fset *token.FileSet,
	gd *ast.GenDecl,
	cleanRel string,
	pkgPath string,
	importMap map[string]string,
	symDB *symbol.SymbolDatabase,
) []*Reference {
	var refs []*Reference

	for _, spec := range gd.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			if s.Name == nil {
				continue
			}
			var sourceID string
			if pkgPath != "" && pkgPath != "." {
				sourceID = fmt.Sprintf("%s.%s", pkgPath, s.Name.Name)
			} else {
				sourceID = s.Name.Name
			}

			if s.Type != nil {
				tRef := resolveTypeRef(fset, s.Type, sourceID, cleanRel, pkgPath, importMap, symDB, RefType, "type_declaration_underlying")
				if tRef != nil {
					refs = append(refs, tRef)
				}
			}

		case *ast.ValueSpec:
			for _, id := range s.Names {
				var sourceID string
				if pkgPath != "" && pkgPath != "." {
					sourceID = fmt.Sprintf("%s.%s", pkgPath, id.Name)
				} else {
					sourceID = id.Name
				}
				if s.Type != nil {
					tRef := resolveTypeRef(fset, s.Type, sourceID, cleanRel, pkgPath, importMap, symDB, RefType, "variable_type_annotation")
					if tRef != nil {
						refs = append(refs, tRef)
					}
				}
			}
		}
	}

	return refs
}

func resolveCallExpr(
	fset *token.FileSet,
	call *ast.CallExpr,
	callerID string,
	cleanRel string,
	pkgPath string,
	enclosingRecv string,
	importMap map[string]string,
	symDB *symbol.SymbolDatabase,
) (*Reference, *CallEdge) {
	if call == nil || call.Fun == nil {
		return nil, nil
	}

	pos := fset.Position(call.Pos())
	sourcePos := symbol.NewSourcePosition(cleanRel, pos.Line, pos.Column, pos.Offset)

	switch fun := call.Fun.(type) {
	case *ast.Ident:
		// Direct function call in same package or builtin
		name := fun.Name
		if goBuiltins[name] {
			return nil, nil
		}

		var targetID string
		if pkgPath != "" && pkgPath != "." {
			targetID = fmt.Sprintf("%s.%s", pkgPath, name)
		} else {
			targetID = name
		}

		callKind := CallDirect
		if targetID == callerID {
			callKind = CallRecursiveDirect
		}

		state := StateResolved
		if symDB != nil && symDB.SymbolByID(targetID) == nil {
			// Check if target is a method in same struct
			if enclosingRecv != "" {
				methodID := fmt.Sprintf("%s.(%s).%s", pkgPath, enclosingRecv, name)
				if pkgPath == "." {
					methodID = fmt.Sprintf("(%s).%s", enclosingRecv, name)
				}
				if symDB.SymbolByID(methodID) != nil {
					targetID = methodID
					callKind = CallMethod
					state = StateResolved
				} else {
					state = StateUnresolvedExternal
				}
			} else {
				state = StateUnresolvedExternal
			}
		}

		ref := NewReference(
			callerID,
			targetID,
			RefFunction,
			cleanRel,
			sourcePos,
			state,
			"static_function_call",
		)

		edge := NewCallEdge(
			callerID,
			targetID,
			callKind,
			cleanRel,
			sourcePos,
		)

		return ref, edge

	case *ast.SelectorExpr:
		methodName := fun.Sel.Name
		if pkgIdent, ok := fun.X.(*ast.Ident); ok {
			// Check if pkgIdent is an imported package
			if impPath, isImport := importMap[pkgIdent.Name]; isImport {
				targetID := fmt.Sprintf("%s.%s", impPath, methodName)
				state := StateResolved
				if symDB != nil && symDB.SymbolByID(targetID) == nil {
					state = StateUnresolvedExternal
				}

				ref := NewReference(
					callerID,
					targetID,
					RefFunction,
					cleanRel,
					sourcePos,
					state,
					"imported_package_function_call",
				)

				edge := NewCallEdge(
					callerID,
					targetID,
					CallDirect,
					cleanRel,
					sourcePos,
				)

				return ref, edge
			}

			// Method invocation on local identifier / receiver
			var targetID string
			if pkgIdent.Name == "s" || pkgIdent.Name == "this" || pkgIdent.Name == "self" || enclosingRecv != "" {
				if pkgPath != "" && pkgPath != "." {
					targetID = fmt.Sprintf("%s.(%s).%s", pkgPath, enclosingRecv, methodName)
				} else {
					targetID = fmt.Sprintf("(%s).%s", enclosingRecv, methodName)
				}
			} else {
				targetID = fmt.Sprintf("%s.%s", pkgIdent.Name, methodName)
			}

			callKind := CallMethod
			state := StateResolved
			if symDB != nil && symDB.SymbolByID(targetID) == nil {
				// Search for matching method by name
				candidates := symDB.SymbolsByName(methodName)
				if len(candidates) == 1 {
					targetID = candidates[0].ID()
					state = StateResolved
				} else if len(candidates) > 1 {
					state = StateAmbiguous
				} else {
					state = StateUnresolvedExternal
				}
			}

			ref := NewReference(
				callerID,
				targetID,
				RefMethod,
				cleanRel,
				sourcePos,
				state,
				"method_invocation_dispatch",
			)

			edge := NewCallEdge(
				callerID,
				targetID,
				callKind,
				cleanRel,
				sourcePos,
			)

			return ref, edge
		}
	}

	return nil, nil
}

func resolveTypeRef(
	fset *token.FileSet,
	expr ast.Expr,
	sourceID string,
	cleanRel string,
	pkgPath string,
	importMap map[string]string,
	symDB *symbol.SymbolDatabase,
	defaultKind ReferenceKind,
	evidence string,
) *Reference {
	if expr == nil {
		return nil
	}

	pos := fset.Position(expr.Pos())
	sourcePos := symbol.NewSourcePosition(cleanRel, pos.Line, pos.Column, pos.Offset)

	switch e := expr.(type) {
	case *ast.Ident:
		name := e.Name
		if goBuiltins[name] {
			return nil
		}

		var targetID string
		if pkgPath != "" && pkgPath != "." {
			targetID = fmt.Sprintf("%s.%s", pkgPath, name)
		} else {
			targetID = name
		}

		state := StateResolved
		if symDB != nil && symDB.SymbolByID(targetID) == nil {
			state = StateUnresolvedExternal
		}

		return NewReference(
			sourceID,
			targetID,
			defaultKind,
			cleanRel,
			sourcePos,
			state,
			evidence,
		)

	case *ast.SelectorExpr:
		if pkgIdent, ok := e.X.(*ast.Ident); ok {
			if impPath, isImport := importMap[pkgIdent.Name]; isImport {
				targetID := fmt.Sprintf("%s.%s", impPath, e.Sel.Name)
				state := StateResolved
				if symDB != nil && symDB.SymbolByID(targetID) == nil {
					state = StateUnresolvedExternal
				}
				return NewReference(
					sourceID,
					targetID,
					defaultKind,
					cleanRel,
					sourcePos,
					state,
					evidence,
				)
			}
		}

	case *ast.StarExpr:
		return resolveTypeRef(fset, e.X, sourceID, cleanRel, pkgPath, importMap, symDB, defaultKind, evidence)

	case *ast.IndexExpr:
		return resolveTypeRef(fset, e.X, sourceID, cleanRel, pkgPath, importMap, symDB, defaultKind, evidence)

	case *ast.IndexListExpr:
		return resolveTypeRef(fset, e.X, sourceID, cleanRel, pkgPath, importMap, symDB, defaultKind, evidence)
	}

	return nil
}

func extractReceiverType(expr ast.Expr) string {
	if expr == nil {
		return ""
	}
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return "*" + extractReceiverType(e.X)
	}
	return ""
}

// computeRecursionAndReachability analyzes the call graph to detect cycles and reachability.
func computeRecursionAndReachability(
	edges []*CallEdge,
	allSymbols []*symbol.Symbol,
) ([]string, []string, [][]string, []string, map[string]ReachabilityState) {
	var (
		entryPoints   []string
		exitPoints    []string
		deadFunctions []string
		cycles        [][]string
	)

	reachMap := make(map[string]ReachabilityState)
	calleesMap := make(map[string][]string)
	callersMap := make(map[string][]string)
	allFuncNodes := make(map[string]bool)

	for _, s := range allSymbols {
		if s.Kind() == symbol.SymbolKindFunction || s.Kind() == symbol.SymbolKindMethod {
			allFuncNodes[s.ID()] = true
			reachMap[s.ID()] = ReachabilityUnknown
		}
	}

	for _, e := range edges {
		allFuncNodes[e.callerID] = true
		allFuncNodes[e.calleeID] = true
		calleesMap[e.callerID] = append(calleesMap[e.callerID], e.calleeID)
		callersMap[e.calleeID] = append(callersMap[e.calleeID], e.callerID)

		// Direct recursion
		if e.callerID == e.calleeID {
			cycles = append(cycles, []string{e.callerID, e.calleeID})
		}
	}

	// Identify Entry Points: main.main, init, or exported library roots with 0 callers
	for node := range allFuncNodes {
		if strings.HasSuffix(node, "main.main") || node == "main" || strings.HasSuffix(node, ".init") || node == "init" {
			entryPoints = append(entryPoints, node)
			reachMap[node] = ReachableConfirmed
		}
	}

	// Identify Exit Points: leaf functions with 0 callees or panic/exit handlers
	for node := range allFuncNodes {
		if len(calleesMap[node]) == 0 {
			exitPoints = append(exitPoints, node)
		}
	}

	// Detect Mutual Recursion via DFS cycle detection
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	path := []string{}

	var dfsCycle func(node string)
	dfsCycle = func(node string) {
		visited[node] = true
		recStack[node] = true
		path = append(path, node)

		for _, neighbor := range calleesMap[node] {
			if !visited[neighbor] {
				dfsCycle(neighbor)
			} else if recStack[neighbor] && neighbor != node {
				// Found mutual recursion cycle
				var cyc []string
				startIdx := -1
				for idx, p := range path {
					if p == neighbor {
						startIdx = idx
						break
					}
				}
				if startIdx != -1 {
					cyc = append(cyc, path[startIdx:]...)
					cyc = append(cyc, neighbor)
					cycles = append(cycles, cyc)
				}
			}
		}

		path = path[:len(path)-1]
		recStack[node] = false
	}

	for node := range allFuncNodes {
		if !visited[node] {
			dfsCycle(node)
		}
	}

	// Reachability traversal from EntryPoints via BFS
	if len(entryPoints) > 0 {
		reachQueue := append([]string{}, entryPoints...)
		reachVisited := make(map[string]bool)
		for _, ep := range entryPoints {
			reachVisited[ep] = true
		}

		for len(reachQueue) > 0 {
			curr := reachQueue[0]
			reachQueue = reachQueue[1:]

			for _, callee := range calleesMap[curr] {
				if !reachVisited[callee] {
					reachVisited[callee] = true
					reachMap[callee] = ReachableConfirmed
					reachQueue = append(reachQueue, callee)
				}
			}
		}

		// Functions not visited from entry points
		for node := range allFuncNodes {
			if !reachVisited[node] {
				if len(callersMap[node]) == 0 {
					// In main package or root package, or unexported function in non-main packages
					if !isExportedSymbol(node) || strings.HasPrefix(node, "main.") || !strings.Contains(node, "/") {
						reachMap[node] = UnreachableConfirmed
						deadFunctions = append(deadFunctions, node)
					} else {
						reachMap[node] = ReachabilityUnknown
					}
				} else {
					reachMap[node] = ReachabilityUnknown
				}
			}
		}
	}

	sort.Strings(entryPoints)
	sort.Strings(exitPoints)
	sort.Strings(deadFunctions)

	return entryPoints, exitPoints, cycles, deadFunctions, reachMap
}

func isExportedSymbol(symbolID string) bool {
	parts := strings.Split(symbolID, ".")
	if len(parts) == 0 {
		return false
	}
	last := parts[len(parts)-1]
	if len(last) == 0 {
		return false
	}
	return last[0] >= 'A' && last[0] <= 'Z'
}
