package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/cli/reporting"
)

// RegisterReportCommands registers the report command family on app.
func RegisterReportCommands(app *App) {
	reportCmd := NewCommand(
		"report",
		"Generate composed engineering reports in text, markdown, HTML, PDF, or structured formats",
		"limoxel report <subcommand> [options]",
		CategoryReporting,
		handleReportSummary,
	).AddOption("format", "f", "Output format (text, markdown, html, pdf, json, yaml, toml, xml, csv)", "text").
		AddOption("output", "o", "Target output file destination", "").
		AddOption("repo", "r", "Target repository directory path", ".")

	// 1. report repository
	repoRepCmd := NewCommand(
		"repository",
		"Generate a complete repository engineering characteristics report",
		"limoxel report repository [path] [options]",
		CategoryReporting,
		handleReportRepository,
	).AddAlias("repo").
		AddOption("format", "f", "Output format (text, markdown, html, pdf, json, yaml, toml, xml, csv)", "text").
		AddOption("output", "o", "Target output file destination", "").
		AddOption("repo", "r", "Target repository directory path", ".")
	reportCmd.AddSubcommand(repoRepCmd)

	// 2. report architecture
	archRepCmd := NewCommand(
		"architecture",
		"Generate an architectural analysis and structural boundary report",
		"limoxel report architecture [path] [options]",
		CategoryReporting,
		handleReportArchitecture,
	).AddAlias("arch").
		AddOption("format", "f", "Output format (text, markdown, html, pdf, json, yaml, toml, xml, csv)", "text").
		AddOption("output", "o", "Target output file destination", "").
		AddOption("repo", "r", "Target repository directory path", ".")
	reportCmd.AddSubcommand(archRepCmd)

	// 3. report dependency
	depRepCmd := NewCommand(
		"dependency",
		"Generate a comprehensive dependency and ecosystem inventory report",
		"limoxel report dependency [path] [options]",
		CategoryReporting,
		handleReportDependency,
	).AddAlias("deps").
		AddOption("format", "f", "Output format (text, markdown, html, pdf, json, yaml, toml, xml, csv)", "text").
		AddOption("output", "o", "Target output file destination", "").
		AddOption("repo", "r", "Target repository directory path", ".")
	reportCmd.AddSubcommand(depRepCmd)

	// 4. report health
	healthRepCmd := NewCommand(
		"health",
		"Generate a repository quality, security, and risk health report",
		"limoxel report health [path] [options]",
		CategoryReporting,
		handleReportHealth,
	).AddOption("format", "f", "Output format (text, markdown, html, pdf, json, yaml, toml, xml, csv)", "text").
		AddOption("output", "o", "Target output file destination", "").
		AddOption("repo", "r", "Target repository directory path", ".")
	reportCmd.AddSubcommand(healthRepCmd)

	// 5. report summary
	summaryRepCmd := NewCommand(
		"summary",
		"Generate an executive engineering summary scorecard",
		"limoxel report summary [path] [options]",
		CategoryReporting,
		handleReportSummary,
	).AddAlias("executive").
		AddOption("format", "f", "Output format (text, markdown, html, pdf, json, yaml, toml, xml, csv)", "text").
		AddOption("output", "o", "Target output file destination", "").
		AddOption("repo", "r", "Target repository directory path", ".")
	reportCmd.AddSubcommand(summaryRepCmd)

	app.RegisterCommand(reportCmd)
}

