package semantic

// ScopeKind defines the semantic scoping granularity for entities and resolution.
type ScopeKind string

const (
	ScopeRepository ScopeKind = "repository"
	ScopePackage    ScopeKind = "package"
	ScopeFile       ScopeKind = "file"
	ScopeGlobal     ScopeKind = "global"
	ScopeLocal      ScopeKind = "local"
	ScopeBlock      ScopeKind = "block"
	ScopeGeneric    ScopeKind = "generic"
)

// String returns the string representation of ScopeKind.
func (s ScopeKind) String() string {
	return string(s)
}

// TypeKind defines the classification of semantic types.
type TypeKind string

const (
	TypePrimitive TypeKind = "primitive"
	TypeCustom    TypeKind = "custom"
	TypeInterface TypeKind = "interface"
	TypeGeneric   TypeKind = "generic"
	TypeAlias     TypeKind = "alias"
	TypeEmbedded  TypeKind = "embedded"
	TypePointer   TypeKind = "pointer"
	TypeArray     TypeKind = "array"
	TypeSlice     TypeKind = "slice"
	TypeMap       TypeKind = "map"
	TypeChan      TypeKind = "channel"
	TypeFunc      TypeKind = "function"
	TypeUnknown   TypeKind = "unknown"
)

// String returns the string representation of TypeKind.
func (t TypeKind) String() string {
	return string(t)
}

// ResolutionState defines the deterministic resolution status of an entity or relationship.
type ResolutionState string

const (
	StateResolved    ResolutionState = "resolved"
	StateUnresolved  ResolutionState = "unresolved"
	StateAmbiguous   ResolutionState = "ambiguous"
	StateInvalid     ResolutionState = "invalid"
	StateUnavailable ResolutionState = "unavailable"
)

// String returns the string representation of ResolutionState.
func (r ResolutionState) String() string {
	return string(r)
}

// VisibilityKind represents the accessibility scope of a semantic symbol or type.
type VisibilityKind string

const (
	VisibilityPublic         VisibilityKind = "public"
	VisibilityPackagePrivate VisibilityKind = "package_private"
	VisibilityLocal          VisibilityKind = "local"
)

// String returns the string representation of VisibilityKind.
func (v VisibilityKind) String() string {
	return string(v)
}

// SemanticRelationKind classifies the engineering meaning of a semantic relationship.
type SemanticRelationKind string

const (
	RelSemanticOwnership      SemanticRelationKind = "semantic_ownership"
	RelSemanticContainment    SemanticRelationKind = "semantic_containment"
	RelSemanticImplementation SemanticRelationKind = "semantic_implementation"
	RelSemanticTypeUsage      SemanticRelationKind = "semantic_type_usage"
	RelSemanticCalls          SemanticRelationKind = "semantic_calls"
	RelSemanticReferences     SemanticRelationKind = "semantic_references"
	RelSemanticEmbeds         SemanticRelationKind = "semantic_embeds"
	RelSemanticAliasOf        SemanticRelationKind = "semantic_alias_of"
	RelSemanticConstrainedBy  SemanticRelationKind = "semantic_constrained_by"
	RelSemanticScope          SemanticRelationKind = "semantic_scope"
)

// String returns the string representation of SemanticRelationKind.
func (r SemanticRelationKind) String() string {
	return string(r)
}

// ValidationSeverity indicates the severity level of a semantic validation finding.
type ValidationSeverity string

const (
	SeverityError   ValidationSeverity = "error"
	SeverityWarning ValidationSeverity = "warning"
	SeverityInfo    ValidationSeverity = "info"
)

// String returns the string representation of ValidationSeverity.
func (v ValidationSeverity) String() string {
	return string(v)
}

// ValidationFindingKind categorizes a semantic validation issue.
type ValidationFindingKind string

const (
	FindingMissingSymbol            ValidationFindingKind = "missing_symbol"
	FindingInvalidType              ValidationFindingKind = "invalid_type"
	FindingInvalidReference         ValidationFindingKind = "invalid_reference"
	FindingDuplicateDefinition      ValidationFindingKind = "duplicate_definition"
	FindingScopeConflict            ValidationFindingKind = "scope_conflict"
	FindingInconsistentRelationship ValidationFindingKind = "inconsistent_relationship"
)

// String returns the string representation of ValidationFindingKind.
func (f ValidationFindingKind) String() string {
	return string(f)
}

// ValidationStatus represents the overall validation outcome of an entity or model.
type ValidationStatus string

const (
	StatusValid       ValidationStatus = "valid"
	StatusInvalid     ValidationStatus = "invalid"
	StatusUnresolved  ValidationStatus = "unresolved"
	StatusAmbiguous   ValidationStatus = "ambiguous"
	StatusUnavailable ValidationStatus = "unavailable"
)

// String returns the string representation of ValidationStatus.
func (s ValidationStatus) String() string {
	return string(s)
}
