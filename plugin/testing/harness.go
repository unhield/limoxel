package plugintesting

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
	"github.com/unhield/limoxel/plugin"
	"github.com/unhield/limoxel/plugin/api"
)

// MockHost implements plugin.Host and plugin.ExtendedHost for testing plugin instances.
type MockHost struct {
	mu           sync.RWMutex
	wsRoot       string
	capabilities map[string]bool
	logger       *TestLogger
	repoFacade   *MockRepoFacade
	repoAPI      *MockRepositoryAPI
	symbolAPI    *MockSymbolAPI
	searchAPI    *MockSearchAPI
	graphAPI     *MockGraphAPI
	analysisAPI  *MockAnalysisAPI
	eventFacade  *MockEventFacade
	eventAPI     *MockEventAPI
}

// NewMockHost constructs an initialized MockHost.
func NewMockHost(workspaceRoot string) *MockHost {
	ws := filepath.Clean(workspaceRoot)
	if ws == "" || ws == "." {
		ws = "."
	}
	m := &MockHost{
		wsRoot: ws,
		capabilities: map[string]bool{
			"repository.metadata": true,
			"repository.files":    true,
			"events.pubsub":       true,
			"logging.structured":  true,
			"symbols.api":         true,
			"search.api":          true,
			"graph.api":           true,
			"analysis.api":        true,
		},
		logger:      &TestLogger{},
		repoFacade:  &MockRepoFacade{ws: ws},
		repoAPI:     &MockRepositoryAPI{ws: ws},
		symbolAPI:   &MockSymbolAPI{},
		searchAPI:   &MockSearchAPI{},
		graphAPI:    &MockGraphAPI{},
		analysisAPI: &MockAnalysisAPI{},
		eventFacade: &MockEventFacade{subs: make(map[string][]plugin.EventHandler)},
		eventAPI:    &MockEventAPI{subs: make(map[string][]api.EventHandler)},
	}
	return m
}

func (m *MockHost) HostVersion() version.SemVer {
	return version.SemVer{Major: 1, Minor: 0, Patch: 0}
}

func (m *MockHost) Capabilities() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []string
	for k, v := range m.capabilities {
		if v {
			out = append(out, k)
		}
	}
	return out
}

func (m *MockHost) HasCapability(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.capabilities[name]
}

func (m *MockHost) Repository() plugin.RepositoryFacade { return m.repoFacade }
func (m *MockHost) Events() plugin.EventFacade          { return m.eventFacade }
func (m *MockHost) Logger() plugin.Logger               { return m.logger }

func (m *MockHost) RepositoryAPI() api.RepositoryAPI { return m.repoAPI }
func (m *MockHost) SymbolAPI() api.SymbolAPI         { return m.symbolAPI }
func (m *MockHost) SearchAPI() api.SearchAPI         { return m.searchAPI }
func (m *MockHost) GraphAPI() api.GraphAPI           { return m.graphAPI }
func (m *MockHost) AnalysisAPI() api.AnalysisAPI     { return m.analysisAPI }
func (m *MockHost) EventAPI() api.EventAPI           { return m.eventAPI }

var (
	_ plugin.Host         = (*MockHost)(nil)
	_ plugin.ExtendedHost = (*MockHost)(nil)
)

// MockRepoFacade implements plugin.RepositoryFacade
type MockRepoFacade struct {
	ws string
}

func (r *MockRepoFacade) WorkspacePath() string { return r.ws }

func (r *MockRepoFacade) Metadata(ctx context.Context) (*plugin.RepositoryMetadata, error) {
	return &plugin.RepositoryMetadata{
		Name:     filepath.Base(r.ws),
		RootPath: r.ws,
		IsGit:    true,
	}, nil
}

func (r *MockRepoFacade) ListFiles(ctx context.Context, prefix string) ([]string, error) {
	return []string{"main.go", "plugin.json", "README.md"}, nil
}

// MockRepositoryAPI implements api.RepositoryAPI
type MockRepositoryAPI struct {
	ws string
}

func (r *MockRepositoryAPI) WorkspacePath() string { return r.ws }

func (r *MockRepositoryAPI) Metadata(ctx context.Context) (*api.RepositoryMetadata, error) {
	return &api.RepositoryMetadata{
		Name: filepath.Base(r.ws),
		VCS:  "git",
	}, nil
}

func (r *MockRepositoryAPI) ListFiles(ctx context.Context, prefix string) ([]string, error) {
	return []string{"main.go", "plugin.json", "README.md"}, nil
}

func (r *MockRepositoryAPI) Info(ctx context.Context) (*api.RepositoryInfo, error) {
	return &api.RepositoryInfo{
		Name:     filepath.Base(r.ws),
		RootPath: r.ws,
		State:    "READY",
		IsGit:    true,
	}, nil
}

func (r *MockRepositoryAPI) Statistics(ctx context.Context) (*api.RepositoryStatistics, error) {
	return &api.RepositoryStatistics{
		TotalFiles:       3,
		TotalDirectories: 1,
		TotalSymbols:     10,
	}, nil
}

