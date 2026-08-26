package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/cli/reporting"
	"github.com/unhield/limoxel/internal/capabilities/intelligence/knowledgegraph"
)

// RegisterExportCommands registers the export command family on app.
func RegisterExportCommands(app *App) {
	exportCmd := NewCommand(
		"export",
		"Export repository models, reports, and knowledge graph diagrams to disk or terminal",
		"limoxel export <subcommand> [options]",
		CategoryReporting,
		handleExportGraph,
	).AddOption("format", "f", "Output format (mermaid, graphviz, svg, png, interactive, json, yaml, csv, markdown, html, pdf)", "mermaid").
		AddOption("output", "o", "Target output file destination", "").
		AddOption("repo", "r", "Target repository directory path", ".")

	// 1. export graph
	graphExportCmd := NewCommand(
		"graph",
		"Export knowledge graph entities and relationships in visualization or structured format",
		"limoxel export graph [target-entity] [options]",
		CategoryReporting,
		handleExportGraph,
	).AddOption("format", "f", "Visualization format (mermaid, graphviz, svg, png, interactive, json, yaml)", "mermaid").
		AddOption("depth", "d", "Maximum graph traversal depth", "3").
		AddOption("output", "o", "Target output file destination", "").
		AddOption("repo", "r", "Target repository directory path", ".")
	exportCmd.AddSubcommand(graphExportCmd)

	// 2. export diagram
	diagExportCmd := NewCommand(
		"diagram",
		"Export specialized architectural or relationship diagrams",
		"limoxel export diagram <dependency|call|package|module|symbol|architecture> [target] [options]",
		CategoryReporting,
		handleExportDiagram,
	).AddOption("format", "f", "Diagram format (mermaid, graphviz, svg, png, interactive)", "mermaid").
		AddOption("depth", "d", "Traversal depth", "3").
		AddOption("output", "o", "Target output file destination", "").
		AddOption("repo", "r", "Target repository directory path", ".")
	exportCmd.AddSubcommand(diagExportCmd)

	app.RegisterCommand(exportCmd)
}

func handleExportGraph(ctx *Context, flags *Flags) error {
	kgModel, err := ctx.EnsureKnowledgeGraph(flags.RepoRoot())
	if err != nil {
		return ExecutionError("export graph", "failed to build knowledge graph", err)
	}

	target := ""
	if flags != nil && flags.NArg() > 0 {
		target = strings.TrimSpace(flags.Arg(0))
	}

	depth := flags.Int("depth", 3)
	visualData := buildGraphVisualData(kgModel, target, depth)

	repFormat, err := reporting.ParseFormat(string(flags.Format()))
	if err != nil || repFormat == reporting.FormatText {
		repFormat = reporting.FormatMermaid
	}

	outputFile := flags.OutputFile()
	if outputFile == "" && flags.String("o", "") != "" {
		outputFile = flags.String("o", "")
	}

	if outputFile != "" && flags.String("format", "") == "" && !flags.Bool("json") && !flags.Bool("yaml") {
		ext := strings.TrimPrefix(filepath.Ext(outputFile), ".")
		if inferred, err := reporting.ParseFormat(ext); err == nil {
			repFormat = inferred
		}
	}

	visExporter := reporting.NewVisualizationExporter()
	structExporter := reporting.NewStructuredExporter()

	return ctx.Formatter().WriteOrPrint(outputFile, func(w io.Writer) error {
		switch repFormat {
		case reporting.FormatJSON, reporting.FormatYAML, reporting.FormatTOML, reporting.FormatXML, reporting.FormatCSV:
			return structExporter.Export(repFormat, visualData, w)
		default:
			return visExporter.Export(repFormat, visualData, w)
		}
	})
}

