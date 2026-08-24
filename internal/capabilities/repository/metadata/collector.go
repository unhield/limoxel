package metadata

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
	"github.com/unhield/limoxel/internal/repository"
)

// Collector coordinates deterministic collection of repository metadata.
type Collector struct {
	discoverer *discovery.Discoverer
}

// New constructs and validates a new immutable Collector instance.
func New(discoverer *discovery.Discoverer) (*Collector, error) {
	if discoverer == nil {
		return nil, ErrNilDiscoverer
	}
	return &Collector{
		discoverer: discoverer,
	}, nil
}

// Discoverer returns the underlying repository Discoverer instance.
func (c *Collector) Discoverer() *discovery.Discoverer {
	if c == nil {
		return nil
	}
	return c.discoverer
}

// Collect compiles a full repository Profile from an existing discovery result.
func (c *Collector) Collect(discResult *discovery.Result) (*Profile, error) {
	if c == nil {
		return nil, ErrNilCollector
	}
	if discResult == nil {
		return nil, ErrNilDiscoveryResult
	}

	root := discResult.Root()
	repoName := filepath.Base(root)
	if discResult.Metadata() != nil && discResult.Metadata().Name() != "" {
		repoName = discResult.Metadata().Name()
	}

	gitDir := filepath.Join(root, ".git")
	info, err := os.Stat(gitDir)
	isGit := err == nil && info.IsDir()

	var (
		owner           string
		currentBranch   string
		defaultBranch   string
		latestCommit    *Commit
		commitStats     *CommitStats
		contributors    []*Contributor
		tags            []*Tag
		releases        []*Release
		historicalStart time.Time
		repositoryAge   time.Duration
		diagnostics     = discResult.Diagnostics()
	)

	if isGit {
		// 1. Extract owner/namespace from .git/config
		owner = c.parseGitConfigOwner(gitDir, &diagnostics)

		// 2. Branch information from discovery metadata
		if discResult.Metadata() != nil {
			currentBranch = discResult.Metadata().CurrentBranch()
			defaultBranch = discResult.Metadata().DefaultBranch()
		}

		// 3. Parse Git reflog history for commits, statistics, and contributors
		latestCommit, commitStats, contributors, historicalStart, repositoryAge = c.parseGitLogs(gitDir, discResult.Metadata(), &diagnostics)

		// 4. Parse Git tags
		tags = c.parseGitTags(gitDir, &diagnostics)

		// 5. Parse local releases from tags
		releases = c.parseLocalReleases(tags)
	}

	profile := NewProfile(
		repoName,
		owner,
		root,
		isGit,
		currentBranch,
		defaultBranch,
		latestCommit,
		commitStats,
		contributors,
		tags,
		releases,
		historicalStart,
		repositoryAge,
		discResult.FileCount(),
		discResult.Metadata().TotalDirectories(),
		discResult.Metadata().TotalBytes(),
		discResult.Languages(),
		discResult.NestedRepositories(),
		diagnostics,
	)

	return profile, nil
}

// CollectRepository executes discovery on an existing Repository instance and compiles its Profile.
func (c *Collector) CollectRepository(repo *repository.Repository) (*Profile, error) {
	if c == nil {
		return nil, ErrNilCollector
	}
	if repo == nil {
		return nil, ErrNilRepository
	}

	discResult, err := c.discoverer.Discover(repo)
	if err != nil {
		return nil, fmt.Errorf("%w: discovery failed: %v", ErrMetadataCollectionFailed, err)
	}

	return c.Collect(discResult)
}

// CollectPath loads, discovers, and compiles a repository Profile for a target filesystem path.
func (c *Collector) CollectPath(path string) (*Profile, error) {
	if c == nil {
		return nil, ErrNilCollector
	}

	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return nil, ErrPathEmpty
	}

	absPath, err := filepath.Abs(filepath.Clean(cleanPath))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMetadataCollectionFailed, err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrPathNotFound, absPath)
		}
		return nil, fmt.Errorf("%w: %v", ErrMetadataCollectionFailed, err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("%w: %s", ErrNotDirectory, absPath)
	}

	discResult, err := c.discoverer.DiscoverPath(absPath)
	if err != nil {
		return nil, fmt.Errorf("%w: discovery failed: %v", ErrMetadataCollectionFailed, err)
	}

	return c.Collect(discResult)
}

