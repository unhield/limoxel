# SDK & Public API User Manual

Project  : Limoxel  
Category : SDK & Public API  
Document : SDK & Public API User Manual  
Version  : 1.0  
Author   : Raj Joshi  

---

# Overview

The Limoxel SDK enables external applications, developer tools, IDE extensions, CI/CD pipelines, and automated agents to interact with Limoxel's repository analysis and engineering intelligence engines.

By exposing stable, strongly-typed Go interfaces and data transfer objects, the SDK allows external software to:
- Open and inspect repository workspaces.
- Discover and query files, packages, and code symbols.
- Execute structured search across multiple codebase domains.
- Traverse and export knowledge graphs (in JSON, Mermaid, and Graphviz formats).
- Perform multidimensional architecture, dependency, health, and quality analyses.
- Navigate definitions, references, call hierarchies, and symbol relationships.
- Obtain deterministic engineering insights, impact assessments, and refactoring advice.
- Stream and subscribe to asynchronous repository and lifecycle events.
- Validate version upgrades and track API deprecation schedules.

All public capabilities are accessible through the canonical Go package:

```go
import "github.com/unhield/limoxel/sdk"
```

---

# What the SDK Provides

The SDK surface is divided into three primary functional tiers:

```
┌────────────────────────────────────────────────────────────────────────┐
│                        PUBLIC SDK CLIENT ENTRYPOINT                    │
│                           sdk.OpenWorkspace()                          │
└──────────────┬─────────────────────────┬──────────────────────┬────────┘
               │                         │                      │
               ▼                         ▼                      ▼
┌──────────────────────────┐ ┌────────────────────────┐ ┌────────────────┐
│         CORE SDK         │ │    INTELLIGENCE SDK    │ │ COMPATIBILITY  │
├──────────────────────────┤ ├────────────────────────┤ ├────────────────┤
│ • Repository Management  │ │ • Knowledge Graph      │ │ • SemVer & Rel │
│ • File Discovery & Meta  │ │ • Multi-Analysis       │ │ • Deprecations │
│ • Package Inspection     │ │ • Code Navigation      │ │ • Upgrades     │
│ • Symbol Resolution      │ │ • Deterministic Reason │ │ • Migration    │
│ • Multi-Domain Search    │ │ • Event Streaming      │ └────────────────┘
└──────────────────────────┘ └────────────────────────┘
```

1. **Core SDK**: Foundational repository access, file discovery, package hierarchies, symbol extraction, and multi-domain search.
2. **Intelligence SDK**: Advanced structural graph querying, deep multi-dimensional analysis, code navigation, deterministic reasoning, and real-time event streaming.
3. **Unified Intelligence Access**: A high-level facade providing simplified request-driven access to analysis, navigation, reasoning, and event pipelines.
4. **Compatibility Framework**: Developer utilities located under `sdk/compatibility` for evaluating upgrade safety, verifying semantic versions, and tracking deprecation records.

---

# Getting Started

### Prerequisites
- Go `1.26.5` or later (tested with Go `1.26.5`).
- A local code repository or directory workspace.

### Quick Start Example
The following minimal program initializes an SDK client, inspects repository statistics, and searches for code symbols:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/unhield/limoxel/sdk"
)

