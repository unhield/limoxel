package xref

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/unhield/limoxel/internal/capabilities/repository/dependency"
	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
	"github.com/unhield/limoxel/internal/repository"
)

// Engine orchestrates cross-reference analysis, call graph construction, navigation, change impact, and validation.
type Engine struct {
	discoverer  *discovery.Discoverer
	symEngine   *symbol.Engine
	depAnalyzer *dependency.Analyzer
	workers     int
}

// New constructs an operational Engine with default worker concurrency.
func New(
	discoverer *discovery.Discoverer,
	symEngine *symbol.Engine,
	depAnalyzer *dependency.Analyzer,
) (*Engine, error) {
	if discoverer == nil {
		return nil, ErrNilDiscoverer
	}
	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}
	return &Engine{
		discoverer:  discoverer,
		symEngine:   symEngine,
		depAnalyzer: depAnalyzer,
		workers:     workers,
	}, nil
}

// NewWithWorkers constructs an operational Engine with custom worker concurrency.
func NewWithWorkers(
	discoverer *discovery.Discoverer,
	symEngine *symbol.Engine,
	depAnalyzer *dependency.Analyzer,
	workers int,
) (*Engine, error) {
	if discoverer == nil {
		return nil, ErrNilDiscoverer
	}
	if workers < 1 {
		workers = 1
	}
	return &Engine{
		discoverer:  discoverer,
		symEngine:   symEngine,
		depAnalyzer: depAnalyzer,
		workers:     workers,
	}, nil
}

// Analyze processes established discovery, symbol, and dependency models to build the complete XRefModel.
func (e *Engine) Analyze(
	discResult *discovery.Result,
	symModel *symbol.SymbolModel,
	depModel *dependency.DependencyModel,
) (*XRefModel, error) {
	return e.AnalyzeIncremental(discResult, symModel, depModel, nil)
}

// AnalyzeIncremental processes discovery and symbol models, reusing cached references when files are unchanged.
func (e *Engine) AnalyzeIncremental(
	discResult *discovery.Result,
	symModel *symbol.SymbolModel,
	depModel *dependency.DependencyModel,
	previous *XRefModel,
) (*XRefModel, error) {
	if e == nil {
		return nil, ErrNilEngine
	}
	if discResult == nil {
		return nil, ErrNilDiscoveryResult
	}
	if symModel == nil {
		return nil, ErrNilSymbolModel
	}

	repoRoot := discResult.Root()
	files := discResult.Files()

	// Map previous references by file path for incremental reuse
	prevRefsByFile := make(map[string][]*Reference)
	prevCallsByFile := make(map[string][]*CallEdge)

	if previous != nil && previous.RepositoryRoot() == filepath.ToSlash(filepath.Clean(repoRoot)) {
		if previous.References() != nil {
			for _, r := range previous.References().AllReferences() {
				// Only cache file-level AST references; structural relationships are recalculated per model
				if !isStructuralRelationshipEvidence(r.Evidence()) {
					prevRefsByFile[r.FilePath()] = append(prevRefsByFile[r.FilePath()], r)
				}
			}
		}
		if previous.CallGraph() != nil {
			for _, c := range previous.CallGraph().AllEdges() {
				prevCallsByFile[c.FilePath()] = append(prevCallsByFile[c.FilePath()], c)
			}
		}
	}

	type fileJob struct {
		absPath string
		relPath string
	}

	type parseResult struct {
		references  []*Reference
		callEdges   []*CallEdge
		diagnostics []*discovery.Diagnostic
	}

	var (
		jobs        []fileJob
		reusedRefs  []*Reference
		reusedCalls []*CallEdge
		diagnostics []*discovery.Diagnostic
	)

	fileHashes := make(map[string]string)

	for _, f := range files {
		cleanRel := filepath.ToSlash(filepath.Clean(f.RelPath()))
		if !strings.HasSuffix(cleanRel, ".go") {
			continue
		}

		absPath := filepath.Join(repoRoot, filepath.FromSlash(cleanRel))
		currentHash, hashErr := computeFileContentHash(absPath)
		if hashErr == nil {
			fileHashes[cleanRel] = currentHash
		}

		// Check if we can reuse cached references based on content hash
		if previous != nil && hashErr == nil && previous.FileHash(cleanRel) == currentHash {
			if prevRefs, ok := prevRefsByFile[cleanRel]; ok && len(prevRefs) > 0 {
				reusedRefs = append(reusedRefs, prevRefs...)
				if prevCalls, ok := prevCallsByFile[cleanRel]; ok {
					reusedCalls = append(reusedCalls, prevCalls...)
				}
				continue
			}
		}

		jobs = append(jobs, fileJob{
			absPath: absPath,
			relPath: cleanRel,
		})
	}

	results := make([]parseResult, len(jobs))
	var wg sync.WaitGroup
	jobChan := make(chan int, len(jobs))

	workerCount := e.workers
	if workerCount > len(jobs) {
		workerCount = len(jobs)
	}
	if workerCount < 1 {
		workerCount = 1
	}

	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobChan {
				job := jobs[idx]
				refs, calls, diags := parseGoFileXRef(job.absPath, job.relPath, symModel)
				results[idx] = parseResult{
					references:  refs,
					callEdges:   calls,
					diagnostics: diags,
				}
			}
		}()
	}

	for i := range jobs {
		jobChan <- i
	}
	close(jobChan)
	wg.Wait()

	allRefs := append([]*Reference{}, reusedRefs...)
	allCalls := append([]*CallEdge{}, reusedCalls...)

	for _, res := range results {
		allRefs = append(allRefs, res.references...)
		allCalls = append(allCalls, res.callEdges...)
		diagnostics = append(diagnostics, res.diagnostics...)
	}

	// Also add semantic structural relationships from SymbolModel as references (e.g. interface implementations, embedding, type aliases, generic constraints)
	if symModel.Relationships() != nil {
		for _, rel := range symModel.Relationships().AllRelationships() {
			if rel.Kind() == symbol.RelFunctionOwnership || rel.Kind() == symbol.RelMethodReceiver {
				continue
			}
			refKind := RefType
			switch rel.Kind() {
			case symbol.RelInterfaceImplementation:
				refKind = RefInterface
			case symbol.RelStructEmbedding:
				refKind = RefStruct
			case symbol.RelTypeAlias:
				refKind = RefType
			case symbol.RelGenericConstraint:
				refKind = RefType
			}
			fPath := ""
			if rel.Position() != nil {
				fPath = rel.Position().File()
			}
			allRefs = append(allRefs, NewReference(
				rel.SourceID(),
				rel.TargetID(),
				refKind,
				fPath,
				rel.Position(),
				StateResolved,
				string(rel.Kind()),
			))
		}
	}

	refDB := NewReferenceDatabase(allRefs)

	allSymbols := symModel.Symbols().AllSymbols()
	entryPoints, exitPoints, cycles, deadFuncs, reachMap := computeRecursionAndReachability(allCalls, allSymbols)
	callGraph := NewCallGraph(allCalls, entryPoints, exitPoints, cycles, deadFuncs, reachMap)

	navEngine := NewNavigationEngine(symModel, refDB, depModel)
	impactAnalyzer := NewChangeImpactAnalyzer(symModel, refDB, callGraph, depModel)
	validationEngine := NewValidationEngine(symModel, refDB, callGraph, depModel)
	valReport := validationEngine.Validate()

	return NewXRefModelWithHashes(
		repoRoot,
		refDB,
		callGraph,
		navEngine,
		impactAnalyzer,
		valReport,
		diagnostics,
		fileHashes,
	), nil
}

