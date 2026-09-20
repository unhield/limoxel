package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/unhield/limoxel/plugin/api"
)

func TestAPITypesAndConstants(t *testing.T) {
	if api.SymbolKindFunction != "function" {
		t.Errorf("unexpected SymbolKindFunction: %v", api.SymbolKindFunction)
	}
	if api.SearchDomainUnified != "unified" {
		t.Errorf("unexpected SearchDomainUnified: %v", api.SearchDomainUnified)
	}
	if api.GraphExportMermaid != "mermaid" {
		t.Errorf("unexpected GraphExportMermaid: %v", api.GraphExportMermaid)
	}

	evt := api.Event{
		ID:        "evt-1",
		Type:      "test.event",
		Timestamp: time.Now().UTC(),
		Workspace: "/test/ws",
		Payload:   map[string]string{"key": "value"},
	}
	if evt.ID != "evt-1" {
		t.Errorf("unexpected event ID: %s", evt.ID)
	}
}

// Ensure mock implementations satisfy the interfaces
type mockRepoAPI struct{}

func (m *mockRepoAPI) WorkspacePath() string                                         { return "/ws" }
func (m *mockRepoAPI) Info(ctx context.Context) (*api.RepositoryInfo, error)         { return nil, nil }
func (m *mockRepoAPI) Metadata(ctx context.Context) (*api.RepositoryMetadata, error) { return nil, nil }
func (m *mockRepoAPI) Statistics(ctx context.Context) (*api.RepositoryStatistics, error) {
	return nil, nil
}
func (m *mockRepoAPI) ListFiles(ctx context.Context, prefix string) ([]string, error) {
	return nil, nil
}
func (m *mockRepoAPI) GetFile(ctx context.Context, relPath string) (*api.FileInfo, error) {
	return nil, nil
}

var _ api.RepositoryAPI = (*mockRepoAPI)(nil)
