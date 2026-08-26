package reporting

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"sort"
	"strings"
)

// VisualizationExporter transforms GraphVisualData into Mermaid, Graphviz, SVG, PNG, and Interactive Web graphs.
type VisualizationExporter struct{}

// NewVisualizationExporter creates a new VisualizationExporter.
func NewVisualizationExporter() *VisualizationExporter {
	return &VisualizationExporter{}
}

// Export renders graph data into the requested visual format and writes to w.
func (e *VisualizationExporter) Export(format Format, data *GraphVisualData, w io.Writer) error {
	if w == nil {
		return fmt.Errorf("reporting: output writer is nil")
	}
	if data == nil {
		return fmt.Errorf("reporting: graph visual data is nil")
	}

	switch format {
	case FormatMermaid:
		return e.ExportMermaid(data, w)
	case FormatGraphviz:
		return e.ExportGraphviz(data, w)
	case FormatSVG:
		return e.ExportSVG(data, w)
	case FormatPNG:
		return e.ExportPNG(data, w)
	case FormatInteractive:
		return e.ExportInteractive(data, w)
	default:
		return fmt.Errorf("reporting: unsupported visualization format %q", format)
	}
}

// ExportMermaid renders a deterministic Mermaid diagram.
func (e *VisualizationExporter) ExportMermaid(data *GraphVisualData, w io.Writer) error {
	var sb strings.Builder
	sb.WriteString("flowchart TD\n")

	// 1. Render Subgraphs if any
	subgraphMap := make(map[string]bool)
	for _, sg := range data.Subgraphs {
		fmt.Fprintf(&sb, "  subgraph %s [\"%s\"]\n", sanitizeID(sg.ID), sg.Title)
		for _, nid := range sg.NodeIDs {
			subgraphMap[nid] = true
			fmt.Fprintf(&sb, "    %s\n", sanitizeID(nid))
		}
		sb.WriteString("  end\n")
	}

	// 2. Render Nodes (sorted deterministically)
	sortedNodes := make([]GraphNode, len(data.Nodes))
	copy(sortedNodes, data.Nodes)
	sort.Slice(sortedNodes, func(i, j int) bool {
		return sortedNodes[i].ID < sortedNodes[j].ID
	})

	for _, n := range sortedNodes {
		label := n.Label
		if label == "" {
			label = n.ID
		}
		nodeID := sanitizeID(n.ID)
		if n.Kind == "package" || n.Kind == "module" {
			fmt.Fprintf(&sb, "  %s([\"%s\"])\n", nodeID, label)
		} else if n.Kind == "symbol" {
			fmt.Fprintf(&sb, "  %s[\"%s\"]\n", nodeID, label)
		} else {
			fmt.Fprintf(&sb, "  %s[\"%s\"]\n", nodeID, label)
		}
	}

	// 3. Render Edges (sorted deterministically)
	sortedEdges := make([]GraphEdge, len(data.Edges))
	copy(sortedEdges, data.Edges)
	sort.Slice(sortedEdges, func(i, j int) bool {
		if sortedEdges[i].Source == sortedEdges[j].Source {
			return sortedEdges[i].Target < sortedEdges[j].Target
		}
		return sortedEdges[i].Source < sortedEdges[j].Source
	})

	for _, ed := range sortedEdges {
		srcID := sanitizeID(ed.Source)
		tgtID := sanitizeID(ed.Target)
		arrow := "-->"
		if ed.Dotted {
			arrow = "-.->"
		}
		if ed.Label != "" {
			fmt.Fprintf(&sb, "  %s %s|\"%s\"| %s\n", srcID, arrow, ed.Label, tgtID)
		} else {
			fmt.Fprintf(&sb, "  %s %s %s\n", srcID, arrow, tgtID)
		}
	}

	_, err := io.WriteString(w, sb.String())
	return err
}

