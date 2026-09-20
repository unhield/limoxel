package marketplace

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/plugin/marketplace/storage"
	"github.com/unhield/limoxel/internal/capabilities/plugin/marketplace/validation"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
	pubmarket "github.com/unhield/limoxel/plugin/marketplace"
)

func createTestPackageArchive(id, name, ver, pub string) []byte {
	manifest := validation.ManifestSchema{
		ID:             id,
		Name:           name,
		Version:        ver,
		Publisher:      pub,
		Description:    "Test description for marketplace integration",
		MinHostVersion: "1.0.0",
		Categories:     []string{"analysis", "tools"},
	}
	manifestBytes, _ := json.Marshal(manifest)

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// plugin.json
	hdr := &tar.Header{
		Name: "plugin.json",
		Mode: 0644,
		Size: int64(len(manifestBytes)),
	}
	_ = tw.WriteHeader(hdr)
	_, _ = tw.Write(manifestBytes)

	// payload
	code := []byte("function execute() { return true; }")
	hdrCode := &tar.Header{
		Name: "main.js",
		Mode: 0644,
		Size: int64(len(code)),
	}
	_ = tw.WriteHeader(hdrCode)
	_, _ = tw.Write(code)

	_ = tw.Close()
	_ = gw.Close()

	return buf.Bytes()
}

