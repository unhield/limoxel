package security

import (
	"context"
	"fmt"

	"github.com/unhield/limoxel/internal/capabilities/plugin/security/monitoring"
	"github.com/unhield/limoxel/internal/capabilities/plugin/security/permissions"
	"github.com/unhield/limoxel/internal/capabilities/plugin/security/sandbox"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
	"github.com/unhield/limoxel/plugin"
	"github.com/unhield/limoxel/plugin/api"
	pubsec "github.com/unhield/limoxel/plugin/security"
)

// SandboxedHostAdapter decorates plugin.ExtendedHost and plugin.Host with fail-closed security enforcement.
type SandboxedHostAdapter struct {
	inner      plugin.ExtendedHost
	pluginID   string
	permEngine *permissions.Engine
	fsGuard    *sandbox.FilesystemGuard
	monitor    *monitoring.SecurityMonitor
}

// NewSandboxedHostAdapter wraps an existing host environment with sandbox and permission controls.
func NewSandboxedHostAdapter(
	inner plugin.ExtendedHost,
	pluginID string,
	engine *permissions.Engine,
	fsGuard *sandbox.FilesystemGuard,
	monitor *monitoring.SecurityMonitor,
) *SandboxedHostAdapter {
	return &SandboxedHostAdapter{
		inner:      inner,
		pluginID:   pluginID,
		permEngine: engine,
		fsGuard:    fsGuard,
		monitor:    monitor,
	}
}

func (h *SandboxedHostAdapter) HostVersion() version.SemVer {
	if h.inner != nil {
		return h.inner.HostVersion()
	}
	return version.Current()
}

func (h *SandboxedHostAdapter) Capabilities() []string {
	if h.inner != nil {
		return h.inner.Capabilities()
	}
	return nil
}

func (h *SandboxedHostAdapter) HasCapability(name string) bool {
	if h.inner != nil {
		return h.inner.HasCapability(name)
	}
	return false
}

func (h *SandboxedHostAdapter) Logger() plugin.Logger {
	if h.inner != nil {
		return h.inner.Logger()
	}
	return nil
}

func (h *SandboxedHostAdapter) Events() plugin.EventFacade {
	if h.inner != nil {
		return h.inner.Events()
	}
	return nil
}

func (h *SandboxedHostAdapter) Repository() plugin.RepositoryFacade {
	return &sandboxedRepoFacade{
		host: h,
	}
}

func (h *SandboxedHostAdapter) RepositoryAPI() api.RepositoryAPI {
	return &sandboxedRepoAPI{host: h}
}

func (h *SandboxedHostAdapter) SymbolAPI() api.SymbolAPI {
	return &sandboxedSymbolAPI{host: h}
}

func (h *SandboxedHostAdapter) SearchAPI() api.SearchAPI {
	return &sandboxedSearchAPI{host: h}
}

func (h *SandboxedHostAdapter) GraphAPI() api.GraphAPI {
	return &sandboxedGraphAPI{host: h}
}

func (h *SandboxedHostAdapter) AnalysisAPI() api.AnalysisAPI {
	return &sandboxedAnalysisAPI{host: h}
}

func (h *SandboxedHostAdapter) EventAPI() api.EventAPI {
	return &sandboxedEventAPI{host: h}
}

// checkPermission evaluates authorization for the requested action against the target resource.
func (h *SandboxedHostAdapter) checkPermission(ctx context.Context, action, resource string) error {
	req := permissions.PermissionRequest{
		PluginID: h.pluginID,
		Action:   action,
		Resource: resource,
	}
	return h.permEngine.Check(ctx, req)
}

// -----------------------------------------------------------------------
// Sandboxed Facade Wrappers
// -----------------------------------------------------------------------

type sandboxedRepoFacade struct {
	host *SandboxedHostAdapter
}

func (r *sandboxedRepoFacade) WorkspacePath() string {
	if r.host.fsGuard != nil {
		return r.host.fsGuard.WorkspaceRoot()
	}
	if r.host.inner != nil && r.host.inner.Repository() != nil {
		return r.host.inner.Repository().WorkspacePath()
	}
	return "."
}

func (r *sandboxedRepoFacade) Metadata(ctx context.Context) (*plugin.RepositoryMetadata, error) {
	if err := r.host.checkPermission(ctx, pubsec.PermRepoMetadata, ""); err != nil {
		return nil, err
	}
	if r.host.inner != nil && r.host.inner.Repository() != nil {
		return r.host.inner.Repository().Metadata(ctx)
	}
	return nil, fmt.Errorf("underlying repository metadata service unavailable")
}

