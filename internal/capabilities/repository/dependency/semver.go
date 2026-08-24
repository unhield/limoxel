package dependency

import (
	"fmt"
	"strconv"
	"strings"
)

// SemanticVersion represents a parsed, normalized semantic version representation.
type SemanticVersion struct {
	raw           string
	major         int
	minor         int
	patch         int
	prerelease    string
	buildMetadata string
	isConstraint  bool
	isValid       bool
}

// ParseSemanticVersion parses a version string into a structured SemanticVersion.
func ParseSemanticVersion(raw string) *SemanticVersion {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return &SemanticVersion{raw: "", isValid: false}
	}

	isConstraint := strings.HasPrefix(clean, "^") ||
		strings.HasPrefix(clean, "~") ||
		strings.HasPrefix(clean, ">=") ||
		strings.HasPrefix(clean, "<=") ||
		strings.HasPrefix(clean, ">") ||
		strings.HasPrefix(clean, "<") ||
		strings.HasPrefix(clean, "=") ||
		strings.Contains(clean, "*") ||
		strings.Contains(clean, "||")

	// Strip constraint prefixes for numeric extraction
	vStr := strings.TrimLeft(clean, "^~>=<vV ")

	// Split build metadata after '+'
	var buildMeta string
	if plusIdx := strings.Index(vStr, "+"); plusIdx != -1 {
		buildMeta = vStr[plusIdx+1:]
		vStr = vStr[:plusIdx]
	}

	// Split prerelease after '-'
	var prerelease string
	if dashIdx := strings.Index(vStr, "-"); dashIdx != -1 {
		prerelease = vStr[dashIdx+1:]
		vStr = vStr[:dashIdx]
	}

	// Split numeric dot segments
	parts := strings.Split(vStr, ".")
	var major, minor, patch int
	var valid bool

	if len(parts) >= 1 {
		if m, err := strconv.Atoi(parts[0]); err == nil {
			major = m
			valid = true
		}
	}
	if len(parts) >= 2 {
		if mi, err := strconv.Atoi(parts[1]); err == nil {
			minor = mi
		}
	}
	if len(parts) >= 3 {
		if p, err := strconv.Atoi(parts[2]); err == nil {
			patch = p
		}
	}

	return &SemanticVersion{
		raw:           clean,
		major:         major,
		minor:         minor,
		patch:         patch,
		prerelease:    prerelease,
		buildMetadata: buildMeta,
		isConstraint:  isConstraint,
		isValid:       valid,
	}
}

// Raw returns the original unparsed version string.
func (sv *SemanticVersion) Raw() string {
	if sv == nil {
		return ""
	}
	return sv.raw
}

// Major returns the major version component.
func (sv *SemanticVersion) Major() int {
	if sv == nil {
		return 0
	}
	return sv.major
}

// Minor returns the minor version component.
func (sv *SemanticVersion) Minor() int {
	if sv == nil {
		return 0
	}
	return sv.minor
}

// Patch returns the patch version component.
func (sv *SemanticVersion) Patch() int {
	if sv == nil {
		return 0
	}
	return sv.patch
}

// Prerelease returns the prerelease tag, if any.
func (sv *SemanticVersion) Prerelease() string {
	if sv == nil {
		return ""
	}
	return sv.prerelease
}

// BuildMetadata returns the build metadata string, if any.
func (sv *SemanticVersion) BuildMetadata() string {
	if sv == nil {
		return ""
	}
	return sv.buildMetadata
}

// IsConstraint reports whether the raw version represents a version constraint.
func (sv *SemanticVersion) IsConstraint() bool {
	if sv == nil {
		return false
	}
	return sv.isConstraint
}

// IsValid reports whether the version string parsed into valid numeric components.
func (sv *SemanticVersion) IsValid() bool {
	if sv == nil {
		return false
	}
	return sv.isValid
}

// String returns a canonical representation of the SemanticVersion.
func (sv *SemanticVersion) String() string {
	if sv == nil {
		return ""
	}
	if !sv.isValid {
		return sv.raw
	}
	res := fmt.Sprintf("%d.%d.%d", sv.major, sv.minor, sv.patch)
	if sv.prerelease != "" {
		res += "-" + sv.prerelease
	}
	if sv.buildMetadata != "" {
		res += "+" + sv.buildMetadata
	}
	return res
}
