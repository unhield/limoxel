package query

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
	"github.com/unhield/limoxel/internal/capabilities/repository/graph"
	"github.com/unhield/limoxel/internal/capabilities/repository/indexing"
	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
)

// sensitiveKeyPattern matches configuration keys that may contain secret credentials or keys.
var sensitiveKeyPattern = regexp.MustCompile(`(?i)(password|secret|token|apikey|api_key|private_key|auth|credential|cert|bearer|access_token|private)`)

// SearchEngine provides deterministic search and fuzzy matching across repository entities.
type SearchEngine struct {
	discResult *discovery.Result
	indexModel *indexing.IndexModel
	symModel   *symbol.SymbolModel
	kg         *graph.KnowledgeGraph
}

// NewSearchEngine constructs a SearchEngine.
func NewSearchEngine(
	discResult *discovery.Result,
	indexModel *indexing.IndexModel,
	symModel *symbol.SymbolModel,
	kg *graph.KnowledgeGraph,
) *SearchEngine {
	return &SearchEngine{
		discResult: discResult,
		indexModel: indexModel,
		symModel:   symModel,
		kg:         kg,
	}
}

// Search executes a search across the specified domain using options.
func (se *SearchEngine) Search(query string, domain SearchDomain, opts SearchOptions) (*SearchResultDTO, error) {
	if se == nil {
		return nil, ErrAnalysisUnavailable
	}
	start := time.Now()
	cleanQuery := strings.TrimSpace(query)
	if cleanQuery == "" {
		return nil, ErrEmptyQuery
	}

	var items []*SearchResultItem

	switch domain {
	case DomainSymbol:
		symItems, err := se.SearchSymbols(cleanQuery, opts)
		if err != nil {
			return nil, err
		}
		items = append(items, symItems...)
	case DomainFile:
		fileItems, err := se.SearchFiles(cleanQuery, opts)
		if err != nil {
			return nil, err
		}
		items = append(items, fileItems...)
	case DomainPackage:
		pkgItems, err := se.SearchPackages(cleanQuery, opts)
		if err != nil {
			return nil, err
		}
		items = append(items, pkgItems...)
	case DomainDocumentation:
		docItems, err := se.SearchDocumentation(cleanQuery, opts)
		if err != nil {
			return nil, err
		}
		items = append(items, docItems...)
	case DomainConfiguration:
		cfgItems, err := se.SearchConfiguration(cleanQuery, opts)
		if err != nil {
			return nil, err
		}
		items = append(items, cfgItems...)
	case DomainAll, "":
		symItems, _ := se.SearchSymbols(cleanQuery, opts)
		fileItems, _ := se.SearchFiles(cleanQuery, opts)
		pkgItems, _ := se.SearchPackages(cleanQuery, opts)
		docItems, _ := se.SearchDocumentation(cleanQuery, opts)
		cfgItems, _ := se.SearchConfiguration(cleanQuery, opts)
		items = append(items, symItems...)
		items = append(items, fileItems...)
		items = append(items, pkgItems...)
		items = append(items, docItems...)
		items = append(items, cfgItems...)
	default:
		return nil, ErrInvalidInput
	}

	items = rankAndSortResults(items, opts)
	return NewSearchResultDTO(cleanQuery, domain, items, time.Since(start)), nil
}

// SearchSymbols performs symbol search.
func (se *SearchEngine) SearchSymbols(query string, opts SearchOptions) ([]*SearchResultItem, error) {
	if se == nil || se.symModel == nil {
		return nil, ErrAnalysisUnavailable
	}
	cleanQuery := strings.TrimSpace(query)
	if cleanQuery == "" {
		return nil, ErrEmptyQuery
	}

	symDB := se.symModel.Symbols()
	if symDB == nil {
		return nil, ErrAnalysisUnavailable
	}

	var items []*SearchResultItem
	for _, s := range symDB.AllSymbols() {
		if s == nil {
			continue
		}
		score := calculateMatchScore(cleanQuery, s.Name(), s.FilePath(), opts)
		if score > 0 && score >= opts.MinScore {
			snippet := s.Signature()
			if snippet == "" {
				snippet = string(s.Kind()) + " " + s.Name()
			}
			items = append(items, NewSearchResultItem(
				s.ID(),
				DomainSymbol,
				s.Name(),
				s.FilePath(),
				s.PackageName(),
				string(s.Kind()),
				score,
				snippet,
				nil,
			))
		}
	}

	return rankAndSortResults(items, opts), nil
}

