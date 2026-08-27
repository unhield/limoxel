package sdk

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/intelligence/analysis"
	"github.com/unhield/limoxel/internal/capabilities/intelligence/crossrepo"
	"github.com/unhield/limoxel/internal/capabilities/intelligence/knowledgegraph"
	"github.com/unhield/limoxel/internal/capabilities/intelligence/navigation"
	"github.com/unhield/limoxel/internal/capabilities/intelligence/semantic"
	"github.com/unhield/limoxel/internal/capabilities/repository/dependency"
	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
	"github.com/unhield/limoxel/internal/capabilities/repository/graph"
	"github.com/unhield/limoxel/internal/capabilities/repository/indexing"
	"github.com/unhield/limoxel/internal/capabilities/repository/language"
	"github.com/unhield/limoxel/internal/capabilities/repository/metadata"
	"github.com/unhield/limoxel/internal/capabilities/repository/query"
	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
	"github.com/unhield/limoxel/internal/capabilities/repository/xref"
	analysissdk "github.com/unhield/limoxel/internal/capabilities/sdk/analysis"
	"github.com/unhield/limoxel/internal/capabilities/sdk/compatibility"
	"github.com/unhield/limoxel/internal/capabilities/sdk/contracts"
	sdkerr "github.com/unhield/limoxel/internal/capabilities/sdk/errors"
	eventsdk "github.com/unhield/limoxel/internal/capabilities/sdk/event"
	"github.com/unhield/limoxel/internal/capabilities/sdk/file"
	graphsdk "github.com/unhield/limoxel/internal/capabilities/sdk/graph"
	"github.com/unhield/limoxel/internal/capabilities/sdk/lifecycle"
	navsdk "github.com/unhield/limoxel/internal/capabilities/sdk/navigation"
	"github.com/unhield/limoxel/internal/capabilities/sdk/pkg"
	reasoningsdk "github.com/unhield/limoxel/internal/capabilities/sdk/reasoning"
	"github.com/unhield/limoxel/internal/capabilities/sdk/repository"
	"github.com/unhield/limoxel/internal/capabilities/sdk/search"
	symsdk "github.com/unhield/limoxel/internal/capabilities/sdk/symbol"
	"github.com/unhield/limoxel/internal/capabilities/sdk/validation"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
	langreg "github.com/unhield/limoxel/internal/language"
)

// Option represents a functional configuration option for initializing an SDK instance.
type Option func(*SDK)

// WithWorkspace sets the active workspace directory for the SDK client.
func WithWorkspace(path string) Option {
	return func(s *SDK) {
		clean := filepath.Clean(strings.TrimSpace(path))
		if clean == "" || clean == "." {
			clean = "."
		}
		s.workspaceRoot = clean
		s.workspaceSet = true
	}
}

// WithCustomVersion allows specifying a custom SemVer (primarily for testing and migration evaluation).
func WithCustomVersion(sv version.SemVer) Option {
	return func(s *SDK) {
		s.version = sv
	}
}

// SDK is the top-level coordinator and entrypoint for Limoxel SDK capabilities.
type SDK struct {
	mu            sync.RWMutex
	workspaceRoot string
	workspaceSet  bool
	version       version.SemVer
	registry      *lifecycle.Registry
	validator     *validation.Validator
	querySvc      *query.RepositoryService
	closed        bool

	// Core SDK Services (Stage 2)
	repoService   contracts.RepositoryManagementContract
	fileService   contracts.FileContract
	pkgService    contracts.PackageContract
	symbolService contracts.SymbolContract
	searchService contracts.SearchContract

	// Intelligence SDK Services (Stage 3)
	graphService     contracts.GraphContract
	analysisService  contracts.AnalysisContract
	navService       contracts.NavigationContract
	reasoningService contracts.ReasoningContract
	eventService     contracts.EventContract
	intelFacade      contracts.IntelligenceContract
}

