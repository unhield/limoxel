package semantic

import (
	"path/filepath"
	"sort"
	"strings"
)

// SemanticScope represents a lexical or semantic scope boundary.
type SemanticScope struct {
	id                string
	kind              ScopeKind
	name              string
	packagePath       string
	filePath          string
	parentID          string
	childIDs          []string
	declaredSymbolIDs []string
	declaredVariables []*SemanticVariable
	lineStart         int
	lineEnd           int
}

// NewSemanticScope creates an immutable SemanticScope.
func NewSemanticScope(
	id string,
	kind ScopeKind,
	name, packagePath, filePath, parentID string,
	childIDs, declaredSymIDs []string,
	declaredVars []*SemanticVariable,
	lineStart, lineEnd int,
) *SemanticScope {
	cList := make([]string, len(childIDs))
	copy(cList, childIDs)
	sort.Strings(cList)

	sList := make([]string, len(declaredSymIDs))
	copy(sList, declaredSymIDs)
	sort.Strings(sList)

	vList := make([]*SemanticVariable, len(declaredVars))
	copy(vList, declaredVars)
	sort.Slice(vList, func(i, j int) bool {
		return vList[i].ID() < vList[j].ID()
	})

	cleanID := strings.TrimSpace(id)
	cleanPkg := filepath.ToSlash(filepath.Clean(packagePath))
	cleanFile := filepath.ToSlash(filepath.Clean(filePath))

	return &SemanticScope{
		id:                cleanID,
		kind:              kind,
		name:              strings.TrimSpace(name),
		packagePath:       cleanPkg,
		filePath:          cleanFile,
		parentID:          strings.TrimSpace(parentID),
		childIDs:          cList,
		declaredSymbolIDs: sList,
		declaredVariables: vList,
		lineStart:         lineStart,
		lineEnd:           lineEnd,
	}
}

func (s *SemanticScope) ID() string          { return s.id }
func (s *SemanticScope) Kind() ScopeKind     { return s.kind }
func (s *SemanticScope) Name() string        { return s.name }
func (s *SemanticScope) PackagePath() string { return s.packagePath }
func (s *SemanticScope) FilePath() string    { return s.filePath }
func (s *SemanticScope) ParentID() string    { return s.parentID }
func (s *SemanticScope) LineStart() int      { return s.lineStart }
func (s *SemanticScope) LineEnd() int        { return s.lineEnd }

func (s *SemanticScope) ChildIDs() []string {
	if s == nil || s.childIDs == nil {
		return nil
	}
	res := make([]string, len(s.childIDs))
	copy(res, s.childIDs)
	return res
}

func (s *SemanticScope) DeclaredSymbolIDs() []string {
	if s == nil || s.declaredSymbolIDs == nil {
		return nil
	}
	res := make([]string, len(s.declaredSymbolIDs))
	copy(res, s.declaredSymbolIDs)
	return res
}

func (s *SemanticScope) DeclaredVariables() []*SemanticVariable {
	if s == nil || s.declaredVariables == nil {
		return nil
	}
	res := make([]*SemanticVariable, len(s.declaredVariables))
	copy(res, s.declaredVariables)
	return res
}

// ScopeLookupResult represents the outcome of resolving a name in a scope hierarchy.
type ScopeLookupResult struct {
	Name            string
	FoundScopeID    string
	Symbol          *SemanticSymbol
	Variable        *SemanticVariable
	ResolutionState ResolutionState
	AmbiguousCount  int
}

// ScopeResolver performs deterministic lexical and semantic scope lookups.
type ScopeResolver struct {
	scopes  map[string]*SemanticScope
	symbols map[string]*SemanticSymbol
}

// NewScopeResolver creates an initialized ScopeResolver.
func NewScopeResolver(scopes map[string]*SemanticScope, symbols map[string]*SemanticSymbol) *ScopeResolver {
	sMap := make(map[string]*SemanticScope, len(scopes))
	for k, v := range scopes {
		sMap[k] = v
	}

	symMap := make(map[string]*SemanticSymbol, len(symbols))
	for k, v := range symbols {
		symMap[k] = v
	}

	return &ScopeResolver{
		scopes:  sMap,
		symbols: symMap,
	}
}

// ResolveInScope resolves an identifier starting from a specific scope, traversing up the parent hierarchy.
func (r *ScopeResolver) ResolveInScope(startScopeID, identifier string) *ScopeLookupResult {
	cleanName := strings.TrimSpace(identifier)
	if cleanName == "" {
		return &ScopeLookupResult{
			Name:            identifier,
			ResolutionState: StateUnresolved,
		}
	}

	currID := strings.TrimSpace(startScopeID)
	visited := make(map[string]bool)

	for currID != "" {
		if visited[currID] {
			break // Cycle protection
		}
		visited[currID] = true

		scope := r.scopes[currID]
		if scope == nil {
			break
		}

		// 1. Check declared variables in current scope
		var matchingVars []*SemanticVariable
		for _, v := range scope.declaredVariables {
			if v != nil && v.Name() == cleanName {
				matchingVars = append(matchingVars, v)
			}
		}

		if len(matchingVars) == 1 {
			return &ScopeLookupResult{
				Name:            cleanName,
				FoundScopeID:    scope.ID(),
				Variable:        matchingVars[0],
				ResolutionState: StateResolved,
			}
		} else if len(matchingVars) > 1 {
			return &ScopeLookupResult{
				Name:            cleanName,
				FoundScopeID:    scope.ID(),
				ResolutionState: StateAmbiguous,
				AmbiguousCount:  len(matchingVars),
			}
		}

		// 2. Check declared symbols in current scope
		var matchingSyms []*SemanticSymbol
		for _, symID := range scope.declaredSymbolIDs {
			sym := r.symbols[symID]
			if sym != nil && sym.Name() == cleanName {
				matchingSyms = append(matchingSyms, sym)
			}
		}

		if len(matchingSyms) == 1 {
			return &ScopeLookupResult{
				Name:            cleanName,
				FoundScopeID:    scope.ID(),
				Symbol:          matchingSyms[0],
				ResolutionState: StateResolved,
			}
		} else if len(matchingSyms) > 1 {
			return &ScopeLookupResult{
				Name:            cleanName,
				FoundScopeID:    scope.ID(),
				ResolutionState: StateAmbiguous,
				AmbiguousCount:  len(matchingSyms),
			}
		}

		currID = scope.ParentID()
	}

	return &ScopeLookupResult{
		Name:            cleanName,
		ResolutionState: StateUnresolved,
	}
}

// FindEnclosingScope finds the innermost scope matching the given file path and line number.
func (r *ScopeResolver) FindEnclosingScope(filePath string, line int) *SemanticScope {
	cleanFile := filepath.ToSlash(filepath.Clean(filePath))
	var bestMatch *SemanticScope
	bestSpan := 1000000000

	for _, s := range r.scopes {
		if s == nil || s.FilePath() != cleanFile {
			continue
		}
		if s.LineStart() <= line && line <= s.LineEnd() {
			span := s.LineEnd() - s.LineStart()
			if span < bestSpan {
				bestSpan = span
				bestMatch = s
			}
		}
	}

	return bestMatch
}
