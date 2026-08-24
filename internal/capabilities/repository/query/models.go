package query

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/repository/graph"
	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
)

// RepositoryMetadataDTO represents immutable high-level metadata about an analyzed repository.
type RepositoryMetadataDTO struct {
	name          string
	owner         string
	root          string
	defaultBranch string
	currentBranch string
	isGit         bool
	totalFiles    int
	totalDirs     int
	totalBytes    int64
	languages     []string
	capabilities  []string
	analysisState string
	analyzedAt    time.Time
}

// NewRepositoryMetadataDTO creates an immutable RepositoryMetadataDTO.
func NewRepositoryMetadataDTO(
	name, owner, root, defaultBranch, currentBranch string,
	isGit bool,
	totalFiles, totalDirs int,
	totalBytes int64,
	languages, capabilities []string,
	analysisState string,
	analyzedAt time.Time,
) *RepositoryMetadataDTO {
	langs := make([]string, len(languages))
	copy(langs, languages)
	sort.Strings(langs)

	caps := make([]string, len(capabilities))
	copy(caps, capabilities)
	sort.Strings(caps)

	return &RepositoryMetadataDTO{
		name:          strings.TrimSpace(name),
		owner:         strings.TrimSpace(owner),
		root:          strings.TrimSpace(root),
		defaultBranch: strings.TrimSpace(defaultBranch),
		currentBranch: strings.TrimSpace(currentBranch),
		isGit:         isGit,
		totalFiles:    totalFiles,
		totalDirs:     totalDirs,
		totalBytes:    totalBytes,
		languages:     langs,
		capabilities:  caps,
		analysisState: strings.TrimSpace(analysisState),
		analyzedAt:    analyzedAt,
	}
}

func (m *RepositoryMetadataDTO) Name() string          { return m.name }
func (m *RepositoryMetadataDTO) Owner() string         { return m.owner }
func (m *RepositoryMetadataDTO) Root() string          { return m.root }
func (m *RepositoryMetadataDTO) DefaultBranch() string { return m.defaultBranch }
func (m *RepositoryMetadataDTO) CurrentBranch() string { return m.currentBranch }
func (m *RepositoryMetadataDTO) IsGit() bool           { return m.isGit }
func (m *RepositoryMetadataDTO) TotalFiles() int       { return m.totalFiles }
func (m *RepositoryMetadataDTO) TotalDirectories() int { return m.totalDirs }
func (m *RepositoryMetadataDTO) TotalBytes() int64     { return m.totalBytes }
func (m *RepositoryMetadataDTO) AnalysisState() string { return m.analysisState }
func (m *RepositoryMetadataDTO) AnalyzedAt() time.Time { return m.analyzedAt }

func (m *RepositoryMetadataDTO) Languages() []string {
	if m == nil || m.languages == nil {
		return nil
	}
	res := make([]string, len(m.languages))
	copy(res, m.languages)
	return res
}

func (m *RepositoryMetadataDTO) Capabilities() []string {
	if m == nil || m.capabilities == nil {
		return nil
	}
	res := make([]string, len(m.capabilities))
	copy(res, m.capabilities)
	return res
}

// RepositoryStatisticsDTO represents deterministic measurements of repository structures.
type RepositoryStatisticsDTO struct {
	fileCount         int
	directoryCount    int
	packageCount      int
	symbolCount       int
	dependencyCount   int
	relationshipCount int
	docCount          int
	configCount       int
	isAvailable       bool
}

// NewRepositoryStatisticsDTO constructs an immutable RepositoryStatisticsDTO.
func NewRepositoryStatisticsDTO(
	fileCount, dirCount, pkgCount, symCount, depCount, relCount, docCount, cfgCount int,
	isAvailable bool,
) *RepositoryStatisticsDTO {
	return &RepositoryStatisticsDTO{
		fileCount:         fileCount,
		directoryCount:    dirCount,
		packageCount:      pkgCount,
		symbolCount:       symCount,
		dependencyCount:   depCount,
		relationshipCount: relCount,
		docCount:          docCount,
		configCount:       cfgCount,
		isAvailable:       isAvailable,
	}
}

