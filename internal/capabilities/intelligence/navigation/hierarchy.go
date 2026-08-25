package navigation

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/intelligence/crossrepo"
	"github.com/unhield/limoxel/internal/capabilities/intelligence/semantic"
	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
)

// HierarchyNavigator provides deterministic symbol, interface, type, and package hierarchy navigation.
type HierarchyNavigator struct {
	symbolDB   *symbol.SymbolDatabase
	semModel   *semantic.SemanticModel
	crossModel *crossrepo.CrossRepoModel
}

// NewHierarchyNavigator constructs a HierarchyNavigator.
func NewHierarchyNavigator(
	symDB *symbol.SymbolDatabase,
	semModel *semantic.SemanticModel,
	crossModel *crossrepo.CrossRepoModel,
) *HierarchyNavigator {
	return &HierarchyNavigator{
		symbolDB:   symDB,
		semModel:   semModel,
		crossModel: crossModel,
	}
}

// GetParentSymbols resolves the parent structural or containing entity for a symbol.
func (n *HierarchyNavigator) GetParentSymbols(symbolID string) ([]*NavigationTarget, error) {
	cleanID := strings.TrimSpace(symbolID)
	if cleanID == "" {
		return nil, ErrEmptyTarget
	}

	var parents []*NavigationTarget

	if n.symbolDB != nil {
		sym := n.symbolDB.SymbolByID(cleanID)
		if sym != nil {
			// 1. If method with receiver, the receiver struct/type is the direct parent
			if sym.ReceiverType() != "" {
				cleanRecv := strings.TrimPrefix(sym.ReceiverType(), "*")
				for _, s := range n.symbolDB.AllSymbols() {
					if s != nil && s.PackagePath() == sym.PackagePath() && s.Name() == cleanRecv {
						parentTgt := NewNavigationTarget(
							"parent:"+s.ID(),
							s.ID(),
							s.Name(),
							string(s.Kind()),
							s.FilePath(),
							s.PackagePath(),
							"",
							"",
							s.Position(),
							NavStateValid,
							NavKindHierarchyParent,
							"method_receiver_parent",
						)
						parents = append(parents, parentTgt)
					}
				}
			}

			// 2. Containing package is always a structural parent
			pkgTgt := NewNavigationTarget(
				"parent:pkg:"+sym.PackagePath(),
				"",
				sym.PackageName(),
				"package",
				filepath.Dir(sym.FilePath()),
				sym.PackagePath(),
				"",
				"",
				nil,
				NavStateValid,
				NavKindHierarchyParent,
				"containing_package",
			)
			parents = append(parents, pkgTgt)
		}
	}

	sort.Slice(parents, func(i, j int) bool {
		return parents[i].ID() < parents[j].ID()
	})

	return parents, nil
}

// GetChildSymbols resolves children composed within or attached to a symbol.
func (n *HierarchyNavigator) GetChildSymbols(symbolID string) ([]*NavigationTarget, error) {
	cleanID := strings.TrimSpace(symbolID)
	if cleanID == "" {
		return nil, ErrEmptyTarget
	}

	var children []*NavigationTarget

	if n.symbolDB != nil {
		targetSym := n.symbolDB.SymbolByID(cleanID)
		if targetSym != nil {
			// 1. If target is a struct/type, find all methods attached to it
			for _, s := range n.symbolDB.AllSymbols() {
				if s == nil {
					continue
				}
				if s.ReceiverType() != "" {
					recv := strings.TrimPrefix(s.ReceiverType(), "*")
					if recv == targetSym.Name() && s.PackagePath() == targetSym.PackagePath() {
						childTgt := NewNavigationTarget(
							"child:"+s.ID(),
							s.ID(),
							s.Name(),
							string(s.Kind()),
							s.FilePath(),
							s.PackagePath(),
							"",
							"",
							s.Position(),
							NavStateValid,
							NavKindHierarchyChild,
							"attached_method",
						)
						children = append(children, childTgt)
					}
				}
			}

			// 2. Fields of struct / Interface methods
			for _, fieldName := range targetSym.Fields() {
				fTgt := NewNavigationTarget(
					"child:"+targetSym.ID()+":"+fieldName,
					targetSym.ID()+":"+fieldName,
					fieldName,
					"field_or_method_member",
					targetSym.FilePath(),
					targetSym.PackagePath(),
					"",
					"",
					targetSym.Position(),
					NavStateValid,
					NavKindHierarchyChild,
					"declared_member",
				)
				children = append(children, fTgt)
			}
		}
	}

	sort.Slice(children, func(i, j int) bool {
		return children[i].ID() < children[j].ID()
	})

	return children, nil
}

