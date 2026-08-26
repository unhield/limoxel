package reporting

import (
	"fmt"
	"strings"
	"time"
)

// Format represents supported presentation, structured, documentation, and visualization export formats.
type Format string

const (
	// Console / Human-readable formats
	FormatText Format = "text"

	// Structured data formats
	FormatJSON Format = "json"
	FormatYAML Format = "yaml"
	FormatTOML Format = "toml"
	FormatXML  Format = "xml"
	FormatCSV  Format = "csv"

	// Document formats
	FormatMarkdown Format = "markdown"
	FormatHTML     Format = "html"
	FormatPDF      Format = "pdf"

	// Visualization formats
	FormatMermaid     Format = "mermaid"
	FormatGraphviz    Format = "graphviz"
	FormatSVG         Format = "svg"
	FormatPNG         Format = "png"
	FormatInteractive Format = "interactive"
)

// ParseFormat normalizes a format string into a valid Format enum.
func ParseFormat(raw string) (Format, error) {
	clean := strings.ToLower(strings.TrimSpace(raw))
	switch clean {
	case "", "text", "txt", "console":
		return FormatText, nil
	case "json":
		return FormatJSON, nil
	case "yaml", "yml":
		return FormatYAML, nil
	case "toml":
		return FormatTOML, nil
	case "xml":
		return FormatXML, nil
	case "csv":
		return FormatCSV, nil
	case "markdown", "md":
		return FormatMarkdown, nil
	case "html", "htm":
		return FormatHTML, nil
	case "pdf":
		return FormatPDF, nil
	case "mermaid", "mmd":
		return FormatMermaid, nil
	case "graphviz", "dot", "gv":
		return FormatGraphviz, nil
	case "svg":
		return FormatSVG, nil
	case "png":
		return FormatPNG, nil
	case "interactive", "web":
		return FormatInteractive, nil
	default:
		return "", fmt.Errorf("unsupported output format %q", raw)
	}
}

// ReportType identifies standard composed engineering report types.
type ReportType string

const (
	ReportRepository       ReportType = "repository"
	ReportArchitecture     ReportType = "architecture"
	ReportDependency       ReportType = "dependency"
	ReportHealth           ReportType = "health"
	ReportExecutiveSummary ReportType = "summary"
)

// DiagramType identifies types of visual relationship graphs.
type DiagramType string

const (
	DiagramDependency   DiagramType = "dependency"
	DiagramCall         DiagramType = "call"
	DiagramPackage      DiagramType = "package"
	DiagramModule       DiagramType = "module"
	DiagramSymbol       DiagramType = "symbol"
	DiagramArchitecture DiagramType = "architecture"
)

// ReportMetadata represents standard header provenance for generated reports.
type ReportMetadata struct {
	Title         string    `json:"title" yaml:"title" toml:"title" xml:"title"`
	ReportType    string    `json:"report_type" yaml:"report_type" toml:"report_type" xml:"report_type"`
	Repository    string    `json:"repository" yaml:"repository" toml:"repository" xml:"repository"`
	RootPath      string    `json:"root_path" yaml:"root_path" toml:"root_path" xml:"root_path"`
	GeneratedAt   time.Time `json:"generated_at" yaml:"generated_at" toml:"generated_at" xml:"generated_at"`
	Generator     string    `json:"generator" yaml:"generator" toml:"generator" xml:"generator"`
	Version       string    `json:"version" yaml:"version" toml:"version" xml:"version"`
	Authoritative bool      `json:"authoritative" yaml:"authoritative" toml:"authoritative" xml:"authoritative"`
}

// DefaultReportMetadata constructs standardized metadata.
func DefaultReportMetadata(title string, repType ReportType, repoName, rootPath string) ReportMetadata {
	return ReportMetadata{
		Title:         title,
		ReportType:    string(repType),
		Repository:    repoName,
		RootPath:      rootPath,
		GeneratedAt:   time.Now().UTC().Truncate(time.Second),
		Generator:     "Limoxel Engineering Intelligence Platform",
		Version:       "1.0.0",
		Authoritative: true,
	}
}

