package navigation

// NavigationState represents the deterministic resolution status of a navigation destination.
type NavigationState string

const (
	// NavStateValid indicates the target exists and was authoritatively resolved.
	NavStateValid NavigationState = "valid"

	// NavStateMissing indicates the target does not exist or has been deleted.
	NavStateMissing NavigationState = "missing"

	// NavStateBroken indicates a reference points to an invalid or unreachable entity.
	NavStateBroken NavigationState = "broken"

	// NavStateAmbiguous indicates multiple valid targets exist without a unique tiebreaker.
	NavStateAmbiguous NavigationState = "ambiguous"

	// NavStateUnresolved indicates resolution could not be completed with available knowledge.
	NavStateUnresolved NavigationState = "unresolved"

	// NavStateUnavailable indicates the required knowledge domain is absent.
	NavStateUnavailable NavigationState = "unavailable"
)

// NavigationKind represents the semantic kind of navigation performed.
type NavigationKind string

const (
	NavKindDefinition         NavigationKind = "definition"
	NavKindDeclaration        NavigationKind = "declaration"
	NavKindImplementation     NavigationKind = "implementation"
	NavKindPackage            NavigationKind = "package"
	NavKindModule             NavigationKind = "module"
	NavKindReference          NavigationKind = "reference"
	NavKindUsage              NavigationKind = "usage"
	NavKindCallIncoming       NavigationKind = "call_incoming"
	NavKindCallOutgoing       NavigationKind = "call_outgoing"
	NavKindHierarchyParent    NavigationKind = "hierarchy_parent"
	NavKindHierarchyChild     NavigationKind = "hierarchy_child"
	NavKindTypeHierarchy      NavigationKind = "type_hierarchy"
	NavKindInterfaceHierarchy NavigationKind = "interface_hierarchy"
	NavKindPackageHierarchy   NavigationKind = "package_hierarchy"
	NavKindDependency         NavigationKind = "dependency"
	NavKindRelationship       NavigationKind = "relationship"
)

// UsageKind represents the specific contextual usage of an engineering entity.
type UsageKind string

const (
	UsageKindCall           UsageKind = "call"
	UsageKindType           UsageKind = "type_usage"
	UsageKindField          UsageKind = "field_usage"
	UsageKindPackage        UsageKind = "package_usage"
	UsageKindDependency     UsageKind = "dependency_usage"
	UsageKindImplementation UsageKind = "implementation_usage"
	UsageKindGeneral        UsageKind = "general_reference"
)

// RelationshipKind represents the nature of an established engineering relationship.
type RelationshipKind string

const (
	RelKindContains   RelationshipKind = "contains"
	RelKindReferences RelationshipKind = "references"
	RelKindCalls      RelationshipKind = "calls"
	RelKindImplements RelationshipKind = "implements"
	RelKindDependsOn  RelationshipKind = "depends_on"
	RelKindOwns       RelationshipKind = "owns"
	RelKindBelongsTo  RelationshipKind = "belongs_to"
	RelKindImports    RelationshipKind = "imports"
)

// ValidationFindingType classifies navigation validation issues.
type ValidationFindingType string

const (
	NavValMissingTarget       ValidationFindingType = "missing_target"
	NavValBrokenReference     ValidationFindingType = "broken_reference"
	NavValDuplicatePath       ValidationFindingType = "duplicate_path"
	NavValAmbiguousTarget     ValidationFindingType = "ambiguous_target"
	NavValInvalidRelationship ValidationFindingType = "invalid_relationship"
)

// Severity represents the severity of a navigation validation finding.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)
