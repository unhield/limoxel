package marketplace

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"sync"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/plugin/marketplace/catalog"
	"github.com/unhield/limoxel/internal/capabilities/plugin/marketplace/community"
	"github.com/unhield/limoxel/internal/capabilities/plugin/marketplace/dependency"
	"github.com/unhield/limoxel/internal/capabilities/plugin/marketplace/discovery"
	"github.com/unhield/limoxel/internal/capabilities/plugin/marketplace/storage"
	"github.com/unhield/limoxel/internal/capabilities/plugin/marketplace/validation"
	"github.com/unhield/limoxel/internal/capabilities/plugin/security/verification"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
	pubmarket "github.com/unhield/limoxel/plugin/marketplace"
)

var (
	// ErrVersionConflict indicates the published version already exists.
	ErrVersionConflict = errors.New("marketplace service: version already published and immutable")
)

// Service coordinates catalog, discovery, community, storage, and validation.
type Service struct {
	mu         sync.RWMutex
	store      storage.Storage
	cat        *catalog.Catalog
	disc       *discovery.Engine
	ratings    *community.RatingManager
	reviews    *community.ReviewManager
	publishers *community.PublisherManager
	tracker    *community.DownloadTracker
	validator  *validation.Pipeline
	resolver   *dependency.Resolver
}

// Config holds initialization parameters for the marketplace service.
type Config struct {
	StorageDir       string
	ValidationPolicy validation.ValidationPolicy
	TrustStore       *verification.TrustStore
	MalwareScanner   validation.MalwareScanner
}

// NewService constructs and initializes a complete Marketplace Service.
func NewService(cfg Config) (*Service, error) {
	store, err := storage.NewFileStorage(cfg.StorageDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	cat, err := catalog.NewCatalog(store)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize catalog: %w", err)
	}

	disc := discovery.NewEngine(cat)
	ratings := community.NewRatingManager(store, cat)
	reviews := community.NewReviewManager(store, ratings)
	publishers := community.NewPublisherManager(store)
	if cfg.TrustStore != nil {
		publishers.SetTrustStore(cfg.TrustStore)
	}
	tracker := community.NewDownloadTracker(cat)
	disc.SetVelocityTracker(tracker)

	pipe := validation.NewPipeline(cfg.ValidationPolicy, cfg.MalwareScanner, cfg.TrustStore)
	resolver := dependency.NewResolver(cat)

	return &Service{
		store:      store,
		cat:        cat,
		disc:       disc,
		ratings:    ratings,
		reviews:    reviews,
		publishers: publishers,
		tracker:    tracker,
		validator:  pipe,
		resolver:   resolver,
	}, nil
}