// New constructs an initialized Limoxel SDK coordinator with Core & Intelligence capability adapters wired.
func New(opts ...Option) (*SDK, error) {
	sdk := &SDK{
		workspaceRoot: ".",
		version:       version.Current(),
		registry:      lifecycle.NewRegistry(),
		validator:     validation.NewValidator(),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(sdk)
		}
	}

	// Initialize underlying analysis pipeline
	reg := langreg.NewRegistry()
	goLang, _ := langreg.New("go", "Go", []string{".go"}, nil, []string{"golang"})
	mdLang, _ := langreg.New("markdown", "Markdown", []string{".md"}, nil, []string{"md"})
	jsonLang, _ := langreg.New("json", "JSON", []string{".json"}, nil, []string{"json"})
	yamlLang, _ := langreg.New("yaml", "YAML", []string{".yaml", ".yml"}, nil, []string{"yaml"})

	_ = reg.Register(goLang)
	_ = reg.Register(mdLang)
	_ = reg.Register(jsonLang)
	_ = reg.Register(yamlLang)

	disc, _ := discovery.New(reg)
	metaCollector, _ := metadata.New(disc)
	langAnalyzer, _ := language.New(disc)
	depAnalyzer, _ := dependency.New(disc)
	indexer, _ := indexing.New(disc)
	symEngine, _ := symbol.New(disc)
	xrefEngine, _ := xref.New(disc, symEngine, depAnalyzer)
	graphEngine, _ := graph.New(disc, metaCollector, langAnalyzer, depAnalyzer, indexer, symEngine, xrefEngine)

	querySvc := query.NewRepositoryService(
		disc,
		metaCollector,
		langAnalyzer,
		depAnalyzer,
		indexer,
		symEngine,
		xrefEngine,
		graphEngine,
	)

	// Stage 3 Event Bus
	evtSvc := eventsdk.NewService()

	var kgModel *knowledgegraph.KnowledgeGraphModel
	var analysisModel *analysis.AnalysisModel
	var navModel *navigation.NavigationModel

	// If workspace was explicitly specified and exists on disk, load and build intelligence models
	if sdk.workspaceSet && sdk.workspaceRoot != "" {
		if info, err := os.Stat(sdk.workspaceRoot); err == nil && info.IsDir() {
			_ = querySvc.Load(sdk.workspaceRoot)

			discRes, _ := disc.DiscoverPath(sdk.workspaceRoot)
			if discRes != nil {
				metaProf, _ := metaCollector.Collect(discRes)
				langModel, _ := langAnalyzer.Analyze(discRes)
				depModel, _ := depAnalyzer.Analyze(discRes)
				idxModel, _ := indexer.Index(discRes)
				symModel, _ := symEngine.Parse(discRes)

				var xrefModel *xref.XRefModel
				if symModel != nil {
					xrefModel, _ = xrefEngine.Analyze(discRes, symModel, depModel)
				}

				var kgModelBase *graph.KnowledgeGraph
				if graphEngine != nil {
					kgModelBase, _ = graphEngine.BuildGraph(discRes, metaProf, langModel, depModel, idxModel, symModel, xrefModel)
				}

				var symDB *symbol.SymbolDatabase
				var symRels []*symbol.SymbolRelationship
				if symModel != nil {
					symDB = symModel.Symbols()
					if symModel.Relationships() != nil {
						symRels = symModel.Relationships().AllRelationships()
					}
				}

				semEngine := semantic.NewEngine()
				semModel, _ := semEngine.Analyze(
					filepath.Base(sdk.workspaceRoot),
					sdk.workspaceRoot,
					symDB,
					symRels,
					xrefModel,
					kgModelBase,
					depModel,
					idxModel,
					langModel,
				)

				crossEngine := crossrepo.NewEngine()
				crossModel, _ := crossEngine.Analyze(crossrepo.AnalysisParams{
					WorkspaceRoot: sdk.workspaceRoot,
					SymbolDB:      symDB,
					XRefModel:     xrefModel,
					SemanticModel: semModel,
				})

				kgEngine := knowledgegraph.New()
				kgModel, _ = kgEngine.Build(knowledgegraph.GraphBuildParams{
					RootPath:        sdk.workspaceRoot,
					DiscoveryResult: discRes,
					SymbolDB:        symDB,
					XRefModel:       xrefModel,
					DependencyModel: depModel,
					LanguageModel:   langModel,
					MetadataProfile: metaProf,
					SemanticModel:   semModel,
					CrossRepoModel:  crossModel,
				})

				anEngine := analysis.New()
				analysisModel, _ = anEngine.Analyze(analysis.AnalysisParams{
					SymbolDB:        symDB,
					XRefModel:       xrefModel,
					DependencyModel: depModel,
					SemanticModel:   semModel,
					CrossRepoModel:  crossModel,
					DiscoveryResult: discRes,
					LanguageModel:   langModel,
				})

				nvEngine := navigation.New()
				navModel, _ = nvEngine.Analyze(navigation.AnalysisParams{
					SymbolDB:       symDB,
					XRefModel:      xrefModel,
					SemanticModel:  semModel,
					CrossRepoModel: crossModel,
				})
			}
		}
	}

	sdk.querySvc = querySvc

	// Core SDK Services (Stage 2)
	sdk.repoService = repository.NewService(querySvc)
	sdk.fileService = file.NewService(querySvc)
	sdk.pkgService = pkg.NewService(querySvc)
	sdk.symbolService = symsdk.NewService(querySvc)
	sdk.searchService = search.NewService(querySvc)

	if sdk.workspaceSet && sdk.workspaceRoot != "" {
		if info, err := os.Stat(sdk.workspaceRoot); err == nil && info.IsDir() {
			_, _ = sdk.repoService.Open(context.Background(), sdk.workspaceRoot)
		}
	}

	// Intelligence SDK Services (Stage 3)
	sdk.graphService = graphsdk.NewService(kgModel)
	sdk.analysisService = analysissdk.NewService(analysisModel)
	sdk.navService = navsdk.NewService(navModel)
	sdk.reasoningService = reasoningsdk.NewService(kgModel)
	sdk.eventService = evtSvc
	sdk.intelFacade = newIntelligenceFacade(sdk.analysisService, sdk.navService, sdk.reasoningService, sdk.eventService)

	// Register baseline contract descriptors
	_ = sdk.RegisterContract(contracts.DefaultRepositoryContractMetadata())
	_ = sdk.RegisterContract(contracts.DefaultFileContractMetadata())
	_ = sdk.RegisterContract(contracts.DefaultPackageContractMetadata())
	_ = sdk.RegisterContract(contracts.DefaultSymbolContractMetadata())
	_ = sdk.RegisterContract(contracts.DefaultGraphContractMetadata())
	_ = sdk.RegisterContract(contracts.DefaultSearchContractMetadata())
	_ = sdk.RegisterContract(contracts.DefaultIntelligenceContractMetadata())
	_ = sdk.RegisterContract(contracts.DefaultAnalysisContractMetadata())
	_ = sdk.RegisterContract(contracts.DefaultNavigationContractMetadata())
	_ = sdk.RegisterContract(contracts.DefaultReasoningContractMetadata())
	_ = sdk.RegisterContract(contracts.DefaultEventContractMetadata())

	// Emit lifecycle event
	_ = eventsdk.EmitLifecycleEvent(context.Background(), evtSvc, contracts.EventTypeSDKInitialized, sdk.workspaceRoot, map[string]string{
		"version": sdk.version.String(),
	})

	return sdk, nil
}