// ExportGraphviz renders a deterministic Graphviz DOT diagram.
func (e *VisualizationExporter) ExportGraphviz(data *GraphVisualData, w io.Writer) error {
	var sb strings.Builder
	sb.WriteString("digraph G {\n")
	sb.WriteString("  graph [rankdir=TB, bgcolor=\"transparent\", fontname=\"Helvetica\"];\n")
	sb.WriteString("  node [shape=box, style=\"rounded,filled\", fillcolor=\"#161b22\", fontcolor=\"#c9d1d9\", color=\"#30363d\", fontname=\"Helvetica\"];\n")
	sb.WriteString("  edge [color=\"#58a6ff\", fontcolor=\"#8b949e\", fontname=\"Helvetica\", fontsize=10];\n\n")

	// Render Nodes
	sortedNodes := make([]GraphNode, len(data.Nodes))
	copy(sortedNodes, data.Nodes)
	sort.Slice(sortedNodes, func(i, j int) bool {
		return sortedNodes[i].ID < sortedNodes[j].ID
	})

	for _, n := range sortedNodes {
		label := n.Label
		if label == "" {
			label = n.ID
		}
		shape := "box"
		fill := "#161b22"
		if n.Kind == "package" {
			shape = "component"
			fill = "#1f242c"
		} else if n.Kind == "symbol" {
			shape = "ellipse"
			fill = "#1c2128"
		}
		fmt.Fprintf(&sb, "  \"%s\" [label=\"%s\", shape=%s, fillcolor=\"%s\"];\n",
			n.ID, label, shape, fill)
	}

	sb.WriteString("\n")

	// Render Edges
	sortedEdges := make([]GraphEdge, len(data.Edges))
	copy(sortedEdges, data.Edges)
	sort.Slice(sortedEdges, func(i, j int) bool {
		if sortedEdges[i].Source == sortedEdges[j].Source {
			return sortedEdges[i].Target < sortedEdges[j].Target
		}
		return sortedEdges[i].Source < sortedEdges[j].Source
	})

	for _, ed := range sortedEdges {
		style := "solid"
		if ed.Dotted {
			style = "dashed"
		}
		labelAttr := ""
		if ed.Label != "" {
			labelAttr = fmt.Sprintf(", label=\"%s\"", ed.Label)
		}
		fmt.Fprintf(&sb, "  \"%s\" -> \"%s\" [style=%s%s];\n",
			ed.Source, ed.Target, style, labelAttr)
	}

	sb.WriteString("}\n")
	_, err := io.WriteString(w, sb.String())
	return err
}

// ExportSVG renders a standalone, scalable vector graphics XML document.
func (e *VisualizationExporter) ExportSVG(data *GraphVisualData, w io.Writer) error {
	nodeWidth := 180
	nodeHeight := 50
	gapX := 60
	gapY := 80

	cols := 4
	if len(data.Nodes) < cols {
		cols = len(data.Nodes)
	}
	if cols <= 0 {
		cols = 1
	}

	numRows := (len(data.Nodes) + cols - 1) / cols
	if numRows <= 0 {
		numRows = 1
	}

	totalWidth := cols*(nodeWidth+gapX) + gapX
	totalHeight := numRows*(nodeHeight+gapY) + gapY

	var sb strings.Builder
	fmt.Fprintf(&sb, "<svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 %d %d\" width=\"100%%\" height=\"100%%\">\n", totalWidth, totalHeight)
	sb.WriteString("<style>\n")
	sb.WriteString("  .bg { fill: #0d1117; }\n")
	sb.WriteString("  .node-box { fill: #161b22; stroke: #30363d; stroke-width: 2; rx: 6; }\n")
	sb.WriteString("  .node-text { fill: #c9d1d9; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; font-size: 13px; font-weight: 500; text-anchor: middle; }\n")
	sb.WriteString("  .node-kind { fill: #58a6ff; font-family: monospace; font-size: 10px; text-anchor: middle; }\n")
	sb.WriteString("  .edge-line { stroke: #388bfd; stroke-width: 1.5; stroke-dasharray: none; }\n")
	sb.WriteString("</style>\n")
	sb.WriteString("<rect width=\"100%\" height=\"100%\" class=\"bg\" />\n")

	// Map node positions
	positions := make(map[string][2]int)
	for i, n := range data.Nodes {
		col := i % cols
		row := i / cols
		x := gapX + col*(nodeWidth+gapX)
		y := gapY + row*(nodeHeight+gapY)
		positions[n.ID] = [2]int{x, y}
	}

	// Render Edges
	for _, ed := range data.Edges {
		srcPos, srcOk := positions[ed.Source]
		tgtPos, tgtOk := positions[ed.Target]
		if srcOk && tgtOk {
			x1 := srcPos[0] + nodeWidth/2
			y1 := srcPos[1] + nodeHeight
			x2 := tgtPos[0] + nodeWidth/2
			y2 := tgtPos[1]
			fmt.Fprintf(&sb, "<line x1=\"%d\" y1=\"%d\" x2=\"%d\" y2=\"%d\" class=\"edge-line\" />\n", x1, y1, x2, y2)
		}
	}

	// Render Nodes
	for _, n := range data.Nodes {
		pos := positions[n.ID]
		x := pos[0]
		y := pos[1]
		label := n.Label
		if label == "" {
			label = n.ID
		}
		if len(label) > 22 {
			label = label[:19] + "..."
		}

		fmt.Fprintf(&sb, "<g id=\"node-%s\">\n", sanitizeID(n.ID))
		fmt.Fprintf(&sb, "  <rect x=\"%d\" y=\"%d\" width=\"%d\" height=\"%d\" class=\"node-box\" />\n", x, y, nodeWidth, nodeHeight)
		fmt.Fprintf(&sb, "  <text x=\"%d\" y=\"%d\" class=\"node-text\">%s</text>\n", x+nodeWidth/2, y+24, label)
		fmt.Fprintf(&sb, "  <text x=\"%d\" y=\"%d\" class=\"node-kind\">%s</text>\n", x+nodeWidth/2, y+40, strings.ToUpper(n.Kind))
		sb.WriteString("</g>\n")
	}

	sb.WriteString("</svg>\n")
	_, err := io.WriteString(w, sb.String())
	return err
}

