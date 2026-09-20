package catalog

import (
	"testing"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/plugin/marketplace/storage"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
	pubmarket "github.com/unhield/limoxel/plugin/marketplace"
)

func TestCatalog_RegistrationAndVersionManagement(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewFileStorage(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	cat, err := NewCatalog(store)
	if err != nil {
		t.Fatal(err)
	}

	detail := pubmarket.PluginDetail{
		PluginSummary: pubmarket.PluginSummary{
			ID:            "plugin-test",
			Name:          "Test Plugin",
			Description:   "Description",
			PublisherID:   "acme",
			LatestVersion: version.SemVer{Major: 1, Minor: 0, Patch: 0},
			Categories:    []pubmarket.Category{pubmarket.CategoryAnalysis},
			Status:        pubmarket.StatusPublished,
		},
	}

	// 1. Register plugin
	if err := cat.RegisterPlugin(detail); err != nil {
		t.Fatalf("failed to register plugin: %v", err)
	}

	got, err := cat.GetPlugin("plugin-test")
	if err != nil || got.ID != "plugin-test" {
		t.Fatalf("unexpected plugin: %+v, err: %v", got, err)
	}

	// 2. Add version 1.0.0
	ver1 := pubmarket.VersionInfo{
		Version:        version.SemVer{Major: 1, Minor: 0, Patch: 0},
		ArtifactDigest: "hash100",
		Status:         pubmarket.StatusPublished,
	}
	if err := cat.AddVersion("plugin-test", ver1); err != nil {
		t.Fatalf("failed to add version 1.0.0: %v", err)
	}

	// 3. Immutability check: adding duplicate 1.0.0 must fail
	if err := cat.AddVersion("plugin-test", ver1); err != ErrVersionExists {
		t.Fatalf("expected ErrVersionExists for duplicate version, got %v", err)
	}

	// 4. Add version 1.1.0
	ver2 := pubmarket.VersionInfo{
		Version:        version.SemVer{Major: 1, Minor: 1, Patch: 0},
		ArtifactDigest: "hash110",
		Status:         pubmarket.StatusPublished,
	}
	if err := cat.AddVersion("plugin-test", ver2); err != nil {
		t.Fatalf("failed to add version 1.1.0: %v", err)
	}

	vers, err := cat.GetVersions("plugin-test")
	if err != nil || len(vers) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(vers))
	}
	// Verify descending order: 1.1.0 first
	if vers[0].Version.String() != "1.1.0" {
		t.Fatalf("expected 1.1.0 first in descending sort, got %s", vers[0].Version.String())
	}

	// 5. Test download tracking
	if err := cat.IncrementDownloads("plugin-test", ver2.Version); err != nil {
		t.Fatalf("failed to increment downloads: %v", err)
	}
	gotUpdated, _ := cat.GetPlugin("plugin-test")
	if gotUpdated.TotalDownloads != 1 {
		t.Fatalf("expected total downloads 1, got %d", gotUpdated.TotalDownloads)
	}

	// 6. Test rating update
	rating := pubmarket.RatingSummary{Average: 4.8, Count: 5}
	if err := cat.UpdateRating("plugin-test", rating); err != nil {
		t.Fatalf("failed to update rating: %v", err)
	}
	gotRated, _ := cat.GetPlugin("plugin-test")
	if gotRated.AverageRating != 4.8 || gotRated.RatingCount != 5 {
		t.Fatalf("unexpected rating: %+v", gotRated.Rating)
	}

	// 7. Verify category counting
	count := cat.GetCategoryCount(pubmarket.CategoryAnalysis)
	if count != 1 {
		t.Fatalf("expected 1 plugin in analysis category, got %d", count)
	}
}

func TestCatalog_PersistenceReload(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := storage.NewFileStorage(tmpDir)

	cat1, _ := NewCatalog(store)
	_ = cat1.RegisterPlugin(pubmarket.PluginDetail{
		PluginSummary: pubmarket.PluginSummary{
			ID:        "p-persist",
			Name:      "Persist",
			Status:    pubmarket.StatusPublished,
			UpdatedAt: time.Now().UTC(),
		},
	})

	// Reconstruct catalog from same store
	cat2, err := NewCatalog(store)
	if err != nil {
		t.Fatalf("failed to reload catalog: %v", err)
	}

	p, err := cat2.GetPlugin("p-persist")
	if err != nil || p.Name != "Persist" {
		t.Fatalf("expected persisted plugin to be reloaded, got %+v (err: %v)", p, err)
	}
}