// PublishPackage executes automated validation and publishes an immutable plugin release.
func (s *Service) PublishPackage(
	ctx context.Context,
	archiveData []byte,
) (*pubmarket.VersionInfo, *validation.ValidationResult, error) {
	if len(archiveData) == 0 {
		return nil, nil, fmt.Errorf("%w: empty package archive", pubmarket.ErrInvalidInput)
	}

	// 1. Run validation pipeline
	valResult, pkg, err := s.validator.ValidatePackage(ctx, archiveData, "")
	if err != nil || !valResult.Passed {
		return nil, valResult, fmt.Errorf("validation failed: %w", err)
	}

	pluginID := pkg.Manifest.ID
	ver := pkg.ParsedVersion
	artifactDigest := valResult.ArtifactDigest

	s.mu.Lock()
	defer s.mu.Unlock()

	// 2. Check namespace authorization and whether version already exists
	existing, err := s.cat.GetPlugin(pluginID)
	if err == nil && existing != nil {
		// Enforce publisher namespace ownership: only the registering publisher can publish updates
		if existing.PublisherID != "" && existing.PublisherID != pkg.Manifest.Publisher {
			return nil, valResult, fmt.Errorf("%w: plugin %s owned by %s, cannot be updated by %s",
				pubmarket.ErrUnauthorizedPublisher, pluginID, existing.PublisherID, pkg.Manifest.Publisher)
		}

		for _, v := range existing.Versions {
			if v.Version.Compare(ver) == 0 {
				return nil, valResult, ErrVersionConflict
			}
		}
	}

	// 3. Store artifact safely in immutable content-addressed storage
	r := bytes.NewReader(archiveData)
	savedDigest, size, err := s.store.SaveArtifact(artifactDigest, r)
	if err != nil {
		return nil, valResult, fmt.Errorf("failed to persist artifact: %w", err)
	}

	// 4. Ensure publisher is registered
	pubProfile, _ := s.publishers.GetPublisher(pkg.Manifest.Publisher)
	if pubProfile == nil {
		// Auto-register publisher if not yet present
		_ = s.publishers.RegisterPublisher(pubmarket.PublisherProfile{
			ID:          pkg.Manifest.Publisher,
			DisplayName: pkg.Manifest.Publisher,
		})
		pubProfile, _ = s.publishers.GetPublisher(pkg.Manifest.Publisher)
	}

	var publisher pubmarket.PublisherProfile
	if pubProfile != nil {
		publisher = *pubProfile
	} else {
		publisher = pubmarket.PublisherProfile{
			ID:          pkg.Manifest.Publisher,
			DisplayName: pkg.Manifest.Publisher,
		}
	}

	// 5. Convert categories
	var categories []pubmarket.Category
	for _, c := range pkg.Manifest.Categories {
		categories = append(categories, pubmarket.Category(c))
	}
	if len(categories) == 0 {
		categories = []pubmarket.Category{pubmarket.CategoryTools}
	}

	// 6. Build version info
	now := time.Now().UTC()
	var minHost *version.SemVer
	if pkg.Manifest.MinHostVersion != "" {
		if mh, err := version.ParseSemVer(pkg.Manifest.MinHostVersion); err == nil {
			minHost = &mh
		}
	}

	versionInfo := pubmarket.VersionInfo{
		Version:        ver,
		ArtifactDigest: savedDigest,
		ArtifactSize:   size,
		DownloadURL:    fmt.Sprintf("/api/v1/plugins/%s/versions/%s/download", pluginID, ver.String()),
		PublishedAt:    now,
		Status:         pubmarket.StatusPublished,
		MinHostVersion: minHost,
	}

	// 7. Register or update plugin in catalog
	if existing == nil {
		newDetail := pubmarket.PluginDetail{
			PluginSummary: pubmarket.PluginSummary{
				ID:             pluginID,
				Name:           pkg.Manifest.Name,
				Description:    pkg.Manifest.Description,
				PublisherID:    pkg.Manifest.Publisher,
				PublisherName:  publisher.DisplayName,
				PublisherValid: publisher.Verified,
				LatestVersion:  ver,
				Categories:     categories,
				Status:         pubmarket.StatusPublished,
				UpdatedAt:      now,
			},
			Capabilities: pkg.Manifest.Capabilities,
			Permissions:  pkg.Manifest.Permissions,
			Versions:     []pubmarket.VersionInfo{versionInfo},
			Publisher:    publisher,
		}
		if err := s.cat.RegisterPlugin(newDetail); err != nil {
			return nil, valResult, fmt.Errorf("failed to register plugin: %w", err)
		}
	} else {
		if err := s.cat.AddVersion(pluginID, versionInfo); err != nil {
			return nil, valResult, fmt.Errorf("failed to add version: %w", err)
		}
	}

	return &versionInfo, valResult, nil
}

// DownloadArtifact retrieves the content stream for a plugin version and records verified download metrics.
func (s *Service) DownloadArtifact(
	ctx context.Context,
	pluginID string,
	ver version.SemVer,
) (io.ReadCloser, *pubmarket.VersionInfo, error) {
	p, err := s.cat.GetPlugin(pluginID)
	if err != nil {
		return nil, nil, pubmarket.ErrNotFound
	}

	var targetVer *pubmarket.VersionInfo
	for _, v := range p.Versions {
		if v.Version.Compare(ver) == 0 {
			targetVer = &v
			break
		}
	}

	if targetVer == nil {
		return nil, nil, pubmarket.ErrVersionNotFound
	}

	stream, _, err := s.store.GetArtifact(targetVer.ArtifactDigest)
	if err != nil {
		return nil, nil, fmt.Errorf("artifact not found in storage: %w", err)
	}

	// Wrap in accountingReadCloser to increment metrics ONLY after complete, verified transfer
	wrappedStream := newAccountingReadCloser(stream, targetVer.ArtifactDigest, func() {
		_ = s.tracker.RecordDownload(pluginID, ver)
	})

	return wrappedStream, targetVer, nil
}

