package semantic

import (
	"path/filepath"
	"strings"
)

// PrimitiveTypeRegistry contains the set of standard language primitive types.
var PrimitiveTypeRegistry = map[string]bool{
	"bool":       true,
	"byte":       true,
	"complex64":  true,
	"complex128": true,
	"error":      true,
	"float32":    true,
	"float64":    true,
	"int":        true,
	"int8":       true,
	"int16":      true,
	"int32":      true,
	"int64":      true,
	"rune":       true,
	"string":     true,
	"uint":       true,
	"uint8":      true,
	"uint16":     true,
	"uint32":     true,
	"uint64":     true,
	"uintptr":    true,
	"any":        true,
	"comparable": true,
}

// TypeResolutionResult represents the outcome of resolving a type expression.
type TypeResolutionResult struct {
	Expression      string
	ResolvedType    *SemanticType
	TypeKind        TypeKind
	ResolutionState ResolutionState
	IsPrimitive     bool
	IsAlias         bool
	AliasTargetID   string
	IsGeneric       bool
	ErrorMessage    string
}

// TypeResolver resolves type expressions to concrete semantic types.
type TypeResolver struct {
	types      map[string]*SemanticType
	interfaces map[string]*SemanticInterface
}

// NewTypeResolver creates an initialized TypeResolver.
func NewTypeResolver(types map[string]*SemanticType, interfaces map[string]*SemanticInterface) *TypeResolver {
	tMap := make(map[string]*SemanticType, len(types))
	for k, v := range types {
		tMap[k] = v
	}

	iMap := make(map[string]*SemanticInterface, len(interfaces))
	for k, v := range interfaces {
		iMap[k] = v
	}

	return &TypeResolver{
		types:      tMap,
		interfaces: iMap,
	}
}

// ResolveType resolves a raw type expression within a specific package context.
func (r *TypeResolver) ResolveType(typeExpr, pkgPath string) *TypeResolutionResult {
	expr := strings.TrimSpace(typeExpr)
	if expr == "" {
		return &TypeResolutionResult{
			Expression:      typeExpr,
			TypeKind:        TypeUnknown,
			ResolutionState: StateUnresolved,
		}
	}

	// 1. Primitive type check
	if PrimitiveTypeRegistry[expr] {
		return &TypeResolutionResult{
			Expression:      expr,
			TypeKind:        TypePrimitive,
			ResolutionState: StateResolved,
			IsPrimitive:     true,
		}
	}

	// 2. Pointer types (*T)
	if strings.HasPrefix(expr, "*") {
		subResult := r.ResolveType(strings.TrimPrefix(expr, "*"), pkgPath)
		return &TypeResolutionResult{
			Expression:      expr,
			ResolvedType:    subResult.ResolvedType,
			TypeKind:        TypePointer,
			ResolutionState: subResult.ResolutionState,
		}
	}

	// 3. Slice and Array types ([]T, [N]T)
	if strings.HasPrefix(expr, "[]") {
		subResult := r.ResolveType(strings.TrimPrefix(expr, "[]"), pkgPath)
		return &TypeResolutionResult{
			Expression:      expr,
			ResolvedType:    subResult.ResolvedType,
			TypeKind:        TypeSlice,
			ResolutionState: subResult.ResolutionState,
		}
	}

	// 4. Map types (map[K]V)
	if strings.HasPrefix(expr, "map[") {
		return &TypeResolutionResult{
			Expression:      expr,
			TypeKind:        TypeMap,
			ResolutionState: StateResolved,
		}
	}

	// 5. Channel types (chan T)
	if strings.HasPrefix(expr, "chan ") || strings.HasPrefix(expr, "<-chan ") || strings.HasPrefix(expr, "chan<- ") {
		return &TypeResolutionResult{
			Expression:      expr,
			TypeKind:        TypeChan,
			ResolutionState: StateResolved,
		}
	}

	// 6. Function types (func(...) ...)
	if strings.HasPrefix(expr, "func(") {
		return &TypeResolutionResult{
			Expression:      expr,
			TypeKind:        TypeFunc,
			ResolutionState: StateResolved,
		}
	}

	// 7. Qualified or Unqualified Custom Type Lookup
	cleanPkg := filepath.ToSlash(filepath.Clean(pkgPath))
	var candidateID string

	if strings.Contains(expr, ".") {
		// e.g. "symbol.SourcePosition" or "pkg/path.Type"
		parts := strings.Split(expr, ".")
		pkgName := parts[0]
		typeName := parts[1]

		for id, t := range r.types {
			if t.Name() == typeName && (filepath.Base(t.PackagePath()) == pkgName || strings.HasSuffix(t.PackagePath(), pkgName)) {
				candidateID = id
				break
			}
		}
	} else {
		// Unqualified lookup in current package first
		for id, t := range r.types {
			if t.Name() == expr && t.PackagePath() == cleanPkg {
				candidateID = id
				break
			}
		}

		// If not in current package, check all packages
		if candidateID == "" {
			var matchingIDs []string
			for id, t := range r.types {
				if t.Name() == expr {
					matchingIDs = append(matchingIDs, id)
				}
			}

			if len(matchingIDs) == 1 {
				candidateID = matchingIDs[0]
			} else if len(matchingIDs) > 1 {
				return &TypeResolutionResult{
					Expression:      expr,
					TypeKind:        TypeCustom,
					ResolutionState: StateAmbiguous,
					ErrorMessage:    "multiple matching types across packages",
				}
			}
		}
	}

	if candidateID != "" {
		targetType := r.types[candidateID]
		if targetType != nil {
			if targetType.IsAlias() {
				finalID, isCyclic := r.ResolveAliasChain(targetType.ID())
				if isCyclic {
					return &TypeResolutionResult{
						Expression:      expr,
						ResolvedType:    targetType,
						TypeKind:        TypeAlias,
						ResolutionState: StateInvalid,
						ErrorMessage:    "cyclic type alias detected",
					}
				}
				finalType := r.types[finalID]
				return &TypeResolutionResult{
					Expression:      expr,
					ResolvedType:    finalType,
					TypeKind:        TypeAlias,
					ResolutionState: StateResolved,
					IsAlias:         true,
					AliasTargetID:   finalID,
				}
			}

			return &TypeResolutionResult{
				Expression:      expr,
				ResolvedType:    targetType,
				TypeKind:        targetType.Kind(),
				ResolutionState: StateResolved,
				IsGeneric:       targetType.Generics() != nil,
			}
		}
	}

	return &TypeResolutionResult{
		Expression:      expr,
		TypeKind:        TypeUnknown,
		ResolutionState: StateUnresolved,
		ErrorMessage:    "type declaration not found in repository",
	}
}

