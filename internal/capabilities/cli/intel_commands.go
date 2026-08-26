package cli

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// RegisterIntelligenceCommands registers all intelligence commands on app.
func RegisterIntelligenceCommands(app *App) {
	intelCmd := NewCommand(
		"intel",
		"Access engineering intelligence, architecture explanation, and reasoning",
		"limoxel intel <subcommand> [args] [options]",
		CategoryIntelligence,
		nil,
	).AddAlias("intelligence")

	// 1. intel inspect <symbol>
	inspectCmd := NewCommand(
		"inspect",
		"Inspect detailed semantics, type information, scope, and relationships for a symbol",
		"limoxel intel inspect <symbol-id|name>",
		CategoryIntelligence,
		handleIntelInspect,
	).AddOption("repo", "r", "Target repository directory path", ".")
	intelCmd.AddSubcommand(inspectCmd)

	// 2. intel explain [component]
	explainCmd := NewCommand(
		"explain",
		"Explain repository architecture, layers, boundaries, and components",
		"limoxel intel explain [component]",
		CategoryIntelligence,
		handleIntelExplain,
	).AddOption("repo", "r", "Target repository directory path", ".")
	intelCmd.AddSubcommand(explainCmd)

	// 3. intel dependencies [package]
	depsCmd := NewCommand(
		"dependencies",
		"Analyze package dependencies, fan-in/fan-out, and circular dependency risks",
		"limoxel intel dependencies [package]",
		CategoryIntelligence,
		handleIntelDependencies,
	).AddAlias("deps").AddOption("repo", "r", "Target repository directory path", ".")
	intelCmd.AddSubcommand(depsCmd)

	// 4. intel health [path]
	healthCmd := NewCommand(
		"health",
		"Calculate overall repository health score, quality breakdown, and risk metrics",
		"limoxel intel health [path]",
		CategoryIntelligence,
		handleIntelHealth,
	).AddOption("repo", "r", "Target repository directory path", ".")
	intelCmd.AddSubcommand(healthCmd)

	// 5. intel impact <symbol|package>
	impactCmd := NewCommand(
		"impact",
		"Perform deterministic multi-hop change impact analysis for a symbol or package",
		"limoxel intel impact <symbol-id|package-path>",
		CategoryIntelligence,
		handleIntelImpact,
	).AddOption("depth", "d", "Maximum traversal depth", "10").
		AddOption("repo", "r", "Target repository directory path", ".")
	intelCmd.AddSubcommand(impactCmd)

	// 6. intel navigate <symbol>
	navCmd := NewCommand(
		"navigate",
		"Navigate definitions, inbound callers, references, and symbol hierarchies",
		"limoxel intel navigate <symbol-id|name>",
		CategoryIntelligence,
		handleIntelNavigate,
	).AddAlias("nav").AddOption("repo", "r", "Target repository directory path", ".")
	intelCmd.AddSubcommand(navCmd)

	// 7. intel recommendations [path]
	recCmd := NewCommand(
		"recommendations",
		"Derive prioritized engineering recommendations across architecture, performance, and dependencies",
		"limoxel intel recommendations [path]",
		CategoryIntelligence,
		handleIntelRecommendations,
	).AddAlias("recommend").AddAlias("rec").AddOption("repo", "r", "Target repository directory path", ".")
	intelCmd.AddSubcommand(recCmd)

	app.RegisterCommand(intelCmd)
}

