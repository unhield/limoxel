package contracts

import (
	"context"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/sdk/lifecycle"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

// RepositoryState represents the operational status of a workspace in the SDK contract.
type RepositoryState string

const (
	// StateUninitialized indicates no repository is currently opened.
	StateUninitialized RepositoryState = "UNINITIALIZED"

	// StateDiscovered indicates the repository directory and manifest have been identified.
	StateDiscovered RepositoryState = "DISCOVERED"

	// StateIndexing indicates analysis and AST indexing are in progress.
	StateIndexing RepositoryState = "INDEXING"

	// StateReady indicates the repository is loaded, indexed, and ready for queries.
	StateReady RepositoryState = "READY"

	// StateClosed indicates the repository has been closed and resources released.
	StateClosed RepositoryState = "CLOSED"

	// StateError indicates a critical failure occurred during repository lifecycle operations.
	StateError RepositoryState = "ERROR"
)

// String returns the string representation of RepositoryState.
func (s RepositoryState) String() string {
	return string(s)
}

// RepositoryInfo represents a public snapshot of repository identity, VCS, languages, and capabilities.
type RepositoryInfo struct {
	Name          string          `json:"name"`
	Owner         string          `json:"owner,omitempty"`
	RootPath      string          `json:"root_path"`
	DefaultBranch string          `json:"default_branch,omitempty"`
	CurrentBranch string          `json:"current_branch,omitempty"`
	IsGit         bool            `json:"is_git"`
	State         RepositoryState `json:"state"`
	Languages     []string        `json:"languages,omitempty"`
	Capabilities  []string        `json:"capabilities,omitempty"`
}

// WorkspaceInfo represents public information about the active working context.
type WorkspaceInfo struct {
	ID            string    `json:"id"`
	RootPath      string    `json:"root_path"`
	Name          string    `json:"name"`
	IsActive      bool      `json:"is_active"`
	Created       time.Time `json:"created"`
	ResourceCount int       `json:"resource_count"`
}

// RepositoryMetadata represents descriptive VCS and manifest attributes of the repository.
type RepositoryMetadata struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Version     string            `json:"version,omitempty"`
	License     string            `json:"license,omitempty"`
	VCS         string            `json:"vcs,omitempty"`
	CommitHash  string            `json:"commit_hash,omitempty"`
	CommitDate  time.Time         `json:"commit_date,omitempty"`
	Tag         string            `json:"tag,omitempty"`
	Authors     []string          `json:"authors,omitempty"`
	Properties  map[string]string `json:"properties,omitempty"`
}

// RepositoryStatistics represents quantitative measurements of the repository.
type RepositoryStatistics struct {
	TotalFiles         int `json:"total_files"`
	TotalDirectories   int `json:"total_directories"`
	TotalPackages      int `json:"total_packages"`
	TotalSymbols       int `json:"total_symbols"`
	TotalRelationships int `json:"total_relationships"`
	TotalDependencies  int `json:"total_dependencies"`
	TotalDocs          int `json:"total_docs"`
	TotalConfigs       int `json:"total_configs"`
}

// RepositoryManagementContract defines the public contract for repository session lifecycle and metadata.
type RepositoryManagementContract interface {
	Contract
	Open(ctx context.Context, path string) (*RepositoryInfo, error)
	Close(ctx context.Context) error
	Info(ctx context.Context) (*RepositoryInfo, error)
	State() RepositoryState
	Metadata(ctx context.Context) (*RepositoryMetadata, error)
	Statistics(ctx context.Context) (*RepositoryStatistics, error)
	Reload(ctx context.Context) error
}

// DefaultRepositoryContractMetadata returns default contract descriptor for Repository Management.
func DefaultRepositoryContractMetadata() BaseContract {
	return NewBaseContract(
		"RepositoryManagementContract",
		lifecycle.CapabilityRepository,
		version.SemVer{Major: 1, Minor: 0, Patch: 0},
		lifecycle.StateSupported,
		"Provides public repository lifecycle, metadata, and quantitative statistics.",
	)
}
