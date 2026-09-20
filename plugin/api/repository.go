package api

import "context"

// RepositoryAPI defines the developer-facing interface for repository workspace queries.
type RepositoryAPI interface {
	// WorkspacePath returns the absolute filesystem path of the repository workspace.
	WorkspacePath() string

	// Info retrieves the operational snapshot of the repository.
	Info(ctx context.Context) (*RepositoryInfo, error)

	// Metadata retrieves descriptive VCS and manifest metadata.
	Metadata(ctx context.Context) (*RepositoryMetadata, error)

	// Statistics returns quantitative metrics across files, symbols, and dependencies.
	Statistics(ctx context.Context) (*RepositoryStatistics, error)

	// ListFiles enumerates workspace file paths matching an optional relative prefix.
	ListFiles(ctx context.Context, prefix string) ([]string, error)

	// GetFile retrieves detailed metadata for a specific relative file path.
	GetFile(ctx context.Context, relPath string) (*FileInfo, error)
}
