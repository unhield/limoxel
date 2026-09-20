package model

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	// ErrInvalidIdentity indicates that a plugin identifier violates naming conventions.
	ErrInvalidIdentity = errors.New("plugin identity: invalid identifier format")

	// identityRegex validates canonical plugin identities:
	// - 3 to 128 characters
	// - lower-case alphanumeric, dots, and hyphens
	// - must begin and end with an alphanumeric character
	// - no consecutive dots or hyphens
	identityRegex = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]*[a-z0-9])?)*$`)
)

// Identity represents an explicit, validated, immutable plugin identifier.
type Identity string

// ParseIdentity validates and constructs an Identity from a string.
func ParseIdentity(raw string) (Identity, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("%w: identity cannot be empty", ErrInvalidIdentity)
	}

	if len(trimmed) < 3 || len(trimmed) > 128 {
		return "", fmt.Errorf("%w: identity length must be between 3 and 128 characters, got %d", ErrInvalidIdentity, len(trimmed))
	}

	if !identityRegex.MatchString(trimmed) {
		return "", fmt.Errorf("%w: %q must consist of lowercase alphanumeric segments separated by dots (e.g. 'com.example.linter')", ErrInvalidIdentity, trimmed)
	}

	return Identity(trimmed), nil
}

// MustParseIdentity parses an Identity or panics if invalid. Intended for static constants.
func MustParseIdentity(raw string) Identity {
	id, err := ParseIdentity(raw)
	if err != nil {
		panic(err)
	}
	return id
}

// String returns the canonical string value of the Identity.
func (id Identity) String() string {
	return string(id)
}

// Namespace returns the namespace segment of the identity (all segments except the last).
// For example, "com.example.linter" returns "com.example". If no dot is present, returns "".
func (id Identity) Namespace() string {
	str := string(id)
	idx := strings.LastIndex(str, ".")
	if idx < 0 {
		return ""
	}
	return str[:idx]
}

// Name returns the base name segment of the identity (the final dot-separated segment).
// For example, "com.example.linter" returns "linter".
func (id Identity) Name() string {
	str := string(id)
	idx := strings.LastIndex(str, ".")
	if idx < 0 {
		return str
	}
	return str[idx+1:]
}

// Compare returns -1 if id < other, 0 if id == other, and 1 if id > other.
func (id Identity) Compare(other Identity) int {
	return strings.Compare(string(id), string(other))
}

// Less returns true if id lexicographically precedes other.
func (id Identity) Less(other Identity) bool {
	return string(id) < string(other)
}
