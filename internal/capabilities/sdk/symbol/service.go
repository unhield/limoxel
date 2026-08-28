package symbol

import (
	"context"
	"strings"
	"sync"

	"github.com/unhield/limoxel/internal/capabilities/repository/query"
	"github.com/unhield/limoxel/internal/capabilities/sdk/contracts"
	sdkerr "github.com/unhield/limoxel/internal/capabilities/sdk/errors"
)

// Service provides the concrete SDK adapter implementation for SymbolContract.
type Service struct {
	mu sync.RWMutex
	contracts.BaseContract
	repoService *query.RepositoryService
}

// Ensure Service implements SymbolContract.
var _ contracts.SymbolContract = (*Service)(nil)

// NewService constructs an initialized Symbol SDK service adapter.
func NewService(repoService *query.RepositoryService) *Service {
	return &Service{
		BaseContract: contracts.DefaultSymbolContractMetadata(),
		repoService:  repoService,
	}
}

// LookupSymbol locates a code symbol by its unique canonical ID or simple name.
func (s *Service) LookupSymbol(ctx context.Context, symbolIDOrName string) (*contracts.SymbolInfo, error) {
	if s == nil || s.repoService == nil {
		return nil, sdkerr.NewUnavailable("SymbolService", "underlying repository service is unavailable")
	}
	clean := strings.TrimSpace(symbolIDOrName)
	if clean == "" {
		return nil, sdkerr.NewInvalidInput("symbol ID or name cannot be empty")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	symAPI := s.repoService.Symbols()
	if symAPI == nil {
		return nil, sdkerr.NewInvalidState("UNINITIALIZED", "symbol API is uninitialized")
	}

	// 1. Attempt exact ID lookup
	symDTO, err := symAPI.FindSymbol(clean)
	if err == nil && symDTO != nil {
		return toContractSymbolInfo(symDTO), nil
	}

	// 2. Attempt name search fallback
	searchEng := s.repoService.Search()
	if searchEng != nil {
		res, err := searchEng.SearchSymbols(clean, query.SearchOptions{MaxResults: 10})
		if err == nil && res != nil && len(res) > 0 {
			for _, m := range res {
				if m.Name() == clean || strings.EqualFold(m.Name(), clean) {
					// Lookup exact symbol by entity ID
					if exactSym, err := symAPI.FindSymbol(m.EntityID()); err == nil && exactSym != nil {
						return toContractSymbolInfo(exactSym), nil
					}
					return &contracts.SymbolInfo{
						ID:         m.EntityID(),
						Name:       m.Name(),
						Package:    m.PackageName(),
						Kind:       contracts.SymbolKind(m.Snippet()),
						DocComment: m.Snippet(),
						Location: contracts.SymbolLocation{
							FilePath: m.Path(),
						},
					}, nil
				}
			}
		}
	}

	return nil, sdkerr.NewNotFound("Symbol", symbolIDOrName)
}

// SymbolHierarchy builds the inheritance, method set, or nesting hierarchy for the specified symbol.
func (s *Service) SymbolHierarchy(ctx context.Context, symbolID string) (*contracts.SymbolHierarchyNode, error) {
	targetSym, err := s.LookupSymbol(ctx, symbolID)
	if err != nil {
		return nil, err
	}

	children := make([]contracts.SymbolHierarchyNode, 0)
	symAPI := s.repoService.Symbols()
	if symAPI != nil {
		allSyms, err := symAPI.ListSymbols(query.ScopeFile, targetSym.Location.FilePath)
		if err == nil {
			for _, sym := range allSyms {
				if sym.Receiver() == targetSym.ID && sym.ID() != targetSym.ID {
					children = append(children, contracts.SymbolHierarchyNode{
						Symbol: *toContractSymbolInfo(sym),
					})
				}
			}
		}
	}

	return &contracts.SymbolHierarchyNode{
		Symbol:   *targetSym,
		Children: children,
	}, nil
}

// SymbolReferences retrieves cross-references or call sites pointing to the specified symbol.
func (s *Service) SymbolReferences(ctx context.Context, symbolID string, opts contracts.PaginationOptions) ([]contracts.SymbolReference, error) {
	targetSym, err := s.LookupSymbol(ctx, symbolID)
	if err != nil {
		return nil, err
	}

	normOpts := opts.Normalize(50, 500)
	refs := make([]contracts.SymbolReference, 0)

	if s.repoService.Graph() != nil {
		rels, err := s.repoService.Graph().LookupRelationships(targetSym.ID, "", "")
		if err == nil {
			for _, r := range rels {
				refs = append(refs, contracts.SymbolReference{
					SourceSymbolID: r.SourceID(),
					SourceFile:     r.SourceID(),
					ReferenceKind:  string(r.Type()),
					Evidence:       string(r.Type()),
				})
			}
		}
	}

	start := normOpts.Offset
	if start > len(refs) {
		return []contracts.SymbolReference{}, nil
	}
	end := start + normOpts.Limit
	if end > len(refs) {
		end = len(refs)
	}

	return refs[start:end], nil
}

// SymbolDocumentation retrieves formatted documentation or doc comment for the specified symbol.
func (s *Service) SymbolDocumentation(ctx context.Context, symbolID string) (string, error) {
	targetSym, err := s.LookupSymbol(ctx, symbolID)
	if err != nil {
		return "", err
	}
	return targetSym.DocComment, nil
}

// SymbolOwnership retrieves the owning package, struct, or interface containing the symbol.
func (s *Service) SymbolOwnership(ctx context.Context, symbolID string) (string, error) {
	targetSym, err := s.LookupSymbol(ctx, symbolID)
	if err != nil {
		return "", err
	}
	if targetSym.Ownership != "" {
		return targetSym.Ownership, nil
	}
	return targetSym.Package, nil
}

func toContractSymbolInfo(sym *query.SymbolDTO) *contracts.SymbolInfo {
	if sym == nil {
		return nil
	}
	return &contracts.SymbolInfo{
		ID:         sym.ID(),
		Name:       sym.Name(),
		Package:    sym.PackageName(),
		Kind:       contracts.SymbolKind(sym.Kind()),
		Signature:  sym.Signature(),
		IsExported: sym.IsExported(),
		DocComment: sym.Doc(),
		Ownership:  sym.Receiver(),
		Location: contracts.SymbolLocation{
			FilePath:  sym.FilePath(),
			StartLine: sym.Line(),
			EndLine:   sym.Line(),
		},
	}
}