// SearchFiles searches repository files.
func (se *SearchEngine) SearchFiles(query string, opts SearchOptions) ([]*SearchResultItem, error) {
	if se == nil || se.discResult == nil {
		return nil, ErrAnalysisUnavailable
	}
	cleanQuery := strings.TrimSpace(query)
	if cleanQuery == "" {
		return nil, ErrEmptyQuery
	}

	var items []*SearchResultItem
	for _, f := range se.discResult.Files() {
		if f == nil {
			continue
		}
		relPath := f.RelPath()
		baseName := filepath.Base(relPath)
		score := calculateMatchScore(cleanQuery, baseName, relPath, opts)
		if score > 0 && score >= opts.MinScore {
			items = append(items, NewSearchResultItem(
				"file:"+relPath,
				DomainFile,
				baseName,
				relPath,
				"",
				"FILE",
				score,
				relPath,
				nil,
			))
		}
	}

	return rankAndSortResults(items, opts), nil
}

// SearchPackages searches repository packages.
func (se *SearchEngine) SearchPackages(query string, opts SearchOptions) ([]*SearchResultItem, error) {
	if se == nil {
		return nil, ErrAnalysisUnavailable
	}
	cleanQuery := strings.TrimSpace(query)
	if cleanQuery == "" {
		return nil, ErrEmptyQuery
	}

	var items []*SearchResultItem
	seenPkgs := make(map[string]bool)

	if se.indexModel != nil {
		for _, pkg := range se.indexModel.Packages() {
			if pkg == nil || seenPkgs[pkg.Path()] {
				continue
			}
			seenPkgs[pkg.Path()] = true
			score := calculateMatchScore(cleanQuery, pkg.Name(), pkg.Path(), opts)
			if score > 0 && score >= opts.MinScore {
				items = append(items, NewSearchResultItem(
					"pkg:"+pkg.Path(),
					DomainPackage,
					pkg.Name(),
					pkg.Path(),
					pkg.Name(),
					"PACKAGE",
					score,
					pkg.Path(),
					nil,
				))
			}
		}
	} else if se.kg != nil {
		for _, n := range se.kg.NodesByType(graph.NodePackage) {
			if n == nil || seenPkgs[n.Path()] {
				continue
			}
			seenPkgs[n.Path()] = true
			score := calculateMatchScore(cleanQuery, n.Name(), n.Path(), opts)
			if score > 0 && score >= opts.MinScore {
				items = append(items, NewSearchResultItem(
					n.ID(),
					DomainPackage,
					n.Name(),
					n.Path(),
					n.Name(),
					"PACKAGE",
					score,
					n.Path(),
					nil,
				))
			}
		}
	}

	return rankAndSortResults(items, opts), nil
}

// SearchDocumentation searches documentation files (markdown, READMEs, etc.).
func (se *SearchEngine) SearchDocumentation(query string, opts SearchOptions) ([]*SearchResultItem, error) {
	if se == nil || se.discResult == nil {
		return nil, ErrAnalysisUnavailable
	}
	cleanQuery := strings.TrimSpace(query)
	if cleanQuery == "" {
		return nil, ErrEmptyQuery
	}

	var items []*SearchResultItem
	for _, f := range se.discResult.Files() {
		if f == nil {
			continue
		}
		relPath := f.RelPath()
		baseName := filepath.Base(relPath)
		lower := strings.ToLower(relPath)
		if strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".markdown") || strings.HasSuffix(lower, ".txt") || strings.Contains(lower, "readme") {
			score := calculateMatchScore(cleanQuery, baseName, relPath, opts)
			if score > 0 && score >= opts.MinScore {
				items = append(items, NewSearchResultItem(
					"doc:"+relPath,
					DomainDocumentation,
					baseName,
					relPath,
					"",
					"DOCUMENTATION",
					score,
					relPath,
					nil,
				))
			}
		}
	}

	return rankAndSortResults(items, opts), nil
}

// SearchConfiguration searches configuration files while masking secrets according to security boundary.
func (se *SearchEngine) SearchConfiguration(query string, opts SearchOptions) ([]*SearchResultItem, error) {
	if se == nil || se.discResult == nil {
		return nil, ErrAnalysisUnavailable
	}
	cleanQuery := strings.TrimSpace(query)
	if cleanQuery == "" {
		return nil, ErrEmptyQuery
	}

	var items []*SearchResultItem
	for _, f := range se.discResult.Files() {
		if f == nil {
			continue
		}
		relPath := f.RelPath()
		baseName := filepath.Base(relPath)
		lower := strings.ToLower(relPath)
		if strings.HasSuffix(lower, ".json") || strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".toml") || strings.HasSuffix(lower, ".xml") || strings.HasSuffix(lower, ".ini") || strings.HasSuffix(lower, ".config") {
			score := calculateMatchScore(cleanQuery, baseName, relPath, opts)
			if score > 0 && score >= opts.MinScore {
				snippet := relPath
				// Check for sensitive credential keys in filename or config name
				if sensitiveKeyPattern.MatchString(relPath) {
					snippet = "***MASKED_CONFIG***"
				}
				items = append(items, NewSearchResultItem(
					"cfg:"+relPath,
					DomainConfiguration,
					baseName,
					relPath,
					"",
					"CONFIGURATION",
					score,
					snippet,
					nil,
				))
			}
		}
	}

	return rankAndSortResults(items, opts), nil
}

