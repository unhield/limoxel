package graph

import (
	"fmt"
	"sort"
	"strings"
)

// Node represents an engineering entity within the knowledge graph.
type Node struct {
	id       string
	nodeType NodeType
	name     string
	path     string
	module   string
	pkg      string
	metadata map[string]string
}

// NewNode constructs an immutable Node.
func NewNode(
	id string,
	nodeType NodeType,
	name string,
	path string,
	module string,
	pkg string,
	metadata map[string]string,
) *Node {
	cleanMeta := make(map[string]string, len(metadata))
	for k, v := range metadata {
		cleanMeta[k] = v
	}

	return &Node{
		id:       strings.TrimSpace(id),
		nodeType: nodeType,
		name:     strings.TrimSpace(name),
		path:     strings.TrimSpace(path),
		module:   strings.TrimSpace(module),
		pkg:      strings.TrimSpace(pkg),
		metadata: cleanMeta,
	}
}

// ID returns the unique deterministic identifier of the node.
func (n *Node) ID() string {
	if n == nil {
		return ""
	}
	return n.id
}

// Type returns the NodeType category of the node.
func (n *Node) Type() NodeType {
	if n == nil {
		return ""
	}
	return n.nodeType
}

// Name returns the human-readable display name.
func (n *Node) Name() string {
	if n == nil {
		return ""
	}
	return n.name
}

// Path returns the repository-relative file or directory path.
func (n *Node) Path() string {
	if n == nil {
		return ""
	}
	return n.path
}

// Module returns the module name if applicable.
func (n *Node) Module() string {
	if n == nil {
		return ""
	}
	return n.module
}

// Package returns the package name/path if applicable.
func (n *Node) Package() string {
	if n == nil {
		return ""
	}
	return n.pkg
}

// Metadata returns a defensive copy of node metadata.
func (n *Node) Metadata() map[string]string {
	if n == nil || n.metadata == nil {
		return make(map[string]string)
	}
	cloned := make(map[string]string, len(n.metadata))
	for k, v := range n.metadata {
		cloned[k] = v
	}
	return cloned
}

// GetMetadata returns a single metadata value by key.
func (n *Node) GetMetadata(key string) (string, bool) {
	if n == nil || n.metadata == nil {
		return "", false
	}
	v, ok := n.metadata[key]
	return v, ok
}

// Relationship represents a directed semantic connection between two nodes.
type Relationship struct {
	id         string
	relType    RelationshipType
	sourceID   string
	targetID   string
	provenance []ProvenanceSource
	metadata   map[string]string
}

// NewRelationship constructs an immutable Relationship.
func NewRelationship(
	relType RelationshipType,
	sourceID string,
	targetID string,
	provenance []ProvenanceSource,
	metadata map[string]string,
) *Relationship {
	sID := strings.TrimSpace(sourceID)
	tID := strings.TrimSpace(targetID)
	canonicalID := fmt.Sprintf("rel:%s:%s->%s", relType, sID, tID)

	cleanMeta := make(map[string]string, len(metadata))
	for k, v := range metadata {
		cleanMeta[k] = v
	}

	var provList []ProvenanceSource
	provSet := make(map[ProvenanceSource]bool, len(provenance))
	for _, p := range provenance {
		if p != "" && !provSet[p] {
			provSet[p] = true
			provList = append(provList, p)
		}
	}
	sort.Slice(provList, func(i, j int) bool {
		return provList[i] < provList[j]
	})

	return &Relationship{
		id:         canonicalID,
		relType:    relType,
		sourceID:   sID,
		targetID:   tID,
		provenance: provList,
		metadata:   cleanMeta,
	}
}

// ID returns the unique canonical relationship identifier.
func (r *Relationship) ID() string {
	if r == nil {
		return ""
	}
	return r.id
}

// Type returns the RelationshipType classification.
func (r *Relationship) Type() RelationshipType {
	if r == nil {
		return ""
	}
	return r.relType
}