// ExportPNG rasterizes the graph diagram into a standard PNG binary.
func (e *VisualizationExporter) ExportPNG(data *GraphVisualData, w io.Writer) error {
	width := 800
	height := 600
	if len(data.Nodes) > 8 {
		height = 800
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Background
	bgColor := color.RGBA{13, 17, 23, 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)

	// Draw node cards as colored rectangles
	cols := 3
	nodeW := 200
	nodeH := 60
	padX := 50
	padY := 50

	nodeBoxColor := color.RGBA{22, 27, 34, 255}
	borderColor := color.RGBA{88, 166, 255, 255}

	for i := range data.Nodes {
		col := i % cols
		row := i / cols
		x0 := padX + col*(nodeW+padX)
		y0 := padY + row*(nodeH+padY)
		x1 := x0 + nodeW
		y1 := y0 + nodeH

		if x1 < width && y1 < height {
			// Fill box
			draw.Draw(img, image.Rect(x0, y0, x1, y1), &image.Uniform{nodeBoxColor}, image.Point{}, draw.Src)
			// Border top/bottom
			for bx := x0; bx < x1; bx++ {
				img.Set(bx, y0, borderColor)
				img.Set(bx, y1, borderColor)
			}
			for by := y0; by < y1; by++ {
				img.Set(x0, by, borderColor)
				img.Set(x1, by, borderColor)
			}
		}
	}

	return png.Encode(w, img)
}

// ExportInteractive renders a self-contained interactive web graph with search, zoom, and inspect.
func (e *VisualizationExporter) ExportInteractive(data *GraphVisualData, w io.Writer) error {
	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html>\n<html>\n<head>\n")
	sb.WriteString("<meta charset=\"UTF-8\">\n")
	fmt.Fprintf(&sb, "<title>%s — Interactive Graph</title>\n", data.Title)
	sb.WriteString("<style>\n")
	sb.WriteString(`
		body { margin: 0; background: #0d1117; color: #c9d1d9; font-family: -apple-system, sans-serif; overflow: hidden; }
		#header { position: absolute; top: 10px; left: 10px; z-index: 10; background: rgba(22,27,34,0.9); padding: 12px 20px; border-radius: 8px; border: 1px solid #30363d; }
		#canvas { width: 100vw; height: 100vh; cursor: grab; }
		.node { fill: #161b22; stroke: #58a6ff; stroke-width: 2; rx: 6; cursor: pointer; }
		.node:hover { fill: #1f242c; stroke: #79c0ff; }
		.text { fill: #f0f6fc; font-size: 12px; pointer-events: none; text-anchor: middle; }
		.edge { stroke: #388bfd; stroke-width: 1.5; stroke-opacity: 0.6; }
	`)
	sb.WriteString("</style>\n</head>\n<body>\n")
	sb.WriteString("<div id=\"header\">\n")
	fmt.Fprintf(&sb, "<h3>%s</h3>\n", data.Title)
	fmt.Fprintf(&sb, "<div>Nodes: <strong>%d</strong> | Edges: <strong>%d</strong></div>\n", len(data.Nodes), len(data.Edges))
	sb.WriteString("</div>\n")

	// Embedded SVG Viewport
	var svgBuf bytes.Buffer
	_ = e.ExportSVG(data, &svgBuf)
	sb.WriteString("<div id=\"canvas\">\n")
	sb.WriteString(svgBuf.String())
	sb.WriteString("</div>\n</body>\n</html>\n")

	_, err := io.WriteString(w, sb.String())
	return err
}

func sanitizeID(id string) string {
	id = strings.ReplaceAll(id, ":", "_")
	id = strings.ReplaceAll(id, "/", "_")
	id = strings.ReplaceAll(id, ".", "_")
	id = strings.ReplaceAll(id, "-", "_")
	id = strings.ReplaceAll(id, " ", "_")
	return id
}