func (r *sandboxedRepoFacade) ListFiles(ctx context.Context, prefix string) ([]string, error) {
	if err := r.host.checkPermission(ctx, pubsec.PermRepoRead, prefix); err != nil {
		return nil, err
	}
	if r.host.inner != nil && r.host.inner.Repository() != nil {
		return r.host.inner.Repository().ListFiles(ctx, prefix)
	}
	return nil, fmt.Errorf("underlying repository file listing service unavailable")
}

// -----------------------------------------------------------------------
// Sandboxed API Implementations
// -----------------------------------------------------------------------

type sandboxedRepoAPI struct{ host *SandboxedHostAdapter }

func (s *sandboxedRepoAPI) WorkspacePath() string {
	if s.host.fsGuard != nil {
		return s.host.fsGuard.WorkspaceRoot()
	}
	if s.host.inner != nil && s.host.inner.RepositoryAPI() != nil {
		return s.host.inner.RepositoryAPI().WorkspacePath()
	}
	return "."
}

func (s *sandboxedRepoAPI) Info(ctx context.Context) (*api.RepositoryInfo, error) {
	if err := s.host.checkPermission(ctx, pubsec.PermRepoMetadata, ""); err != nil {
		return nil, err
	}
	if s.host.inner != nil && s.host.inner.RepositoryAPI() != nil {
		return s.host.inner.RepositoryAPI().Info(ctx)
	}
	return nil, fmt.Errorf("repository API unavailable")
}

func (s *sandboxedRepoAPI) Metadata(ctx context.Context) (*api.RepositoryMetadata, error) {
	if err := s.host.checkPermission(ctx, pubsec.PermRepoMetadata, ""); err != nil {
		return nil, err
	}
	if s.host.inner != nil && s.host.inner.RepositoryAPI() != nil {
		return s.host.inner.RepositoryAPI().Metadata(ctx)
	}
	return nil, fmt.Errorf("repository API unavailable")
}

func (s *sandboxedRepoAPI) Statistics(ctx context.Context) (*api.RepositoryStatistics, error) {
	if err := s.host.checkPermission(ctx, pubsec.PermRepoMetadata, ""); err != nil {
		return nil, err
	}
	if s.host.inner != nil && s.host.inner.RepositoryAPI() != nil {
		return s.host.inner.RepositoryAPI().Statistics(ctx)
	}
	return nil, fmt.Errorf("repository API unavailable")
}

func (s *sandboxedRepoAPI) ListFiles(ctx context.Context, prefix string) ([]string, error) {
	if err := s.host.checkPermission(ctx, pubsec.PermRepoRead, prefix); err != nil {
		return nil, err
	}
	if s.host.inner != nil && s.host.inner.RepositoryAPI() != nil {
		return s.host.inner.RepositoryAPI().ListFiles(ctx, prefix)
	}
	return nil, fmt.Errorf("repository API unavailable")
}

func (s *sandboxedRepoAPI) GetFile(ctx context.Context, relPath string) (*api.FileInfo, error) {
	if err := s.host.checkPermission(ctx, pubsec.PermFileRead, relPath); err != nil {
		return nil, err
	}
	// Validate path against filesystem sandbox
	if s.host.fsGuard != nil {
		if _, err := s.host.fsGuard.ValidateRead(relPath); err != nil {
			s.host.monitor.Record(monitoring.SecurityEvent{
				Category: monitoring.CategorySandbox,
				Severity: monitoring.SeverityError,
				PluginID: s.host.pluginID,
				Action:   "file.read",
				Resource: relPath,
				Decision: "VIOLATION",
				Reason:   err.Error(),
			})
			return nil, err
		}
	}
	if s.host.inner != nil && s.host.inner.RepositoryAPI() != nil {
		return s.host.inner.RepositoryAPI().GetFile(ctx, relPath)
	}
	return nil, fmt.Errorf("repository API unavailable")
}

type sandboxedSymbolAPI struct{ host *SandboxedHostAdapter }

func (s *sandboxedSymbolAPI) LookupSymbol(ctx context.Context, symbolIDOrName string) (*api.SymbolInfo, error) {
	if err := s.host.checkPermission(ctx, pubsec.PermRepoRead, symbolIDOrName); err != nil {
		return nil, err
	}
	if s.host.inner != nil && s.host.inner.SymbolAPI() != nil {
		return s.host.inner.SymbolAPI().LookupSymbol(ctx, symbolIDOrName)
	}
	return nil, fmt.Errorf("symbol API unavailable")
}