// Version returns the active SDK Semantic Version.
func (s *SDK) Version() version.SemVer {
	if s == nil {
		return version.Current()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.version
}

// Workspace returns the configured workspace root path.
func (s *SDK) Workspace() string {
	if s == nil {
		return "."
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.workspaceRoot
}

// Registry returns the lifecycle registry of public contracts.
func (s *SDK) Registry() *lifecycle.Registry {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.registry
}

// Validator returns the validation engine.
func (s *SDK) Validator() *validation.Validator {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.validator
}

// Repository returns the Repository SDK service adapter.
func (s *SDK) Repository() contracts.RepositoryManagementContract {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.repoService
}

// Files returns the File SDK service adapter.
func (s *SDK) Files() contracts.FileContract {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.fileService
}

// Packages returns the Package SDK service adapter.
func (s *SDK) Packages() contracts.PackageContract {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.pkgService
}

// Symbols returns the Symbol SDK service adapter.
func (s *SDK) Symbols() contracts.SymbolContract {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.symbolService
}

// Search returns the Search SDK service adapter.
func (s *SDK) Search() contracts.SearchContract {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.searchService
}

// Graph returns the Knowledge Graph SDK service adapter (Stage 3).
func (s *SDK) Graph() contracts.GraphContract {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.graphService
}

// Analysis returns the Engineering Analysis SDK service adapter (Stage 3).
func (s *SDK) Analysis() contracts.AnalysisContract {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.analysisService
}

// Navigation returns the Code Navigation SDK service adapter (Stage 3).
func (s *SDK) Navigation() contracts.NavigationContract {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.navService
}

// Reasoning returns the Deterministic Reasoning SDK service adapter (Stage 3).
func (s *SDK) Reasoning() contracts.ReasoningContract {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.reasoningService
}

// Events returns the Event SDK service adapter (Stage 3).
func (s *SDK) Events() contracts.EventContract {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.eventService
}

// Intelligence returns the unified Intelligence Contract facade (Stage 3).
func (s *SDK) Intelligence() contracts.IntelligenceContract {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.intelFacade
}

// RegisterContract validates and registers an API descriptor in the lifecycle registry.
func (s *SDK) RegisterContract(contract contracts.BaseContract) error {
	if s == nil {
		return sdkerr.NewInvalidState("UNINITIALIZED", "sdk is nil")
	}
	if err := contracts.ValidateContract(contract); err != nil {
		return sdkerr.Wrap(err, sdkerr.CategoryInvalidInput, "ERR_VALIDATION_FAILED", "contract validation failed")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.registry.Register(contract.Descriptor)
}

// ValidateRelease checks proposed release changes against SemVer compatibility rules.
func (s *SDK) ValidateRelease(target contracts.BaseContract, changes []compatibility.APIChange) (*compatibility.CompatibilityDecision, error) {
	if s == nil {
		return nil, sdkerr.NewInvalidState("UNINITIALIZED", "sdk is nil")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	desc, ok := s.registry.Lookup(target.Name())
	if !ok {
		return nil, sdkerr.NewNotFound("Contract", target.Name())
	}

	relKind := version.ClassifyRelease(desc.Since, target.Since())
	eval := compatibility.NewEvaluator()
	decision := eval.Evaluate(changes, relKind)
	return &decision, nil
}

// Close gracefully releases underlying resources and closes active capability sessions.
func (s *SDK) Close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	// Emit lifecycle event before closing
	if evtSvc, ok := s.eventService.(*eventsdk.Service); ok {
		_ = eventsdk.EmitLifecycleEvent(context.Background(), evtSvc, contracts.EventTypeSDKClosed, s.workspaceRoot, map[string]string{
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		_ = evtSvc.Close()
	}

	if s.repoService != nil {
		_ = s.repoService.Close(context.Background())
	}
	if s.querySvc != nil {
		_ = s.querySvc.Close()
	}

	s.closed = true
	return nil
}

// intelligenceFacade unites Analysis, Navigation, Reasoning, and Events into contracts.IntelligenceContract.
type intelligenceFacade struct {
	analysis  contracts.AnalysisContract
	nav       contracts.NavigationContract
	reasoning contracts.ReasoningContract
	event     contracts.EventContract
}

func newIntelligenceFacade(
	analysis contracts.AnalysisContract,
	nav contracts.NavigationContract,
	reasoning contracts.ReasoningContract,
	event contracts.EventContract,
) contracts.IntelligenceContract {
	return &intelligenceFacade{
		analysis:  analysis,
		nav:       nav,
		reasoning: reasoning,
		event:     event,
	}
}

func (f *intelligenceFacade) Name() string { return "IntelligenceContract" }
func (f *intelligenceFacade) Capability() lifecycle.CapabilityKind {
	return lifecycle.CapabilityIntelligence
}
func (f *intelligenceFacade) Since() version.SemVer {
	return version.SemVer{Major: 1, Minor: 3, Patch: 0}
}
func (f *intelligenceFacade) Lifecycle() lifecycle.LifecycleState { return lifecycle.StateSupported }
func (f *intelligenceFacade) Validate() error                     { return nil }

func (f *intelligenceFacade) Analyze(ctx context.Context, req contracts.AnalysisRequest) (*contracts.AnalysisResult, error) {
	if f == nil || f.analysis == nil {
		return nil, sdkerr.NewUnavailable("AnalysisCapability", "analysis capability unavailable")
	}
	return f.analysis.Analyze(ctx, req)
}

func (f *intelligenceFacade) Navigate(ctx context.Context, symbolID string, relKind string) (*contracts.NavigationResult, error) {
	if f == nil || f.nav == nil {
		return nil, sdkerr.NewUnavailable("NavigationCapability", "navigation capability unavailable")
	}
	return f.nav.Navigate(ctx, symbolID, relKind)
}

func (f *intelligenceFacade) Reason(ctx context.Context, req contracts.ReasoningRequest) (*contracts.ReasoningResult, error) {
	if f == nil || f.reasoning == nil {
		return nil, sdkerr.NewUnavailable("ReasoningCapability", "reasoning capability unavailable")
	}
	return f.reasoning.Reason(ctx, req)
}

func (f *intelligenceFacade) Events(ctx context.Context, eventType string) (<-chan contracts.IntelligenceEvent, error) {
	if f == nil || f.event == nil {
		return nil, sdkerr.NewUnavailable("EventCapability", "event capability unavailable")
	}

	rawCh, err := f.event.Events(ctx, eventType)
	if err != nil {
		return nil, err
	}

	out := make(chan contracts.IntelligenceEvent, 64)
	go func() {
		defer close(out)
		for evt := range rawCh {
			var pMap map[string]string
			if p, ok := evt.Payload().(map[string]string); ok {
				pMap = p
			}
			out <- contracts.IntelligenceEvent{
				ID:        evt.ID(),
				Type:      string(evt.Type()),
				Timestamp: evt.Timestamp().UnixNano(),
				EntityID:  evt.Workspace(),
				Payload:   pMap,
			}
		}
	}()

	return out, nil
}
