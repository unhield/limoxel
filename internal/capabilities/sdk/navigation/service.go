package navigation

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/unhield/limoxel/internal/capabilities/intelligence/navigation"
	"github.com/unhield/limoxel/internal/capabilities/sdk/contracts"
	sdkerr "github.com/unhield/limoxel/internal/capabilities/sdk/errors"
)

// Service adapts internal Navigation intelligence capabilities to the public NavigationContract.
type Service struct {
	mu sync.RWMutex
	contracts.BaseContract
	model  *navigation.NavigationModel
	engine *navigation.Engine
}

// Ensure Service implements NavigationContract.
var _ contracts.NavigationContract = (*Service)(nil)

// NewService constructs a new Navigation SDK service adapter.
func NewService(model *navigation.NavigationModel) *Service {
	return &Service{
		BaseContract: contracts.DefaultNavigationContractMetadata(),
		model:        model,
		engine:       navigation.New(),
	}
}

// SetModel updates the active navigation model thread-safely.
func (s *Service) SetModel(model *navigation.NavigationModel) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.model = model
}

// GoToDefinition resolves the canonical definition target for a symbol.
func (s *Service) GoToDefinition(ctx context.Context, symbolIDOrName string) (*contracts.DefinitionResult, error) {
	if s == nil {
		return nil, sdkerr.NewUnavailable("NavigationService", "navigation service is nil")
	}
	if strings.TrimSpace(symbolIDOrName) == "" {
		return nil, sdkerr.NewInvalidInput("symbolIDOrName cannot be empty")
	}
	if err := ctx.Err(); err != nil {
		return nil, sdkerr.Wrap(err, sdkerr.CategoryInternal, "ERR_CONTEXT_CANCELLED", "context cancelled")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.model == nil {
		return nil, sdkerr.NewUnavailable("NavigationModel", "navigation model is not initialized")
	}

	defRes := s.model.Definition(symbolIDOrName)
	if defRes == nil {
		for symID, d := range s.model.Definitions() {
			if symID == symbolIDOrName || (d.Target() != nil && d.Target().Name() == symbolIDOrName) || strings.HasSuffix(symID, "."+symbolIDOrName) {
				defRes = d
				break
			}
		}
	}
	if defRes == nil {
		return nil, sdkerr.NewNotFound("Definition", symbolIDOrName)
	}

	var tgt *contracts.NavigationTarget
	if defRes.Target() != nil {
		t := convertNavTarget(defRes.Target())
		tgt = &t
	}

	var candidates []contracts.NavigationTarget
	for _, c := range defRes.Candidates() {
		if c != nil {
			candidates = append(candidates, convertNavTarget(c))
		}
	}

	return &contracts.DefinitionResult{
		Target:     tgt,
		Candidates: candidates,
		State:      string(defRes.State()),
	}, nil
}

// FindReferences finds all reference and usage locations for a symbol.
func (s *Service) FindReferences(ctx context.Context, symbolIDOrName string, opts contracts.PaginationOptions) (*contracts.ReferenceResult, error) {
	if s == nil {
		return nil, sdkerr.NewUnavailable("NavigationService", "navigation service is nil")
	}
	if strings.TrimSpace(symbolIDOrName) == "" {
		return nil, sdkerr.NewInvalidInput("symbolIDOrName cannot be empty")
	}
	if err := ctx.Err(); err != nil {
		return nil, sdkerr.Wrap(err, sdkerr.CategoryInternal, "ERR_CONTEXT_CANCELLED", "context cancelled")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.model == nil {
		return nil, sdkerr.NewUnavailable("NavigationModel", "navigation model is not initialized")
	}

	refRes := s.model.References(symbolIDOrName)
	if refRes == nil {
		for symID, d := range s.model.Definitions() {
			if symID == symbolIDOrName || (d.Target() != nil && d.Target().Name() == symbolIDOrName) || strings.HasSuffix(symID, "."+symbolIDOrName) {
				refRes = s.model.References(symID)
				if refRes != nil {
					break
				}
			}
		}
	}
	var targets []contracts.NavigationTarget
	if refRes != nil {
		for _, r := range refRes.References() {
			if r != nil {
				targets = append(targets, convertNavTarget(r))
			}
		}
	}

	// Apply pagination
	total := len(targets)
	norm := opts.Normalize(100, 500)
	offset := norm.Offset
	if offset > total {
		offset = total
	}

	end := offset + norm.Limit
	if end > total {
		end = total
	}

	var paginated []contracts.NavigationTarget
	if offset < total {
		paginated = targets[offset:end]
	}

	return &contracts.ReferenceResult{
		SymbolID:   symbolIDOrName,
		References: paginated,
		TotalCount: total,
	}, nil
}

// CallHierarchy resolves incoming callers and outgoing callees for a symbol.
func (s *Service) CallHierarchy(ctx context.Context, symbolIDOrName string) (*contracts.CallHierarchyNode, error) {
	if s == nil {
		return nil, sdkerr.NewUnavailable("NavigationService", "navigation service is nil")
	}
	if strings.TrimSpace(symbolIDOrName) == "" {
		return nil, sdkerr.NewInvalidInput("symbolIDOrName cannot be empty")
	}
	if err := ctx.Err(); err != nil {
		return nil, sdkerr.Wrap(err, sdkerr.CategoryInternal, "ERR_CONTEXT_CANCELLED", "context cancelled")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.model == nil {
		return nil, sdkerr.NewUnavailable("NavigationModel", "navigation model is not initialized")
	}

	callNode := s.model.CallHierarchyNode(symbolIDOrName)
	if callNode == nil {
		for symID, cn := range s.model.CallHierarchyNodes() {
			if symID == symbolIDOrName || cn.Name() == symbolIDOrName || strings.HasSuffix(symID, "."+symbolIDOrName) {
				callNode = cn
				break
			}
		}
	}
	if callNode == nil {
		return &contracts.CallHierarchyNode{
			Item: contracts.CallHierarchyItem{
				SymbolID: symbolIDOrName,
				Name:     symbolIDOrName,
				Kind:     "function",
			},
			Callers: nil,
			Callees: nil,
		}, nil
	}

	var callers []contracts.CallHierarchyItem
	for _, c := range callNode.IncomingCallers() {
		callers = append(callers, contracts.CallHierarchyItem{
			SymbolID: c,
			Name:     c,
			Kind:     "function",
			Package:  callNode.PackagePath(),
		})
	}

	var callees []contracts.CallHierarchyItem
	for _, c := range callNode.OutgoingCallees() {
		callees = append(callees, contracts.CallHierarchyItem{
			SymbolID: c,
			Name:     c,
			Kind:     "function",
			Package:  callNode.PackagePath(),
		})
	}

	return &contracts.CallHierarchyNode{
		Item: contracts.CallHierarchyItem{
			SymbolID: callNode.SymbolID(),
			Name:     callNode.Name(),
			Kind:     "function",
			Package:  callNode.PackagePath(),
		},
		Callers: callers,
		Callees: callees,
	}, nil
}

// SymbolHierarchy resolves parent and child symbol relationships.
func (s *Service) SymbolHierarchy(ctx context.Context, symbolIDOrName string) (*contracts.NavSymbolHierarchyNode, error) {
	if s == nil {
		return nil, sdkerr.NewUnavailable("NavigationService", "navigation service is nil")
	}
	if strings.TrimSpace(symbolIDOrName) == "" {
		return nil, sdkerr.NewInvalidInput("symbolIDOrName cannot be empty")
	}
	if err := ctx.Err(); err != nil {
		return nil, sdkerr.Wrap(err, sdkerr.CategoryInternal, "ERR_CONTEXT_CANCELLED", "context cancelled")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.model == nil {
		return nil, sdkerr.NewUnavailable("NavigationModel", "navigation model is not initialized")
	}

	node := s.model.SymbolHierarchyNode(symbolIDOrName)
	if node == nil {
		for symID, sn := range s.model.SymbolHierarchyNodes() {
			if symID == symbolIDOrName || sn.Name() == symbolIDOrName || strings.HasSuffix(symID, "."+symbolIDOrName) {
				node = sn
				break
			}
		}
	}
	if node == nil {
		return &contracts.NavSymbolHierarchyNode{
			SymbolID: symbolIDOrName,
			Name:     symbolIDOrName,
			Kind:     "symbol",
		}, nil
	}

	var children []contracts.NavSymbolHierarchyNode
	for _, c := range node.Children() {
		if c != nil {
			children = append(children, contracts.NavSymbolHierarchyNode{
				SymbolID: c.SymbolID(),
				Name:     c.Name(),
				Kind:     c.Kind(),
				ParentID: c.ParentID(),
			})
		}
	}

	return &contracts.NavSymbolHierarchyNode{
		SymbolID: node.SymbolID(),
		Name:     node.Name(),
		Kind:     node.Kind(),
		ParentID: node.ParentID(),
		Children: children,
	}, nil
}

// NavigationContext retrieves enriched contextual intelligence around a symbol.
func (s *Service) NavigationContext(ctx context.Context, symbolIDOrName string) (*contracts.NavigationContextResult, error) {
	if s == nil {
		return nil, sdkerr.NewUnavailable("NavigationService", "navigation service is nil")
	}
	if strings.TrimSpace(symbolIDOrName) == "" {
		return nil, sdkerr.NewInvalidInput("symbolIDOrName cannot be empty")
	}
	if err := ctx.Err(); err != nil {
		return nil, sdkerr.Wrap(err, sdkerr.CategoryInternal, "ERR_CONTEXT_CANCELLED", "context cancelled")
	}

	defRes, err := s.GoToDefinition(ctx, symbolIDOrName)
	if err != nil {
		return nil, err
	}

	var tgt contracts.NavigationTarget
	if defRes.Target != nil {
		tgt = *defRes.Target
	} else if len(defRes.Candidates) > 0 {
		tgt = defRes.Candidates[0]
	}

	refRes, _ := s.FindReferences(ctx, symbolIDOrName, contracts.PaginationOptions{Limit: 10})
	var rels []string
	if refRes != nil {
		for _, r := range refRes.References {
			rels = append(rels, fmt.Sprintf("referenced_by:%s", r.TargetID))
		}
	}

	return &contracts.NavigationContextResult{
		Target:            tgt,
		ContainingFile:    tgt.Location.FilePath,
		ContainingPackage: tgt.Package,
		RelatedSymbols:    refRes.References,
		Relationships:     rels,
	}, nil
}

// Navigate executes a unified navigation query across relationship kinds.
func (s *Service) Navigate(ctx context.Context, symbolID string, relKind string) (*contracts.NavigationResult, error) {
	if s == nil {
		return nil, sdkerr.NewUnavailable("NavigationService", "navigation service is nil")
	}
	if strings.TrimSpace(symbolID) == "" {
		return nil, sdkerr.NewInvalidInput("symbolID cannot be empty")
	}
	if err := ctx.Err(); err != nil {
		return nil, sdkerr.Wrap(err, sdkerr.CategoryInternal, "ERR_CONTEXT_CANCELLED", "context cancelled")
	}

	switch strings.ToLower(relKind) {
	case "def", "definition":
		defRes, err := s.GoToDefinition(ctx, symbolID)
		if err != nil {
			return nil, err
		}
		var targets []contracts.NavigationTarget
		if defRes.Target != nil {
			targets = append(targets, *defRes.Target)
		}
		targets = append(targets, defRes.Candidates...)
		return &contracts.NavigationResult{
			SourceID:     symbolID,
			Relationship: "definition",
			Targets:      targets,
		}, nil

	default: // references
		refRes, err := s.FindReferences(ctx, symbolID, contracts.PaginationOptions{})
		if err != nil {
			return nil, err
		}
		return &contracts.NavigationResult{
			SourceID:     symbolID,
			Relationship: "references",
			Targets:      refRes.References,
		}, nil
	}
}

// Helpers

func convertNavTarget(t *navigation.NavigationTarget) contracts.NavigationTarget {
	if t == nil {
		return contracts.NavigationTarget{}
	}
	var loc contracts.SymbolLocation
	loc.FilePath = t.FilePath()
	if t.Position() != nil {
		loc.StartLine = t.Position().Line()
		loc.EndLine = t.Position().Line()
		loc.StartColumn = t.Position().Column()
		loc.EndColumn = t.Position().Column()
	}

	return contracts.NavigationTarget{
		TargetID:         t.ID(),
		TargetName:       t.Name(),
		TargetKind:       t.Kind(),
		Location:         loc,
		RelationshipKind: string(t.NavKind()),
		Package:          t.PackagePath(),
	}
}
