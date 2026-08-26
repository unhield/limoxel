package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"

	"github.com/unhield/limoxel/internal/capabilities/cli/config"
	"github.com/unhield/limoxel/internal/capabilities/cli/diagnostics"
	"github.com/unhield/limoxel/internal/capabilities/intelligence/analysis"
	"github.com/unhield/limoxel/internal/capabilities/intelligence/crossrepo"
	"github.com/unhield/limoxel/internal/capabilities/intelligence/knowledgegraph"
	"github.com/unhield/limoxel/internal/capabilities/intelligence/navigation"
	"github.com/unhield/limoxel/internal/capabilities/intelligence/reasoning"
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
	origcli "github.com/unhield/limoxel/internal/cli"
	"github.com/unhield/limoxel/internal/engine"
	langreg "github.com/unhield/limoxel/internal/language"
)

// Context encapsulates execution state, IO streams, and capability lifecycles for CLI commands.
type Context struct {
	mu                 sync.RWMutex
	app                *App
	repoPath           string
	configManager      *config.Manager
	diagnosticsManager *diagnostics.Manager
	bootstrap          *origcli.Bootstrap
	engine             *engine.Engine
	repoService        *query.RepositoryService
	discResult         *discovery.Result
	profile            *metadata.Profile
	structModel        *language.StructureModel
	depModel           *dependency.DependencyModel
	indexModel         *indexing.IndexModel
	symModel           *symbol.SymbolModel
	xrefModel          *xref.XRefModel
	kgEngine           *knowledgegraph.Engine
	kgModel            *knowledgegraph.KnowledgeGraphModel
	reasonEngine       *reasoning.Engine
	navEngine          *navigation.Engine
	navModel           *navigation.NavigationModel
	analEngine         *analysis.Engine
	analModel          *analysis.AnalysisModel
	formatter          *Formatter
	interactive        bool
}

// NewContext constructs an initialized execution Context.
func NewContext(app *App, stdout, stderr io.Writer, format OutputFormat) (*Context, error) {
	fmtInstance, err := NewFormatter(stdout, stderr, format)
	if err != nil {
		return nil, err
	}

	return &Context{
		app:          app,
		repoPath:     ".",
		formatter:    fmtInstance,
		kgEngine:     knowledgegraph.New(),
		reasonEngine: reasoning.New(),
		navEngine:    navigation.New(),
		analEngine:   analysis.New(),
	}, nil
}

// App returns the associated CLI App coordinator.
func (c *Context) App() *App {
	if c == nil {
		return nil
	}
	return c.app
}

// Formatter returns the active Formatter.
func (c *Context) Formatter() *Formatter {
	if c == nil {
		return nil
	}
	return c.formatter
}

// ConfigManager returns the configuration manager instance if initialized.
func (c *Context) ConfigManager() *config.Manager {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.configManager
}

// SetConfigManager updates the active configuration manager instance.
func (c *Context) SetConfigManager(cm *config.Manager) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.configManager = cm
}

