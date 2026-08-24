package metadata

import (
	"fmt"
	"strings"
	"time"
)

// Commit represents an immutable historical commit record.
type Commit struct {
	sha       string
	author    string
	email     string
	timestamp time.Time
	message   string
}

// NewCommit creates a new immutable Commit record.
func NewCommit(sha, author, email string, timestamp time.Time, message string) *Commit {
	return &Commit{
		sha:       strings.TrimSpace(sha),
		author:    strings.TrimSpace(author),
		email:     strings.TrimSpace(email),
		timestamp: timestamp,
		message:   strings.TrimSpace(message),
	}
}

// SHA returns the commit SHA hash identifier.
func (c *Commit) SHA() string {
	if c == nil {
		return ""
	}
	return c.sha
}

// Author returns the author's name.
func (c *Commit) Author() string {
	if c == nil {
		return ""
	}
	return c.author
}

// Email returns the author's email address.
func (c *Commit) Email() string {
	if c == nil {
		return ""
	}
	return c.email
}

// Timestamp returns the commit creation timestamp.
func (c *Commit) Timestamp() time.Time {
	if c == nil {
		return time.Time{}
	}
	return c.timestamp
}

// Message returns the commit message.
func (c *Commit) Message() string {
	if c == nil {
		return ""
	}
	return c.message
}

// String returns a human-readable representation of the Commit.
func (c *Commit) String() string {
	if c == nil {
		return ""
	}
	return fmt.Sprintf("Commit<%s>[author=%s, date=%s]", c.sha, c.author, c.timestamp.Format(time.RFC3339))
}

// CommitStats represents aggregated commit history metrics.
type CommitStats struct {
	totalCommits int
	earliestSHA  string
	earliestTime time.Time
	latestSHA    string
	latestTime   time.Time
	timeRange    time.Duration
}

// NewCommitStats creates a new immutable CommitStats record.
func NewCommitStats(totalCommits int, earliestSHA string, earliestTime time.Time, latestSHA string, latestTime time.Time) *CommitStats {
	var duration time.Duration
	if !earliestTime.IsZero() && !latestTime.IsZero() && latestTime.After(earliestTime) {
		duration = latestTime.Sub(earliestTime)
	}

	return &CommitStats{
		totalCommits: totalCommits,
		earliestSHA:  strings.TrimSpace(earliestSHA),
		earliestTime: earliestTime,
		latestSHA:    strings.TrimSpace(latestSHA),
		latestTime:   latestTime,
		timeRange:    duration,
	}
}

// TotalCommits returns the count of available local commits.
func (cs *CommitStats) TotalCommits() int {
	if cs == nil {
		return 0
	}
	return cs.totalCommits
}

// EarliestSHA returns the SHA identifier of the earliest available commit.
func (cs *CommitStats) EarliestSHA() string {
	if cs == nil {
		return ""
	}
	return cs.earliestSHA
}

// EarliestTime returns the timestamp of the earliest available commit.
func (cs *CommitStats) EarliestTime() time.Time {
	if cs == nil {
		return time.Time{}
	}
	return cs.earliestTime
}

// LatestSHA returns the SHA identifier of the latest commit.
func (cs *CommitStats) LatestSHA() string {
	if cs == nil {
		return ""
	}
	return cs.latestSHA
}

// LatestTime returns the timestamp of the latest commit.
func (cs *CommitStats) LatestTime() time.Time {
	if cs == nil {
		return time.Time{}
	}
	return cs.latestTime
}

// TimeRange returns the duration between the earliest and latest available commits.
func (cs *CommitStats) TimeRange() time.Duration {
	if cs == nil {
		return 0
	}
	return cs.timeRange
}

// Contributor represents an individual author identified in repository history.
type Contributor struct {
	name               string
	email              string
	commitCount        int
	firstContribution  time.Time
	latestContribution time.Time
}

// NewContributor creates a new immutable Contributor record.
func NewContributor(name, email string, commitCount int, firstContribution, latestContribution time.Time) *Contributor {
	return &Contributor{
		name:               strings.TrimSpace(name),
		email:              strings.TrimSpace(email),
		commitCount:        commitCount,
		firstContribution:  firstContribution,
		latestContribution: latestContribution,
	}
}

