package adapter

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/sdk/contracts"
	"github.com/unhield/limoxel/plugin"
	"github.com/unhield/limoxel/plugin/api"
	"github.com/unhield/limoxel/sdk"
)

// ExtendedHostAdapter decorates HostAdapter with full access to Limoxel SDK capability APIs.
type ExtendedHostAdapter struct {
	*HostAdapter
	bridge *SDKBridge
}

// NewExtendedHostAdapter initializes an ExtendedHostAdapter backed by the Limoxel SDK client.
func NewExtendedHostAdapter(workspaceRoot string, client *sdk.Client, customCaps ...string) (*ExtendedHostAdapter, error) {
	wsClean := filepath.Clean(workspaceRoot)
	if wsClean == "" || wsClean == "." {
		wsClean = "."
	}

	base := NewHostAdapter(wsClean, append(customCaps,
		"repository.api",
		"symbols.api",
		"search.api",
		"graph.api",
		"analysis.api",
		"events.api",
	)...)

	if client == nil {
		var err error
		client, err = sdk.New(sdk.WithWorkspace(wsClean))
		if err != nil {
			return nil, fmt.Errorf("failed to initialize sdk client for extended host: %w", err)
		}
	}

	bridge := NewSDKBridge(wsClean, client, base.Events())

	return &ExtendedHostAdapter{
		HostAdapter: base,
		bridge:      bridge,
	}, nil
}

func (e *ExtendedHostAdapter) RepositoryAPI() api.RepositoryAPI { return e.bridge.Repository() }
func (e *ExtendedHostAdapter) SymbolAPI() api.SymbolAPI         { return e.bridge.Symbols() }
func (e *ExtendedHostAdapter) SearchAPI() api.SearchAPI         { return e.bridge.Search() }
func (e *ExtendedHostAdapter) GraphAPI() api.GraphAPI           { return e.bridge.Graph() }
func (e *ExtendedHostAdapter) AnalysisAPI() api.AnalysisAPI     { return e.bridge.Analysis() }
func (e *ExtendedHostAdapter) EventAPI() api.EventAPI           { return e.bridge.Events() }

var _ plugin.ExtendedHost = (*ExtendedHostAdapter)(nil)

// SDKBridge delegates plugin API calls directly to underlying Limoxel SDK contracts.
type SDKBridge struct {
	mu            sync.RWMutex
	workspaceRoot string
	client        *sdk.Client
	eventsFacade  plugin.EventFacade

	repoAPI     api.RepositoryAPI
	symbolAPI   api.SymbolAPI
	searchAPI   api.SearchAPI
	graphAPI    api.GraphAPI
	analysisAPI api.AnalysisAPI
	eventAPI    api.EventAPI
}

// NewSDKBridge constructs an initialized bridge delegating to the provided SDK client.
func NewSDKBridge(workspaceRoot string, client *sdk.Client, eventsFacade plugin.EventFacade) *SDKBridge {
	b := &SDKBridge{
		workspaceRoot: workspaceRoot,
		client:        client,
		eventsFacade:  eventsFacade,
	}

	b.repoAPI = &bridgeRepoAPI{b: b}
	b.symbolAPI = &bridgeSymbolAPI{b: b}
	b.searchAPI = &bridgeSearchAPI{b: b}
	b.graphAPI = &bridgeGraphAPI{b: b}
	b.analysisAPI = &bridgeAnalysisAPI{b: b}
	b.eventAPI = &bridgeEventAPI{b: b}

	return b
}

func (b *SDKBridge) Repository() api.RepositoryAPI { return b.repoAPI }
func (b *SDKBridge) Symbols() api.SymbolAPI        { return b.symbolAPI }
func (b *SDKBridge) Search() api.SearchAPI         { return b.searchAPI }
func (b *SDKBridge) Graph() api.GraphAPI           { return b.graphAPI }
func (b *SDKBridge) Analysis() api.AnalysisAPI     { return b.analysisAPI }
func (b *SDKBridge) Events() api.EventAPI          { return b.eventAPI }

// Repository Bridge
type bridgeRepoAPI struct{ b *SDKBridge }

func (r *bridgeRepoAPI) WorkspacePath() string { return r.b.workspaceRoot }

