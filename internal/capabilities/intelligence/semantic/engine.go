package semantic

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/repository/dependency"
	"github.com/unhield/limoxel/internal/capabilities/repository/graph"
	"github.com/unhield/limoxel/internal/capabilities/repository/indexing"
	"github.com/unhield/limoxel/internal/capabilities/repository/language"
	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
	"github.com/unhield/limoxel/internal/capabilities/repository/xref"
)

// Engine defines the primary Semantic Intelligence capability coordinator.
type Engine struct {
	mu             sync.RWMutex
	model          *SemanticModel
	scopeResolver  *ScopeResolver
	typeResolver   *TypeResolver
	symbolResolver *SymbolResolver
	validator      *SemanticValidator
}

// NewEngine creates a new Semantic Intelligence Engine instance.
func NewEngine() *Engine {
	return &Engine{}
}

// Analyze constructs the complete SemanticModel by synthesizing extracted repository models.
func (e *Engine) Analyze(
	repoName, repoRoot string,
	symDB *symbol.SymbolDatabase,
	symRels []*symbol.SymbolRelationship,
	xrefModel *xref.XRefModel,
	kg *graph.KnowledgeGraph,
	depModel *dependency.DependencyModel,
	idxModel *indexing.IndexModel,
	structModel *language.StructureModel,
) (*SemanticModel, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	startTime := time.Now()
	cleanRoot := filepath.ToSlash(filepath.Clean(repoRoot))

	// Data containers
	scopes := make(map[string]*SemanticScope)
	syms := make(map[string]*SemanticSymbol)
	types := make(map[string]*SemanticType)
	ifaces := make(map[string]*SemanticInterface)
	funcs := make(map[string]*SemanticFunction)
	vars := make(map[string]*SemanticVariable)
	generics := make(map[string]*SemanticGeneric)
	var rels []*SemanticRelationship

	// 1. Create Repository Scope
	repoScopeID := "scope:repo:" + cleanRoot
	scopes[repoScopeID] = NewSemanticScope(
		repoScopeID,
		ScopeRepository,
		repoName,
		"",
		"",
		"",
		nil,
		nil,
		nil,
		1,
		1000000,
	)

	// Map to track package symbols & scopes
	pkgMap := make(map[string]*SemanticPackage)
	pkgScopeMap := make(map[string]*SemanticScope)

	// 2. Ingest Symbols from SymbolDatabase
	if symDB != nil {
		allRawSyms := symDB.AllSymbols()

		// First pass: Build package and file scopes
		for _, s := range allRawSyms {
			pkgPath := s.PackagePath()
			pkgScopeID := "scope:pkg:" + pkgPath
			if _, exists := pkgScopeMap[pkgPath]; !exists {
				pkgScope := NewSemanticScope(
					pkgScopeID,
					ScopePackage,
					s.PackageName(),
					pkgPath,
					"",
					repoScopeID,
					nil,
					nil,
					nil,
					1,
					1000000,
				)
				pkgScopeMap[pkgPath] = pkgScope
				scopes[pkgScopeID] = pkgScope
			}

			filePath := s.FilePath()
			fileScopeID := "scope:file:" + filePath
			if _, exists := scopes[fileScopeID]; !exists {
				scopes[fileScopeID] = NewSemanticScope(
					fileScopeID,
					ScopeFile,
					filepath.Base(filePath),
					pkgPath,
					filePath,
					pkgScopeID,
					nil,
					nil,
					nil,
					1,
					1000000,
				)
			}
		}

		// Second pass: Ingest symbols and construct specialized semantic entities
		for _, s := range allRawSyms {
			symID := s.ID()
			pkgPath := s.PackagePath()
			filePath := s.FilePath()
			line := 0
			if s.Position() != nil {
				line = s.Position().Line()
			}

			fileScopeID := "scope:file:" + filePath
			visibility := DetermineVisibility(s.Name(), s.Kind())
			ownership := pkgPath
			if s.ReceiverType() != "" {
				ownership = s.ReceiverType()
			}

			// Ingest doc text
			docText := ""
			if s.Doc() != nil {
				docText = s.Doc().Content()
			}

			// Ingest Generics if present
			var genEntity *SemanticGeneric
			if len(s.Generics()) > 0 {
				genID := "gen:" + symID
				genConstraints := make(map[string]string)
				for _, g := range s.Generics() {
					parts := strings.Split(g, " ")
					if len(parts) >= 2 {
						genConstraints[parts[0]] = parts[1]
					} else {
						genConstraints[g] = "any"
					}
				}
				genEntity = NewSemanticGeneric(genID, symID, s.Generics(), genConstraints, nil, StateResolved)
				generics[genID] = genEntity
			}

			// Categorize by symbol kind
			switch s.Kind() {
			case symbol.SymbolKindStruct, symbol.SymbolKindType:
				tKind := TypeCustom
				tID := "type:" + symID
				typeSym := NewSemanticType(
					tID,
					s.Name(),
					tKind,
					pkgPath,
					filePath,
					s.TypeDefinition(),
					s.IsAlias(),
					"",
					s.IsExported(),
					nil,
					nil,
					nil,
					nil,
					genEntity,
					StateResolved,
				)
				types[tID] = typeSym

			case symbol.SymbolKindAlias:
				tID := "type:" + symID
				typeSym := NewSemanticType(
					tID,
					s.Name(),
					TypeAlias,
					pkgPath,
					filePath,
					s.TypeDefinition(),
					true,
					s.TypeDefinition(),
					s.IsExported(),
					nil,
					nil,
					nil,
					nil,
					genEntity,
					StateResolved,
				)
				types[tID] = typeSym

			case symbol.SymbolKindInterface:
				ifaceID := "iface:" + symID
				ifaceSym := NewSemanticInterface(
					ifaceID,
					s.Name(),
					pkgPath,
					filePath,
					nil,
					nil,
					nil,
					s.IsExported(),
				)
				ifaces[ifaceID] = ifaceSym
				// Also represent interface as a SemanticType
				tID := "type:" + symID
				types[tID] = NewSemanticType(
					tID,
					s.Name(),
					TypeInterface,
					pkgPath,
					filePath,
					"interface",
					false,
					"",
					s.IsExported(),
					nil,
					nil,
					nil,
					nil,
					genEntity,
					StateResolved,
				)

			case symbol.SymbolKindFunction, symbol.SymbolKindMethod:
				funcScopeID := fmt.Sprintf("scope:func:%s", symID)
				scopes[funcScopeID] = NewSemanticScope(
					funcScopeID,
					ScopeLocal,
					s.Name(),
					pkgPath,
					filePath,
					fileScopeID,
					nil,
					nil,
					nil,
					line,
					line+50,
				)

				funcSym := NewSemanticFunction(
					"func:"+symID,
					s.Name(),
					pkgPath,
					filePath,
					s.ReceiverType(),
					s.IsPointerReceiver(),
					nil,
					nil,
					s.IsExported(),
					visibility,
					s.Signature(),
					funcScopeID,
					nil,
					nil,
					genEntity,
				)
				funcs[funcSym.ID()] = funcSym

			case symbol.SymbolKindVariable, symbol.SymbolKindConstant:
				scopeKind := ScopePackage
				vScopeID := "scope:pkg:" + pkgPath
				if s.ReceiverType() != "" {
					scopeKind = ScopeLocal
					vScopeID = fileScopeID
				}
				varSym := NewSemanticVariable(
					"var:"+symID,
					s.Name(),
					pkgPath,
					filePath,
					scopeKind,
					vScopeID,
					s.TypeDefinition(),
					"",
					s.IsExported(),
					visibility,
					line,
				)
				vars[varSym.ID()] = varSym
			}

			// Create unified SemanticSymbol
			semSym := NewSemanticSymbol(
				symID,
				s.Name(),
				s.Kind(),
				pkgPath,
				filePath,
				line,
				s.IsExported(),
				visibility,
				ownership,
				fileScopeID,
				"type:"+symID,
				s.Signature(),
				docText,
				nil,
				nil,
				nil,
			)
			syms[symID] = semSym
		}
	}

	// 3. Attach Methods & Fields to Types and Implementors to Interfaces
	for _, fn := range funcs {
		if fn.ReceiverType() != "" {
			for _, t := range types {
				if t.Name() == fn.ReceiverType() && t.PackagePath() == fn.PackagePath() {
					// Attach method
					updatedMethods := append(t.Methods(), fn)
					types[t.ID()] = NewSemanticType(
						t.ID(),
						t.Name(),
						t.Kind(),
						t.PackagePath(),
						t.FilePath(),
						t.UnderlyingType(),
						t.IsAlias(),
						t.AliasTarget(),
						t.IsExported(),
						t.Fields(),
						updatedMethods,
						t.EmbeddedTypes(),
						t.ImplementedInterfaces(),
						t.Generics(),
						t.ResolutionState(),
					)
				}
			}
		}
	}

	// 4. Ingest Relationships from symbol.SymbolRelationship and xref.XRefModel
	for _, sr := range symRels {
		if sr == nil {
			continue
		}
		var relKind SemanticRelationKind
		switch sr.Kind() {
		case symbol.RelFunctionOwnership:
			relKind = RelSemanticOwnership
		case symbol.RelMethodReceiver:
			relKind = RelSemanticOwnership
		case symbol.RelInterfaceImplementation:
			relKind = RelSemanticImplementation
		case symbol.RelStructEmbedding:
			relKind = RelSemanticEmbeds
		case symbol.RelTypeAlias:
			relKind = RelSemanticAliasOf
		case symbol.RelGenericConstraint:
			relKind = RelSemanticConstrainedBy
		default:
			relKind = RelSemanticReferences
		}

		rels = append(rels, NewSemanticRelationship(
			"",
			relKind,
			sr.SourceID(),
			sr.TargetID(),
			sr.Evidence(),
			"symbol_engine",
			nil,
		))
	}

	// 5. Ingest XRef Call Graph and References
	if xrefModel != nil {
		if xrefModel.References() != nil {
			for _, ref := range xrefModel.References().AllReferences() {
				if ref == nil {
					continue
				}
				src := syms[ref.SourceSymbolID()]
				if src != nil {
					// Record reference
					srcRefs := append(src.References(), ref.TargetSymbolID())
					syms[src.ID()] = NewSemanticSymbol(
						src.ID(),
						src.Name(),
						src.Kind(),
						src.PackagePath(),
						src.FilePath(),
						src.Line(),
						src.IsExported(),
						src.Visibility(),
						src.Ownership(),
						src.ScopeID(),
						src.TypeID(),
						src.Signature(),
						src.Doc(),
						srcRefs,
						src.Calls(),
						src.CalledBy(),
					)
				}

				rels = append(rels, NewSemanticRelationship(
					ref.ID(),
					RelSemanticReferences,
					ref.SourceSymbolID(),
					ref.TargetSymbolID(),
					ref.Evidence(),
					"xref_engine",
					nil,
				))
			}
		}

		if xrefModel.CallGraph() != nil {
			for _, edge := range xrefModel.CallGraph().AllEdges() {
				if edge == nil {
					continue
				}
				callerID := "func:" + edge.CallerID()
				calleeID := "func:" + edge.CalleeID()

				if callerFunc, exists := funcs[callerID]; exists {
					updatedCalls := append(callerFunc.Calls(), calleeID)
					funcs[callerID] = NewSemanticFunction(
						callerFunc.ID(),
						callerFunc.Name(),
						callerFunc.PackagePath(),
						callerFunc.FilePath(),
						callerFunc.ReceiverType(),
						callerFunc.IsPointerReceiver(),
						callerFunc.Parameters(),
						callerFunc.ReturnTypes(),
						callerFunc.IsExported(),
						callerFunc.Visibility(),
						callerFunc.Signature(),
						callerFunc.ScopeID(),
						updatedCalls,
						callerFunc.CalledBy(),
						callerFunc.Generics(),
					)
				}

				if calleeFunc, exists := funcs[calleeID]; exists {
					updatedCalledBy := append(calleeFunc.CalledBy(), callerID)
					funcs[calleeID] = NewSemanticFunction(
						calleeFunc.ID(),
						calleeFunc.Name(),
						calleeFunc.PackagePath(),
						calleeFunc.FilePath(),
						calleeFunc.ReceiverType(),
						calleeFunc.IsPointerReceiver(),
						calleeFunc.Parameters(),
						calleeFunc.ReturnTypes(),
						calleeFunc.IsExported(),
						calleeFunc.Visibility(),
						calleeFunc.Signature(),
						calleeFunc.ScopeID(),
						calleeFunc.Calls(),
						updatedCalledBy,
						calleeFunc.Generics(),
					)
				}

				rels = append(rels, NewSemanticRelationship(
					edge.ID(),
					RelSemanticCalls,
					edge.CallerID(),
					edge.CalleeID(),
					"xref call edge",
					"xref_engine",
					nil,
				))
			}
		}
	}

	// 6. Build ScopeResolver, TypeResolver, SymbolResolver
	scopeResolver := NewScopeResolver(scopes, syms)
	typeResolver := NewTypeResolver(types, ifaces)
	symbolResolver := NewSymbolResolver(syms, scopeResolver)

	// 7. Check Interface Implementations and populate Implementors
	for ifaceID, iface := range ifaces {
		var implementors []string
		for typeID := range types {
			if typeResolver.CheckInterfaceSatisfaction(typeID, ifaceID) {
				implementors = append(implementors, typeID)
			}
		}
		sort.Strings(implementors)
		ifaces[ifaceID] = NewSemanticInterface(
			iface.ID(),
			iface.Name(),
			iface.PackagePath(),
			iface.FilePath(),
			iface.Methods(),
			iface.EmbeddedInterfaces(),
			implementors,
			iface.IsExported(),
		)
	}

	// 8. Construct SemanticPackage objects
	packagePaths := make(map[string]bool)
	for _, s := range syms {
		packagePaths[s.PackagePath()] = true
	}

	var packages []*SemanticPackage
	for pkgPath := range packagePaths {
		var pSyms []*SemanticSymbol
		for _, s := range syms {
			if s.PackagePath() == pkgPath {
				pSyms = append(pSyms, s)
			}
		}

		var pTypes []*SemanticType
		for _, t := range types {
			if t.PackagePath() == pkgPath {
				pTypes = append(pTypes, t)
			}
		}

		var pIfaces []*SemanticInterface
		for _, i := range ifaces {
			if i.PackagePath() == pkgPath {
				pIfaces = append(pIfaces, i)
			}
		}

		var pFuncs []*SemanticFunction
		for _, f := range funcs {
			if f.PackagePath() == pkgPath {
				pFuncs = append(pFuncs, f)
			}
		}

		var pVars []*SemanticVariable
		for _, v := range vars {
			if v.PackagePath() == pkgPath {
				pVars = append(pVars, v)
			}
		}

		pkgName := filepath.Base(pkgPath)
		if len(pSyms) > 0 {
			// Find name from first symbol if available
			pkgName = filepath.Base(pkgPath)
		}

		sp := NewSemanticPackage(
			pkgName,
			pkgPath,
			pSyms,
			pTypes,
			pIfaces,
			pFuncs,
			pVars,
			nil,
		)
		packages = append(packages, sp)
		pkgMap[pkgPath] = sp
	}

	// 9. Construct SemanticRepository
	repo := NewSemanticRepository(
		repoName,
		repoRoot,
		packages,
		len(syms),
		len(types),
		len(ifaces),
		len(funcs),
		len(vars),
		startTime,
	)

	// 10. Run Semantic Validation
	validator := NewSemanticValidator(typeResolver, symbolResolver)
	validationReport := validator.Validate(syms, types, ifaces, funcs, vars, scopes)

	// 11. Finalize and store model
	semanticModel := NewSemanticModel(
		repo,
		syms,
		types,
		ifaces,
		funcs,
		vars,
		generics,
		rels,
		scopes,
		validationReport,
		startTime,
	)

	e.model = semanticModel
	e.scopeResolver = scopeResolver
	e.typeResolver = typeResolver
	e.symbolResolver = symbolResolver
	e.validator = validator

	return semanticModel, nil
}

// Model returns the active immutable SemanticModel.
func (e *Engine) Model() *SemanticModel {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.model
}

// ScopeResolver returns the initialized ScopeResolver.
func (e *Engine) ScopeResolver() *ScopeResolver {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.scopeResolver
}

// TypeResolver returns the initialized TypeResolver.
func (e *Engine) TypeResolver() *TypeResolver {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.typeResolver
}

// SymbolResolver returns the initialized SymbolResolver.
func (e *Engine) SymbolResolver() *SymbolResolver {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.symbolResolver
}

// Validator returns the initialized SemanticValidator.
func (e *Engine) Validator() *SemanticValidator {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.validator
}
