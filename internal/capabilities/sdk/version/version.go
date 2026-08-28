package version

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	canonversion "github.com/unhield/limoxel/internal/version"
)

var (
	// ErrInvalidSemVer indicates a malformed semantic version string.
	ErrInvalidSemVer = errors.New("version: invalid semantic version format")

	// semVerRegex validates standard Semantic Versioning 2.0.0 strings.
	semVerRegex = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z.-]+))?(?:\+([0-9A-Za-z.-]+))?$`)
)

// SemVer represents a parsed Semantic Version (MAJOR.MINOR.PATCH[-PRERELEASE][+BUILD]).
type SemVer struct {
	Major         int
	Minor         int
	Patch         int
	PreRelease    string
	BuildMetadata string
}

// NewSemVer constructs a SemVer from explicit version components.
func NewSemVer(major, minor, patch int, preRelease, buildMetadata string) (SemVer, error) {
	if major < 0 || minor < 0 || patch < 0 {
		return SemVer{}, fmt.Errorf("%w: version numbers must be non-negative", ErrInvalidSemVer)
	}
	return SemVer{
		Major:         major,
		Minor:         minor,
		Patch:         patch,
		PreRelease:    strings.TrimSpace(preRelease),
		BuildMetadata: strings.TrimSpace(buildMetadata),
	}, nil
}

// ParseSemVer parses a SemVer string. Leading 'v' is accepted.
func ParseSemVer(v string) (SemVer, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return SemVer{}, fmt.Errorf("%w: version string cannot be empty", ErrInvalidSemVer)
	}

	matches := semVerRegex.FindStringSubmatch(v)
	if len(matches) < 4 {
		return SemVer{}, fmt.Errorf("%w: %q does not match MAJOR.MINOR.PATCH", ErrInvalidSemVer, v)
	}

	maj, err := strconv.Atoi(matches[1])
	if err != nil {
		return SemVer{}, fmt.Errorf("%w: invalid major version: %v", ErrInvalidSemVer, err)
	}
	min, err := strconv.Atoi(matches[2])
	if err != nil {
		return SemVer{}, fmt.Errorf("%w: invalid minor version: %v", ErrInvalidSemVer, err)
	}
	pat, err := strconv.Atoi(matches[3])
	if err != nil {
		return SemVer{}, fmt.Errorf("%w: invalid patch version: %v", ErrInvalidSemVer, err)
	}

	var pre, build string
	if len(matches) > 4 {
		pre = matches[4]
	}
	if len(matches) > 5 {
		build = matches[5]
	}

	return SemVer{
		Major:         maj,
		Minor:         min,
		Patch:         pat,
		PreRelease:    pre,
		BuildMetadata: build,
	}, nil
}

// Current returns the current canonical Limoxel application version as a parsed SemVer.
func Current() SemVer {
	sv, err := ParseSemVer(canonversion.Version)
	if err != nil {
		// Fallback safe default if parsing somehow fails
		return SemVer{Major: 1, Minor: 4, Patch: 0}
	}
	return sv
}

// String formats the SemVer as standard MAJOR.MINOR.PATCH[-PRERELEASE][+BUILD].
func (s SemVer) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d.%d.%d", s.Major, s.Minor, s.Patch))
	if s.PreRelease != "" {
		sb.WriteString("-")
		sb.WriteString(s.PreRelease)
	}
	if s.BuildMetadata != "" {
		sb.WriteString("+")
		sb.WriteString(s.BuildMetadata)
	}
	return sb.String()
}

// CoreString formats only the core MAJOR.MINOR.PATCH segment.
func (s SemVer) CoreString() string {
	return fmt.Sprintf("%d.%d.%d", s.Major, s.Minor, s.Patch)
}

// Compare returns:
//
//	-1 if s < other
//	 0 if s == other
//	 1 if s > other
func (s SemVer) Compare(other SemVer) int {
	if s.Major != other.Major {
		if s.Major < other.Major {
			return -1
		}
		return 1
	}
	if s.Minor != other.Minor {
		if s.Minor < other.Minor {
			return -1
		}
		return 1
	}
	if s.Patch != other.Patch {
		if s.Patch < other.Patch {
			return -1
		}
		return 1
	}

	// SemVer spec: a version with prerelease has lower precedence than one without.
	if s.PreRelease != "" && other.PreRelease == "" {
		return -1
	}
	if s.PreRelease == "" && other.PreRelease != "" {
		return 1
	}
	if s.PreRelease != other.PreRelease {
		if s.PreRelease < other.PreRelease {
			return -1
		}
		return 1
	}

	return 0
}

// IsCompatibleWith returns true if other is backward-compatible with s under standard Semantic Versioning rules:
// - Same Major version (if Major > 0)
// - other >= s (so other contains all features of s)
func (s SemVer) IsCompatibleWith(other SemVer) bool {
	if s.Major == 0 && other.Major == 0 {
		// In 0.y.z development, minor bump may break
		return s.Minor == other.Minor && other.Patch >= s.Patch
	}
	if s.Major != other.Major {
		return false
	}
	return other.Compare(s) >= 0
}

// VersionDiff represents the nature of difference between two semantic versions.
type VersionDiff int

const (
	// DiffNone indicates versions are equal in precedence.
	DiffNone VersionDiff = iota
	// DiffPatch indicates only patch version differ.
	DiffPatch
	// DiffMinor indicates minor version differ.
	DiffMinor
	// DiffMajor indicates major version differ (breaking).
	DiffMajor
	// DiffPreRelease indicates only pre-release differ.
	DiffPreRelease
)

// String returns the string representation of VersionDiff.
func (d VersionDiff) String() string {
	switch d {
	case DiffNone:
		return "none"
	case DiffPatch:
		return "patch"
	case DiffMinor:
		return "minor"
	case DiffMajor:
		return "major"
	case DiffPreRelease:
		return "prerelease"
	default:
		return "unknown"
	}
}

// Diff determines the highest-order component that differs between s and other.
func (s SemVer) Diff(other SemVer) VersionDiff {
	if s.Major != other.Major {
		return DiffMajor
	}
	if s.Minor != other.Minor {
		return DiffMinor
	}
	if s.Patch != other.Patch {
		return DiffPatch
	}
	if s.PreRelease != other.PreRelease {
		return DiffPreRelease
	}
	return DiffNone
}

// ReleaseKind classifies the intended scope of a release under Semantic Versioning.
type ReleaseKind string

const (
	// ReleaseMajor indicates an incompatible / breaking public API release.
	ReleaseMajor ReleaseKind = "MAJOR"
	// ReleaseMinor indicates a backward-compatible functionality addition.
	ReleaseMinor ReleaseKind = "MINOR"
	// ReleasePatch indicates a backward-compatible bug/security fix.
	ReleasePatch ReleaseKind = "PATCH"
)

// ClassifyRelease calculates whether transitioning from current to next is a Major, Minor, or Patch release.
func ClassifyRelease(current, next SemVer) ReleaseKind {
	diff := current.Diff(next)
	switch diff {
	case DiffMajor:
		return ReleaseMajor
	case DiffMinor:
		return ReleaseMinor
	default:
		return ReleasePatch
	}
}
