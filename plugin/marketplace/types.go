package marketplace

import (
	"errors"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

// Common marketplace errors.
var (
	ErrNotFound              = errors.New("marketplace: resource not found")
	ErrInvalidInput          = errors.New("marketplace: invalid input")
	ErrVersionNotFound       = errors.New("marketplace: version not found")
	ErrAlreadyExists         = errors.New("marketplace: resource already exists")
	ErrUnauthorizedPublisher = errors.New("marketplace: publisher not authorized for this plugin")
	ErrDependencyMissing     = errors.New("marketplace: required dependency missing in catalog")
	ErrDependencyConflict    = errors.New("marketplace: dependency version conflict")
	ErrCircularDependency    = errors.New("marketplace: circular dependency detected")
)

// PublicationStatus describes the review and publication state of a plugin.
type PublicationStatus string

const (
	// StatusSubmitted indicates the plugin package has been uploaded and queued.
	StatusSubmitted PublicationStatus = "SUBMITTED"
	// StatusValidating indicates the package is undergoing automated validation.
	StatusValidating PublicationStatus = "VALIDATING"
	// StatusApproved indicates the package passed validation and review.
	StatusApproved PublicationStatus = "APPROVED"
	// StatusPublished indicates the package is discoverable and downloadable.
	StatusPublished PublicationStatus = "PUBLISHED"
	// StatusRejected indicates publication was rejected due to validation failure.
	StatusRejected PublicationStatus = "REJECTED"
	// StatusSuspended indicates the package was withdrawn from distribution.
	StatusSuspended PublicationStatus = "SUSPENDED"
)

// IsDiscoverable reports whether the status allows discovery in the catalog.
func (s PublicationStatus) IsDiscoverable() bool {
	return s == StatusPublished
}

// Category represents a classification category in the marketplace.
type Category string

const (
	CategoryAll           Category = "all"
	CategoryAnalysis      Category = "analysis"
	CategorySecurity      Category = "security"
	CategoryVisualization Category = "visualization"
	CategoryIntegration   Category = "integration"
	CategoryLanguage      Category = "language"
	CategoryTools         Category = "tools"
	CategoryUtility       Category = "utility"
)

// AllCategories returns the list of official marketplace categories.
func AllCategories() []Category {
	return []Category{
		CategoryAnalysis,
		CategorySecurity,
		CategoryVisualization,
		CategoryIntegration,
		CategoryLanguage,
		CategoryTools,
		CategoryUtility,
	}
}

// PublisherProfile represents public marketplace identity information for a publisher.
type PublisherProfile struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"display_name"`
	Website     string    `json:"website,omitempty"`
	Email       string    `json:"email,omitempty"`
	KeyID       string    `json:"key_id,omitempty"`
	Verified    bool      `json:"verified"`
	CreatedAt   time.Time `json:"created_at"`
	PluginCount int       `json:"plugin_count"`
}

// VersionInfo captures metadata and artifact pointers for a published version.
type VersionInfo struct {
	Version        version.SemVer    `json:"version"`
	ArtifactDigest string            `json:"artifact_digest"` // SHA-256 hex digest
	ArtifactSize   int64             `json:"artifact_size"`   // Package size in bytes
	DownloadURL    string            `json:"download_url"`
	MinHostVersion *version.SemVer   `json:"min_host_version,omitempty"`
	MaxHostVersion *version.SemVer   `json:"max_host_version,omitempty"`
	PublishedAt    time.Time         `json:"published_at"`
	Downloads      int64             `json:"downloads"`
	Changelog      string            `json:"changelog,omitempty"`
	Status         PublicationStatus `json:"status"`
}

// PluginSummary provides lightweight metadata for catalog browsing and search.
type PluginSummary struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Description    string            `json:"description"`
	PublisherID    string            `json:"publisher_id"`
	PublisherName  string            `json:"publisher_name"`
	PublisherValid bool              `json:"publisher_valid"`
	LatestVersion  version.SemVer    `json:"latest_version"`
	Categories     []Category        `json:"categories"`
	Tags           []string          `json:"tags,omitempty"`
	TotalDownloads int64             `json:"total_downloads"`
	AverageRating  float64           `json:"average_rating"`
	RatingCount    int               `json:"rating_count"`
	Featured       bool              `json:"featured"`
	Status         PublicationStatus `json:"status"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// PluginDetail provides complete metadata, versions, and dependencies.
type PluginDetail struct {
	PluginSummary
	License      string              `json:"license,omitempty"`
	Homepage     string              `json:"homepage,omitempty"`
	Repository   string              `json:"repository,omitempty"`
	Versions     []VersionInfo       `json:"versions"`
	Dependencies []DependencySummary `json:"dependencies,omitempty"`
	Capabilities []string            `json:"capabilities,omitempty"`
	Permissions  []string            `json:"permissions,omitempty"`
	Changelog    []ChangelogEntry    `json:"changelog,omitempty"`
	Publisher    PublisherProfile    `json:"publisher"`
	Reviews      []Review            `json:"reviews,omitempty"`
	Rating       RatingSummary       `json:"rating"`
}

// DependencySummary describes an external plugin dependency.
type DependencySummary struct {
	ID         string `json:"id"`
	Constraint string `json:"constraint"`
	Optional   bool   `json:"optional"`
}

// RatingSummary aggregates consumer ratings for a plugin.
type RatingSummary struct {
	Average float64     `json:"average"` // 1.0 to 5.0
	Count   int         `json:"count"`   // Total number of ratings
	Stars   map[int]int `json:"stars"`   // Star breakdown: 1 -> count, 2 -> count, ...
}

// Review represents a community review submitted for a published plugin.
type Review struct {
	ID        string    `json:"id"`
	PluginID  string    `json:"plugin_id"`
	Version   string    `json:"version"`
	AuthorID  string    `json:"author_id"`
	Rating    int       `json:"rating"` // 1 to 5
	Title     string    `json:"title"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ChangelogEntry records release notes for a specific version.
type ChangelogEntry struct {
	Version version.SemVer `json:"version"`
	Date    time.Time      `json:"date"`
	Summary string         `json:"summary"`
	Changes []string       `json:"changes,omitempty"`
}

// SortBy specifies the sorting criteria for discovery and search queries.
type SortBy string

const (
	SortByRelevance SortBy = "relevance"
	SortByDownloads SortBy = "downloads"
	SortByRating    SortBy = "rating"
	SortByUpdated   SortBy = "updated"
	SortByName      SortBy = "name"
)

// SortOrder specifies ascending or descending order.
type SortOrder string

const (
	OrderAsc  SortOrder = "asc"
	OrderDesc SortOrder = "desc"
)

// SearchQuery configures catalog search and filtering parameters.
type SearchQuery struct {
	Keyword   string    `json:"keyword,omitempty"`
	Category  Category  `json:"category,omitempty"`
	Publisher string    `json:"publisher,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
	Featured  *bool     `json:"featured,omitempty"`
	MinRating float64   `json:"min_rating,omitempty"`
	Sort      SortBy    `json:"sort,omitempty"`
	Order     SortOrder `json:"order,omitempty"`
	Offset    int       `json:"offset,omitempty"`
	Limit     int       `json:"limit,omitempty"`
}

// SearchResult returns paginated results from a catalog search query.
type SearchResult struct {
	Items  []PluginSummary `json:"items"`
	Total  int             `json:"total"`
	Offset int             `json:"offset"`
	Limit  int             `json:"limit"`
}

// UpdateCheckItem specifies an installed plugin version to check for updates.
type UpdateCheckItem struct {
	PluginID       string         `json:"plugin_id"`
	CurrentVersion version.SemVer `json:"current_version"`
}

// UpdateAvailableInfo describes an available newer version.
type UpdateAvailableInfo struct {
	PluginID       string         `json:"plugin_id"`
	CurrentVersion version.SemVer `json:"current_version"`
	LatestVersion  version.SemVer `json:"latest_version"`
	ArtifactDigest string         `json:"artifact_digest"`
	DownloadURL    string         `json:"download_url"`
	Changelog      string         `json:"changelog,omitempty"`
	HasUpdate      bool           `json:"has_update"`
}

// ResolvedDependency describes a dependency matched and selected during resolution.
type ResolvedDependency struct {
	ID             string         `json:"id"`
	Version        version.SemVer `json:"version"`
	ArtifactDigest string         `json:"artifact_digest"`
	DownloadURL    string         `json:"download_url"`
	Optional       bool           `json:"optional"`
}

// DependencyResolutionPlan contains the ordered installation plan for a plugin and its dependencies.
type DependencyResolutionPlan struct {
	TargetPlugin  string                        `json:"target_plugin"`
	TargetVersion version.SemVer                `json:"target_version"`
	InstallOrder  []ResolvedDependency          `json:"install_order"`
	Dependencies  map[string]ResolvedDependency `json:"dependencies"`
	ResolvedAt    time.Time                     `json:"resolved_at"`
}