func (s *sandboxedSymbolAPI) FindSymbols(ctx context.Context, pattern string, pkgScope string, opts api.PaginationOptions) ([]api.SymbolInfo, error) {
	if err := s.host.checkPermission(ctx, pubsec.PermRepoRead, pattern); err != nil {
		return nil, err
	}
	if s.host.inner != nil && s.host.inner.SymbolAPI() != nil {
		return s.host.inner.SymbolAPI().FindSymbols(ctx, pattern, pkgScope, opts)
	}
	return nil, fmt.Errorf("symbol API unavailable")
}

func (s *sandboxedSymbolAPI) SymbolHierarchy(ctx context.Context, symbolID string) (*api.SymbolHierarchyNode, error) {
	if err := s.host.checkPermission(ctx, pubsec.PermRepoRead, symbolID); err != nil {
		return nil, err
	}
	if s.host.inner != nil && s.host.inner.SymbolAPI() != nil {
		return s.host.inner.SymbolAPI().SymbolHierarchy(ctx, symbolID)
	}
	return nil, fmt.Errorf("symbol API unavailable")
}

func (s *sandboxedSymbolAPI) SymbolReferences(ctx context.Context, symbolID string, opts api.PaginationOptions) ([]api.SymbolReference, error) {
	if err := s.host.checkPermission(ctx, pubsec.PermRepoRead, symbolID); err != nil {
		return nil, err
	}
	if s.host.inner != nil && s.host.inner.SymbolAPI() != nil {
		return s.host.inner.SymbolAPI().SymbolReferences(ctx, symbolID, opts)
	}
	return nil, fmt.Errorf("symbol API unavailable")
}

func (s *sandboxedSymbolAPI) SymbolDocumentation(ctx context.Context, symbolID string) (string, error) {
	if err := s.host.checkPermission(ctx, pubsec.PermRepoRead, symbolID); err != nil {
		return "", err
	}
	if s.host.inner != nil && s.host.inner.SymbolAPI() != nil {
		return s.host.inner.SymbolAPI().SymbolDocumentation(ctx, symbolID)
	}
	return "", fmt.Errorf("symbol API unavailable")
}

type sandboxedSearchAPI struct{ host *SandboxedHostAdapter }

func (s *sandboxedSearchAPI) Search(ctx context.Context, q api.SearchQuery) (*api.SearchResult, error) {
	if err := s.host.checkPermission(ctx, pubsec.PermRepoRead, q.Query); err != nil {
		return nil, err
	}
	if s.host.inner != nil && s.host.inner.SearchAPI() != nil {
		return s.host.inner.SearchAPI().Search(ctx, q)
	}
	return nil, fmt.Errorf("search API unavailable")
}

func (s *sandboxedSearchAPI) SearchSymbols(ctx context.Context, query string, opts api.PaginationOptions) (*api.SearchResult, error) {
	if err := s.host.checkPermission(ctx, pubsec.PermRepoRead, query); err != nil {
		return nil, err
	}
	if s.host.inner != nil && s.host.inner.SearchAPI() != nil {
		return s.host.inner.SearchAPI().SearchSymbols(ctx, query, opts)
	}
	return nil, fmt.Errorf("search API unavailable")
}

func (s *sandboxedSearchAPI) SearchPackages(ctx context.Context, query string, opts api.PaginationOptions) (*api.SearchResult, error) {
	if err := s.host.checkPermission(ctx, pubsec.PermRepoRead, query); err != nil {
		return nil, err
	}
	if s.host.inner != nil && s.host.inner.SearchAPI() != nil {
		return s.host.inner.SearchAPI().SearchPackages(ctx, query, opts)
	}
	return nil, fmt.Errorf("search API unavailable")
}

func (s *sandboxedSearchAPI) SearchFiles(ctx context.Context, pattern string, opts api.PaginationOptions) (*api.SearchResult, error) {
	if err := s.host.checkPermission(ctx, pubsec.PermRepoRead, pattern); err != nil {
		return nil, err
	}
	if s.host.inner != nil && s.host.inner.SearchAPI() != nil {
		return s.host.inner.SearchAPI().SearchFiles(ctx, pattern, opts)
	}
	return nil, fmt.Errorf("search API unavailable")
}

