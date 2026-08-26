package cli

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/repository/query"
)

// RegisterSearchCommands registers all search commands on app.
func RegisterSearchCommands(app *App) {
	searchCmd := NewCommand(
		"search",
		"Perform repository-aware engineering search across entities and files",
		"limoxel search <subcommand|query> [options]",
		CategorySearch,
		handleSearchUnified,
	).AddOption("repo", "r", "Target repository directory path", ".").
		AddOption("limit", "l", "Maximum number of search results to return", "20")

	// 1. search symbol <query>
	symSearchCmd := NewCommand(
		"symbol",
		"Search symbols by identifier name, kind, or signature",
		"limoxel search symbol <query>",
		CategorySearch,
		handleSearchDomain(query.DomainSymbol),
	).AddOption("limit", "l", "Maximum number of search results to return", "20").
		AddOption("repo", "r", "Target repository directory path", ".")
	searchCmd.AddSubcommand(symSearchCmd)

	// 2. search package <query>
	pkgSearchCmd := NewCommand(
		"package",
		"Search packages by name or path",
		"limoxel search package <query>",
		CategorySearch,
		handleSearchDomain(query.DomainPackage),
	).AddAlias("pkg").AddOption("limit", "l", "Maximum number of search results to return", "20").
		AddOption("repo", "r", "Target repository directory path", ".")
	searchCmd.AddSubcommand(pkgSearchCmd)

	// 3. search module <query>
	modSearchCmd := NewCommand(
		"module",
		"Search modules defined across the repository",
		"limoxel search module <query>",
		CategorySearch,
		handleSearchModule,
	).AddAlias("mod").AddOption("limit", "l", "Maximum number of search results to return", "20").
		AddOption("repo", "r", "Target repository directory path", ".")
	searchCmd.AddSubcommand(modSearchCmd)

	// 4. search file <query>
	fileSearchCmd := NewCommand(
		"file",
		"Search files by name, path pattern, or extension",
		"limoxel search file <query>",
		CategorySearch,
		handleSearchDomain(query.DomainFile),
	).AddOption("limit", "l", "Maximum number of search results to return", "20").
		AddOption("repo", "r", "Target repository directory path", ".")
	searchCmd.AddSubcommand(fileSearchCmd)

	// 5. search dependency <query>
	depSearchCmd := NewCommand(
		"dependency",
		"Search external and internal dependencies",
		"limoxel search dependency <query>",
		CategorySearch,
		handleSearchDependency,
	).AddAlias("dep").AddOption("limit", "l", "Maximum number of search results to return", "20").
		AddOption("repo", "r", "Target repository directory path", ".")
	searchCmd.AddSubcommand(depSearchCmd)

	// 6. search doc <query>
	docSearchCmd := NewCommand(
		"doc",
		"Search documentation entries, comments, and docstrings",
		"limoxel search doc <query>",
		CategorySearch,
		handleSearchDomain(query.DomainDocumentation),
	).AddAlias("documentation").AddOption("limit", "l", "Maximum number of search results to return", "20").
		AddOption("repo", "r", "Target repository directory path", ".")
	searchCmd.AddSubcommand(docSearchCmd)

	// 7. search config <query>
	configSearchCmd := NewCommand(
		"config",
		"Search configuration files, settings, and entries",
		"limoxel search config <query>",
		CategorySearch,
		handleSearchDomain(query.DomainConfiguration),
	).AddAlias("configuration").AddOption("limit", "l", "Maximum number of search results to return", "20").
		AddOption("repo", "r", "Target repository directory path", ".")
	searchCmd.AddSubcommand(configSearchCmd)

	app.RegisterCommand(searchCmd)
}

type searchItemJSON struct {
	EntityID    string  `json:"entity_id"`
	Domain      string  `json:"domain"`
	Name        string  `json:"name"`
	Path        string  `json:"path"`
	PackageName string  `json:"package_name"`
	Scope       string  `json:"scope"`
	Score       float64 `json:"score"`
	Snippet     string  `json:"snippet"`
}

type searchResultJSON struct {
	Query             string           `json:"query"`
	Domain            string           `json:"domain"`
	TotalMatches      int              `json:"total_matches"`
	ExecutionDuration string           `json:"execution_duration"`
	Items             []searchItemJSON `json:"items"`
}

