package navigation

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/intelligence/crossrepo"
	"github.com/unhield/limoxel/internal/capabilities/intelligence/semantic"
	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
	"github.com/unhield/limoxel/internal/capabilities/repository/xref"
)

// DefinitionNavigator provides deterministic navigation to definitions, declarations, implementations, packages, and modules.
type DefinitionNavigator struct {
	symbolDB   *symbol.SymbolDatabase
	xrefModel  *xref.XRefModel
	semModel   *semantic.SemanticModel
	crossModel *crossrepo.CrossRepoModel
}

// NewDefinitionNavigator constructs a DefinitionNavigator.
func NewDefinitionNavigator(
	symDB *symbol.SymbolDatabase,
	xrefModel *xref.XRefModel,
	semModel *semantic.SemanticModel,
	crossModel *crossrepo.CrossRepoModel,
) *DefinitionNavigator {
	return &DefinitionNavigator{
		symbolDB:   symDB,
		xrefModel:  xrefModel,
		semModel:   semModel,
		crossModel: crossModel,
	}
}

// GoToDefinition resolves the authoritative definition target for a symbol or entity identifier.
func (n *DefinitionNavigator) GoToDefinition(symbolID string) (*DefinitionResult, error) {
	cleanID := strings.TrimSpace(symbolID)
	if cleanID == "" {
		return nil, ErrEmptyTarget
	}

	if n == nil || (n.symbolDB == nil && n.semModel == nil) {
		return nil, ErrNilEngine
	}

	// 1. Direct lookup by symbol ID in symbol DB
	if n.symbolDB != nil {
		sym := n.symbolDB.SymbolByID(cleanID)
		if sym != nil {
			modPath := ""
			repoPath := ""
			if n.xrefModel != nil {
				repoPath = n.xrefModel.RepositoryRoot()
			}
			tgt := NewNavigationTarget(
				"def:"+sym.ID(),
				sym.ID(),
				sym.Name(),
				string(sym.Kind()),
				sym.FilePath(),
				sym.PackagePath(),
				modPath,
				repoPath,
				sym.Position(),
				NavStateValid,
				NavKindDefinition,
				"symbol_database",
			)
			return NewDefinitionResult(tgt, []*NavigationTarget{tgt}, NavStateValid, "symbol_database"), nil
		}

		// 2. Query candidates by name in case of name-based lookup
		var candidates []*NavigationTarget
		for _, s := range n.symbolDB.AllSymbols() {
			if s != nil && s.Name() == cleanID {
				modPath := ""
				repoPath := ""
				if n.xrefModel != nil {
					repoPath = n.xrefModel.RepositoryRoot()
				}
				c := NewNavigationTarget(
					"def:"+s.ID(),
					s.ID(),
					s.Name(),
					string(s.Kind()),
					s.FilePath(),
					s.PackagePath(),
					modPath,
					repoPath,
					s.Position(),
					NavStateValid,
					NavKindDefinition,
					"symbol_database",
				)
				candidates = append(candidates, c)
			}
		}

		if len(candidates) == 1 {
			return NewDefinitionResult(candidates[0], candidates, NavStateValid, "symbol_database"), nil
		} else if len(candidates) > 1 {
			return NewDefinitionResult(nil, candidates, NavStateAmbiguous, "symbol_database:multiple_matches"), ErrTargetAmbiguous
		}
	}

	// 3. Fallback to Semantic Model
	if n.semModel != nil {
		sym := n.semModel.SymbolByID(cleanID)
		if sym != nil {
			pos := symbol.NewSourcePosition(sym.FilePath(), sym.Line(), 1, 0)
			tgt := NewNavigationTarget(
				"def:"+sym.ID(),
				sym.ID(),
				sym.Name(),
				string(sym.Kind()),
				sym.FilePath(),
				sym.PackagePath(),
				"",
				"",
				pos,
				NavStateValid,
				NavKindDefinition,
				"semantic_model",
			)
			return NewDefinitionResult(tgt, []*NavigationTarget{tgt}, NavStateValid, "semantic_model"), nil
		}
	}

	return nil, ErrSymbolNotFound
}