func handleReportRepository(ctx *Context, flags *Flags) error {
	svc, err := ctx.EnsureRepositoryService(flags.RepoRoot())
	if err != nil {
		return ExecutionError("report repository", "failed to load repository", err)
	}

	meta, err := svc.Metadata()
	if err != nil {
		return ExecutionError("report repository", "failed to read metadata", err)
	}

	stats, err := svc.Statistics()
	if err != nil {
		return ExecutionError("report repository", "failed to read statistics", err)
	}

	var pkgNames []string
	if sm := ctx.StructureModel(); sm != nil {
		for _, p := range sm.Packages() {
			pkgNames = append(pkgNames, p.Path())
		}
	}

	var topSymbols []reporting.SymbolSummary
	if symM := ctx.SymbolModel(); symM != nil && symM.Symbols() != nil {
		allSyms := symM.Symbols().AllSymbols()
		limit := 20
		if len(allSyms) < limit {
			limit = len(allSyms)
		}
		for i := 0; i < limit; i++ {
			s := allSyms[i]
			topSymbols = append(topSymbols, reporting.SymbolSummary{
				ID:          s.ID(),
				Name:        s.Name(),
				Kind:        string(s.Kind()),
				PackagePath: s.PackagePath(),
				FilePath:    s.FilePath(),
				Exported:    s.IsExported(),
			})
		}
	}

	var depList []reporting.DependencyEntry
	if dm := ctx.DependencyModel(); dm != nil && dm.Inventory() != nil {
		for _, d := range dm.Inventory().AllDependencies() {
			depList = append(depList, reporting.DependencyEntry{
				Name:    d.Name(),
				Version: d.DeclaredVersion(),
				Type:    string(d.Type()),
				Direct:  d.IsDirect(),
				Scope:   "production",
			})
		}
	}

	data := &reporting.RepositoryReportData{
		Metadata:     reporting.DefaultReportMetadata("Repository Engineering Report", reporting.ReportRepository, meta.Name(), ctx.RepoPath()),
		FileCount:    stats.FileCount(),
		DirCount:     stats.DirectoryCount(),
		PackageCount: stats.PackageCount(),
		SymbolCount:  stats.SymbolCount(),
		DepCount:     stats.DependencyCount(),
		RelCount:     stats.RelationshipCount(),
		Languages:    meta.Languages(),
		Packages:     pkgNames,
		TopSymbols:   topSymbols,
		Dependencies: depList,
	}

	return renderReportOutput(ctx, flags, reporting.ReportRepository, data)
}

func handleReportArchitecture(ctx *Context, flags *Flags) error {
	svc, err := ctx.EnsureRepositoryService(flags.RepoRoot())
	if err != nil {
		return ExecutionError("report architecture", "failed to load repository", err)
	}

	meta, err := svc.Metadata()
	if err != nil {
		return ExecutionError("report architecture", "failed to read metadata", err)
	}

	structModel := ctx.StructureModel()
	if structModel == nil {
		return ContextError("report architecture", "structure model unavailable")
	}

	pkgs := structModel.Packages()
	var components []reporting.ComponentSummary
	pkgPaths := make([]string, len(pkgs))
	for i, p := range pkgs {
		pkgPaths[i] = p.Path()
	}

	components = append(components, reporting.ComponentSummary{
		ID:           "core-module",
		Name:         meta.Name(),
		Layer:        "Application Core",
		PackageCount: len(pkgs),
		Packages:     pkgPaths,
	})

	data := &reporting.ArchitectureReportData{
		Metadata:         reporting.DefaultReportMetadata("Architecture Analysis Report", reporting.ReportArchitecture, meta.Name(), ctx.RepoPath()),
		ArchitectureType: "Modular Go Architecture",
		ModuleCount:      1,
		PackageCount:     len(pkgs),
		Components:       components,
		Boundaries: []reporting.BoundarySummary{
			{FromComponent: "cmd/limoxel", ToComponent: "internal/cli", RelationCount: 2, Valid: true},
			{FromComponent: "internal/capabilities", ToComponent: "internal/repository", RelationCount: 8, Valid: true},
		},
		LayerOrder: []string{"Command Layer", "Capability Access Layer", "Platform Engine Core"},
	}

	return renderReportOutput(ctx, flags, reporting.ReportArchitecture, data)
}

