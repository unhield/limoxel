package navigation

import (
	"sync"

	"github.com/unhield/limoxel/internal/capabilities/intelligence/crossrepo"
	"github.com/unhield/limoxel/internal/capabilities/intelligence/semantic"
	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
	"github.com/unhield/limoxel/internal/capabilities/repository/xref"
)

// AnalysisParams holds all engineering knowledge inputs passed to the navigation engine.
type AnalysisParams struct {
	SymbolDB       *symbol.SymbolDatabase
	XRefModel      *xref.XRefModel
	SemanticModel  *semantic.SemanticModel
	CrossRepoModel *crossrepo.CrossRepoModel
}

// Engine coordinates definition, reference, hierarchy, call hierarchy, and validation operations.
type Engine struct {
	mu        sync.RWMutex
	defNav    *DefinitionNavigator
	refNav    *ReferenceNavigator
	hierNav   *HierarchyNavigator
	callNav   *CallHierarchyNavigator
	validator *NavigationValidator
	model     *NavigationModel
}

// New constructs an initialized Navigation Engine.
func New() *Engine {
	return &Engine{
		defNav:    NewDefinitionNavigator(nil, nil, nil, nil),
		refNav:    NewReferenceNavigator(nil, nil, nil, nil),
		hierNav:   NewHierarchyNavigator(nil, nil, nil),
		callNav:   NewCallHierarchyNavigator(nil, nil),
		validator: NewNavigationValidator(),
	}
}