func handleIntelInspect(ctx *Context, flags *Flags) error {
	if flags == nil || flags.NArg() == 0 {
		return UsageError("intel inspect", "symbol name or ID required. Usage: limoxel intel inspect <symbol>")
	}
	targetSymbol := strings.TrimSpace(flags.Arg(0))

	kgModel, err := ctx.EnsureKnowledgeGraph(flags.RepoRoot())
	if err != nil {
		return ExecutionError("intel inspect", "failed to construct knowledge graph", err)
	}

	entity := kgModel.EntityByID(targetSymbol)
	if entity == nil {
		// Search by name
		for _, e := range kgModel.Entities() {
			if e.Name() == targetSymbol {
				entity = e
				break
			}
		}
	}

	if entity == nil {
		return NewCLIError(ErrCatNotFound, fmt.Sprintf("symbol %q not found in knowledge graph", targetSymbol), "intel inspect", nil, ExitFailure)
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
				"type":         string(entity.Type()),
				"name":         entity.Name(),
				"package_path": entity.PackagePath(),
				"file_path":    entity.FilePath(),
				"attributes":   entity.Attributes(),
			},
			"inbound_relationships":  inJSON,
			"outbound_relationships": outJSON,
		})
	}

	ctx.Formatter().RenderSection("Symbol Inspection")
	locStr := entity.FilePath()
	if entity.Position() != nil {
		locStr = fmt.Sprintf("%s:%d", entity.FilePath(), entity.Position().Line())
	}

	if err := ctx.Formatter().RenderKeyValue([][2]string{
		{"ID", entity.ID()},
		{"Type", string(entity.Type())},
		{"Name", entity.Name()},
		{"Package", entity.PackagePath()},
		{"Location", locStr},
		{"Inbound Relationships", strconv.Itoa(len(inRels))},
		{"Outbound Relationships", strconv.Itoa(len(outRels))},
	}); err != nil {
		return err
	}

	if len(inRels) > 0 {
		ctx.Formatter().RenderSection("Inbound Relationships (Callers / Importers)")
		var rows [][]string
		for _, r := range inRels {
			rows = append(rows, []string{r.SourceID(), string(r.Kind()), r.Evidence()})
		}
		_ = ctx.Formatter().RenderTable([]string{"Source Entity", "Relationship", "Evidence"}, rows)
	}

	if len(outRels) > 0 {
		ctx.Formatter().RenderSection("Outbound Relationships (Callees / Dependencies)")
		var rows [][]string
		for _, r := range outRels {
			rows = append(rows, []string{string(r.Kind()), r.TargetID(), r.Evidence()})
		}
		_ = ctx.Formatter().RenderTable([]string{"Relationship", "Target Entity", "Evidence"}, rows)
	}

	return nil
}

func handleIntelExplain(ctx *Context, flags *Flags) error {
	svc, err := ctx.EnsureRepositoryService(flags.RepoRoot())
	if err != nil {
		return ExecutionError("intel explain", "failed to load repository", err)
	}

	structModel := ctx.StructureModel()
	if structModel == nil {
		return ContextError("intel explain", "structure model unavailable")
	}

	meta, err := svc.Metadata()
	if err != nil {
		return ExecutionError("intel explain", "failed to read metadata", err)
	}

	pkgs := structModel.Packages()

	if ctx.Formatter().Format() == FormatJSON {
		pkgPaths := make([]string, len(pkgs))
		for i, p := range pkgs {
			pkgPaths[i] = p.Path()
		}
		return ctx.Formatter().RenderJSON(map[string]any{
			"root":           ctx.RepoPath(),
			"package_count":  len(pkgs),
			"packages":       pkgPaths,
			"structure_type": "Modular Go Architecture",
		})
	}

	ctx.Formatter().RenderSection("Architecture Overview")
	_ = ctx.Formatter().RenderKeyValue([][2]string{
		{"Repository Root", ctx.RepoPath()},
		{"Total Packages", strconv.Itoa(len(pkgs))},
		{"Module Name", meta.Name()},
	})

	ctx.Formatter().RenderSection("Package Hierarchy")
	var rows [][]string
	for i, p := range pkgs {
		rows = append(rows, []string{strconv.Itoa(i + 1), p.Path()})
	}
	return ctx.Formatter().RenderTable([]string{"#", "Package Path"}, rows)
}

func handleIntelDependencies(ctx *Context, flags *Flags) error {
	_, err := ctx.EnsureRepositoryService(flags.RepoRoot())
	if err != nil {
		return ExecutionError("intel dependencies", "failed to load repository", err)
	}

	depModel := ctx.DependencyModel()
	if depModel == nil || depModel.Inventory() == nil {
		return ContextError("intel dependencies", "dependency model unavailable")
	}

	deps := depModel.Inventory().AllDependencies()

	if ctx.Formatter().Format() == FormatJSON {
		type jsonDep struct {
			Name            string `json:"name"`
			DeclaredVersion string `json:"declared_version"`
			IsDirect        bool   `json:"is_direct"`
			Type            string `json:"type"`
		}
		var jList []jsonDep
		for _, d := range deps {
			jList = append(jList, jsonDep{
				Name:            d.Name(),
				DeclaredVersion: d.DeclaredVersion(),
				IsDirect:        d.IsDirect(),
				Type:            string(d.Type()),
			})
		}
		return ctx.Formatter().RenderJSON(jList)
	}

	ctx.Formatter().RenderSection("Dependency Analysis")
	_ = ctx.Formatter().RenderKeyValue([][2]string{
		{"Total Dependencies", strconv.Itoa(len(deps))},
		{"Direct Dependencies", strconv.Itoa(len(depModel.Inventory().DirectDependencies()))},
		{"Transitive Dependencies", strconv.Itoa(len(depModel.Inventory().IndirectDependencies()))},
	})

	if len(deps) > 0 {
		var rows [][]string
		for _, d := range deps {
			directStr := "transitive"
			if d.IsDirect() {
				directStr = "direct"
			}
			rows = append(rows, []string{d.Name(), d.DeclaredVersion(), directStr, string(d.Type())})
		}
		return ctx.Formatter().RenderTable([]string{"Dependency", "Version", "Scope", "Type"}, rows)
	}
	return nil
}