func (c *Collector) parseGitConfigOwner(gitDir string, diagnostics *[]*discovery.Diagnostic) string {
	configFile := filepath.Join(gitDir, "config")
	f, err := os.Open(configFile)
	if err != nil {
		return ""
	}
	defer f.Close()

	var inRemoteOrigin bool
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") {
			inRemoteOrigin = strings.EqualFold(line, `[remote "origin"]`)
			continue
		}

		if inRemoteOrigin && strings.HasPrefix(line, "url") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				rawURL := strings.TrimSpace(parts[1])
				return extractOwnerFromURL(rawURL)
			}
		}
	}

	if scanErr := scanner.Err(); scanErr != nil && diagnostics != nil {
		*diagnostics = append(*diagnostics, discovery.NewDiagnostic(
			discovery.SeverityWarning,
			"GIT_CONFIG_SCAN_ERROR",
			scanErr.Error(),
			".git/config",
			false,
		))
	}

	return ""
}

func extractOwnerFromURL(rawURL string) string {
	clean := strings.TrimSpace(rawURL)
	clean = strings.TrimSuffix(clean, ".git")

	// Handle SCP-style: git@github.com:owner/repo
	if strings.Contains(clean, "@") && strings.Contains(clean, ":") {
		parts := strings.SplitN(clean, ":", 2)
		if len(parts) == 2 {
			clean = parts[1]
		}
	} else {
		// Handle URL style: https://github.com/owner/repo
		if idx := strings.Index(clean, "://"); idx != -1 {
			clean = clean[idx+3:]
		}
		// Strip domain
		if slashIdx := strings.Index(clean, "/"); slashIdx != -1 {
			clean = clean[slashIdx+1:]
		}
	}

	clean = filepath.ToSlash(clean)
	segments := strings.Split(clean, "/")
	if len(segments) >= 2 {
		// All segments except repository name
		return strings.Join(segments[:len(segments)-1], "/")
	}

	return ""
}

