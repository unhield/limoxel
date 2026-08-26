package cli

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/repository/query"
)

// RegisterRepositoryCommands registers all repository-level commands on app.
func RegisterRepositoryCommands(app *App) {
	repoCmd := NewCommand(
		"repo",
		"Inspect and manage Limoxel repository workspaces",
		"limoxel repo <subcommand> [options]",
		CategoryRepository,
		nil,
	).AddAlias("repository")

	// 1. repo init
	initCmd := NewCommand(
		"init",
		"Initialize a new repository workspace context",
		"limoxel repo init [path]",
		CategoryRepository,
		handleRepoInit,
	).AddOption("repo", "r", "Target repository directory path", ".")
	repoCmd.AddSubcommand(initCmd)
	app.RegisterCommand(initCmd) // Top-level shortcut

	// 2. repo open
	openCmd := NewCommand(
		"open",
		"Open and load an existing repository workspace",
		"limoxel repo open [path]",
		CategoryRepository,
		handleRepoOpen,
	).AddOption("repo", "r", "Target repository directory path", ".")
	repoCmd.AddSubcommand(openCmd)
	app.RegisterCommand(openCmd) // Top-level shortcut

	// 3. repo scan
	scanCmd := NewCommand(
		"scan",
		"Scan repository filesystem and enumerate files and packages",
		"limoxel repo scan [path]",
		CategoryRepository,
		handleRepoScan,
	).AddOption("repo", "r", "Target repository directory path", ".")
	repoCmd.AddSubcommand(scanCmd)
	app.RegisterCommand(scanCmd) // Top-level shortcut

	// 4. repo analyze
	analyzeCmd := NewCommand(
		"analyze",
		"Perform full repository analysis and index creation",
		"limoxel repo analyze [path]",
		CategoryRepository,
		handleRepoAnalyze,
	).AddOption("repo", "r", "Target repository directory path", ".")
	repoCmd.AddSubcommand(analyzeCmd)
	app.RegisterCommand(analyzeCmd) // Top-level shortcut

	// 5. repo validate
	validateCmd := NewCommand(
		"validate",
		"Validate repository structure, syntax, and consistency",
		"limoxel repo validate [path]",
		CategoryRepository,
		handleRepoValidate,
	).AddOption("repo", "r", "Target repository directory path", ".")
	repoCmd.AddSubcommand(validateCmd)
	app.RegisterCommand(validateCmd) // Top-level shortcut

	// 6. repo reload
	reloadCmd := NewCommand(
		"reload",
		"Invalidate cached indexes and reload repository state",
		"limoxel repo reload [path]",
		CategoryRepository,
		handleRepoReload,
	).AddOption("repo", "r", "Target repository directory path", ".")
	repoCmd.AddSubcommand(reloadCmd)
	app.RegisterCommand(reloadCmd) // Top-level shortcut

	// 7. repo close
	closeCmd := NewCommand(
		"close",
		"Close active repository session and release resources",
		"limoxel repo close",
		CategoryRepository,
		handleRepoClose,
	)
	repoCmd.AddSubcommand(closeCmd)
	app.RegisterCommand(closeCmd) // Top-level shortcut

	// 8. repo info
	infoCmd := NewCommand(
		"info",
		"Display repository metadata, languages, and VCS details",
		"limoxel repo info [path]",
		CategoryRepository,
		handleRepoInfo,
	).AddOption("repo", "r", "Target repository directory path", ".")
	repoCmd.AddSubcommand(infoCmd)
	app.RegisterCommand(infoCmd) // Top-level shortcut

	// 9. repo statistics
	statsCmd := NewCommand(
		"statistics",
		"Display quantitative repository metrics and breakdown",
		"limoxel repo statistics [path]",
		CategoryRepository,
		handleRepoStats,
	).AddAlias("stats").AddOption("repo", "r", "Target repository directory path", ".")
	repoCmd.AddSubcommand(statsCmd)
	app.RegisterCommand(statsCmd) // Top-level shortcut

	app.RegisterCommand(repoCmd)
}

func getTargetRepoPath(ctx *Context, flags *Flags) string {
	if flags != nil && flags.NArg() > 0 {
		return flags.Arg(0)
	}
	if flags != nil {
		return flags.RepoRoot()
	}
	return ctx.RepoPath()
}

func handleRepoInit(ctx *Context, flags *Flags) error {
	path := getTargetRepoPath(ctx, flags)
	ctx.SetRepoPath(path)

	eng, err := ctx.EnsureEngine()
	if err != nil {
		return ExecutionError("repo init", "failed to initialize engine workspace", err)
	}

	ws := eng.Workspace()
	if ws == nil {
		return ExecutionError("repo init", "engine workspace is uninitialized", nil)
	}

	if err := ctx.Formatter().RenderSuccess(fmt.Sprintf("Initialized repository workspace at %q", ws.Root())); err != nil {
		return err
	}
	return nil
}

