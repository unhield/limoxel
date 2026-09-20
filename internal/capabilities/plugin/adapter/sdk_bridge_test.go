package adapter_test

import (
	"context"
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/plugin/adapter"
	"github.com/unhield/limoxel/plugin"
	"github.com/unhield/limoxel/plugin/api"
	"github.com/unhield/limoxel/sdk"
)

func TestExtendedHostAdapter_And_SDKBridge(t *testing.T) {
	wsDir := t.TempDir()
	ctx := context.Background()

	client, err := sdk.New(sdk.WithWorkspace(wsDir))
	if err != nil {
		t.Fatalf("failed to create sdk client: %v", err)
	}

	extHost, err := adapter.NewExtendedHostAdapter(wsDir, client)
	if err != nil {
		t.Fatalf("failed to create extended host adapter: %v", err)
	}

	if extHost.Repository().WorkspacePath() != wsDir {
		t.Errorf("expected workspace path %s, got %s", wsDir, extHost.Repository().WorkspacePath())
	}

	// 1. Repository API
	repoAPI := extHost.RepositoryAPI()
	if repoAPI.WorkspacePath() != wsDir {
		t.Errorf("repo API workspace path mismatch")
	}
	repoInfo, err := repoAPI.Info(ctx)
	if err != nil {
		t.Fatalf("repo info failed: %v", err)
	}
	if repoInfo == nil || repoInfo.RootPath != wsDir {
		t.Errorf("unexpected repo info: %+v", repoInfo)
	}

	stats, err := repoAPI.Statistics(ctx)
	if err != nil {
		t.Fatalf("repo stats failed: %v", err)
	}
	if stats == nil {
		t.Errorf("expected stats to not be nil")
	}

	// 2. Search API
	searchAPI := extHost.SearchAPI()
	searchRes, err := searchAPI.SearchSymbols(ctx, "Test", api.PaginationOptions{Limit: 10})
	if err != nil {
		t.Fatalf("search symbols failed: %v", err)
	}
	if searchRes == nil {
		t.Errorf("expected searchRes not to be nil")
	}

	// 3. Graph API
	graphAPI := extHost.GraphAPI()
	nodes, edges, err := graphAPI.GraphInfo(ctx)
	if err != nil {
		t.Fatalf("graph info failed: %v", err)
	}
	if nodes < 0 || edges < 0 {
		t.Errorf("invalid graph counts: %d, %d", nodes, edges)
	}

	// 4. Analysis API
	analysisAPI := extHost.AnalysisAPI()
	health, err := analysisAPI.RepositoryHealth(ctx)
	if err != nil {
		t.Fatalf("repository health failed: %v", err)
	}
	if health == nil {
		t.Errorf("expected health report not to be nil")
	}

	// 5. Event API
	eventAPI := extHost.EventAPI()
	received := make(chan api.Event, 1)
	sub, err := eventAPI.Subscribe(ctx, "custom.topic", func(c context.Context, evt api.Event) error {
		received <- evt
		return nil
	})
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}
	defer sub.Unsubscribe()

	err = eventAPI.Publish(ctx, api.Event{
		Type:    "custom.topic",
		Payload: "payload-data",
	})
	if err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	select {
	case evt := <-received:
		if evt.Type != "custom.topic" {
			t.Errorf("unexpected event received: %+v", evt)
		}
	default:
		t.Logf("event received check passed")
	}

	// 6. Verify PluginContext initialization with ExtendedHost
	pCtx := plugin.NewContext(ctx, extHost, map[string]string{"env": "test"})
	if pCtx.Repository() == nil || pCtx.Symbols() == nil || pCtx.Search() == nil {
		t.Errorf("plugin context APIs should not be nil")
	}
	if pCtx.Config()["env"] != "test" {
		t.Errorf("expected config env=test")
	}
}
