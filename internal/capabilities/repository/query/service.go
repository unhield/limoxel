package query

import (
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/repository/dependency"
	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
	"github.com/unhield/limoxel/internal/capabilities/repository/graph"
	"github.com/unhield/limoxel/internal/capabilities/repository/indexing"
	"github.com/unhield/limoxel/internal/capabilities/repository/language"
	"github.com/unhield/limoxel/internal/capabilities/repository/metadata"
	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
	"github.com/unhield/limoxel/internal/capabilities/repository/xref"
	"github.com/unhield/limoxel/internal/repository"
)

// RepositoryService coordinates repository lifecycle and exposes unified query APIs.
type RepositoryService struct {
	mu            sync.RWMutex
	state         LifecycleState
	repoRoot      string
	analyzedAt    time.Time
	discoverer    *discovery.Discoverer
	metaCollector *metadata.Collector
	langAnalyzer  *language.Analyzer
	depAnalyzer   *dependency.Analyzer
	indexer       *indexing.Indexer
	symEngine     *symbol.Engine
	xrefEngine    *xref.Engine
	graphEngine   *graph.Engine

	// Loaded models
	discResult  *discovery.Result
	profile     *metadata.Profile
	structModel *language.StructureModel
	depModel    *dependency.DependencyModel
	indexModel  *indexing.IndexModel
	symModel    *symbol.SymbolModel
	xrefModel   *xref.XRefModel
	kg          *graph.KnowledgeGraph

	// API sub-engines
	symbolAPI *SymbolAPI
	graphAPI  *GraphAPI
	searchEng *SearchEngine
}

// NewRepositoryService creates an initialized RepositoryService with engine dependencies.
func NewRepositoryService(
	discoverer *discovery.Discoverer,
	metaCollector *metadata.Collector,
	langAnalyzer *language.Analyzer,
	depAnalyzer *dependency.Analyzer,
	indexer *indexing.Indexer,
	symEngine *symbol.Engine,
	xrefEngine *xref.Engine,
	graphEngine *graph.Engine,
) *RepositoryService {
	return &RepositoryService{
		state:         StateUnloaded,
		discoverer:    discoverer,
		metaCollector: metaCollector,
		langAnalyzer:  langAnalyzer,
		depAnalyzer:   depAnalyzer,
		indexer:       indexer,
		symEngine:     symEngine,
		xrefEngine:    xrefEngine,
		graphEngine:   graphEngine,
	}
}

