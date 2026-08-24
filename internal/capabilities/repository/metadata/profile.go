package metadata

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
)

// Profile represents an immutable, comprehensive repository metadata profile.
type Profile struct {
	name             string
	owner            string
	root             string
	isGit            bool
	currentBranch    string
	defaultBranch    string
	latestCommit     *Commit
	commitStats      *CommitStats
	contributors     []*Contributor
	tags             []*Tag
	releases         []*Release
	historicalStart  time.Time
	repositoryAge    time.Duration
	totalFiles       int
	totalDirectories int
	totalBytes       int64
	languages        []*discovery.LanguageDistribution
	nestedRepos      []string
	diagnostics      []*discovery.Diagnostic
}

// NewProfile constructs an immutable Profile record with defensively copied collections.
func NewProfile(
	name string,
	owner string,
	root string,
	isGit bool,
	currentBranch string,
	defaultBranch string,
	latestCommit *Commit,
	commitStats *CommitStats,
	contributors []*Contributor,
	tags []*Tag,
	releases []*Release,
	historicalStart time.Time,
	repositoryAge time.Duration,
	totalFiles int,
	totalDirectories int,
	totalBytes int64,
	languages []*discovery.LanguageDistribution,
	nestedRepos []string,
	diagnostics []*discovery.Diagnostic,
) *Profile {
	// Defensive copy & deterministic sort for contributors: commitCount desc, name asc, email asc
	contribs := make([]*Contributor, len(contributors))
	copy(contribs, contributors)
	sort.Slice(contribs, func(i, j int) bool {
		if contribs[i].commitCount != contribs[j].commitCount {
			return contribs[i].commitCount > contribs[j].commitCount
		}
		if contribs[i].name != contribs[j].name {
			return contribs[i].name < contribs[j].name
		}
		return contribs[i].email < contribs[j].email
	})

	// Defensive copy & deterministic sort for tags: name asc
	tagList := make([]*Tag, len(tags))
	copy(tagList, tags)
	sort.Slice(tagList, func(i, j int) bool {
		return tagList[i].name < tagList[j].name
	})

	// Defensive copy & deterministic sort for releases: publishedAt desc, tagName asc
	releaseList := make([]*Release, len(releases))
	copy(releaseList, releases)
	sort.Slice(releaseList, func(i, j int) bool {
		if !releaseList[i].publishedAt.Equal(releaseList[j].publishedAt) {
			return releaseList[i].publishedAt.After(releaseList[j].publishedAt)
		}
		return releaseList[i].tagName < releaseList[j].tagName
	})

	// Defensive copy for languages
	langList := make([]*discovery.LanguageDistribution, len(languages))
	copy(langList, languages)

	// Defensive copy & sort for nested repos
	nRepos := make([]string, len(nestedRepos))
	copy(nRepos, nestedRepos)
	sort.Strings(nRepos)

	// Defensive copy for diagnostics
	diagList := make([]*discovery.Diagnostic, len(diagnostics))
	copy(diagList, diagnostics)

	return &Profile{
		name:             strings.TrimSpace(name),
		owner:            strings.TrimSpace(owner),
		root:             filepath.Clean(root),
		isGit:            isGit,
		currentBranch:    strings.TrimSpace(currentBranch),
		defaultBranch:    strings.TrimSpace(defaultBranch),
		latestCommit:     latestCommit,
		commitStats:      commitStats,
		contributors:     contribs,
		tags:             tagList,
		releases:         releaseList,
		historicalStart:  historicalStart,
		repositoryAge:    repositoryAge,
		totalFiles:       totalFiles,
		totalDirectories: totalDirectories,
		totalBytes:       totalBytes,
		languages:        langList,
		nestedRepos:      nRepos,
		diagnostics:      diagList,
	}
}

// Name returns the repository identifier name.
func (p *Profile) Name() string {
	if p == nil {
		return ""
	}
	return p.name
}

// Owner returns the repository owner or namespace, if locally available.
func (p *Profile) Owner() string {
	if p == nil {
		return ""
	}
	return p.owner
}

// Root returns the cleaned absolute path to the repository root directory.
func (p *Profile) Root() string {
	if p == nil {
		return ""
	}
	return p.root
}

