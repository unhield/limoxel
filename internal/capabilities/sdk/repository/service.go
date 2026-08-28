package repository

import (
	"context"
	"path/filepath"
	"strings"
	"sync"

	"github.com/unhield/limoxel/internal/capabilities/repository/query"
	"github.com/unhield/limoxel/internal/capabilities/sdk/contracts"
	sdkerr "github.com/unhield/limoxel/internal/capabilities/sdk/errors"
)

// Service provides the concrete SDK adapter implementation for RepositoryManagementContract.
type Service struct {
	mu sync.RWMutex
	contracts.BaseContract
	repoService *query.RepositoryService
	repoPath    string
	state       contracts.RepositoryState
	info        *contracts.RepositoryInfo
}

// Ensure Service implements RepositoryManagementContract.
var _ contracts.RepositoryManagementContract = (*Service)(nil)

// NewService constructs an initialized Repository SDK service adapter.
func NewService(repoService *query.RepositoryService) *Service {
	return &Service{
		BaseContract: contracts.DefaultRepositoryContractMetadata(),
		repoService:  repoService,
		state:        contracts.StateUninitialized,
	}
}

// Open loads and analyzes the repository located at path.
func (s *Service) Open(ctx context.Context, path string) (*contracts.RepositoryInfo, error) {
	if s == nil {
		return nil, sdkerr.NewInvalidState("NIL_SERVICE", "repository service is nil")
	}

	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return nil, sdkerr.NewInvalidInput("repository path cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.state = contracts.StateIndexing

	if s.repoService == nil {
		s.state = contracts.StateError
		return nil, sdkerr.NewUnavailable("QueryService", "underlying query service is uninitialized")
	}

	if err := s.repoService.Load(cleanPath); err != nil {
		s.state = contracts.StateError
		return nil, sdkerr.Wrap(err, sdkerr.CategoryNotFound, "ERR_OPEN_FAILED", "failed to open repository")
	}

	metaDTO, err := s.repoService.Metadata()
	if err != nil {
		s.state = contracts.StateError
		return nil, sdkerr.Wrap(err, sdkerr.CategoryInternal, "ERR_METADATA_FAILED", "failed to retrieve repository metadata")
	}

	s.repoPath = cleanPath
	s.state = contracts.StateReady
	s.info = &contracts.RepositoryInfo{
		Name:          metaDTO.Name(),
		Owner:         metaDTO.Owner(),
		RootPath:      metaDTO.Root(),
		DefaultBranch: metaDTO.DefaultBranch(),
		CurrentBranch: metaDTO.CurrentBranch(),
		IsGit:         metaDTO.IsGit(),
		State:         contracts.StateReady,
		Languages:     metaDTO.Languages(),
		Capabilities:  metaDTO.Capabilities(),
	}

	return s.info, nil
}

// Close closes the active repository session and releases loaded resources.
func (s *Service) Close(ctx context.Context) error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.state = contracts.StateClosed
	s.info = nil
	s.repoPath = ""

	if s.repoService != nil {
		return s.repoService.Close()
	}
	return nil
}

// Info returns the current snapshot of repository information.
func (s *Service) Info(ctx context.Context) (*contracts.RepositoryInfo, error) {
	if s == nil {
		return nil, sdkerr.NewInvalidState("NIL_SERVICE", "repository service is nil")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.state == contracts.StateUninitialized {
		return nil, sdkerr.NewInvalidState(string(s.state), "no repository currently opened")
	}
	if s.state == contracts.StateClosed {
		return nil, sdkerr.NewInvalidState(string(s.state), "repository session is closed")
	}
	if s.info == nil {
		return nil, sdkerr.NewNotFound("Repository", s.repoPath)
	}

	// Defensive copy
	infoCopy := *s.info
	if s.info.Languages != nil {
		infoCopy.Languages = make([]string, len(s.info.Languages))
		copy(infoCopy.Languages, s.info.Languages)
	}
	if s.info.Capabilities != nil {
		infoCopy.Capabilities = make([]string, len(s.info.Capabilities))
		copy(infoCopy.Capabilities, s.info.Capabilities)
	}

	return &infoCopy, nil
}

// State returns the current operational lifecycle state of the repository.
func (s *Service) State() contracts.RepositoryState {
	if s == nil {
		return contracts.StateClosed
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// Metadata returns descriptive VCS and manifest attributes of the repository.
func (s *Service) Metadata(ctx context.Context) (*contracts.RepositoryMetadata, error) {
	if s == nil {
		return nil, sdkerr.NewInvalidState("NIL_SERVICE", "repository service is nil")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.state != contracts.StateReady || s.repoService == nil {
		return nil, sdkerr.NewInvalidState(string(s.state), "repository is not in ready state")
	}

	metaDTO, err := s.repoService.Metadata()
	if err != nil {
		return nil, sdkerr.Wrap(err, sdkerr.CategoryInternal, "ERR_METADATA_FAILED", "failed to fetch repository metadata")
	}

	return &contracts.RepositoryMetadata{
		Name:        metaDTO.Name(),
		Description: "Repository at " + filepath.Base(metaDTO.Root()),
		Version:     "1.0.0",
		VCS:         map[bool]string{true: "git", false: "filesystem"}[metaDTO.IsGit()],
		Properties: map[string]string{
			"owner":          metaDTO.Owner(),
			"default_branch": metaDTO.DefaultBranch(),
			"current_branch": metaDTO.CurrentBranch(),
			"status":         metaDTO.AnalysisState(),
		},
	}, nil
}

// Statistics returns quantitative measurements of the repository.
func (s *Service) Statistics(ctx context.Context) (*contracts.RepositoryStatistics, error) {
	if s == nil {
		return nil, sdkerr.NewInvalidState("NIL_SERVICE", "repository service is nil")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.state != contracts.StateReady || s.repoService == nil {
		return nil, sdkerr.NewInvalidState(string(s.state), "repository is not in ready state")
	}

	statsDTO, err := s.repoService.Statistics()
	if err != nil {
		return nil, sdkerr.Wrap(err, sdkerr.CategoryInternal, "ERR_STATISTICS_FAILED", "failed to fetch repository statistics")
	}

	return &contracts.RepositoryStatistics{
		TotalFiles:         statsDTO.FileCount(),
		TotalDirectories:   statsDTO.DirectoryCount(),
		TotalPackages:      statsDTO.PackageCount(),
		TotalSymbols:       statsDTO.SymbolCount(),
		TotalRelationships: statsDTO.RelationshipCount(),
		TotalDependencies:  statsDTO.DependencyCount(),
		TotalDocs:          statsDTO.DocCount(),
		TotalConfigs:       statsDTO.ConfigCount(),
	}, nil
}

// Reload re-runs discovery and indexing on the currently opened repository path.
func (s *Service) Reload(ctx context.Context) error {
	if s == nil {
		return sdkerr.NewInvalidState("NIL_SERVICE", "repository service is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.repoPath == "" || s.repoService == nil {
		return sdkerr.NewInvalidState(string(s.state), "no active repository to reload")
	}

	s.state = contracts.StateIndexing
	if err := s.repoService.Load(s.repoPath); err != nil {
		s.state = contracts.StateError
		return sdkerr.Wrap(err, sdkerr.CategoryInternal, "ERR_RELOAD_FAILED", "failed to reload repository")
	}

	s.state = contracts.StateReady
	return nil
}
