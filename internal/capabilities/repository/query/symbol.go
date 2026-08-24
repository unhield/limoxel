package query

import (
	"sort"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
)

// SymbolAPI provides deterministic query and lookup operations over established Symbol Engine data.
type SymbolAPI struct {
	symModel *symbol.SymbolModel
}

// NewSymbolAPI constructs a SymbolAPI instance.
func NewSymbolAPI(symModel *symbol.SymbolModel) *SymbolAPI {
	return &SymbolAPI{symModel: symModel}
}

// FindSymbol resolves a symbol by its unique canonical ID.
func (api *SymbolAPI) FindSymbol(id string) (*SymbolDTO, error) {
	if api == nil || api.symModel == nil {
		return nil, ErrAnalysisUnavailable
	}
	cleanID := strings.TrimSpace(id)
	if cleanID == "" {
		return nil, ErrInvalidInput
	}

	symDB := api.symModel.Symbols()
	if symDB == nil {
		return nil, ErrAnalysisUnavailable
	}

	sym := symDB.SymbolByID(cleanID)
	if sym == nil {
		return nil, ErrSymbolNotFound
	}

	return toSymbolDTO(sym), nil
}

// ListSymbols returns all symbols available within the specified scope.
func (api *SymbolAPI) ListSymbols(scope ScopeKind, target string) ([]*SymbolDTO, error) {
	if api == nil || api.symModel == nil {
		return nil, ErrAnalysisUnavailable
	}

	symDB := api.symModel.Symbols()
	if symDB == nil {
		return nil, ErrAnalysisUnavailable
	}

	cleanTarget := strings.TrimSpace(target)
	var rawSymbols []*symbol.Symbol

	switch scope {
	case ScopeRepository, "":
		rawSymbols = symDB.AllSymbols()
	case ScopePackage:
		if cleanTarget == "" {
			return nil, ErrInvalidInput
		}
		rawSymbols = symDB.SymbolsByPackage(cleanTarget)
		if len(rawSymbols) == 0 {
			for _, s := range symDB.AllSymbols() {
				if s.PackageName() == cleanTarget || strings.HasSuffix(s.PackagePath(), "/"+cleanTarget) {
					rawSymbols = append(rawSymbols, s)
				}
			}
		}
	case ScopeFile:
		if cleanTarget == "" {
			return nil, ErrInvalidInput
		}
		rawSymbols = symDB.SymbolsByFile(cleanTarget)
	case ScopeModule:
		// Module-level scope matches symbols located in paths prefixed with module target
		for _, s := range symDB.AllSymbols() {
			if cleanTarget == "" || strings.HasPrefix(s.FilePath(), cleanTarget) {
				rawSymbols = append(rawSymbols, s)
			}
		}
	default:
		return nil, ErrInvalidInput
	}

	return toSortedSymbolDTOs(rawSymbols), nil
}

// LookupByType returns all symbols of a specific symbol.SymbolKind.
func (api *SymbolAPI) LookupByType(kind symbol.SymbolKind) ([]*SymbolDTO, error) {
	if api == nil || api.symModel == nil {
		return nil, ErrAnalysisUnavailable
	}
	if kind == "" {
		return nil, ErrInvalidInput
	}

	symDB := api.symModel.Symbols()
	if symDB == nil {
		return nil, ErrAnalysisUnavailable
	}

	rawSymbols := symDB.SymbolsByKind(kind)
	return toSortedSymbolDTOs(rawSymbols), nil
}

// LookupByPackage returns all symbols defined within a package path or name.
func (api *SymbolAPI) LookupByPackage(pkgPath string) ([]*SymbolDTO, error) {
	if api == nil || api.symModel == nil {
		return nil, ErrAnalysisUnavailable
	}
	cleanPkg := strings.TrimSpace(pkgPath)
	if cleanPkg == "" {
		return nil, ErrInvalidInput
	}

	symDB := api.symModel.Symbols()
	if symDB == nil {
		return nil, ErrAnalysisUnavailable
	}

	rawSymbols := symDB.SymbolsByPackage(cleanPkg)
	if len(rawSymbols) == 0 {
		for _, s := range symDB.AllSymbols() {
			if s.PackageName() == cleanPkg || strings.HasSuffix(s.PackagePath(), "/"+cleanPkg) {
				rawSymbols = append(rawSymbols, s)
			}
		}
	}
	return toSortedSymbolDTOs(rawSymbols), nil
}

// LookupByName performs name-based lookup across symbols.
// Returns all matching symbols deterministically sorted by ID; ambiguity remains explicit.
func (api *SymbolAPI) LookupByName(name string) ([]*SymbolDTO, error) {
	if api == nil || api.symModel == nil {
		return nil, ErrAnalysisUnavailable
	}
	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		return nil, ErrInvalidInput
	}

	symDB := api.symModel.Symbols()
	if symDB == nil {
		return nil, ErrAnalysisUnavailable
	}

	rawSymbols := symDB.SymbolsByName(cleanName)
	if len(rawSymbols) == 0 {
		return nil, ErrSymbolNotFound
	}

	return toSortedSymbolDTOs(rawSymbols), nil
}

// Helper: toSymbolDTO converts an authoritative symbol.Symbol to immutable SymbolDTO.
func toSymbolDTO(s *symbol.Symbol) *SymbolDTO {
	if s == nil {
		return nil
	}
	line := 0
	if s.Position() != nil {
		line = s.Position().Line()
	}
	docText := ""
	if s.Doc() != nil {
		docText = s.Doc().Content()
	}
	return NewSymbolDTO(
		s.ID(),
		s.Name(),
		s.Kind(),
		s.FilePath(),
		s.PackageName(),
		s.ReceiverType(),
		s.IsExported(),
		s.Signature(),
		line,
		docText,
	)
}

// Helper: toSortedSymbolDTOs converts and deterministically sorts symbol DTOs by ID.
func toSortedSymbolDTOs(raw []*symbol.Symbol) []*SymbolDTO {
	var dtos []*SymbolDTO
	for _, s := range raw {
		if s != nil {
			dtos = append(dtos, toSymbolDTO(s))
		}
	}
	sort.Slice(dtos, func(i, j int) bool {
		return dtos[i].ID() < dtos[j].ID()
	})
	return dtos
}