func handleRepoOpen(ctx *Context, flags *Flags) error {
	path := getTargetRepoPath(ctx, flags)
	svc, err := ctx.EnsureRepositoryService(path)
	if err != nil {
		return ExecutionError("repo open", fmt.Sprintf("failed to open repository at %q", path), err)
	}

	meta, err := svc.Metadata()
	if err != nil {
		return ExecutionError("repo open", "failed to retrieve repository metadata", err)
	}

	if ctx.Formatter().Format() == FormatJSON {
		return ctx.Formatter().RenderJSON(map[string]any{
			"name":           meta.Name(),
			"owner":          meta.Owner(),
			"root":           meta.Root(),
			"default_branch": meta.DefaultBranch(),
			"current_branch": meta.CurrentBranch(),
			"is_git":         meta.IsGit(),
			"total_files":    meta.TotalFiles(),
			"total_dirs":     meta.TotalDirectories(),
			"languages":      meta.Languages(),
			"capabilities":   meta.Capabilities(),
			"analysis_state": meta.AnalysisState(),
			"analyzed_at":    meta.AnalyzedAt(),
		})
	}

	_ = ctx.Formatter().RenderSuccess(fmt.Sprintf("Opened repository: %s (%s)", meta.Name(), meta.Root()))
	return ctx.Formatter().RenderKeyValue([][2]string{
		{"Name", meta.Name()},
		{"Root Path", meta.Root()},
		{"Languages", strings.Join(meta.Languages(), ", ")},
		{"Default Branch", meta.DefaultBranch()},
		{"State", string(svc.LifecycleState())},
	})
}

func handleRepoScan(ctx *Context, flags *Flags) error {
	path := getTargetRepoPath(ctx, flags)
	_, err := ctx.EnsureRepositoryService(path)
	if err != nil {
		return ExecutionError("repo scan", fmt.Sprintf("failed to scan repository at %q", path), err)
	}

	disc := ctx.DiscoveryResult()
	if disc == nil {
		return ContextError("repo scan", "discovery result not available")
	}

	files := disc.Files()

	if ctx.Formatter().Format() == FormatJSON {
		filePaths := make([]string, len(files))
		for i, f := range files {
			filePaths[i] = f.AbsPath()
		}
		return ctx.Formatter().RenderJSON(map[string]any{
			"root":        disc.Root(),
			"files_count": len(files),
			"files":       filePaths,
		})
	}

	_ = ctx.Formatter().RenderSuccess(fmt.Sprintf("Scanned %d files in %q", len(files), disc.Root()))

	// Show sample of files
	limit := 10
	if len(files) < limit {
		limit = len(files)
	}
	var sampleRows [][]string
	for i := 0; i < limit; i++ {
		sampleRows = append(sampleRows, []string{strconv.Itoa(i + 1), filepath.ToSlash(files[i].RelPath())})
	}
	if len(files) > limit {
		sampleRows = append(sampleRows, []string{"...", fmt.Sprintf("(and %d more files)", len(files)-limit)})
	}
	return ctx.Formatter().RenderTable([]string{"#", "Discovered File"}, sampleRows)
}

func handleRepoAnalyze(ctx *Context, flags *Flags) error {
	path := getTargetRepoPath(ctx, flags)
	svc, err := ctx.EnsureRepositoryService(path)
	if err != nil {
		return ExecutionError("repo analyze", fmt.Sprintf("failed to analyze repository at %q", path), err)
	}

	stats, err := svc.Statistics()
	if err != nil {
		return ExecutionError("repo analyze", "failed to calculate repository statistics", err)
	}

	if ctx.Formatter().Format() == FormatJSON {
		return ctx.Formatter().RenderJSON(map[string]any{
			"file_count":         stats.FileCount(),
			"directory_count":    stats.DirectoryCount(),
			"package_count":      stats.PackageCount(),
			"symbol_count":       stats.SymbolCount(),
			"dependency_count":   stats.DependencyCount(),
			"relationship_count": stats.RelationshipCount(),
			"doc_count":          stats.DocCount(),
			"config_count":       stats.ConfigCount(),
			"is_available":       stats.IsAvailable(),
		})
	}

	_ = ctx.Formatter().RenderSuccess(fmt.Sprintf("Repository analysis complete for %q (State: %s)", ctx.RepoPath(), svc.LifecycleState()))
	return ctx.Formatter().RenderKeyValue([][2]string{
		{"Total Files", strconv.Itoa(stats.FileCount())},
		{"Total Packages", strconv.Itoa(stats.PackageCount())},
		{"Total Dependencies", strconv.Itoa(stats.DependencyCount())},
		{"Total Relationships", strconv.Itoa(stats.RelationshipCount())},
		{"Indexed Symbols", strconv.Itoa(stats.SymbolCount())},
	})
}

