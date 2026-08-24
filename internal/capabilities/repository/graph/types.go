package graph

// NodeType identifies the category of an engineering entity in the knowledge graph.
type NodeType string

const (
	// NodeRepository represents the analyzed repository root entity.
	NodeRepository NodeType = "repository"

	// NodeModule represents a detected repository module / build unit.
	NodeModule NodeType = "module"

	// NodePackage represents an established package within a module or repository.
	NodePackage NodeType = "package"

	// NodeFile represents an indexed source, test, doc, or config file.
	NodeFile NodeType = "file"

	// NodeSymbol represents an AST symbol (struct, interface, func, method, etc.).
	NodeSymbol NodeType = "symbol"

	// NodeDoc represents a documentation artifact (README, markdown docs).
	NodeDoc NodeType = "documentation"

	// NodeConfig represents a configuration artifact (JSON, YAML, TOML, etc.).
	NodeConfig NodeType = "configuration"
)

// String returns the string representation of NodeType.
func (t NodeType) String() string {
	return string(t)
}

// RelationshipType defines the semantic connection between two graph nodes.
type RelationshipType string

const (
	// RelContains represents structural hierarchy (Repo->Mod, Mod->Pkg, Pkg->File, File->Sym).
	RelContains RelationshipType = "contains"

	// RelImports represents source or package import dependencies.
	RelImports RelationshipType = "imports"

	// RelImplements represents concrete types satisfying interface contracts.
	RelImplements RelationshipType = "implements"

	// RelCalls represents function or method invocation dispatches.
	RelCalls RelationshipType = "calls"

	// RelReferences represents symbol references, type usages, and field access.
	RelReferences RelationshipType = "references"

	// RelDependsOn represents project-level and package-level external dependencies.
	RelDependsOn RelationshipType = "depends_on"

	// RelDocuments represents documentation artifacts describing code entities.
	RelDocuments RelationshipType = "documents"

	// RelConfigures represents configuration artifacts configuring components or modules.
	RelConfigures RelationshipType = "configures"
)

// String returns the string representation of RelationshipType.
func (r RelationshipType) String() string {
	return string(r)
}

// Direction defines the edge traversal direction during graph queries.
type Direction string

const (
	// DirOutbound traverses along source -> target edge direction.
	DirOutbound Direction = "outbound"

	// DirInbound traverses in reverse along target <- source edge direction.
	DirInbound Direction = "inbound"

	// DirBoth traverses in both outbound and inbound directions.
	DirBoth Direction = "both"
)

// String returns the string representation of Direction.
func (d Direction) String() string {
	return string(d)
}

// ProvenanceSource identifies the authoritative upstream capability establishing a fact or edge.
type ProvenanceSource string

const (
	// ProvDiscovery indicates the edge was established by Repository Discovery.
	ProvDiscovery ProvenanceSource = "discovery"

	// ProvMetadata indicates the edge was established by Repository Metadata.
	ProvMetadata ProvenanceSource = "metadata"

	// ProvLanguage indicates the edge was established by Language Detection.
	ProvLanguage ProvenanceSource = "language"

	// ProvDependency indicates the edge was established by Dependency Analysis.
	ProvDependency ProvenanceSource = "dependency"

	// ProvIndexing indicates the edge was established by Source Code Indexing.
	ProvIndexing ProvenanceSource = "indexing"

	// ProvSymbol indicates the edge was established by AST & Symbol Engine.
	ProvSymbol ProvenanceSource = "symbol"

	// ProvXRef indicates the edge was established by Cross-Reference Engine.
	ProvXRef ProvenanceSource = "xref"
)

// String returns the string representation of ProvenanceSource.
func (p ProvenanceSource) String() string {
	return string(p)
}

// ValidationSeverity defines the classification level of graph integrity findings.
type ValidationSeverity string

const (
	// ValMissingNode indicates a relationship references a non-existent node ID.
	ValMissingNode ValidationSeverity = "MISSING_NODE"

	// ValInvalidEdge indicates an illegal source/target type combination.
	ValInvalidEdge ValidationSeverity = "INVALID_EDGE"

	// ValDuplicateEdge indicates duplicate relationship identities.
	ValDuplicateEdge ValidationSeverity = "DUPLICATE_EDGE"

	// ValOrphanNode indicates an isolated node with zero incoming/outgoing edges.
	ValOrphanNode ValidationSeverity = "ORPHAN_NODE"
)

// String returns the string representation of ValidationSeverity.
func (v ValidationSeverity) String() string {
	return string(v)
}