func (s *RepositoryStatisticsDTO) FileCount() int         { return s.fileCount }
func (s *RepositoryStatisticsDTO) DirectoryCount() int    { return s.directoryCount }
func (s *RepositoryStatisticsDTO) PackageCount() int      { return s.packageCount }
func (s *RepositoryStatisticsDTO) SymbolCount() int       { return s.symbolCount }
func (s *RepositoryStatisticsDTO) DependencyCount() int   { return s.dependencyCount }
func (s *RepositoryStatisticsDTO) RelationshipCount() int { return s.relationshipCount }
func (s *RepositoryStatisticsDTO) DocCount() int          { return s.docCount }
func (s *RepositoryStatisticsDTO) ConfigCount() int       { return s.configCount }
func (s *RepositoryStatisticsDTO) IsAvailable() bool      { return s.isAvailable }

// SymbolDTO represents an immutable symbol record.
type SymbolDTO struct {
	id          string
	name        string
	kind        symbol.SymbolKind
	filePath    string
	packageName string
	receiver    string
	isExported  bool
	signature   string
	line        int
	doc         string
}

// NewSymbolDTO constructs an immutable SymbolDTO.
func NewSymbolDTO(
	id, name string,
	kind symbol.SymbolKind,
	filePath, packageName, receiver string,
	isExported bool,
	signature string,
	line int,
	doc string,
) *SymbolDTO {
	return &SymbolDTO{
		id:          strings.TrimSpace(id),
		name:        strings.TrimSpace(name),
		kind:        kind,
		filePath:    strings.TrimSpace(filePath),
		packageName: strings.TrimSpace(packageName),
		receiver:    strings.TrimSpace(receiver),
		isExported:  isExported,
		signature:   strings.TrimSpace(signature),
		line:        line,
		doc:         strings.TrimSpace(doc),
	}
}

func (s *SymbolDTO) ID() string              { return s.id }
func (s *SymbolDTO) Name() string            { return s.name }
func (s *SymbolDTO) Kind() symbol.SymbolKind { return s.kind }
func (s *SymbolDTO) FilePath() string        { return s.filePath }
func (s *SymbolDTO) PackageName() string     { return s.packageName }
func (s *SymbolDTO) Receiver() string        { return s.receiver }
func (s *SymbolDTO) IsExported() bool        { return s.isExported }
func (s *SymbolDTO) Signature() string       { return s.signature }
func (s *SymbolDTO) Line() int               { return s.line }
func (s *SymbolDTO) Doc() string             { return s.doc }

// DependencyDTO represents an immutable external or internal dependency.
type DependencyDTO struct {
	name     string
	version  string
	isDirect bool
	depType  string
}

// NewDependencyDTO creates an immutable DependencyDTO.
func NewDependencyDTO(name, version string, isDirect bool, depType string) *DependencyDTO {
	return &DependencyDTO{
		name:     strings.TrimSpace(name),
		version:  strings.TrimSpace(version),
		isDirect: isDirect,
		depType:  strings.TrimSpace(depType),
	}
}

func (d *DependencyDTO) Name() string    { return d.name }
func (d *DependencyDTO) Version() string { return d.version }
func (d *DependencyDTO) IsDirect() bool  { return d.isDirect }
func (d *DependencyDTO) Type() string    { return d.depType }

// CallEdgeDTO represents an immutable function or method invocation edge.
type CallEdgeDTO struct {
	callerID   string
	calleeID   string
	kind       string
	filePath   string
	lineNumber int
}

// NewCallEdgeDTO creates an immutable CallEdgeDTO.
func NewCallEdgeDTO(callerID, calleeID, kind, filePath string, lineNumber int) *CallEdgeDTO {
	return &CallEdgeDTO{
		callerID:   strings.TrimSpace(callerID),
		calleeID:   strings.TrimSpace(calleeID),
		kind:       strings.TrimSpace(kind),
		filePath:   strings.TrimSpace(filePath),
		lineNumber: lineNumber,
	}
}

func (c *CallEdgeDTO) CallerID() string { return c.callerID }
func (c *CallEdgeDTO) CalleeID() string { return c.calleeID }
func (c *CallEdgeDTO) Kind() string     { return c.kind }
func (c *CallEdgeDTO) FilePath() string { return c.filePath }
func (c *CallEdgeDTO) LineNumber() int  { return c.lineNumber }