// CountEntry represents a name-count pair for XML, JSON, YAML, and TOML compatibility.
type CountEntry struct {
	Name  string `json:"name" yaml:"name" toml:"name" xml:"name,attr"`
	Count int    `json:"count" yaml:"count" toml:"count" xml:",chardata"`
}

// MetricEntry represents a key-value metric pair for structured serialization.
type MetricEntry struct {
	Key   string `json:"key" yaml:"key" toml:"key" xml:"key,attr"`
	Value string `json:"value" yaml:"value" toml:"value" xml:",chardata"`
}

// RepositoryReportData holds composed data for a full repository characteristics report.
type RepositoryReportData struct {
	Metadata     ReportMetadata    `json:"metadata" yaml:"metadata" toml:"metadata" xml:"metadata"`
	FileCount    int               `json:"file_count" yaml:"file_count" toml:"file_count" xml:"file_count"`
	DirCount     int               `json:"directory_count" yaml:"directory_count" toml:"directory_count" xml:"directory_count"`
	PackageCount int               `json:"package_count" yaml:"package_count" toml:"package_count" xml:"package_count"`
	SymbolCount  int               `json:"symbol_count" yaml:"symbol_count" toml:"symbol_count" xml:"symbol_count"`
	DepCount     int               `json:"dependency_count" yaml:"dependency_count" toml:"dependency_count" xml:"dependency_count"`
	RelCount     int               `json:"relationship_count" yaml:"relationship_count" toml:"relationship_count" xml:"relationship_count"`
	Languages    []string          `json:"languages" yaml:"languages" toml:"languages" xml:"languages>language"`
	Packages     []string          `json:"packages" yaml:"packages" toml:"packages" xml:"packages>package"`
	TopSymbols   []SymbolSummary   `json:"top_symbols" yaml:"top_symbols" toml:"top_symbols" xml:"top_symbols>symbol"`
	Dependencies []DependencyEntry `json:"dependencies" yaml:"dependencies" toml:"dependencies" xml:"dependencies>dependency"`
}

// ArchitectureReportData holds composed data for an architectural analysis report.
type ArchitectureReportData struct {
	Metadata         ReportMetadata     `json:"metadata" yaml:"metadata" toml:"metadata" xml:"metadata"`
	ArchitectureType string             `json:"architecture_type" yaml:"architecture_type" toml:"architecture_type" xml:"architecture_type"`
	ModuleCount      int                `json:"module_count" yaml:"module_count" toml:"module_count" xml:"module_count"`
	PackageCount     int                `json:"package_count" yaml:"package_count" toml:"package_count" xml:"package_count"`
	Components       []ComponentSummary `json:"components" yaml:"components" toml:"components" xml:"components>component"`
	Boundaries       []BoundarySummary  `json:"boundaries" yaml:"boundaries" toml:"boundaries" xml:"boundaries>boundary"`
	LayerOrder       []string           `json:"layer_order" yaml:"layer_order" toml:"layer_order" xml:"layer_order>layer"`
}

// DependencyReportData holds composed data for dependency analysis.
type DependencyReportData struct {
	Metadata        ReportMetadata    `json:"metadata" yaml:"metadata" toml:"metadata" xml:"metadata"`
	TotalCount      int               `json:"total_count" yaml:"total_count" toml:"total_count" xml:"total_count"`
	DirectCount     int               `json:"direct_count" yaml:"direct_count" toml:"direct_count" xml:"direct_count"`
	TransitiveCount int               `json:"transitive_count" yaml:"transitive_count" toml:"transitive_count" xml:"transitive_count"`
	CircularRisk    bool              `json:"circular_risk" yaml:"circular_risk" toml:"circular_risk" xml:"circular_risk"`
	DirectList      []DependencyEntry `json:"direct_dependencies" yaml:"direct_dependencies" toml:"direct_dependencies" xml:"direct_dependencies>dependency"`
	TransitiveList  []DependencyEntry `json:"transitive_dependencies" yaml:"transitive_dependencies" toml:"transitive_dependencies" xml:"transitive_dependencies>dependency"`
}