func TestService_PublishAndDownloadFlow(t *testing.T) {
	tmpDir := t.TempDir()
	svc, err := NewService(Config{
		StorageDir:       tmpDir,
		ValidationPolicy: validation.DefaultPolicy(),
	})
	if err != nil {
		t.Fatal(err)
	}

	client := NewLocalClient(svc)
	ctx := context.Background()

	// 1. Publish package
	archiveData := createTestPackageArchive("code-analyzer", "Code Analyzer", "1.0.0", "acme-devs")
	vInfo, valRes, err := svc.PublishPackage(ctx, archiveData)
	if err != nil {
		t.Fatalf("failed to publish package: %v, errors: %+v", err, valRes.Errors)
	}

	if vInfo.Version.String() != "1.0.0" {
		t.Fatalf("expected version 1.0.0, got %s", vInfo.Version.String())
	}
	if vInfo.ArtifactDigest == "" {
		t.Fatal("expected artifact digest to be non-empty")
	}

	// 2. Immutability check: publish duplicate version must fail
	_, _, err = svc.PublishPackage(ctx, archiveData)
	if err != ErrVersionConflict {
		t.Fatalf("expected ErrVersionConflict, got %v", err)
	}

	// 3. Search for published plugin
	res, err := client.Search(ctx, pubmarket.SearchQuery{Keyword: "analyzer"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 1 || res.Items[0].ID != "code-analyzer" {
		t.Fatalf("expected 1 search match for code-analyzer, got %+v", res)
	}

	// 4. Download artifact to file
	downloadDest := filepath.Join(tmpDir, "downloads", "code-analyzer-1.0.0.tar.gz")
	downloadedInfo, err := client.DownloadArtifact(ctx, "code-analyzer", vInfo.Version, downloadDest)
	if err != nil {
		t.Fatalf("failed to download artifact: %v", err)
	}
	if downloadedInfo.ArtifactDigest != vInfo.ArtifactDigest {
		t.Fatalf("downloaded digest mismatch: %s != %s", downloadedInfo.ArtifactDigest, vInfo.ArtifactDigest)
	}

	// Verify file exists on disk
	fi, err := os.Stat(downloadDest)
	if err != nil || fi.Size() == 0 {
		t.Fatalf("downloaded file missing or empty: %v", err)
	}

	// 5. Verify download metrics
	detail, err := client.GetPlugin(ctx, "code-analyzer")
	if err != nil {
		t.Fatal(err)
	}
	if detail.TotalDownloads != 1 {
		t.Fatalf("expected 1 download recorded, got %d", detail.TotalDownloads)
	}
}

func TestService_CommunityInteractions(t *testing.T) {
	tmpDir := t.TempDir()
	svc, err := NewService(Config{
		StorageDir:       tmpDir,
		ValidationPolicy: validation.DefaultPolicy(),
	})
	if err != nil {
		t.Fatal(err)
	}

	client := NewLocalClient(svc)
	ctx := context.Background()

	archiveData := createTestPackageArchive("cool-formatter", "Cool Formatter", "1.0.0", "format-team")
	_, _, err = svc.PublishPackage(ctx, archiveData)
	if err != nil {
		t.Fatal(err)
	}

	// 1. Submit Rating from user1 (5 stars)
	rSummary, err := client.SubmitRating(ctx, "cool-formatter", "user1", 5)
	if err != nil {
		t.Fatal(err)
	}
	if rSummary.Average != 5.0 || rSummary.Count != 1 {
		t.Fatalf("unexpected rating summary: %+v", rSummary)
	}

	// 2. Submit Rating from user2 (3 stars)
	rSummary, err = client.SubmitRating(ctx, "cool-formatter", "user2", 3)
	if err != nil {
		t.Fatal(err)
	}
	// Average = (5+3)/2 = 4.0
	if rSummary.Average != 4.0 || rSummary.Count != 2 {
		t.Fatalf("expected avg 4.0 after user2, got %+v", rSummary)
	}

	// 3. Submit Review
	err = client.SubmitReview(ctx, pubmarket.Review{
		PluginID: "cool-formatter",
		Version:  "1.0.0",
		AuthorID: "user3",
		Rating:   4,
		Title:    "Clean formatting",
		Comment:  "Format is very fast and reliable.",
	})
	if err != nil {
		t.Fatal(err)
	}

	reviews, err := svc.GetReviews(ctx, "cool-formatter")
	if err != nil || len(reviews) != 1 {
		t.Fatalf("expected 1 review, got %d", len(reviews))
	}
	if reviews[0].Title != "Clean formatting" {
		t.Fatalf("unexpected review title: %s", reviews[0].Title)
	}
}

func TestService_UpdateChecks(t *testing.T) {
	tmpDir := t.TempDir()
	svc, err := NewService(Config{
		StorageDir:       tmpDir,
		ValidationPolicy: validation.DefaultPolicy(),
	})
	if err != nil {
		t.Fatal(err)
	}

	client := NewLocalClient(svc)
	ctx := context.Background()

	// Publish v1.0.0
	arc1 := createTestPackageArchive("smart-tool", "Smart Tool", "1.0.0", "tool-makers")
	_, _, err = svc.PublishPackage(ctx, arc1)
	if err != nil {
		t.Fatal(err)
	}

	// Publish v1.2.0
	arc2 := createTestPackageArchive("smart-tool", "Smart Tool", "1.2.0", "tool-makers")
	_, _, err = svc.PublishPackage(ctx, arc2)
	if err != nil {
		t.Fatal(err)
	}

	// Check update for user on v1.0.0 -> should report update available
	installed := []pubmarket.UpdateCheckItem{
		{
			PluginID:       "smart-tool",
			CurrentVersion: version.SemVer{Major: 1, Minor: 0, Patch: 0},
		},
	}
	updates, err := client.CheckUpdates(ctx, installed)
	if err != nil {
		t.Fatal(err)
	}
	if len(updates) != 1 || !updates[0].HasUpdate {
		t.Fatalf("expected update available for v1.0.0, got: %+v", updates)
	}
	if updates[0].LatestVersion.String() != "1.2.0" {
		t.Fatalf("expected latest version 1.2.0, got %s", updates[0].LatestVersion.String())
	}

	// Check update for user on v1.2.0 -> should report no update
	installedUpToDate := []pubmarket.UpdateCheckItem{
		{
			PluginID:       "smart-tool",
			CurrentVersion: version.SemVer{Major: 1, Minor: 2, Patch: 0},
		},
	}
	updates, err = client.CheckUpdates(ctx, installedUpToDate)
	if err != nil {
		t.Fatal(err)
	}
	if len(updates) != 1 || updates[0].HasUpdate {
		t.Fatalf("expected no update for v1.2.0, got: %+v", updates)
	}
}

func TestService_SafeExtractor(t *testing.T) {
	extractor := NewSafeExtractor()
	destDir := t.TempDir()

	// 1. Valid archive extracts successfully
	validData := createTestPackageArchive("extract-test", "Extract Test", "1.0.0", "pub")
	r := bytes.NewReader(validData)
	if err := extractor.Extract(r, destDir); err != nil {
		t.Fatalf("safe extraction failed on valid archive: %v", err)
	}

	manifestPath := filepath.Join(destDir, "plugin.json")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("expected plugin.json extracted: %v", err)
	}

	// 2. Malicious archive with path traversal must be rejected
	var evilBuf bytes.Buffer
	gw := gzip.NewWriter(&evilBuf)
	tw := tar.NewWriter(gw)
	_ = tw.WriteHeader(&tar.Header{
		Name: "../outside.txt",
		Mode: 0644,
		Size: 4,
	})
	_, _ = tw.Write([]byte("evil"))
	_ = tw.Close()
	_ = gw.Close()

	evilDir := filepath.Join(destDir, "evil_target")
	err := extractor.Extract(bytes.NewReader(evilBuf.Bytes()), evilDir)
	if err == nil {
		t.Fatal("expected traversal archive to fail extraction")
	}
}

func TestService_Concurrency(t *testing.T) {
	tmpDir := t.TempDir()
	svc, err := NewService(Config{
		StorageDir:       tmpDir,
		ValidationPolicy: validation.DefaultPolicy(),
	})
	if err != nil {
		t.Fatal(err)
	}

	client := NewLocalClient(svc)
	ctx := context.Background()

	// Pre-publish a plugin
	arc := createTestPackageArchive("concurrent-tool", "Concurrent Tool", "1.0.0", "multi-dev")
	_, _, err = svc.PublishPackage(ctx, arc)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	workers := 10

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Perform searches
			_, _ = client.Search(ctx, pubmarket.SearchQuery{Keyword: "concurrent"})

			// Perform ratings
			author := filepath.Join("author", string(rune('A'+workerID)))
			_, _ = client.SubmitRating(ctx, "concurrent-tool", author, (workerID%5)+1)

			// Perform update checks
			_, _ = client.CheckUpdates(ctx, []pubmarket.UpdateCheckItem{
				{PluginID: "concurrent-tool", CurrentVersion: version.SemVer{Major: 1, Minor: 0, Patch: 0}},
			})
		}(i)
	}

	wg.Wait()

	// Verify ratings and state consistency
	detail, err := client.GetPlugin(ctx, "concurrent-tool")
	if err != nil {
		t.Fatal(err)
	}
	if detail.RatingCount != workers {
		t.Fatalf("expected %d ratings recorded, got %d", workers, detail.RatingCount)
	}
}

