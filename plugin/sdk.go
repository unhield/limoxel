package plugin

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/unhield/limoxel/plugin/api"
)

// Authoritative Plugin SDK Version
const (
	SDKVersion    = "1.0.0"
	SDKAPIVersion = "1.0.0"
)

// Canonical Error Definitions
var (
	ErrInvalidRequest        = errors.New("plugin sdk: invalid request")
	ErrCapabilityUnavailable = errors.New("plugin sdk: requested capability is unavailable")
	ErrIncompatibleSDK       = errors.New("plugin sdk: incompatible sdk or host version")
	ErrLifecycleViolation    = errors.New("plugin sdk: invalid lifecycle state for operation")
	ErrRuntimeFailure        = errors.New("plugin sdk: runtime execution failure")
	ErrTransportFailure      = errors.New("plugin sdk: transport or protocol error")
	ErrTimeout               = errors.New("plugin sdk: operation timed out")
	ErrCancelled             = errors.New("plugin sdk: operation was cancelled")
)

// Context provides a unified developer-facing context for a plugin instance.
type Context interface {
	// Context returns the active cancellation context.
	Context() context.Context

	// Host returns the foundational host interface.
	Host() Host

	// Config returns the configuration parameters passed to the plugin.
	Config() map[string]string

	// Logger returns the structured logger for this plugin.
	Logger() Logger

	// Repository returns the repository workspace API facade.
	Repository() api.RepositoryAPI

	// Symbols returns the symbol resolution and hierarchy API facade.
	Symbols() api.SymbolAPI

	// Search returns the multi-domain repository search API facade.
	Search() api.SearchAPI

	// Graph returns the Knowledge Graph inspection and traversal API facade.
	Graph() api.GraphAPI

	// Analysis returns the code analysis and repository health API facade.
	Analysis() api.AnalysisAPI

	// Events returns the pub/sub event routing API facade.
	Events() api.EventAPI
}

// ExtendedHost is an optional interface that a Host may implement to provide rich APIs.
type ExtendedHost interface {
	Host
	RepositoryAPI() api.RepositoryAPI
	SymbolAPI() api.SymbolAPI
	SearchAPI() api.SearchAPI
	GraphAPI() api.GraphAPI
	AnalysisAPI() api.AnalysisAPI
	EventAPI() api.EventAPI
}

// PluginContext implements Context.
type PluginContext struct {
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	host     Host
	config   map[string]string
	logger   Logger
	repo     api.RepositoryAPI
	symbols  api.SymbolAPI
	search   api.SearchAPI
	graph    api.GraphAPI
	analysis api.AnalysisAPI
	events   api.EventAPI
}

// NewContext constructs an initialized PluginContext using the host.
func NewContext(ctx context.Context, host Host, cfg map[string]string) *PluginContext {
	if ctx == nil {
		ctx = context.Background()
	}
	cCtx, cancel := context.WithCancel(ctx)

	cfgCopy := make(map[string]string, len(cfg))
	for k, v := range cfg {
		cfgCopy[k] = v
	}

	pCtx := &PluginContext{
		ctx:    cCtx,
		cancel: cancel,
		host:   host,
		config: cfgCopy,
	}

	if host != nil {
		pCtx.logger = host.Logger()

		// If the host provides direct extended APIs, consume them
		if ext, ok := host.(ExtendedHost); ok {
			pCtx.repo = ext.RepositoryAPI()
			pCtx.symbols = ext.SymbolAPI()
			pCtx.search = ext.SearchAPI()
			pCtx.graph = ext.GraphAPI()
			pCtx.analysis = ext.AnalysisAPI()
			pCtx.events = ext.EventAPI()
		} else {
			// Provide standard fallback adapters over Host facades
			pCtx.repo = &fallbackRepoAdapter{facade: host.Repository()}
			pCtx.events = &fallbackEventAdapter{facade: host.Events()}
			pCtx.symbols = &unavailableSymbolAPI{}
			pCtx.search = &unavailableSearchAPI{}
			pCtx.graph = &unavailableGraphAPI{}
			pCtx.analysis = &unavailableAnalysisAPI{}
		}
	} else {
		pCtx.logger = &noopLogger{}
		pCtx.repo = &unavailableRepoAPI{}
		pCtx.events = &unavailableEventAPI{}
		pCtx.symbols = &unavailableSymbolAPI{}
		pCtx.search = &unavailableSearchAPI{}
		pCtx.graph = &unavailableGraphAPI{}
		pCtx.analysis = &unavailableAnalysisAPI{}
	}

	return pCtx
}