// AnalyzeRepository extracts cross-references for a given domain repository.
func (e *Engine) AnalyzeRepository(repo *repository.Repository) (*XRefModel, error) {
	if e == nil {
		return nil, ErrNilEngine
	}
	if repo == nil {
		return nil, ErrNilRepository
	}

	discResult, err := e.discoverer.Discover(repo)
	if err != nil {
		return nil, err
	}

	symModel, err := e.symEngine.Parse(discResult)
	if err != nil {
		return nil, err
	}

	var depModel *dependency.DependencyModel
	if e.depAnalyzer != nil {
		depModel, _ = e.depAnalyzer.Analyze(discResult)
	}

	return e.Analyze(discResult, symModel, depModel)
}

// AnalyzePath extracts cross-references for a repository path on disk.
func (e *Engine) AnalyzePath(path string) (*XRefModel, error) {
	if e == nil {
		return nil, ErrNilEngine
	}
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return nil, ErrPathEmpty
	}

	discResult, err := e.discoverer.DiscoverPath(cleanPath)
	if err != nil {
		return nil, err
	}

	symModel, err := e.symEngine.Parse(discResult)
	if err != nil {
		return nil, err
	}

	var depModel *dependency.DependencyModel
	if e.depAnalyzer != nil {
		depModel, _ = e.depAnalyzer.Analyze(discResult)
	}

	return e.Analyze(discResult, symModel, depModel)
}

// computeFileContentHash returns hex-encoded SHA-256 of file contents.
func computeFileContentHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

func isStructuralRelationshipEvidence(evidence string) bool {
	return evidence == string(symbol.RelInterfaceImplementation) ||
		evidence == string(symbol.RelStructEmbedding) ||
		evidence == string(symbol.RelTypeAlias) ||
		evidence == string(symbol.RelGenericConstraint) ||
		evidence == string(symbol.RelFunctionOwnership) ||
		evidence == string(symbol.RelMethodReceiver)
}