func (s *sandboxedSearchAPI) SearchDocs(ctx context.Context, query string, opts api.PaginationOptions) (*api.SearchResult, error) {
	if err := s.host.checkPermission(ctx, pubsec.PermRepoRead, query); err != nil {
		return nil, err
	}
	if s.host.inner != nil && s.host.inner.SearchAPI() != nil {
		return s.host.inner.SearchAPI().SearchDocs(ctx, query, opts)
	}
	return nil, fmt.Errorf("search API unavailable")
}

func (s *sandboxedSearchAPI) SearchConfigs(ctx context.Context, key string, opts api.PaginationOptions) (*api.SearchResult, error) {
	if err := s.host.checkPermission(ctx, pubsec.PermConfigRead, key); err != nil {
		return nil, err
	}
	if s.host.inner != nil && s.host.inner.SearchAPI() != nil {
		return s.host.inner.SearchAPI().SearchConfigs(ctx, key, opts)
	}
	return nil, fmt.Errorf("search API unavailable")
}

type sandboxedGraphAPI struct{ host *SandboxedHostAdapter }

func (s *sandboxedGraphAPI) GraphInfo(ctx context.Context) (int, int, error) {
	if err := s.host.checkPermission(ctx, pubsec.PermRepoRead, ""); err != nil {
		return 0, 0, err
	}
	if s.host.inner != nil && s.host.inner.GraphAPI() != nil {
		return s.host.inner.GraphAPI().GraphInfo(ctx)
	}
	return 0, 0, fmt.Errorf("graph API unavailable")
}

func (s *sandboxedGraphAPI) GetNode(ctx context.Context, nodeID string) (*api.GraphNode, error) {
	if err := s.host.checkPermission(ctx, pubsec.PermRepoRead, nodeID); err != nil {
		return nil, err
	}
	if s.host.inner != nil && s.host.inner.GraphAPI() != nil {
		return s.host.inner.GraphAPI().GetNode(ctx, nodeID)
	}
	return nil, fmt.Errorf("graph API unavailable")
}

func (s *sandboxedGraphAPI) GetRelationship(ctx context.Context, relID string) (*api.GraphRelationship, error) {
	if err := s.host.checkPermission(ctx, pubsec.PermRepoRead, relID); err != nil {
		return nil, err
	}
	if s.host.inner != nil && s.host.inner.GraphAPI() != nil {
		return s.host.inner.GraphAPI().GetRelationship(ctx, relID)
	}
	return nil, fmt.Errorf("graph API unavailable")
}

func (s *sandboxedGraphAPI) TraverseNodes(ctx context.Context, startNodeID string, filter api.GraphFilter) ([]api.GraphNode, error) {
	if err := s.host.checkPermission(ctx, pubsec.PermRepoRead, startNodeID); err != nil {
		return nil, err
	}
	if s.host.inner != nil && s.host.inner.GraphAPI() != nil {
		return s.host.inner.GraphAPI().TraverseNodes(ctx, startNodeID, filter)
	}
	return nil, fmt.Errorf("graph API unavailable")
}

func (s *sandboxedGraphAPI) TraverseRelationships(ctx context.Context, startNodeID string, filter api.GraphFilter) ([]api.GraphRelationship, error) {
	if err := s.host.checkPermission(ctx, pubsec.PermRepoRead, startNodeID); err != nil {
		return nil, err
	}
	if s.host.inner != nil && s.host.inner.GraphAPI() != nil {
		return s.host.inner.GraphAPI().TraverseRelationships(ctx, startNodeID, filter)
	}
	return nil, fmt.Errorf("graph API unavailable")
}

func (s *sandboxedGraphAPI) GetNeighbors(ctx context.Context, nodeID string, direction string, kinds ...string) ([]api.GraphNode, error) {
	if err := s.host.checkPermission(ctx, pubsec.PermRepoRead, nodeID); err != nil {
		return nil, err
	}
	if s.host.inner != nil && s.host.inner.GraphAPI() != nil {
		return s.host.inner.GraphAPI().GetNeighbors(ctx, nodeID, direction, kinds...)
	}
	return nil, fmt.Errorf("graph API unavailable")
}

func (s *sandboxedGraphAPI) ExportGraph(ctx context.Context, filter api.GraphFilter, format api.GraphExportFormat) (*api.GraphExportResult, error) {
	if err := s.host.checkPermission(ctx, pubsec.PermRepoRead, ""); err != nil {
		return nil, err
	}
	if s.host.inner != nil && s.host.inner.GraphAPI() != nil {
		return s.host.inner.GraphAPI().ExportGraph(ctx, filter, format)
	}
	return nil, fmt.Errorf("graph API unavailable")
}