func handleIntelHealth(ctx *Context, flags *Flags) error {
	_, analModel, err := ctx.EnsureAnalysisEngine(flags.RepoRoot())
	if err != nil {
		return ExecutionError("intel health", "failed to calculate health", err)
	}

	healthReport := analModel.HealthReport()

	if ctx.Formatter().Format() == FormatJSON {
		var jsonFindings []map[string]any
		for _, f := range analModel.AllFindings() {
			loc := f.FilePath()
			if f.Location() != nil {
				loc = fmt.Sprintf("%s:%d", f.FilePath(), f.Location().Line())
			}
			jsonFindings = append(jsonFindings, map[string]any{
				"id":               f.ID(),
				"rule_id":          string(f.RuleID()),
				"severity":         string(f.Severity()),
				"category":         string(f.Category()),
				"title":            f.Title(),
				"description":      f.Description(),
				"location":         loc,
				"remediation_hint": f.RemediationHint(),
			})
		}
		var healthMap map[string]any
		if healthReport != nil {
			healthMap = map[string]any{
				"overall_score": healthReport.OverallScore(),
				"grade":         healthReport.Grade(),
				"analyzed_at":   healthReport.AnalyzedAt(),
			}
		}
		return ctx.Formatter().RenderJSON(map[string]any{
			"health":   healthMap,
			"findings": jsonFindings,
		})
	}

	scoreStr := "N/A"
	gradeStr := "N/A"
	if healthReport != nil {
		scoreStr = fmt.Sprintf("%.1f / 100", healthReport.OverallScore())
		gradeStr = healthReport.Grade()
	}

	ctx.Formatter().RenderSection("Repository Health & Quality")
	_ = ctx.Formatter().RenderKeyValue([][2]string{
		{"Overall Health Score", scoreStr},
		{"Health Grade", gradeStr},
		{"Total Findings", strconv.Itoa(len(analModel.AllFindings()))},
	})

	findings := analModel.AllFindings()
	if len(findings) > 0 {
		ctx.Formatter().RenderSection("Active Findings")
		var rows [][]string
		limit := 10
		if len(findings) < limit {
			limit = len(findings)
		}
		for i := 0; i < limit; i++ {
			f := findings[i]
			locStr := f.FilePath()
			if f.Location() != nil {
				locStr = fmt.Sprintf("%s:%d", f.FilePath(), f.Location().Line())
			}
			rows = append(rows, []string{string(f.Severity()), string(f.RuleID()), f.Title(), locStr})
		}
		_ = ctx.Formatter().RenderTable([]string{"Severity", "Rule ID", "Finding", "Location"}, rows)
	}

	return nil
}

func handleIntelImpact(ctx *Context, flags *Flags) error {
	if flags == nil || flags.NArg() == 0 {
		return UsageError("intel impact", "target symbol or package required. Usage: limoxel intel impact <target>")
	}
	targetID := strings.TrimSpace(flags.Arg(0))
	depth := flags.Int("depth", 10)

	reasonEngine, kgModel, err := ctx.EnsureReasoningEngine(flags.RepoRoot())
	if err != nil {
		return ExecutionError("intel impact", "failed to initialize reasoning engine", err)
	}

	impactAnalyzer := reasonEngine.Impact()
	impactAnalyzer.SetMaxDepth(depth)

	result, err := impactAnalyzer.Analyze(kgModel, targetID)
	if err != nil {
		return ExecutionError("intel impact", fmt.Sprintf("impact analysis failed for %q", targetID), err)
	}

	if ctx.Formatter().Format() == FormatJSON {
		return ctx.Formatter().RenderJSON(result)
	}

	ctx.Formatter().RenderSection("Change Impact Analysis")
	_ = ctx.Formatter().RenderKeyValue([][2]string{
		{"Target Entity ID", result.TargetEntityID},
		{"Impact Scope", string(result.Scope)},
		{"Total Affected Count", strconv.Itoa(result.TotalAffectedCount)},
		{"Cross-Module Impact", strconv.FormatBool(result.CrossModuleImpacted)},
		{"Affected Symbols", strconv.Itoa(len(result.AffectedSymbols))},
		{"Affected Packages", strconv.Itoa(len(result.AffectedPackages))},
	})

	if len(result.AffectedSymbols) > 0 {
		ctx.Formatter().RenderSection("Affected Symbols")
		var rows [][]string
		for _, sym := range result.AffectedSymbols {
			directStr := "transitive"
			if sym.Direct {
				directStr = "direct"
			}
			rows = append(rows, []string{sym.Name, sym.FilePath, directStr, strconv.Itoa(sym.Distance), sym.Evidence})
		}
		_ = ctx.Formatter().RenderTable([]string{"Symbol", "File Path", "Impact", "Hops", "Evidence"}, rows)
	}

	return nil
}

