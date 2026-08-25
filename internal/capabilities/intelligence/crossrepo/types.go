package crossrepo

// BoundaryKind represents the granularity of an engineering boundary.
type BoundaryKind string

const (
	BoundaryFile       BoundaryKind = "file"
	BoundaryPackage    BoundaryKind = "package"
	BoundaryModule     BoundaryKind = "module"
	BoundaryRepository BoundaryKind = "repository"
	BoundaryWorkspace  BoundaryKind = "workspace"
)

func (b BoundaryKind) String() string { return string(b) }

// FileRelationKind classifies relationships between source, test, doc, and config files.
type FileRelationKind string

const (
	FileRelImport            FileRelationKind = "import"
	FileRelReference         FileRelationKind = "reference"
	FileRelSymbolOwnership   FileRelationKind = "symbol_ownership"
	FileRelPackageMembership FileRelationKind = "package_membership"
	FileRelDependency        FileRelationKind = "dependency"
	FileRelConfigUsage       FileRelationKind = "config_usage"
	FileRelDocAssociation    FileRelationKind = "doc_association"
	FileRelTestSource        FileRelationKind = "test_source"
)

func (r FileRelationKind) String() string { return string(r) }

// PackageCommunicationKind defines the mechanism through which packages interact.
type PackageCommunicationKind string

const (
	PkgCommImport            PackageCommunicationKind = "import"
	PkgCommCall              PackageCommunicationKind = "call"
	PkgCommTypeUsage         PackageCommunicationKind = "type_usage"
	PkgCommInterfaceContract PackageCommunicationKind = "interface_contract"
	PkgCommSharedConfig      PackageCommunicationKind = "shared_config"
)

func (k PackageCommunicationKind) String() string { return string(k) }

// APIVisibility identifies the intended exposure boundary of an API or contract.
type APIVisibility string

const (
	APIVisibilityInternal APIVisibility = "internal"
	APIVisibilityPublic   APIVisibility = "public"
)

func (v APIVisibility) String() string { return string(v) }

// ModuleRelationKind classifies relationships between modules.
type ModuleRelationKind string

const (
	ModuleRelDependency ModuleRelationKind = "dependency"
	ModuleRelOwnership  ModuleRelationKind = "ownership"
	ModuleRelShared     ModuleRelationKind = "shared"
	ModuleRelHierarchy  ModuleRelationKind = "hierarchy"
)

func (k ModuleRelationKind) String() string { return string(k) }

// VersionCompatibilityState indicates the compatibility status between versioned entities.
type VersionCompatibilityState string

const (
	CompatCompatible   VersionCompatibilityState = "compatible"
	CompatIncompatible VersionCompatibilityState = "incompatible"
	CompatUnresolved   VersionCompatibilityState = "unresolved"
	CompatUnavailable  VersionCompatibilityState = "unavailable"
)

func (s VersionCompatibilityState) String() string { return string(s) }

// WorkspaceRelationKind classifies relationships connecting repositories in a workspace.
type WorkspaceRelationKind string

const (
	WorkspaceRelDependency    WorkspaceRelationKind = "dependency"
	WorkspaceRelSharedModule  WorkspaceRelationKind = "shared_module"
	WorkspaceRelSharedPackage WorkspaceRelationKind = "shared_package"
	WorkspaceRelSharedConfig  WorkspaceRelationKind = "shared_config"
	WorkspaceRelSharedArch    WorkspaceRelationKind = "shared_architecture"
)

func (k WorkspaceRelationKind) String() string { return string(k) }

// EvolutionKind classifies a type of historical structural change.
type EvolutionKind string

const (
	EvolutionAddition     EvolutionKind = "addition"
	EvolutionRemoval      EvolutionKind = "removal"
	EvolutionModification EvolutionKind = "modification"
	EvolutionMove         EvolutionKind = "move"
	EvolutionStructural   EvolutionKind = "structural"
)

func (k EvolutionKind) String() string { return string(k) }

// ValidationSeverity indicates the severity level of a cross-boundary validation finding.
type ValidationSeverity string

const (
	SeverityError   ValidationSeverity = "error"
	SeverityWarning ValidationSeverity = "warning"
	SeverityInfo    ValidationSeverity = "info"
)

func (s ValidationSeverity) String() string { return string(s) }

// ValidationStatus represents the overall validation outcome of an entity or model.
type ValidationStatus string

const (
	StatusValid       ValidationStatus = "valid"
	StatusInvalid     ValidationStatus = "invalid"
	StatusUnresolved  ValidationStatus = "unresolved"
	StatusAmbiguous   ValidationStatus = "ambiguous"
	StatusUnavailable ValidationStatus = "unavailable"
)

func (s ValidationStatus) String() string { return string(s) }

// ValidationFindingKind categorizes a cross-boundary validation issue.
type ValidationFindingKind string

const (
	FindingMissingCrossTarget            ValidationFindingKind = "missing_cross_target"
	FindingUnresolvedCrossReference      ValidationFindingKind = "unresolved_cross_reference"
	FindingInvalidCrossDependency        ValidationFindingKind = "invalid_cross_dependency"
	FindingInconsistentCrossRelationship ValidationFindingKind = "inconsistent_cross_relationship"
	FindingConflictingCrossConfig        ValidationFindingKind = "conflicting_cross_config"
)

func (k ValidationFindingKind) String() string { return string(k) }
