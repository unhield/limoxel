package community

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"strings"
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/plugin/marketplace/storage"
	"github.com/unhield/limoxel/internal/capabilities/plugin/security/verification"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
	pubmarket "github.com/unhield/limoxel/plugin/marketplace"
)

type mockRatingUpdater struct {
	updatedID     string
	updatedRating pubmarket.RatingSummary
}

func (m *mockRatingUpdater) UpdateRating(id string, r pubmarket.RatingSummary) error {
	m.updatedID = id
	m.updatedRating = r
	return nil
}

type mockDownloadNotifier struct {
	downloadedID  string
	downloadedVer version.SemVer
}

func (m *mockDownloadNotifier) IncrementDownloads(id string, ver version.SemVer) error {
	m.downloadedID = id
	m.downloadedVer = ver
	return nil
}

func TestCommunity_Ratings(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := storage.NewFileStorage(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	updater := &mockRatingUpdater{}
	rm := NewRatingManager(store, updater)

	// 1. Invalid star values
	if _, err := rm.RecordRating("p1", "alice", 0); err != ErrInvalidRating {
		t.Fatalf("expected ErrInvalidRating for 0, got %v", err)
	}
	if _, err := rm.RecordRating("p1", "alice", 6); err != ErrInvalidRating {
		t.Fatalf("expected ErrInvalidRating for 6, got %v", err)
	}

	// 2. Record rating 5 from Alice
	sum, err := rm.RecordRating("p1", "alice", 5)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Count != 1 || sum.Average != 5.0 || sum.Stars[5] != 1 {
		t.Fatalf("unexpected summary: %+v", sum)
	}
	if updater.updatedID != "p1" || updater.updatedRating.Average != 5.0 {
		t.Fatalf("updater not notified: %+v", updater)
	}

	// 3. Record rating 4 from Bob
	sum, err = rm.RecordRating("p1", "bob", 4)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Count != 2 || sum.Average != 4.5 || sum.Stars[4] != 1 || sum.Stars[5] != 1 {
		t.Fatalf("unexpected summary after bob: %+v", sum)
	}

	// 4. Single-vote-per-user: Alice changes rating from 5 to 3
	sum, err = rm.RecordRating("p1", "alice", 3)
	if err != nil {
		t.Fatal(err)
	}
	// Total count remains 2 (Alice + Bob). Average = (3+4)/2 = 3.5.
	if sum.Count != 2 || sum.Average != 3.5 || sum.Stars[3] != 1 || sum.Stars[5] != 0 {
		t.Fatalf("expected vote update, got %+v", sum)
	}
}

func TestCommunity_Reviews(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := storage.NewFileStorage(tmpDir)
	rm := NewRatingManager(store, nil)
	revManager := NewReviewManager(store, rm)

	// 1. Submit valid review
	rev := pubmarket.Review{
		PluginID: "p1",
		Version:  "1.0.0",
		AuthorID: "carol",
		Rating:   4,
		Title:    "Great plugin!\x00",
		Comment:  "Works as advertised.\x07",
	}
	if err := revManager.SubmitReview(rev); err != nil {
		t.Fatalf("failed to submit review: %v", err)
	}

	reviews, err := revManager.GetReviews("p1")
	if err != nil || len(reviews) != 1 {
		t.Fatalf("expected 1 review, got %d, err: %v", len(reviews), err)
	}

	// Verify control character sanitization
	if strings.Contains(reviews[0].Title, "\x00") || strings.Contains(reviews[0].Comment, "\x07") {
		t.Errorf("control characters were not sanitized: %+v", reviews[0])
	}

	// 2. Review title too long
	longTitle := strings.Repeat("A", 201)
	if err := revManager.SubmitReview(pubmarket.Review{
		PluginID: "p1",
		AuthorID: "david",
		Rating:   3,
		Title:    longTitle,
	}); err != ErrReviewTooLong {
		t.Fatalf("expected ErrReviewTooLong, got %v", err)
	}
}

func TestCommunity_Publishers(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := storage.NewFileStorage(tmpDir)
	pubManager := NewPublisherManager(store)

	// 1. Invalid publisher ID
	if err := pubManager.RegisterPublisher(pubmarket.PublisherProfile{ID: "x"}); !errors.Is(err, ErrInvalidPublisherID) {
		t.Fatalf("expected ErrInvalidPublisherID, got %v", err)
	}

	// 2. Register valid publisher
	p := pubmarket.PublisherProfile{
		ID:          "acme-corp",
		DisplayName: "Acme Corporation",
		Email:       "contact@acme.org",
	}
	if err := pubManager.RegisterPublisher(p); err != nil {
		t.Fatalf("failed to register publisher: %v", err)
	}

	got, err := pubManager.GetPublisher("acme-corp")
	if err != nil || got.DisplayName != "Acme Corporation" {
		t.Fatalf("failed to get publisher: %+v, err: %v", got, err)
	}
	if got.Verified {
		t.Fatalf("publisher should not be verified yet")
	}

	// 3. Verify publisher without trust store
	if err := pubManager.VerifyPublisher("acme-corp", "key-ed25519-abc123"); err != nil {
		t.Fatalf("failed to verify publisher: %v", err)
	}

	verified, err := pubManager.GetPublisher("acme-corp")
	if err != nil || !verified.Verified || verified.KeyID != "key-ed25519-abc123" {
		t.Fatalf("expected verified publisher, got %+v", verified)
	}

	// 4. Test with TrustStore attached
	ts := verification.NewTrustStore()
	pubKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := ts.RegisterKey("acme-corp", "key-valid", pubKey, "Acme Key"); err != nil {
		t.Fatal(err)
	}
	if err := ts.RegisterKey("other-corp", "key-other", pubKey, "Other Key"); err != nil {
		t.Fatal(err)
	}
	if err := ts.RegisterKey("acme-corp", "key-revoked", pubKey, "Revoked Key"); err != nil {
		t.Fatal(err)
	}
	if err := ts.RevokeKey("key-revoked"); err != nil {
		t.Fatal(err)
	}

	pubManager.SetTrustStore(ts)

	// Attempt verification with unknown key
	if err := pubManager.VerifyPublisher("acme-corp", "unknown-key"); err == nil {
		t.Fatal("expected error verifying with unknown key")
	}

	// Attempt verification with revoked key
	if err := pubManager.VerifyPublisher("acme-corp", "key-revoked"); err == nil {
		t.Fatal("expected error verifying with revoked key")
	}

	// Attempt verification with key belonging to different publisher
	if err := pubManager.VerifyPublisher("acme-corp", "key-other"); err == nil {
		t.Fatal("expected error verifying with key belonging to different publisher")
	}

	// Successful verification with valid registered key
	if err := pubManager.VerifyPublisher("acme-corp", "key-valid"); err != nil {
		t.Fatalf("expected success verifying with valid key, got: %v", err)
	}
}

func TestCommunity_Metrics(t *testing.T) {
	notifier := &mockDownloadNotifier{}
	tracker := NewDownloadTracker(notifier)

	ver := version.SemVer{Major: 1, Minor: 0, Patch: 0}
	if err := tracker.RecordDownload("my-plugin", ver); err != nil {
		t.Fatalf("failed to record download: %v", err)
	}

	if tracker.GetCount("my-plugin") != 1 {
		t.Fatalf("expected count 1, got %d", tracker.GetCount("my-plugin"))
	}

	if notifier.downloadedID != "my-plugin" || notifier.downloadedVer.String() != "1.0.0" {
		t.Fatalf("notifier not called properly: %+v", notifier)
	}
}