// FuzzySearch performs approximate string matching across target domain entities.
func (se *SearchEngine) FuzzySearch(query string, domain SearchDomain, opts SearchOptions) ([]*SearchResultItem, error) {
	if se == nil {
		return nil, ErrAnalysisUnavailable
	}
	cleanQuery := strings.TrimSpace(query)
	if cleanQuery == "" {
		return nil, ErrEmptyQuery
	}

	switch domain {
	case DomainSymbol, DomainFile, DomainPackage, DomainDocumentation, DomainConfiguration, DomainAll, "":
		// Valid domains
	default:
		return nil, ErrInvalidInput
	}

	var allCandidates []*SearchResultItem

	// 1. Symbols
	if domain == DomainSymbol || domain == DomainAll || domain == "" {
		if se.symModel != nil && se.symModel.Symbols() != nil {
			for _, s := range se.symModel.Symbols().AllSymbols() {
				if s == nil {
					continue
				}
				score := fuzzyScore(cleanQuery, s.Name(), opts.CaseSensitive)
				if score >= 0.3 {
					allCandidates = append(allCandidates, NewSearchResultItem(
						s.ID(),
						DomainSymbol,
						s.Name(),
						s.FilePath(),
						s.PackageName(),
						string(s.Kind()),
						score,
						s.Signature(),
						nil,
					))
				}
			}
		} else if domain == DomainSymbol {
			return nil, ErrAnalysisUnavailable
		}
	}

	// 2. Files
	if domain == DomainFile || domain == DomainAll || domain == "" {
		if se.discResult != nil {
			for _, f := range se.discResult.Files() {
				if f == nil {
					continue
				}
				baseName := filepath.Base(f.RelPath())
				score := fuzzyScore(cleanQuery, baseName, opts.CaseSensitive)
				if score >= 0.3 {
					allCandidates = append(allCandidates, NewSearchResultItem(
						"file:"+f.RelPath(),
						DomainFile,
						baseName,
						f.RelPath(),
						"",
						"FILE",
						score,
						f.RelPath(),
						nil,
					))
				}
			}
		} else if domain == DomainFile {
			return nil, ErrAnalysisUnavailable
		}
	}

	// 3. Packages
	if domain == DomainPackage || domain == DomainAll || domain == "" {
		seenPkgs := make(map[string]bool)
		if se.indexModel != nil {
			for _, pkg := range se.indexModel.Packages() {
				if pkg == nil || seenPkgs[pkg.Path()] {
					continue
				}
				seenPkgs[pkg.Path()] = true
				score := fuzzyScore(cleanQuery, pkg.Name(), opts.CaseSensitive)
				if score >= 0.3 {
					allCandidates = append(allCandidates, NewSearchResultItem(
						"pkg:"+pkg.Path(),
						DomainPackage,
						pkg.Name(),
						pkg.Path(),
						pkg.Name(),
						"PACKAGE",
						score,
						pkg.Path(),
						nil,
					))
				}
			}
		} else if se.kg != nil {
			for _, n := range se.kg.NodesByType(graph.NodePackage) {
				if n == nil || seenPkgs[n.Path()] {
					continue
				}
				seenPkgs[n.Path()] = true
				score := fuzzyScore(cleanQuery, n.Name(), opts.CaseSensitive)
				if score >= 0.3 {
					allCandidates = append(allCandidates, NewSearchResultItem(
						n.ID(),
						DomainPackage,
						n.Name(),
						n.Path(),
						n.Name(),
						"PACKAGE",
						score,
						n.Path(),
						nil,
					))
				}
			}
		} else if domain == DomainPackage {
			return nil, ErrAnalysisUnavailable
		}
	}

	// 4. Documentation
	if domain == DomainDocumentation || domain == DomainAll || domain == "" {
		if se.discResult != nil {
			for _, f := range se.discResult.Files() {
				if f == nil {
					continue
				}
				relPath := f.RelPath()
				lower := strings.ToLower(relPath)
				if strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".markdown") || strings.HasSuffix(lower, ".txt") || strings.Contains(lower, "readme") {
					baseName := filepath.Base(relPath)
					score := fuzzyScore(cleanQuery, baseName, opts.CaseSensitive)
					if score >= 0.3 {
						allCandidates = append(allCandidates, NewSearchResultItem(
							"doc:"+relPath,
							DomainDocumentation,
							baseName,
							relPath,
							"",
							"DOCUMENTATION",
							score,
							relPath,
							nil,
						))
					}
				}
			}
		} else if domain == DomainDocumentation {
			return nil, ErrAnalysisUnavailable
		}
	}

	// 5. Configuration
	if domain == DomainConfiguration || domain == DomainAll || domain == "" {
		if se.discResult != nil {
			for _, f := range se.discResult.Files() {
				if f == nil {
					continue
				}
				relPath := f.RelPath()
				lower := strings.ToLower(relPath)
				if strings.HasSuffix(lower, ".json") || strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".toml") || strings.HasSuffix(lower, ".xml") || strings.HasSuffix(lower, ".ini") || strings.HasSuffix(lower, ".config") {
					baseName := filepath.Base(relPath)
					score := fuzzyScore(cleanQuery, baseName, opts.CaseSensitive)
					if score >= 0.3 {
						snippet := relPath
						if sensitiveKeyPattern.MatchString(relPath) {
							snippet = "***MASKED_CONFIG***"
						}
						allCandidates = append(allCandidates, NewSearchResultItem(
							"cfg:"+relPath,
							DomainConfiguration,
							baseName,
							relPath,
							"",
							"CONFIGURATION",
							score,
							snippet,
							nil,
						))
					}
				}
			}
		} else if domain == DomainConfiguration {
			return nil, ErrAnalysisUnavailable
		}
	}

	return rankAndSortResults(allCandidates, opts), nil
}