// CheckUpdates inspects installed versions and returns any available newer releases.
func (s *Service) CheckUpdates(
	ctx context.Context,
	installed []pubmarket.UpdateCheckItem,
) ([]pubmarket.UpdateAvailableInfo, error) {
	var updates []pubmarket.UpdateAvailableInfo

	for _, item := range installed {
		detail, err := s.cat.GetPlugin(item.PluginID)
		if err != nil {
			continue
		}

		info := pubmarket.UpdateAvailableInfo{
			PluginID:       item.PluginID,
			CurrentVersion: item.CurrentVersion,
			LatestVersion:  detail.LatestVersion,
			HasUpdate:      false,
		}

		// Check if latest version is strictly newer than current
		if detail.LatestVersion.Compare(item.CurrentVersion) > 0 {
			info.HasUpdate = true
			for _, v := range detail.Versions {
				if v.Version.Compare(detail.LatestVersion) == 0 {
					info.ArtifactDigest = v.ArtifactDigest
					info.DownloadURL = v.DownloadURL
					break
				}
			}
		}

		updates = append(updates, info)
	}

	return updates, nil
}

// Search queries the discovery engine.
func (s *Service) Search(ctx context.Context, q pubmarket.SearchQuery) (*pubmarket.SearchResult, error) {
	res, err := s.disc.Search(q)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// GetPlugin retrieves detail for a plugin.
func (s *Service) GetPlugin(ctx context.Context, id string) (*pubmarket.PluginDetail, error) {
	d, err := s.cat.GetPlugin(id)
	if err != nil {
		return nil, pubmarket.ErrNotFound
	}
	return d, nil
}

// GetVersions returns all versions for a plugin.
func (s *Service) GetVersions(ctx context.Context, id string) ([]pubmarket.VersionInfo, error) {
	vers, err := s.cat.GetVersions(id)
	if err != nil {
		return nil, pubmarket.ErrNotFound
	}
	return vers, nil
}

// GetFeatured retrieves featured plugins.
func (s *Service) GetFeatured(ctx context.Context, limit int) ([]pubmarket.PluginSummary, error) {
	return s.disc.GetFeatured(limit)
}

// GetTrending retrieves trending plugins.
func (s *Service) GetTrending(ctx context.Context, limit int) ([]pubmarket.PluginSummary, error) {
	return s.disc.GetTrending(limit)
}

// GetCategories returns official marketplace categories.
func (s *Service) GetCategories(ctx context.Context) ([]pubmarket.Category, error) {
	return pubmarket.AllCategories(), nil
}

// GetPublisher returns publisher profile.
func (s *Service) GetPublisher(ctx context.Context, publisherID string) (*pubmarket.PublisherProfile, error) {
	pub, err := s.publishers.GetPublisher(publisherID)
	if err != nil {
		return nil, pubmarket.ErrNotFound
	}
	return pub, nil
}

// SubmitReview records a review and updates ratings.
func (s *Service) SubmitReview(ctx context.Context, review pubmarket.Review) error {
	return s.reviews.SubmitReview(review)
}

// SubmitRating records a star rating.
func (s *Service) SubmitRating(ctx context.Context, pluginID, authorID string, stars int) (*pubmarket.RatingSummary, error) {
	return s.ratings.RecordRating(pluginID, authorID, stars)
}

// GetReviews retrieves reviews for a plugin.
func (s *Service) GetReviews(ctx context.Context, pluginID string) ([]pubmarket.Review, error) {
	return s.reviews.GetReviews(pluginID)
}

// ResolveDependencies computes an installation plan resolving all required dependencies from the catalog.
func (s *Service) ResolveDependencies(
	ctx context.Context,
	pluginID string,
	ver version.SemVer,
) (*pubmarket.DependencyResolutionPlan, error) {
	return s.resolver.Resolve(pluginID, ver)
}

type accountingReadCloser struct {
	io.ReadCloser
	hasher         hash.Hash
	expectedDigest string
	onComplete     func()
	completed      bool
}

func newAccountingReadCloser(rc io.ReadCloser, expectedDigest string, onComplete func()) *accountingReadCloser {
	return &accountingReadCloser{
		ReadCloser:     rc,
		hasher:         sha256.New(),
		expectedDigest: expectedDigest,
		onComplete:     onComplete,
	}
}

func (a *accountingReadCloser) Read(p []byte) (int, error) {
	n, err := a.ReadCloser.Read(p)
	if n > 0 {
		_, _ = a.hasher.Write(p[:n])
	}
	if err == io.EOF && !a.completed {
		actualDigest := hex.EncodeToString(a.hasher.Sum(nil))
		if a.expectedDigest == "" || actualDigest == a.expectedDigest {
			a.completed = true
			if a.onComplete != nil {
				a.onComplete()
			}
		}
	}
	return n, err
}

func (a *accountingReadCloser) Close() error {
	return a.ReadCloser.Close()
}
