package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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
				ArtifactDigest: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
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
	if !errors.Is(err, ErrDigestMismatch) {
		t.Fatalf("expected ErrDigestMismatch, got: %v", err)
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

func TestFileStorage_AdversarialIdentifierMatrix(t *testing.T) {
	maliciousIDs := []struct {
		category string
		id       string
	}{
		// Traversal
		{"traversal", "../escape"},
		{"traversal", "../../escape"},
		{"traversal", "foo/../../escape"},
		{"traversal", ".../../escape"},
		{"traversal", "..//..//escape"},
		{"traversal", "a/../b"},

		// Windows traversal
		{"windows_traversal", `..\escape`},
		{"windows_traversal", `..\..\escape`},
		{"windows_traversal", `foo\..\..\escape`},
		{"windows_traversal", `C:\escape`},
		{"windows_traversal", `C:escape`},
		{"windows_traversal", `C:/escape`},

		// Absolute / Rooted
		{"absolute", "/etc/passwd"},
		{"absolute", "/tmp/escape"},
		{"unc", `\\server\share\escape`},
		{"unc", `//server/share/escape`},

		// Special names & invalid characters
		{"special", "."},
		{"special", ".."},
		{"special", ""},
		{"special", "bad\x00id"},
		{"special", "bad\nid"},
		{"special", "bad\x1fid"},
		{"special", "bad\x7fid"},
		{"special", "trailing-dot."},
		{"special", "contains..double-dot"},
		{"special", "-starts-with-hyphen"},
		{"special", ".starts-with-dot"},
	}

	tmpDir := t.TempDir()
	store, err := NewFileStorage(tmpDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	for _, tc := range maliciousIDs {
		t.Run(tc.category+"_"+tc.id, func(t *testing.T) {
			// SavePluginDetail
			detail := pubmarket.PluginDetail{
				PluginSummary: pubmarket.PluginSummary{
					ID: tc.id,
				},
			}
			if err := store.SavePluginDetail(detail); err == nil || !errors.Is(err, ErrInvalidInput) {
				t.Errorf("SavePluginDetail(%q): expected ErrInvalidInput, got %v", tc.id, err)
			}

			// GetPluginDetail
			if _, err := store.GetPluginDetail(tc.id); err == nil || !errors.Is(err, ErrInvalidInput) {
				t.Errorf("GetPluginDetail(%q): expected ErrInvalidInput, got %v", tc.id, err)
			}

			// DeletePlugin
			if err := store.DeletePlugin(tc.id); err == nil || !errors.Is(err, ErrInvalidInput) {
				t.Errorf("DeletePlugin(%q): expected ErrInvalidInput, got %v", tc.id, err)
			}

			// SavePublisher
			pub := pubmarket.PublisherProfile{ID: tc.id}
			if err := store.SavePublisher(pub); err == nil || !errors.Is(err, ErrInvalidInput) {
				t.Errorf("SavePublisher(%q): expected ErrInvalidInput, got %v", tc.id, err)
			}

			// GetPublisher
			if _, err := store.GetPublisher(tc.id); err == nil || !errors.Is(err, ErrInvalidInput) {
				t.Errorf("GetPublisher(%q): expected ErrInvalidInput, got %v", tc.id, err)
			}

			// SaveReviews
			if err := store.SaveReviews(tc.id, nil); err == nil || !errors.Is(err, ErrInvalidInput) {
				t.Errorf("SaveReviews(%q): expected ErrInvalidInput, got %v", tc.id, err)
			}

			// GetReviews
			if _, err := store.GetReviews(tc.id); err == nil || !errors.Is(err, ErrInvalidInput) {
				t.Errorf("GetReviews(%q): expected ErrInvalidInput, got %v", tc.id, err)
			}

			// SaveRatings
			if err := store.SaveRatings(tc.id, nil); err == nil || !errors.Is(err, ErrInvalidInput) {
				t.Errorf("SaveRatings(%q): expected ErrInvalidInput, got %v", tc.id, err)
			}

			// GetRatings
			if _, err := store.GetRatings(tc.id); err == nil || !errors.Is(err, ErrInvalidInput) {
				t.Errorf("GetRatings(%q): expected ErrInvalidInput, got %v", tc.id, err)
			}

			// WithStorageLock
			if err := WithStorageLock(store.BaseDir(), tc.id, func() error { return nil }); err == nil || !errors.Is(err, ErrInvalidInput) {
				t.Errorf("WithStorageLock(%q): expected ErrInvalidInput, got %v", tc.id, err)
			}
		})
	}
}

func TestFileStorage_ArtifactDigestAbuse(t *testing.T) {
	maliciousDigests := []struct {
		name   string
		digest string
	}{
		{"short", "0123456789abcdef"},
		{"short_1", "a"},
		{"long", strings.Repeat("a", 65)},
		{"non_hex", strings.Repeat("g", 64)},
		{"non_hex_symbol", strings.Repeat("0", 63) + "!"},
		{"contains_slash", "../" + strings.Repeat("a", 61)},
		{"contains_backslash", `..\` + strings.Repeat("a", 61)},
		{"contains_dotdot", ".." + strings.Repeat("a", 62)},
		{"absolute_path", "/etc/passwd" + strings.Repeat("0", 53)},
		{"windows_drive", `C:\` + strings.Repeat("0", 61)},
		{"empty", ""},
	}

	tmpDir := t.TempDir()
	store, err := NewFileStorage(tmpDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	payload := []byte("test artifact payload")

	for _, tc := range maliciousDigests {
		t.Run(tc.name, func(t *testing.T) {
			// SaveArtifact with malicious expectedDigest
			_, _, err := store.SaveArtifact(tc.digest, bytes.NewReader(payload))
			if tc.digest != "" && (err == nil || !errors.Is(err, ErrInvalidInput)) {
				t.Errorf("SaveArtifact with expectedDigest %q: expected ErrInvalidInput, got %v", tc.digest, err)
			}

			// GetArtifact
			_, _, err = store.GetArtifact(tc.digest)
			if err == nil || !errors.Is(err, ErrInvalidInput) {
				t.Errorf("GetArtifact(%q): expected ErrInvalidInput, got %v", tc.digest, err)
			}

			// ArtifactExists
			exists := store.ArtifactExists(tc.digest)
			if exists {
				t.Errorf("ArtifactExists(%q): expected false, got true", tc.digest)
			}
		})
	}
}

func TestFileStorage_BaseDirectoryEscape(t *testing.T) {
	parentDir := t.TempDir()
	sentinelPath := filepath.Join(parentDir, "outside_sentinel.txt")
	sentinelData := []byte("CRITICAL_OUTSIDE_SENTINEL_DATA")
	if err := os.WriteFile(sentinelPath, sentinelData, 0600); err != nil {
		t.Fatalf("failed to write outside sentinel: %v", err)
	}

	storeDir := filepath.Join(parentDir, "marketplace_store")
	store, err := NewFileStorage(storeDir)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	maliciousPayloads := []string{
		"../../outside_sentinel.txt",
		"../../../outside_sentinel.txt",
		`..\..\outside_sentinel.txt`,
		`..\..\outside_file.tmp`,
		"../escape_file",
		`C:\escape_file`,
		"/tmp/escape_file",
	}

	for _, malID := range maliciousPayloads {
		// Attempt write operations with malicious identifier
		_ = store.SavePluginDetail(pubmarket.PluginDetail{
			PluginSummary: pubmarket.PluginSummary{ID: malID},
		})
		_ = store.SavePublisher(pubmarket.PublisherProfile{ID: malID})
		_ = store.SaveReviews(malID, []pubmarket.Review{{ID: "rev", Comment: "injected"}})
		_ = store.SaveRatings(malID, map[string]int{"u": 5})
		_ = store.DeletePlugin(malID)
		_ = WithStorageLock(store.BaseDir(), malID, func() error { return nil })
	}

	// 1. Verify outside sentinel file was not modified or removed
	currentSentinel, err := os.ReadFile(sentinelPath)
	if err != nil {
		t.Fatalf("sentinel file was deleted or cannot be read: %v", err)
	}
	if !bytes.Equal(currentSentinel, sentinelData) {
		t.Fatalf("sentinel file was tampered with! expected %s, got %s", sentinelData, currentSentinel)
	}

	// 2. Verify no files or directories were created outside storeDir in parentDir
	entries, err := os.ReadDir(parentDir)
	if err != nil {
		t.Fatalf("failed to read parent dir: %v", err)
	}
	for _, e := range entries {
		if e.Name() != "outside_sentinel.txt" && e.Name() != "marketplace_store" {
			t.Errorf("illegal file/dir created in parent directory: %s", e.Name())
		}
	}

	// 3. Inspect storeDir: verify every entry is within the allowed subdirectories
	allowedTopLevel := map[string]bool{
		"metadata":  true,
		"artifacts": true,
		"community": true,
	}

	err = filepath.WalkDir(storeDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == storeDir {
			return nil
		}
		rel, relErr := filepath.Rel(storeDir, path)
		if relErr != nil {
			return relErr
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if !allowedTopLevel[parts[0]] {
			return fmt.Errorf("unexpected file outside designated directories: %s", rel)
		}
		return nil
	})
	if err != nil {
		t.Errorf("filesystem walk found illegal file: %v", err)
	}
}

func TestFileStorage_LegitimateIdentifiers(t *testing.T) {
	validIDs := []string{
		"acme",
		"acme-corp",
		"plugin-sample",
		"plugin-stars",
		"user-1",
		"my.cool-plugin_v1",
		"org.example.formatter",
	}

	tmpDir := t.TempDir()
	store, err := NewFileStorage(tmpDir)
	if err != nil {
		t.Fatalf("failed to initialize storage: %v", err)
	}

	for _, id := range validIDs {
		t.Run("ID_"+id, func(t *testing.T) {
			// Publisher
			pub := pubmarket.PublisherProfile{
				ID:          id,
				DisplayName: "Valid Publisher " + id,
			}
			if err := store.SavePublisher(pub); err != nil {
				t.Fatalf("SavePublisher(%q) failed: %v", id, err)
			}
			gotPub, err := store.GetPublisher(id)
			if err != nil {
				t.Fatalf("GetPublisher(%q) failed: %v", id, err)
			}
			if gotPub.DisplayName != pub.DisplayName {
				t.Fatalf("unexpected publisher: %+v", gotPub)
			}

			// Plugin
			detail := pubmarket.PluginDetail{
				PluginSummary: pubmarket.PluginSummary{
					ID:          id,
					Name:        "Plugin " + id,
					PublisherID: id,
				},
			}
			if err := store.SavePluginDetail(detail); err != nil {
				t.Fatalf("SavePluginDetail(%q) failed: %v", id, err)
			}
			gotDetail, err := store.GetPluginDetail(id)
			if err != nil {
				t.Fatalf("GetPluginDetail(%q) failed: %v", id, err)
			}
			if gotDetail.Name != detail.Name {
				t.Fatalf("unexpected plugin detail: %+v", gotDetail)
			}

			// Ratings
			ratings := map[string]int{"u1": 5}
			if err := store.SaveRatings(id, ratings); err != nil {
				t.Fatalf("SaveRatings(%q) failed: %v", id, err)
			}
			gotRatings, err := store.GetRatings(id)
			if err != nil || gotRatings["u1"] != 5 {
				t.Fatalf("GetRatings(%q) failed: %v, ratings=%+v", id, err, gotRatings)
			}

			// Reviews
			reviews := []pubmarket.Review{{ID: "r1", PluginID: id, Comment: "good"}}
			if err := store.SaveReviews(id, reviews); err != nil {
				t.Fatalf("SaveReviews(%q) failed: %v", id, err)
			}
			gotReviews, err := store.GetReviews(id)
			if err != nil || len(gotReviews) != 1 {
				t.Fatalf("GetReviews(%q) failed: %v, reviews=%+v", id, err, gotReviews)
			}

			// Delete
			if err := store.DeletePlugin(id); err != nil {
				t.Fatalf("DeletePlugin(%q) failed: %v", id, err)
			}
			_, err = store.GetPluginDetail(id)
			if !errors.Is(err, ErrNotFound) {
				t.Fatalf("expected ErrNotFound after DeletePlugin(%q), got %v", id, err)
			}
		})
	}
}

func TestStorage_StaleLockRecovery(t *testing.T) {
	tmpDir := t.TempDir()
	relPath := "test_lock"

	lock, err := NewStorageLock(tmpDir, relPath)
	if err != nil {
		t.Fatalf("failed to create storage lock: %v", err)
	}

	// Create a simulated stale lock file (written 60 seconds ago)
	stalePID := 999999
	staleTS := time.Now().Unix() - 60
	staleData := fmt.Sprintf("%d\n%d\n", stalePID, staleTS)
	if err := os.WriteFile(lock.lockPath, []byte(staleData), 0600); err != nil {
		t.Fatalf("failed to write stale lock: %v", err)
	}

	// Now try to acquire the lock; it should detect the stale timestamp, remove the file, and succeed
	lock.SetTimeout(2 * time.Second)
	if err := lock.Lock(); err != nil {
		t.Fatalf("failed to acquire lock over stale lock: %v", err)
	}
	defer func() { _ = lock.Unlock() }()

	// Verify lock file exists and has current PID
	data, err := os.ReadFile(lock.lockPath)
	if err != nil {
		t.Fatalf("failed to read lock file: %v", err)
	}
	if !strings.Contains(string(data), fmt.Sprintf("%d", os.Getpid())) {
		t.Fatalf("expected lock file to contain current pid %d, got: %s", os.Getpid(), string(data))
	}
}