// Helpers: Deterministic matching and scoring algorithms

func calculateMatchScore(query, name, path string, opts SearchOptions) float64 {
	q := query
	n := name
	p := path
	if !opts.CaseSensitive {
		q = strings.ToLower(q)
		n = strings.ToLower(n)
		p = strings.ToLower(p)
	}

	// Wildcard / match-all query
	if q == "*" || q == "." || q == "" {
		return 1.0
	}

	// Glob pattern match
	if matched, _ := filepath.Match(q, n); matched {
		return 0.95
	}
	if matched, _ := filepath.Match(q, p); matched {
		return 0.90
	}

	if opts.ExactMatch {
		if n == q {
			return 1.0
		}
		return 0.0
	}

	// Exact match
	if n == q {
		return 1.0
	}
	// Prefix match on name
	if strings.HasPrefix(n, q) {
		return 0.8 + 0.1*(float64(len(q))/float64(len(n)))
	}
	// Substring match on name
	if strings.Contains(n, q) {
		return 0.6 + 0.1*(float64(len(q))/float64(len(n)))
	}
	// Substring match on path
	if strings.Contains(p, q) {
		return 0.4
	}

	return 0.0
}

func fuzzyScore(query, target string, caseSensitive bool) float64 {
	q := query
	t := target
	if !caseSensitive {
		q = strings.ToLower(q)
		t = strings.ToLower(t)
	}

	if q == t {
		return 1.0
	}
	if strings.HasPrefix(t, q) {
		return 0.9
	}
	if strings.Contains(t, q) {
		return 0.75
	}

	// Normalized Levenshtein distance
	dist := levenshteinDistance(q, t)
	maxLen := len(q)
	if len(t) > maxLen {
		maxLen = len(t)
	}
	if maxLen == 0 {
		return 1.0
	}

	sim := 1.0 - float64(dist)/float64(maxLen)
	if sim < 0 {
		return 0.0
	}
	return sim
}

func levenshteinDistance(s1, s2 string) int {
	r1, r2 := []rune(s1), []rune(s2)
	n, m := len(r1), len(r2)
	if n == 0 {
		return m
	}
	if m == 0 {
		return n
	}

	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
		dp[i][0] = i
	}
	for j := 0; j <= m; j++ {
		dp[0][j] = j
	}

	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			cost := 1
			if r1[i-1] == r2[j-1] {
				cost = 0
			}
			dp[i][j] = min(
				dp[i-1][j]+1,      // deletion
				dp[i][j-1]+1,      // insertion
				dp[i-1][j-1]+cost, // substitution
			)
		}
	}

	return dp[n][m]
}

func min(a, b, c int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}

// rankAndSortResults performs deterministic ranking: Score desc, then EntityID asc.
func rankAndSortResults(items []*SearchResultItem, opts SearchOptions) []*SearchResultItem {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Score() != items[j].Score() {
			return items[i].Score() > items[j].Score()
		}
		if items[i].Domain() != items[j].Domain() {
			return items[i].Domain() < items[j].Domain()
		}
		return items[i].EntityID() < items[j].EntityID()
	})

	if opts.MaxResults > 0 && len(items) > opts.MaxResults {
		return items[:opts.MaxResults]
	}
	return items
}