// GoToDeclaration resolves declaration targets separately from definitions where distinguished.
func (n *DefinitionNavigator) GoToDeclaration(symbolID string) (*DefinitionResult, error) {
	cleanID := strings.TrimSpace(symbolID)
	if cleanID == "" {
		return nil, ErrEmptyTarget
	}

	// In Go semantics, interface method signatures and type declarations represent canonical declaration sites.
	if n.symbolDB != nil {
		sym := n.symbolDB.SymbolByID(cleanID)
		if sym != nil {
			// If method on receiver, look up corresponding interface declaration if defined
			if sym.ReceiverType() != "" && n.symbolDB != nil {
				for _, iface := range n.symbolDB.AllSymbols() {
					if iface != nil && iface.Kind() == symbol.SymbolKindInterface {
						for _, m := range iface.Fields() {
							if m == sym.Name() {
								declTgt := NewNavigationTarget(
									"decl:"+iface.ID()+":"+sym.Name(),
									iface.ID(),
									sym.Name(),
									"interface_method_declaration",
									iface.FilePath(),
									iface.PackagePath(),
									"",
									"",
									iface.Position(),
									NavStateValid,
									NavKindDeclaration,
									"interface_contract",
								)
								return NewDefinitionResult(declTgt, []*NavigationTarget{declTgt}, NavStateValid, "interface_contract"), nil
							}
						}
					}
				}
			}

			tgt := NewNavigationTarget(
				"decl:"+sym.ID(),
				sym.ID(),
				sym.Name(),
				string(sym.Kind()),
				sym.FilePath(),
				sym.PackagePath(),
				"",
				"",
				sym.Position(),
				NavStateValid,
				NavKindDeclaration,
				"symbol_declaration",
			)
			return NewDefinitionResult(tgt, []*NavigationTarget{tgt}, NavStateValid, "symbol_declaration"), nil
		}
	}

	return nil, ErrSymbolNotFound
}

// GoToImplementation resolves concrete implementations of an interface or abstraction.
func (n *DefinitionNavigator) GoToImplementation(interfaceID string) ([]*NavigationTarget, error) {
	cleanID := strings.TrimSpace(interfaceID)
	if cleanID == "" {
		return nil, ErrEmptyTarget
	}

	var results []*NavigationTarget

	// 1. Use Semantic Model Type Hierarchy if present
	if n.semModel != nil {
		ifaceType := n.semModel.TypeByID(cleanID)
		if ifaceType != nil {
			for _, t := range n.semModel.AllTypes() {
				if t == nil || t.Kind() == semantic.TypeInterface {
					continue
				}
				for _, ifaceName := range t.ImplementedInterfaces() {
					if ifaceName == ifaceType.Name() || ifaceName == ifaceType.ID() {
						pos := symbol.NewSourcePosition(t.FilePath(), 1, 1, 0)
						tgt := NewNavigationTarget(
							"impl:"+t.ID(),
							t.ID(),
							t.Name(),
							string(t.Kind()),
							t.FilePath(),
							t.PackagePath(),
							"",
							"",
							pos,
							NavStateValid,
							NavKindImplementation,
							"semantic_interface_implementation",
						)
						results = append(results, tgt)
					}
				}
			}
		}
	}

	// 2. Fallback to SymbolDB receiver methods matching interface method sets
	if len(results) == 0 && n.symbolDB != nil {
		targetSym := n.symbolDB.SymbolByID(cleanID)
		if targetSym != nil && targetSym.Kind() == symbol.SymbolKindInterface {
			reqMethods := targetSym.Fields()
			structMethods := make(map[string]map[string]bool)
			structSymbols := make(map[string]*symbol.Symbol)

			for _, s := range n.symbolDB.AllSymbols() {
				if s == nil {
					continue
				}
				if s.Kind() == symbol.SymbolKindStruct || s.Kind() == symbol.SymbolKindType {
					structSymbols[s.Name()] = s
				}
				if s.ReceiverType() != "" {
					recv := strings.TrimPrefix(s.ReceiverType(), "*")
					if structMethods[recv] == nil {
						structMethods[recv] = make(map[string]bool)
					}
					structMethods[recv][s.Name()] = true
				}
			}

			for structName, methods := range structMethods {
				implements := true
				for _, req := range reqMethods {
					if !methods[req] {
						implements = false
						break
					}
				}
				if implements && len(reqMethods) > 0 {
					sSym := structSymbols[structName]
					pos := (*symbol.SourcePosition)(nil)
					fPath := ""
					pPath := ""
					kind := "struct"
					sID := "sym:" + structName
					if sSym != nil {
						pos = sSym.Position()
						fPath = sSym.FilePath()
						pPath = sSym.PackagePath()
						kind = string(sSym.Kind())
						sID = sSym.ID()
					}
					tgt := NewNavigationTarget(
						"impl:"+sID,
						sID,
						structName,
						kind,
						fPath,
						pPath,
						"",
						"",
						pos,
						NavStateValid,
						NavKindImplementation,
						"structural_method_set",
					)
					results = append(results, tgt)
				}
			}
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].ID() < results[j].ID()
	})

	return results, nil
}