// SourceID returns the source node ID.
func (r *Relationship) SourceID() string {
	if r == nil {
		return ""
	}
	return r.sourceID
}

// TargetID returns the target node ID.
func (r *Relationship) TargetID() string {
	if r == nil {
		return ""
	}
	return r.targetID
}

// Provenance returns a defensive copy of upstream sources that established this relationship.
func (r *Relationship) Provenance() []ProvenanceSource {
	if r == nil || r.provenance == nil {
		return nil
	}
	cloned := make([]ProvenanceSource, len(r.provenance))
	copy(cloned, r.provenance)
	return cloned
}

// Metadata returns a defensive copy of relationship metadata.
func (r *Relationship) Metadata() map[string]string {
	if r == nil || r.metadata == nil {
		return make(map[string]string)
	}
	cloned := make(map[string]string, len(r.metadata))
	for k, v := range r.metadata {
		cloned[k] = v
	}
	return cloned
}

// GetMetadata returns a metadata value by key.
func (r *Relationship) GetMetadata(key string) (string, bool) {
	if r == nil || r.metadata == nil {
		return "", false
	}
	v, ok := r.metadata[key]
	return v, ok
}

// ValidationIssue represents a graph integrity observation.
type ValidationIssue struct {
	severity       ValidationSeverity
	code           string
	message        string
	relationshipID string
	sourceID       string
	targetID       string
	nodeID         string
}

// NewValidationIssue constructs an immutable ValidationIssue.
func NewValidationIssue(
	severity ValidationSeverity,
	code string,
	message string,
	relationshipID string,
	sourceID string,
	targetID string,
	nodeID string,
) *ValidationIssue {
	return &ValidationIssue{
		severity:       severity,
		code:           strings.TrimSpace(code),
		message:        strings.TrimSpace(message),
		relationshipID: strings.TrimSpace(relationshipID),
		sourceID:       strings.TrimSpace(sourceID),
		targetID:       strings.TrimSpace(targetID),
		nodeID:         strings.TrimSpace(nodeID),
	}
}

// Severity returns the issue severity.
func (v *ValidationIssue) Severity() ValidationSeverity {
	if v == nil {
		return ""
	}
	return v.severity
}

// Code returns the deterministic issue code.
func (v *ValidationIssue) Code() string {
	if v == nil {
		return ""
	}
	return v.code
}

// Message returns the explanatory description.
func (v *ValidationIssue) Message() string {
	if v == nil {
		return ""
	}
	return v.message
}

// RelationshipID returns the offending relationship ID if applicable.
func (v *ValidationIssue) RelationshipID() string {
	if v == nil {
		return ""
	}
	return v.relationshipID
}

// SourceID returns the offending source node ID if applicable.
func (v *ValidationIssue) SourceID() string {
	if v == nil {
		return ""
	}
	return v.sourceID
}

// TargetID returns the offending target node ID if applicable.
func (v *ValidationIssue) TargetID() string {
	if v == nil {
		return ""
	}
	return v.targetID
}

// NodeID returns the offending node ID if applicable.
func (v *ValidationIssue) NodeID() string {
	if v == nil {
		return ""
	}
	return v.nodeID
}

// ValidationReport consolidates graph integrity results.
type ValidationReport struct {
	issues         []*ValidationIssue
	missingNodes   []*ValidationIssue
	invalidEdges   []*ValidationIssue
	duplicateEdges []*ValidationIssue
	orphanNodes    []*ValidationIssue
}

