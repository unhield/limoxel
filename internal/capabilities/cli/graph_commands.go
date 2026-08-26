package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/intelligence/knowledgegraph"
)

// RegisterGraphCommands registers all knowledge graph commands on app.
func RegisterGraphCommands(app *App) {
	graphCmd := NewCommand(
		"graph",
		"Query and inspect the Limoxel Engineering Knowledge Graph",
		"limoxel graph <subcommand> [args] [options]",
		CategoryGraph,
		handleGraphRepo,
	).AddAlias("kg").AddOption("repo", "r", "Target repository directory path", ".")

	// 1. graph repo [path]
	repoGraphCmd := NewCommand(
		"repo",
		"Display repository-level knowledge graph entity metrics and relations",
		"limoxel graph repo [path]",
		CategoryGraph,
		handleGraphRepo,
	).AddAlias("repository")
	graphCmd.AddSubcommand(repoGraphCmd)

	// 2. graph package [pkg]
	pkgGraphCmd := NewCommand(
		"package",
		"Query package nodes and inter-package dependency relationships",
		"limoxel graph package [package-name]",
		CategoryGraph,
		handleGraphPackage,
	).AddAlias("pkg")
	graphCmd.AddSubcommand(pkgGraphCmd)

	// 3. graph dependency [dep]
	depGraphCmd := NewCommand(
		"dependency",
		"Query external and internal dependency graph edges",
		"limoxel graph dependency [dependency-name]",
		CategoryGraph,
		handleGraphDependency,
	).AddAlias("dep")
	graphCmd.AddSubcommand(depGraphCmd)

	// 4. graph call <symbol>
	callGraphCmd := NewCommand(
		"call",
		"Query call graph relationships (callers and callees) for a symbol",
		"limoxel graph call <symbol-id|name>",
		CategoryGraph,
		handleGraphCall,
	)
	graphCmd.AddSubcommand(callGraphCmd)

	// 5. graph module [module]
	modGraphCmd := NewCommand(
		"module",
		"Query module-level graph entities and containment relationships",
		"limoxel graph module [module-name]",
		CategoryGraph,
		handleGraphModule,
	).AddAlias("mod")
	graphCmd.AddSubcommand(modGraphCmd)

	// 6. graph symbol <symbol>
	symGraphCmd := NewCommand(
		"symbol",
		"Query symbol graph relationships (definitions, references, implementations)",
		"limoxel graph symbol <symbol-id|name>",
		CategoryGraph,
		handleGraphSymbol,
	).AddAlias("sym")
	graphCmd.AddSubcommand(symGraphCmd)

	app.RegisterCommand(graphCmd)
}

func handleGraphRepo(ctx *Context, flags *Flags) error {
	kgModel, err := ctx.EnsureKnowledgeGraph(flags.RepoRoot())
	if err != nil {
		return ExecutionError("graph repo", "failed to construct knowledge graph", err)
	}

	entities := kgModel.Entities()
	rels := kgModel.Relationships()
	insights := kgModel.Insights()

	if ctx.Formatter().Format() == FormatJSON {
		return ctx.Formatter().RenderJSON(map[string]any{
			"root_path":          kgModel.RootPath(),
			"entities_count":     len(entities),
			"relationship_count": len(rels),
			"insights_count":     len(insights),
		})
	}

	ctx.Formatter().RenderSection("Engineering Knowledge Graph")
	_ = ctx.Formatter().RenderKeyValue([][2]string{
		{"Root Path", kgModel.RootPath()},
		{"Total Graph Entities", strconv.Itoa(len(entities))},
		{"Total Relationships", strconv.Itoa(len(rels))},
		{"Derived Insights", strconv.Itoa(len(insights))},
	})

	// Breakdown of entities by type
	typeCounts := make(map[knowledgegraph.EntityType]int)
	for _, e := range entities {
		typeCounts[e.Type()]++
	}

	ctx.Formatter().RenderSection("Entity Breakdown")
	var eRows [][]string
	for eType, count := range typeCounts {
		eRows = append(eRows, []string{string(eType), strconv.Itoa(count)})
	}
	_ = ctx.Formatter().RenderTable([]string{"Entity Type", "Count"}, eRows)

	// Breakdown of relationships by kind
	relCounts := make(map[knowledgegraph.RelationshipKind]int)
	for _, r := range rels {
		relCounts[r.Kind()]++
	}

	ctx.Formatter().RenderSection("Relationship Breakdown")
	var rRows [][]string
	for rKind, count := range relCounts {
		rRows = append(rRows, []string{string(rKind), strconv.Itoa(count)})
	}
	return ctx.Formatter().RenderTable([]string{"Relationship Kind", "Count"}, rRows)
}

