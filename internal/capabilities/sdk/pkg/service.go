package pkg

import (
	"context"
	"path/filepath"
	"strings"
	"sync"

	"github.com/unhield/limoxel/internal/capabilities/repository/query"
	"github.com/unhield/limoxel/internal/capabilities/sdk/contracts"
	sdkerr "github.com/unhield/limoxel/internal/capabilities/sdk/errors"
)

// Service provides the concrete SDK adapter implementation for PackageContract.
type Service struct {
	mu sync.RWMutex
	contracts.BaseContract
	repoService *query.RepositoryService
}

// Ensure Service implements PackageContract.
var _ contracts.PackageContract = (*Service)(nil)

// NewService constructs an initialized Package SDK service adapter.
func NewService(repoService *query.RepositoryService) *Service {
	return &Service{
		BaseContract: contracts.DefaultPackageContractMetadata(),
		repoService:  repoService,
	}
}

// DiscoverPackages retrieves packages within the repository matching the provided filter and pagination.
func (s *Service) DiscoverPackages(ctx context.Context, filter contracts.PackageFilter, opts contracts.PaginationOptions) ([]contracts.PackageInfo, error) {
	if s == nil || s.repoService == nil {
		return nil, sdkerr.NewUnavailable("PackageService", "underlying repository service is unavailable")
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

	res, err := searchEng.SearchPackages(pattern, query.SearchOptions{MaxResults: 1000})
	if err != nil && !strings.Contains(err.Error(), "no matches") {
		return nil, sdkerr.Wrap(err, sdkerr.CategoryInternal, "ERR_DISCOVERY_FAILED", "package discovery failed")
	}

	pkgMap := make(map[string]*contracts.PackageInfo)
	for _, m := range res {
		pkgPath := m.Path()
		if pkgPath == "" {
			pkgPath = m.PackageName()
		}
		if pkgPath == "" {
			pkgPath = m.Name()
		}
		if filter.Module != "" && !strings.HasPrefix(pkgPath, filter.Module) {
			continue
		}

		if _, exists := pkgMap[pkgPath]; !exists {
			pkgName := m.Name()
			if pkgName == "" {
				pkgName = filepath.Base(pkgPath)
			}
			pkgMap[pkgPath] = &contracts.PackageInfo{
				Path:        pkgPath,
				Name:        pkgName,
				DocComment:  m.Snippet(),
				FileCount:   1,
				SymbolCount: 0,
			}
		} else {
			pkgMap[pkgPath].FileCount++
		}
	}

	// Update symbol counts from SymbolAPI
	symAPI := s.repoService.Symbols()
	if symAPI != nil {
		for pPath, pInfo := range pkgMap {
			syms, _ := symAPI.ListSymbols(query.ScopePackage, pPath)
			pInfo.SymbolCount = len(syms)
		}
	}

	results := make([]contracts.PackageInfo, 0, len(pkgMap))
	for _, p := range pkgMap {
		results = append(results, *p)
	}

	// Apply pagination
	start := normOpts.Offset
	if start > len(results) {
		return []contracts.PackageInfo{}, nil
	}
	end := start + normOpts.Limit
	if end > len(results) {
		end = len(results)
	}

	return results[start:end], nil
}

// LookupPackage locates a package by full path or base package name.
func (s *Service) LookupPackage(ctx context.Context, pkgPathOrName string) (*contracts.PackageInfo, error) {
	if s == nil || s.repoService == nil {
		return nil, sdkerr.NewUnavailable("PackageService", "underlying repository service is unavailable")
	}
	clean := strings.TrimSpace(pkgPathOrName)
	if clean == "" {
		return nil, sdkerr.NewInvalidInput("package path or name cannot be empty")
	}

	pkgs, err := s.DiscoverPackages(ctx, contracts.PackageFilter{}, contracts.PaginationOptions{Limit: 1000})
	if err != nil {
		return nil, err
	}

	for _, p := range pkgs {
		if p.Path == clean || p.Name == clean || strings.HasSuffix(p.Path, "/"+clean) {
			return &p, nil
		}
	}

	return nil, sdkerr.NewNotFound("Package", pkgPathOrName)
}

// GetPackageStatistics calculates quantitative metrics for the specified package.
func (s *Service) GetPackageStatistics(ctx context.Context, pkgPathOrName string) (*contracts.PackageStatistics, error) {
	pkgInfo, err := s.LookupPackage(ctx, pkgPathOrName)
	if err != nil {
		return nil, err
	}

	exportedSyms := 0
	internalSyms := 0
	symAPI := s.repoService.Symbols()
	if symAPI != nil {
		syms, _ := symAPI.ListSymbols(query.ScopePackage, pkgInfo.Path)
		for _, sym := range syms {
			if sym.IsExported() {
				exportedSyms++
			} else {
				internalSyms++
			}
		}
	}

	return &contracts.PackageStatistics{
		Path:                pkgInfo.Path,
		FileCount:           pkgInfo.FileCount,
		SymbolCount:         pkgInfo.SymbolCount,
		ExportedSymbolCount: exportedSyms,
		InternalSymbolCount: internalSyms,
		ImportCount:         0,
		DependentCount:      0,
		TestFileCount:       0,
	}, nil
}

// GetPackageHierarchy builds a nested tree of sub-packages under the target package namespace.
func (s *Service) GetPackageHierarchy(ctx context.Context, pkgPathOrName string) (*contracts.PackageHierarchyNode, error) {
	rootPkg, err := s.LookupPackage(ctx, pkgPathOrName)
	if err != nil {
		return nil, err
	}

	allPkgs, _ := s.DiscoverPackages(ctx, contracts.PackageFilter{}, contracts.PaginationOptions{Limit: 1000})
	children := make([]contracts.PackageHierarchyNode, 0)

	prefix := rootPkg.Path + "/"
	for _, p := range allPkgs {
		if strings.HasPrefix(p.Path, prefix) && p.Path != rootPkg.Path {
			children = append(children, contracts.PackageHierarchyNode{
				Package: p,
			})
		}
	}

	return &contracts.PackageHierarchyNode{
		Package:  *rootPkg,
		Children: children,
	}, nil
}

// GetPackageRelationships retrieves dependency connections to and from the package.
func (s *Service) GetPackageRelationships(ctx context.Context, pkgPathOrName string) ([]contracts.PackageRelationship, error) {
	pkgInfo, err := s.LookupPackage(ctx, pkgPathOrName)
	if err != nil {
		return nil, err
	}

	rels := make([]contracts.PackageRelationship, 0)
	if s.repoService.Graph() != nil {
		kgRels, err := s.repoService.Graph().LookupRelationships(pkgInfo.Path, "", "")
		if err == nil {
			for _, r := range kgRels {
				rels = append(rels, contracts.PackageRelationship{
					SourcePackage: r.SourceID(),
					TargetPackage: r.TargetID(),
					Kind:          string(r.Type()),
					Weight:        1,
				})
			}
		}
	}

	return rels, nil
}