// GoToPackage resolves the containing package for a symbol, file, or package identifier.
func (n *DefinitionNavigator) GoToPackage(entityID string) (*NavigationTarget, error) {
	cleanID := strings.TrimSpace(entityID)
	if cleanID == "" {
		return nil, ErrEmptyTarget
	}

	// 1. Symbol lookup
	if n.symbolDB != nil {
		sym := n.symbolDB.SymbolByID(cleanID)
		if sym != nil {
			pkgPath := sym.PackagePath()
			tgt := NewNavigationTarget(
				"pkg:"+pkgPath,
				"",
				sym.PackageName(),
				"package",
				filepath.Dir(sym.FilePath()),
				pkgPath,
				"",
				"",
				nil,
				NavStateValid,
				NavKindPackage,
				"symbol_package_path",
			)
			return tgt, nil
		}
	}

	// 2. Direct package path lookup
	cleanPkg := filepath.ToSlash(filepath.Clean(cleanID))
	pkgName := filepath.Base(cleanPkg)
	tgt := NewNavigationTarget(
		"pkg:"+cleanPkg,
		"",
		pkgName,
		"package",
		cleanPkg,
		cleanPkg,
		"",
		"",
		nil,
		NavStateValid,
		NavKindPackage,
		"direct_package_path",
	)
	return tgt, nil
}

// GoToModule resolves the containing module for an entity or package.
func (n *DefinitionNavigator) GoToModule(entityID string) (*NavigationTarget, error) {
	cleanID := strings.TrimSpace(entityID)
	if cleanID == "" {
		return nil, ErrEmptyTarget
	}

	modPath := "main"
	if n.crossModel != nil && n.crossModel.Workspace() != nil {
		for _, repo := range n.crossModel.Workspace().Repositories() {
			matched := false
			for _, p := range repo.Packages() {
				if p == cleanID || strings.HasPrefix(cleanID, p) {
					matched = true
					break
				}
			}
			if matched && len(repo.Modules()) > 0 {
				modPath = repo.Modules()[0]
				break
			}
			for _, m := range repo.Modules() {
				if strings.Contains(cleanID, m) || m == cleanID {
					modPath = m
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}
	}

	tgt := NewNavigationTarget(
		"mod:"+modPath,
		"",
		filepath.Base(modPath),
		"module",
		modPath,
		"",
		modPath,
		"",
		nil,
		NavStateValid,
		NavKindModule,
		"module_resolution",
	)
	return tgt, nil
}
