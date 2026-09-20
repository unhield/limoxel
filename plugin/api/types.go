package api

import (
	"context"
	"time"
)

// PaginationOptions specifies pagination parameters for listing and query endpoints.
type PaginationOptions struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

// Finding represents a diagnostic finding or observation produced by engineering analysis.
type Finding struct {
	ID          string            `json:"id"`
	RuleID      string            `json:"rule_id"`
	Severity    string            `json:"severity"`
	Category    string            `json:"category"`
	Message     string            `json:"message"`
	File        string            `json:"file,omitempty"`
	Line        int               `json:"line,omitempty"`
	Remediation string            `json:"remediation,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// RepositoryInfo provides an operational snapshot of a repository.
type RepositoryInfo struct {
	Name          string            `json:"name"`
	Owner         string            `json:"owner,omitempty"`
	RootPath      string            `json:"root_path"`
	DefaultBranch string            `json:"default_branch,omitempty"`
	CurrentBranch string            `json:"current_branch,omitempty"`
	IsGit         bool              `json:"is_git"`
	State         string            `json:"state"`
	Languages     []string          `json:"languages,omitempty"`
	Capabilities  []string          `json:"capabilities,omitempty"`
	Properties    map[string]string `json:"properties,omitempty"`
}

// RepositoryMetadata encapsulates detailed repository descriptive attributes.
type RepositoryMetadata struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Version     string            `json:"version,omitempty"`
	License     string            `json:"license,omitempty"`
	VCS         string            `json:"vcs,omitempty"`
	CommitHash  string            `json:"commit_hash,omitempty"`
	CommitDate  time.Time         `json:"commit_date,omitempty"`
	Authors     []string          `json:"authors,omitempty"`
	Properties  map[string]string `json:"properties,omitempty"`
}

// RepositoryStatistics aggregates quantitative metrics of the repository workspace.
type RepositoryStatistics struct {
	TotalFiles         int `json:"total_files"`
	TotalDirectories   int `json:"total_directories"`
	TotalPackages      int `json:"total_packages"`
	TotalSymbols       int `json:"total_symbols"`
	TotalRelationships int `json:"total_relationships"`
	TotalDependencies  int `json:"total_dependencies"`
}

// FileInfo provides structured metadata about an individual repository file.
type FileInfo struct {
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	Extension   string    `json:"extension,omitempty"`
	Language    string    `json:"language,omitempty"`
	IsDirectory bool      `json:"is_directory"`
	ModTime     time.Time `json:"mod_time"`
}

// SymbolKind represents the category of a code symbol.
type SymbolKind string

const (
	SymbolKindFunction  SymbolKind = "function"
	SymbolKindMethod    SymbolKind = "method"
	SymbolKindStruct    SymbolKind = "struct"
	SymbolKindInterface SymbolKind = "interface"
	SymbolKindType      SymbolKind = "type"
	SymbolKindVariable  SymbolKind = "variable"
	SymbolKindConstant  SymbolKind = "constant"
	SymbolKindPackage   SymbolKind = "package"
	SymbolKindFile      SymbolKind = "file"
)

// SymbolLocation specifies the source code coordinates of a symbol.
type SymbolLocation struct {
	FilePath    string `json:"file_path"`
	StartLine   int    `json:"start_line"`
	EndLine     int    `json:"end_line"`
	StartColumn int    `json:"start_column,omitempty"`
	EndColumn   int    `json:"end_column,omitempty"`
}

// SymbolInfo describes a code symbol within the repository.
type SymbolInfo struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Package    string         `json:"package"`
	Kind       SymbolKind     `json:"kind"`
	Location   SymbolLocation `json:"location"`
	Signature  string         `json:"signature,omitempty"`
	IsExported bool           `json:"is_exported"`
	DocComment string         `json:"doc_comment,omitempty"`
}

// SymbolReference captures an edge or call site referencing a symbol.
type SymbolReference struct {
	SourceSymbolID string `json:"source_symbol_id,omitempty"`
	SourceFile     string `json:"source_file"`
	SourceLine     int    `json:"source_line"`
	ReferenceKind  string `json:"reference_kind"`
	Evidence       string `json:"evidence,omitempty"`
}

// SymbolHierarchyNode represents a node in a symbol inheritance or call tree.
type SymbolHierarchyNode struct {
	Symbol   SymbolInfo            `json:"symbol"`
	Children []SymbolHierarchyNode `json:"children,omitempty"`
}

// SearchDomain designates the target entity domain for a search query.
type SearchDomain string

const (
	SearchDomainUnified       SearchDomain = "unified"
	SearchDomainSymbol        SearchDomain = "symbol"
	SearchDomainPackage       SearchDomain = "package"
	SearchDomainFile          SearchDomain = "file"
	SearchDomainDocumentation SearchDomain = "documentation"
	SearchDomainConfiguration SearchDomain = "configuration"
)

// SearchQuery encapsulates query parameters, target domain, and filters.
type SearchQuery struct {
	Query      string            `json:"query"`
	Domain     SearchDomain      `json:"domain"`
	Pagination PaginationOptions `json:"pagination"`
	Filters    map[string]string `json:"filters,omitempty"`
}

// SearchMatch represents an individual match returned from a search.
type SearchMatch struct {
	Domain   SearchDomain `json:"domain"`
	Name     string       `json:"name"`
	Package  string       `json:"package,omitempty"`
	Location string       `json:"location,omitempty"`
	Snippet  string       `json:"snippet,omitempty"`
	Score    float64      `json:"score"`
}

// SearchResult bundles the matches and metadata of a completed search.
type SearchResult struct {
	Query        string        `json:"query"`
	Domain       SearchDomain  `json:"domain"`
	TotalMatches int           `json:"total_matches"`
	Matches      []SearchMatch `json:"matches"`
	DurationMs   int64         `json:"duration_ms"`
}

// GraphNode represents an entity in the Limoxel Knowledge Graph.
type GraphNode struct {
	ID         string            `json:"id"`
	Kind       string            `json:"kind"`
	Name       string            `json:"name"`
	Package    string            `json:"package,omitempty"`
	Location   string            `json:"location,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
}