func toSearchResultJSON(result *query.SearchResultDTO) *searchResultJSON {
	if result == nil {
		return nil
	}
	items := make([]searchItemJSON, len(result.Items()))
	for i, it := range result.Items() {
		items[i] = searchItemJSON{
			EntityID:    it.EntityID(),
			Domain:      string(it.Domain()),
			Name:        it.Name(),
			Path:        it.Path(),
			PackageName: it.PackageName(),
			Scope:       it.Scope(),
			Score:       it.Score(),
			Snippet:     it.Snippet(),
		}
	}
	return &searchResultJSON{
		Query:             result.Query(),
		Domain:            string(result.Domain()),
		TotalMatches:      result.TotalMatches(),
		ExecutionDuration: result.ExecutionDuration().String(),
		Items:             items,
	}
}

func handleSearchUnified(ctx *Context, flags *Flags) error {
	if flags == nil || flags.NArg() == 0 {
		return UsageError("search", "search query required. Usage: limoxel search <query> or limoxel search <subcommand> <query>")
	}

	queryStr := strings.TrimSpace(flags.Arg(0))
	if queryStr == "" {
		return UsageError("search", "search query cannot be empty")
	}

	repoPath := flags.RepoRoot()
	svc, err := ctx.EnsureRepositoryService(repoPath)
	if err != nil {
		return ExecutionError("search", "failed to load repository", err)
	}

	limit := flags.Int("limit", 20)
	opts := query.SearchOptions{
		MaxResults:    limit,
		ExactMatch:    false,
		CaseSensitive: false,
		MinScore:      0.0,
	}

	result, err := svc.Search().Search(queryStr, query.DomainAll, opts)
	if err != nil {
		return ExecutionError("search", fmt.Sprintf("search failed for query %q", queryStr), err)
	}

	if ctx.Formatter().Format() == FormatJSON {
		return ctx.Formatter().RenderJSON(toSearchResultJSON(result))
	}

	if result.TotalMatches() == 0 {
		return ctx.Formatter().RenderInfo(fmt.Sprintf("No results found matching %q", queryStr))
	}

	_ = ctx.Formatter().RenderSuccess(fmt.Sprintf("Found %d results for query %q (in %v)", result.TotalMatches(), queryStr, result.ExecutionDuration()))

	var rows [][]string
	for i, item := range result.Items() {
		rows = append(rows, []string{
			strconv.Itoa(i + 1),
			string(item.Domain()),
			item.Name(),
			item.PackageName(),
			filepath.ToSlash(item.Path()),
			fmt.Sprintf("%.2f", item.Score()),
		})
	}
	return ctx.Formatter().RenderTable([]string{"#", "Domain", "Name", "Package", "Location", "Score"}, rows)
}

func handleSearchDomain(domain query.SearchDomain) HandlerFunc {
	return func(ctx *Context, flags *Flags) error {
		if flags == nil || flags.NArg() == 0 {
			return UsageError(fmt.Sprintf("search %s", domain), fmt.Sprintf("query string required. Usage: limoxel search %s <query>", domain))
		}
		queryStr := strings.TrimSpace(flags.Arg(0))
		if queryStr == "" {
			return UsageError(fmt.Sprintf("search %s", domain), "query string cannot be empty")
		}

		limit := flags.Int("limit", 20)
		repoPath := flags.RepoRoot()

		svc, err := ctx.EnsureRepositoryService(repoPath)
		if err != nil {
			return ExecutionError(fmt.Sprintf("search %s", domain), "failed to load repository", err)
		}

		opts := query.SearchOptions{
			MaxResults:    limit,
			ExactMatch:    false,
			CaseSensitive: false,
			MinScore:      0.0,
		}

		result, err := svc.Search().Search(queryStr, domain, opts)
		if err != nil {
			return ExecutionError(fmt.Sprintf("search %s", domain), fmt.Sprintf("search failed for query %q", queryStr), err)
		}

		if ctx.Formatter().Format() == FormatJSON {
			return ctx.Formatter().RenderJSON(toSearchResultJSON(result))
		}

		if result.TotalMatches() == 0 {
			return ctx.Formatter().RenderInfo(fmt.Sprintf("No %s matches found for %q", domain, queryStr))
		}

		_ = ctx.Formatter().RenderSuccess(fmt.Sprintf("Found %d %s matches for %q (in %v)", result.TotalMatches(), domain, queryStr, result.ExecutionDuration()))

		var rows [][]string
		for i, item := range result.Items() {
			rows = append(rows, []string{
				strconv.Itoa(i + 1),
				item.Name(),
				item.PackageName(),
				filepath.ToSlash(item.Path()),
				item.Snippet(),
			})
		}
		return ctx.Formatter().RenderTable([]string{"#", "Name", "Package", "Path", "Snippet"}, rows)
	}
}