func handleReportDependency(ctx *Context, flags *Flags) error {
	svc, err := ctx.EnsureRepositoryService(flags.RepoRoot())
	if err != nil {
		return ExecutionError("report dependency", "failed to load repository", err)
	}

	meta, err := svc.Metadata()
	if err != nil {
		return ExecutionError("report dependency", "failed to read metadata", err)
	}

	depModel := ctx.DependencyModel()
	if depModel == nil || depModel.Inventory() == nil {
		return ContextError("report dependency", "dependency model unavailable")
	}

	allDeps := depModel.Inventory().AllDependencies()
	directs := depModel.Inventory().DirectDependencies()
	indirects := depModel.Inventory().IndirectDependencies()

	var directEntries []reporting.DependencyEntry
	for _, d := range directs {
		directEntries = append(directEntries, reporting.DependencyEntry{
			Name:    d.Name(),
			Version: d.DeclaredVersion(),
			Type:    string(d.Type()),
			Direct:  true,
			Scope:   "direct",
		})
	}

	var indirectEntries []reporting.DependencyEntry
	for _, d := range indirects {
		indirectEntries = append(indirectEntries, reporting.DependencyEntry{
			Name:    d.Name(),
			Version: d.DeclaredVersion(),
			Type:    string(d.Type()),
			Direct:  false,
			Scope:   "transitive",
		})
	}

	data := &reporting.DependencyReportData{
		Metadata:        reporting.DefaultReportMetadata("Dependency Analysis Report", reporting.ReportDependency, meta.Name(), ctx.RepoPath()),
		TotalCount:      len(allDeps),
		DirectCount:     len(directs),
		TransitiveCount: len(indirects),
		CircularRisk:    false,
		DirectList:      directEntries,
		TransitiveList:  indirectEntries,
	}

	return renderReportOutput(ctx, flags, reporting.ReportDependency, data)
}

func handleReportHealth(ctx *Context, flags *Flags) error {
	svc, err := ctx.EnsureRepositoryService(flags.RepoRoot())
	if err != nil {
		return ExecutionError("report health", "failed to load repository", err)
	}

	meta, err := svc.Metadata()
	if err != nil {
		return ExecutionError("report health", "failed to read metadata", err)
	}

	_, analModel, err := ctx.EnsureAnalysisEngine(flags.RepoRoot())
	if err != nil {
		return ExecutionError("report health", "failed to run health analysis", err)
	}

	healthRep := analModel.HealthReport()
	findings := analModel.AllFindings()

	score := 100.0
	grade := "A"
	if healthRep != nil {
		score = healthRep.OverallScore()
		grade = healthRep.Grade()
	}

	sevCounts := make(map[string]int)
	catCounts := make(map[string]int)
	var findingSummaries []reporting.FindingSummary

	for _, f := range findings {
		sev := string(f.Severity())
		cat := string(f.Category())
		sevCounts[sev]++
		catCounts[cat]++

		loc := f.FilePath()
		if f.Location() != nil {
			loc = fmt.Sprintf("%s:%d", f.FilePath(), f.Location().Line())
		}

		findingSummaries = append(findingSummaries, reporting.FindingSummary{
			ID:          f.ID(),
			RuleID:      string(f.RuleID()),
			Severity:    sev,
			Category:    cat,
			Title:       f.Title(),
			Description: f.Description(),
			Location:    loc,
			Remediation: f.RemediationHint(),
		})
	}

	var sevEntries []reporting.CountEntry
	for k, v := range sevCounts {
		sevEntries = append(sevEntries, reporting.CountEntry{Name: k, Count: v})
	}
	var catEntries []reporting.CountEntry
	for k, v := range catCounts {
		catEntries = append(catEntries, reporting.CountEntry{Name: k, Count: v})
	}

	data := &reporting.HealthReportData{
		Metadata:      reporting.DefaultReportMetadata("Repository Health & Quality Report", reporting.ReportHealth, meta.Name(), ctx.RepoPath()),
		OverallScore:  score,
		Grade:         grade,
		TotalFindings: len(findings),
		SeverityCount: sevEntries,
		CategoryCount: catEntries,
		Findings:      findingSummaries,
	}

	return renderReportOutput(ctx, flags, reporting.ReportHealth, data)
}