// SetAPIs configures custom or extended API implementations on the PluginContext.
func (c *PluginContext) SetAPIs(
	repo api.RepositoryAPI,
	symbols api.SymbolAPI,
	search api.SearchAPI,
	graph api.GraphAPI,
	analysis api.AnalysisAPI,
	events api.EventAPI,
) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if repo != nil {
		c.repo = repo
	}
	if symbols != nil {
		c.symbols = symbols
	}
	if search != nil {
		c.search = search
	}
	if graph != nil {
		c.graph = graph
	}
	if analysis != nil {
		c.analysis = analysis
	}
	if events != nil {
		c.events = events
	}
}

func (c *PluginContext) Context() context.Context {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ctx
}

func (c *PluginContext) Cancel() {
	c.cancel()
}

func (c *PluginContext) Host() Host {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.host
}

func (c *PluginContext) Config() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make(map[string]string, len(c.config))
	for k, v := range c.config {
		out[k] = v
	}
	return out
}

func (c *PluginContext) Logger() Logger {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.logger
}

func (c *PluginContext) Repository() api.RepositoryAPI {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.repo
}

func (c *PluginContext) Symbols() api.SymbolAPI {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.symbols
}

func (c *PluginContext) Search() api.SearchAPI {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.search
}

func (c *PluginContext) Graph() api.GraphAPI {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.graph
}

func (c *PluginContext) Analysis() api.AnalysisAPI {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.analysis
}

func (c *PluginContext) Events() api.EventAPI {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.events
}

// BasePlugin provides a canonical, embedding-friendly implementation of Plugin.
type BasePlugin struct {
	mu           sync.RWMutex
	PluginID     string
	SpecManifest Manifest
	PluginCtx    Context
	IsStarted    bool
}

// NewBasePlugin initializes a BasePlugin with identifier and manifest.
func NewBasePlugin(id string, manifest Manifest) *BasePlugin {
	return &BasePlugin{
		PluginID:     id,
		SpecManifest: manifest,
	}
}

func (b *BasePlugin) ID() string {
	return b.PluginID
}

func (b *BasePlugin) Manifest() Manifest {
	return b.SpecManifest
}

func (b *BasePlugin) Init(ctx context.Context, host Host) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.PluginCtx = NewContext(ctx, host, nil)
	return nil
}

func (b *BasePlugin) Start(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.IsStarted = true
	return nil
}

func (b *BasePlugin) Stop(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.IsStarted = false
	if pCtx, ok := b.PluginCtx.(*PluginContext); ok {
		pCtx.Cancel()
	}
	return nil
}

func (b *BasePlugin) GetContext() Context {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.PluginCtx
}

// Fallback Adapters

type fallbackRepoAdapter struct {
	facade RepositoryFacade
}

func (f *fallbackRepoAdapter) WorkspacePath() string {
	if f.facade == nil {
		return ""
	}
	return f.facade.WorkspacePath()
}

func (f *fallbackRepoAdapter) Info(ctx context.Context) (*api.RepositoryInfo, error) {
	if f.facade == nil {
		return nil, ErrCapabilityUnavailable
	}
	meta, err := f.facade.Metadata(ctx)
	if err != nil {
		return nil, err
	}
	return &api.RepositoryInfo{
		Name:          meta.Name,
		RootPath:      meta.RootPath,
		DefaultBranch: meta.DefaultBranch,
		CurrentBranch: meta.CurrentBranch,
		IsGit:         meta.IsGit,
		State:         "READY",
		Properties:    meta.Properties,
	}, nil
}

func (f *fallbackRepoAdapter) Metadata(ctx context.Context) (*api.RepositoryMetadata, error) {
	if f.facade == nil {
		return nil, ErrCapabilityUnavailable
	}
	m, err := f.facade.Metadata(ctx)
	if err != nil {
		return nil, err
	}
	return &api.RepositoryMetadata{
		Name:       m.Name,
		VCS:        "git",
		Properties: m.Properties,
	}, nil
}

