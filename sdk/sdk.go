package sdk

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	internalsdk "github.com/unhield/limoxel/internal/capabilities/sdk"
	"github.com/unhield/limoxel/internal/capabilities/sdk/contracts"
	sdkerr "github.com/unhield/limoxel/internal/capabilities/sdk/errors"
	"github.com/unhield/limoxel/internal/capabilities/sdk/lifecycle"
	"github.com/unhield/limoxel/internal/capabilities/sdk/validation"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
	canonversion "github.com/unhield/limoxel/internal/version"
)

// Re-exported contracts for public consumer convenience.
type (
	RepositoryManagementContract = contracts.RepositoryManagementContract
	FileContract                 = contracts.FileContract
	PackageContract              = contracts.PackageContract
	SymbolContract               = contracts.SymbolContract
	SearchContract               = contracts.SearchContract
	GraphContract                = contracts.GraphContract
	AnalysisContract             = contracts.AnalysisContract
	NavigationContract           = contracts.NavigationContract
	ReasoningContract            = contracts.ReasoningContract
	EventContract                = contracts.EventContract
	IntelligenceContract         = contracts.IntelligenceContract
)

// Re-exported contract data transfer objects and models.
type (
	PaginationOptions          = contracts.PaginationOptions
	ContextOptions             = contracts.ContextOptions
	RepositoryInfo             = contracts.RepositoryInfo
	RepositoryMetadata         = contracts.RepositoryMetadata
	RepositoryStatistics       = contracts.RepositoryStatistics
	RepositoryState            = contracts.RepositoryState
	FileInfo                   = contracts.FileInfo
	FileMetadata               = contracts.FileMetadata
	FileIndexStatus            = contracts.FileIndexStatus
	FileRelationship           = contracts.FileRelationship
	FileFilter                 = contracts.FileFilter
	PackageInfo                = contracts.PackageInfo
	PackageStatistics          = contracts.PackageStatistics
	PackageHierarchyNode       = contracts.PackageHierarchyNode
	PackageRelationship        = contracts.PackageRelationship
	PackageFilter              = contracts.PackageFilter
	SymbolKind                 = contracts.SymbolKind
	SymbolLocation             = contracts.SymbolLocation
	SymbolInfo                 = contracts.SymbolInfo
	SymbolReference            = contracts.SymbolReference
	SymbolHierarchyNode        = contracts.SymbolHierarchyNode
	SearchDomain               = contracts.SearchDomain
	SearchQuery                = contracts.SearchQuery
	SearchMatch                = contracts.SearchMatch
	SearchResult               = contracts.SearchResult
	GraphNode                  = contracts.GraphNode
	GraphRelationship          = contracts.GraphRelationship
	GraphFilter                = contracts.GraphFilter
	GraphExportFormat          = contracts.GraphExportFormat
	GraphExportResult          = contracts.GraphExportResult
	ArchitectureAnalysisResult = contracts.ArchitectureAnalysisResult
	DependencyAnalysisResult   = contracts.DependencyAnalysisResult
	RepositoryHealthReport     = contracts.RepositoryHealthReport
	HealthDimensionResult      = contracts.HealthDimensionResult
	CodeQualityReport          = contracts.CodeQualityReport
	ConfigurationReport        = contracts.ConfigurationReport
	Finding                    = contracts.Finding
	AnalysisRequest            = contracts.AnalysisRequest
	AnalysisResult             = contracts.AnalysisResult
	NavigationResult           = contracts.NavigationResult
	ReasoningRequest           = contracts.ReasoningRequest
	ReasoningResult            = contracts.ReasoningResult
	IntelligenceEvent          = contracts.IntelligenceEvent
	DefinitionResult           = contracts.DefinitionResult
	ReferenceResult            = contracts.ReferenceResult
	CallHierarchyNode          = contracts.CallHierarchyNode
	CallHierarchyItem          = contracts.CallHierarchyItem
	NavSymbolHierarchyNode     = contracts.NavSymbolHierarchyNode
	NavigationContextResult    = contracts.NavigationContextResult
	NavigationTarget           = contracts.NavigationTarget
	ImpactResult               = contracts.ImpactResult
	RecommendationResult       = contracts.RecommendationResult
	BreakingChangeResult       = contracts.BreakingChangeResult
	RefactoringResult          = contracts.RefactoringResult
	EngineeringInsight         = contracts.EngineeringInsight
	EventType                  = contracts.EventType
	Event                      = contracts.Event
	SDKEvent                   = contracts.SDKEvent
	EventHandler               = contracts.EventHandler
	Subscription               = contracts.Subscription
)

// Re-exported export format constants.
const (
	ExportFormatJSON     = contracts.ExportFormatJSON
	ExportFormatMermaid  = contracts.ExportFormatMermaid
	ExportFormatGraphviz = contracts.ExportFormatGraphviz
)

// Re-exported event types.
const (
	EventTypeRepositoryOpened   = contracts.EventTypeRepositoryOpened
	EventTypeRepositoryClosed   = contracts.EventTypeRepositoryClosed
	EventTypeRepositoryReloaded = contracts.EventTypeRepositoryReloaded
	EventTypeIndexStarted       = contracts.EventTypeIndexStarted
	EventTypeIndexCompleted     = contracts.EventTypeIndexCompleted
	EventTypeIndexFailed        = contracts.EventTypeIndexFailed
	EventTypeAnalysisStarted    = contracts.EventTypeAnalysisStarted
	EventTypeAnalysisCompleted  = contracts.EventTypeAnalysisCompleted
	EventTypePluginLoaded       = contracts.EventTypePluginLoaded
	EventTypePluginUnloaded     = contracts.EventTypePluginUnloaded
	EventTypeSDKInitialized     = contracts.EventTypeSDKInitialized
	EventTypeSDKClosed          = contracts.EventTypeSDKClosed
)

