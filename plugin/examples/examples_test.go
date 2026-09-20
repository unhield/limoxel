package examples_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/unhield/limoxel/plugin"
	"github.com/unhield/limoxel/plugin/api"
	plugintesting "github.com/unhield/limoxel/plugin/testing"
)

// 1. Basic Plugin Example
type BasicSamplePlugin struct {
	*plugin.BasePlugin
	initialized bool
	started     bool
	stopped     bool
}

func NewBasicSamplePlugin() *BasicSamplePlugin {
	manifest := plugin.Manifest{
		ID:          "com.example.basic",
		Name:        "Basic Sample Plugin",
		Version:     "1.0.0",
		Description: "Demonstrates standard lifecycle integration.",
	}
	return &BasicSamplePlugin{
		BasePlugin: plugin.NewBasePlugin("com.example.basic", manifest),
	}
}

func (p *BasicSamplePlugin) Init(ctx context.Context, host plugin.Host) error {
	if err := p.BasePlugin.Init(ctx, host); err != nil {
		return err
	}
	p.initialized = true
	p.GetContext().Logger().Info("initialized basic sample plugin")
	return nil
}

func (p *BasicSamplePlugin) Start(ctx context.Context) error {
	if err := p.BasePlugin.Start(ctx); err != nil {
		return err
	}
	p.started = true
	return nil
}

func (p *BasicSamplePlugin) Stop(ctx context.Context) error {
	p.stopped = true
	return p.BasePlugin.Stop(ctx)
}

// 2. Repository Plugin Example
type RepositorySamplePlugin struct {
	*plugin.BasePlugin
	filesFound []string
	repoName   string
}

func NewRepositorySamplePlugin() *RepositorySamplePlugin {
	manifest := plugin.Manifest{
		ID:      "com.example.repo",
		Name:    "Repository Sample Plugin",
		Version: "1.0.0",
		Capabilities: []plugin.Capability{
			{Name: "repository.metadata", Version: "1.0.0"},
			{Name: "repository.files", Version: "1.0.0"},
		},
	}
	return &RepositorySamplePlugin{
		BasePlugin: plugin.NewBasePlugin("com.example.repo", manifest),
	}
}

func (p *RepositorySamplePlugin) Start(ctx context.Context) error {
	if err := p.BasePlugin.Start(ctx); err != nil {
		return err
	}
	repo := p.GetContext().Repository()
	meta, err := repo.Metadata(ctx)
	if err == nil && meta != nil {
		p.repoName = meta.Name
	}
	files, err := repo.ListFiles(ctx, "")
	if err == nil {
		p.filesFound = files
	}
	return nil
}

// 3. Intelligence Plugin Example
type IntelligenceSamplePlugin struct {
	*plugin.BasePlugin
	symbolResolved bool
	searchMatches  int
	graphNodes     int
	healthScore    float64
}

func NewIntelligenceSamplePlugin() *IntelligenceSamplePlugin {
	manifest := plugin.Manifest{
		ID:      "com.example.intelligence",
		Name:    "Intelligence Sample Plugin",
		Version: "1.0.0",
		Capabilities: []plugin.Capability{
			{Name: "symbols.api", Version: "1.0.0"},
			{Name: "search.api", Version: "1.0.0"},
			{Name: "graph.api", Version: "1.0.0"},
			{Name: "analysis.api", Version: "1.0.0"},
		},
	}
	return &IntelligenceSamplePlugin{
		BasePlugin: plugin.NewBasePlugin("com.example.intelligence", manifest),
	}
}

func (p *IntelligenceSamplePlugin) Start(ctx context.Context) error {
	if err := p.BasePlugin.Start(ctx); err != nil {
		return err
	}
	pCtx := p.GetContext()

	// Symbol lookup
	sym, err := pCtx.Symbols().LookupSymbol(ctx, "ExampleFunction")
	if err == nil && sym != nil {
		p.symbolResolved = true
	}

	// Search
	res, err := pCtx.Search().SearchSymbols(ctx, "Example", api.PaginationOptions{Limit: 10})
	if err == nil && res != nil {
		p.searchMatches = res.TotalMatches
	}

	// Graph
	nodes, _, err := pCtx.Graph().GraphInfo(ctx)
	if err == nil {
		p.graphNodes = nodes
	}

	// Analysis
	health, err := pCtx.Analysis().RepositoryHealth(ctx)
	if err == nil && health != nil {
		p.healthScore = health.OverallScore
	}

	return nil
}

