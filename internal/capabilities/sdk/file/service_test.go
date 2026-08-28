package file_test

import (
	"context"
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/sdk/contracts"
	"github.com/unhield/limoxel/internal/capabilities/sdk/file"
	"github.com/unhield/limoxel/internal/capabilities/sdk/testutil"
)

func TestFileServiceOperations(t *testing.T) {
	ctx := context.Background()
	querySvc, _ := testutil.SetupTestRepository(t)

	svc := file.NewService(querySvc)

	// 1. DiscoverFiles
	files, err := svc.DiscoverFiles(ctx, contracts.FileFilter{}, contracts.PaginationOptions{Limit: 20})
	if err != nil {
		t.Fatalf("DiscoverFiles failed: %v", err)
	}
	if len(files) == 0 {
		t.Errorf("expected files to be discovered, got 0")
	}

	// 2. LookupFile
	fileInfo, err := svc.LookupFile(ctx, "main.go")
	if err != nil {
		t.Fatalf("LookupFile failed for main.go: %v", err)
	}
	if fileInfo.Name != "main.go" {
		t.Errorf("got %q, want main.go", fileInfo.Name)
	}

	// 3. GetFileMetadata
	meta, err := svc.GetFileMetadata(ctx, "main.go")
	if err != nil {
		t.Fatalf("GetFileMetadata failed: %v", err)
	}
	if meta.IsTest {
		t.Errorf("expected main.go not to be a test file")
	}

	// 4. GetFileIndexStatus
	status, err := svc.GetFileIndexStatus(ctx, "pkg/math/math.go")
	if err != nil {
		t.Fatalf("GetFileIndexStatus failed: %v", err)
	}
	if !status.IsIndexed {
		t.Errorf("expected file to be indexed")
	}

	// 5. GetFileRelationships
	_, err = svc.GetFileRelationships(ctx, "main.go")
	if err != nil {
		t.Fatalf("GetFileRelationships failed: %v", err)
	}
}

func TestFileServiceErrors(t *testing.T) {
	ctx := context.Background()
	svc := file.NewService(nil)

	if _, err := svc.DiscoverFiles(ctx, contracts.FileFilter{}, contracts.PaginationOptions{}); err == nil {
		t.Errorf("expected error on nil service DiscoverFiles")
	}
	if _, err := svc.LookupFile(ctx, "main.go"); err == nil {
		t.Errorf("expected error on nil service LookupFile")
	}
	if _, err := svc.LookupFile(ctx, ""); err == nil {
		t.Errorf("expected error for empty file path")
	}

	var nilSvc *file.Service
	if _, err := nilSvc.DiscoverFiles(ctx, contracts.FileFilter{}, contracts.PaginationOptions{}); err == nil {
		t.Errorf("expected error on typed nil service DiscoverFiles")
	}
	if _, err := nilSvc.LookupFile(ctx, "main.go"); err == nil {
		t.Errorf("expected error on typed nil service LookupFile")
	}
	if _, err := nilSvc.GetFileMetadata(ctx, "main.go"); err == nil {
		t.Errorf("expected error on typed nil service GetFileMetadata")
	}
	if _, err := nilSvc.GetFileIndexStatus(ctx, "main.go"); err == nil {
		t.Errorf("expected error on typed nil service GetFileIndexStatus")
	}
	if _, err := nilSvc.GetFileRelationships(ctx, "main.go"); err == nil {
		t.Errorf("expected error on typed nil service GetFileRelationships")
	}
}
