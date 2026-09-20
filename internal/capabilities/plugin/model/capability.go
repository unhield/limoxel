package model

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

var (
	// ErrInvalidCapability indicates an invalid capability specification.
	ErrInvalidCapability = errors.New("plugin capability: invalid capability definition")

	// capabilityNameRegex validates capability names (e.g. "repository.files", "linter.golang").
	capabilityNameRegex = regexp.MustCompile(`^[a-z0-9]+(?:\.[a-z0-9-]+)*$`)
)

// Capability defines a specific service or feature provided by a plugin.
type Capability struct {
	Name        string         `json:"name"`
	Version     version.SemVer `json:"version"`
	Description string         `json:"description,omitempty"`
}

// NewCapability constructs and validates a Capability instance.
func NewCapability(name string, ver version.SemVer, desc string) (Capability, error) {
	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		return Capability{}, fmt.Errorf("%w: capability name cannot be empty", ErrInvalidCapability)
	}

	if !capabilityNameRegex.MatchString(cleanName) {
		return Capability{}, fmt.Errorf("%w: capability name %q must be dot-separated lower-case alphanumeric", ErrInvalidCapability, cleanName)
	}

	return Capability{
		Name:        cleanName,
		Version:     ver,
		Description: strings.TrimSpace(desc),
	}, nil
}

// RequiredCapability specifies an environmental capability dependency.
type RequiredCapability struct {
	Name       string          `json:"name"`
	MinVersion *version.SemVer `json:"min_version,omitempty"`
	MaxVersion *version.SemVer `json:"max_version,omitempty"`
	Optional   bool            `json:"optional,omitempty"`
}

// NewRequiredCapability constructs and validates a RequiredCapability.
func NewRequiredCapability(name string, minVer, maxVer *version.SemVer, optional bool) (RequiredCapability, error) {
	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		return RequiredCapability{}, fmt.Errorf("%w: required capability name cannot be empty", ErrInvalidCapability)
	}

	if !capabilityNameRegex.MatchString(cleanName) {
		return RequiredCapability{}, fmt.Errorf("%w: required capability name %q must be dot-separated lower-case alphanumeric", ErrInvalidCapability, cleanName)
	}

	if minVer != nil && maxVer != nil && minVer.Compare(*maxVer) > 0 {
		return RequiredCapability{}, fmt.Errorf("%w: min_version %s cannot exceed max_version %s", ErrInvalidCapability, minVer, maxVer)
	}

	return RequiredCapability{
		Name:       cleanName,
		MinVersion: minVer,
		MaxVersion: maxVer,
		Optional:   optional,
	}, nil
}

// IsSatisfiedBy checks whether a provided capability satisfies this requirement.
func (rc RequiredCapability) IsSatisfiedBy(cap Capability) bool {
	if rc.Name != cap.Name {
		return false
	}
	if rc.MinVersion != nil && cap.Version.Compare(*rc.MinVersion) < 0 {
		return false
	}
	if rc.MaxVersion != nil && cap.Version.Compare(*rc.MaxVersion) > 0 {
		return false
	}
	return true
}
