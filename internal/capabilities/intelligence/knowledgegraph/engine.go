package knowledgegraph

import (
	"sync"
	"time"
)

// Engine is the thread-safe coordinator for Stage 5 Knowledge Graph Intelligence.
type Engine struct {
	mu            sync.RWMutex
	builder       *KnowledgeGraphBuilder
	enrichment    *EnrichmentEngine
	reasoning     *GraphReasoningEngine
	contextGen    *ContextGenerator
	insightEngine *InsightEngine
	model         *KnowledgeGraphModel
	queryEngine   *GraphQueryEngine
}

// New constructs an initialized Knowledge Graph Intelligence Engine.
func New() *Engine {
	return &Engine{
		builder:       NewKnowledgeGraphBuilder(),
		enrichment:    NewEnrichmentEngine(),
		reasoning:     NewGraphReasoningEngine(),
		contextGen:    NewContextGenerator(),
		insightEngine: NewInsightEngine(),
	}
}

// Build creates, enriches, and indexes the complete KnowledgeGraphModel from repository parameters.
func (e *Engine) Build(params GraphBuildParams) (*KnowledgeGraphModel, error) {
	if e == nil {
		return nil, ErrNilEngine
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// 1. Build base graph from repository capabilities
	baseEntities, baseRels := e.builder.BuildBaseGraph(params)

	// 2. Enrich graph with semantic, ownership, dependency, doc, and config relations
	enrichedEntities, enrichedRels := e.enrichment.Enrich(baseEntities, baseRels, params.SymbolDB, params.LanguageModel)

	// 3. Construct temporary model for insight derivation
	tempModel := NewKnowledgeGraphModel(params.RootPath, enrichedEntities, enrichedRels, nil, time.Now().UTC())

	// 4. Derive engineering insights
	insights := e.insightEngine.DeriveInsights(tempModel)

	// 5. Finalize immutable model and initialize query engine
	e.model = NewKnowledgeGraphModel(params.RootPath, enrichedEntities, enrichedRels, insights, time.Now().UTC())
	e.queryEngine = NewGraphQueryEngine(e.model)

	return e.model, nil
}

// Model returns the active immutable KnowledgeGraphModel.
func (e *Engine) Model() *KnowledgeGraphModel {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.model
}

// Query returns the query engine for graph traversals and lookups.
func (e *Engine) Query() *GraphQueryEngine {
	if e == nil {
		return nil
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.queryEngine == nil && e.model != nil {
		return NewGraphQueryEngine(e.model)
	}
	return e.queryEngine
}

// Context returns the ContextGenerator.
func (e *Engine) Context() *ContextGenerator {
	if e == nil {
		return nil
	}
	return e.contextGen
}

// Reasoning returns the GraphReasoningEngine.
func (e *Engine) Reasoning() *GraphReasoningEngine {
	if e == nil {
		return nil
	}
	return e.reasoning
}

// Insights returns the InsightEngine.
func (e *Engine) Insights() *InsightEngine {
	if e == nil {
		return nil
	}
	return e.insightEngine
}