// RelationshipDTO represents an immutable graph relationship.
type RelationshipDTO struct {
	id         string
	relType    graph.RelationshipType
	sourceID   string
	targetID   string
	provenance []string
	metadata   map[string]string
}

// NewRelationshipDTO creates an immutable RelationshipDTO.
func NewRelationshipDTO(
	id string,
	relType graph.RelationshipType,
	sourceID, targetID string,
	provenance []string,
	metadata map[string]string,
) *RelationshipDTO {
	provList := make([]string, len(provenance))
	copy(provList, provenance)
	sort.Strings(provList)

	meta := make(map[string]string, len(metadata))
	for k, v := range metadata {
		meta[k] = v
	}

	return &RelationshipDTO{
		id:         strings.TrimSpace(id),
		relType:    relType,
		sourceID:   strings.TrimSpace(sourceID),
		targetID:   strings.TrimSpace(targetID),
		provenance: provList,
		metadata:   meta,
	}
}

func (r *RelationshipDTO) ID() string                   { return r.id }
func (r *RelationshipDTO) Type() graph.RelationshipType { return r.relType }
func (r *RelationshipDTO) SourceID() string             { return r.sourceID }
func (r *RelationshipDTO) TargetID() string             { return r.targetID }

func (r *RelationshipDTO) Provenance() []string {
	if r == nil || r.provenance == nil {
		return nil
	}
	res := make([]string, len(r.provenance))
	copy(res, r.provenance)
	return res
}

func (r *RelationshipDTO) Metadata() map[string]string {
	if r == nil || r.metadata == nil {
		return nil
	}
	res := make(map[string]string, len(r.metadata))
	for k, v := range r.metadata {
		res[k] = v
	}
	return res
}

// GraphNodeDTO represents an immutable knowledge graph node.
type GraphNodeDTO struct {
	id          string
	nodeType    graph.NodeType
	name        string
	path        string
	module      string
	packageName string
	metadata    map[string]string
}

// NewGraphNodeDTO creates an immutable GraphNodeDTO.
func NewGraphNodeDTO(
	id string,
	nodeType graph.NodeType,
	name, path, module, packageName string,
	metadata map[string]string,
) *GraphNodeDTO {
	meta := make(map[string]string, len(metadata))
	for k, v := range metadata {
		meta[k] = v
	}

	return &GraphNodeDTO{
		id:          strings.TrimSpace(id),
		nodeType:    nodeType,
		name:        strings.TrimSpace(name),
		path:        strings.TrimSpace(path),
		module:      strings.TrimSpace(module),
		packageName: strings.TrimSpace(packageName),
		metadata:    meta,
	}
}

func (n *GraphNodeDTO) ID() string           { return n.id }
func (n *GraphNodeDTO) Type() graph.NodeType { return n.nodeType }
func (n *GraphNodeDTO) Name() string         { return n.name }
func (n *GraphNodeDTO) Path() string         { return n.path }
func (n *GraphNodeDTO) Module() string       { return n.module }
func (n *GraphNodeDTO) Package() string      { return n.packageName }

func (n *GraphNodeDTO) Metadata() map[string]string {
	if n == nil || n.metadata == nil {
		return nil
	}
	res := make(map[string]string, len(n.metadata))
	for k, v := range n.metadata {
		res[k] = v
	}
	return res
}

// TraversalResultDTO represents the result of a bounded graph traversal.
type TraversalResultDTO struct {
	startNodeID   string
	direction     graph.Direction
	maxDepth      int
	nodes         []*GraphNodeDTO
	relationships []*RelationshipDTO
}

// NewTraversalResultDTO constructs an immutable TraversalResultDTO.
func NewTraversalResultDTO(
	startNodeID string,
	direction graph.Direction,
	maxDepth int,
	nodes []*GraphNodeDTO,
	relationships []*RelationshipDTO,
) *TraversalResultDTO {
	nList := make([]*GraphNodeDTO, len(nodes))
	copy(nList, nodes)

	rList := make([]*RelationshipDTO, len(relationships))
	copy(rList, relationships)

	return &TraversalResultDTO{
		startNodeID:   strings.TrimSpace(startNodeID),
		direction:     direction,
		maxDepth:      maxDepth,
		nodes:         nList,
		relationships: rList,
	}
}

