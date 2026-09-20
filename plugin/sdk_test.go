package plugin_test

import (
	"context"
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
	"github.com/unhield/limoxel/plugin"
	"github.com/unhield/limoxel/plugin/api"
)

type dummyHost struct{}

func (d *dummyHost) HostVersion() version.SemVer {
	return version.SemVer{Major: 1, Minor: 0, Patch: 0}
}
func (d *dummyHost) Capabilities() []string              { return []string{"repo"} }
func (d *dummyHost) HasCapability(name string) bool      { return name == "repo" }
func (d *dummyHost) Repository() plugin.RepositoryFacade { return &dummyRepoFacade{} }
func (d *dummyHost) Events() plugin.EventFacade          { return &dummyEventFacade{} }
func (d *dummyHost) Logger() plugin.Logger               { return &dummyLogger{} }

type dummyRepoFacade struct{}

func (r *dummyRepoFacade) WorkspacePath() string { return "/test/workspace" }
func (r *dummyRepoFacade) Metadata(ctx context.Context) (*plugin.RepositoryMetadata, error) {
	return &plugin.RepositoryMetadata{
		Name:     "dummy-repo",
		RootPath: "/test/workspace",
	}, nil
}
func (r *dummyRepoFacade) ListFiles(ctx context.Context, prefix string) ([]string, error) {
	return []string{"main.go", "plugin.json"}, nil
}

type dummyEventFacade struct{}

func (e *dummyEventFacade) Subscribe(topic string, handler plugin.EventHandler) (plugin.Subscription, error) {
	return &dummySub{}, nil
}
func (e *dummyEventFacade) Publish(ctx context.Context, topic string, payload []byte) error {
	return nil
}

type dummySub struct{}

func (s *dummySub) Unsubscribe() error { return nil }

type dummyLogger struct{}

func (l *dummyLogger) Debug(msg string, kv ...any) {}
func (l *dummyLogger) Info(msg string, kv ...any)  {}
func (l *dummyLogger) Warn(msg string, kv ...any)  {}
func (l *dummyLogger) Error(msg string, kv ...any) {}

func TestBasePlugin_And_Context(t *testing.T) {
	manifest := plugin.Manifest{
		ID:      "com.example.test",
		Name:    "Test Plugin",
		Version: "1.0.0",
	}

	p := plugin.NewBasePlugin("com.example.test", manifest)
	if p.ID() != "com.example.test" {
		t.Fatalf("expected ID com.example.test, got %s", p.ID())
	}
	if p.Manifest().Name != "Test Plugin" {
		t.Fatalf("expected Name Test Plugin, got %s", p.Manifest().Name)
	}

	ctx := context.Background()
	host := &dummyHost{}

	// Initialize
	if err := p.Init(ctx, host); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	pCtx := p.GetContext()
	if pCtx == nil {
		t.Fatalf("expected non-nil context")
	}

	// Start
	if err := p.Start(ctx); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	if !p.IsStarted {
		t.Fatalf("expected IsStarted to be true")
	}

	// Repository API through fallback adapter
	repo := pCtx.Repository()
	if repo.WorkspacePath() != "/test/workspace" {
		t.Errorf("workspace path mismatch: %s", repo.WorkspacePath())
	}
	meta, err := repo.Metadata(ctx)
	if err != nil || meta.Name != "dummy-repo" {
		t.Errorf("metadata mismatch: %v, %v", meta, err)
	}
	files, err := repo.ListFiles(ctx, "")
	if err != nil || len(files) != 2 {
		t.Errorf("list files mismatch: %v, %v", files, err)
	}

	// Events API through fallback adapter
	events := pCtx.Events()
	sub, err := events.Subscribe(ctx, "test.topic", func(c context.Context, evt api.Event) error {
		return nil
	})
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}
	_ = sub.Unsubscribe()

	// Stop
	if err := p.Stop(ctx); err != nil {
		t.Fatalf("stop failed: %v", err)
	}
	if p.IsStarted {
		t.Fatalf("expected IsStarted to be false")
	}
}
