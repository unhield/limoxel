package plugintesting_test

import (
	"context"
	"testing"

	"github.com/unhield/limoxel/plugin"
	"github.com/unhield/limoxel/plugin/api"
	plugintesting "github.com/unhield/limoxel/plugin/testing"
)

type sampleTestPlugin struct {
	*plugin.BasePlugin
	started bool
	stopped bool
}

func (s *sampleTestPlugin) Start(ctx context.Context) error {
	_ = s.BasePlugin.Start(ctx)
	s.started = true
	return nil
}

func (s *sampleTestPlugin) Stop(ctx context.Context) error {
	_ = s.BasePlugin.Stop(ctx)
	s.stopped = true
	return nil
}

func TestTestHarness_Lifecycle(t *testing.T) {
	wsDir := t.TempDir()
	harness := plugintesting.NewTestHarness(wsDir)

	manifest := plugin.Manifest{
		ID:      "com.example.harnesstest",
		Name:    "HarnessTest",
		Version: "1.0.0",
	}

	p := &sampleTestPlugin{
		BasePlugin: plugin.NewBasePlugin("com.example.harnesstest", manifest),
	}

	ctx := context.Background()
	if err := harness.ExecuteLifecycle(ctx, p); err != nil {
		t.Fatalf("execute lifecycle failed: %v", err)
	}

	if !p.started || !p.stopped {
		t.Errorf("lifecycle flags mismatch: started=%v, stopped=%v", p.started, p.stopped)
	}
}

func TestMockHost_APIs(t *testing.T) {
	wsDir := t.TempDir()
	host := plugintesting.NewMockHost(wsDir)
	ctx := context.Background()

	// Verify all 6 mock APIs
	info, err := host.RepositoryAPI().Info(ctx)
	if err != nil || info == nil {
		t.Errorf("repo api error: %v", err)
	}

	sym, err := host.SymbolAPI().LookupSymbol(ctx, "TestFunc")
	if err != nil || sym.Name != "TestFunc" {
		t.Errorf("symbol api error: %v", err)
	}

	searchRes, err := host.SearchAPI().SearchSymbols(ctx, "foo", api.PaginationOptions{})
	if err != nil || searchRes == nil {
		t.Errorf("search api error: %v", err)
	}

	nodes, edges, err := host.GraphAPI().GraphInfo(ctx)
	if err != nil || nodes <= 0 || edges <= 0 {
		t.Errorf("graph api error: %v", err)
	}

	health, err := host.AnalysisAPI().RepositoryHealth(ctx)
	if err != nil || health.Grade != "A" {
		t.Errorf("analysis api error: %v", err)
	}

	received := false
	_, err = host.EventAPI().Subscribe(ctx, "test.evt", func(c context.Context, evt api.Event) error {
		received = true
		return nil
	})
	if err != nil {
		t.Errorf("subscribe error: %v", err)
	}
	_ = host.EventAPI().Publish(ctx, api.Event{Type: "test.evt"})
	if !received {
		t.Errorf("expected event to be received by subscriber")
	}
}