func TestService_PublisherNamespaceAuthorization(t *testing.T) {
	tmpDir := t.TempDir()
	svc, err := NewService(Config{
		StorageDir:       tmpDir,
		ValidationPolicy: validation.DefaultPolicy(),
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// 1. Publisher "alice" publishes "auth-plugin" v1.0.0
	arcAlice := createTestPackageArchive("auth-plugin", "Auth Plugin", "1.0.0", "alice")
	_, _, err = svc.PublishPackage(ctx, arcAlice)
	if err != nil {
		t.Fatalf("alice publish should succeed: %v", err)
	}

	// 2. Attacker "bob" attempts to publish update v1.1.0 to alice's plugin -> MUST BE REJECTED
	arcBob := createTestPackageArchive("auth-plugin", "Auth Plugin", "1.1.0", "bob")
	_, _, err = svc.PublishPackage(ctx, arcBob)
	if err == nil {
		t.Fatal("expected unauthorized update by bob to fail, but it succeeded")
	}
	if !errors.Is(err, pubmarket.ErrUnauthorizedPublisher) {
		t.Fatalf("expected ErrUnauthorizedPublisher, got: %v", err)
	}

	// 3. Legitimate publisher "alice" publishes v1.1.0 -> MUST SUCCEED
	arcAlice2 := createTestPackageArchive("auth-plugin", "Auth Plugin", "1.1.0", "alice")
	vInfo, _, err := svc.PublishPackage(ctx, arcAlice2)
	if err != nil {
		t.Fatalf("alice update should succeed: %v", err)
	}
	if vInfo.Version.String() != "1.1.0" {
		t.Fatalf("expected 1.1.0, got %s", vInfo.Version.String())
	}
}

func TestService_DownloadAccountingVerification(t *testing.T) {
	tmpDir := t.TempDir()
	svc, err := NewService(Config{
		StorageDir:       tmpDir,
		ValidationPolicy: validation.DefaultPolicy(),
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	client := NewLocalClient(svc)

	arc := createTestPackageArchive("download-meter", "Download Meter", "1.0.0", "meter-org")
	vInfo, _, err := svc.PublishPackage(ctx, arc)
	if err != nil {
		t.Fatal(err)
	}

	// Case 1: Partial transfer aborted before EOF -> download count must NOT increment
	stream, _, err := svc.DownloadArtifact(ctx, "download-meter", vInfo.Version)
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 10)
	_, _ = stream.Read(buf) // read only 10 bytes, no EOF
	_ = stream.Close()

	detail, err := client.GetPlugin(ctx, "download-meter")
	if err != nil {
		t.Fatal(err)
	}
	if detail.TotalDownloads != 0 {
		t.Fatalf("expected 0 downloads after partial aborted transfer, got %d", detail.TotalDownloads)
	}

	// Case 2: Complete verified transfer -> download count must increment to 1
	destFile := filepath.Join(tmpDir, "meter.tar.gz")
	_, err = client.DownloadArtifact(ctx, "download-meter", vInfo.Version, destFile)
	if err != nil {
		t.Fatalf("complete download failed: %v", err)
	}

	detail, err = client.GetPlugin(ctx, "download-meter")
	if err != nil {
		t.Fatal(err)
	}
	if detail.TotalDownloads != 1 {
		t.Fatalf("expected 1 download after completed transfer, got %d", detail.TotalDownloads)
	}
}

func TestService_AdversarialExtractionSecurity(t *testing.T) {
	extractor := NewSafeExtractor()
	targetDir := t.TempDir()

	makeArchive := func(hdr *tar.Header, content []byte) io.Reader {
		var buf bytes.Buffer
		gw := gzip.NewWriter(&buf)
		tw := tar.NewWriter(gw)
		_ = tw.WriteHeader(hdr)
		if len(content) > 0 {
			_, _ = tw.Write(content)
		}
		_ = tw.Close()
		_ = gw.Close()
		return bytes.NewReader(buf.Bytes())
	}

	tests := []struct {
		name    string
		hdr     *tar.Header
		content []byte
	}{
		{
			name: "symlink entry rejected",
			hdr: &tar.Header{
				Name:     "symlink_file",
				Typeflag: tar.TypeSymlink,
				Linkname: "/etc/passwd",
			},
		},
		{
			name: "hardlink entry rejected",
			hdr: &tar.Header{
				Name:     "hardlink_file",
				Typeflag: tar.TypeLink,
				Linkname: "../root.txt",
			},
		},
		{
			name: "character device rejected",
			hdr: &tar.Header{
				Name:     "dev_null",
				Typeflag: tar.TypeChar,
			},
		},
		{
			name: "windows alternate data stream rejected",
			hdr: &tar.Header{
				Name:     "test.txt:hidden_stream",
				Typeflag: tar.TypeReg,
				Size:     4,
			},
			content: []byte("evil"),
		},
		{
			name: "windows drive letter rejected",
			hdr: &tar.Header{
				Name:     "C:/escaped.txt",
				Typeflag: tar.TypeReg,
				Size:     4,
			},
			content: []byte("evil"),
		},
		{
			name: "unc path rejected",
			hdr: &tar.Header{
				Name:     "//attacker/share/evil.txt",
				Typeflag: tar.TypeReg,
				Size:     4,
			},
			content: []byte("evil"),
		},
		{
			name: "reserved device name CON rejected",
			hdr: &tar.Header{
				Name:     "CON.txt",
				Typeflag: tar.TypeReg,
				Size:     4,
			},
			content: []byte("evil"),
		},
		{
			name: "reserved device name NUL rejected",
			hdr: &tar.Header{
				Name:     "sub/NUL",
				Typeflag: tar.TypeReg,
				Size:     4,
			},
			content: []byte("evil"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := makeArchive(tc.hdr, tc.content)
			outDir := filepath.Join(targetDir, tc.name)
			err := extractor.Extract(r, outDir)
			if err == nil {
				t.Fatalf("expected extraction failure for %s, got nil", tc.name)
			}
			if !errors.Is(err, ErrExtractionSecurity) {
				t.Fatalf("expected ErrExtractionSecurity for %s, got %v", tc.name, err)
			}
		})
	}
}

func TestService_HTTPNetworkBoundary(t *testing.T) {
	tmpDir := t.TempDir()
	svc, err := NewService(Config{
		StorageDir:       tmpDir,
		ValidationPolicy: validation.DefaultPolicy(),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Start local test HTTP server
	server := httptest.NewServer(NewHTTPHandler(svc))
	defer server.Close()

	// Construct HTTP remote client
	client := NewHTTPClient(server.URL, server.Client())
	ctx := context.Background()

	// Pre-publish a plugin directly through service
	arc := createTestPackageArchive("network-plugin", "Network Plugin", "1.0.0", "net-corp")
	vInfo, _, err := svc.PublishPackage(ctx, arc)
	if err != nil {
		t.Fatal(err)
	}

	// 1. Search over HTTP
	res, err := client.Search(ctx, pubmarket.SearchQuery{Keyword: "Network"})
	if err != nil {
		t.Fatalf("HTTP Search failed: %v", err)
	}
	if res.Total != 1 || res.Items[0].ID != "network-plugin" {
		t.Fatalf("unexpected HTTP search result: %+v", res)
	}

	// 2. GetPlugin over HTTP
	detail, err := client.GetPlugin(ctx, "network-plugin")
	if err != nil {
		t.Fatalf("HTTP GetPlugin failed: %v", err)
	}
	if detail.Name != "Network Plugin" {
		t.Fatalf("expected Network Plugin, got %s", detail.Name)
	}

	// 3. GetVersions over HTTP
	versions, err := client.GetVersions(ctx, "network-plugin")
	if err != nil || len(versions) != 1 {
		t.Fatalf("HTTP GetVersions failed: %v", err)
	}

	// 4. SubmitRating over HTTP
	rSum, err := client.SubmitRating(ctx, "network-plugin", "http-user", 5)
	if err != nil || rSum.Count != 1 || rSum.Average != 5.0 {
		t.Fatalf("HTTP SubmitRating failed: %v, summary: %+v", err, rSum)
	}

	// 5. SubmitReview over HTTP
	err = client.SubmitReview(ctx, pubmarket.Review{
		PluginID: "network-plugin",
		Version:  "1.0.0",
		AuthorID: "http-user",
		Rating:   5,
		Title:    "HTTP Review",
		Comment:  "Works cleanly over network",
	})
	if err != nil {
		t.Fatalf("HTTP SubmitReview failed: %v", err)
	}

	// 6. CheckUpdates over HTTP
	updates, err := client.CheckUpdates(ctx, []pubmarket.UpdateCheckItem{
		{PluginID: "network-plugin", CurrentVersion: version.SemVer{Major: 0, Minor: 9, Patch: 0}},
	})
	if err != nil || len(updates) != 1 || !updates[0].HasUpdate {
		t.Fatalf("HTTP CheckUpdates failed: %v, updates: %+v", err, updates)
	}

	// 7. DownloadArtifact over HTTP
	destDownload := filepath.Join(tmpDir, "http_download.tar.gz")
	downloadedInfo, err := client.DownloadArtifact(ctx, "network-plugin", vInfo.Version, destDownload)
	if err != nil {
		t.Fatalf("HTTP DownloadArtifact failed: %v", err)
	}
	if downloadedInfo.ArtifactDigest != vInfo.ArtifactDigest {
		t.Fatalf("HTTP downloaded digest mismatch: %s != %s", downloadedInfo.ArtifactDigest, vInfo.ArtifactDigest)
	}

	// 8. ResolveDependencies over HTTP
	plan, err := client.ResolveDependencies(ctx, "network-plugin", vInfo.Version)
	if err != nil {
		t.Fatalf("HTTP ResolveDependencies failed: %v", err)
	}
	if plan.TargetPlugin != "network-plugin" || len(plan.InstallOrder) != 1 {
		t.Fatalf("unexpected resolution plan over HTTP: %+v", plan)
	}
}

func TestService_DownloadVelocityTrending(t *testing.T) {
	tmpDir := t.TempDir()
	svc, err := NewService(Config{
		StorageDir:       tmpDir,
		ValidationPolicy: validation.DefaultPolicy(),
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// Publish Plugin A (older popular plugin with downloads)
	arcA := createTestPackageArchive("plugin-alpha", "Plugin Alpha", "1.0.0", "alpha-dev")
	_, _, err = svc.PublishPackage(ctx, arcA)
	if err != nil {
		t.Fatal(err)
	}

	// Publish Plugin B (new breakout plugin with rapid downloads)
	arcB := createTestPackageArchive("plugin-beta", "Plugin Beta", "1.0.0", "beta-dev")
	verB, _, err := svc.PublishPackage(ctx, arcB)
	if err != nil {
		t.Fatal(err)
	}

	// Simulate high velocity for Plugin B by downloading 5 times
	for i := 0; i < 5; i++ {
		stream, _, err := svc.DownloadArtifact(ctx, "plugin-beta", verB.Version)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, stream)
		_ = stream.Close()
	}

	trending, err := svc.GetTrending(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}

	if len(trending) < 2 {
		t.Fatalf("expected at least 2 trending candidates, got %d", len(trending))
	}

	// Plugin Beta must be ranked #1 trending due to download velocity
	if trending[0].ID != "plugin-beta" {
		t.Fatalf("expected plugin-beta to rank #1 trending due to velocity, but got %s", trending[0].ID)
	}
}

func TestStorage_ProcessFileLock(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "data.json")

	lock1 := storage.NewProcessFileLock(targetPath)
	if err := lock1.Lock(); err != nil {
		t.Fatalf("first lock acquisition failed: %v", err)
	}

	// Verify second lock with short timeout times out while lock1 is held
	lock2 := storage.NewProcessFileLock(targetPath)
	lock2.SetTimeout(50 * time.Millisecond)
	err := lock2.Lock()
	if err == nil {
		t.Fatal("expected lock2 acquisition to time out while lock1 is held")
	}
	if !errors.Is(err, storage.ErrLockTimeout) {
		t.Fatalf("expected ErrLockTimeout, got: %v", err)
	}

	// Release lock1
	if err := lock1.Unlock(); err != nil {
		t.Fatalf("unlock failed: %v", err)
	}

	// Now lock2 should succeed
	lock2.SetTimeout(500 * time.Millisecond)
	if err := lock2.Lock(); err != nil {
		t.Fatalf("lock2 acquisition failed after lock1 released: %v", err)
	}
	_ = lock2.Unlock()
}
