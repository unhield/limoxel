package sdk_test

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/unhield/limoxel/sdk"
)

func TestSecurity_InvalidPathHandling(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Non-existent path fails fast
	nonExistent := filepath.Join(t.TempDir(), "does_not_exist_12345")
	client, err := sdk.OpenWorkspace(ctx, nonExistent)
	if err == nil {
		if client != nil {
			_ = client.Close()
		}
		t.Error("expected error when opening non-existent workspace path, got nil")
	}

	// 2. Valid isolated temp dir
	validTemp := t.TempDir()
	clientValid, err := sdk.New(sdk.WithWorkspace(validTemp))
	if err != nil {
		t.Fatalf("sdk.New with temp dir failed: %v", err)
	}
	defer clientValid.Close()
	if clientValid.Workspace() != validTemp {
		t.Errorf("expected workspace %q, got %q", validTemp, clientValid.Workspace())
	}
}

func TestSecurity_ContextCancellation(t *testing.T) {
	// Pre-cancelled context must fail immediately
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	repoPath := createPerfSampleRepo(t)
	client, err := sdk.OpenWorkspace(cancelledCtx, repoPath)
	if err == nil {
		if client != nil {
			_ = client.Close()
		}
		t.Error("expected error with pre-cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		t.Logf("Returned error under cancelled context: %v", err)
	}
}

func TestSecurity_ClientCloseIdempotence(t *testing.T) {
	repoPath := createPerfSampleRepo(t)
	client, err := sdk.New(sdk.WithWorkspace(repoPath))
	if err != nil {
		t.Fatalf("sdk.New failed: %v", err)
	}

	// First close
	if err := client.Close(); err != nil {
		t.Errorf("first Close() failed: %v", err)
	}

	// Second close must be safe and idempotent
	if err := client.Close(); err != nil {
		t.Errorf("second Close() should be safe and return nil, got: %v", err)
	}
}

func TestSecurity_MalformedJSONPayloadDefense(t *testing.T) {
	// Malformed integer for stats
	var stats sdk.RepositoryStatistics
	if err := json.Unmarshal([]byte(`{"total_files": "not-an-integer"}`), &stats); err == nil {
		t.Error("expected error unmarshaling string into int field, got nil")
	}

	// Malformed float for health report
	var health sdk.RepositoryHealthReport
	if err := json.Unmarshal([]byte(`{"overall_score": [1, 2, 3]}`), &health); err == nil {
		t.Error("expected error unmarshaling array into float field, got nil")
	}

	// Syntax error
	if err := json.Unmarshal([]byte(`{incomplete-json`), &stats); err == nil {
		t.Error("expected error on broken JSON syntax, got nil")
	}
}

func TestSecurity_PaginationBoundsHandling(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	repoPath := createPerfSampleRepo(t)
	client, err := sdk.OpenWorkspace(ctx, repoPath)
	if err != nil {
		t.Fatalf("OpenWorkspace failed: %v", err)
	}
	defer client.Close()

	// Negative limit or offset should be safely handled
	opts := sdk.PaginationOptions{
		Limit:  -100,
		Offset: -50,
	}
	res, err := client.Search().SearchSymbols(ctx, "Execute", opts)
	if err != nil {
		t.Logf("Negative pagination error handled safely: %v", err)
	} else if res != nil {
		t.Logf("Negative pagination returned: %d matches", res.TotalMatches)
	}
}