func handleRepoValidate(ctx *Context, flags *Flags) error {
	path := getTargetRepoPath(ctx, flags)
	svc, err := ctx.EnsureRepositoryService(path)
	if err != nil {
		return ExecutionError("repo validate", fmt.Sprintf("validation failed on %q", path), err)
	}

	state := svc.LifecycleState()
	if state != query.StateReady {
		return ExecutionError("repo validate", fmt.Sprintf("repository is in unexpected state %q", state), nil)
	}

	disc := ctx.DiscoveryResult()
	fileCount := 0
	if disc != nil {
		fileCount = disc.FileCount()
	}

	pkgCount := 0
	if sm := ctx.StructureModel(); sm != nil {
		pkgCount = len(sm.Packages())
	}

	if ctx.Formatter().Format() == FormatJSON {
		return ctx.Formatter().RenderJSON(map[string]any{
			"valid":    true,
			"root":     ctx.RepoPath(),
			"state":    string(state),
			"files":    fileCount,
			"packages": pkgCount,
		})
	}

	_ = ctx.Formatter().RenderSuccess(fmt.Sprintf("Repository validation PASSED for %q", ctx.RepoPath()))
	return ctx.Formatter().RenderKeyValue([][2]string{
		{"Root Path", ctx.RepoPath()},
		{"State", string(state)},
		{"Validated Files", strconv.Itoa(fileCount)},
		{"Validated Packages", strconv.Itoa(pkgCount)},
	})
}

func handleRepoReload(ctx *Context, flags *Flags) error {
	path := getTargetRepoPath(ctx, flags)
	ctx.Reset()

	_, err := ctx.EnsureRepositoryService(path)
	if err != nil {
		return ExecutionError("repo reload", fmt.Sprintf("failed to reload repository at %q", path), err)
	}

	if err := ctx.Formatter().RenderSuccess(fmt.Sprintf("Reloaded repository state at %q", ctx.RepoPath())); err != nil {
		return err
	}
	return nil
}

func handleRepoClose(ctx *Context, flags *Flags) error {
	ctx.Reset()
	return ctx.Formatter().RenderSuccess("Closed active repository session")
}

func handleRepoInfo(ctx *Context, flags *Flags) error {
	path := getTargetRepoPath(ctx, flags)
	svc, err := ctx.EnsureRepositoryService(path)
	if err != nil {
		return ExecutionError("repo info", fmt.Sprintf("failed to read repository info for %q", path), err)
	}

	meta, err := svc.Metadata()
	if err != nil {
		return ExecutionError("repo info", "failed to read metadata", err)
	}

	if ctx.Formatter().Format() == FormatJSON {
		return ctx.Formatter().RenderJSON(map[string]any{
			"name":           meta.Name(),
			"owner":          meta.Owner(),
			"root":           meta.Root(),
			"default_branch": meta.DefaultBranch(),
			"current_branch": meta.CurrentBranch(),
			"is_git":         meta.IsGit(),
			"total_files":    meta.TotalFiles(),
			"total_dirs":     meta.TotalDirectories(),
			"languages":      meta.Languages(),
			"capabilities":   meta.Capabilities(),
			"analysis_state": meta.AnalysisState(),
			"analyzed_at":    meta.AnalyzedAt(),
		})
	}

	ctx.Formatter().RenderSection("Repository Information")
	return ctx.Formatter().RenderKeyValue([][2]string{
		{"Name", meta.Name()},
		{"Root Path", meta.Root()},
		{"All Languages", strings.Join(meta.Languages(), ", ")},
		{"Default Branch", meta.DefaultBranch()},
		{"Current Branch", meta.CurrentBranch()},
		{"State", string(svc.LifecycleState())},
	})
}

func handleRepoStats(ctx *Context, flags *Flags) error {
	path := getTargetRepoPath(ctx, flags)
	svc, err := ctx.EnsureRepositoryService(path)
	if err != nil {
		return ExecutionError("repo statistics", fmt.Sprintf("failed to read statistics for %q", path), err)
	}

	stats, err := svc.Statistics()
	if err != nil {
		return ExecutionError("repo statistics", "failed to read statistics", err)
	}

	if ctx.Formatter().Format() == FormatJSON {
		return ctx.Formatter().RenderJSON(map[string]any{
			"file_count":         stats.FileCount(),
			"directory_count":    stats.DirectoryCount(),
			"package_count":      stats.PackageCount(),
			"symbol_count":       stats.SymbolCount(),
			"dependency_count":   stats.DependencyCount(),
			"relationship_count": stats.RelationshipCount(),
			"doc_count":          stats.DocCount(),
			"config_count":       stats.ConfigCount(),
			"is_available":       stats.IsAvailable(),
		})
	}

	ctx.Formatter().RenderSection("Repository Statistics")
	return ctx.Formatter().RenderKeyValue([][2]string{
		{"Total Files", strconv.Itoa(stats.FileCount())},
		{"Total Directories", strconv.Itoa(stats.DirectoryCount())},
		{"Total Packages", strconv.Itoa(stats.PackageCount())},
		{"Total Dependencies", strconv.Itoa(stats.DependencyCount())},
		{"Total Relationships", strconv.Itoa(stats.RelationshipCount())},
		{"Total Symbols", strconv.Itoa(stats.SymbolCount())},
		{"Total Docs", strconv.Itoa(stats.DocCount())},
		{"Total Configs", strconv.Itoa(stats.ConfigCount())},
	})
}
