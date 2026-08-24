package xref

// ReferenceKind represents the syntactic classification of a cross-reference.
type ReferenceKind string

const (
	// RefFunction represents a reference to a standalone function.
	RefFunction ReferenceKind = "function_reference"

	// RefMethod represents a reference to a method with receiver binding.
	RefMethod ReferenceKind = "method_reference"

	// RefInterface represents a reference to an interface type.
	RefInterface ReferenceKind = "interface_reference"

	// RefStruct represents a reference to a struct type or instantiation.
	RefStruct ReferenceKind = "struct_reference"

	// RefConstant represents a reference to a declared constant.
	RefConstant ReferenceKind = "constant_reference"

	// RefVariable represents a reference to a declared variable.
	RefVariable ReferenceKind = "variable_reference"

	// RefType represents a reference to a defined type or alias.
	RefType ReferenceKind = "type_reference"

	// RefUnknown represents an unclassified reference.
	RefUnknown ReferenceKind = "unknown_reference"
)

// CallKind represents the nature of a call-graph edge.
type CallKind string

const (
	// CallDirect represents a direct static invocation of a package function.
	CallDirect CallKind = "direct_call"

	// CallMethod represents a method invocation on a concrete receiver.
	CallMethod CallKind = "method_invocation"

	// CallInterface represents a method invocation dispatched through an interface.
	CallInterface CallKind = "interface_invocation"

	// CallRecursiveDirect represents a function calling itself directly.
	CallRecursiveDirect CallKind = "recursive_direct"

	// CallRecursiveMutual represents a function participating in a mutual recursion cycle.
	CallRecursiveMutual CallKind = "recursive_mutual"
)

// ResolutionState represents the deterministic confidence and resolution status of a relationship.
type ResolutionState string

const (
	// StateResolved indicates the reference was authoritatively resolved to an internal symbol.
	StateResolved ResolutionState = "resolved"

	// StateAmbiguous indicates multiple valid candidate declarations exist without a deterministic tiebreaker.
	StateAmbiguous ResolutionState = "ambiguous"

	// StateUnresolvedExternal indicates the reference legitimately points to an external or standard-library package.
	StateUnresolvedExternal ResolutionState = "unresolved_external"

	// StateBroken indicates the reference refers to an internal target symbol that does not exist.
	StateBroken ResolutionState = "broken"

	// StateUnknown indicates resolution could not be determined due to incomplete structural information.
	StateUnknown ResolutionState = "unknown"
)

// ReachabilityState represents whether a function is reachable from defined repository entry points.
type ReachabilityState string

const (
	// ReachableConfirmed indicates the function has a verified path from an entry point.
	ReachableConfirmed ReachabilityState = "reachable_confirmed"

	// UnreachableConfirmed indicates the function is proven to have no incoming call paths from entry points.
	UnreachableConfirmed ReachabilityState = "unreachable_confirmed"

	// ReachabilityUnknown indicates reachability cannot be determined with complete certainty.
	ReachabilityUnknown ReachabilityState = "reachability_unknown"
)

// ImpactSeverity categorizes the proximity of a change's impact.
type ImpactSeverity string

const (
	// ImpactDirect indicates a directly referencing symbol or calling entity.
	ImpactDirect ImpactSeverity = "direct"

	// ImpactTransitive indicates a transitively affected entity through caller/dependency chains.
	ImpactTransitive ImpactSeverity = "transitive"

	// ImpactStructural indicates an affected interface or struct embedding relationship.
	ImpactStructural ImpactSeverity = "structural"

	// ImpactDependency indicates an affected downstream package or file.
	ImpactDependency ImpactSeverity = "dependency"
)

// BreakingCategory categorizes the certainty of an identified breaking change.
type BreakingCategory string

const (
	// BreakingConfirmed represents an authoritatively proven breaking change.
	BreakingConfirmed BreakingCategory = "breaking_confirmed"

	// BreakingPotential represents a probable breaking change subject to runtime dynamic dispatch.
	BreakingPotential BreakingCategory = "breaking_potential"

	// BreakingUnknown represents an unresolved change whose breaking consequence cannot be determined.
	BreakingUnknown BreakingCategory = "breaking_unknown"

	// NonBreaking represents an addition or change proven not to invalidate existing references.
	NonBreaking BreakingCategory = "non_breaking"
)

// ValidationSeverity classifies the type of a relationship integrity issue.
type ValidationSeverity string

const (
	// ValidationBrokenRef represents a broken reference pointing to a missing internal symbol.
	ValidationBrokenRef ValidationSeverity = "broken_reference"

	// ValidationMissingSymbol represents an expected symbol declaration not found in scope.
	ValidationMissingSymbol ValidationSeverity = "missing_symbol"

	// ValidationDuplicateSymbol represents two colliding symbol declarations in the same package scope.
	ValidationDuplicateSymbol ValidationSeverity = "duplicate_symbol"

	// ValidationInvalidImport represents an import targeting a non-existent internal path.
	ValidationInvalidImport ValidationSeverity = "invalid_import"

	// ValidationCircularRef represents an invalid circular dependency or call cycle.
	ValidationCircularRef ValidationSeverity = "circular_reference"
)
