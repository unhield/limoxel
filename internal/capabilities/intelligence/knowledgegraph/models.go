package knowledgegraph

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
)

// GraphEntity represents an immutable node in the knowledge graph.
type GraphEntity struct {
	id          string
	entityType  EntityType
	name        string
	packagePath string
	filePath    string
	pos         *symbol.SourcePosition
	attributes  map[string]string
	provenance  string
}

// NewGraphEntity constructs an immutable GraphEntity.
func NewGraphEntity(
	id string,
	entityType EntityType,
	name string,
	packagePath string,
	filePath string,
	pos *symbol.SourcePosition,
	attributes map[string]string,
	provenance string,
) *GraphEntity {
	copiedAttrs := make(map[string]string)
	for k, v := range attributes {
		copiedAttrs[k] = v
	}
	return &GraphEntity{
		id:          id,
		entityType:  entityType,
		name:        name,
		packagePath: packagePath,
		filePath:    filePath,
		pos:         pos,
		attributes:  copiedAttrs,
		provenance:  provenance,
	}
}

func (e *GraphEntity) ID() string                       { return e.id }
func (e *GraphEntity) Type() EntityType                 { return e.entityType }
func (e *GraphEntity) Name() string                     { return e.name }
func (e *GraphEntity) PackagePath() string              { return e.packagePath }
func (e *GraphEntity) FilePath() string                 { return e.filePath }
func (e *GraphEntity) Position() *symbol.SourcePosition { return e.pos }
func (e *GraphEntity) Provenance() string               { return e.provenance }

func (e *GraphEntity) Attributes() map[string]string {
	cp := make(map[string]string, len(e.attributes))
	for k, v := range e.attributes {
		cp[k] = v
	}
	return cp
}

func (e *GraphEntity) Attribute(key string) string {
	if e.attributes == nil {
		return ""
	}
	return e.attributes[key]
}

// GraphRelationship represents an immutable edge in the knowledge graph.
type GraphRelationship struct {
	id         string
	sourceID   string
	targetID   string
	kind       RelationshipKind
	evidence   string
	provenance string
	confidence float64
	attributes map[string]string
}

// NewGraphRelationship constructs an immutable GraphRelationship.
func NewGraphRelationship(
	sourceID string,
	targetID string,
	kind RelationshipKind,
	evidence string,
	provenance string,
	confidence float64,
	attributes map[string]string,
) *GraphRelationship {
	id := CanonicalRelationshipID(sourceID, targetID, kind)
	copiedAttrs := make(map[string]string)
	for k, v := range attributes {
		copiedAttrs[k] = v
	}
	return &GraphRelationship{
		id:         id,
		sourceID:   sourceID,
		targetID:   targetID,
		kind:       kind,
		evidence:   evidence,
		provenance: provenance,
		confidence: confidence,
		attributes: copiedAttrs,
	}
}

func (r *GraphRelationship) ID() string             { return r.id }
func (r *GraphRelationship) SourceID() string       { return r.sourceID }
func (r *GraphRelationship) TargetID() string       { return r.targetID }
func (r *GraphRelationship) Kind() RelationshipKind { return r.kind }
func (r *GraphRelationship) Evidence() string       { return r.evidence }
func (r *GraphRelationship) Provenance() string     { return r.provenance }
func (r *GraphRelationship) Confidence() float64    { return r.confidence }

func (r *GraphRelationship) Attributes() map[string]string {
	cp := make(map[string]string, len(r.attributes))
	for k, v := range r.attributes {
		cp[k] = v
	}
	return cp
}

// GraphPath represents a sequential traversal path between entities.
type GraphPath struct {
	startID       string
	endID         string
	entities      []*GraphEntity
	relationships []*GraphRelationship
}

// NewGraphPath constructs an immutable GraphPath.
func NewGraphPath(entities []*GraphEntity, relationships []*GraphRelationship) *GraphPath {
	var startID, endID string
	if len(entities) > 0 {
		startID = entities[0].ID()
		endID = entities[len(entities)-1].ID()
	}
	return &GraphPath{
		startID:       startID,
		endID:         endID,
		entities:      append([]*GraphEntity(nil), entities...),
		relationships: append([]*GraphRelationship(nil), relationships...),
	}
}