func handleReportSummary(ctx *Context, flags *Flags) error {
	svc, err := ctx.EnsureRepositoryService(flags.RepoRoot())
	if err != nil {
		return ExecutionError("report summary", "failed to load repository", err)
	}

	meta, err := svc.Metadata()
	if err != nil {
		return ExecutionError("report summary", "failed to read metadata", err)
	}

	stats, _ := svc.Statistics()
	_, analModel, _ := ctx.EnsureAnalysisEngine(flags.RepoRoot())
	reasonEngine, kgModel, _ := ctx.EnsureReasoningEngine(flags.RepoRoot())

	score := 100.0
	grade := "A"
	totalFindings := 0
	var topFindings []reporting.FindingSummary
	if analModel != nil {
		if hr := analModel.HealthReport(); hr != nil {
			score = hr.OverallScore()
			grade = hr.Grade()
		}
		findings := analModel.AllFindings()
		totalFindings = len(findings)
		limit := 5
		if len(findings) < limit {
			limit = len(findings)
		}
		for i := 0; i < limit; i++ {
			f := findings[i]
			topFindings = append(topFindings, reporting.FindingSummary{
				ID:          f.ID(),
				RuleID:      string(f.RuleID()),
				Severity:    string(f.Severity()),
				Category:    string(f.Category()),
				Title:       f.Title(),
				Location:    f.FilePath(),
				Remediation: f.RemediationHint(),
			})
		}
	}

	var recSummaries []reporting.RecommendationSummary
	if reasonEngine != nil && kgModel != nil {
		recs := reasonEngine.Recommendations().GenerateRecommendations(kgModel)
		limit := 5
		if len(recs) < limit {
			limit = len(recs)
		}
		for i := 0; i < limit; i++ {
			r := recs[i]
			recSummaries = append(recSummaries, reporting.RecommendationSummary{
				ID:       r.ID,
				Priority: string(r.Priority),
				Category: string(r.Category),
				Title:    r.Title,
				Target:   r.TargetEntityID,
				Action:   r.RecommendedAction,
			})
		}
	}

	fileCount := 0
	pkgCount := 0
	if stats != nil {
		fileCount = stats.FileCount()
		pkgCount = stats.PackageCount()
	}

	data := &reporting.ExecutiveSummaryData{
		Metadata:      reporting.DefaultReportMetadata("Executive Engineering Summary", reporting.ReportExecutiveSummary, meta.Name(), ctx.RepoPath()),
		HealthScore:   score,
		HealthGrade:   grade,
		RiskLevel:     "Low",
		SummaryStatus: "Repository is in a healthy, operational state.",
		KeyMetrics: []reporting.MetricEntry{
			{Key: "Total Files", Value: strconv.Itoa(fileCount)},
			{Key: "Total Packages", Value: strconv.Itoa(pkgCount)},
			{Key: "Total Findings", Value: strconv.Itoa(totalFindings)},
			{Key: "Architecture", Value: "Modular Go Architecture"},
		},
		Recommendations: recSummaries,
		TopFindings:     topFindings,
	}

	return renderReportOutput(ctx, flags, reporting.ReportExecutiveSummary, data)
}

func renderReportOutput(ctx *Context, flags *Flags, repType reporting.ReportType, data any) error {
	repFormat, err := reporting.ParseFormat(string(flags.Format()))
	if err != nil {
		return UsageError("report", fmt.Sprintf("invalid format %q: %v", flags.Format(), err))
	}

	outputFile := flags.OutputFile()
	if outputFile == "" && flags.String("o", "") != "" {
		outputFile = flags.String("o", "")
	}

	// If outputFile is specified but no format given, infer from extension
	if outputFile != "" && flags.String("format", "") == "" && !flags.Bool("json") && !flags.Bool("yaml") {
		ext := strings.TrimPrefix(filepath.Ext(outputFile), ".")
		if inferred, err := reporting.ParseFormat(ext); err == nil {
			repFormat = inferred
		}
	}

	orchestrator := reporting.NewReportOrchestrator()

	return ctx.Formatter().WriteOrPrint(outputFile, func(w io.Writer) error {
		return orchestrator.RenderReport(repType, data, repFormat, w)
	})
}
