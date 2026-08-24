package graph

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ExportEngine provides deterministic graph serialization into JSON, DOT, GraphML, Mermaid, and Internal API DTOs.
type ExportEngine struct {
	graph *KnowledgeGraph
}

// NewExportEngine constructs an ExportEngine.
func NewExportEngine(graph *KnowledgeGraph) *ExportEngine {
	return &ExportEngine{graph: graph}
}

// ToInternalAPI converts the graph to a structured InternalGraphModel DTO.
func (ee *ExportEngine) ToInternalAPI() *InternalGraphModel {
	if ee == nil || ee.graph == nil {
		return nil
	}

	nodes := ee.graph.AllNodes()
	nodeDTOs := make([]*NodeDTO, len(nodes))
	for i, n := range nodes {
		nodeDTOs[i] = &NodeDTO{
			ID:       n.ID(),
			Type:     string(n.Type()),
			Name:     n.Name(),
			Path:     n.Path(),
			Module:   n.Module(),
			Package:  n.Package(),
			Metadata: n.Metadata(),
		}
	}

	rels := ee.graph.AllRelationships()
	relDTOs := make([]*RelationshipDTO, len(rels))
	for i, r := range rels {
		var provStrs []string
		for _, p := range r.Provenance() {
			provStrs = append(provStrs, string(p))
		}
		relDTOs[i] = &RelationshipDTO{
			ID:         r.ID(),
			Type:       string(r.Type()),
			SourceID:   r.SourceID(),
			TargetID:   r.TargetID(),
			Provenance: provStrs,
			Metadata:   r.Metadata(),
		}
	}

	return &InternalGraphModel{
		SchemaVersion:      ee.graph.SchemaVersion(),
		RepositoryRoot:     ee.graph.RepositoryRoot(),
		TotalNodes:         ee.graph.TotalNodes(),
		TotalRelationships: ee.graph.TotalRelationships(),
		Nodes:              nodeDTOs,
		Relationships:      relDTOs,
	}
}

// ToJSON serializes the knowledge graph into deterministic JSON format.
func (ee *ExportEngine) ToJSON() ([]byte, error) {
	if ee == nil || ee.graph == nil {
		return nil, ErrNilEngine
	}
	dto := ee.ToInternalAPI()
	return json.MarshalIndent(dto, "", "  ")
}

// ToDOT exports the graph into deterministic Graphviz DOT format.
func (ee *ExportEngine) ToDOT() string {
	if ee == nil || ee.graph == nil {
		return "digraph KnowledgeGraph {}\n"
	}

	var b strings.Builder
	b.WriteString("digraph KnowledgeGraph {\n")
	b.WriteString("  rankdir=LR;\n")
	b.WriteString("  node [shape=box, fontname=\"Helvetica\"];\n")
	b.WriteString("  edge [fontname=\"Helvetica\", fontsize=10];\n\n")

	nodes := ee.graph.AllNodes()
	for _, n := range nodes {
		safeID := sanitizeDOTIdentifier(n.ID())
		label := fmt.Sprintf("%s\\n[%s]", escapeDOTString(n.Name()), n.Type())
		shape := "box"
		switch n.Type() {
		case NodeRepository:
			shape = "component"
		case NodeModule:
			shape = "folder"
		case NodePackage:
			shape = "tab"
		case NodeFile:
			shape = "note"
		case NodeSymbol:
			shape = "ellipse"
		case NodeDoc:
			shape = "plaintext"
		case NodeConfig:
			shape = "parallelogram"
		}
		b.WriteString(fmt.Sprintf("  %s [label=\"%s\", shape=%s];\n", safeID, label, shape))
	}

	b.WriteString("\n")

	rels := ee.graph.AllRelationships()
	for _, r := range rels {
		srcID := sanitizeDOTIdentifier(r.SourceID())
		tgtID := sanitizeDOTIdentifier(r.TargetID())
		label := escapeDOTString(string(r.Type()))
		b.WriteString(fmt.Sprintf("  %s -> %s [label=\"%s\"];\n", srcID, tgtID, label))
	}

	b.WriteString("}\n")
	return b.String()
}