func handleIntelNavigate(ctx *Context, flags *Flags) error {
	if flags == nil || flags.NArg() == 0 {
		return UsageError("intel navigate", "target symbol required. Usage: limoxel intel navigate <symbol>")
	}
	target := strings.TrimSpace(flags.Arg(0))

	_, navModel, err := ctx.EnsureNavigationEngine(flags.RepoRoot())
	if err != nil {
		return ExecutionError("intel navigate", "failed to initialize navigation engine", err)
	}

	def := navModel.Definition(target)
	refs := navModel.References(target)
	calls := navModel.CallHierarchyNode(target)

	if ctx.Formatter().Format() == FormatJSON {
		var defMap map[string]any
		if def != nil && def.Target() != nil {
			tgt := def.Target()
			defMap = map[string]any{
				"symbol_id":    tgt.SymbolID(),
				"name":         tgt.Name(),
				"kind":         tgt.Kind(),
				"package_path": tgt.PackagePath(),
				"file_path":    tgt.FilePath(),
			}
		}
		var refList []map[string]any
		if refs != nil {
			for _, r := range refs.References() {
				refList = append(refList, map[string]any{
					"symbol_id":    r.SymbolID(),
					"name":         r.Name(),
					"kind":         r.Kind(),
					"package_path": r.PackagePath(),
					"file_path":    r.FilePath(),
				})
			}
		}
		var callMap map[string]any
		if calls != nil {
			callMap = map[string]any{
				"symbol_id":        calls.SymbolID(),
				"incoming_callers": calls.IncomingCallers(),
				"outgoing_callees": calls.OutgoingCallees(),
			}
		}
		return ctx.Formatter().RenderJSON(map[string]any{
			"target":     target,
			"definition": defMap,
			"references": refList,
			"calls":      callMap,
		})
	}

	ctx.Formatter().RenderSection(fmt.Sprintf("Navigation for %q", target))
	if def != nil && def.Target() != nil {
		tgt := def.Target()
		locStr := tgt.FilePath()
		if tgt.Position() != nil {
			locStr = fmt.Sprintf("%s:%d", filepath.Base(tgt.FilePath()), tgt.Position().Line())
		}
		_ = ctx.Formatter().RenderKeyValue([][2]string{
			{"Symbol ID", tgt.SymbolID()},
			{"Symbol Name", tgt.Name()},
			{"Kind", tgt.Kind()},
			{"Package", tgt.PackagePath()},
			{"Defined At", locStr},
		})
	}

	if refs != nil && len(refs.References()) > 0 {
		ctx.Formatter().RenderSection("Inbound References")
		var rows [][]string
		for _, ref := range refs.References() {
			locStr := ref.FilePath()
			if ref.Position() != nil {
				locStr = fmt.Sprintf("%s:%d", filepath.Base(ref.FilePath()), ref.Position().Line())
			}
			rows = append(rows, []string{
				ref.Name(),
				ref.PackagePath(),
				locStr,
				ref.Kind(),
			})
		}
		_ = ctx.Formatter().RenderTable([]string{"Symbol", "Package", "Location", "Kind"}, rows)
	}

	return nil
}

func handleIntelRecommendations(ctx *Context, flags *Flags) error {
	reasonEngine, kgModel, err := ctx.EnsureReasoningEngine(flags.RepoRoot())
	if err != nil {
		return ExecutionError("intel recommendations", "failed to initialize reasoning engine", err)
	}

	recs := reasonEngine.Recommendations().GenerateRecommendations(kgModel)

	if ctx.Formatter().Format() == FormatJSON {
		return ctx.Formatter().RenderJSON(recs)
	}

	ctx.Formatter().RenderSection("Engineering Recommendations")
	if len(recs) == 0 {
		return ctx.Formatter().RenderSuccess("No actionable architectural or quality defects detected. Excellent code health!")
	}

	var rows [][]string
	for _, r := range recs {
		rows = append(rows, []string{string(r.Priority), string(r.Category), r.Title, r.TargetEntityID, r.RecommendedAction})
	}
	return ctx.Formatter().RenderTable([]string{"Priority", "Category", "Issue", "Target", "Recommended Action"}, rows)
}