func (f *fallbackRepoAdapter) Statistics(ctx context.Context) (*api.RepositoryStatistics, error) {
	if f.facade == nil {
		return nil, ErrCapabilityUnavailable
	}
	files, err := f.facade.ListFiles(ctx, "")
	if err != nil {
		return nil, err
	}
	return &api.RepositoryStatistics{
		TotalFiles: len(files),
	}, nil
}

func (f *fallbackRepoAdapter) ListFiles(ctx context.Context, prefix string) ([]string, error) {
	if f.facade == nil {
		return nil, ErrCapabilityUnavailable
	}
	return f.facade.ListFiles(ctx, prefix)
}

func (f *fallbackRepoAdapter) GetFile(ctx context.Context, relPath string) (*api.FileInfo, error) {
	return &api.FileInfo{
		Path: relPath,
	}, nil
}

type fallbackEventAdapter struct {
	facade EventFacade
}

func (f *fallbackEventAdapter) Publish(ctx context.Context, evt api.Event) error {
	if f.facade == nil {
		return ErrCapabilityUnavailable
	}
	return f.facade.Publish(ctx, evt.Type, []byte(fmt.Sprintf("%v", evt.Payload)))
}

func (f *fallbackEventAdapter) Subscribe(ctx context.Context, eventType string, handler api.EventHandler) (api.Subscription, error) {
	if f.facade == nil {
		return nil, ErrCapabilityUnavailable
	}
	sub, err := f.facade.Subscribe(eventType, func(c context.Context, topic string, payload []byte) error {
		return handler(c, api.Event{
			ID:        fmt.Sprintf("evt-%d", time.Now().UnixNano()),
			Type:      topic,
			Timestamp: time.Now().UTC(),
			Payload:   payload,
		})
	})
	if err != nil {
		return nil, err
	}
	return &wrappedSubscription{sub: sub, id: fmt.Sprintf("sub-%s", eventType)}, nil
}

func (f *fallbackEventAdapter) SubscribeAll(ctx context.Context, handler api.EventHandler) (api.Subscription, error) {
	return f.Subscribe(ctx, "*", handler)
}

type wrappedSubscription struct {
	sub Subscription
	id  string
}

func (w *wrappedSubscription) ID() string {
	return w.id
}

func (w *wrappedSubscription) Unsubscribe() error {
	if w.sub != nil {
		return w.sub.Unsubscribe()
	}
	return nil
}

// Fallback unavailables
type unavailableRepoAPI struct{}

func (u *unavailableRepoAPI) WorkspacePath() string { return "" }
func (u *unavailableRepoAPI) Info(ctx context.Context) (*api.RepositoryInfo, error) {
	return nil, ErrCapabilityUnavailable
}
func (u *unavailableRepoAPI) Metadata(ctx context.Context) (*api.RepositoryMetadata, error) {
	return nil, ErrCapabilityUnavailable
}
func (u *unavailableRepoAPI) Statistics(ctx context.Context) (*api.RepositoryStatistics, error) {
	return nil, ErrCapabilityUnavailable
}
func (u *unavailableRepoAPI) ListFiles(ctx context.Context, prefix string) ([]string, error) {
	return nil, ErrCapabilityUnavailable
}
func (u *unavailableRepoAPI) GetFile(ctx context.Context, relPath string) (*api.FileInfo, error) {
	return nil, ErrCapabilityUnavailable
}

type unavailableSymbolAPI struct{}

func (u *unavailableSymbolAPI) LookupSymbol(ctx context.Context, id string) (*api.SymbolInfo, error) {
	return nil, ErrCapabilityUnavailable
}
func (u *unavailableSymbolAPI) FindSymbols(ctx context.Context, pattern, pkgScope string, opts api.PaginationOptions) ([]api.SymbolInfo, error) {
	return nil, ErrCapabilityUnavailable
}
func (u *unavailableSymbolAPI) SymbolHierarchy(ctx context.Context, id string) (*api.SymbolHierarchyNode, error) {
	return nil, ErrCapabilityUnavailable
}
func (u *unavailableSymbolAPI) SymbolReferences(ctx context.Context, id string, opts api.PaginationOptions) ([]api.SymbolReference, error) {
	return nil, ErrCapabilityUnavailable
}
func (u *unavailableSymbolAPI) SymbolDocumentation(ctx context.Context, id string) (string, error) {
	return "", ErrCapabilityUnavailable
}

type unavailableSearchAPI struct{}