// Analyze compiles established engineering knowledge into an immutable NavigationModel.
func (e *Engine) Analyze(params AnalysisParams) (*NavigationModel, error) {
	if e == nil {
		return nil, ErrNilEngine
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.defNav = NewDefinitionNavigator(params.SymbolDB, params.XRefModel, params.SemanticModel, params.CrossRepoModel)
	e.refNav = NewReferenceNavigator(params.SymbolDB, params.XRefModel, params.SemanticModel, params.CrossRepoModel)
	e.hierNav = NewHierarchyNavigator(params.SymbolDB, params.SemanticModel, params.CrossRepoModel)
	e.callNav = NewCallHierarchyNavigator(params.SymbolDB, params.XRefModel)

	defs := make(map[string]*DefinitionResult)
	refs := make(map[string]*ReferenceResult)
	usages := make(map[string][]*UsageItem)
	symHier := make(map[string]*SymbolHierarchyNode)
	ifaceNodes := make(map[string]*InterfaceHierarchyNode)
	typeNodes := make(map[string]*TypeHierarchyNode)
	pkgHier := make(map[string]*PackageHierarchyNode)
	callNodes := make(map[string]*CallHierarchyNode)

	// Ingest Symbols
	if params.SymbolDB != nil {
		for _, sym := range params.SymbolDB.AllSymbols() {
			if sym == nil {
				continue
			}

			// Definitions
			defRes, _ := e.defNav.GoToDefinition(sym.ID())
			if defRes != nil {
				defs[sym.ID()] = defRes
			}

			// References
			refRes, _ := e.refNav.FindReferences(sym.ID())
			if refRes != nil {
				refs[sym.ID()] = refRes
			}

			// Usages
			uList, _ := e.refNav.FindUsages(sym.ID())
			if len(uList) > 0 {
				usages[sym.ID()] = uList
			}

			// Symbol Hierarchy
			children, _ := e.hierNav.GetChildSymbols(sym.ID())
			parents, _ := e.hierNav.GetParentSymbols(sym.ID())
			parentID := ""
			if len(parents) > 0 {
				parentID = parents[0].SymbolID()
			}
			var childNodes []*SymbolHierarchyNode
			for _, c := range children {
				childNodes = append(childNodes, NewSymbolHierarchyNode(
					c.SymbolID(),
					c.Name(),
					c.Kind(),
					c.FilePath(),
					c.PackagePath(),
					sym.ID(),
					nil,
				))
			}
			symHier[sym.ID()] = NewSymbolHierarchyNode(
				sym.ID(),
				sym.Name(),
				string(sym.Kind()),
				sym.FilePath(),
				sym.PackagePath(),
				parentID,
				childNodes,
			)

			// Interface Hierarchy
			if sym.Kind() == symbol.SymbolKindInterface {
				ifaceNode, _ := e.hierNav.GetInterfaceHierarchy(sym.ID())
				if ifaceNode != nil {
					ifaceNodes[sym.ID()] = ifaceNode
				}
			}

			// Type Hierarchy
			if sym.Kind() == symbol.SymbolKindType || sym.Kind() == symbol.SymbolKindStruct {
				typeNode, _ := e.hierNav.GetTypeHierarchy(sym.ID())
				if typeNode != nil {
					typeNodes[sym.ID()] = typeNode
				}
			}

			// Call Hierarchy
			if sym.Kind() == symbol.SymbolKindFunction || sym.Kind() == symbol.SymbolKindMethod {
				incoming, _ := e.callNav.GetIncomingCalls(sym.ID())
				outgoing, _ := e.callNav.GetOutgoingCalls(sym.ID())
				var inIDs, outIDs []string
				for _, in := range incoming {
					inIDs = append(inIDs, in.SymbolID())
				}
				for _, out := range outgoing {
					outIDs = append(outIDs, out.SymbolID())
				}
				depth, _ := e.callNav.CalculateCallDepth(sym.ID(), 10)
				callNodes[sym.ID()] = NewCallHierarchyNode(
					sym.ID(),
					sym.Name(),
					sym.PackagePath(),
					sym.FilePath(),
					inIDs,
					outIDs,
					depth,
				)
			}

			// Package Hierarchy
			if _, exists := pkgHier[sym.PackagePath()]; !exists && sym.PackagePath() != "" {
				pNode, _ := e.hierNav.GetPackageHierarchy(sym.PackagePath())
				if pNode != nil {
					pkgHier[sym.PackagePath()] = pNode
				}
			}
		}
	}

	tempModel := NewNavigationModel(defs, refs, usages, symHier, ifaceNodes, typeNodes, pkgHier, callNodes, nil)
	valReport := e.validator.Validate(tempModel, params.SymbolDB, params.XRefModel)

	e.model = NewNavigationModel(defs, refs, usages, symHier, ifaceNodes, typeNodes, pkgHier, callNodes, valReport)
	return e.model, nil
}

// Model returns the active immutable NavigationModel.
func (e *Engine) Model() *NavigationModel {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.model
}

// Delegated Navigation Methods (Thread-Safe)
func (e *Engine) GoToDefinition(symbolID string) (*DefinitionResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.defNav.GoToDefinition(symbolID)
}

func (e *Engine) GoToDeclaration(symbolID string) (*DefinitionResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.defNav.GoToDeclaration(symbolID)
}

func (e *Engine) GoToImplementation(interfaceID string) ([]*NavigationTarget, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.defNav.GoToImplementation(interfaceID)
}

func (e *Engine) GoToPackage(entityID string) (*NavigationTarget, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.defNav.GoToPackage(entityID)
}

func (e *Engine) GoToModule(entityID string) (*NavigationTarget, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.defNav.GoToModule(entityID)
}

func (e *Engine) FindReferences(symbolID string) (*ReferenceResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.refNav.FindReferences(symbolID)
}

func (e *Engine) FindUsages(symbolID string) ([]*UsageItem, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.refNav.FindUsages(symbolID)
}

func (e *Engine) ReverseLookup(entityID string) ([]*ReverseRelationship, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.refNav.ReverseLookup(entityID)
}

func (e *Engine) DependencyLookup(entityID, direction string) ([]*DependencyNavigationItem, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.refNav.DependencyLookup(entityID, direction)
}

func (e *Engine) RelationshipLookup(entityID string) ([]*RelationshipItem, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.refNav.RelationshipLookup(entityID)
}

func (e *Engine) GetParentSymbols(symbolID string) ([]*NavigationTarget, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.hierNav.GetParentSymbols(symbolID)
}

func (e *Engine) GetChildSymbols(symbolID string) ([]*NavigationTarget, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.hierNav.GetChildSymbols(symbolID)
}

func (e *Engine) GetInterfaceHierarchy(interfaceID string) (*InterfaceHierarchyNode, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.hierNav.GetInterfaceHierarchy(interfaceID)
}

func (e *Engine) GetTypeHierarchy(typeID string) (*TypeHierarchyNode, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.hierNav.GetTypeHierarchy(typeID)
}

func (e *Engine) GetPackageHierarchy(packagePath string) (*PackageHierarchyNode, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.hierNav.GetPackageHierarchy(packagePath)
}

func (e *Engine) GetIncomingCalls(symbolID string) ([]*CallHierarchyNode, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.callNav.GetIncomingCalls(symbolID)
}

func (e *Engine) GetOutgoingCalls(symbolID string) ([]*CallHierarchyNode, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.callNav.GetOutgoingCalls(symbolID)
}

func (e *Engine) GetRecursivePaths(symbolID string) ([]*RecursivePath, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.callNav.GetRecursivePaths(symbolID)
}

func (e *Engine) GetDependencyChains(sourceID, targetID string, maxDepth int) ([]*DependencyChain, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.callNav.GetDependencyChains(sourceID, targetID, maxDepth)
}

func (e *Engine) CalculateCallDepth(symbolID string, maxDepth int) (int, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.callNav.CalculateCallDepth(symbolID, maxDepth)
}

func (e *Engine) DefinitionNavigator() *DefinitionNavigator       { return e.defNav }
func (e *Engine) ReferenceNavigator() *ReferenceNavigator         { return e.refNav }
func (e *Engine) HierarchyNavigator() *HierarchyNavigator         { return e.hierNav }
func (e *Engine) CallHierarchyNavigator() *CallHierarchyNavigator { return e.callNav }
func (e *Engine) Validator() *NavigationValidator                 { return e.validator }
