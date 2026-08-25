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

// ReferenceNavigator provides deterministic navigation across references, usages, reverse lookups, and dependencies.
type ReferenceNavigator struct {
	symbolDB   *symbol.SymbolDatabase
	xrefModel  *xref.XRefModel
	semModel   *semantic.SemanticModel
	crossModel *crossrepo.CrossRepoModel
}

// NewReferenceNavigator constructs a ReferenceNavigator.
func NewReferenceNavigator(
	symDB *symbol.SymbolDatabase,
	xrefModel *xref.XRefModel,
	semModel *semantic.SemanticModel,
	crossModel *crossrepo.CrossRepoModel,
) *ReferenceNavigator {
	return &ReferenceNavigator{
		symbolDB:   symDB,
		xrefModel:  xrefModel,
		semModel:   semModel,
		crossModel: crossModel,
	}
}

// FindReferences identifies all engineering entities that reference a specified symbol.
func (n *ReferenceNavigator) FindReferences(symbolID string) (*ReferenceResult, error) {
	cleanID := strings.TrimSpace(symbolID)
	if cleanID == "" {
		return nil, ErrEmptyTarget
	}

	var targets []*NavigationTarget

	if n.xrefModel != nil && n.xrefModel.References() != nil {
		for _, ref := range n.xrefModel.References().AllReferences() {
			if ref == nil {
				continue
			}
			if ref.TargetSymbolID() == cleanID {
				srcSym := (*symbol.Symbol)(nil)
				if n.symbolDB != nil {
					srcSym = n.symbolDB.SymbolByID(ref.SourceSymbolID())
				}
				name := ref.SourceSymbolID()
				pkgPath := filepath.Dir(ref.FilePath())
				if srcSym != nil {
					name = srcSym.Name()
					pkgPath = srcSym.PackagePath()
				}
				tgt := NewNavigationTarget(
					"ref:"+ref.ID(),
					ref.SourceSymbolID(),
					name,
					string(ref.Kind()),
					ref.FilePath(),
					pkgPath,
					"",
					"",
					ref.Position(),
					NavStateValid,
					NavKindReference,
					"cross_reference_database",
				)
				targets = append(targets, tgt)
			}
		}
	}

	return NewReferenceResult(cleanID, targets), nil
}

// FindUsages finds actual contextual usages of an engineering entity.
func (n *ReferenceNavigator) FindUsages(symbolID string) ([]*UsageItem, error) {
	cleanID := strings.TrimSpace(symbolID)
	if cleanID == "" {
		return nil, ErrEmptyTarget
	}

	var usages []*UsageItem

	// 1. Process XRef references
	if n.xrefModel != nil && n.xrefModel.References() != nil {
		for _, ref := range n.xrefModel.References().AllReferences() {
			if ref == nil || ref.TargetSymbolID() != cleanID {
				continue
			}

			uKind := UsageKindGeneral
			switch ref.Kind() {
			case xref.RefFunction, xref.RefMethod:
				uKind = UsageKindCall
			case xref.RefStruct, xref.RefType, xref.RefInterface:
				uKind = UsageKindType
			case xref.RefVariable, xref.RefConstant:
				uKind = UsageKindField
			}

			usage := NewUsageItem(
				ref.SourceSymbolID(),
				cleanID,
				uKind,
				ref.FilePath(),
				ref.Position(),
				ref.Evidence(),
				"xref_reference",
			)
			usages = append(usages, usage)
		}

		// Process CallGraph edges for calls
		if n.xrefModel.CallGraph() != nil {
			for _, edge := range n.xrefModel.CallGraph().AllEdges() {
				if edge == nil || edge.CalleeID() != cleanID {
					continue
				}
				usage := NewUsageItem(
					edge.CallerID(),
					cleanID,
					UsageKindCall,
					edge.FilePath(),
					edge.Position(),
					"call edge invocation",
					"call_graph",
				)
				usages = append(usages, usage)
			}
		}
	}

	// 2. Process CrossRepo Package Communications
	if n.crossModel != nil {
		for _, comm := range n.crossModel.PackageCommunications() {
			if comm == nil {
				continue
			}
			for _, sym := range comm.SymbolsUsed() {
				if sym == cleanID || strings.HasSuffix(cleanID, "."+sym) {
					usage := NewUsageItem(
						comm.SourcePackage(),
						comm.TargetPackage(),
						UsageKindPackage,
						comm.SourcePackage(),
						nil,
						"package communication: "+string(comm.Kind()),
						"cross_package_analyzer",
					)
					usages = append(usages, usage)
				}
			}
		}
	}

	// Sort usages deterministically
	sort.Slice(usages, func(i, j int) bool {
		if usages[i].FilePath() == usages[j].FilePath() {
			if usages[i].Position() != nil && usages[j].Position() != nil {
				return usages[i].Position().Line() < usages[j].Position().Line()
			}
			return usages[i].ID() < usages[j].ID()
		}
		return usages[i].FilePath() < usages[j].FilePath()
	})

	return usages, nil
}