func (c *Collector) parseGitLogs(
	gitDir string,
	meta *discovery.Metadata,
	diagnostics *[]*discovery.Diagnostic,
) (*Commit, *CommitStats, []*Contributor, time.Time, time.Duration) {
	logFile := filepath.Join(gitDir, "logs", "HEAD")
	f, err := os.Open(logFile)
	if err != nil {
		// Fallback: If reflog is not present, use latest commit from discovery metadata
		if meta != nil && meta.LatestCommit() != "" {
			latest := NewCommit(meta.LatestCommit(), "", "", time.Time{}, "")
			stats := NewCommitStats(1, meta.LatestCommit(), time.Time{}, meta.LatestCommit(), time.Time{})
			return latest, stats, nil, time.Time{}, 0
		}
		return nil, nil, nil, time.Time{}, 0
	}
	defer f.Close()

	type authorKey struct {
		name  string
		email string
	}

	type authorTracker struct {
		count int
		first time.Time
		last  time.Time
	}

	var (
		commits      []*Commit
		authors      = make(map[authorKey]*authorTracker)
		earliestTime time.Time
		latestTime   time.Time
		scanner      = bufio.NewScanner(f)
	)

	for scanner.Scan() {
		line := scanner.Text()
		commit := parseReflogLine(line)
		if commit == nil {
			continue
		}

		commits = append(commits, commit)

		if !commit.timestamp.IsZero() {
			if earliestTime.IsZero() || commit.timestamp.Before(earliestTime) {
				earliestTime = commit.timestamp
			}
			if latestTime.IsZero() || commit.timestamp.After(latestTime) {
				latestTime = commit.timestamp
			}
		}

		if commit.author != "" || commit.email != "" {
			k := authorKey{name: commit.author, email: commit.email}
			tr, exists := authors[k]
			if !exists {
				tr = &authorTracker{
					first: commit.timestamp,
					last:  commit.timestamp,
				}
				authors[k] = tr
			}
			tr.count++
			if !commit.timestamp.IsZero() {
				if tr.first.IsZero() || commit.timestamp.Before(tr.first) {
					tr.first = commit.timestamp
				}
				if tr.last.IsZero() || commit.timestamp.After(tr.last) {
					tr.last = commit.timestamp
				}
			}
		}
	}

	if scanErr := scanner.Err(); scanErr != nil && diagnostics != nil {
		*diagnostics = append(*diagnostics, discovery.NewDiagnostic(
			discovery.SeverityWarning,
			"GIT_LOGS_SCAN_ERROR",
			scanErr.Error(),
			".git/logs/HEAD",
			false,
		))
	}

	if len(commits) == 0 {
		if meta != nil && meta.LatestCommit() != "" {
			latest := NewCommit(meta.LatestCommit(), "", "", time.Time{}, "")
			stats := NewCommitStats(1, meta.LatestCommit(), time.Time{}, meta.LatestCommit(), time.Time{})
			return latest, stats, nil, time.Time{}, 0
		}
		return nil, nil, nil, time.Time{}, 0
	}

	latestCommit := commits[len(commits)-1]
	earliestSHA := commits[0].SHA()
	latestSHA := latestCommit.SHA()

	commitStats := NewCommitStats(len(commits), earliestSHA, earliestTime, latestSHA, latestTime)

	var contributors []*Contributor
	for k, tr := range authors {
		contributors = append(contributors, NewContributor(
			k.name,
			k.email,
			tr.count,
			tr.first,
			tr.last,
		))
	}

	sort.Slice(contributors, func(i, j int) bool {
		if contributors[i].commitCount != contributors[j].commitCount {
			return contributors[i].commitCount > contributors[j].commitCount
		}
		if contributors[i].name != contributors[j].name {
			return contributors[i].name < contributors[j].name
		}
		return contributors[i].email < contributors[j].email
	})

	var repoAge time.Duration
	if !earliestTime.IsZero() && !latestTime.IsZero() && latestTime.After(earliestTime) {
		repoAge = latestTime.Sub(earliestTime)
	}

	return latestCommit, commitStats, contributors, earliestTime, repoAge
}

func parseReflogLine(line string) *Commit {
	clean := strings.TrimSpace(line)
	if clean == "" {
		return nil
	}

	tabIdx := strings.Index(clean, "\t")
	var msg string
	header := clean
	if tabIdx != -1 {
		header = clean[:tabIdx]
		msg = strings.TrimSpace(clean[tabIdx+1:])
	}

	fields := strings.Fields(header)
	if len(fields) < 2 {
		return nil
	}

	newSHA := fields[1]

	// Extract email inside <...>
	openIdx := strings.Index(header, "<")
	closeIdx := strings.Index(header, ">")

	var author, email string
	var commitTime time.Time

	if openIdx != -1 && closeIdx != -1 && closeIdx > openIdx {
		email = header[openIdx+1 : closeIdx]
		// Author is between newSHA (fields[1]) and openIdx
		authorPart := strings.TrimSpace(header[len(fields[0])+len(fields[1])+2 : openIdx])
		author = authorPart

		// Timestamp and TZ follow after closeIdx
		afterClose := strings.TrimSpace(header[closeIdx+1:])
		timeParts := strings.Fields(afterClose)
		if len(timeParts) >= 1 {
			if sec, err := strconv.ParseInt(timeParts[0], 10, 64); err == nil {
				commitTime = time.Unix(sec, 0).UTC()
			}
		}
	}

	return NewCommit(newSHA, author, email, commitTime, msg)
}

