package file

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/unhield/limoxel/internal/capabilities/repository/query"
	"github.com/unhield/limoxel/internal/capabilities/sdk/contracts"
	sdkerr "github.com/unhield/limoxel/internal/capabilities/sdk/errors"
)

// Service provides the concrete SDK adapter implementation for FileContract.
type Service struct {
	mu sync.RWMutex
	contracts.BaseContract
	repoService *query.RepositoryService
}

// Ensure Service implements FileContract.
var _ contracts.FileContract = (*Service)(nil)

// NewService constructs an initialized File SDK service adapter.
func NewService(repoService *query.RepositoryService) *Service {
	return &Service{
		BaseContract: contracts.DefaultFileContractMetadata(),
		repoService:  repoService,
	}
}

// DiscoverFiles retrieves files in the repository matching the provided filter and pagination.
func (s *Service) DiscoverFiles(ctx context.Context, filter contracts.FileFilter, opts contracts.PaginationOptions) ([]contracts.FileInfo, error) {
	if s == nil || s.repoService == nil {
		return nil, sdkerr.NewUnavailable("FileService", "underlying repository service is unavailable")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	searchEng := s.repoService.Search()
	if searchEng == nil {
		return nil, sdkerr.NewInvalidState("UNINITIALIZED", "repository search engine is uninitialized")
	}

	normOpts := opts.Normalize(50, 500)
	pattern := strings.TrimSpace(filter.Pattern)
	if pattern == "" {
		pattern = "*"
	}

	searchRes, err := searchEng.SearchFiles(pattern, query.SearchOptions{
		MaxResults: 10000,
	})
	if err != nil && !strings.Contains(err.Error(), "no matches") {
		return nil, sdkerr.Wrap(err, sdkerr.CategoryInternal, "ERR_DISCOVERY_FAILED", "file discovery failed")
	}

	matches := make([]contracts.FileInfo, 0)
	for _, item := range searchRes {
		relPath := item.Path()
		if !filter.IncludeTests && (strings.HasSuffix(relPath, "_test.go") || strings.Contains(relPath, "testdata")) {
			continue
		}
		if filter.Language != "" && !strings.EqualFold(filepath.Ext(relPath), "."+strings.TrimPrefix(filter.Language, ".")) {
			continue
		}
		if filter.Package != "" && !strings.Contains(item.PackageName(), filter.Package) {
			continue
		}

		matches = append(matches, contracts.FileInfo{
			Path:      relPath,
			Name:      filepath.Base(relPath),
			Extension: filepath.Ext(relPath),
			Size:      0,
			Package:   item.PackageName(),
		})
	}

	// Apply pagination
	start := normOpts.Offset
	if start > len(matches) {
		return []contracts.FileInfo{}, nil
	}
	end := start + normOpts.Limit
	if end > len(matches) {
		end = len(matches)
	}

	return matches[start:end], nil
}

// LookupFile locates a file by repository-relative or absolute path.
func (s *Service) LookupFile(ctx context.Context, path string) (*contracts.FileInfo, error) {
	if s == nil || s.repoService == nil {
		return nil, sdkerr.NewUnavailable("FileService", "underlying repository service is unavailable")
	}
	cleanPath := filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
	if cleanPath == "" || cleanPath == "." {
		return nil, sdkerr.NewInvalidInput("file path cannot be empty")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	searchEng := s.repoService.Search()
	if searchEng == nil {
		return nil, sdkerr.NewInvalidState("UNINITIALIZED", "repository search engine is uninitialized")
	}

	res, err := searchEng.SearchFiles(filepath.Base(cleanPath), query.SearchOptions{MaxResults: 100})
	if err != nil {
		return nil, sdkerr.NewNotFound("File", path)
	}

	for _, m := range res {
		if filepath.ToSlash(m.Path()) == cleanPath || strings.HasSuffix(filepath.ToSlash(m.Path()), "/"+cleanPath) {
			return &contracts.FileInfo{
				Path:      m.Path(),
				Name:      filepath.Base(m.Path()),
				Extension: filepath.Ext(m.Path()),
				Package:   m.PackageName(),
			}, nil
		}
	}

	return nil, sdkerr.NewNotFound("File", path)
}

// GetFileMetadata retrieves detailed filesystem metadata and classification attributes for a file.
func (s *Service) GetFileMetadata(ctx context.Context, path string) (*contracts.FileMetadata, error) {
	fileInfo, err := s.LookupFile(ctx, path)
	if err != nil {
		return nil, err
	}

	relPath := fileInfo.Path
	isTest := strings.HasSuffix(relPath, "_test.go") || strings.Contains(relPath, "testdata")
	isGen := strings.Contains(relPath, ".pb.go") || strings.Contains(relPath, "_generated.go")
	isVendor := strings.HasPrefix(relPath, "vendor/") || strings.Contains(relPath, "/vendor/")

	var size int64
	var modTime os.FileInfo
	if fi, statErr := os.Stat(relPath); statErr == nil {
		size = fi.Size()
		modTime = fi
	}
	fileInfo.Size = size

	meta := &contracts.FileMetadata{
		File:        *fileInfo,
		IsTest:      isTest,
		IsGenerated: isGen,
		IsVendor:    isVendor,
		Properties: map[string]string{
			"relative_path": relPath,
			"extension":     fileInfo.Extension,
		},
	}
	if modTime != nil {
		meta.LastModified = modTime.ModTime()
	}

	return meta, nil
}

// GetFileIndexStatus reports the indexing state and symbol count for a file.
func (s *Service) GetFileIndexStatus(ctx context.Context, path string) (*contracts.FileIndexStatus, error) {
	fileInfo, err := s.LookupFile(ctx, path)
	if err != nil {
		return nil, err
	}

	symAPI := s.repoService.Symbols()
	symCount := 0
	if symAPI != nil {
		syms, _ := symAPI.ListSymbols(query.ScopeFile, fileInfo.Path)
		symCount = len(syms)
	}

	return &contracts.FileIndexStatus{
		Path:        fileInfo.Path,
		IsIndexed:   true,
		SymbolCount: symCount,
	}, nil
}

// GetFileRelationships retrieves relationships connected to the specified file.
func (s *Service) GetFileRelationships(ctx context.Context, path string) ([]contracts.FileRelationship, error) {
	fileInfo, err := s.LookupFile(ctx, path)
	if err != nil {
		return nil, err
	}

	rels := make([]contracts.FileRelationship, 0)
	if s.repoService.Graph() != nil {
		kgRels, err := s.repoService.Graph().LookupRelationships(fileInfo.Path, "", "")
		if err == nil {
			for _, r := range kgRels {
				rels = append(rels, contracts.FileRelationship{
					SourcePath: r.SourceID(),
					TargetPath: r.TargetID(),
					Kind:       string(r.Type()),
					Weight:     1,
				})
			}
		}
	}

	return rels, nil
}
