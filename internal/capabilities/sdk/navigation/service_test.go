package navigation_test

import (
	"context"
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/intelligence/navigation"
	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
	"github.com/unhield/limoxel/internal/capabilities/sdk/contracts"
	navsdk "github.com/unhield/limoxel/internal/capabilities/sdk/navigation"
)

func createTestNavigationModel() *navigation.NavigationModel {
	pos := symbol.NewSourcePosition("pkg/math/math.go", 10, 5, 120)
	t1 := navigation.NewNavigationTarget("sym:math.Add", "sym:math.Add", "Add", "function", "pkg/math/math.go", "pkg/math", "sample_repo", "sample_repo", pos, navigation.NavStateValid, navigation.NavKindDefinition, "test")
	t2 := navigation.NewNavigationTarget("main.go:5", "sym:math.Add", "Add", "call", "main.go", "main", "sample_repo", "sample_repo", pos, navigation.NavStateValid, navigation.NavKindReference, "test")

	defRes := navigation.NewDefinitionResult(t1, nil, navigation.NavStateValid, "test")
	refRes := navigation.NewReferenceResult("sym:math.Add", []*navigation.NavigationTarget{t2})

	defs := map[string]*navigation.DefinitionResult{"sym:math.Add": defRes}
	refs := map[string]*navigation.ReferenceResult{"sym:math.Add": refRes}

	callNode := navigation.NewCallHierarchyNode("sym:math.Add", "Add", "pkg/math", "pkg/math/math.go", nil, nil, 1)
	calls := map[string]*navigation.CallHierarchyNode{"sym:math.Add": callNode}

	symNode := navigation.NewSymbolHierarchyNode("sym:math.Add", "Add", "function", "pkg/math/math.go", "pkg/math", "", nil)
	symHiers := map[string]*navigation.SymbolHierarchyNode{"sym:math.Add": symNode}

	return navigation.NewNavigationModel(defs, refs, nil, symHiers, nil, nil, nil, calls, nil)
}

func TestNavigationServiceOperations(t *testing.T) {
	ctx := context.Background()
	model := createTestNavigationModel()
	svc := navsdk.NewService(model)

	// 1. GoToDefinition
	def, err := svc.GoToDefinition(ctx, "sym:math.Add")
	if err != nil {
		t.Fatalf("GoToDefinition failed: %v", err)
	}
	if def.Target == nil || def.Target.TargetName != "Add" {
		t.Errorf("definition target mismatch: %+v", def.Target)
	}

	// 2. FindReferences
	refs, err := svc.FindReferences(ctx, "sym:math.Add", contracts.PaginationOptions{})
	if err != nil {
		t.Fatalf("FindReferences failed: %v", err)
	}
	if refs.TotalCount != 1 {
		t.Errorf("expected 1 reference, got %d", refs.TotalCount)
	}

	// 3. CallHierarchy
	callNode, err := svc.CallHierarchy(ctx, "sym:math.Add")
	if err != nil {
		t.Fatalf("CallHierarchy failed: %v", err)
	}
	if callNode.Item.Name != "Add" {
		t.Errorf("got %q, want Add", callNode.Item.Name)
	}

	// 4. SymbolHierarchy
	symNode, err := svc.SymbolHierarchy(ctx, "sym:math.Add")
	if err != nil {
		t.Fatalf("SymbolHierarchy failed: %v", err)
	}
	if symNode.Name != "Add" {
		t.Errorf("got %q, want Add", symNode.Name)
	}

	// 5. NavigationContext
	navCtx, err := svc.NavigationContext(ctx, "sym:math.Add")
	if err != nil {
		t.Fatalf("NavigationContext failed: %v", err)
	}
	if navCtx.Target.TargetName != "Add" {
		t.Errorf("context target mismatch: %+v", navCtx.Target)
	}

	// 6. Navigate
	navRes, err := svc.Navigate(ctx, "sym:math.Add", "def")
	if err != nil {
		t.Fatalf("Navigate failed: %v", err)
	}
	if len(navRes.Targets) == 0 {
		t.Errorf("expected at least 1 target from Navigate(def)")
	}
}

func TestNavigationServiceErrorsAndNil(t *testing.T) {
	ctx := context.Background()
	svc := navsdk.NewService(nil)

	if _, err := svc.GoToDefinition(ctx, ""); err == nil {
		t.Errorf("expected error for empty symbol")
	}
	if _, err := svc.GoToDefinition(ctx, "missing"); err == nil {
		t.Errorf("expected error on uninitialized GoToDefinition")
	}
	if _, err := svc.FindReferences(ctx, "", contracts.PaginationOptions{}); err == nil {
		t.Errorf("expected error for empty symbol")
	}
	if _, err := svc.CallHierarchy(ctx, ""); err == nil {
		t.Errorf("expected error for empty symbol")
	}
	if _, err := svc.SymbolHierarchy(ctx, ""); err == nil {
		t.Errorf("expected error for empty symbol")
	}
	if _, err := svc.NavigationContext(ctx, ""); err == nil {
		t.Errorf("expected error for empty symbol")
	}
	if _, err := svc.Navigate(ctx, "", "def"); err == nil {
		t.Errorf("expected error for empty symbol")
	}

	var nilSvc *navsdk.Service
	if _, err := nilSvc.GoToDefinition(ctx, "sym"); err == nil {
		t.Errorf("expected error on typed nil service GoToDefinition")
	}
	if _, err := nilSvc.FindReferences(ctx, "sym", contracts.PaginationOptions{}); err == nil {
		t.Errorf("expected error on typed nil service FindReferences")
	}
	if _, err := nilSvc.CallHierarchy(ctx, "sym"); err == nil {
		t.Errorf("expected error on typed nil service CallHierarchy")
	}
	if _, err := nilSvc.SymbolHierarchy(ctx, "sym"); err == nil {
		t.Errorf("expected error on typed nil service SymbolHierarchy")
	}
	if _, err := nilSvc.NavigationContext(ctx, "sym"); err == nil {
		t.Errorf("expected error on typed nil service NavigationContext")
	}
	if _, err := nilSvc.Navigate(ctx, "sym", "def"); err == nil {
		t.Errorf("expected error on typed nil service Navigate")
	}
}