func (r *MockRepositoryAPI) GetFile(ctx context.Context, relPath string) (*api.FileInfo, error) {
	return &api.FileInfo{
		Path: relPath,
		Size: 1024,
	}, nil
}

var _ api.RepositoryAPI = (*MockRepositoryAPI)(nil)

// MockSymbolAPI
type MockSymbolAPI struct{}

func (s *MockSymbolAPI) LookupSymbol(ctx context.Context, symbolIDOrName string) (*api.SymbolInfo, error) {
	return &api.SymbolInfo{
		ID:      symbolIDOrName,
		Name:    symbolIDOrName,
		Package: "main",
		Kind:    api.SymbolKindFunction,
	}, nil
}

func (s *MockSymbolAPI) FindSymbols(ctx context.Context, pattern, pkgScope string, opts api.PaginationOptions) ([]api.SymbolInfo, error) {
	return []api.SymbolInfo{
		{ID: "sym-1", Name: pattern, Package: pkgScope, Kind: api.SymbolKindFunction},
	}, nil
}

func (s *MockSymbolAPI) SymbolHierarchy(ctx context.Context, symbolID string) (*api.SymbolHierarchyNode, error) {
	return &api.SymbolHierarchyNode{
		Symbol: api.SymbolInfo{ID: symbolID, Name: symbolID, Kind: api.SymbolKindType},
	}, nil
}

func (s *MockSymbolAPI) SymbolReferences(ctx context.Context, symbolID string, opts api.PaginationOptions) ([]api.SymbolReference, error) {
	return []api.SymbolReference{
		{SourceSymbolID: "caller", SourceFile: "main.go", SourceLine: 42},
	}, nil
}

func (s *MockSymbolAPI) SymbolDocumentation(ctx context.Context, symbolID string) (string, error) {
	return fmt.Sprintf("Doc for %s", symbolID), nil
}

var _ api.SymbolAPI = (*MockSymbolAPI)(nil)

// MockSearchAPI
type MockSearchAPI struct{}

func (s *MockSearchAPI) Search(ctx context.Context, q api.SearchQuery) (*api.SearchResult, error) {
	return &api.SearchResult{Query: q.Query, TotalMatches: 1}, nil
}
func (s *MockSearchAPI) SearchSymbols(ctx context.Context, query string, opts api.PaginationOptions) (*api.SearchResult, error) {
	return &api.SearchResult{Query: query, TotalMatches: 1}, nil
}
func (s *MockSearchAPI) SearchPackages(ctx context.Context, query string, opts api.PaginationOptions) (*api.SearchResult, error) {
	return &api.SearchResult{Query: query, TotalMatches: 1}, nil
}
func (s *MockSearchAPI) SearchFiles(ctx context.Context, pattern string, opts api.PaginationOptions) (*api.SearchResult, error) {
	return &api.SearchResult{Query: pattern, TotalMatches: 1}, nil
}
func (s *MockSearchAPI) SearchDocs(ctx context.Context, query string, opts api.PaginationOptions) (*api.SearchResult, error) {
	return &api.SearchResult{Query: query, TotalMatches: 1}, nil
}
func (s *MockSearchAPI) SearchConfigs(ctx context.Context, key string, opts api.PaginationOptions) (*api.SearchResult, error) {
	return &api.SearchResult{Query: key, TotalMatches: 1}, nil
}

var _ api.SearchAPI = (*MockSearchAPI)(nil)

// MockGraphAPI
type MockGraphAPI struct{}

func (g *MockGraphAPI) GraphInfo(ctx context.Context) (int, int, error) {
	return 100, 250, nil
}
func (g *MockGraphAPI) GetNode(ctx context.Context, nodeID string) (*api.GraphNode, error) {
	return &api.GraphNode{ID: nodeID, Name: nodeID}, nil
}
func (g *MockGraphAPI) GetRelationship(ctx context.Context, relID string) (*api.GraphRelationship, error) {
	return &api.GraphRelationship{ID: relID}, nil
}
func (g *MockGraphAPI) TraverseNodes(ctx context.Context, startNodeID string, filter api.GraphFilter) ([]api.GraphNode, error) {
	return []api.GraphNode{{ID: startNodeID}}, nil
}
func (g *MockGraphAPI) TraverseRelationships(ctx context.Context, startNodeID string, filter api.GraphFilter) ([]api.GraphRelationship, error) {
	return []api.GraphRelationship{{ID: "rel-1", SourceID: startNodeID}}, nil
}
func (g *MockGraphAPI) GetNeighbors(ctx context.Context, nodeID string, direction string, kinds ...string) ([]api.GraphNode, error) {
	return []api.GraphNode{{ID: "neighbor-1"}}, nil
}
func (g *MockGraphAPI) ExportGraph(ctx context.Context, filter api.GraphFilter, format api.GraphExportFormat) (*api.GraphExportResult, error) {
	return &api.GraphExportResult{Format: format, Content: "graph-content"}, nil
}

var _ api.GraphAPI = (*MockGraphAPI)(nil)

