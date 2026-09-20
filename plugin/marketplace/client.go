package marketplace

import (
	"context"

	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

// Client defines the public developer and tooling interface for interacting with the marketplace.
type Client interface {
	// Search queries published plugins using keywords, category filters, and sorting.
	Search(ctx context.Context, query SearchQuery) (*SearchResult, error)

	// GetPlugin retrieves complete detail for a specific plugin by identifier.
	GetPlugin(ctx context.Context, pluginID string) (*PluginDetail, error)

	// GetVersions returns all published version entries for a plugin.
	GetVersions(ctx context.Context, pluginID string) ([]VersionInfo, error)

	// GetFeatured returns curated featured plugins.
	GetFeatured(ctx context.Context, limit int) ([]PluginSummary, error)

	// GetTrending returns high-velocity trending plugins.
	GetTrending(ctx context.Context, limit int) ([]PluginSummary, error)

	// GetCategories returns available classification categories.
	GetCategories(ctx context.Context) ([]Category, error)

	// GetPublisher retrieves public profile details for a publisher.
	GetPublisher(ctx context.Context, publisherID string) (*PublisherProfile, error)

	// CheckUpdates compares currently installed versions against catalog releases.
	CheckUpdates(ctx context.Context, installed []UpdateCheckItem) ([]UpdateAvailableInfo, error)

	// DownloadArtifact downloads the package artifact for a plugin version to destination.
	DownloadArtifact(ctx context.Context, pluginID string, ver version.SemVer, destPath string) (*VersionInfo, error)

	// SubmitReview posts a user review for a specific plugin version.
	SubmitReview(ctx context.Context, review Review) error

	// SubmitRating records a user rating (1-5 stars) and returns updated aggregate metrics.
	SubmitRating(ctx context.Context, pluginID, authorID string, stars int) (*RatingSummary, error)

	// ResolveDependencies computes an installation plan resolving all required dependencies from the catalog.
	ResolveDependencies(ctx context.Context, pluginID string, ver version.SemVer) (*DependencyResolutionPlan, error)
}