func (u *unavailableSearchAPI) Search(ctx context.Context, q api.SearchQuery) (*api.SearchResult, error) {
	return nil, ErrCapabilityUnavailable
}
func (u *unavailableSearchAPI) SearchSymbols(ctx context.Context, q string, opts api.PaginationOptions) (*api.SearchResult, error) {
	return nil, ErrCapabilityUnavailable
}
func (u *unavailableSearchAPI) SearchPackages(ctx context.Context, q string, opts api.PaginationOptions) (*api.SearchResult, error) {
	return nil, ErrCapabilityUnavailable
}
func (u *unavailableSearchAPI) SearchFiles(ctx context.Context, p string, opts api.PaginationOptions) (*api.SearchResult, error) {
	return nil, ErrCapabilityUnavailable
}
func (u *unavailableSearchAPI) SearchDocs(ctx context.Context, q string, opts api.PaginationOptions) (*api.SearchResult, error) {
	return nil, ErrCapabilityUnavailable
}
func (u *unavailableSearchAPI) SearchConfigs(ctx context.Context, k string, opts api.PaginationOptions) (*api.SearchResult, error) {
	return nil, ErrCapabilityUnavailable
}

type unavailableGraphAPI struct{}

func (u *unavailableGraphAPI) GraphInfo(ctx context.Context) (int, int, error) {
	return 0, 0, ErrCapabilityUnavailable
}
func (u *unavailableGraphAPI) GetNode(ctx context.Context, nodeID string) (*api.GraphNode, error) {
	return nil, ErrCapabilityUnavailable
}
func (u *unavailableGraphAPI) GetRelationship(ctx context.Context, relID string) (*api.GraphRelationship, error) {
	return nil, ErrCapabilityUnavailable
}
func (u *unavailableGraphAPI) TraverseNodes(ctx context.Context, startNodeID string, filter api.GraphFilter) ([]api.GraphNode, error) {
	return nil, ErrCapabilityUnavailable
}
func (u *unavailableGraphAPI) TraverseRelationships(ctx context.Context, startNodeID string, filter api.GraphFilter) ([]api.GraphRelationship, error) {
	return nil, ErrCapabilityUnavailable
}
func (u *unavailableGraphAPI) GetNeighbors(ctx context.Context, nodeID string, direction string, kinds ...string) ([]api.GraphNode, error) {
	return nil, ErrCapabilityUnavailable
}
func (u *unavailableGraphAPI) ExportGraph(ctx context.Context, filter api.GraphFilter, format api.GraphExportFormat) (*api.GraphExportResult, error) {
	return nil, ErrCapabilityUnavailable
}

type unavailableAnalysisAPI struct{}

func (u *unavailableAnalysisAPI) AnalyzeArchitecture(ctx context.Context, scope string) (*api.ArchitectureReport, error) {
	return nil, ErrCapabilityUnavailable
}
func (u *unavailableAnalysisAPI) AnalyzeDependencies(ctx context.Context, scope string) (*api.DependencyReport, error) {
	return nil, ErrCapabilityUnavailable
}
func (u *unavailableAnalysisAPI) RepositoryHealth(ctx context.Context) (*api.HealthReport, error) {
	return nil, ErrCapabilityUnavailable
}
func (u *unavailableAnalysisAPI) AnalyzeQuality(ctx context.Context, scope string) (*api.QualityReport, error) {
	return nil, ErrCapabilityUnavailable
}
func (u *unavailableAnalysisAPI) AnalyzeConfiguration(ctx context.Context) (*api.ConfigurationReport, error) {
	return nil, ErrCapabilityUnavailable
}

type unavailableEventAPI struct{}

func (u *unavailableEventAPI) Publish(ctx context.Context, evt api.Event) error {
	return ErrCapabilityUnavailable
}
func (u *unavailableEventAPI) Subscribe(ctx context.Context, eventType string, handler api.EventHandler) (api.Subscription, error) {
	return nil, ErrCapabilityUnavailable
}
func (u *unavailableEventAPI) SubscribeAll(ctx context.Context, handler api.EventHandler) (api.Subscription, error) {
	return nil, ErrCapabilityUnavailable
}

type noopLogger struct{}

func (n *noopLogger) Debug(msg string, keyvals ...any) {}
func (n *noopLogger) Info(msg string, keyvals ...any)  {}
func (n *noopLogger) Warn(msg string, keyvals ...any)  {}
func (n *noopLogger) Error(msg string, keyvals ...any) {}