// MockAnalysisAPI
type MockAnalysisAPI struct{}

func (a *MockAnalysisAPI) AnalyzeArchitecture(ctx context.Context, scope string) (*api.ArchitectureReport, error) {
	return &api.ArchitectureReport{TotalComponents: 5, LayerCount: 3}, nil
}
func (a *MockAnalysisAPI) AnalyzeDependencies(ctx context.Context, scope string) (*api.DependencyReport, error) {
	return &api.DependencyReport{TotalDependencies: 12, DirectDependencies: 4}, nil
}
func (a *MockAnalysisAPI) RepositoryHealth(ctx context.Context) (*api.HealthReport, error) {
	return &api.HealthReport{OverallScore: 92.5, Grade: "A", Status: "HEALTHY"}, nil
}
func (a *MockAnalysisAPI) AnalyzeQuality(ctx context.Context, scope string) (*api.QualityReport, error) {
	return &api.QualityReport{MaintainabilityScore: 88.0, ComplexityScore: 15.0}, nil
}
func (a *MockAnalysisAPI) AnalyzeConfiguration(ctx context.Context) (*api.ConfigurationReport, error) {
	return &api.ConfigurationReport{TotalConfigs: 3, ValidConfigs: 3}, nil
}

var _ api.AnalysisAPI = (*MockAnalysisAPI)(nil)

// MockEventFacade implements plugin.EventFacade
type MockEventFacade struct {
	mu   sync.RWMutex
	subs map[string][]plugin.EventHandler
}

func (e *MockEventFacade) Subscribe(topic string, handler plugin.EventHandler) (plugin.Subscription, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.subs[topic] = append(e.subs[topic], handler)
	return &testSub{}, nil
}

func (e *MockEventFacade) Publish(ctx context.Context, topic string, payload []byte) error {
	e.mu.RLock()
	handlers := e.subs[topic]
	all := e.subs["*"]
	e.mu.RUnlock()

	for _, h := range append(handlers, all...) {
		_ = h(ctx, topic, payload)
	}
	return nil
}

// MockEventAPI implements api.EventAPI
type MockEventAPI struct {
	mu   sync.RWMutex
	subs map[string][]api.EventHandler
}

func (e *MockEventAPI) Publish(ctx context.Context, evt api.Event) error {
	e.mu.RLock()
	var handlers []api.EventHandler
	for pattern, hs := range e.subs {
		if pattern == "*" || pattern == evt.Type || (strings.HasSuffix(pattern, ".*") && strings.HasPrefix(evt.Type, strings.TrimSuffix(pattern, ".*"))) {
			handlers = append(handlers, hs...)
		}
	}
	e.mu.RUnlock()

	for _, h := range handlers {
		_ = h(ctx, evt)
	}
	return nil
}

func (e *MockEventAPI) Subscribe(ctx context.Context, eventType string, handler api.EventHandler) (api.Subscription, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.subs[eventType] = append(e.subs[eventType], handler)
	return &testSub{}, nil
}

func (e *MockEventAPI) SubscribeAll(ctx context.Context, handler api.EventHandler) (api.Subscription, error) {
	return e.Subscribe(ctx, "*", handler)
}

var _ api.EventAPI = (*MockEventAPI)(nil)

type testSub struct{}

func (s *testSub) ID() string         { return "test-sub" }
func (s *testSub) Unsubscribe() error { return nil }

// TestLogger
type TestLogger struct {
	mu   sync.Mutex
	Logs []string
}

func (l *TestLogger) log(level, msg string, kv ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.Logs = append(l.Logs, fmt.Sprintf("[%s] %s %v", level, msg, kv))
}

func (l *TestLogger) Debug(msg string, kv ...any) { l.log("DEBUG", msg, kv...) }
func (l *TestLogger) Info(msg string, kv ...any)  { l.log("INFO", msg, kv...) }
func (l *TestLogger) Warn(msg string, kv ...any)  { l.log("WARN", msg, kv...) }
func (l *TestLogger) Error(msg string, kv ...any) { l.log("ERROR", msg, kv...) }

// TestHarness bundles test helpers and a mock host for convenient plugin testing.
type TestHarness struct {
	HostInstance *MockHost
}

// NewTestHarness constructs a TestHarness with an active MockHost.
func NewTestHarness(workspaceRoot string) *TestHarness {
	return &TestHarness{
		HostInstance: NewMockHost(workspaceRoot),
	}
}

// ExecuteLifecycle runs Init, Start, and Stop on the given plugin, asserting clean execution.
func (h *TestHarness) ExecuteLifecycle(ctx context.Context, p plugin.Plugin) error {
	if err := p.Init(ctx, h.HostInstance); err != nil {
		return fmt.Errorf("plugin init failed: %w", err)
	}
	if err := p.Start(ctx); err != nil {
		return fmt.Errorf("plugin start failed: %w", err)
	}
	if err := p.Stop(ctx); err != nil {
		return fmt.Errorf("plugin stop failed: %w", err)
	}
	return nil
}
