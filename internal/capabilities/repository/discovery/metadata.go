package discovery

import (
	"fmt"
	"sort"
)

// Metadata represents deterministic repository-level metadata collected during discovery.
type Metadata struct {
	name               string
	root               string
	isGit              bool
	currentBranch      string
	defaultBranch      string
	latestCommit       string
	totalFiles         int
	totalDirectories   int
	totalBytes         int64
	nestedRepositories []string
}

// NewMetadata creates an immutable Metadata record.
func NewMetadata(
	name string,
	root string,
	isGit bool,
	currentBranch string,
	defaultBranch string,
	latestCommit string,
	totalFiles int,
	totalDirectories int,
	totalBytes int64,
	nestedRepos []string,
) *Metadata {
	repos := make([]string, len(nestedRepos))
	copy(repos, nestedRepos)
	sort.Strings(repos)

	return &Metadata{
		name:               name,
		root:               root,
		isGit:              isGit,
		currentBranch:      currentBranch,
		defaultBranch:      defaultBranch,
		latestCommit:       latestCommit,
		totalFiles:         totalFiles,
		totalDirectories:   totalDirectories,
		totalBytes:         totalBytes,
		nestedRepositories: repos,
	}
}

// Name returns the repository identity name.
func (m *Metadata) Name() string {
	if m == nil {
		return ""
	}
	return m.name
}

// Root returns the cleaned absolute root path of the repository.
func (m *Metadata) Root() string {
	if m == nil {
		return ""
	}
	return m.root
}

// IsGit reports whether the repository is a Git repository.
func (m *Metadata) IsGit() bool {
	if m == nil {
		return false
	}
	return m.isGit
}

// CurrentBranch returns the currently active Git branch, or empty if unavailable.
func (m *Metadata) CurrentBranch() string {
	if m == nil {
		return ""
	}
	return m.currentBranch
}

// DefaultBranch returns the repository default branch, or empty if unavailable.
func (m *Metadata) DefaultBranch() string {
	if m == nil {
		return ""
	}
	return m.defaultBranch
}

// LatestCommit returns the latest commit hash (e.g. SHA-1), or empty if unavailable.
func (m *Metadata) LatestCommit() string {
	if m == nil {
		return ""
	}
	return m.latestCommit
}

// TotalFiles returns the total number of regular discovered files.
func (m *Metadata) TotalFiles() int {
	if m == nil {
		return 0
	}
	return m.totalFiles
}

// TotalDirectories returns the total number of traversed directories.
func (m *Metadata) TotalDirectories() int {
	if m == nil {
		return 0
	}
	return m.totalDirectories
}

// TotalBytes returns the sum of file sizes in bytes across all discovered files.
func (m *Metadata) TotalBytes() int64 {
	if m == nil {
		return 0
	}
	return m.totalBytes
}

// NestedRepositories returns a defensive copy of relative paths to nested repositories.
func (m *Metadata) NestedRepositories() []string {
	if m == nil || len(m.nestedRepositories) == 0 {
		return nil
	}
	cloned := make([]string, len(m.nestedRepositories))
	copy(cloned, m.nestedRepositories)
	return cloned
}

// String returns a human-readable representation of the Metadata.
func (m *Metadata) String() string {
	if m == nil {
		return ""
	}
	return fmt.Sprintf("Metadata<%s>[files=%d, bytes=%d, isGit=%t, branch=%s]", m.name, m.totalFiles, m.totalBytes, m.isGit, m.currentBranch)
}