// ToGraphML exports the graph into deterministic GraphML XML format.
func (ee *ExportEngine) ToGraphML() string {
	if ee == nil || ee.graph == nil {
		return "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<graphml xmlns=\"http://graphml.graphdrawing.org/xmlns\"/>\n"
	}

	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	b.WriteString("<graphml xmlns=\"http://graphml.graphdrawing.org/xmlns\"\n")
	b.WriteString("         xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\"\n")
	b.WriteString("         xsi:schemaLocation=\"http://graphml.graphdrawing.org/xmlns http://graphml.graphdrawing.org/xmlns/1.0/graphml.xsd\">\n")
	b.WriteString("  <key id=\"d0\" for=\"node\" attr.name=\"type\" attr.type=\"string\"/>\n")
	b.WriteString("  <key id=\"d1\" for=\"node\" attr.name=\"name\" attr.type=\"string\"/>\n")
	b.WriteString("  <key id=\"d2\" for=\"node\" attr.name=\"path\" attr.type=\"string\"/>\n")
	b.WriteString("  <key id=\"d3\" for=\"edge\" attr.name=\"type\" attr.type=\"string\"/>\n")
	b.WriteString("  <graph id=\"G\" edgedefault=\"directed\">\n")

	nodes := ee.graph.AllNodes()
	for _, n := range nodes {
		b.WriteString(fmt.Sprintf("    <node id=\"%s\">\n", escapeXML(n.ID())))
		b.WriteString(fmt.Sprintf("      <data key=\"d0\">%s</data>\n", escapeXML(string(n.Type()))))
		b.WriteString(fmt.Sprintf("      <data key=\"d1\">%s</data>\n", escapeXML(n.Name())))
		if n.Path() != "" {
			b.WriteString(fmt.Sprintf("      <data key=\"d2\">%s</data>\n", escapeXML(n.Path())))
		}
		b.WriteString("    </node>\n")
	}

	rels := ee.graph.AllRelationships()
	for _, r := range rels {
		b.WriteString(fmt.Sprintf("    <edge id=\"%s\" source=\"%s\" target=\"%s\">\n", escapeXML(r.ID()), escapeXML(r.SourceID()), escapeXML(r.TargetID())))
		b.WriteString(fmt.Sprintf("      <data key=\"d3\">%s</data>\n", escapeXML(string(r.Type()))))
		b.WriteString("    </edge>\n")
	}

	b.WriteString("  </graph>\n")
	b.WriteString("</graphml>\n")
	return b.String()
}

// ToMermaid exports the graph into deterministic Mermaid flowchart syntax.
func (ee *ExportEngine) ToMermaid() string {
	if ee == nil || ee.graph == nil {
		return "flowchart TD\n"
	}

	var b strings.Builder
	b.WriteString("flowchart TD\n")

	nodes := ee.graph.AllNodes()
	for _, n := range nodes {
		safeID := sanitizeMermaidID(n.ID())
		label := fmt.Sprintf("%s (%s)", n.Name(), n.Type())
		b.WriteString(fmt.Sprintf("  %s[\"%s\"]\n", safeID, escapeMermaidString(label)))
	}

	rels := ee.graph.AllRelationships()
	for _, r := range rels {
		srcID := sanitizeMermaidID(r.SourceID())
		tgtID := sanitizeMermaidID(r.TargetID())
		b.WriteString(fmt.Sprintf("  %s -->|\"%s\"| %s\n", srcID, r.Type(), tgtID))
	}

	return b.String()
}

func sanitizeDOTIdentifier(id string) string {
	var b strings.Builder
	b.WriteString("node_")
	for _, ch := range id {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
			b.WriteRune(ch)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}

func escapeDOTString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

func sanitizeMermaidID(id string) string {
	var b strings.Builder
	b.WriteString("id_")
	for _, ch := range id {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
			b.WriteRune(ch)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}

func escapeMermaidString(s string) string {
	s = strings.ReplaceAll(s, "\"", "'")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}