// Re-exported version and error types.
type (
	SemVer           = version.SemVer
	ReleaseKind      = version.ReleaseKind
	SDKError         = sdkerr.SDKError
	SemanticCategory = sdkerr.SemanticCategory
)

// Re-exported error category constants.
const (
	CategoryInvalidInput       = sdkerr.CategoryInvalidInput
	CategoryNotFound           = sdkerr.CategoryNotFound
	CategoryUnsupported        = sdkerr.CategoryUnsupported
	CategoryInvalidState       = sdkerr.CategoryInvalidState
	CategoryLifecycleViolation = sdkerr.CategoryLifecycleViolation
	CategoryUnavailable        = sdkerr.CategoryUnavailable
	CategoryInternal           = sdkerr.CategoryInternal
)

// Option represents a functional configuration option for initializing a Limoxel Client.
type Option func(*options)

type options struct {
	workspaceRoot string
	customVersion *version.SemVer
}

// WithWorkspace configures the root repository workspace path for the SDK client.
func WithWorkspace(path string) Option {
	return func(o *options) {
		clean := filepath.Clean(strings.TrimSpace(path))
		if clean == "" || clean == "." {
			clean = "."
		}
		o.workspaceRoot = clean
	}
}

// WithCustomVersion sets a custom semantic version (primarily used in testing and migration evaluation).
func WithCustomVersion(sv version.SemVer) Option {
	return func(o *options) {
		o.customVersion = &sv
	}
}

// Client is the primary entrypoint and coordinator for interacting with Limoxel SDK capabilities.
// It provides thread-safe access to Core Repository and Intelligence capabilities.
type Client struct {
	mu       sync.RWMutex
	internal *internalsdk.SDK
	closed   bool
}

// New initializes a new Limoxel SDK Client instance with the provided options.
func New(opts ...Option) (*Client, error) {
	cfg := &options{
		workspaceRoot: ".",
	}
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}

	var internalOpts []internalsdk.Option
	if cfg.workspaceRoot != "" {
		internalOpts = append(internalOpts, internalsdk.WithWorkspace(cfg.workspaceRoot))
	}
	if cfg.customVersion != nil {
		internalOpts = append(internalOpts, internalsdk.WithCustomVersion(*cfg.customVersion))
	}

	sdkInstance, err := internalsdk.New(internalOpts...)
	if err != nil {
		return nil, err
	}

	return &Client{
		internal: sdkInstance,
	}, nil
}

// Workspace returns the active workspace directory configured for this client.
func (c *Client) Workspace() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.internal == nil {
		return ""
	}
	return c.internal.Workspace()
}

// Version returns the current SemVer of the SDK.
func (c *Client) Version() version.SemVer {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.internal == nil {
		return version.Current()
	}
	return c.internal.Version()
}

// VersionString returns the canonical version string (e.g. "1.4.0").
func VersionString() string {
	return canonversion.Version
}

// Repository returns the Core Repository Management capability contract.
func (c *Client) Repository() contracts.RepositoryManagementContract {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.internal.Repository()
}

// Files returns the Core File inspection and querying capability contract.
func (c *Client) Files() contracts.FileContract {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.internal.Files()
}

// Packages returns the Core Package discovery and dependency capability contract.
func (c *Client) Packages() contracts.PackageContract {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.internal.Packages()
}

// Symbols returns the Core Symbol resolution and querying capability contract.
func (c *Client) Symbols() contracts.SymbolContract {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.internal.Symbols()
}

// Search returns the Core Multi-Entity Search capability contract.
func (c *Client) Search() contracts.SearchContract {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.internal.Search()
}

// Graph returns the Intelligence Knowledge Graph capability contract.
func (c *Client) Graph() contracts.GraphContract {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.internal.Graph()
}

// Analysis returns the Intelligence Code Analysis & Health capability contract.
func (c *Client) Analysis() contracts.AnalysisContract {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.internal.Analysis()
}

// Navigation returns the Intelligence Semantic Navigation capability contract.
func (c *Client) Navigation() contracts.NavigationContract {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.internal.Navigation()
}

// Reasoning returns the Intelligence Impact & Refactoring Reasoning capability contract.
func (c *Client) Reasoning() contracts.ReasoningContract {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.internal.Reasoning()
}

// Events returns the Intelligence Event Streaming & Lifecycle capability contract.
func (c *Client) Events() contracts.EventContract {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.internal.Events()
}

// Intelligence returns the unified facade across all engineering intelligence capabilities.
func (c *Client) Intelligence() contracts.IntelligenceContract {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.internal.Intelligence()
}

// Registry returns the API Lifecycle Registry for introspection.
func (c *Client) Registry() *lifecycle.Registry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.internal.Registry()
}

// Validator returns the API Contract Validator.
func (c *Client) Validator() *validation.Validator {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.internal.Validator()
}

// Close gracefully terminates all underlying indexers, background workers, and caches.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	if c.internal != nil {
		return c.internal.Close()
	}
	return nil
}

// OpenWorkspace is a convenience helper that initializes a client and opens the repository workspace.
func OpenWorkspace(ctx context.Context, path string) (*Client, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	clean := filepath.Clean(strings.TrimSpace(path))
	if clean == "" {
		clean = "."
	}
	if _, err := os.Stat(clean); err != nil {
		return nil, fmt.Errorf("invalid workspace path %q: %w", path, err)
	}
	client, err := New(WithWorkspace(clean))
	if err != nil {
		return nil, err
	}
	if _, err := client.Repository().Open(ctx, clean); err != nil {
		_ = client.Close()
		return nil, err
	}
	return client, nil
}