// GetInterfaceHierarchy navigates interface embedding and implementing entities.
func (n *HierarchyNavigator) GetInterfaceHierarchy(interfaceID string) (*InterfaceHierarchyNode, error) {
	cleanID := strings.TrimSpace(interfaceID)
	if cleanID == "" {
		return nil, ErrEmptyTarget
	}

	name := cleanID
	pkgPath := ""
	var embedded []string
	var implementors []string

	if n.semModel != nil {
		ifaceType := n.semModel.TypeByID(cleanID)
		if ifaceType != nil {
			name = ifaceType.Name()
			pkgPath = ifaceType.PackagePath()
			embedded = ifaceType.EmbeddedTypes()
			for _, other := range n.semModel.AllTypes() {
				if other == nil || other.Kind() == semantic.TypeInterface {
					continue
				}
				for _, iface := range other.ImplementedInterfaces() {
					if iface == ifaceType.Name() || iface == ifaceType.ID() {
						implementors = append(implementors, other.ID())
					}
				}
			}
		}
	}

	if n.symbolDB != nil && len(implementors) == 0 {
		sym := n.symbolDB.SymbolByID(cleanID)
		if sym != nil {
			name = sym.Name()
			pkgPath = sym.PackagePath()
		}
	}

	return NewInterfaceHierarchyNode(cleanID, name, pkgPath, embedded, implementors), nil
}

// GetTypeHierarchy navigates type definitions, embedding, aliases, and implementations.
func (n *HierarchyNavigator) GetTypeHierarchy(typeID string) (*TypeHierarchyNode, error) {
	cleanID := strings.TrimSpace(typeID)
	if cleanID == "" {
		return nil, ErrEmptyTarget
	}

	name := cleanID
	pkgPath := ""
	baseType := ""
	aliasedType := ""
	var embedded []string
	var implementations []string

	if n.semModel != nil {
		t := n.semModel.TypeByID(cleanID)
		if t != nil {
			name = t.Name()
			pkgPath = t.PackagePath()
			baseType = t.UnderlyingType()
			aliasedType = t.AliasTarget()
			embedded = t.EmbeddedTypes()
			implementations = t.ImplementedInterfaces()
		}
	}

	if n.symbolDB != nil {
		sym := n.symbolDB.SymbolByID(cleanID)
		if sym != nil {
			name = sym.Name()
			pkgPath = sym.PackagePath()
			if sym.IsAlias() {
				aliasedType = sym.TypeDefinition()
			}
		}
	}

	return NewTypeHierarchyNode(cleanID, name, pkgPath, baseType, aliasedType, embedded, implementations), nil
}

// GetPackageHierarchy navigates the package containment hierarchy.
func (n *HierarchyNavigator) GetPackageHierarchy(packagePath string) (*PackageHierarchyNode, error) {
	cleanPkg := filepath.ToSlash(filepath.Clean(packagePath))
	if cleanPkg == "" || cleanPkg == "." {
		return nil, ErrEmptyTarget
	}

	var files []string
	var childPkgs []string
	var exportedSyms []string

	// 1. Files and Exported Symbols from SymbolDB
	if n.symbolDB != nil {
		fileMap := make(map[string]bool)
		for _, sym := range n.symbolDB.AllSymbols() {
			if sym == nil {
				continue
			}
			symPkg := filepath.ToSlash(filepath.Clean(sym.PackagePath()))
			if symPkg == cleanPkg {
				if sym.FilePath() != "" {
					fileMap[sym.FilePath()] = true
				}
				if sym.IsExported() {
					exportedSyms = append(exportedSyms, sym.Name())
				}
			} else if strings.HasPrefix(symPkg, cleanPkg+"/") {
				// Child sub-package
				sub := strings.TrimPrefix(symPkg, cleanPkg+"/")
				parts := strings.Split(sub, "/")
				childPkgs = append(childPkgs, cleanPkg+"/"+parts[0])
			}
		}
		for f := range fileMap {
			files = append(files, f)
		}
	}

	modPath := ""
	repoRoot := ""
	if n.crossModel != nil && n.crossModel.Workspace() != nil {
		for _, r := range n.crossModel.Workspace().Repositories() {
			for _, p := range r.Packages() {
				if p == cleanPkg {
					repoRoot = r.Root()
					if len(r.Modules()) > 0 {
						modPath = r.Modules()[0]
					}
					break
				}
			}
		}
	}

	return NewPackageHierarchyNode(cleanPkg, modPath, repoRoot, files, childPkgs, exportedSyms), nil
}
