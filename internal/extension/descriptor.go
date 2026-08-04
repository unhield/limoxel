package extension

import (
	"fmt"
	"strings"
)

// Descriptor represents an immutable metadata descriptor for a registered extension.
type Descriptor struct {
	id          string
	name        string
	version     string
	author      string
	description string
	metadata    map[string]string
}

// NewDescriptor constructs and validates a new immutable Descriptor.
func NewDescriptor(id string, name string, version string, author string, description string, metadata map[string]string) (*Descriptor, error) {
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

	cleanVer := strings.TrimSpace(version)
	if cleanVer == "" {
		cleanVer = "1.0.0"
	}

	metaCopy := make(map[string]string)
	for k, v := range metadata {
		cleanK := strings.TrimSpace(k)
		if cleanK != "" {
			metaCopy[cleanK] = v
		}
	}

	return &Descriptor{
		id:          strings.ToLower(cleanID),
		name:        cleanName,
		version:     cleanVer,
		author:      strings.TrimSpace(author),
		description: strings.TrimSpace(description),
		metadata:    metaCopy,
	}, nil
}

// ID returns the canonical lower-case extension identifier string.
func (d *Descriptor) ID() string {
	if d == nil {
		return ""
	}
	return d.id
}

// Name returns the human-readable extension name.
func (d *Descriptor) Name() string {
	if d == nil {
		return ""
	}
	return d.name
}

// Version returns the semantic version string of the extension.
func (d *Descriptor) Version() string {
	if d == nil {
		return ""
	}
	return d.version
}

// Author returns the author or organization string of the extension.
func (d *Descriptor) Author() string {
	if d == nil {
		return ""
	}
	return d.author
}

// Description returns the textual description of the extension.
func (d *Descriptor) Description() string {
	if d == nil {
		return ""
	}
	return d.description
}

// Metadata returns a defensive copy of the extension metadata map.
func (d *Descriptor) Metadata() map[string]string {
	if d == nil || len(d.metadata) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(d.metadata))
	for k, v := range d.metadata {
		cloned[k] = v
	}
	return cloned
}

// String returns a human-readable representation of the Descriptor.
func (d *Descriptor) String() string {
	if d == nil {
		return "Extension<nil>"
	}
	return fmt.Sprintf("Extension<%s>(name=%s, v=%s)", d.id, d.name, d.version)
}