func (c *Collector) parseGitTags(gitDir string, diagnostics *[]*discovery.Diagnostic) []*Tag {
	tagMap := make(map[string]*Tag)

	// 1. Scan loose tags in .git/refs/tags
	tagsDir := filepath.Join(gitDir, "refs", "tags")
	if info, err := os.Stat(tagsDir); err == nil && info.IsDir() {
		_ = filepath.WalkDir(tagsDir, func(path string, dEntry fs.DirEntry, err error) error {
			if err != nil || dEntry.IsDir() {
				return nil
			}

			rel, relErr := filepath.Rel(tagsDir, path)
			if relErr != nil {
				return nil
			}
			tagName := filepath.ToSlash(rel)

			if bytes, readErr := os.ReadFile(path); readErr == nil {
				sha := strings.TrimSpace(string(bytes))
				tagMap[tagName] = NewTag(tagName, sha, "lightweight", time.Time{})
			}
			return nil
		})
	}

	// 2. Scan packed-refs for tags
	packedFile := filepath.Join(gitDir, "packed-refs")
	if f, err := os.Open(packedFile); err == nil {
		defer f.Close()
		scanner := bufio.NewScanner(f)
		var lastTag string

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "#") || line == "" {
				continue
			}

			// Handle peeled annotated tags: ^<commit_sha>
			if strings.HasPrefix(line, "^") {
				peeledSHA := strings.TrimPrefix(line, "^")
				if lastTag != "" {
					if existing, ok := tagMap[lastTag]; ok {
						tagMap[lastTag] = NewTag(existing.Name(), peeledSHA, "annotated", existing.Timestamp())
					}
				}
				continue
			}

			parts := strings.Fields(line)
			if len(parts) >= 2 && strings.HasPrefix(parts[1], "refs/tags/") {
				tagName := strings.TrimPrefix(parts[1], "refs/tags/")
				sha := parts[0]
				tagMap[tagName] = NewTag(tagName, sha, "lightweight", time.Time{})
				lastTag = tagName
			} else {
				lastTag = ""
			}
		}

		if scanErr := scanner.Err(); scanErr != nil && diagnostics != nil {
			*diagnostics = append(*diagnostics, discovery.NewDiagnostic(
				discovery.SeverityWarning,
				"GIT_TAGS_PACKED_REFS_SCAN_ERROR",
				scanErr.Error(),
				".git/packed-refs",
				false,
			))
		}
	}

	result := make([]*Tag, 0, len(tagMap))
	for _, t := range tagMap {
		result = append(result, t)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].name < result[j].name
	})

	return result
}

func (c *Collector) parseLocalReleases(tags []*Tag) []*Release {
	if len(tags) == 0 {
		return nil
	}

	var releases []*Release

	for _, t := range tags {
		name := t.Name()
		lower := strings.ToLower(name)

		// Identify version / release tags: v1.0.0, 1.0.0, release-1.0, etc.
		if strings.HasPrefix(lower, "v") || strings.HasPrefix(lower, "release") || (len(name) > 0 && name[0] >= '0' && name[0] <= '9') {
			isPrerelease := strings.Contains(lower, "alpha") ||
				strings.Contains(lower, "beta") ||
				strings.Contains(lower, "rc") ||
				strings.Contains(lower, "preview") ||
				strings.Contains(lower, "dev")

			releaseName := strings.TrimPrefix(name, "release-")
			releaseName = strings.TrimPrefix(releaseName, "release/")

			releases = append(releases, NewRelease(
				releaseName,
				t.Name(),
				t.CommitSHA(),
				isPrerelease,
				t.Timestamp(),
			))
		}
	}

	sort.Slice(releases, func(i, j int) bool {
		if !releases[i].publishedAt.Equal(releases[j].publishedAt) {
			return releases[i].publishedAt.After(releases[j].publishedAt)
		}
		return releases[i].tagName < releases[j].tagName
	})

	return releases
}
