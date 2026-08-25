package analysis

import (
	"sync"

	"github.com/unhield/limoxel/internal/capabilities/intelligence/crossrepo"
	"github.com/unhield/limoxel/internal/capabilities/intelligence/semantic"
	"github.com/unhield/limoxel/internal/capabilities/repository/dependency"
	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
	"github.com/unhield/limoxel/internal/capabilities/repository/language"
	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
	"github.com/unhield/limoxel/internal/capabilities/repository/xref"
)

// AnalysisParams packages all established repository and intelligence models for analysis.
type AnalysisParams struct {
	SymbolDB        *symbol.SymbolDatabase
	XRefModel       *xref.XRefModel
	DependencyModel *dependency.DependencyModel
	SemanticModel   *semantic.SemanticModel
	CrossRepoModel  *crossrepo.CrossRepoModel
	DiscoveryResult *discovery.Result
	LanguageModel   *language.StructureModel
	ConfigEntries   []*RawConfigEntry
}

// Engine coordinates all Stage 4 analyzers and constructs the immutable AnalysisModel.
type Engine struct {
	mu           sync.RWMutex
	qualityNav   *CodeQualityAnalyzer
	depNav       *DependencyAnalyzer
	archNav      *ArchitectureAnalyzer
	configNav    *ConfigurationAnalyzer
	healthEngine *RepositoryHealthEngine
	model        *AnalysisModel
}

// New constructs an initialized Engineering Analysis Engine.
func New() *Engine {
	return &Engine{
		qualityNav:   NewCodeQualityAnalyzer(nil, nil, nil, nil, nil),
		depNav:       NewDependencyAnalyzer(nil, nil, nil, nil),
		archNav:      NewArchitectureAnalyzer(nil, nil, nil),
		configNav:    NewConfigurationAnalyzer(nil, nil),
		healthEngine: NewRepositoryHealthEngine(),
	}
}

// Analyze executes all 4 analyzers, aggregates findings, and produces the complete AnalysisModel.
func (e *Engine) Analyze(params AnalysisParams) (*AnalysisModel, error) {
	if e == nil {
		return nil, ErrNilEngine
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.qualityNav = NewCodeQualityAnalyzer(
		params.SymbolDB,
		params.XRefModel,
		params.SemanticModel,
		params.CrossRepoModel,
		params.DiscoveryResult,
	)
	e.depNav = NewDependencyAnalyzer(
		params.DependencyModel,
		params.CrossRepoModel,
		params.SymbolDB,
		params.XRefModel,
	)
	e.archNav = NewArchitectureAnalyzer(
		params.CrossRepoModel,
		params.SymbolDB,
		params.XRefModel,
	)
	e.configNav = NewConfigurationAnalyzer(
		params.LanguageModel,
		params.ConfigEntries,
	)

	// Execute analyzers
	qualRes := e.qualityNav.Analyze()
	depRes := e.depNav.Analyze()
	archRes := e.archNav.Analyze()
	confRes := e.configNav.Analyze()

	// Gather all findings
	var allFindings []*Finding
	if qualRes != nil {
		allFindings = append(allFindings, qualRes.Findings()...)
	}
	if depRes != nil {
		allFindings = append(allFindings, depRes.Findings()...)
	}
	if archRes != nil {
		allFindings = append(allFindings, archRes.Findings()...)
	}
	if confRes != nil {
		allFindings = append(allFindings, confRes.Findings()...)
	}
	dedupedFindings := DeduplicateFindings(allFindings)

	// Compute health report
	healthReport := e.healthEngine.ComputeHealth(
		dedupedFindings,
		params.SymbolDB,
		params.DiscoveryResult,
		params.LanguageModel,
	)

	e.model = NewAnalysisModel(qualRes, depRes, archRes, confRes, healthReport)
	return e.model, nil
}

// Model returns the active immutable AnalysisModel.
func (e *Engine) Model() *AnalysisModel {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.model
}

// Sub-analyzer accessors
func (e *Engine) CodeQualityAnalyzer() *CodeQualityAnalyzer     { return e.qualityNav }
func (e *Engine) DependencyAnalyzer() *DependencyAnalyzer       { return e.depNav }
func (e *Engine) ArchitectureAnalyzer() *ArchitectureAnalyzer   { return e.archNav }
func (e *Engine) ConfigurationAnalyzer() *ConfigurationAnalyzer { return e.configNav }
func (e *Engine) HealthEngine() *RepositoryHealthEngine         { return e.healthEngine }