func handleGraphPackage(ctx *Context, flags *Flags) error {
	kgModel, err := ctx.EnsureKnowledgeGraph(flags.RepoRoot())
	if err != nil {
		return ExecutionError("graph package", "failed to construct knowledge graph", err)
	}

	pkgFilter := ""
	if flags != nil && flags.NArg() > 0 {
		pkgFilter = strings.ToLower(flags.Arg(0))
	}

	var pkgEntities []*knowledgegraph.GraphEntity
	for _, e := range kgModel.EntitiesByType(knowledgegraph.EntityPackage) {
		if pkgFilter == "" || strings.Contains(strings.ToLower(e.Name()), pkgFilter) || strings.Contains(strings.ToLower(e.PackagePath()), pkgFilter) {
			pkgEntities = append(pkgEntities, e)
		}
	}

	if ctx.Formatter().Format() == FormatJSON {
		var jList []map[string]any
		for _, p := range pkgEntities {
			jList = append(jList, map[string]any{
				"id":           p.ID(),
				"name":         p.Name(),
				"package_path": p.PackagePath(),
			})
		}
		return ctx.Formatter().RenderJSON(jList)
	}

	ctx.Formatter().RenderSection("Package Graph Nodes")
	var rows [][]string
	for _, p := range pkgEntities {
		inRels := len(kgModel.InboundRelationships(p.ID()))
		outRels := len(kgModel.OutboundRelationships(p.ID()))
		rows = append(rows, []string{p.ID(), p.Name(), p.PackagePath(), strconv.Itoa(inRels), strconv.Itoa(outRels)})
	}
	return ctx.Formatter().RenderTable([]string{"Package ID", "Name", "Path", "Inbound Rels", "Outbound Rels"}, rows)
}

func handleGraphDependency(ctx *Context, flags *Flags) error {
	kgModel, err := ctx.EnsureKnowledgeGraph(flags.RepoRoot())
	if err != nil {
		return ExecutionError("graph dependency", "failed to construct knowledge graph", err)
	}

	depFilter := ""
	if flags != nil && flags.NArg() > 0 {
		depFilter = strings.ToLower(flags.Arg(0))
	}

	var depRels []*knowledgegraph.GraphRelationship
	for _, r := range kgModel.Relationships() {
		if r.Kind() == knowledgegraph.RelDependsOn || r.Kind() == knowledgegraph.RelImports {
			if depFilter == "" || strings.Contains(strings.ToLower(r.SourceID()), depFilter) || strings.Contains(strings.ToLower(r.TargetID()), depFilter) {
				depRels = append(depRels, r)
			}
		}
	}

	if ctx.Formatter().Format() == FormatJSON {
		var jList []map[string]any
		for _, r := range depRels {
			jList = append(jList, map[string]any{
				"source_id": r.SourceID(),
				"kind":      string(r.Kind()),
				"target_id": r.TargetID(),
				"evidence":  r.Evidence(),
			})
		}
		return ctx.Formatter().RenderJSON(jList)
	}

	ctx.Formatter().RenderSection("Dependency Graph Edges")
	var rows [][]string
	for _, r := range depRels {
		rows = append(rows, []string{r.SourceID(), string(r.Kind()), r.TargetID(), r.Evidence()})
	}
	return ctx.Formatter().RenderTable([]string{"Source", "Kind", "Target", "Evidence"}, rows)
}

func handleGraphCall(ctx *Context, flags *Flags) error {
	if flags == nil || flags.NArg() == 0 {
		return UsageError("graph call", "target symbol ID or name required. Usage: limoxel graph call <symbol>")
	}
	target := strings.TrimSpace(flags.Arg(0))

	kgModel, err := ctx.EnsureKnowledgeGraph(flags.RepoRoot())
	if err != nil {
		return ExecutionError("graph call", "failed to construct knowledge graph", err)
	}

	entity := kgModel.EntityByID(target)
	if entity == nil {
		for _, e := range kgModel.EntitiesByType(knowledgegraph.EntitySymbol) {
			if e.Name() == target {
				entity = e
				break
			}
		}
	}

	if entity == nil {
		return NewCLIError(ErrCatNotFound, fmt.Sprintf("symbol %q not found in graph", target), "graph call", nil, ExitFailure)
	}

	// Gather callers (inbound RelCalls)
	var callers []string
	for _, in := range kgModel.InboundRelationships(entity.ID()) {
		if in.Kind() == knowledgegraph.RelCalls {
			callers = append(callers, in.SourceID())
		}
	}

	// Gather callees (outbound RelCalls)
	var callees []string
	for _, out := range kgModel.OutboundRelationships(entity.ID()) {
		if out.Kind() == knowledgegraph.RelCalls {
			callees = append(callees, out.TargetID())
		}
	}

	if ctx.Formatter().Format() == FormatJSON {
		return ctx.Formatter().RenderJSON(map[string]any{
			"symbol_id":   entity.ID(),
			"symbol_name": entity.Name(),
			"callers":     callers,
			"callees":     callees,
		})
	}

	ctx.Formatter().RenderSection(fmt.Sprintf("Call Graph for %q", entity.Name()))
	_ = ctx.Formatter().RenderKeyValue([][2]string{
		{"Symbol ID", entity.ID()},
		{"Symbol Name", entity.Name()},
		{"Inbound Callers", strconv.Itoa(len(callers))},
		{"Outbound Callees", strconv.Itoa(len(callees))},
	})

	if len(callers) > 0 {
		ctx.Formatter().RenderSection("Inbound Callers")
		var rows [][]string
		for i, c := range callers {
			rows = append(rows, []string{strconv.Itoa(i + 1), c})
		}
		_ = ctx.Formatter().RenderTable([]string{"#", "Caller Symbol"}, rows)
	}

	if len(callees) > 0 {
		ctx.Formatter().RenderSection("Outbound Callees")
		var rows [][]string
		for i, c := range callees {
			rows = append(rows, []string{strconv.Itoa(i + 1), c})
		}
		_ = ctx.Formatter().RenderTable([]string{"#", "Callee Symbol"}, rows)
	}

	return nil
}