// Name returns the contributor's name.
func (c *Contributor) Name() string {
	if c == nil {
		return ""
	}
	return c.name
}

// Email returns the contributor's email address.
func (c *Contributor) Email() string {
	if c == nil {
		return ""
	}
	return c.email
}

// CommitCount returns the number of commits authored by this contributor.
func (c *Contributor) CommitCount() int {
	if c == nil {
		return 0
	}
	return c.commitCount
}

// FirstContribution returns the timestamp of the contributor's first commit.
func (c *Contributor) FirstContribution() time.Time {
	if c == nil {
		return time.Time{}
	}
	return c.firstContribution
}

// LatestContribution returns the timestamp of the contributor's latest commit.
func (c *Contributor) LatestContribution() time.Time {
	if c == nil {
		return time.Time{}
	}
	return c.latestContribution
}

// String returns a human-readable representation of the Contributor.
func (c *Contributor) String() string {
	if c == nil {
		return ""
	}
	return fmt.Sprintf("Contributor<%s <%s>>[commits=%d]", c.name, c.email, c.commitCount)
}

// Tag represents an immutable repository tag marker.
type Tag struct {
	name      string
	commitSHA string
	tagType   string
	timestamp time.Time
}

// NewTag creates a new immutable Tag record.
func NewTag(name, commitSHA, tagType string, timestamp time.Time) *Tag {
	return &Tag{
		name:      strings.TrimSpace(name),
		commitSHA: strings.TrimSpace(commitSHA),
		tagType:   strings.TrimSpace(tagType),
		timestamp: timestamp,
	}
}

// Name returns the tag name string (e.g. "v1.0.0").
func (t *Tag) Name() string {
	if t == nil {
		return ""
	}
	return t.name
}

// CommitSHA returns the commit SHA associated with the tag.
func (t *Tag) CommitSHA() string {
	if t == nil {
		return ""
	}
	return t.commitSHA
}

// TagType returns the tag type (e.g. "lightweight" or "annotated").
func (t *Tag) TagType() string {
	if t == nil {
		return ""
	}
	return t.tagType
}

// Timestamp returns the tag creation timestamp, if available.
func (t *Tag) Timestamp() time.Time {
	if t == nil {
		return time.Time{}
	}
	return t.timestamp
}

// String returns a human-readable representation of the Tag.
func (t *Tag) String() string {
	if t == nil {
		return ""
	}
	return fmt.Sprintf("Tag<%s>[sha=%s, type=%s]", t.name, t.commitSHA, t.tagType)
}

// Release represents locally available release information derived from repository tags.
type Release struct {
	name         string
	tagName      string
	commitSHA    string
	isPrerelease bool
	publishedAt  time.Time
}

// NewRelease creates a new immutable Release record.
func NewRelease(name, tagName, commitSHA string, isPrerelease bool, publishedAt time.Time) *Release {
	return &Release{
		name:         strings.TrimSpace(name),
		tagName:      strings.TrimSpace(tagName),
		commitSHA:    strings.TrimSpace(commitSHA),
		isPrerelease: isPrerelease,
		publishedAt:  publishedAt,
	}
}

// Name returns the release title or name.
func (r *Release) Name() string {
	if r == nil {
		return ""
	}
	return r.name
}

// TagName returns the underlying Git tag associated with the release.
func (r *Release) TagName() string {
	if r == nil {
		return ""
	}
	return r.tagName
}

// CommitSHA returns the commit SHA of the release.
func (r *Release) CommitSHA() string {
	if r == nil {
		return ""
	}
	return r.commitSHA
}

// IsPrerelease reports whether the release is flagged as a prerelease or alpha/beta/rc.
func (r *Release) IsPrerelease() bool {
	if r == nil {
		return false
	}
	return r.isPrerelease
}

// PublishedAt returns the release publication timestamp.
func (r *Release) PublishedAt() time.Time {
	if r == nil {
		return time.Time{}
	}
	return r.publishedAt
}

// String returns a human-readable representation of the Release.
func (r *Release) String() string {
	if r == nil {
		return ""
	}
	return fmt.Sprintf("Release<%s>[tag=%s, prerelease=%t]", r.name, r.tagName, r.isPrerelease)
}