func (r *bridgeRepoAPI) Info(ctx context.Context) (*api.RepositoryInfo, error) {
	if r.b.client == nil {
		return nil, plugin.ErrCapabilityUnavailable
	}
	info, err := r.b.client.Repository().Info(ctx)
	if err != nil {
		return nil, err
	}
	return &api.RepositoryInfo{
		Name:          info.Name,
		Owner:         info.Owner,
		RootPath:      info.RootPath,
		DefaultBranch: info.DefaultBranch,
		CurrentBranch: info.CurrentBranch,
		IsGit:         info.IsGit,
		State:         string(info.State),
		Languages:     info.Languages,
		Capabilities:  info.Capabilities,
	}, nil
}

func (r *bridgeRepoAPI) Metadata(ctx context.Context) (*api.RepositoryMetadata, error) {
	if r.b.client == nil {
		return nil, plugin.ErrCapabilityUnavailable
	}
	meta, err := r.b.client.Repository().Metadata(ctx)
	if err != nil {
		return nil, err
	}
	return &api.RepositoryMetadata{
		Name:        meta.Name,
		Description: meta.Description,
		Version:     meta.Version,
		License:     meta.License,
		VCS:         meta.VCS,
		CommitHash:  meta.CommitHash,
		CommitDate:  meta.CommitDate,
		Authors:     meta.Authors,
		Properties:  meta.Properties,
	}, nil
}

func (r *bridgeRepoAPI) Statistics(ctx context.Context) (*api.RepositoryStatistics, error) {
	if r.b.client == nil {
		return nil, plugin.ErrCapabilityUnavailable
	}
	stats, err := r.b.client.Repository().Statistics(ctx)
	if err != nil {
		return nil, err
	}
	return &api.RepositoryStatistics{
		TotalFiles:         stats.TotalFiles,
		TotalDirectories:   stats.TotalDirectories,
		TotalPackages:      stats.TotalPackages,
		TotalSymbols:       stats.TotalSymbols,
		TotalRelationships: stats.TotalRelationships,
		TotalDependencies:  stats.TotalDependencies,
	}, nil
}

func (r *bridgeRepoAPI) ListFiles(ctx context.Context, prefix string) ([]string, error) {
	if r.b.client == nil {
		return nil, plugin.ErrCapabilityUnavailable
	}
	files, err := r.b.client.Files().DiscoverFiles(ctx, contracts.FileFilter{
		Pattern: prefix,
	}, contracts.PaginationOptions{})
	if err != nil {
		return nil, err
	}
	out := make([]string, len(files))
	for i, f := range files {
		out[i] = f.Path
	}
	return out, nil
}

func (r *bridgeRepoAPI) GetFile(ctx context.Context, relPath string) (*api.FileInfo, error) {
	if r.b.client == nil {
		return nil, plugin.ErrCapabilityUnavailable
	}
	f, err := r.b.client.Files().LookupFile(ctx, relPath)
	if err != nil {
		return nil, err
	}
	return &api.FileInfo{
		Path:      f.Path,
		Size:      f.Size,
		Extension: f.Extension,
		Language:  f.Language,
	}, nil
}

// Symbol Bridge
type bridgeSymbolAPI struct{ b *SDKBridge }

func (s *bridgeSymbolAPI) LookupSymbol(ctx context.Context, symbolIDOrName string) (*api.SymbolInfo, error) {
	if s.b.client == nil {
		return nil, plugin.ErrCapabilityUnavailable
	}
	sym, err := s.b.client.Symbols().LookupSymbol(ctx, symbolIDOrName)
	if err != nil {
		return nil, err
	}
	return mapSymbolInfo(sym), nil
}

func (s *bridgeSymbolAPI) FindSymbols(ctx context.Context, pattern string, pkgScope string, opts api.PaginationOptions) ([]api.SymbolInfo, error) {
	if s.b.client == nil {
		return nil, plugin.ErrCapabilityUnavailable
	}
	// Use search contract for symbol queries across scope
	res, err := s.b.client.Search().SearchSymbols(ctx, pattern, contracts.PaginationOptions{
		Offset: opts.Offset,
		Limit:  opts.Limit,
	})
	if err != nil {
		return nil, err
	}
	var out []api.SymbolInfo
	for _, m := range res.Matches {
		out = append(out, api.SymbolInfo{
			ID:      m.Name,
			Name:    m.Name,
			Package: m.Package,
			Kind:    api.SymbolKind(m.Domain),
		})
	}
	return out, nil
}

