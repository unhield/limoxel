package crossrepo

import (
	"sync"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/intelligence/semantic"
	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
	"github.com/unhield/limoxel/internal/capabilities/repository/xref"
)

// AnalysisParams packages inputs required for Cross Repository Intelligence analysis.
type AnalysisParams struct {
	WorkspaceRoot     string
	Repositories      []RepositoryInput
	SymbolDB          *symbol.SymbolDatabase
	XRefModel         *xref.XRefModel
	SemanticModel     *semantic.SemanticModel
	KnownConfigs      []string
	Modules           []ModuleInfo
	Commits           []*CommitEvent
	TotalFiles        int
	TotalPackages     int
	TotalModules      int
	TotalSymbols      int
	TotalDependencies int
}

// Engine coordinates cross-repository intelligence analysis.
type Engine struct {
	mu           sync.RWMutex
	model        *CrossRepoModel
	crossFile    *CrossFileAnalyzer
	crossPackage *CrossPackageAnalyzer
	crossModule  *CrossModuleAnalyzer
	workspace    *WorkspaceAnalyzer
	evolution    *EvolutionAnalyzer
	validator    *CrossRepoValidator
}

// NewEngine creates a new Cross Repository Intelligence Engine.
func NewEngine() *Engine {
	return &Engine{
		crossFile:    NewCrossFileAnalyzer(),
		crossPackage: NewCrossPackageAnalyzer(),
		crossModule:  NewCrossModuleAnalyzer(),
		workspace:    NewWorkspaceAnalyzer(),
		evolution:    NewEvolutionAnalyzer(),
		validator:    NewCrossRepoValidator(),
	}
}

// Analyze executes the complete Cross Repository Intelligence pipeline.
func (e *Engine) Analyze(params AnalysisParams) (*CrossRepoModel, error) {
	if e == nil {
		return nil, ErrNilEngine
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now().UTC()

	// 1. Task 1 — Cross-File Analysis
	fileRels, symbolProps, crossFileDeps, sharedConfigs := e.crossFile.Analyze(
		params.SymbolDB,
		params.XRefModel,
		params.SemanticModel,
		params.KnownConfigs,
	)

	// 2. Task 2 — Cross-Package Analysis
	pkgComms, pkgContracts, apis := e.crossPackage.Analyze(
		params.SymbolDB,
		params.XRefModel,
		params.SemanticModel,
	)

	// 3. Task 3 — Cross-Module Analysis
	modRels, sharedMods, modHierarchy, versionCompats := e.crossModule.Analyze(
		params.Modules,
	)

	// 4. Task 4 — Workspace Intelligence
	wsModel := e.workspace.Analyze(
		params.WorkspaceRoot,
		params.Repositories,
		sharedConfigs,
	)

	// 5. Task 5 — Repository Evolution
	repoID := "repo:" + params.WorkspaceRoot
	if len(params.Repositories) > 0 {
		repoID = "repo:" + params.Repositories[0].Root
	}
	evoModel := e.evolution.Analyze(
		repoID,
		params.Commits,
		params.TotalFiles,
		params.TotalPackages,
		params.TotalModules,
		params.TotalSymbols,
		params.TotalDependencies,
	)

	// 6. Validation
	valReport := e.validator.Validate(
		fileRels,
		symbolProps,
		crossFileDeps,
		sharedConfigs,
		pkgComms,
		modRels,
		versionCompats,
		wsModel,
	)

	// Construct immutable root model
	model := NewCrossRepoModel(
		fileRels,
		symbolProps,
		crossFileDeps,
		sharedConfigs,
		pkgComms,
		pkgContracts,
		apis,
		modRels,
		sharedMods,
		modHierarchy,
		versionCompats,
		wsModel,
		evoModel,
		valReport,
		now,
	)

	e.model = model
	return model, nil
}

// Model returns the most recently computed CrossRepoModel.
func (e *Engine) Model() *CrossRepoModel {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.model
}

// CrossFileAnalyzer returns the sub-analyzer for cross-file intelligence.
func (e *Engine) CrossFileAnalyzer() *CrossFileAnalyzer {
	if e == nil {
		return nil
	}
	return e.crossFile
}

// CrossPackageAnalyzer returns the sub-analyzer for cross-package intelligence.
func (e *Engine) CrossPackageAnalyzer() *CrossPackageAnalyzer {
	if e == nil {
		return nil
	}
	return e.crossPackage
}

// CrossModuleAnalyzer returns the sub-analyzer for cross-module intelligence.
func (e *Engine) CrossModuleAnalyzer() *CrossModuleAnalyzer {
	if e == nil {
		return nil
	}
	return e.crossModule
}

// WorkspaceAnalyzer returns the sub-analyzer for workspace intelligence.
func (e *Engine) WorkspaceAnalyzer() *WorkspaceAnalyzer {
	if e == nil {
		return nil
	}
	return e.workspace
}

// EvolutionAnalyzer returns the sub-analyzer for repository evolution.
func (e *Engine) EvolutionAnalyzer() *EvolutionAnalyzer {
	if e == nil {
		return nil
	}
	return e.evolution
}

// Validator returns the sub-validator for cross-boundary integrity.
func (e *Engine) Validator() *CrossRepoValidator {
	if e == nil {
		return nil
	}
	return e.validator
}
