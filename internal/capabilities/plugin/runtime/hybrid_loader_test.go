package runtime

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/plugin/loader"
	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
	"github.com/unhield/limoxel/plugin"
)

type dummyInProcessPlugin struct{}

func (d *dummyInProcessPlugin) ID() string { return "org.example.inproc" }
func (d *dummyInProcessPlugin) Manifest() plugin.Manifest {
	return plugin.Manifest{ID: "org.example.inproc"}
}
func (d *dummyInProcessPlugin) Init(ctx context.Context, host plugin.Host) error { return nil }
func (d *dummyInProcessPlugin) Start(ctx context.Context) error                  { return nil }
func (d *dummyInProcessPlugin) Stop(ctx context.Context) error                   { return nil }

func TestHybridLoader_Routing(t *testing.T) {
	inProc := loader.NewInProcessLoader()
	inProc.RegisterFactory(model.Identity("org.example.inproc"), func() (plugin.Plugin, error) {
		return &dummyInProcessPlugin{}, nil
	})

	resolver := func(m *model.Manifest, dir string) (string, []string, map[string]string, error) {
		return os.Args[0], []string{"-test.run=TestHelperProcess", "--"}, map[string]string{
			"GO_WANT_HELPER_PROCESS": "1",
			"HELPER_MODE":            "server",
		}, nil
	}
	proc := NewProcessLoader(resolver)

	hybrid := NewHybridLoader(inProc, proc)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. In-process manifest
	inProcManifestJSON := []byte(`{
		"schema_version": "1.0.0",
		"id": "org.example.inproc",
		"name": "In Process Plugin",
		"version": "1.0.0"
	}`)
	inProcManifest, _ := model.ParseManifestJSON(inProcManifestJSON)

	inst1, err := hybrid.Load(ctx, inProcManifest, ".", nil)
	if err != nil {
		t.Fatalf("hybrid Load in-process failed: %v", err)
	}
	if _, ok := inst1.(*processInstance); ok {
		t.Fatal("expected in-process instance, got processInstance")
	}
	_ = hybrid.Unload(ctx, inst1)

	// 2. Out-of-process manifest
	outProcManifestJSON := []byte(`{
		"schema_version": "1.0.0",
		"id": "org.example.outproc",
		"name": "Out of Process Plugin",
		"version": "1.0.0",
		"metadata": {
			"custom_attributes": {
				"mode": "isolated"
			}
		}
	}`)
	outProcManifest, _ := model.ParseManifestJSON(outProcManifestJSON)

	inst2, err := hybrid.Load(ctx, outProcManifest, ".", nil)
	if err != nil {
		t.Fatalf("hybrid Load out-of-process failed: %v", err)
	}
	if _, ok := inst2.(*processInstance); !ok {
		t.Fatalf("expected *processInstance, got %T", inst2)
	}
	_ = hybrid.Unload(ctx, inst2)
}