func handleSearchModule(ctx *Context, flags *Flags) error {
	if flags == nil || flags.NArg() == 0 {
		return UsageError("search module", "query string required. Usage: limoxel search module <query>")
	}
	queryStr := strings.ToLower(flags.Arg(0))

	svc, err := ctx.EnsureRepositoryService(flags.RepoRoot())
	if err != nil {
		return ExecutionError("search module", "failed to load repository", err)
	}

	meta, err := svc.Metadata()
	if err != nil {
		return ExecutionError("search module", "failed to read metadata", err)
	}

	var matchedModules []string
	if strings.Contains(strings.ToLower(meta.Name()), queryStr) {
		matchedModules = append(matchedModules, meta.Name())
	}

	if ctx.Formatter().Format() == FormatJSON {
		return ctx.Formatter().RenderJSON(matchedModules)
	}

	if len(matchedModules) == 0 {
		return ctx.Formatter().RenderInfo(fmt.Sprintf("No modules found matching %q", queryStr))
	}

	_ = ctx.Formatter().RenderSuccess(fmt.Sprintf("Found %d modules matching %q", len(matchedModules), queryStr))
	var rows [][]string
	for i, m := range matchedModules {
		rows = append(rows, []string{strconv.Itoa(i + 1), m})
	}
	return ctx.Formatter().RenderTable([]string{"#", "Module Name"}, rows)
}

func handleSearchDependency(ctx *Context, flags *Flags) error {
	if flags == nil || flags.NArg() == 0 {
		return UsageError("search dependency", "query string required. Usage: limoxel search dependency <query>")
	}
	queryStr := strings.ToLower(flags.Arg(0))

	_, err := ctx.EnsureRepositoryService(flags.RepoRoot())
	if err != nil {
		return ExecutionError("search dependency", "failed to load repository", err)
	}

	depModel := ctx.DependencyModel()
	if depModel == nil || depModel.Inventory() == nil {
		return ContextError("search dependency", "dependency model unavailable")
	}

	var matchedDeps []*query.DependencyDTO
	for _, dep := range depModel.Inventory().AllDependencies() {
		if strings.Contains(strings.ToLower(dep.Name()), queryStr) {
			matchedDeps = append(matchedDeps, query.NewDependencyDTO(
				dep.Name(),
				dep.DeclaredVersion(),
				dep.IsDirect(),
				string(dep.Type()),
			))
		}
	}

	if ctx.Formatter().Format() == FormatJSON {
		type jsonDep struct {
			Name     string `json:"name"`
			Version  string `json:"version"`
			IsDirect bool   `json:"is_direct"`
			Type     string `json:"type"`
		}
		var jList []jsonDep
		for _, d := range matchedDeps {
			jList = append(jList, jsonDep{
				Name:     d.Name(),
				Version:  d.Version(),
				IsDirect: d.IsDirect(),
				Type:     d.Type(),
			})
		}
		return ctx.Formatter().RenderJSON(jList)
	}

	if len(matchedDeps) == 0 {
		return ctx.Formatter().RenderInfo(fmt.Sprintf("No dependencies found matching %q", queryStr))
	}

	_ = ctx.Formatter().RenderSuccess(fmt.Sprintf("Found %d dependencies matching %q", len(matchedDeps), queryStr))
	var rows [][]string
	for _, d := range matchedDeps {
		directStr := "transitive"
		if d.IsDirect() {
			directStr = "direct"
		}
		rows = append(rows, []string{d.Name(), d.Version(), directStr, d.Type()})
	}
	return ctx.Formatter().RenderTable([]string{"Dependency", "Version", "Scope", "Type"}, rows)
}