func handleExportDiagram(ctx *Context, flags *Flags) error {
	if flags == nil || flags.NArg() == 0 {
		return UsageError("export diagram", "diagram type required (dependency, call, package, module, symbol, architecture)")
	}

	diagTypeStr := strings.ToLower(flags.Arg(0))
	target := ""
	if flags.NArg() > 1 {
		target = flags.Arg(1)
	}

	kgModel, err := ctx.EnsureKnowledgeGraph(flags.RepoRoot())
	if err != nil {
		return ExecutionError("export diagram", "failed to build knowledge graph", err)
	}

	depth := flags.Int("depth", 3)
	visualData := buildDiagramVisualData(kgModel, diagTypeStr, target, depth)

	repFormat, err := reporting.ParseFormat(string(flags.Format()))
	if err != nil || repFormat == reporting.FormatText {
		repFormat = reporting.FormatMermaid
	}

	outputFile := flags.OutputFile()
	if outputFile == "" && flags.String("o", "") != "" {
		outputFile = flags.String("o", "")
	}

	if outputFile != "" && flags.String("format", "") == "" {
		ext := strings.TrimPrefix(filepath.Ext(outputFile), ".")
		if inferred, err := reporting.ParseFormat(ext); err == nil {
			repFormat = inferred
		}
	}

	visExporter := reporting.NewVisualizationExporter()
	return ctx.Formatter().WriteOrPrint(outputFile, func(w io.Writer) error {
		return visExporter.Export(repFormat, visualData, w)
	})
}

func buildGraphVisualData(kgModel *knowledgegraph.KnowledgeGraphModel, target string, maxDepth int) *reporting.GraphVisualData {
	data := &reporting.GraphVisualData{
		Title:       "Limoxel Knowledge Graph",
		DiagramType: reporting.DiagramArchitecture,
	}

	if kgModel == nil {
		return data
	}

	// Filter entities
	allEntities := kgModel.Entities()
	allRels := kgModel.Relationships()

	limit := 50
	if len(allEntities) < limit {
		limit = len(allEntities)
	}

	for i := 0; i < limit; i++ {
		e := allEntities[i]
		data.Nodes = append(data.Nodes, reporting.GraphNode{
			ID:    e.ID(),
			Label: e.Name(),
			Kind:  string(e.Type()),
			Color: "#58a6ff",
		})
	}

	relLimit := 100
	if len(allRels) < relLimit {
		relLimit = len(allRels)
	}

	for i := 0; i < relLimit; i++ {
		r := allRels[i]
		data.Edges = append(data.Edges, reporting.GraphEdge{
			Source: r.SourceID(),
			Target: r.TargetID(),
			Label:  string(r.Kind()),
			Kind:   string(r.Kind()),
			Dotted: r.Kind() == knowledgegraph.RelDependsOn,
		})
	}

	return data
}

func buildDiagramVisualData(kgModel *knowledgegraph.KnowledgeGraphModel, diagType, target string, maxDepth int) *reporting.GraphVisualData {
	data := &reporting.GraphVisualData{
		Title:       fmt.Sprintf("%s Diagram", strings.Title(diagType)),
		DiagramType: reporting.DiagramType(diagType),
	}

	if kgModel == nil {
		return data
	}

	switch diagType {
	case "dependency", "deps":
		for _, e := range kgModel.EntitiesByType(knowledgegraph.EntityPackage) {
			data.Nodes = append(data.Nodes, reporting.GraphNode{
				ID:    e.ID(),
				Label: e.Name(),
				Kind:  "package",
			})
		}
		for _, r := range kgModel.Relationships() {
			if r.Kind() == knowledgegraph.RelDependsOn || r.Kind() == knowledgegraph.RelImports {
				data.Edges = append(data.Edges, reporting.GraphEdge{
					Source: r.SourceID(),
					Target: r.TargetID(),
					Label:  string(r.Kind()),
					Kind:   string(r.Kind()),
				})
			}
		}

	case "call", "calls":
		for _, e := range kgModel.EntitiesByType(knowledgegraph.EntitySymbol) {
			data.Nodes = append(data.Nodes, reporting.GraphNode{
				ID:    e.ID(),
				Label: e.Name(),
				Kind:  "symbol",
			})
		}
		for _, r := range kgModel.Relationships() {
			if r.Kind() == knowledgegraph.RelCalls {
				data.Edges = append(data.Edges, reporting.GraphEdge{
					Source: r.SourceID(),
					Target: r.TargetID(),
					Label:  "calls",
					Kind:   "calls",
				})
			}
		}

	default:
		return buildGraphVisualData(kgModel, target, maxDepth)
	}

	return data
}
