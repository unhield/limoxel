package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"testing"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
	pubmarket "github.com/unhield/limoxel/plugin/marketplace"
)

func TestFileStorage_PluginAndPublisherLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewFileStorage(tmpDir)
	if err != nil {
		t.Fatalf("failed to initialize storage: %v", err)
	}

	// 1. Publisher Save and Get
	pub := pubmarket.PublisherProfile{
		ID:          "acme",
		DisplayName: "Acme Corp",
		Website:     "https://acme.org",
		Verified:    true,
		CreatedAt:   time.Now().UTC(),
		PluginCount: 1,
	}

	if err := store.SavePublisher(pub); err != nil {
		t.Fatalf("failed to save publisher: %v", err)
	}

	gotPub, err := store.GetPublisher("acme")
	if err != nil {
		t.Fatalf("failed to get publisher: %v", err)
	}
	if gotPub.DisplayName != "Acme Corp" || !gotPub.Verified {
		t.Fatalf("unexpected publisher data: %+v", gotPub)
	}

	pubs, err := store.ListPublishers()
	if err != nil || len(pubs) != 1 {
		t.Fatalf("expected 1 publisher, got %d (err: %v)", len(pubs), err)
	}

	// 2. Plugin Save, Get, List
	detail := pubmarket.PluginDetail{
		PluginSummary: pubmarket.PluginSummary{
			ID:            "plugin-sample",
			Name:          "Sample Plugin",
			Description:   "Description text",
			PublisherID:   "acme",
			PublisherName: "Acme Corp",
			LatestVersion: version.SemVer{Major: 1, Minor: 2, Patch: 0},
			Status:        pubmarket.StatusPublished,
			UpdatedAt:     time.Now().UTC(),
		},
		License: "Apache-2.0",
		Versions: []pubmarket.VersionInfo{
			{
				Version:        version.SemVer{Major: 1, Minor: 2, Patch: 0},
				ArtifactDigest: "0123456789abcdef",
				Status:         pubmarket.StatusPublished,
			},
		},
	}

	if err := store.SavePluginDetail(detail); err != nil {
		t.Fatalf("failed to save plugin detail: %v", err)
	}

	gotDetail, err := store.GetPluginDetail("plugin-sample")
	if err != nil {
		t.Fatalf("failed to get plugin detail: %v", err)
	}
	if gotDetail.Name != "Sample Plugin" || len(gotDetail.Versions) != 1 {
		t.Fatalf("unexpected plugin detail: %+v", gotDetail)
	}

	list, err := store.ListPluginDetails()
	if err != nil || len(list) != 1 {
		t.Fatalf("expected 1 plugin detail, got %d (err: %v)", len(list), err)
	}

	// 3. Delete Plugin
	if err := store.DeletePlugin("plugin-sample"); err != nil {
		t.Fatalf("delete plugin failed: %v", err)
	}
	_, err = store.GetPluginDetail("plugin-sample")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after deletion, got %v", err)
	}
}

func TestFileStorage_ArtifactStorageAndDigestEnforcement(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewFileStorage(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	payload := []byte("this is a test plugin archive payload")
	hasher := sha256.New()
	hasher.Write(payload)
	expectedDigest := hex.EncodeToString(hasher.Sum(nil))

	// 1. Successful save with matching digest
	digest, size, err := store.SaveArtifact(expectedDigest, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("expected save artifact to succeed: %v", err)
	}
	if digest != expectedDigest || size != int64(len(payload)) {
		t.Fatalf("unexpected result: digest=%s, size=%d", digest, size)
	}
	if !store.ArtifactExists(digest) {
		t.Fatal("expected artifact to exist")
	}

	// 2. Read artifact back
	rc, readSize, err := store.GetArtifact(digest)
	if err != nil {
		t.Fatalf("failed to get artifact: %v", err)
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	if readSize != int64(len(payload)) || !bytes.Equal(data, payload) {
		t.Fatal("retrieved artifact does not match stored bytes")
	}

	// 3. Digest mismatch failure
	wrongDigest := "0000000000000000000000000000000000000000000000000000000000000000"
	_, _, err = store.SaveArtifact(wrongDigest, bytes.NewReader(payload))
	if err == nil {
		t.Fatal("expected digest mismatch to return error")
	}
}

func TestFileStorage_ReviewsAndRatings(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewFileStorage(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	pluginID := "plugin-stars"

	// 1. Ratings
	userRatings := map[string]int{
		"user-1": 5,
		"user-2": 4,
	}
	if err := store.SaveRatings(pluginID, userRatings); err != nil {
		t.Fatal(err)
	}
	gotRatings, err := store.GetRatings(pluginID)
	if err != nil {
		t.Fatal(err)
	}
	if len(gotRatings) != 2 || gotRatings["user-1"] != 5 {
		t.Fatalf("unexpected ratings map: %+v", gotRatings)
	}

	// 2. Reviews
	reviews := []pubmarket.Review{
		{
			ID:        "rev-1",
			PluginID:  pluginID,
			Version:   "1.0.0",
			AuthorID:  "user-1",
			Rating:    5,
			Title:     "Great plugin",
			Comment:   "Worked smoothly",
			CreatedAt: time.Now().UTC(),
		},
	}
	if err := store.SaveReviews(pluginID, reviews); err != nil {
		t.Fatal(err)
	}
	gotReviews, err := store.GetReviews(pluginID)
	if err != nil {
		t.Fatal(err)
	}
	if len(gotReviews) != 1 || gotReviews[0].Title != "Great plugin" {
		t.Fatalf("unexpected reviews: %+v", gotReviews)
	}
}
