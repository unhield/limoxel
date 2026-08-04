package parser

import (
	"fmt"
	"strings"
)

// Descriptor represents an immutable metadata descriptor for a registered parser.
type Descriptor struct {
	id         string
	name       string
	languageID string
	version    string
}

// NewDescriptor constructs and validates a new immutable Descriptor.
func NewDescriptor(id string, name string, languageID string, version string) (*Descriptor, error) {
	cleanID := strings.TrimSpace(id)
	if cleanID == "" {
		return nil, ErrInvalidID
	}
	if strings.Contains(cleanID, " ") {
		return nil, fmt.Errorf("%w: ID cannot contain spaces", ErrInvalidID)
	}

	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		return nil, ErrInvalidName
	}

	cleanLangID := strings.TrimSpace(languageID)
	if cleanLangID == "" {
		return nil, ErrInvalidLanguageID
	}

	cleanVer := strings.TrimSpace(version)
	if cleanVer == "" {
		cleanVer = "1.0.0"
	}

	return &Descriptor{
		id:         strings.ToLower(cleanID),
		name:       cleanName,
		languageID: strings.ToLower(cleanLangID),
		version:    cleanVer,
	}, nil
}

// ID returns the canonical lower-case parser identifier string.
func (d *Descriptor) ID() string {
	if d == nil {
		return ""
	}
	return d.id
}

// Name returns the human-readable parser name.
func (d *Descriptor) Name() string {
	if d == nil {
		return ""
	}
	return d.name
}

// LanguageID returns the target lower-case language identifier string.
func (d *Descriptor) LanguageID() string {
	if d == nil {
		return ""
	}
	return d.languageID
}

// Version returns the semantic version string of the parser.
func (d *Descriptor) Version() string {
	if d == nil {
		return ""
	}
	return d.version
}