func (s *bridgeSymbolAPI) SymbolHierarchy(ctx context.Context, symbolID string) (*api.SymbolHierarchyNode, error) {
	if s.b.client == nil {
		return nil, plugin.ErrCapabilityUnavailable
	}
	node, err := s.b.client.Symbols().SymbolHierarchy(ctx, symbolID)
	if err != nil {
		return nil, err
	}
	return mapSymbolHierarchy(node), nil
}

func (s *bridgeSymbolAPI) SymbolReferences(ctx context.Context, symbolID string, opts api.PaginationOptions) ([]api.SymbolReference, error) {
	if s.b.client == nil {
		return nil, plugin.ErrCapabilityUnavailable
	}
	refs, err := s.b.client.Symbols().SymbolReferences(ctx, symbolID, contracts.PaginationOptions{
		Offset: opts.Offset,
		Limit:  opts.Limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]api.SymbolReference, len(refs))
	for i, r := range refs {
		out[i] = api.SymbolReference{
			SourceSymbolID: r.SourceSymbolID,
			SourceFile:     r.SourceFile,
			SourceLine:     r.SourceLine,
			ReferenceKind:  r.ReferenceKind,
			Evidence:       r.Evidence,
		}
	}
	return out, nil
}

func (s *bridgeSymbolAPI) SymbolDocumentation(ctx context.Context, symbolID string) (string, error) {
	if s.b.client == nil {
		return "", plugin.ErrCapabilityUnavailable
	}
	return s.b.client.Symbols().SymbolDocumentation(ctx, symbolID)
}

func mapSymbolInfo(s *contracts.SymbolInfo) *api.SymbolInfo {
	if s == nil {
		return nil
	}
	return &api.SymbolInfo{
		ID:      s.ID,
		Name:    s.Name,
		Package: s.Package,
		Kind:    api.SymbolKind(s.Kind),
		Location: api.SymbolLocation{
			FilePath:    s.Location.FilePath,
			StartLine:   s.Location.StartLine,
			EndLine:     s.Location.EndLine,
			StartColumn: s.Location.StartColumn,
			EndColumn:   s.Location.EndColumn,
		},
		Signature:  s.Signature,
		IsExported: s.IsExported,
		DocComment: s.DocComment,
	}
}

func mapSymbolHierarchy(n *contracts.SymbolHierarchyNode) *api.SymbolHierarchyNode {
	if n == nil {
		return nil
	}
	out := &api.SymbolHierarchyNode{
		Symbol: *mapSymbolInfo(&n.Symbol),
	}
	for _, c := range n.Children {
		if mapped := mapSymbolHierarchy(&c); mapped != nil {
			out.Children = append(out.Children, *mapped)
		}
	}
	return out
}

// Search Bridge
type bridgeSearchAPI struct{ b *SDKBridge }

func (s *bridgeSearchAPI) Search(ctx context.Context, q api.SearchQuery) (*api.SearchResult, error) {
	if s.b.client == nil {
		return nil, plugin.ErrCapabilityUnavailable
	}
	res, err := s.b.client.Search().Search(ctx, contracts.SearchQuery{
		Query:   q.Query,
		Domain:  contracts.SearchDomain(q.Domain),
		Filters: q.Filters,
		Pagination: contracts.PaginationOptions{
			Offset: q.Pagination.Offset,
			Limit:  q.Pagination.Limit,
		},
	})
	if err != nil {
		return nil, err
	}
	return mapSearchResult(res), nil
}

func (s *bridgeSearchAPI) SearchSymbols(ctx context.Context, query string, opts api.PaginationOptions) (*api.SearchResult, error) {
	if s.b.client == nil {
		return nil, plugin.ErrCapabilityUnavailable
	}
	res, err := s.b.client.Search().SearchSymbols(ctx, query, contracts.PaginationOptions{
		Offset: opts.Offset,
		Limit:  opts.Limit,
	})
	if err != nil {
		return nil, err
	}
	return mapSearchResult(res), nil
}

func (s *bridgeSearchAPI) SearchPackages(ctx context.Context, query string, opts api.PaginationOptions) (*api.SearchResult, error) {
	if s.b.client == nil {
		return nil, plugin.ErrCapabilityUnavailable
	}
	res, err := s.b.client.Search().SearchPackages(ctx, query, contracts.PaginationOptions{
		Offset: opts.Offset,
		Limit:  opts.Limit,
	})
	if err != nil {
		return nil, err
	}
	return mapSearchResult(res), nil
}

