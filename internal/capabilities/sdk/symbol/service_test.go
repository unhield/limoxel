package symbol_test

import (
	"context"
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/sdk/contracts"
	"github.com/unhield/limoxel/internal/capabilities/sdk/symbol"
	"github.com/unhield/limoxel/internal/capabilities/sdk/testutil"
)

func TestSymbolServiceOperations(t *testing.T) {
	ctx := context.Background()
	querySvc, _ := testutil.SetupTestRepository(t)

	svc := symbol.NewService(querySvc)

	// 1. LookupSymbol
	sym, err := svc.LookupSymbol(ctx, "Add")
	if err != nil {
		t.Fatalf("LookupSymbol failed for Add: %v", err)
	}
	if sym.Name != "Add" {
		t.Errorf("got %q, want Add", sym.Name)
	}

	// 2. SymbolHierarchy
	hier, err := svc.SymbolHierarchy(ctx, sym.ID)
	if err != nil {
		t.Fatalf("SymbolHierarchy failed: %v", err)
	}
	if hier.Symbol.Name != "Add" {
		t.Errorf("hierarchy symbol mismatch: %s", hier.Symbol.Name)
	}

	// 3. SymbolReferences
	_, err = svc.SymbolReferences(ctx, sym.ID, contracts.PaginationOptions{})
	if err != nil {
		t.Fatalf("SymbolReferences failed: %v", err)
	}

	// 4. SymbolDocumentation
	doc, err := svc.SymbolDocumentation(ctx, sym.ID)
	if err != nil {
		t.Fatalf("SymbolDocumentation failed: %v", err)
	}
	if doc == "" {
		t.Logf("symbol doc comment is empty (acceptable for basic AST)")
	}

	// 5. SymbolOwnership
	own, err := svc.SymbolOwnership(ctx, sym.ID)
	if err != nil {
		t.Fatalf("SymbolOwnership failed: %v", err)
	}
	if own == "" {
		t.Errorf("expected non-empty ownership")
	}
}

func TestSymbolServiceErrors(t *testing.T) {
	ctx := context.Background()
	svc := symbol.NewService(nil)

	if _, err := svc.LookupSymbol(ctx, "Add"); err == nil {
		t.Errorf("expected error on nil service LookupSymbol")
	}
	if _, err := svc.LookupSymbol(ctx, ""); err == nil {
		t.Errorf("expected error for empty symbol ID")
	}

	var nilSvc *symbol.Service
	if _, err := nilSvc.LookupSymbol(ctx, "Add"); err == nil {
		t.Errorf("expected error on typed nil service LookupSymbol")
	}
	if _, err := nilSvc.SymbolHierarchy(ctx, "Add"); err == nil {
		t.Errorf("expected error on typed nil service SymbolHierarchy")
	}
	if _, err := nilSvc.SymbolReferences(ctx, "Add", contracts.PaginationOptions{}); err == nil {
		t.Errorf("expected error on typed nil service SymbolReferences")
	}
	if _, err := nilSvc.SymbolDocumentation(ctx, "Add"); err == nil {
		t.Errorf("expected error on typed nil service SymbolDocumentation")
	}
	if _, err := nilSvc.SymbolOwnership(ctx, "Add"); err == nil {
		t.Errorf("expected error on typed nil service SymbolOwnership")
	}
}
