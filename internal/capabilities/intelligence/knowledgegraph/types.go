package knowledgegraph

// EntityType represents the category of an engineering entity node.
type EntityType string

const (
	EntityRepository    EntityType = "repository"
	EntityPackage       EntityType = "package"
	EntityModule        EntityType = "module"
	EntityFile          EntityType = "file"
	EntitySymbol        EntityType = "symbol"
	EntityDocumentation EntityType = "documentation"
	EntityConfiguration EntityType = "configuration"
	EntityArchComponent EntityType = "arch_component"
)

// RelationshipKind represents the semantic or structural category of an edge.
type RelationshipKind string

const (
	RelOwns        RelationshipKind = "owns"
	RelBelongsTo   RelationshipKind = "belongs_to"
	RelDependsOn   RelationshipKind = "depends_on"
	RelImports     RelationshipKind = "imports"
	RelCalls       RelationshipKind = "calls"
	RelImplements  RelationshipKind = "implements"
	RelEmbeds      RelationshipKind = "embeds"
	RelDocuments   RelationshipKind = "documents"
	RelConfigures  RelationshipKind = "configures"
	RelSemantic    RelationshipKind = "semantic"
	RelDerivesFrom RelationshipKind = "derives_from"
)

// Direction represents traversal orientation.
type Direction string

const (
	DirOutbound      Direction = "outbound"
	DirInbound       Direction = "inbound"
	DirBidirectional Direction = "bidirectional"
)

// InsightCategory represents the engineering domain of an insight.
type InsightCategory string

const (
	InsightComplexity   InsightCategory = "complexity"
	InsightDependency   InsightCategory = "dependency"
	InsightArchitecture InsightCategory = "architecture"
	InsightGrowth       InsightCategory = "growth"
	InsightRisk         InsightCategory = "risk"
)

// InsightSeverity represents the importance of an engineering insight.
type InsightSeverity string

const (
	SeverityInfo     InsightSeverity = "info"
	SeverityLow      InsightSeverity = "low"
	SeverityMedium   InsightSeverity = "medium"
	SeverityHigh     InsightSeverity = "high"
	SeverityCritical InsightSeverity = "critical"
)