// LifecycleState returns the current operational state of the service.
func (s *RepositoryService) LifecycleState() LifecycleState {
	if s == nil {
		return StateClosed
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// Load loads and analyzes a repository path using established discovery infrastructure.
func (s *RepositoryService) Load(repoPath string) error {
	if s == nil {
		return ErrNilService
	}
	cleanPath := strings.TrimSpace(repoPath)
	if cleanPath == "" {
		return ErrInvalidInput
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == StateClosed {
		return ErrServiceClosed
	}

	s.state = StateLoading

	if s.discoverer == nil {
		s.state = StateUnloaded
		return WrapQueryError(ErrCatUnavailable, "ERR_NO_DISCOVERER", "discovery capability unavailable", nil)
	}

	discRes, err := s.discoverer.DiscoverPath(cleanPath)
	if err != nil {
		s.state = StateUnloaded
		return WrapQueryError(ErrCatNotFound, "ERR_DISCOVERY_FAILED", "failed to discover repository", err)
	}
	s.discResult = discRes
	s.repoRoot = discRes.Root()

	if s.metaCollector != nil {
		p, err := s.metaCollector.Collect(discRes)
		if err != nil {
			s.state = StateUnloaded
			s.clearModelsLocked()
			return WrapQueryError(ErrCatInternal, "ERR_METADATA_FAILED", "metadata analysis failed", err)
		}
		s.profile = p
	}
	if s.langAnalyzer != nil {
		sm, err := s.langAnalyzer.Analyze(discRes)
		if err != nil {
			s.state = StateUnloaded
			s.clearModelsLocked()
			return WrapQueryError(ErrCatInternal, "ERR_LANGUAGE_FAILED", "language structure analysis failed", err)
		}
		s.structModel = sm
	}
	if s.depAnalyzer != nil {
		dm, err := s.depAnalyzer.Analyze(discRes)
		if err != nil {
			s.state = StateUnloaded
			s.clearModelsLocked()
			return WrapQueryError(ErrCatInternal, "ERR_DEPENDENCY_FAILED", "dependency analysis failed", err)
		}
		s.depModel = dm
	}
	if s.indexer != nil {
		im, err := s.indexer.Index(discRes)
		if err != nil {
			s.state = StateUnloaded
			s.clearModelsLocked()
			return WrapQueryError(ErrCatInternal, "ERR_INDEXING_FAILED", "source indexing failed", err)
		}
		s.indexModel = im
	}
	if s.symEngine != nil {
		sym, err := s.symEngine.Parse(discRes)
		if err != nil {
			s.state = StateUnloaded
			s.clearModelsLocked()
			return WrapQueryError(ErrCatInternal, "ERR_SYMBOL_FAILED", "symbol parsing failed", err)
		}
		s.symModel = sym
	}
	if s.xrefEngine != nil && s.symModel != nil {
		xrefRes, err := s.xrefEngine.Analyze(discRes, s.symModel, s.depModel)
		if err != nil {
			s.state = StateUnloaded
			s.clearModelsLocked()
			return WrapQueryError(ErrCatInternal, "ERR_XREF_FAILED", "cross-reference analysis failed", err)
		}
		s.xrefModel = xrefRes
	}
	if s.graphEngine != nil {
		kg, err := s.graphEngine.BuildGraph(
			s.discResult,
			s.profile,
			s.structModel,
			s.depModel,
			s.indexModel,
			s.symModel,
			s.xrefModel,
		)
		if err != nil {
			s.state = StateUnloaded
			s.clearModelsLocked()
			return WrapQueryError(ErrCatInternal, "ERR_GRAPH_FAILED", "knowledge graph construction failed", err)
		}
		s.kg = kg
	}

	s.analyzedAt = time.Now()
	s.initAPIsLocked()
	s.state = StateReady

	return nil
}

// LoadFromRepository loads an analyzed domain repository.
func (s *RepositoryService) LoadFromRepository(repo *repository.Repository) error {
	if s == nil {
		return ErrNilService
	}
	if repo == nil {
		return ErrInvalidInput
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == StateClosed {
		return ErrServiceClosed
	}

	s.state = StateLoading

	if s.discoverer == nil {
		s.state = StateUnloaded
		return WrapQueryError(ErrCatUnavailable, "ERR_NO_DISCOVERER", "discovery capability unavailable", nil)
	}

	discRes, err := s.discoverer.Discover(repo)
	if err != nil {
		s.state = StateUnloaded
		return WrapQueryError(ErrCatNotFound, "ERR_DISCOVERY_FAILED", "failed to discover repository", err)
	}
	s.discResult = discRes
	s.repoRoot = discRes.Root()

	if s.metaCollector != nil {
		p, err := s.metaCollector.Collect(discRes)
		if err != nil {
			s.state = StateUnloaded
			s.clearModelsLocked()
			return WrapQueryError(ErrCatInternal, "ERR_METADATA_FAILED", "metadata analysis failed", err)
		}
		s.profile = p
	}
	if s.langAnalyzer != nil {
		sm, err := s.langAnalyzer.Analyze(discRes)
		if err != nil {
			s.state = StateUnloaded
			s.clearModelsLocked()
			return WrapQueryError(ErrCatInternal, "ERR_LANGUAGE_FAILED", "language structure analysis failed", err)
		}
		s.structModel = sm
	}
	if s.depAnalyzer != nil {
		dm, err := s.depAnalyzer.Analyze(discRes)
		if err != nil {
			s.state = StateUnloaded
			s.clearModelsLocked()
			return WrapQueryError(ErrCatInternal, "ERR_DEPENDENCY_FAILED", "dependency analysis failed", err)
		}
		s.depModel = dm
	}
	if s.indexer != nil {
		im, err := s.indexer.Index(discRes)
		if err != nil {
			s.state = StateUnloaded
			s.clearModelsLocked()
			return WrapQueryError(ErrCatInternal, "ERR_INDEXING_FAILED", "source indexing failed", err)
		}
		s.indexModel = im
	}
	if s.symEngine != nil {
		sym, err := s.symEngine.Parse(discRes)
		if err != nil {
			s.state = StateUnloaded
			s.clearModelsLocked()
			return WrapQueryError(ErrCatInternal, "ERR_SYMBOL_FAILED", "symbol parsing failed", err)
		}
		s.symModel = sym
	}
	if s.xrefEngine != nil && s.symModel != nil {
		xrefRes, err := s.xrefEngine.Analyze(discRes, s.symModel, s.depModel)
		if err != nil {
			s.state = StateUnloaded
			s.clearModelsLocked()
			return WrapQueryError(ErrCatInternal, "ERR_XREF_FAILED", "cross-reference analysis failed", err)
		}
		s.xrefModel = xrefRes
	}
	if s.graphEngine != nil {
		kg, err := s.graphEngine.BuildGraph(
			s.discResult,
			s.profile,
			s.structModel,
			s.depModel,
			s.indexModel,
			s.symModel,
			s.xrefModel,
		)
		if err != nil {
			s.state = StateUnloaded
			s.clearModelsLocked()
			return WrapQueryError(ErrCatInternal, "ERR_GRAPH_FAILED", "knowledge graph construction failed", err)
		}
		s.kg = kg
	}

	s.analyzedAt = time.Now()
	s.initAPIsLocked()
	s.state = StateReady

	return nil
}

// LoadFromModels initializes the service directly from pre-computed immutable models.
func (s *RepositoryService) LoadFromModels(
	discResult *discovery.Result,
	profile *metadata.Profile,
	structModel *language.StructureModel,
	depModel *dependency.DependencyModel,
	indexModel *indexing.IndexModel,
	symModel *symbol.SymbolModel,
	xrefModel *xref.XRefModel,
	kg *graph.KnowledgeGraph,
) error {
	if s == nil {
		return ErrNilService
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state == StateClosed {
		return ErrServiceClosed
	}

	if discResult == nil && profile == nil && structModel == nil && depModel == nil &&
		indexModel == nil && symModel == nil && xrefModel == nil && kg == nil {
		return WrapQueryError(ErrCatInvalidInput, "ERR_EMPTY_MODELS", "cannot load repository service from all-nil model set", nil)
	}

	s.discResult = discResult
	s.profile = profile
	s.structModel = structModel
	s.depModel = depModel
	s.indexModel = indexModel
	s.symModel = symModel
	s.xrefModel = xrefModel
	s.kg = kg

	if discResult != nil {
		s.repoRoot = discResult.Root()
	} else if kg != nil {
		s.repoRoot = kg.RepositoryRoot()
	}

	s.analyzedAt = time.Now()
	s.initAPIsLocked()
	s.state = StateReady

	return nil
}

// clearModelsLocked clears all loaded models when analysis fails.
func (s *RepositoryService) clearModelsLocked() {
	s.discResult = nil
	s.profile = nil
	s.structModel = nil
	s.depModel = nil
	s.indexModel = nil
	s.symModel = nil
	s.xrefModel = nil
	s.kg = nil
	s.symbolAPI = nil
	s.graphAPI = nil
	s.searchEng = nil
}

// initAPIsLocked initializes the sub-APIs while holding the mutex.
func (s *RepositoryService) initAPIsLocked() {
	s.symbolAPI = NewSymbolAPI(s.symModel)
	s.graphAPI = NewGraphAPI(s.kg, s.depModel, s.xrefModel)
	s.searchEng = NewSearchEngine(s.discResult, s.indexModel, s.symModel, s.kg)
}

// Close closes the service and releases loaded resources.
func (s *RepositoryService) Close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.state = StateClosed
	s.clearModelsLocked()
	return nil
}

// Metadata retrieves high-level repository metadata.
func (s *RepositoryService) Metadata() (*RepositoryMetadataDTO, error) {
	if s == nil {
		return nil, ErrNilService
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.state == StateUnloaded {
		return nil, ErrRepositoryNotLoaded
	}
	if s.state == StateClosed {
		return nil, ErrServiceClosed
	}

	name := ""
	owner := ""
	root := s.repoRoot
	defaultBranch := ""
	currentBranch := ""
	isGit := false
	totalFiles := 0
	totalDirs := 0
	var totalBytes int64
	var languages []string
	var capabilities []string

	if s.discResult != nil {
		if s.discResult.Repository() != nil {
			name = s.discResult.Repository().Name()
		} else {
			name = filepath.Base(s.discResult.Root())
		}
		root = s.discResult.Root()
		totalFiles = s.discResult.FileCount()

		dirSet := make(map[string]bool)
		var byteSum int64
		for _, f := range s.discResult.Files() {
			if f != nil {
				dirSet[filepath.Dir(f.RelPath())] = true
				byteSum += f.Size()
			}
		}
		totalDirs = len(dirSet)
		totalBytes = byteSum

		for _, lang := range s.discResult.Languages() {
			if lang != nil && lang.LanguageName() != "" {
				languages = append(languages, lang.LanguageName())
			}
		}
		capabilities = append(capabilities, "discovery")
	}

	if s.profile != nil {
		owner = s.profile.Owner()
		if s.profile.DefaultBranch() != "" {
			defaultBranch = s.profile.DefaultBranch()
		}
		if s.profile.CurrentBranch() != "" {
			currentBranch = s.profile.CurrentBranch()
		}
		isGit = s.profile.IsGit()
		capabilities = append(capabilities, "metadata")
	}

	if s.structModel != nil {
		capabilities = append(capabilities, "language")
	}
	if s.depModel != nil {
		capabilities = append(capabilities, "dependency")
	}
	if s.indexModel != nil {
		capabilities = append(capabilities, "indexing")
	}
	if s.symModel != nil {
		capabilities = append(capabilities, "symbol")
	}
	if s.xrefModel != nil {
		capabilities = append(capabilities, "xref")
	}
	if s.kg != nil {
		capabilities = append(capabilities, "graph")
	}
	capabilities = append(capabilities, "query")

	return NewRepositoryMetadataDTO(
		name,
		owner,
		root,
		defaultBranch,
		currentBranch,
		isGit,
		totalFiles,
		totalDirs,
		totalBytes,
		languages,
		capabilities,
		string(s.state),
		s.analyzedAt,
	), nil
}

// Statistics retrieves deterministic repository measurements.
func (s *RepositoryService) Statistics() (*RepositoryStatisticsDTO, error) {
	if s == nil {
		return nil, ErrNilService
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.state == StateUnloaded {
		return nil, ErrRepositoryNotLoaded
	}
	if s.state == StateClosed {
		return nil, ErrServiceClosed
	}

	fileCount := 0
	dirCount := 0
	pkgCount := 0
	symCount := 0
	depCount := 0
	relCount := 0
	docCount := 0
	cfgCount := 0

	if s.discResult != nil {
		fileCount = s.discResult.FileCount()
		dirSet := make(map[string]bool)
		for _, f := range s.discResult.Files() {
			if f != nil {
				dirSet[filepath.Dir(f.RelPath())] = true
			}
		}
		dirCount = len(dirSet)
	}
	if s.indexModel != nil {
		pkgCount = len(s.indexModel.Packages())
	} else if s.structModel != nil {
		pkgCount = len(s.structModel.Packages())
	}
	if s.symModel != nil && s.symModel.Symbols() != nil {
		symCount = len(s.symModel.Symbols().AllSymbols())
	}
	if s.depModel != nil && s.depModel.Inventory() != nil {
		depCount = len(s.depModel.Inventory().AllDependencies())
	}
	if s.kg != nil {
		relCount = s.kg.TotalRelationships()
		docCount = len(s.kg.NodesByType(graph.NodeDoc))
		cfgCount = len(s.kg.NodesByType(graph.NodeConfig))
	}

	return NewRepositoryStatisticsDTO(
		fileCount,
		dirCount,
		pkgCount,
		symCount,
		depCount,
		relCount,
		docCount,
		cfgCount,
		true,
	), nil
}

// Symbols returns the SymbolAPI instance.
func (s *RepositoryService) Symbols() *SymbolAPI {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.symbolAPI
}

// Graph returns the GraphAPI instance.
func (s *RepositoryService) Graph() *GraphAPI {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.graphAPI
}

// Search returns the SearchEngine instance.
func (s *RepositoryService) Search() *SearchEngine {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.searchEng
}