// HealthReportData holds composed data for repository quality and risk health.
type HealthReportData struct {
	Metadata      ReportMetadata   `json:"metadata" yaml:"metadata" toml:"metadata" xml:"metadata"`
	OverallScore  float64          `json:"overall_score" yaml:"overall_score" toml:"overall_score" xml:"overall_score"`
	Grade         string           `json:"grade" yaml:"grade" toml:"grade" xml:"grade"`
	TotalFindings int              `json:"total_findings" yaml:"total_findings" toml:"total_findings" xml:"total_findings"`
	SeverityCount []CountEntry     `json:"severity_count" yaml:"severity_count" toml:"severity_count" xml:"severity_counts>entry"`
	CategoryCount []CountEntry     `json:"category_count" yaml:"category_count" toml:"category_count" xml:"category_counts>entry"`
	Findings      []FindingSummary `json:"findings" yaml:"findings" toml:"findings" xml:"findings>finding"`
}

// ExecutiveSummaryData holds high-level overview metrics and status scorecard.
type ExecutiveSummaryData struct {
	Metadata        ReportMetadata          `json:"metadata" yaml:"metadata" toml:"metadata" xml:"metadata"`
	HealthScore     float64                 `json:"health_score" yaml:"health_score" toml:"health_score" xml:"health_score"`
	HealthGrade     string                  `json:"health_grade" yaml:"health_grade" toml:"health_grade" xml:"health_grade"`
	RiskLevel       string                  `json:"risk_level" yaml:"risk_level" toml:"risk_level" xml:"risk_level"`
	SummaryStatus   string                  `json:"summary_status" yaml:"summary_status" toml:"summary_status" xml:"summary_status"`
	KeyMetrics      []MetricEntry           `json:"key_metrics" yaml:"key_metrics" toml:"key_metrics" xml:"key_metrics>metric"`
	Recommendations []RecommendationSummary `json:"recommendations" yaml:"recommendations" toml:"recommendations" xml:"recommendations>recommendation"`
	TopFindings     []FindingSummary        `json:"top_findings" yaml:"top_findings" toml:"top_findings" xml:"top_findings>finding"`
}

// SymbolSummary describes a symbol for reporting.
type SymbolSummary struct {
	ID          string `json:"id" yaml:"id" toml:"id" xml:"id"`
	Name        string `json:"name" yaml:"name" toml:"name" xml:"name"`
	Kind        string `json:"kind" yaml:"kind" toml:"kind" xml:"kind"`
	PackagePath string `json:"package_path" yaml:"package_path" toml:"package_path" xml:"package_path"`
	FilePath    string `json:"file_path" yaml:"file_path" toml:"file_path" xml:"file_path"`
	Exported    bool   `json:"exported" yaml:"exported" toml:"exported" xml:"exported"`
}

// DependencyEntry describes a dependency for reporting.
type DependencyEntry struct {
	Name    string `json:"name" yaml:"name" toml:"name" xml:"name"`
	Version string `json:"version" yaml:"version" toml:"version" xml:"version"`
	Type    string `json:"type" yaml:"type" toml:"type" xml:"type"`
	Direct  bool   `json:"direct" yaml:"direct" toml:"direct" xml:"direct"`
	Scope   string `json:"scope" yaml:"scope" toml:"scope" xml:"scope"`
}

// ComponentSummary describes an architectural component.
type ComponentSummary struct {
	ID           string   `json:"id" yaml:"id" toml:"id" xml:"id"`
	Name         string   `json:"name" yaml:"name" toml:"name" xml:"name"`
	Layer        string   `json:"layer" yaml:"layer" toml:"layer" xml:"layer"`
	PackageCount int      `json:"package_count" yaml:"package_count" toml:"package_count" xml:"package_count"`
	Packages     []string `json:"packages" yaml:"packages" toml:"packages" xml:"packages>package"`
}