func main() {
	// Create context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. Open repository workspace
	client, err := sdk.OpenWorkspace(ctx, ".")
	if err != nil {
		log.Fatalf("failed to open workspace: %v", err)
	}
	defer client.Close()

	// 2. Query Repository Statistics
	stats, err := client.Repository().Statistics(ctx)
	if err != nil {
		log.Fatalf("failed to fetch stats: %v", err)
	}
	fmt.Printf("Repository Stats: Files=%d, Packages=%d, Symbols=%d\n",
		stats.TotalFiles, stats.TotalPackages, stats.TotalSymbols)

	// 3. Search Symbols
	results, err := client.Search().SearchSymbols(ctx, "Execute", sdk.PaginationOptions{Limit: 5})
	if err != nil {
		log.Fatalf("search failed: %v", err)
	}
	fmt.Printf("Found %d matches for 'Execute':\n", results.TotalMatches)
	for _, match := range results.Matches {
		fmt.Printf(" - [%s] %s (%s:%d)\n", match.Kind, match.Name, match.Path, match.Line)
	}
}
```

---

# SDK Client

The `*sdk.Client` struct is the primary entrypoint coordinating all capability contracts. It is thread-safe and manages repository session state.

### Client Initialization & Construction

#### `sdk.OpenWorkspace(ctx, path) (*Client, error)`
Convenience helper that initializes a client with the specified workspace path and opens the repository session.

```go
client, err := sdk.OpenWorkspace(ctx, "/path/to/repo")
```

#### `sdk.New(opts ...Option) (*Client, error)`
Constructs a new `Client` using functional options.

```go
client, err := sdk.New(
	sdk.WithWorkspace("/path/to/repo"),
	// sdk.WithCustomVersion(sv), // optional testing override
)
```

### Client Methods

| Method | Return Type | Description |
| :--- | :--- | :--- |
| `Workspace()` | `string` | Returns the configured repository root path. |
| `Version()` | `version.SemVer` | Returns the semantic version of the SDK (`1.4.0`). |
| `VersionString()` | `string` | Returns the formatted version string (e.g., `"1.4.0"`). |
| `Close()` | `error` | Releases all underlying resources. Safe and idempotent. |
| `Repository()` | `RepositoryManagementContract` | Provides repository lifecycle, metadata, and statistics. |
| `Files()` | `FileContract` | Provides file discovery, status, and relationships. |
| `Packages()` | `PackageContract` | Provides package discovery, hierarchies, and metrics. |
| `Symbols()` | `SymbolContract` | Provides symbol lookup, references, and hierarchies. |
| `Search()` | `SearchContract` | Provides multi-domain search across the codebase. |
| `Graph()` | `GraphContract` | Provides knowledge graph traversal and diagram exports. |
| `Analysis()` | `AnalysisContract` | Provides architecture, health, and quality assessments. |
| `Navigation()` | `NavigationContract` | Provides definition lookup, reference trees, and call graphs. |
| `Reasoning()` | `ReasoningContract` | Provides impact analysis, recommendations, and insights. |
| `Events()` | `EventContract` | Provides event subscription and event streaming. |
| `Intelligence()` | `IntelligenceContract` | Provides unified request-based access to intelligence capabilities. |

---

# Core SDK

The Core SDK provides fundamental repository inspection, file indexing, package organization, and symbol resolution capabilities.

## Repository

The Repository contract manages session lifecycle, state transitions, and descriptive VCS metadata.

### Public Interface: `RepositoryManagementContract`
```go
type RepositoryManagementContract interface {
	Open(ctx context.Context, path string) (*RepositoryInfo, error)
	Close(ctx context.Context) error
	Info(ctx context.Context) (*RepositoryInfo, error)
	State() RepositoryState
	Metadata(ctx context.Context) (*RepositoryMetadata, error)
	Statistics(ctx context.Context) (*RepositoryStatistics, error)
	Reload(ctx context.Context) error
}
```

### Lifecycle States
- `StateUninitialized`: Client instantiated, repository not yet opened.
- `StateReady`: Repository opened and indexed.
- `StateIndexing`: Analysis/indexing in progress.
- `StateClosed`: Repository session closed.
- `StateError`: Encountered an unrecoverable indexing error.

### Key Models
```go
type RepositoryStatistics struct {
	TotalFiles         int `json:"total_files"`
	TotalDirectories   int `json:"total_directories"`
	TotalPackages      int `json:"total_packages"`
	TotalSymbols       int `json:"total_symbols"`
	TotalRelationships int `json:"total_relationships"`
	TotalDependencies  int `json:"total_dependencies"`
	TotalDocs          int `json:"total_docs"`
	TotalConfigs       int `json:"total_configs"`
}
```

---

## File

The File contract provides discovery and metadata extraction across all workspace files.

### Public Interface: `FileContract`
```go
type FileContract interface {
	DiscoverFiles(ctx context.Context, filter FileFilter, page PaginationOptions) ([]FileInfo, error)
	GetFileInfo(ctx context.Context, path string) (*FileInfo, error)
	FileStatus(ctx context.Context, path string) (*FileIndexStatus, error)
	FileRelationships(ctx context.Context, path string) (*FileRelationshipResult, error)
}
```

### Example: Discovering Go Files
```go
filter := sdk.FileFilter{
	Extension: ".go",
	MaxDepth:  4,
}
page := sdk.PaginationOptions{Limit: 25, Offset: 0}
files, err := client.Files().DiscoverFiles(ctx, filter, page)
```

---

## Package

The Package contract exposes package structures, hierarchy trees, and dependency relationships.

### Public Interface: `PackageContract`
```go
type PackageContract interface {
	DiscoverPackages(ctx context.Context, filter PackageFilter, page PaginationOptions) ([]PackageInfo, error)
	GetPackageInfo(ctx context.Context, packagePath string) (*PackageInfo, error)
	PackageStatistics(ctx context.Context, packagePath string) (*PackageStatistics, error)
	PackageHierarchy(ctx context.Context, rootPackage string) (*PackageHierarchyNode, error)
	PackageRelationships(ctx context.Context, packagePath string) ([]PackageRelationship, error)
}
```

---

## Symbol

The Symbol contract extracts and queries functions, types, interfaces, variables, and constants.

### Public Interface: `SymbolContract`
```go
type SymbolContract interface {
	GetSymbolInfo(ctx context.Context, symbolID string) (*SymbolInfo, error)
	FindSymbolReferences(ctx context.Context, symbolID string, page PaginationOptions) ([]SymbolReference, error)
	SymbolHierarchy(ctx context.Context, symbolID string) (*SymbolHierarchyNode, error)
	SymbolDocumentation(ctx context.Context, symbolID string) (string, error)
	SymbolOwner(ctx context.Context, symbolID string) (*PackageInfo, error)
}
```

### Symbol Kinds
Supported kinds include `function`, `method`, `struct`, `interface`, `variable`, `constant`, and `type`.

---

## Search

The Search contract performs unified and domain-specific lookups across the codebase.

### Public Interface: `SearchContract`
```go
type SearchContract interface {
	Search(ctx context.Context, query SearchQuery) (*SearchResult, error)
	SearchSymbols(ctx context.Context, pattern string, page PaginationOptions) (*SearchResult, error)
	SearchPackages(ctx context.Context, pattern string, page PaginationOptions) (*SearchResult, error)
	SearchFiles(ctx context.Context, pattern string, page PaginationOptions) (*SearchResult, error)
	SearchDocs(ctx context.Context, pattern string, page PaginationOptions) (*SearchResult, error)
	SearchConfigs(ctx context.Context, pattern string, page PaginationOptions) (*SearchResult, error)
}
```

### Search Domains
- `SearchDomainSymbol`: Code symbols and signatures.
- `SearchDomainPackage`: Package and namespace identifiers.
- `SearchDomainFile`: File paths and names.
- `SearchDomainDoc`: Markdown, comments, and documentation files.
- `SearchDomainConfig`: Configuration files (`.json`, `.yaml`, `.toml`, `go.mod`).

---

# Intelligence SDK

The Intelligence SDK exposes multidimensional analysis, graph modeling, code navigation, reasoning, and event streaming.

---

## Graph

The Graph contract provides knowledge graph queries and export capabilities.

### Public Interface: `GraphContract`
```go
type GraphContract interface {
	GraphInfo(ctx context.Context) (*GraphInfo, error)
	Nodes(ctx context.Context, filter GraphFilter, page PaginationOptions) ([]GraphNode, error)
	Relationships(ctx context.Context, filter GraphFilter, page PaginationOptions) ([]GraphRelationship, error)
	Neighbors(ctx context.Context, nodeID string, depth int) ([]GraphNode, error)
	FindPaths(ctx context.Context, fromNodeID, toNodeID string, maxDepth int) ([][]GraphNode, error)
	ExportGraph(ctx context.Context, filter GraphFilter, format GraphExportFormat) (*GraphExportResult, error)
}
```

### Supported Export Formats
- `sdk.ExportFormatJSON`: Raw graph nodes and edge lists for programmatic processing.
- `sdk.ExportFormatMermaid`: Mermaid diagram syntax for documentation and GitHub rendering.
- `sdk.ExportFormatGraphviz`: DOT syntax for Graphviz visualization.

```go
exportRes, err := client.Graph().ExportGraph(ctx, sdk.GraphFilter{MaxDepth: 2}, sdk.ExportFormatMermaid)
if err == nil {
	fmt.Println(exportRes.Content)
}
```

---

## Analysis

The Analysis contract computes multidimensional repository health, architectural coupling, and dependency graphs.

### Public Interface: `AnalysisContract`
```go
type AnalysisContract interface {
	AnalyzeArchitecture(ctx context.Context) (*ArchitectureAnalysisResult, error)
	AnalyzeDependencies(ctx context.Context) (*DependencyAnalysisResult, error)
	RepositoryHealth(ctx context.Context) (*RepositoryHealthReport, error)
	AnalyzeQuality(ctx context.Context) (*CodeQualityReport, error)
	AnalyzeConfiguration(ctx context.Context) (*ConfigurationReport, error)
	Analyze(ctx context.Context, req AnalysisRequest) (*AnalysisResult, error)
}
```

### Health Report Structure
```go
type RepositoryHealthReport struct {
	OverallScore float64                 `json:"overall_score"` // 0.0 - 100.0
	Grade        string                  `json:"grade"`         // "A", "B", "C", "D", "F"
	Status       string                  `json:"status"`        // "HEALTHY", "WARNING", "CRITICAL"
	Dimensions   []HealthDimensionResult `json:"dimensions"`
	Timestamp    time.Time               `json:"timestamp"`
}
```

---

## Navigation

The Navigation contract provides IDE-grade symbol resolution, reference searching, and call hierarchy traversal.

### Public Interface: `NavigationContract`
```go
type NavigationContract interface {
	GoToDefinition(ctx context.Context, symbolID string) (*DefinitionResult, error)
	FindReferences(ctx context.Context, symbolID string, page PaginationOptions) (*ReferenceResult, error)
	CallHierarchy(ctx context.Context, symbolID string, direction string) (*CallHierarchyNode, error)
	SymbolHierarchy(ctx context.Context, symbolID string) (*NavSymbolHierarchyNode, error)
	NavigationContext(ctx context.Context, symbolID string, opts ContextOptions) (*NavigationContextResult, error)
	NavigateRelationship(ctx context.Context, sourceID, relationshipKind string) ([]NavigationTarget, error)
}
```

---

## Reasoning

The Reasoning contract provides deterministic impact evaluation and refactoring recommendations.

### Public Interface: `ReasoningContract`
```go
type ReasoningContract interface {
	ImpactAnalysis(ctx context.Context, entityID string) (*ImpactResult, error)
	Recommendations(ctx context.Context, scope string) ([]RecommendationResult, error)
	BreakingChanges(ctx context.Context, targetEntity string) (*BreakingChangeResult, error)
	RefactoringAdvice(ctx context.Context, entityID string) (*RefactoringResult, error)
	EngineeringInsights(ctx context.Context) ([]EngineeringInsight, error)
	Reason(ctx context.Context, req ReasoningRequest) (*ReasoningResult, error)
}
```

### Example: Impact Analysis
```go
impact, err := client.Reasoning().ImpactAnalysis(ctx, "pkg/service.Execute")
if err == nil {
	fmt.Printf("Risk Level: %s (Score: %.2f)\n", impact.RiskLevel, impact.ImpactScore)
	fmt.Printf("Directly Impacted: %v\n", impact.DirectlyImpacted)
}
```

---

## Events

The Event contract enables real-time notification streaming for workspace mutations and indexing lifecycle changes.

### Public Interface: `EventContract`
```go
type EventContract interface {
	Subscribe(ctx context.Context, eventType EventType, handler EventHandler) (Subscription, error)
	SubscribeAll(ctx context.Context, handler EventHandler) (Subscription, error)
	EventStream(ctx context.Context, bufferSize int) (<-chan Event, error)
}
```

### Event Model
```go
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Workspace string                 `json:"workspace"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
	Metadata  map[string]string      `json:"metadata,omitempty"`
}
```