func (p *GraphPath) StartID() string          { return p.startID }
func (p *GraphPath) EndID() string            { return p.endID }
func (p *GraphPath) Entities() []*GraphEntity { return append([]*GraphEntity(nil), p.entities...) }
func (p *GraphPath) Relationships() []*GraphRelationship {
	return append([]*GraphRelationship(nil), p.relationships...)
}
func (p *GraphPath) Length() int { return len(p.relationships) }

// EngineeringInsight represents an evidence-backed engineering observation.
type EngineeringInsight struct {
	id          string
	category    InsightCategory
	severity    InsightSeverity
	title       string
	description string
	targetID    string
	evidence    string
	provenance  string
	metrics     map[string]float64
}

// NewEngineeringInsight constructs an immutable EngineeringInsight.
func NewEngineeringInsight(
	category InsightCategory,
	severity InsightSeverity,
	title string,
	description string,
	targetID string,
	evidence string,
	provenance string,
	metrics map[string]float64,
) *EngineeringInsight {
	hasher := sha256.New()
	hasher.Write(fmt.Appendf(nil, "%s:%s:%s:%s", category, severity, targetID, title))
	id := fmt.Sprintf("insight:%s:%s:%s", category, targetID, hex.EncodeToString(hasher.Sum(nil))[:12])

	copiedMetrics := make(map[string]float64)
	for k, v := range metrics {
		copiedMetrics[k] = v
	}

	return &EngineeringInsight{
		id:          id,
		category:    category,
		severity:    severity,
		title:       title,
		description: description,
		targetID:    targetID,
		evidence:    evidence,
		provenance:  provenance,
		metrics:     copiedMetrics,
	}
}

func (i *EngineeringInsight) ID() string                { return i.id }
func (i *EngineeringInsight) Category() InsightCategory { return i.category }
func (i *EngineeringInsight) Severity() InsightSeverity { return i.severity }
func (i *EngineeringInsight) Title() string             { return i.title }
func (i *EngineeringInsight) Description() string       { return i.description }
func (i *EngineeringInsight) TargetID() string          { return i.targetID }
func (i *EngineeringInsight) Evidence() string          { return i.evidence }
func (i *EngineeringInsight) Provenance() string        { return i.provenance }

func (i *EngineeringInsight) Metrics() map[string]float64 {
	cp := make(map[string]float64, len(i.metrics))
	for k, v := range i.metrics {
		cp[k] = v
	}
	return cp
}

// Canonical ID Builders
func CanonicalEntityID(entityType EntityType, qualifier string) string {
	return fmt.Sprintf("%s:%s", entityType, qualifier)
}

func CanonicalRelationshipID(sourceID, targetID string, kind RelationshipKind) string {
	return fmt.Sprintf("rel:%s->%s:%s", sourceID, targetID, kind)
}

// KnowledgeGraphModel represents the complete, indexed, and deterministic knowledge graph.
type KnowledgeGraphModel struct {
	rootPath      string
	entities      []*GraphEntity
	relationships []*GraphRelationship
	insights      []*EngineeringInsight
	entityMap     map[string]*GraphEntity
	relMap        map[string]*GraphRelationship
	outEdges      map[string][]*GraphRelationship
	inEdges       map[string][]*GraphRelationship
	typeIndex     map[EntityType][]*GraphEntity
	generatedAt   time.Time
}

// NewKnowledgeGraphModel constructs an indexed KnowledgeGraphModel.
func NewKnowledgeGraphModel(
	rootPath string,
	entities []*GraphEntity,
	relationships []*GraphRelationship,
	insights []*EngineeringInsight,
	generatedAt time.Time,
) *KnowledgeGraphModel {
	sortedEntities := DeduplicateAndSortEntities(entities)
	sortedRels := DeduplicateAndSortRelationships(relationships)
	sortedInsights := DeduplicateAndSortInsights(insights)

	entityMap := make(map[string]*GraphEntity, len(sortedEntities))
	typeIndex := make(map[EntityType][]*GraphEntity)
	for _, e := range sortedEntities {
		if e != nil {
			entityMap[e.ID()] = e
			typeIndex[e.Type()] = append(typeIndex[e.Type()], e)
		}
	}

	relMap := make(map[string]*GraphRelationship, len(sortedRels))
	outEdges := make(map[string][]*GraphRelationship)
	inEdges := make(map[string][]*GraphRelationship)

	for _, r := range sortedRels {
		if r != nil {
			relMap[r.ID()] = r
			outEdges[r.SourceID()] = append(outEdges[r.SourceID()], r)
			inEdges[r.TargetID()] = append(inEdges[r.TargetID()], r)
		}
	}

	return &KnowledgeGraphModel{
		rootPath:      rootPath,
		entities:      sortedEntities,
		relationships: sortedRels,
		insights:      sortedInsights,
		entityMap:     entityMap,
		relMap:        relMap,
		outEdges:      outEdges,
		inEdges:       inEdges,
		typeIndex:     typeIndex,
		generatedAt:   generatedAt,
	}
}