// ResolveAliasChain walks through alias chains (A -> B -> C) to find the terminal type, detecting cycles.
func (r *TypeResolver) ResolveAliasChain(startTypeID string) (finalTypeID string, isCyclic bool) {
	curr := startTypeID
	visited := make(map[string]bool)

	for {
		if visited[curr] {
			return curr, true
		}
		visited[curr] = true

		t := r.types[curr]
		if t == nil || !t.IsAlias() || t.AliasTarget() == "" {
			return curr, false
		}

		targetID := t.AliasTarget()
		if r.types[targetID] == nil {
			// Target might be an unqualified type name in the same package
			targetID = "type:" + t.PackagePath() + "." + t.AliasTarget()
			if r.types[targetID] == nil {
				return curr, false
			}
		}
		curr = targetID
	}
}

// CheckInterfaceSatisfaction tests whether a concrete type satisfies an interface contract by method signature comparison.
func (r *TypeResolver) CheckInterfaceSatisfaction(typeID, ifaceID string) bool {
	t := r.types[typeID]
	iface := r.interfaces[ifaceID]
	if t == nil || iface == nil {
		return false
	}

	ifaceMethods := iface.Methods()
	if len(ifaceMethods) == 0 {
		return true // Empty interface is satisfied by everything
	}

	typeMethods := t.Methods()
	typeMethodMap := make(map[string]*SemanticFunction, len(typeMethods))
	for _, m := range typeMethods {
		typeMethodMap[m.Name()] = m
	}

	for _, reqMethod := range ifaceMethods {
		tm, exists := typeMethodMap[reqMethod.Name()]
		if !exists {
			return false
		}
		if reqMethod.Signature() != "" && tm.Signature() != "" && reqMethod.Signature() != tm.Signature() {
			return false
		}
	}

	return true
}