### Supported Event Types
- `EventTypeRepositoryOpened`: Fired when a workspace session begins.
- `EventTypeRepositoryClosed`: Fired when a workspace session is terminated.
- `EventTypeRepositoryReloaded`: Fired when workspace indexing is refreshed.
- `EventTypeIndexStarted`: Fired when background indexing commences.
- `EventTypeIndexCompleted`: Fired upon successful indexing completion.
- `EventTypeIndexFailed`: Fired if indexing encounters errors.
- `EventTypeAnalysisStarted`: Fired when multi-dimensional analysis starts.
- `EventTypeAnalysisCompleted`: Fired when analysis completes.

---

# Unified Intelligence API

In addition to specialized contracts (`Analysis()`, `Navigation()`, `Reasoning()`, `Events()`), the SDK provides a unified intelligence facade via `client.Intelligence()`.

### When to Use
- **Specialized Contracts**: Recommended when writing strongly-typed, focused integrations (e.g., dedicated health checkers, graph exporters).
- **Unified Intelligence Facade**: Recommended when writing generic tool adapters, AI agent tool interfaces, or RPC gateways where uniform request/response DTO structures are preferred.

### Public Interface: `IntelligenceContract`
```go
type IntelligenceContract interface {
	Analyze(ctx context.Context, req AnalysisRequest) (*AnalysisResult, error)
	Navigate(ctx context.Context, symbolID string, relKind string) (*NavigationResult, error)
	Reason(ctx context.Context, req ReasoningRequest) (*ReasoningResult, error)
	Events(ctx context.Context, eventType string) (<-chan IntelligenceEvent, error)
}
```