// BoundarySummary describes structural boundaries between components.
type BoundarySummary struct {
	FromComponent string `json:"from_component" yaml:"from_component" toml:"from_component" xml:"from_component"`
	ToComponent   string `json:"to_component" yaml:"to_component" toml:"to_component" xml:"to_component"`
	RelationCount int    `json:"relation_count" yaml:"relation_count" toml:"relation_count" xml:"relation_count"`
	Valid         bool   `json:"valid" yaml:"valid" toml:"valid" xml:"valid"`
}

// FindingSummary describes an analysis defect or rule finding.
type FindingSummary struct {
	ID          string `json:"id" yaml:"id" toml:"id" xml:"id"`
	RuleID      string `json:"rule_id" yaml:"rule_id" toml:"rule_id" xml:"rule_id"`
	Severity    string `json:"severity" yaml:"severity" toml:"severity" xml:"severity"`
	Category    string `json:"category" yaml:"category" toml:"category" xml:"category"`
	Title       string `json:"title" yaml:"title" toml:"title" xml:"title"`
	Description string `json:"description" yaml:"description" toml:"description" xml:"description"`
	Location    string `json:"location" yaml:"location" toml:"location" xml:"location"`
	Remediation string `json:"remediation" yaml:"remediation" toml:"remediation" xml:"remediation"`
}

// RecommendationSummary describes an intelligence recommendation.
type RecommendationSummary struct {
	ID       string `json:"id" yaml:"id" toml:"id" xml:"id"`
	Priority string `json:"priority" yaml:"priority" toml:"priority" xml:"priority"`
	Category string `json:"category" yaml:"category" toml:"category" xml:"category"`
	Title    string `json:"title" yaml:"title" toml:"title" xml:"title"`
	Target   string `json:"target" yaml:"target" toml:"target" xml:"target"`
	Action   string `json:"action" yaml:"action" toml:"action" xml:"action"`
}

// GraphVisualData encapsulates graph nodes and edges for visual rendering.
type GraphVisualData struct {
	Title       string      `json:"title" yaml:"title" toml:"title" xml:"title"`
	DiagramType DiagramType `json:"diagram_type" yaml:"diagram_type" toml:"diagram_type" xml:"diagram_type"`
	Nodes       []GraphNode `json:"nodes" yaml:"nodes" toml:"nodes" xml:"nodes>node"`
	Edges       []GraphEdge `json:"edges" yaml:"edges" toml:"edges" xml:"edges>edge"`
	Subgraphs   []Subgraph  `json:"subgraphs" yaml:"subgraphs" toml:"subgraphs" xml:"subgraphs>subgraph"`
}

// GraphNode represents a visual node in a graph.
type GraphNode struct {
	ID        string `json:"id" yaml:"id" toml:"id" xml:"id"`
	Label     string `json:"label" yaml:"label" toml:"label" xml:"label"`
	Kind      string `json:"kind" yaml:"kind" toml:"kind" xml:"kind"`
	Shape     string `json:"shape" yaml:"shape" toml:"shape" xml:"shape"`
	Color     string `json:"color" yaml:"color" toml:"color" xml:"color"`
	ClusterID string `json:"cluster_id" yaml:"cluster_id" toml:"cluster_id" xml:"cluster_id"`
}

// GraphEdge represents a directional edge between two nodes.
type GraphEdge struct {
	Source string  `json:"source" yaml:"source" toml:"source" xml:"source"`
	Target string  `json:"target" yaml:"target" toml:"target" xml:"target"`
	Label  string  `json:"label" yaml:"label" toml:"label" xml:"label"`
	Kind   string  `json:"kind" yaml:"kind" toml:"kind" xml:"kind"`
	Dotted bool    `json:"dotted" yaml:"dotted" toml:"dotted" xml:"dotted"`
	Weight float64 `json:"weight" yaml:"weight" toml:"weight" xml:"weight"`
}

// Subgraph represents a grouped cluster of nodes.
type Subgraph struct {
	ID      string   `json:"id" yaml:"id" toml:"id" xml:"id"`
	Title   string   `json:"title" yaml:"title" toml:"title" xml:"title"`
	NodeIDs []string `json:"node_ids" yaml:"node_ids" toml:"node_ids" xml:"node_ids>node_id"`
}
