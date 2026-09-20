package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

var (
	// ErrInvalidDependency indicates a malformed dependency specification.
	ErrInvalidDependency = errors.New("plugin dependency: invalid dependency declaration")
)

// Dependency describes an inter-plugin dependency required by a plugin.
type Dependency struct {
	ID         Identity `json:"id"`
	Constraint string   `json:"constraint"`
	Optional   bool     `json:"optional,omitempty"`
}

// NewDependency constructs and validates a Dependency instance.
func NewDependency(id Identity, constraint string, optional bool) (Dependency, error) {
	if string(id) == "" {
		return Dependency{}, fmt.Errorf("%w: dependency ID cannot be empty", ErrInvalidDependency)
	}

	cleanConstraint := strings.TrimSpace(constraint)
	if cleanConstraint == "" {
		cleanConstraint = "*"
	}

	// Validate that the constraint string can be parsed
	if err := validateConstraint(cleanConstraint); err != nil {
		return Dependency{}, fmt.Errorf("%w: invalid version constraint %q for %s: %v", ErrInvalidDependency, cleanConstraint, id, err)
	}

	return Dependency{
		ID:         id,
		Constraint: cleanConstraint,
		Optional:   optional,
	}, nil
}

// MatchesVersion checks whether a given SemVer satisfies the dependency constraint.
func (d Dependency) MatchesVersion(v version.SemVer) bool {
	return EvaluateConstraint(d.Constraint, v)
}

// EvaluateConstraint evaluates whether a target version satisfies a version constraint string.
func EvaluateConstraint(constraint string, v version.SemVer) bool {
	c := strings.TrimSpace(constraint)
	if c == "" || c == "*" || c == "latest" {
		return true
	}

	// Split by comma or whitespace for compound constraints (e.g. ">= 1.0.0, < 2.0.0")
	parts := strings.FieldsFunc(c, func(r rune) bool {
		return r == ','
	})

	for _, part := range parts {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		if !evaluateSingleClause(p, v) {
			return false
		}
	}
	return true
}

func evaluateSingleClause(clause string, v version.SemVer) bool {
	clause = strings.TrimSpace(clause)

	switch {
	case strings.HasPrefix(clause, ">="):
		target, err := version.ParseSemVer(strings.TrimSpace(clause[2:]))
		if err != nil {
			return false
		}
		return v.Compare(target) >= 0

	case strings.HasPrefix(clause, "<="):
		target, err := version.ParseSemVer(strings.TrimSpace(clause[2:]))
		if err != nil {
			return false
		}
		return v.Compare(target) <= 0

	case strings.HasPrefix(clause, ">"):
		target, err := version.ParseSemVer(strings.TrimSpace(clause[1:]))
		if err != nil {
			return false
		}
		return v.Compare(target) > 0

	case strings.HasPrefix(clause, "<"):
		target, err := version.ParseSemVer(strings.TrimSpace(clause[1:]))
		if err != nil {
			return false
		}
		return v.Compare(target) < 0

	case strings.HasPrefix(clause, "="):
		target, err := version.ParseSemVer(strings.TrimSpace(clause[1:]))
		if err != nil {
			return false
		}
		return v.Compare(target) == 0

	case strings.HasPrefix(clause, "^"):
		// Caret range: ^1.2.3 allows >= 1.2.3 and < 2.0.0
		target, err := version.ParseSemVer(strings.TrimSpace(clause[1:]))
		if err != nil {
			return false
		}
		if v.Compare(target) < 0 {
			return false
		}
		if target.Major > 0 {
			return v.Major == target.Major
		}
		if target.Minor > 0 {
			return v.Minor == target.Minor
		}
		return v.Patch == target.Patch

	case strings.HasPrefix(clause, "~"):
		// Tilde range: ~1.2.3 allows >= 1.2.3 and < 1.3.0
		target, err := version.ParseSemVer(strings.TrimSpace(clause[1:]))
		if err != nil {
			return false
		}
		if v.Compare(target) < 0 {
			return false
		}
		return v.Major == target.Major && v.Minor == target.Minor

	default:
		// Exact match
		target, err := version.ParseSemVer(clause)
		if err != nil {
			return false
		}
		return v.Compare(target) == 0
	}
}

func validateConstraint(c string) error {
	if c == "*" || c == "latest" {
		return nil
	}
	parts := strings.FieldsFunc(c, func(r rune) bool {
		return r == ','
	})
	for _, part := range parts {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		clean := p
		for _, prefix := range []string{">=", "<=", ">", "<", "=", "^", "~"} {
			if strings.HasPrefix(p, prefix) {
				clean = strings.TrimSpace(p[len(prefix):])
				break
			}
		}
		if _, err := version.ParseSemVer(clean); err != nil {
			return fmt.Errorf("invalid semver clause %q: %w", p, err)
		}
	}
	return nil
}