// IsGit reports whether local source control (Git) was detected.
func (p *Profile) IsGit() bool {
	if p == nil {
		return false
	}
	return p.isGit
}

// CurrentBranch returns the currently active Git branch name, or empty if unavailable.
func (p *Profile) CurrentBranch() string {
	if p == nil {
		return ""
	}
	return p.currentBranch
}

// DefaultBranch returns the default branch name (e.g. "main"), or empty if unavailable.
func (p *Profile) DefaultBranch() string {
	if p == nil {
		return ""
	}
	return p.defaultBranch
}

// LatestCommit returns the latest commit descriptor, or nil if unavailable.
func (p *Profile) LatestCommit() *Commit {
	if p == nil {
		return nil
	}
	return p.latestCommit
}

// CommitStats returns aggregated commit metrics, or nil if unavailable.
func (p *Profile) CommitStats() *CommitStats {
	if p == nil {
		return nil
	}
	return p.commitStats
}

// Contributors returns a defensive copy of contributors in deterministic sorted order.
func (p *Profile) Contributors() []*Contributor {
	if p == nil || len(p.contributors) == 0 {
		return nil
	}
	cloned := make([]*Contributor, len(p.contributors))
	copy(cloned, p.contributors)
	return cloned
}

// Tags returns a defensive copy of repository tags in deterministic sorted order.
func (p *Profile) Tags() []*Tag {
	if p == nil || len(p.tags) == 0 {
		return nil
	}
	cloned := make([]*Tag, len(p.tags))
	copy(cloned, p.tags)
	return cloned
}

// Releases returns a defensive copy of local release records in deterministic sorted order.
func (p *Profile) Releases() []*Release {
	if p == nil || len(p.releases) == 0 {
		return nil
	}
	cloned := make([]*Release, len(p.releases))
	copy(cloned, p.releases)
	return cloned
}

// HistoricalStart returns the timestamp of the earliest available commit, or zero time if unavailable.
func (p *Profile) HistoricalStart() time.Time {
	if p == nil {
		return time.Time{}
	}
	return p.historicalStart
}

// RepositoryAge returns the duration between the earliest and latest available commits.
func (p *Profile) RepositoryAge() time.Duration {
	if p == nil {
		return 0
	}
	return p.repositoryAge
}

// TotalFiles returns the count of discovered regular files.
func (p *Profile) TotalFiles() int {
	if p == nil {
		return 0
	}
	return p.totalFiles
}

// TotalDirectories returns the count of traversed directories.
func (p *Profile) TotalDirectories() int {
	if p == nil {
		return 0
	}
	return p.totalDirectories
}

// TotalBytes returns the sum of file sizes in bytes.
func (p *Profile) TotalBytes() int64 {
	if p == nil {
		return 0
	}
	return p.totalBytes
}

// Languages returns a defensive copy of language distributions.
func (p *Profile) Languages() []*discovery.LanguageDistribution {
	if p == nil || len(p.languages) == 0 {
		return nil
	}
	cloned := make([]*discovery.LanguageDistribution, len(p.languages))
	copy(cloned, p.languages)
	return cloned
}

// NestedRepositories returns a defensive copy of relative paths to nested repositories.
func (p *Profile) NestedRepositories() []string {
	if p == nil || len(p.nestedRepos) == 0 {
		return nil
	}
	cloned := make([]string, len(p.nestedRepos))
	copy(cloned, p.nestedRepos)
	return cloned
}

// Diagnostics returns a defensive copy of metadata diagnostics.
func (p *Profile) Diagnostics() []*discovery.Diagnostic {
	if p == nil || len(p.diagnostics) == 0 {
		return nil
	}
	cloned := make([]*discovery.Diagnostic, len(p.diagnostics))
	copy(cloned, p.diagnostics)
	return cloned
}

// String returns a human-readable summary of the Profile.
func (p *Profile) String() string {
	if p == nil {
		return ""
	}
	return fmt.Sprintf("Profile<%s>[owner=%s, files=%d, bytes=%d, isGit=%t, branch=%s]", p.name, p.owner, p.totalFiles, p.totalBytes, p.isGit, p.currentBranch)
}