func (m *KnowledgeGraphModel) RootPath() string { return m.rootPath }
func (m *KnowledgeGraphModel) Entities() []*GraphEntity {
	return append([]*GraphEntity(nil), m.entities...)
}
func (m *KnowledgeGraphModel) Relationships() []*GraphRelationship {
	return append([]*GraphRelationship(nil), m.relationships...)
}
func (m *KnowledgeGraphModel) Insights() []*EngineeringInsight {
	return append([]*EngineeringInsight(nil), m.insights...)
}
func (m *KnowledgeGraphModel) GeneratedAt() time.Time  { return m.generatedAt }
func (m *KnowledgeGraphModel) TotalEntities() int      { return len(m.entities) }
func (m *KnowledgeGraphModel) TotalRelationships() int { return len(m.relationships) }
func (m *KnowledgeGraphModel) TotalInsights() int      { return len(m.insights) }

func (m *KnowledgeGraphModel) EntityByID(id string) *GraphEntity {
	return m.entityMap[id]
}

func (m *KnowledgeGraphModel) RelationshipByID(id string) *GraphRelationship {
	return m.relMap[id]
}

func (m *KnowledgeGraphModel) EntitiesByType(t EntityType) []*GraphEntity {
	return append([]*GraphEntity(nil), m.typeIndex[t]...)
}

func (m *KnowledgeGraphModel) OutboundRelationships(sourceID string) []*GraphRelationship {
	return append([]*GraphRelationship(nil), m.outEdges[sourceID]...)
}

func (m *KnowledgeGraphModel) InboundRelationships(targetID string) []*GraphRelationship {
	return append([]*GraphRelationship(nil), m.inEdges[targetID]...)
}

func (m *KnowledgeGraphModel) RelationshipsBetween(sourceID, targetID string) []*GraphRelationship {
	var result []*GraphRelationship
	for _, r := range m.outEdges[sourceID] {
		if r.TargetID() == targetID {
			result = append(result, r)
		}
	}
	return result
}

// Deterministic Deduplication and Sorting Helpers
func DeduplicateAndSortEntities(entities []*GraphEntity) []*GraphEntity {
	seen := make(map[string]*GraphEntity)
	for _, e := range entities {
		if e == nil {
			continue
		}
		if _, exists := seen[e.ID()]; !exists {
			seen[e.ID()] = e
		}
	}
	result := make([]*GraphEntity, 0, len(seen))
	for _, e := range seen {
		result = append(result, e)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID() < result[j].ID()
	})
	return result
}

func DeduplicateAndSortRelationships(rels []*GraphRelationship) []*GraphRelationship {
	seen := make(map[string]*GraphRelationship)
	for _, r := range rels {
		if r == nil {
			continue
		}
		if _, exists := seen[r.ID()]; !exists {
			seen[r.ID()] = r
		}
	}
	result := make([]*GraphRelationship, 0, len(seen))
	for _, r := range seen {
		result = append(result, r)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID() < result[j].ID()
	})
	return result
}

func DeduplicateAndSortInsights(insights []*EngineeringInsight) []*EngineeringInsight {
	seen := make(map[string]*EngineeringInsight)
	for _, in := range insights {
		if in == nil {
			continue
		}
		if _, exists := seen[in.ID()]; !exists {
			seen[in.ID()] = in
		}
	}
	result := make([]*EngineeringInsight, 0, len(seen))
	for _, in := range seen {
		result = append(result, in)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID() < result[j].ID()
	})
	return result
}