func (s *bridgeSearchAPI) SearchFiles(ctx context.Context, pattern string, opts api.PaginationOptions) (*api.SearchResult, error) {
	if s.b.client == nil {
		return nil, plugin.ErrCapabilityUnavailable
	}
	res, err := s.b.client.Search().SearchFiles(ctx, pattern, contracts.PaginationOptions{
		Offset: opts.Offset,
		Limit:  opts.Limit,
	})
	if err != nil {
		return nil, err
	}
	return mapSearchResult(res), nil
}

func (s *bridgeSearchAPI) SearchDocs(ctx context.Context, query string, opts api.PaginationOptions) (*api.SearchResult, error) {
	if s.b.client == nil {
		return nil, plugin.ErrCapabilityUnavailable
	}
	res, err := s.b.client.Search().SearchDocs(ctx, query, contracts.PaginationOptions{
		Offset: opts.Offset,
		Limit:  opts.Limit,
	})
	if err != nil {
		return nil, err
	}
	return mapSearchResult(res), nil
}

func (s *bridgeSearchAPI) SearchConfigs(ctx context.Context, key string, opts api.PaginationOptions) (*api.SearchResult, error) {
	if s.b.client == nil {
		return nil, plugin.ErrCapabilityUnavailable
	}
	res, err := s.b.client.Search().SearchConfigs(ctx, key, contracts.PaginationOptions{
		Offset: opts.Offset,
		Limit:  opts.Limit,
	})
	if err != nil {
		return nil, err
	}
	return mapSearchResult(res), nil
}

func mapSearchResult(r *contracts.SearchResult) *api.SearchResult {
	if r == nil {
		return nil
	}
	matches := make([]api.SearchMatch, len(r.Matches))
	for i, m := range r.Matches {
		matches[i] = api.SearchMatch{
			Domain:   api.SearchDomain(m.Domain),
			Name:     m.Name,
			Package:  m.Package,
			Location: m.Location,
			Snippet:  m.Snippet,
			Score:    m.Score,
		}
	}
	return &api.SearchResult{
		Query:        r.Query,
		Domain:       api.SearchDomain(r.Domain),
		TotalMatches: r.TotalMatches,
		Matches:      matches,
		DurationMs:   r.DurationMs,
	}
}

// Graph Bridge
type bridgeGraphAPI struct{ b *SDKBridge }

func (g *bridgeGraphAPI) GraphInfo(ctx context.Context) (int, int, error) {
	if g.b.client == nil {
		return 0, 0, plugin.ErrCapabilityUnavailable
	}
	return g.b.client.Graph().GraphInfo(ctx)
}