// 4. Event Pub/Sub Plugin Example
type EventSamplePlugin struct {
	*plugin.BasePlugin
	eventsReceived []string
	sub            api.Subscription
}

func NewEventSamplePlugin() *EventSamplePlugin {
	manifest := plugin.Manifest{
		ID:      "com.example.events",
		Name:    "Event Sample Plugin",
		Version: "1.0.0",
		Capabilities: []plugin.Capability{
			{Name: "events.pubsub", Version: "1.0.0"},
		},
	}
	return &EventSamplePlugin{
		BasePlugin: plugin.NewBasePlugin("com.example.events", manifest),
	}
}

func (p *EventSamplePlugin) Start(ctx context.Context) error {
	if err := p.BasePlugin.Start(ctx); err != nil {
		return err
	}
	sub, err := p.GetContext().Events().Subscribe(ctx, "build.*", func(c context.Context, evt api.Event) error {
		p.eventsReceived = append(p.eventsReceived, fmt.Sprintf("%s:%v", evt.Type, evt.Payload))
		return nil
	})
	if err != nil {
		return err
	}
	p.sub = sub
	return nil
}

func (p *EventSamplePlugin) Stop(ctx context.Context) error {
	if p.sub != nil {
		_ = p.sub.Unsubscribe()
	}
	return p.BasePlugin.Stop(ctx)
}

// Tests exercising all 4 examples against TestHarness
func TestExamples_Execution(t *testing.T) {
	wsDir := t.TempDir()
	harness := plugintesting.NewTestHarness(wsDir)
	ctx := context.Background()

	// 1. Basic Plugin
	basic := NewBasicSamplePlugin()
	if err := harness.ExecuteLifecycle(ctx, basic); err != nil {
		t.Fatalf("basic plugin lifecycle failed: %v", err)
	}
	if !basic.initialized || !basic.started || !basic.stopped {
		t.Errorf("basic plugin state flags mismatch: init=%v, start=%v, stop=%v",
			basic.initialized, basic.started, basic.stopped)
	}

	// 2. Repository Plugin
	repoPlug := NewRepositorySamplePlugin()
	if err := harness.ExecuteLifecycle(ctx, repoPlug); err != nil {
		t.Fatalf("repo plugin lifecycle failed: %v", err)
	}
	if len(repoPlug.filesFound) == 0 {
		t.Errorf("expected repo plugin to discover files")
	}

	// 3. Intelligence Plugin
	intelPlug := NewIntelligenceSamplePlugin()
	if err := harness.ExecuteLifecycle(ctx, intelPlug); err != nil {
		t.Fatalf("intelligence plugin lifecycle failed: %v", err)
	}
	if !intelPlug.symbolResolved {
		t.Errorf("expected symbol resolution in intelligence plugin")
	}
	if intelPlug.graphNodes <= 0 || intelPlug.healthScore <= 0 {
		t.Errorf("expected non-zero graph and health metrics: nodes=%d, score=%f",
			intelPlug.graphNodes, intelPlug.healthScore)
	}

	// 4. Event Plugin
	evtPlug := NewEventSamplePlugin()
	if err := evtPlug.Init(ctx, harness.HostInstance); err != nil {
		t.Fatalf("event plugin init failed: %v", err)
	}
	if err := evtPlug.Start(ctx); err != nil {
		t.Fatalf("event plugin start failed: %v", err)
	}

	// Dispatch matching events
	_ = harness.HostInstance.EventAPI().Publish(ctx, api.Event{
		Type:    "build.started",
		Payload: "build-101",
	})
	_ = harness.HostInstance.EventAPI().Publish(ctx, api.Event{
		Type:    "build.finished",
		Payload: "success",
	})

	if len(evtPlug.eventsReceived) != 2 {
		t.Errorf("expected 2 events received, got %d: %v", len(evtPlug.eventsReceived), evtPlug.eventsReceived)
	}

	if err := evtPlug.Stop(ctx); err != nil {
		t.Fatalf("event plugin stop failed: %v", err)
	}
}
