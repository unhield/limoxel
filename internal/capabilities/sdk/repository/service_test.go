package repository_test

import (
	"context"
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/sdk/contracts"
	"github.com/unhield/limoxel/internal/capabilities/sdk/repository"
	"github.com/unhield/limoxel/internal/capabilities/sdk/testutil"
)

func TestRepositoryServiceLifecycleAndOperations(t *testing.T) {
	ctx := context.Background()
	querySvc, repoRoot := testutil.SetupTestRepository(t)

	svc := repository.NewService(querySvc)
	if svc.State() != contracts.StateUninitialized {
		t.Errorf("expected StateUninitialized, got %v", svc.State())
	}

	info, err := svc.Open(ctx, repoRoot)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	if info.RootPath != repoRoot {
		t.Errorf("got root %q, want %q", info.RootPath, repoRoot)
	}
	if svc.State() != contracts.StateReady {
		t.Errorf("expected StateReady, got %v", svc.State())
	}

	fetchedInfo, err := svc.Info(ctx)
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}
	if fetchedInfo.Name != info.Name {
		t.Errorf("info name mismatch: %s vs %s", fetchedInfo.Name, info.Name)
	}

	meta, err := svc.Metadata(ctx)
	if err != nil {
		t.Fatalf("Metadata failed: %v", err)
	}
	if meta.Name != info.Name {
		t.Errorf("metadata name mismatch: %s vs %s", meta.Name, info.Name)
	}

	stats, err := svc.Statistics(ctx)
	if err != nil {
		t.Fatalf("Statistics failed: %v", err)
	}
	if stats.TotalFiles == 0 {
		t.Errorf("expected TotalFiles > 0, got %d", stats.TotalFiles)
	}

	if err := svc.Reload(ctx); err != nil {
		t.Fatalf("Reload failed: %v", err)
	}
	if svc.State() != contracts.StateReady {
		t.Errorf("expected StateReady after reload, got %v", svc.State())
	}

	if err := svc.Close(ctx); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if svc.State() != contracts.StateClosed {
		t.Errorf("expected StateClosed, got %v", svc.State())
	}

	// Operations on closed service should fail gracefully
	if _, err := svc.Info(ctx); err == nil {
		t.Errorf("expected error on Info() after close")
	}
	if _, err := svc.Metadata(ctx); err == nil {
		t.Errorf("expected error on Metadata() after close")
	}
	if _, err := svc.Statistics(ctx); err == nil {
		t.Errorf("expected error on Statistics() after close")
	}
}

func TestRepositoryServiceNilAndErrors(t *testing.T) {
	ctx := context.Background()
	svc := repository.NewService(nil)

	if _, err := svc.Open(ctx, ""); err == nil {
		t.Errorf("expected error opening empty path")
	}
	if _, err := svc.Open(ctx, "non_existent_path"); err == nil {
		t.Errorf("expected error opening with nil query service")
	}
	if _, err := svc.Metadata(ctx); err == nil {
		t.Errorf("expected error for uninitialized metadata")
	}
	if _, err := svc.Statistics(ctx); err == nil {
		t.Errorf("expected error for uninitialized statistics")
	}
	if err := svc.Reload(ctx); err == nil {
		t.Errorf("expected error on Reload with nil query service")
	}

	var nilSvc *repository.Service
	if nilSvc.State() != contracts.StateClosed {
		t.Errorf("expected StateClosed on nil repository service")
	}
	if err := nilSvc.Close(ctx); err != nil {
		t.Errorf("unexpected error on nil service Close: %v", err)
	}
	if _, err := nilSvc.Open(ctx, "path"); err == nil {
		t.Errorf("expected error on nil service Open")
	}
	if _, err := nilSvc.Info(ctx); err == nil {
		t.Errorf("expected error on nil service Info")
	}
	if _, err := nilSvc.Metadata(ctx); err == nil {
		t.Errorf("expected error on nil service Metadata")
	}
	if _, err := nilSvc.Statistics(ctx); err == nil {
		t.Errorf("expected error on nil service Statistics")
	}
	if err := nilSvc.Reload(ctx); err == nil {
		t.Errorf("expected error on nil service Reload")
	}
}
