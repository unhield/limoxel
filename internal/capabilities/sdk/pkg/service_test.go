package pkg_test

import (
	"context"
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/sdk/contracts"
	"github.com/unhield/limoxel/internal/capabilities/sdk/pkg"
	"github.com/unhield/limoxel/internal/capabilities/sdk/testutil"
)

func TestPackageServiceOperations(t *testing.T) {
	ctx := context.Background()
	querySvc, _ := testutil.SetupTestRepository(t)

	svc := pkg.NewService(querySvc)

	// 1. DiscoverPackages
	pkgs, err := svc.DiscoverPackages(ctx, contracts.PackageFilter{}, contracts.PaginationOptions{Limit: 20})
	if err != nil {
		t.Fatalf("DiscoverPackages failed: %v", err)
	}
	if len(pkgs) == 0 {
		t.Errorf("expected packages to be discovered, got 0")
	}

	// 2. LookupPackage
	pkgInfo, err := svc.LookupPackage(ctx, "math")
	if err != nil {
		t.Fatalf("LookupPackage failed: %v", err)
	}
	if pkgInfo.Name != "math" {
		t.Errorf("got %q, want math", pkgInfo.Name)
	}

	// 3. GetPackageStatistics
	stats, err := svc.GetPackageStatistics(ctx, "math")
	if err != nil {
		t.Fatalf("GetPackageStatistics failed: %v", err)
	}
	if stats.Path == "" {
		t.Errorf("expected non-empty path in stats")
	}

	// 4. GetPackageHierarchy
	hier, err := svc.GetPackageHierarchy(ctx, "math")
	if err != nil {
		t.Fatalf("GetPackageHierarchy failed: %v", err)
	}
	if hier.Package.Name != "math" {
		t.Errorf("hierarchy package mismatch: %s", hier.Package.Name)
	}

	// 5. GetPackageRelationships
	_, err = svc.GetPackageRelationships(ctx, "math")
	if err != nil {
		t.Fatalf("GetPackageRelationships failed: %v", err)
	}
}

func TestPackageServiceErrors(t *testing.T) {
	ctx := context.Background()
	svc := pkg.NewService(nil)

	if _, err := svc.DiscoverPackages(ctx, contracts.PackageFilter{}, contracts.PaginationOptions{}); err == nil {
		t.Errorf("expected error on nil service DiscoverPackages")
	}
	if _, err := svc.LookupPackage(ctx, "nonexistent"); err == nil {
		t.Errorf("expected error on nil service LookupPackage")
	}
	if _, err := svc.LookupPackage(ctx, ""); err == nil {
		t.Errorf("expected error for empty package name")
	}

	var nilSvc *pkg.Service
	if _, err := nilSvc.DiscoverPackages(ctx, contracts.PackageFilter{}, contracts.PaginationOptions{}); err == nil {
		t.Errorf("expected error on typed nil service DiscoverPackages")
	}
	if _, err := nilSvc.LookupPackage(ctx, "math"); err == nil {
		t.Errorf("expected error on typed nil service LookupPackage")
	}
	if _, err := nilSvc.GetPackageStatistics(ctx, "math"); err == nil {
		t.Errorf("expected error on typed nil service GetPackageStatistics")
	}
	if _, err := nilSvc.GetPackageHierarchy(ctx, "math"); err == nil {
		t.Errorf("expected error on typed nil service GetPackageHierarchy")
	}
	if _, err := nilSvc.GetPackageRelationships(ctx, "math"); err == nil {
		t.Errorf("expected error on typed nil service GetPackageRelationships")
	}
}