func handleGraphModule(ctx *Context, flags *Flags) error {
	kgModel, err := ctx.EnsureKnowledgeGraph(flags.RepoRoot())
	if err != nil {
		return ExecutionError("graph module", "failed to construct knowledge graph", err)
	}

	modEntities := kgModel.EntitiesByType(knowledgegraph.EntityModule)

	if ctx.Formatter().Format() == FormatJSON {
		var jList []map[string]any
		for _, m := range modEntities {
			jList = append(jList, map[string]any{
				"id":   m.ID(),
				"name": m.Name(),
			})
		}
		return ctx.Formatter().RenderJSON(jList)
	}

	ctx.Formatter().RenderSection("Module Graph Nodes")
	var rows [][]string
	for _, m := range modEntities {
		outRels := len(kgModel.OutboundRelationships(m.ID()))
		rows = append(rows, []string{m.ID(), m.Name(), strconv.Itoa(outRels)})
	}
	return ctx.Formatter().RenderTable([]string{"Module ID", "Name", "Contained Entities"}, rows)
}

func handleGraphSymbol(ctx *Context, flags *Flags) error {
	if flags == nil || flags.NArg() == 0 {
		return UsageError("graph symbol", "symbol ID or name required. Usage: limoxel graph symbol <symbol>")
	}
	target := strings.TrimSpace(flags.Arg(0))

	kgModel, err := ctx.EnsureKnowledgeGraph(flags.RepoRoot())
	if err != nil {
		return ExecutionError("graph symbol", "failed to construct knowledge graph", err)
	}

	entity := kgModel.EntityByID(target)
	if entity == nil {
		for _, e := range kgModel.EntitiesByType(knowledgegraph.EntitySymbol) {
			if e.Name() == target {
				entity = e
				break
			}
		}
	}

	if entity == nil {
		return NewCLIError(ErrCatNotFound, fmt.Sprintf("symbol %q not found in graph", target), "graph symbol", nil, ExitFailure)
	}

	inRels := kgModel.InboundRelationships(entity.ID())
	outRels := kgModel.OutboundRelationships(entity.ID())

	if ctx.Formatter().Format() == FormatJSON {
		type jsonRel struct {
			SourceID string `json:"source_id"`
			TargetID string `json:"target_id"`
			Kind     string `json:"kind"`
			Evidence string `json:"evidence"`
		}
		var inJSON []jsonRel
		for _, r := range inRels {
			inJSON = append(inJSON, jsonRel{
				SourceID: r.SourceID(),
				TargetID: r.TargetID(),
				Kind:     string(r.Kind()),
				Evidence: r.Evidence(),
			})
		}
		var outJSON []jsonRel
		for _, r := range outRels {
			outJSON = append(outJSON, jsonRel{
				SourceID: r.SourceID(),
				TargetID: r.TargetID(),
				Kind:     string(r.Kind()),
				Evidence: r.Evidence(),
			})
		}
		return ctx.Formatter().RenderJSON(map[string]any{
			"entity": map[string]any{
				"id":           entity.ID(),
				"name":         entity.Name(),
				"package_path": entity.PackagePath(),
			},
			"inbound_relationships":  inJSON,
			"outbound_relationships": outJSON,
		})
	}

	ctx.Formatter().RenderSection(fmt.Sprintf("Symbol Graph: %s", entity.Name()))
	_ = ctx.Formatter().RenderKeyValue([][2]string{
		{"ID", entity.ID()},
		{"Name", entity.Name()},
		{"Package", entity.PackagePath()},
		{"Inbound Edges", strconv.Itoa(len(inRels))},
		{"Outbound Edges", strconv.Itoa(len(outRels))},
	})

	var allEdges [][]string
	for _, r := range inRels {
		allEdges = append(allEdges, []string{"INBOUND", r.SourceID(), string(r.Kind()), entity.ID(), r.Evidence()})
	}
	for _, r := range outRels {
		allEdges = append(allEdges, []string{"OUTBOUND", entity.ID(), string(r.Kind()), r.TargetID(), r.Evidence()})
	}

	if len(allEdges) > 0 {
		ctx.Formatter().RenderSection("Graph Relationships")
		return ctx.Formatter().RenderTable([]string{"Direction", "Source", "Kind", "Target", "Evidence"}, allEdges)
	}
	return nil
}