---

# Common SDK Concepts

## Context and Cancellation
All I/O, analysis, search, and navigation methods accept a `context.Context` parameter.
- When `ctx.Done()` is triggered (by timeout or cancellation), ongoing operations abort immediately and return `context.Canceled` or `context.DeadlineExceeded`.
- Long-running operations such as graph traversal and deep architectural analysis honor context deadlines promptly.

## Pagination
List and search operations accept `sdk.PaginationOptions`:
```go
type PaginationOptions struct {
	Limit  int `json:"limit"`  // Maximum results to return (default 20, max 1000)
	Offset int `json:"offset"` // Zero-based starting offset
}
```
*Tip: Negative limits and offsets are automatically normalized to safe bounds.*

## Error Handling & Categories
The SDK uses structured errors conforming to the `SDKError` interface. Errors expose a machine-readable `SemanticCategory`:

```go
type SDKError interface {
	error
	Category() SemanticCategory
	Code() string
	Message() string
}
```

| Category Constant | Meaning | Typical Scenario |
| :--- | :--- | :--- |
| `CategoryInvalidInput` | Caller provided malformed arguments | Empty symbol ID, invalid format string |
| `CategoryNotFound` | Requested resource does not exist | File or symbol not found in graph |
| `CategoryUnsupported` | Requested feature not supported | Unsupported export format |
| `CategoryInvalidState` | Client is not in a valid state | Calling operations after `Close()` |
| `CategoryLifecycleViolation`| Lifecycle sequence error | Opening an already-opened workspace |
| `CategoryUnavailable` | Underlying service is not ready | Indexing in progress or failed |
| `CategoryInternal` | Unexpected internal failure | Filesystem I/O error |

