package plugin_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	capPlugin "github.com/unhield/limoxel/internal/capabilities/plugin"
	"github.com/unhield/limoxel/internal/capabilities/plugin/loader"
	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
	"github.com/unhield/limoxel/plugin"
)

type concurrentDummyPlugin struct {
	id string
}

func (p *concurrentDummyPlugin) ID() string                                       { return p.id }
func (p *concurrentDummyPlugin) Manifest() plugin.Manifest                        { return plugin.Manifest{ID: p.id} }
func (p *concurrentDummyPlugin) Init(ctx context.Context, host plugin.Host) error { return nil }
func (p *concurrentDummyPlugin) Start(ctx context.Context) error                  { return nil }
func (p *concurrentDummyPlugin) Stop(ctx context.Context) error                   { return nil }

func TestService_ConcurrencyStress(t *testing.T) {
	ctx := context.Background()
	ldr := loader.NewInProcessLoader()
	svc, err := capPlugin.NewService(capPlugin.ServiceOptions{
		WorkspaceRoot: t.TempDir(),
		Loader:        ldr,
	})
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}

	numPlugins := 10
	for i := 0; i < numPlugins; i++ {
		idStr := fmt.Sprintf("org.limoxel.worker%d", i)
		id := model.MustParseIdentity(idStr)
		manifestJSON := fmt.Sprintf(`{
			"schema_version": "1.0.0",
			"id": "%s",
			"name": "Worker %d",
			"version": "1.0.0"
		}`, idStr, i)
		m, err := model.ParseManifestJSON([]byte(manifestJSON))
		if err != nil {
			t.Fatalf("ParseManifestJSON(%s) failed: %v", idStr, err)
		}
		ldr.RegisterFactory(id, func() (plugin.Plugin, error) {
			return &concurrentDummyPlugin{id: idStr}, nil
		})

		if _, installErr := svc.InstallManifest(ctx, m, t.TempDir()); installErr != nil {
			t.Fatalf("InstallManifest(%s) failed: %v", idStr, installErr)
		}
	}

	// Concurrent operations: Activate, Suspend, Get, List, Publish
	var wg sync.WaitGroup
	routines := 20
	iterations := 50

	for r := 0; r < routines; r++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()
			for it := 0; it < iterations; it++ {
				targetIdx := (routineID + it) % numPlugins
				targetID := model.MustParseIdentity(fmt.Sprintf("org.limoxel.worker%d", targetIdx))

				switch (routineID + it) % 4 {
				case 0:
					_ = svc.Activate(ctx, targetID)
				case 1:
					_ = svc.Suspend(ctx, targetID)
				case 2:
					_, _ = svc.Get(targetID)
					_ = svc.List()
					_ = svc.ListByState(plugin.StateActive)
				case 3:
					_ = svc.Host().Events().Publish(ctx, "worker.heartbeat", []byte("ping"))
				}
			}
		}(r)
	}

	wg.Wait()
}