func (g *bridgeGraphAPI) GetNode(ctx context.Context, nodeID string) (*api.GraphNode, error) {
	if g.b.client == nil {
		return nil, plugin.ErrCapabilityUnavailable
	}
	node, err := g.b.client.Graph().GetNode(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	return &api.GraphNode{
		ID:         node.ID,
		Kind:       node.Kind,
		Name:       node.Name,
		Package:    node.Package,
		Location:   node.Location,
		Properties: node.Properties,
	}, nil
}

func (g *bridgeGraphAPI) GetRelationship(ctx context.Context, relID string) (*api.GraphRelationship, error) {
	if g.b.client == nil {
		return nil, plugin.ErrCapabilityUnavailable
	}
	rel, err := g.b.client.Graph().GetRelationship(ctx, relID)
	if err != nil {
		return nil, err
	}
	return &api.GraphRelationship{
		ID:         rel.ID,
		SourceID:   rel.SourceID,
		TargetID:   rel.TargetID,
		Kind:       rel.Kind,
		Evidence:   rel.Evidence,
		Properties: rel.Properties,
	}, nil
}

func (g *bridgeGraphAPI) TraverseNodes(ctx context.Context, startNodeID string, filter api.GraphFilter) ([]api.GraphNode, error) {
	if g.b.client == nil {
		return nil, plugin.ErrCapabilityUnavailable
	}
	nodes, err := g.b.client.Graph().TraverseNodes(ctx, startNodeID, contracts.GraphFilter{
		EntityTypes:       filter.EntityTypes,
		RelationshipKinds: filter.RelationshipKinds,
		PackageScope:      filter.PackageScope,
		MaxDepth:          filter.MaxDepth,
	})
	if err != nil {
		return nil, err
	}
	out := make([]api.GraphNode, len(nodes))
	for i, n := range nodes {
		out[i] = api.GraphNode{
			ID:         n.ID,
			Kind:       n.Kind,
			Name:       n.Name,
			Package:    n.Package,
			Location:   n.Location,
			Properties: n.Properties,
		}
	}
	return out, nil
}

func (g *bridgeGraphAPI) TraverseRelationships(ctx context.Context, startNodeID string, filter api.GraphFilter) ([]api.GraphRelationship, error) {
	if g.b.client == nil {
		return nil, plugin.ErrCapabilityUnavailable
	}
	rels, err := g.b.client.Graph().TraverseRelationships(ctx, startNodeID, contracts.GraphFilter{
		EntityTypes:       filter.EntityTypes,
		RelationshipKinds: filter.RelationshipKinds,
		PackageScope:      filter.PackageScope,
		MaxDepth:          filter.MaxDepth,
	})
	if err != nil {
		return nil, err
	}
	out := make([]api.GraphRelationship, len(rels))
	for i, r := range rels {
		out[i] = api.GraphRelationship{
			ID:         r.ID,
			SourceID:   r.SourceID,
			TargetID:   r.TargetID,
			Kind:       r.Kind,
			Evidence:   r.Evidence,
			Properties: r.Properties,
		}
	}
	return out, nil
}

func (g *bridgeGraphAPI) GetNeighbors(ctx context.Context, nodeID string, direction string, kinds ...string) ([]api.GraphNode, error) {
	if g.b.client == nil {
		return nil, plugin.ErrCapabilityUnavailable
	}
	nodes, err := g.b.client.Graph().GetNeighbors(ctx, nodeID, direction, kinds...)
	if err != nil {
		return nil, err
	}
	out := make([]api.GraphNode, len(nodes))
	for i, n := range nodes {
		out[i] = api.GraphNode{
			ID:         n.ID,
			Kind:       n.Kind,
			Name:       n.Name,
			Package:    n.Package,
			Location:   n.Location,
			Properties: n.Properties,
		}
	}
	return out, nil
}

func (g *bridgeGraphAPI) ExportGraph(ctx context.Context, filter api.GraphFilter, format api.GraphExportFormat) (*api.GraphExportResult, error) {
	if g.b.client == nil {
		return nil, plugin.ErrCapabilityUnavailable
	}
	res, err := g.b.client.Graph().ExportGraph(ctx, contracts.GraphFilter{
		EntityTypes:       filter.EntityTypes,
		RelationshipKinds: filter.RelationshipKinds,
		PackageScope:      filter.PackageScope,
		MaxDepth:          filter.MaxDepth,
	}, contracts.GraphExportFormat(format))
	if err != nil {
		return nil, err
	}
	return &api.GraphExportResult{
		Format:    api.GraphExportFormat(res.Format),
		Content:   res.Content,
		NodeCount: res.NodeCount,
		EdgeCount: res.EdgeCount,
	}, nil
}

// Analysis Bridge
type bridgeAnalysisAPI struct{ b *SDKBridge }

func (a *bridgeAnalysisAPI) AnalyzeArchitecture(ctx context.Context, scope string) (*api.ArchitectureReport, error) {
	if a.b.client == nil {
		return nil, plugin.ErrCapabilityUnavailable
	}
	rep, err := a.b.client.Analysis().AnalyzeArchitecture(ctx, scope)
	if err != nil {
		return nil, err
	}
	return &api.ArchitectureReport{
		TotalComponents: rep.TotalComponents,
		LayerCount:      rep.LayerCount,
		Violations:      mapFindings(rep.Violations),
		Findings:        mapFindings(rep.Findings),
		Metrics:         rep.Metrics,
	}, nil
}

func (a *bridgeAnalysisAPI) AnalyzeDependencies(ctx context.Context, scope string) (*api.DependencyReport, error) {
	if a.b.client == nil {
		return nil, plugin.ErrCapabilityUnavailable
	}
	rep, err := a.b.client.Analysis().AnalyzeDependencies(ctx, scope)
	if err != nil {
		return nil, err
	}
	return &api.DependencyReport{
		TotalDependencies:      rep.TotalDependencies,
		DirectDependencies:     rep.DirectDependencies,
		TransitiveDependencies: rep.TransitiveDependencies,
		CircularDependencies:   rep.CircularDependencies,
		Findings:               mapFindings(rep.Findings),
		Metrics:                rep.Metrics,
	}, nil
}

func (a *bridgeAnalysisAPI) RepositoryHealth(ctx context.Context) (*api.HealthReport, error) {
	if a.b.client == nil {
		return nil, plugin.ErrCapabilityUnavailable
	}
	rep, err := a.b.client.Analysis().RepositoryHealth(ctx)
	if err != nil {
		return nil, err
	}
	dims := make([]api.HealthDimension, len(rep.Dimensions))
	for i, d := range rep.Dimensions {
		dims[i] = api.HealthDimension{
			Name:       d.Name,
			Score:      d.Score,
			Confidence: d.Confidence,
			Coverage:   d.Coverage,
			Weight:     d.Weight,
			Metrics:    d.Metrics,
		}
	}
	return &api.HealthReport{
		OverallScore: rep.OverallScore,
		Grade:        rep.Grade,
		Status:       rep.Status,
		Dimensions:   dims,
		Findings:     mapFindings(rep.Findings),
	}, nil
}

func (a *bridgeAnalysisAPI) AnalyzeQuality(ctx context.Context, scope string) (*api.QualityReport, error) {
	if a.b.client == nil {
		return nil, plugin.ErrCapabilityUnavailable
	}
	rep, err := a.b.client.Analysis().AnalyzeQuality(ctx, scope)
	if err != nil {
		return nil, err
	}
	return &api.QualityReport{
		MaintainabilityScore: rep.MaintainabilityScore,
		ComplexityScore:      rep.ComplexityScore,
		TestabilityScore:     rep.TestabilityScore,
		TotalIssues:          rep.TotalIssues,
		Findings:             mapFindings(rep.Findings),
		Metrics:              rep.Metrics,
	}, nil
}

func (a *bridgeAnalysisAPI) AnalyzeConfiguration(ctx context.Context) (*api.ConfigurationReport, error) {
	if a.b.client == nil {
		return nil, plugin.ErrCapabilityUnavailable
	}
	rep, err := a.b.client.Analysis().AnalyzeConfiguration(ctx)
	if err != nil {
		return nil, err
	}
	return &api.ConfigurationReport{
		TotalConfigs:  rep.TotalConfigs,
		ValidConfigs:  rep.ValidConfigs,
		DriftWarnings: rep.DriftWarnings,
		Findings:      mapFindings(rep.Findings),
		Metrics:       rep.Metrics,
	}, nil
}

func mapFindings(fs []contracts.Finding) []api.Finding {
	out := make([]api.Finding, len(fs))
	for i, f := range fs {
		out[i] = api.Finding{
			RuleID:      f.RuleID,
			Severity:    f.Severity,
			Message:     f.Message,
			File:        f.Location,
			Remediation: f.Recommendation,
		}
	}
	return out
}

// Event Bridge
type bridgeEventAPI struct{ b *SDKBridge }

func (e *bridgeEventAPI) Publish(ctx context.Context, evt api.Event) error {
	if e.b.eventsFacade != nil {
		return e.b.eventsFacade.Publish(ctx, evt.Type, []byte(fmt.Sprintf("%v", evt.Payload)))
	}
	return plugin.ErrCapabilityUnavailable
}

func (e *bridgeEventAPI) Subscribe(ctx context.Context, eventType string, handler api.EventHandler) (api.Subscription, error) {
	if e.b.eventsFacade == nil {
		return nil, plugin.ErrCapabilityUnavailable
	}
	sub, err := e.b.eventsFacade.Subscribe(eventType, func(c context.Context, topic string, payload []byte) error {
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
	return &bridgeSubscriptionWrapper{sub: sub, id: fmt.Sprintf("sub-%s", eventType)}, nil
}

func (e *bridgeEventAPI) SubscribeAll(ctx context.Context, handler api.EventHandler) (api.Subscription, error) {
	return e.Subscribe(ctx, "*", handler)
}

type bridgeSubscriptionWrapper struct {
	sub plugin.Subscription
	id  string
}

func (w *bridgeSubscriptionWrapper) ID() string {
	return w.id
}

func (w *bridgeSubscriptionWrapper) Unsubscribe() error {
	if w.sub != nil {
		return w.sub.Unsubscribe()
	}
	return nil
}