func (t *TraversalResultDTO) StartNodeID() string        { return t.startNodeID }
func (t *TraversalResultDTO) Direction() graph.Direction { return t.direction }
func (t *TraversalResultDTO) MaxDepth() int              { return t.maxDepth }
func (t *TraversalResultDTO) TotalNodes() int            { return len(t.nodes) }
func (t *TraversalResultDTO) TotalRelationships() int    { return len(t.relationships) }

func (t *TraversalResultDTO) Nodes() []*GraphNodeDTO {
	if t == nil || t.nodes == nil {
		return nil
	}
	res := make([]*GraphNodeDTO, len(t.nodes))
	copy(res, t.nodes)
	return res
}

func (t *TraversalResultDTO) Relationships() []*RelationshipDTO {
	if t == nil || t.relationships == nil {
		return nil
	}
	res := make([]*RelationshipDTO, len(t.relationships))
	copy(res, t.relationships)
	return res
}

// SearchResultItem represents an individual match from the SearchEngine.
type SearchResultItem struct {
	entityID    string
	domain      SearchDomain
	name        string
	path        string
	packageName string
	scope       string
	score       float64
	snippet     string
	highlights  []int
}

// NewSearchResultItem creates an immutable SearchResultItem.
func NewSearchResultItem(
	entityID string,
	domain SearchDomain,
	name, path, packageName, scope string,
	score float64,
	snippet string,
	highlights []int,
) *SearchResultItem {
	hl := make([]int, len(highlights))
	copy(hl, highlights)

	return &SearchResultItem{
		entityID:    strings.TrimSpace(entityID),
		domain:      domain,
		name:        strings.TrimSpace(name),
		path:        strings.TrimSpace(path),
		packageName: strings.TrimSpace(packageName),
		scope:       strings.TrimSpace(scope),
		score:       score,
		snippet:     strings.TrimSpace(snippet),
		highlights:  hl,
	}
}

func (s *SearchResultItem) EntityID() string     { return s.entityID }
func (s *SearchResultItem) Domain() SearchDomain { return s.domain }
func (s *SearchResultItem) Name() string         { return s.name }
func (s *SearchResultItem) Path() string         { return s.path }
func (s *SearchResultItem) PackageName() string  { return s.packageName }
func (s *SearchResultItem) Scope() string        { return s.scope }
func (s *SearchResultItem) Score() float64       { return s.score }
func (s *SearchResultItem) Snippet() string      { return s.snippet }

func (s *SearchResultItem) Highlights() []int {
	if s == nil || s.highlights == nil {
		return nil
	}
	res := make([]int, len(s.highlights))
	copy(res, s.highlights)
	return res
}

func (s *SearchResultItem) String() string {
	if s == nil {
		return ""
	}
	return fmt.Sprintf("[%s] %s (%s) score=%.2f", s.domain, s.name, s.entityID, s.score)
}

// SearchResultDTO encapsulates a complete search query response.
type SearchResultDTO struct {
	query             string
	domain            SearchDomain
	totalMatches      int
	items             []*SearchResultItem
	executionDuration time.Duration
}

// NewSearchResultDTO constructs an immutable SearchResultDTO.
func NewSearchResultDTO(
	query string,
	domain SearchDomain,
	items []*SearchResultItem,
	duration time.Duration,
) *SearchResultDTO {
	cleanItems := make([]*SearchResultItem, len(items))
	copy(cleanItems, items)

	return &SearchResultDTO{
		query:             strings.TrimSpace(query),
		domain:            domain,
		totalMatches:      len(cleanItems),
		items:             cleanItems,
		executionDuration: duration,
	}
}

func (s *SearchResultDTO) Query() string                    { return s.query }
func (s *SearchResultDTO) Domain() SearchDomain             { return s.domain }
func (s *SearchResultDTO) TotalMatches() int                { return s.totalMatches }
func (s *SearchResultDTO) ExecutionDuration() time.Duration { return s.executionDuration }

func (s *SearchResultDTO) Items() []*SearchResultItem {
	if s == nil || s.items == nil {
		return nil
	}
	res := make([]*SearchResultItem, len(s.items))
	copy(res, s.items)
	return res
}
