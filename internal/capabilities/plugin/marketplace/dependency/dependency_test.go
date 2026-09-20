package dependency_test

import (
	"fmt"
	"testing"

	marketdep "github.com/unhield/limoxel/internal/capabilities/plugin/marketplace/dependency"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
	pubmarket "github.com/unhield/limoxel/plugin/marketplace"
)

type mockCatalog struct {
	plugins map[string]*pubmarket.PluginDetail
}

func (m *mockCatalog) GetPlugin(id string) (*pubmarket.PluginDetail, error) {
	p, ok := m.plugins[id]
	if !ok {
		return nil, fmt.Errorf("plugin %s not found", id)
	}
	return p, nil
}

func semver(s string) version.SemVer {
	v, err := version.ParseSemVer(s)
	if err != nil {
		panic(err)
	}
	return v
}

func TestDependencyResolver_SinglePlugin(t *testing.T) {
	cat := &mockCatalog{
		plugins: map[string]*pubmarket.PluginDetail{
			"test.plugin": {
				PluginSummary: pubmarket.PluginSummary{
					ID:            "test.plugin",
					LatestVersion: semver("1.0.0"),
				},
				Versions: []pubmarket.VersionInfo{
					{
						Version:        semver("1.0.0"),
						ArtifactDigest: "abc123digest",
						Status:         pubmarket.StatusPublished,
					},
				},
			},
		},
	}

	res := marketdep.NewResolver(cat)
	plan, err := res.Resolve("test.plugin", semver("1.0.0"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(plan.InstallOrder) != 1 {
		t.Fatalf("expected 1 install item, got %d", len(plan.InstallOrder))
	}
	if plan.InstallOrder[0].ID != "test.plugin" {
		t.Fatalf("expected test.plugin, got %s", plan.InstallOrder[0].ID)
	}
}

func TestDependencyResolver_Transitive(t *testing.T) {
	cat := &mockCatalog{
		plugins: map[string]*pubmarket.PluginDetail{
			"app": {
				PluginSummary: pubmarket.PluginSummary{
					ID:            "app",
					LatestVersion: semver("1.0.0"),
				},
				Versions: []pubmarket.VersionInfo{
					{Version: semver("1.0.0"), Status: pubmarket.StatusPublished, ArtifactDigest: "app100"},
				},
				Dependencies: []pubmarket.DependencySummary{
					{ID: "lib-b", Constraint: ">= 1.0.0"},
				},
			},
			"lib-b": {
				PluginSummary: pubmarket.PluginSummary{
					ID:            "lib-b",
					LatestVersion: semver("1.2.0"),
				},
				Versions: []pubmarket.VersionInfo{
					{Version: semver("1.0.0"), Status: pubmarket.StatusPublished, ArtifactDigest: "b100"},
					{Version: semver("1.2.0"), Status: pubmarket.StatusPublished, ArtifactDigest: "b120"},
				},
				Dependencies: []pubmarket.DependencySummary{
					{ID: "lib-c", Constraint: "^2.0.0"},
				},
			},
			"lib-c": {
				PluginSummary: pubmarket.PluginSummary{
					ID:            "lib-c",
					LatestVersion: semver("2.1.0"),
				},
				Versions: []pubmarket.VersionInfo{
					{Version: semver("2.0.0"), Status: pubmarket.StatusPublished, ArtifactDigest: "c200"},
					{Version: semver("2.1.0"), Status: pubmarket.StatusPublished, ArtifactDigest: "c210"},
				},
			},
		},
	}

	res := marketdep.NewResolver(cat)
	plan, err := res.Resolve("app", semver("1.0.0"))
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	if len(plan.InstallOrder) != 3 {
		t.Fatalf("expected 3 items in install order, got %d", len(plan.InstallOrder))
	}

	// Verify topological order: lib-c must be installed before lib-b, and lib-b before app
	var order []string
	for _, item := range plan.InstallOrder {
		order = append(order, item.ID)
	}

	idxC := indexOf(order, "lib-c")
	idxB := indexOf(order, "lib-b")
	idxApp := indexOf(order, "app")

	if idxC == -1 || idxB == -1 || idxApp == -1 {
		t.Fatalf("missing expected plugin in order: %v", order)
	}
	if !(idxC < idxB && idxB < idxApp) {
		t.Fatalf("topological order violated: %v", order)
	}

	// Verify highest compatible version selected for lib-b (1.2.0) and lib-c (2.1.0)
	if plan.Dependencies["lib-b"].Version.String() != "1.2.0" {
		t.Errorf("expected lib-b version 1.2.0, got %s", plan.Dependencies["lib-b"].Version.String())
	}
	if plan.Dependencies["lib-c"].Version.String() != "2.1.0" {
		t.Errorf("expected lib-c version 2.1.0, got %s", plan.Dependencies["lib-c"].Version.String())
	}
}

func TestDependencyResolver_MissingDependency(t *testing.T) {
	cat := &mockCatalog{
		plugins: map[string]*pubmarket.PluginDetail{
			"app": {
				PluginSummary: pubmarket.PluginSummary{ID: "app", LatestVersion: semver("1.0.0")},
				Versions:      []pubmarket.VersionInfo{{Version: semver("1.0.0"), Status: pubmarket.StatusPublished}},
				Dependencies: []pubmarket.DependencySummary{
					{ID: "nonexistent", Constraint: ">= 1.0.0", Optional: false},
				},
			},
		},
	}

	res := marketdep.NewResolver(cat)
	_, err := res.Resolve("app", semver("1.0.0"))
	if err == nil {
		t.Fatal("expected error for missing dependency, got nil")
	}
	if !isError(err, pubmarket.ErrDependencyMissing) {
		t.Errorf("expected ErrDependencyMissing, got %v", err)
	}
}

func TestDependencyResolver_IncompatibleConstraint(t *testing.T) {
	cat := &mockCatalog{
		plugins: map[string]*pubmarket.PluginDetail{
			"app": {
				PluginSummary: pubmarket.PluginSummary{ID: "app", LatestVersion: semver("1.0.0")},
				Versions:      []pubmarket.VersionInfo{{Version: semver("1.0.0"), Status: pubmarket.StatusPublished}},
				Dependencies: []pubmarket.DependencySummary{
					{ID: "dep", Constraint: ">= 3.0.0"},
				},
			},
			"dep": {
				PluginSummary: pubmarket.PluginSummary{ID: "dep", LatestVersion: semver("2.0.0")},
				Versions: []pubmarket.VersionInfo{
					{Version: semver("1.0.0"), Status: pubmarket.StatusPublished},
					{Version: semver("2.0.0"), Status: pubmarket.StatusPublished},
				},
			},
		},
	}

	res := marketdep.NewResolver(cat)
	_, err := res.Resolve("app", semver("1.0.0"))
	if err == nil {
		t.Fatal("expected error for incompatible constraint, got nil")
	}
	if !isError(err, pubmarket.ErrDependencyConflict) {
		t.Errorf("expected ErrDependencyConflict, got %v", err)
	}
}

func TestDependencyResolver_CircularDependency(t *testing.T) {
	cat := &mockCatalog{
		plugins: map[string]*pubmarket.PluginDetail{
			"plugin-a": {
				PluginSummary: pubmarket.PluginSummary{ID: "plugin-a", LatestVersion: semver("1.0.0")},
				Versions:      []pubmarket.VersionInfo{{Version: semver("1.0.0"), Status: pubmarket.StatusPublished}},
				Dependencies: []pubmarket.DependencySummary{
					{ID: "plugin-b", Constraint: ">= 1.0.0"},
				},
			},
			"plugin-b": {
				PluginSummary: pubmarket.PluginSummary{ID: "plugin-b", LatestVersion: semver("1.0.0")},
				Versions:      []pubmarket.VersionInfo{{Version: semver("1.0.0"), Status: pubmarket.StatusPublished}},
				Dependencies: []pubmarket.DependencySummary{
					{ID: "plugin-a", Constraint: ">= 1.0.0"},
				},
			},
		},
	}

	res := marketdep.NewResolver(cat)
	_, err := res.Resolve("plugin-a", semver("1.0.0"))
	if err == nil {
		t.Fatal("expected cycle detection error, got nil")
	}
	if !isError(err, pubmarket.ErrCircularDependency) {
		t.Errorf("expected ErrCircularDependency, got %v", err)
	}
}

func indexOf(slice []string, val string) int {
	for i, v := range slice {
		if v == val {
			return i
		}
	}
	return -1
}

func isError(err, target error) bool {
	if err == nil || target == nil {
		return false
	}
	for err != nil {
		if err == target {
			return true
		}
		if u, ok := err.(interface{ Unwrap() error }); ok {
			err = u.Unwrap()
		} else {
			break
		}
	}
	return false
}