// GraphRelationship represents a directed edge between two knowledge graph nodes.
type GraphRelationship struct {
	ID         string            `json:"id"`
	SourceID   string            `json:"source_id"`
	TargetID   string            `json:"target_id"`
	Kind       string            `json:"kind"`
	Evidence   string            `json:"evidence,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
}

// GraphFilter defines constraints for graph traversal and export.
type GraphFilter struct {
	EntityTypes       []string `json:"entity_types,omitempty"`
	RelationshipKinds []string `json:"relationship_kinds,omitempty"`
	PackageScope      string   `json:"package_scope,omitempty"`
	MaxDepth          int      `json:"max_depth,omitempty"`
}

// GraphExportFormat specifies output graph representation.
type GraphExportFormat string

const (
	GraphExportJSON     GraphExportFormat = "json"
	GraphExportMermaid  GraphExportFormat = "mermaid"
	GraphExportGraphviz GraphExportFormat = "graphviz"
)

// GraphExportResult contains serialized graph definitions.
type GraphExportResult struct {
	Format    GraphExportFormat `json:"format"`
	Content   string            `json:"content"`
	NodeCount int               `json:"node_count"`
	EdgeCount int               `json:"edge_count"`
}

// ArchitectureReport encapsulates structural component breakdown and boundaries.
type ArchitectureReport struct {
	TotalComponents int                `json:"total_components"`
	LayerCount      int                `json:"layer_count"`
	Violations      []Finding          `json:"violations,omitempty"`
	Findings        []Finding          `json:"findings,omitempty"`
	Metrics         map[string]float64 `json:"metrics,omitempty"`
}

// DependencyReport encapsulates direct, transitive, and circular dependency observations.
type DependencyReport struct {
	TotalDependencies      int                `json:"total_dependencies"`
	DirectDependencies     int                `json:"direct_dependencies"`
	TransitiveDependencies int                `json:"transitive_dependencies"`
	CircularDependencies   int                `json:"circular_dependencies"`
	Findings               []Finding          `json:"findings,omitempty"`
	Metrics                map[string]float64 `json:"metrics,omitempty"`
}

// HealthDimension represents a single scored dimension of repository health.
type HealthDimension struct {
	Name       string             `json:"name"`
	Score      float64            `json:"score"`
	Confidence float64            `json:"confidence"`
	Coverage   float64            `json:"coverage"`
	Weight     float64            `json:"weight"`
	Metrics    map[string]float64 `json:"metrics,omitempty"`
}

// HealthReport provides a multidimensional repository health assessment.
type HealthReport struct {
	OverallScore float64           `json:"overall_score"`
	Grade        string            `json:"grade"`
	Status       string            `json:"status"`
	Dimensions   []HealthDimension `json:"dimensions"`
	Findings     []Finding         `json:"findings,omitempty"`
}

// QualityReport encapsulates code quality, maintainability, and complexity metrics.
type QualityReport struct {
	MaintainabilityScore float64            `json:"maintainability_score"`
	ComplexityScore      float64            `json:"complexity_score"`
	TestabilityScore     float64            `json:"testability_score"`
	TotalIssues          int                `json:"total_issues"`
	Findings             []Finding          `json:"findings,omitempty"`
	Metrics              map[string]float64 `json:"metrics,omitempty"`
}

// ConfigurationReport evaluates repository configuration validity and drift.
type ConfigurationReport struct {
	TotalConfigs  int                `json:"total_configs"`
	ValidConfigs  int                `json:"valid_configs"`
	DriftWarnings int                `json:"drift_warnings"`
	Findings      []Finding          `json:"findings,omitempty"`
	Metrics       map[string]float64 `json:"metrics,omitempty"`
}

// Event represents an immutable typed event delivered to plugins.
type Event struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Timestamp time.Time         `json:"timestamp"`
	Workspace string            `json:"workspace,omitempty"`
	Payload   any               `json:"payload,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// EventHandler defines the callback signature for asynchronous event notification.
type EventHandler func(ctx context.Context, evt Event) error

// Subscription manages an active event listener lifecycle.
type Subscription interface {
	ID() string
	Unsubscribe() error
}
