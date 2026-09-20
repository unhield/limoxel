package marketplace

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

func TestPublicationStatus_IsDiscoverable(t *testing.T) {
	if !StatusPublished.IsDiscoverable() {
		t.Fatal("expected StatusPublished to be discoverable")
	}
	if StatusSubmitted.IsDiscoverable() {
		t.Fatal("expected StatusSubmitted not to be discoverable")
	}
	if StatusValidating.IsDiscoverable() {
		t.Fatal("expected StatusValidating not to be discoverable")
	}
	if StatusRejected.IsDiscoverable() {
		t.Fatal("expected StatusRejected not to be discoverable")
	}
	if StatusSuspended.IsDiscoverable() {
		t.Fatal("expected StatusSuspended not to be discoverable")
	}
}

func TestAllCategories(t *testing.T) {
	cats := AllCategories()
	if len(cats) == 0 {
		t.Fatal("expected non-empty categories")
	}
	seen := make(map[Category]bool)
	for _, c := range cats {
		if seen[c] {
			t.Fatalf("duplicate category: %s", c)
		}
		seen[c] = true
	}
}

func TestPluginDetail_Serialization(t *testing.T) {
	detail := PluginDetail{
		PluginSummary: PluginSummary{
			ID:             "test-plugin",
			Name:           "Test Plugin",
			Description:    "A test plugin for validation",
			PublisherID:    "acme",
			PublisherName:  "Acme Corp",
			PublisherValid: true,
			LatestVersion:  version.SemVer{Major: 1, Minor: 0, Patch: 0},
			Categories:     []Category{CategoryAnalysis, CategoryTools},
			TotalDownloads: 1500,
			AverageRating:  4.5,
			RatingCount:    10,
			Featured:       true,
			Status:         StatusPublished,
			UpdatedAt:      time.Now().UTC(),
		},
		License:    "MIT",
		Repository: "https://github.com/acme/test-plugin",
		Versions: []VersionInfo{
			{
				Version:        version.SemVer{Major: 1, Minor: 0, Patch: 0},
				ArtifactDigest: "abc123sha256",
				ArtifactSize:   2048,
				PublishedAt:    time.Now().UTC(),
				Downloads:      1500,
				Status:         StatusPublished,
			},
		},
		Publisher: PublisherProfile{
			ID:          "acme",
			DisplayName: "Acme Corp",
			Verified:    true,
			PluginCount: 1,
		},
		Rating: RatingSummary{
			Average: 4.5,
			Count:   10,
			Stars:   map[int]int{5: 5, 4: 5},
		},
	}

	data, err := json.Marshal(detail)
	if err != nil {
		t.Fatalf("failed to marshal plugin detail: %v", err)
	}

	var parsed PluginDetail
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal plugin detail: %v", err)
	}

	if parsed.ID != detail.ID {
		t.Fatalf("expected ID %s, got %s", detail.ID, parsed.ID)
	}
	if parsed.Publisher.DisplayName != "Acme Corp" {
		t.Fatalf("expected publisher Acme Corp, got %s", parsed.Publisher.DisplayName)
	}
	if len(parsed.Versions) != 1 {
		t.Fatalf("expected 1 version, got %d", len(parsed.Versions))
	}
}