```go
if err != nil {
	var sdkErr sdk.SDKError
	if errors.As(err, &sdkErr) {
		fmt.Printf("Error [%s / %s]: %s\n", sdkErr.Category(), sdkErr.Code(), sdkErr.Message())
	}
}
```

## Thread Safety
- The `*sdk.Client` instance and all capability accessors are thread-safe and support concurrent reads across multiple goroutines.
- `client.Close()` is thread-safe, idempotent, and safely halts background subscriptions.

---

# Versioning and Compatibility

Limoxel SDK follows strict [Semantic Versioning (SemVer 2.0.0)](https://semver.org/).

### SemVer Format
Versions are structured as `MAJOR.MINOR.PATCH` (e.g., `1.4.0`):
- **MAJOR**: Breaking API contract changes or signature modifications.
- **MINOR**: Backward-compatible new capabilities, contracts, or options.
- **PATCH**: Backward-compatible bug fixes and performance enhancements.

### Public Compatibility Tools (`sdk/compatibility`)
Consumers can evaluate planned version migrations programmatically:

```go
import "github.com/unhield/limoxel/sdk/compatibility"

validator := compatibility.NewUpgradeValidator()
decision := validator.ValidateUpgrade(currentVer, targetVer, changes)
if decision.Compatible {
	fmt.Println("Upgrade is fully backward-compatible.")
} else {
	fmt.Printf("Breaking changes detected: %v\n", decision.BreakingChanges)
}
```

---

# Deprecations

When a public API symbol is scheduled for retirement, it undergoes a structured lifecycle:
1. **Introduced**: The symbol is part of the supported public surface.
2. **Deprecated**: The symbol is marked deprecated with doc comments and registered in `compatibility.DeprecationTracker`. It remains fully functional for at least one minor release cycle.
3. **Removed**: The symbol is removed only in a subsequent MAJOR version release.

### Querying Deprecations
```go
tracker := compatibility.NewDeprecationTracker()
if record, isDeprecated := tracker.Lookup("client.OldMethod"); isDeprecated {
	fmt.Printf("Deprecated since %s: Use %s instead. %s\n",
		record.Since, record.Replacement, record.MigrationGuidance)
}
```
*(Note: As of version `1.4.0`, all public APIs are in `StateSupported` status with zero deprecated symbols).*

---

# Examples

The SDK includes 6 complete, tested example projects located under [`sdk/examples/`](file:///d:/limoxel/sdk/examples):

| Example Directory | Capability Demonstrated |
| :--- | :--- |
| [`01_basic_usage`](file:///d:/limoxel/sdk/examples/01_basic_usage) | Client initialization, workspace opening, repository statistics, file discovery. |
| [`02_repository_analysis`](file:///d:/limoxel/sdk/examples/02_repository_analysis) | Health score evaluation, architectural analysis, code quality dimensions. |
| [`03_knowledge_graph`](file:///d:/limoxel/sdk/examples/03_knowledge_graph) | Graph node traversal, relationship querying, Mermaid diagram export. |
| [`04_code_navigation`](file:///d:/limoxel/sdk/examples/04_code_navigation) | Go-to-definition, reference discovery, call hierarchy inspection. |
| [`05_intelligence_reasoning`](file:///d:/limoxel/sdk/examples/05_intelligence_reasoning) | Deterministic impact analysis, breaking change detection, engineering insights. |
| [`06_event_streaming`](file:///d:/limoxel/sdk/examples/06_event_streaming) | Real-time event streaming and typed event subscription handlers. |

---

# Templates

Production-ready application scaffolding templates are available under [`sdk/templates/`](file:///d:/limoxel/sdk/templates):

| Template Directory | Target Application Type |
| :--- | :--- |
| [`starter`](file:///d:/limoxel/sdk/templates/starter) | Minimal single-file Go script for repository inspection. |
| [`cli`](file:///d:/limoxel/sdk/templates/cli) | Multi-command CLI tool using Cobra/flag patterns. |
| [`integration`](file:///d:/limoxel/sdk/templates/integration) | CI/CD automated health-check gate with exit-code thresholds. |
| [`service`](file:///d:/limoxel/sdk/templates/service) | Standalone REST/HTTP API microservice exposing SDK capabilities. |
| [`enterprise`](file:///d:/limoxel/sdk/templates/enterprise) | High-throughput concurrent worker pool for batch repository analysis. |

---

# Developer Portal

A lightweight, zero-dependency Developer Portal is included in [`sdk/portal/`](file:///d:/limoxel/sdk/portal). It can be served locally or hosted as static documentation:

- `index.html`: Developer Portal landing page with quickstart guide and feature cards.
- `docs.html`: Comprehensive API contract references and guide.
- `api-explorer.html`: Interactive client API explorer detailing all 11 capability contracts.
- `examples.html`: Code recipe browser with copyable snippets.
- `changelog.html`: Release changelog and SemVer history.

To view locally:
```bash
# Serve using Python or any static web server:
cd sdk/portal && python -m http.server 8080
```

---

# Distribution and Releases

### Verification & Integrity
SDK distribution packages include SHA-256 cryptographic verification manifests generated via [`sdk/distribution/`](file:///d:/limoxel/sdk/distribution). Consumers can verify package integrity prior to integration:

```go
import "github.com/unhield/limoxel/sdk/distribution"

verifier := distribution.NewIntegrityVerifier()
valid, failedFiles, err := verifier.VerifyManifest("path/to/SHA256SUMS")
if !valid {
	log.Fatalf("Integrity verification failed for: %v", failedFiles)
}
```

---

# Practical Integration Patterns

### Pattern 1: CI/CD Quality Gate
```go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/unhield/limoxel/sdk"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := sdk.OpenWorkspace(ctx, ".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening workspace: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	health, err := client.Analysis().RepositoryHealth(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Health check failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Repository Health: %.1f/100 (Grade %s)\n", health.OverallScore, health.Grade)
	if health.OverallScore < 75.0 {
		fmt.Fprintf(os.Stderr, "FAILED: Health score below threshold (75.0)\n")
		os.Exit(1)
	}
	fmt.Println("PASSED: Quality gate check passed.")
}
```

### Pattern 2: Mermaid Graph Documentation Generator
```go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/unhield/limoxel/sdk"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := sdk.OpenWorkspace(ctx, ".")
	if err != nil {
		panic(err)
	}
	defer client.Close()

	res, err := client.Graph().ExportGraph(ctx, sdk.GraphFilter{MaxDepth: 2}, sdk.ExportFormatMermaid)
	if err != nil {
		panic(err)
	}

	_ = os.WriteFile("architecture.mermaid", []byte(res.Content), 0644)
	fmt.Println("Wrote architecture.mermaid successfully")
}
```

---

# Troubleshooting

| Issue | Common Cause | Resolution |
| :--- | :--- | :--- |
| `invalid workspace path` | Target directory does not exist or is inaccessible | Verify path with `filepath.Abs()` and ensure read permissions. |
| `context deadline exceeded` | Large repository analysis timed out | Increase `context.WithTimeout` duration. |
| `repository closed` / `invalid state` | Method called after `client.Close()` | Ensure `client.Close()` is only deferred or called after all operations complete. |
| `unsupported export format` | Invalid graph export string passed | Use constants: `sdk.ExportFormatJSON`, `sdk.ExportFormatMermaid`, `sdk.ExportFormatGraphviz`. |
| `symbol not found` | Symbol identifier mistyped | Use `client.Search().SearchSymbols()` to discover valid symbol IDs. |

---

# API Surface Summary

| Capability Contract | Accessor Method | Primary Responsibilities |
| :--- | :--- | :--- |
| `RepositoryManagementContract` | `client.Repository()` | Workspace session lifecycle, VCS metadata, quantitative statistics. |
| `FileContract` | `client.Files()` | File discovery, indexing status, file dependency relationships. |
| `PackageContract` | `client.Packages()` | Package discovery, hierarchy navigation, package metrics. |
| `SymbolContract` | `client.Symbols()` | Function/type lookup, documentation, reference lists. |
| `SearchContract` | `client.Search()` | Unified and domain-specific search (symbols, files, docs, configs). |
| `GraphContract` | `client.Graph()` | Knowledge graph node traversal, neighbor querying, Mermaid/DOT export. |
| `AnalysisContract` | `client.Analysis()` | Architecture coupling, dependency graphs, multidimensional health reports. |
| `NavigationContract` | `client.Navigation()` | Definition lookup, call hierarchies, symbol trees, context mapping. |
| `ReasoningContract` | `client.Reasoning()` | Deterministic impact analysis, breaking changes, engineering insights. |
| `EventContract` | `client.Events()` | Real-time event streaming and typed lifecycle subscriptions. |
| `IntelligenceContract` | `client.Intelligence()` | Unified request-driven facade for analysis, navigation, reasoning, and events. |

---

# Recommended Consumer Workflow

1. **Initialize**: Instantiate client with `sdk.OpenWorkspace(ctx, path)`.
2. **Coordinate Lifecycles**: Ensure `defer client.Close()` is registered.
3. **Query Core Entities**: Locate packages or symbols using `client.Search()` or `client.Packages()`.
4. **Compute Intelligence**: Run health or impact assessments with `client.Analysis()` and `client.Reasoning()`.
5. **Handle Results & Errors**: Check structured errors via `errors.As(err, &sdkErr)`.
6. **Stream Changes**: If monitoring long-lived processes, subscribe to `client.Events()`.

---

# Limitations

- Graph exports for extremely large workspaces (>100,000 symbols) should specify a `MaxDepth` filter (e.g., `MaxDepth: 2`) to ensure diagram readability and avoid memory bloat.
- Repository analysis requires local filesystem access to the workspace directory.

---

# Authority

This document defines the authoritative, public-facing user manual for the Limoxel SDK & Public API. It governs public consumer interactions and remains consistent with the SDK Foundation and public capability specifications.

---

# Applicability

This manual applies to all external software developers, integration engineers, IDE plugin authors, and automated tools consuming the public Limoxel SDK surface (`github.com/unhield/limoxel/sdk`).

It covers:
- Core Repository capabilities (`Repository`, `File`, `Package`, `Symbol`, `Search`).
- Intelligence capabilities (`Graph`, `Analysis`, `Navigation`, `Reasoning`, `Events`).
- Unified Intelligence Facade.
- Common concepts (Context, Pagination, Error Model, Thread Safety).
- Compatibility, Semantic Versioning, and Deprecation tracking.
- Examples, Templates, Developer Portal, and Distribution tooling.

---

# Change Policy

Modifications to the public SDK must preserve stable public contracts, predictable error behavior, backward compatibility, and semantic versioning guarantees. Any changes to the public API surface must be reflected accurately in this user manual.

This specification remains authoritative until an approved revision supersedes it.

---