// ReverseLookup follows an engineering relationship in the opposite direction.
func (n *ReferenceNavigator) ReverseLookup(entityID string) ([]*ReverseRelationship, error) {
	cleanID := strings.TrimSpace(entityID)
	if cleanID == "" {
		return nil, ErrEmptyTarget
	}

	var results []*ReverseRelationship

	// Inbound References
	var refSources []string
	if n.xrefModel != nil && n.xrefModel.References() != nil {
		for _, ref := range n.xrefModel.References().AllReferences() {
			if ref != nil && ref.TargetSymbolID() == cleanID {
				refSources = append(refSources, ref.SourceSymbolID())
			}
		}
	}
	if len(refSources) > 0 {
		results = append(results, NewReverseRelationship(cleanID, refSources, RelKindReferences, "xref_database"))
	}

	// Inbound Calls
	var callSources []string
	if n.xrefModel != nil && n.xrefModel.CallGraph() != nil {
		for _, edge := range n.xrefModel.CallGraph().AllEdges() {
			if edge != nil && edge.CalleeID() == cleanID {
				callSources = append(callSources, edge.CallerID())
			}
		}
	}
	if len(callSources) > 0 {
		results = append(results, NewReverseRelationship(cleanID, callSources, RelKindCalls, "call_graph"))
	}

	// Inbound Implementations (which types implement this interface)
	if n.semModel != nil {
		var implSources []string
		for _, t := range n.semModel.AllTypes() {
			if t == nil {
				continue
			}
			for _, iface := range t.ImplementedInterfaces() {
				if iface == cleanID || strings.HasSuffix(cleanID, ":"+iface) {
					implSources = append(implSources, t.ID())
				}
			}
		}
		if len(implSources) > 0 {
			results = append(results, NewReverseRelationship(cleanID, implSources, RelKindImplements, "semantic_model"))
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].ID() < results[j].ID()
	})

	return results, nil
}

// DependencyLookup resolves outbound or inbound dependency relationships.
func (n *ReferenceNavigator) DependencyLookup(entityID, direction string) ([]*DependencyNavigationItem, error) {
	cleanID := strings.TrimSpace(entityID)
	if cleanID == "" {
		return nil, ErrEmptyTarget
	}

	cleanDir := strings.ToLower(strings.TrimSpace(direction))
	if cleanDir == "" {
		cleanDir = "outbound"
	}

	var results []*DependencyNavigationItem

	if n.crossModel != nil {
		// Cross-file dependencies
		for _, dep := range n.crossModel.CrossFileDependencies() {
			if dep == nil {
				continue
			}
			if cleanDir == "outbound" && dep.SourceFile() == cleanID {
				results = append(results, NewDependencyNavigationItem(
					dep.SourceFile(),
					dep.TargetFile(),
					"outbound",
					"cross_file_dependency",
					"crossrepo_intelligence",
				))
			} else if cleanDir == "inbound" && dep.TargetFile() == cleanID {
				results = append(results, NewDependencyNavigationItem(
					dep.TargetFile(),
					dep.SourceFile(),
					"inbound",
					"cross_file_dependent",
					"crossrepo_intelligence",
				))
			}
		}

		// Module relationships
		for _, mod := range n.crossModel.ModuleRelationships() {
			if mod == nil {
				continue
			}
			if cleanDir == "outbound" && mod.SourceModule() == cleanID {
				results = append(results, NewDependencyNavigationItem(
					mod.SourceModule(),
					mod.TargetModule(),
					"outbound",
					string(mod.Kind()),
					"cross_module_analyzer",
				))
			} else if cleanDir == "inbound" && mod.TargetModule() == cleanID {
				results = append(results, NewDependencyNavigationItem(
					mod.TargetModule(),
					mod.SourceModule(),
					"inbound",
					string(mod.Kind()),
					"cross_module_analyzer",
				))
			}
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].ID() < results[j].ID()
	})

	return results, nil
}

// RelationshipLookup navigates all established relationships connecting to an entity.
func (n *ReferenceNavigator) RelationshipLookup(entityID string) ([]*RelationshipItem, error) {
	cleanID := strings.TrimSpace(entityID)
	if cleanID == "" {
		return nil, ErrEmptyTarget
	}

	var items []*RelationshipItem

	// 1. Symbol ownership / containment
	if n.symbolDB != nil {
		sym := n.symbolDB.SymbolByID(cleanID)
		if sym != nil {
			items = append(items, NewRelationshipItem(
				sym.PackagePath(),
				sym.ID(),
				RelKindContains,
				"downward",
				"symbol_database",
			))
			items = append(items, NewRelationshipItem(
				sym.ID(),
				sym.PackagePath(),
				RelKindBelongsTo,
				"upward",
				"symbol_database",
			))
		}
	}

	// 2. References
	if n.xrefModel != nil && n.xrefModel.References() != nil {
		for _, ref := range n.xrefModel.References().AllReferences() {
			if ref == nil {
				continue
			}
			if ref.SourceSymbolID() == cleanID {
				items = append(items, NewRelationshipItem(
					ref.SourceSymbolID(),
					ref.TargetSymbolID(),
					RelKindReferences,
					"outbound",
					"xref_references",
				))
			} else if ref.TargetSymbolID() == cleanID {
				items = append(items, NewRelationshipItem(
					ref.TargetSymbolID(),
					ref.SourceSymbolID(),
					RelKindReferences,
					"inbound",
					"xref_references",
				))
			}
		}
	}

	// 3. Calls
	if n.xrefModel != nil && n.xrefModel.CallGraph() != nil {
		for _, edge := range n.xrefModel.CallGraph().AllEdges() {
			if edge == nil {
				continue
			}
			if edge.CallerID() == cleanID {
				items = append(items, NewRelationshipItem(
					edge.CallerID(),
					edge.CalleeID(),
					RelKindCalls,
					"outbound",
					"call_graph",
				))
			} else if edge.CalleeID() == cleanID {
				items = append(items, NewRelationshipItem(
					edge.CalleeID(),
					edge.CallerID(),
					RelKindCalls,
					"inbound",
					"call_graph",
				))
			}
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].ID() < items[j].ID()
	})

	return items, nil
}