// EnsureConfigManager returns the initialized Configuration Manager, loading defaults and files if needed.
func (c *Context) EnsureConfigManager() (*config.Manager, error) {
	if c == nil {
		return nil, ErrNilContext
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.configManager != nil {
		return c.configManager, nil
	}

	mgr, err := config.NewManager(func(o *config.ManagerOptions) {
		o.WorkspaceDir = c.repoPath
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize configuration manager: %w", err)
	}

	c.configManager = mgr
	return c.configManager, nil
}

// Config returns the resolved EffectiveConfig snapshot.
func (c *Context) Config() *config.EffectiveConfig {
	if c == nil {
		return nil
	}
	mgr, err := c.EnsureConfigManager()
	if err != nil {
		return nil
	}
	return mgr.Effective()
}

// DiagnosticsManager returns the active diagnostics manager.
func (c *Context) DiagnosticsManager() *diagnostics.Manager {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.diagnosticsManager
}

// SetDiagnosticsManager updates the active diagnostics manager.
func (c *Context) SetDiagnosticsManager(dm *diagnostics.Manager) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.diagnosticsManager = dm
}

// EnsureDiagnosticsManager returns or constructs an operational DiagnosticsManager.
func (c *Context) EnsureDiagnosticsManager() (*diagnostics.Manager, error) {
	if c == nil {
		return nil, ErrNilContext
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.diagnosticsManager != nil {
		return c.diagnosticsManager, nil
	}

	cfg := c.Config()
	mgr, err := diagnostics.NewManager(diagnostics.ManagerOptions{
		WorkspaceDir: c.repoPath,
		Config:       cfg,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize diagnostics manager: %w", err)
	}

	c.diagnosticsManager = mgr
	return c.diagnosticsManager, nil
}

// Logger returns the active structured Logger.
func (c *Context) Logger() *diagnostics.Logger {
	dm, err := c.EnsureDiagnosticsManager()
	if err != nil || dm == nil {
		return nil
	}
	return dm.Logger()
}

// Tracer returns the active execution Tracer.
func (c *Context) Tracer() *diagnostics.Tracer {
	dm, err := c.EnsureDiagnosticsManager()
	if err != nil || dm == nil {
		return nil
	}
	return dm.Tracer()
}

// RepoPath returns the active repository directory path.
func (c *Context) RepoPath() string {
	if c == nil {
		return "."
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.repoPath
}

// SetRepoPath updates the active repository path.
func (c *Context) SetRepoPath(path string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	clean := filepath.Clean(strings.TrimSpace(path))
	if clean == "" || clean == "." {
		clean = "."
	}
	c.repoPath = clean
}

// DiscoveryResult returns the loaded discovery.Result model.
func (c *Context) DiscoveryResult() *discovery.Result {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.discResult
}

// MetadataProfile returns the loaded metadata.Profile model.
func (c *Context) MetadataProfile() *metadata.Profile {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.profile
}

// StructureModel returns the loaded language.StructureModel.
func (c *Context) StructureModel() *language.StructureModel {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.structModel
}

// DependencyModel returns the loaded dependency.DependencyModel.
func (c *Context) DependencyModel() *dependency.DependencyModel {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.depModel
}

// SymbolModel returns the loaded symbol.SymbolModel.
func (c *Context) SymbolModel() *symbol.SymbolModel {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.symModel
}

// XRefModel returns the loaded xref.XRefModel.
func (c *Context) XRefModel() *xref.XRefModel {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.xrefModel
}

// IsInteractive returns true if running in interactive REPL mode.
func (c *Context) IsInteractive() bool {
	if c == nil {
		return false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.interactive
}

// SetInteractive sets the interactive mode state.
func (c *Context) SetInteractive(b bool) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.interactive = b
}

// EnsureEngine lazily initializes and returns the Engine Foundation coordinator.
func (c *Context) EnsureEngine() (*engine.Engine, error) {
	if c == nil {
		return nil, ErrNilContext
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.engine != nil {
		return c.engine, nil
	}

	cfg, err := origcli.NewConfig("limoxel", c.app.Version(), c.repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create engine config: %w", err)
	}

	boot, err := origcli.NewBootstrap(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create engine bootstrap: %w", err)
	}

	eng, err := boot.Initialize()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize engine: %w", err)
	}

	c.bootstrap = boot
	c.engine = eng
	return eng, nil
}

// EnsureRepositoryService lazily initializes and loads the RepositoryService for repoPath.
func (c *Context) EnsureRepositoryService(repoPath string) (*query.RepositoryService, error) {
	if c == nil {
		return nil, ErrNilContext
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	cleanPath := filepath.Clean(strings.TrimSpace(repoPath))
	if cleanPath == "" || cleanPath == "." {
		cleanPath = c.repoPath
	}

	// Return cached service if already loaded for same path
	if c.repoService != nil && c.repoService.LifecycleState() == query.StateReady && c.repoPath == cleanPath {
		return c.repoService, nil
	}

	// Instantiate language registry
	reg := langreg.NewRegistry()
	goLang, _ := langreg.New("go", "Go", []string{".go"}, nil, []string{"golang"})
	mdLang, _ := langreg.New("markdown", "Markdown", []string{".md"}, nil, []string{"md"})
	jsonLang, _ := langreg.New("json", "JSON", []string{".json"}, nil, []string{"json"})
	yamlLang, _ := langreg.New("yaml", "YAML", []string{".yaml", ".yml"}, nil, []string{"yaml"})
	_ = reg.Register(goLang)
	_ = reg.Register(mdLang)
	_ = reg.Register(jsonLang)
	_ = reg.Register(yamlLang)

	// Instantiate pipeline components
	disc, err := discovery.New(reg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize discoverer: %w", err)
	}
	metaCollector, _ := metadata.New(disc)
	langAnalyzer, _ := language.New(disc)
	depAnalyzer, _ := dependency.New(disc)
	indexer, _ := indexing.New(disc)
	symEngine, _ := symbol.New(disc)
	xrefEngine, _ := xref.New(disc, symEngine, depAnalyzer)
	graphEngine, _ := graph.New(disc, metaCollector, langAnalyzer, depAnalyzer, indexer, symEngine, xrefEngine)

	svc := query.NewRepositoryService(
		disc,
		metaCollector,
		langAnalyzer,
		depAnalyzer,
		indexer,
		symEngine,
		xrefEngine,
		graphEngine,
	)

	if err := svc.Load(cleanPath); err != nil {
		return nil, fmt.Errorf("failed to load repository at %q: %w", cleanPath, err)
	}

	// Extract and cache loaded models
	discRes, _ := disc.DiscoverPath(cleanPath)
	prof, _ := metaCollector.Collect(discRes)
	sm, _ := langAnalyzer.Analyze(discRes)
	dm, _ := depAnalyzer.Analyze(discRes)
	im, _ := indexer.Index(discRes)
	symM, _ := symEngine.Parse(discRes)
	var xrefM *xref.XRefModel
	if symM != nil {
		xrefM, _ = xrefEngine.Analyze(discRes, symM, dm)
	}

	c.repoService = svc
	c.discResult = discRes
	c.profile = prof
	c.structModel = sm
	c.depModel = dm
	c.indexModel = im
	c.symModel = symM
	c.xrefModel = xrefM
	c.repoPath = cleanPath

	return svc, nil
}

// EnsureKnowledgeGraph lazily builds and returns the complete KnowledgeGraphModel.
func (c *Context) EnsureKnowledgeGraph(repoPath string) (*knowledgegraph.KnowledgeGraphModel, error) {
	if c == nil {
		return nil, ErrNilContext
	}

	_, err := c.EnsureRepositoryService(repoPath)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.kgModel != nil && c.kgModel.RootPath() == c.repoPath {
		return c.kgModel, nil
	}

	var symDB *symbol.SymbolDatabase
	if c.symModel != nil {
		symDB = c.symModel.Symbols()
	}

	// Build knowledge graph from repository models
	params := knowledgegraph.GraphBuildParams{
		RootPath:        c.repoPath,
		DiscoveryResult: c.discResult,
		SymbolDB:        symDB,
		XRefModel:       c.xrefModel,
		DependencyModel: c.depModel,
		LanguageModel:   c.structModel,
		MetadataProfile: c.profile,
	}

	model, err := c.kgEngine.Build(params)
	if err != nil {
		return nil, fmt.Errorf("failed to build knowledge graph: %w", err)
	}

	c.kgModel = model
	return model, nil
}

// EnsureReasoningEngine returns initialized Reasoning Engine and active KnowledgeGraphModel.
func (c *Context) EnsureReasoningEngine(repoPath string) (*reasoning.Engine, *knowledgegraph.KnowledgeGraphModel, error) {
	kgModel, err := c.EnsureKnowledgeGraph(repoPath)
	if err != nil {
		return nil, nil, err
	}
	return c.reasonEngine, kgModel, nil
}

// EnsureNavigationEngine returns initialized Navigation Engine and analyzed NavigationModel.
func (c *Context) EnsureNavigationEngine(repoPath string) (*navigation.Engine, *navigation.NavigationModel, error) {
	_, err := c.EnsureRepositoryService(repoPath)
	if err != nil {
		return nil, nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.navModel != nil {
		return c.navEngine, c.navModel, nil
	}

	var symDB *symbol.SymbolDatabase
	var symRels []*symbol.SymbolRelationship
	if c.symModel != nil {
		symDB = c.symModel.Symbols()
		if c.symModel.Relationships() != nil {
			symRels = c.symModel.Relationships().AllRelationships()
		}
	}

	semEngine := semantic.NewEngine()
	repoName := filepath.Base(c.repoPath)
	if c.profile != nil && c.profile.Name() != "" {
		repoName = c.profile.Name()
	}
	semModel, _ := semEngine.Analyze(
		repoName,
		c.repoPath,
		symDB,
		symRels,
		c.xrefModel,
		nil,
		c.depModel,
		c.indexModel,
		c.structModel,
	)

	crEngine := crossrepo.NewEngine()
	crModel, _ := crEngine.Analyze(crossrepo.AnalysisParams{
		WorkspaceRoot: c.repoPath,
		SymbolDB:      symDB,
		XRefModel:     c.xrefModel,
		SemanticModel: semModel,
	})

	navParams := navigation.AnalysisParams{
		SymbolDB:       symDB,
		XRefModel:      c.xrefModel,
		SemanticModel:  semModel,
		CrossRepoModel: crModel,
	}

	navModel, err := c.navEngine.Analyze(navParams)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to analyze navigation model: %w", err)
	}

	c.navModel = navModel
	return c.navEngine, navModel, nil
}

// EnsureAnalysisEngine returns initialized Analysis Engine and analyzed AnalysisModel.
func (c *Context) EnsureAnalysisEngine(repoPath string) (*analysis.Engine, *analysis.AnalysisModel, error) {
	_, err := c.EnsureRepositoryService(repoPath)
	if err != nil {
		return nil, nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.analModel != nil {
		return c.analEngine, c.analModel, nil
	}

	var symDB *symbol.SymbolDatabase
	var symRels []*symbol.SymbolRelationship
	if c.symModel != nil {
		symDB = c.symModel.Symbols()
		if c.symModel.Relationships() != nil {
			symRels = c.symModel.Relationships().AllRelationships()
		}
	}

	semEngine := semantic.NewEngine()
	repoName := filepath.Base(c.repoPath)
	if c.profile != nil && c.profile.Name() != "" {
		repoName = c.profile.Name()
	}
	semModel, _ := semEngine.Analyze(
		repoName,
		c.repoPath,
		symDB,
		symRels,
		c.xrefModel,
		nil,
		c.depModel,
		c.indexModel,
		c.structModel,
	)

	crEngine := crossrepo.NewEngine()
	crModel, _ := crEngine.Analyze(crossrepo.AnalysisParams{
		WorkspaceRoot: c.repoPath,
		SymbolDB:      symDB,
		XRefModel:     c.xrefModel,
		SemanticModel: semModel,
	})

	analParams := analysis.AnalysisParams{
		SymbolDB:        symDB,
		XRefModel:       c.xrefModel,
		DependencyModel: c.depModel,
		SemanticModel:   semModel,
		CrossRepoModel:  crModel,
		DiscoveryResult: c.discResult,
		LanguageModel:   c.structModel,
	}

	analModel, err := c.analEngine.Analyze(analParams)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to analyze repository health: %w", err)
	}

	c.analModel = analModel
	return c.analEngine, analModel, nil
}

// Reset clears cached repository services and capability models.
func (c *Context) Reset() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.repoService != nil {
		_ = c.repoService.Close()
	}
	c.repoService = nil
	c.discResult = nil
	c.profile = nil
	c.structModel = nil
	c.depModel = nil
	c.indexModel = nil
	c.symModel = nil
	c.xrefModel = nil
	c.kgModel = nil
	c.navModel = nil
	c.analModel = nil
}