// NewValidationReport constructs an immutable ValidationReport.
func NewValidationReport(issues []*ValidationIssue) *ValidationReport {
	var missing []*ValidationIssue
	var invalid []*ValidationIssue
	var dups []*ValidationIssue
	var orphans []*ValidationIssue

	cleanIssues := make([]*ValidationIssue, len(issues))
	copy(cleanIssues, issues)

	sort.Slice(cleanIssues, func(i, j int) bool {
		if cleanIssues[i].severity != cleanIssues[j].severity {
			return cleanIssues[i].severity < cleanIssues[j].severity
		}
		if cleanIssues[i].code != cleanIssues[j].code {
			return cleanIssues[i].code < cleanIssues[j].code
		}
		if cleanIssues[i].nodeID != cleanIssues[j].nodeID {
			return cleanIssues[i].nodeID < cleanIssues[j].nodeID
		}
		return cleanIssues[i].relationshipID < cleanIssues[j].relationshipID
	})

	for _, iss := range cleanIssues {
		switch iss.severity {
		case ValMissingNode:
			missing = append(missing, iss)
		case ValInvalidEdge:
			invalid = append(invalid, iss)
		case ValDuplicateEdge:
			dups = append(dups, iss)
		case ValOrphanNode:
			orphans = append(orphans, iss)
		}
	}

	return &ValidationReport{
		issues:         cleanIssues,
		missingNodes:   missing,
		invalidEdges:   invalid,
		duplicateEdges: dups,
		orphanNodes:    orphans,
	}
}

// TotalIssues returns the total count of validation issues.
func (vr *ValidationReport) TotalIssues() int {
	if vr == nil {
		return 0
	}
	return len(vr.issues)
}

// Issues returns a defensive copy of all validation issues.
func (vr *ValidationReport) Issues() []*ValidationIssue {
	if vr == nil || vr.issues == nil {
		return nil
	}
	cloned := make([]*ValidationIssue, len(vr.issues))
	copy(cloned, vr.issues)
	return cloned
}

// MissingNodes returns missing-node issues.
func (vr *ValidationReport) MissingNodes() []*ValidationIssue {
	if vr == nil || vr.missingNodes == nil {
		return nil
	}
	cloned := make([]*ValidationIssue, len(vr.missingNodes))
	copy(cloned, vr.missingNodes)
	return cloned
}

// InvalidEdges returns invalid-edge issues.
func (vr *ValidationReport) InvalidEdges() []*ValidationIssue {
	if vr == nil || vr.invalidEdges == nil {
		return nil
	}
	cloned := make([]*ValidationIssue, len(vr.invalidEdges))
	copy(cloned, vr.invalidEdges)
	return cloned
}

// DuplicateEdges returns duplicate-edge issues.
func (vr *ValidationReport) DuplicateEdges() []*ValidationIssue {
	if vr == nil || vr.duplicateEdges == nil {
		return nil
	}
	cloned := make([]*ValidationIssue, len(vr.duplicateEdges))
	copy(cloned, vr.duplicateEdges)
	return cloned
}

// OrphanNodes returns orphan-node issues.
func (vr *ValidationReport) OrphanNodes() []*ValidationIssue {
	if vr == nil || vr.orphanNodes == nil {
		return nil
	}
	cloned := make([]*ValidationIssue, len(vr.orphanNodes))
	copy(cloned, vr.orphanNodes)
	return cloned
}

// HasErrors returns true if missing nodes or invalid edges exist.
func (vr *ValidationReport) HasErrors() bool {
	if vr == nil {
		return false
	}
	return len(vr.missingNodes) > 0 || len(vr.invalidEdges) > 0
}

// NodeDTO represents a serialized node for JSON/Internal API export.
type NodeDTO struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	Name     string            `json:"name"`
	Path     string            `json:"path,omitempty"`
	Module   string            `json:"module,omitempty"`
	Package  string            `json:"package,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// RelationshipDTO represents a serialized relationship for JSON/Internal API export.
type RelationshipDTO struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	SourceID   string            `json:"source_id"`
	TargetID   string            `json:"target_id"`
	Provenance []string          `json:"provenance,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// InternalGraphModel represents a structured read-oriented representation for Limoxel consumers.
type InternalGraphModel struct {
	SchemaVersion      string             `json:"schema_version"`
	RepositoryRoot     string             `json:"repository_root"`
	TotalNodes         int                `json:"total_nodes"`
	TotalRelationships int                `json:"total_relationships"`
	Nodes              []*NodeDTO         `json:"nodes"`
	Relationships      []*RelationshipDTO `json:"relationships"`
}