type sandboxedAnalysisAPI struct{ host *SandboxedHostAdapter }

func (s *sandboxedAnalysisAPI) AnalyzeArchitecture(ctx context.Context, scope string) (*api.ArchitectureReport, error) {
	if err := s.host.checkPermission(ctx, pubsec.PermRepoAnalysis, scope); err != nil {
		return nil, err
	}
	if s.host.inner != nil && s.host.inner.AnalysisAPI() != nil {
		return s.host.inner.AnalysisAPI().AnalyzeArchitecture(ctx, scope)
	}
	return nil, fmt.Errorf("analysis API unavailable")
}

func (s *sandboxedAnalysisAPI) AnalyzeDependencies(ctx context.Context, scope string) (*api.DependencyReport, error) {
	if err := s.host.checkPermission(ctx, pubsec.PermRepoAnalysis, scope); err != nil {
		return nil, err
	}
	if s.host.inner != nil && s.host.inner.AnalysisAPI() != nil {
		return s.host.inner.AnalysisAPI().AnalyzeDependencies(ctx, scope)
	}
	return nil, fmt.Errorf("analysis API unavailable")
}

func (s *sandboxedAnalysisAPI) RepositoryHealth(ctx context.Context) (*api.HealthReport, error) {
	if err := s.host.checkPermission(ctx, pubsec.PermRepoAnalysis, ""); err != nil {
		return nil, err
	}
	if s.host.inner != nil && s.host.inner.AnalysisAPI() != nil {
		return s.host.inner.AnalysisAPI().RepositoryHealth(ctx)
	}
	return nil, fmt.Errorf("analysis API unavailable")
}

func (s *sandboxedAnalysisAPI) AnalyzeQuality(ctx context.Context, scope string) (*api.QualityReport, error) {
	if err := s.host.checkPermission(ctx, pubsec.PermRepoAnalysis, scope); err != nil {
		return nil, err
	}
	if s.host.inner != nil && s.host.inner.AnalysisAPI() != nil {
		return s.host.inner.AnalysisAPI().AnalyzeQuality(ctx, scope)
	}
	return nil, fmt.Errorf("analysis API unavailable")
}

func (s *sandboxedAnalysisAPI) AnalyzeConfiguration(ctx context.Context) (*api.ConfigurationReport, error) {
	if err := s.host.checkPermission(ctx, pubsec.PermRepoAnalysis, ""); err != nil {
		return nil, err
	}
	if s.host.inner != nil && s.host.inner.AnalysisAPI() != nil {
		return s.host.inner.AnalysisAPI().AnalyzeConfiguration(ctx)
	}
	return nil, fmt.Errorf("analysis API unavailable")
}

type sandboxedEventAPI struct{ host *SandboxedHostAdapter }

func (s *sandboxedEventAPI) Publish(ctx context.Context, evt api.Event) error {
	if err := s.host.checkPermission(ctx, "events.publish", evt.Type); err != nil {
		return err
	}
	if s.host.inner != nil && s.host.inner.EventAPI() != nil {
		return s.host.inner.EventAPI().Publish(ctx, evt)
	}
	return fmt.Errorf("event API unavailable")
}

func (s *sandboxedEventAPI) Subscribe(ctx context.Context, eventType string, handler api.EventHandler) (api.Subscription, error) {
	if err := s.host.checkPermission(ctx, "events.subscribe", eventType); err != nil {
		return nil, err
	}
	if s.host.inner != nil && s.host.inner.EventAPI() != nil {
		return s.host.inner.EventAPI().Subscribe(ctx, eventType, handler)
	}
	return nil, fmt.Errorf("event API unavailable")
}

func (s *sandboxedEventAPI) SubscribeAll(ctx context.Context, handler api.EventHandler) (api.Subscription, error) {
	if err := s.host.checkPermission(ctx, "events.subscribe", "*"); err != nil {
		return nil, err
	}
	if s.host.inner != nil && s.host.inner.EventAPI() != nil {
		return s.host.inner.EventAPI().SubscribeAll(ctx, handler)
	}
	return nil, fmt.Errorf("event API unavailable")
}

var _ plugin.ExtendedHost = (*SandboxedHostAdapter)(nil)
