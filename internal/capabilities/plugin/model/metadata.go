package model

import (
	"sort"
	"strings"
)

// Metadata encapsulates descriptive, display, and categorization metadata for a plugin.
type Metadata struct {
	name             string
	description      string
	publisher        string
	license          string
	homepage         string
	tags             []string
	customAttributes map[string]string
}

// NewMetadata constructs a Metadata instance with defensive copying of mutable collections.
func NewMetadata(name, desc, publisher, license, homepage string, tags []string, custom map[string]string) Metadata {
	var cleanTags []string
	if len(tags) > 0 {
		seen := make(map[string]struct{}, len(tags))
		for _, tag := range tags {
			t := strings.TrimSpace(strings.ToLower(tag))
			if t != "" {
				if _, ok := seen[t]; !ok {
					seen[t] = struct{}{}
					cleanTags = append(cleanTags, t)
				}
			}
		}
		sort.Strings(cleanTags)
	}

	cleanCustom := make(map[string]string, len(custom))
	for k, v := range custom {
		kTrim := strings.TrimSpace(k)
		if kTrim != "" {
			cleanCustom[kTrim] = strings.TrimSpace(v)
		}
	}

	return Metadata{
		name:             strings.TrimSpace(name),
		description:      strings.TrimSpace(desc),
		publisher:        strings.TrimSpace(publisher),
		license:          strings.TrimSpace(license),
		homepage:         strings.TrimSpace(homepage),
		tags:             cleanTags,
		customAttributes: cleanCustom,
	}
}

// Name returns the human-readable display name.
func (m Metadata) Name() string {
	return m.name
}

// Description returns the summary description.
func (m Metadata) Description() string {
	return m.description
}

// Publisher returns the publisher organization or author name.
func (m Metadata) Publisher() string {
	return m.publisher
}

// License returns the license identifier (e.g. "MIT", "Apache-2.0").
func (m Metadata) License() string {
	return m.license
}

// Homepage returns the project URL or repository link.
func (m Metadata) Homepage() string {
	return m.homepage
}

// Tags returns a copy of the tags slice.
func (m Metadata) Tags() []string {
	if m.tags == nil {
		return nil
	}
	cp := make([]string, len(m.tags))
	copy(cp, m.tags)
	return cp
}

// CustomAttributes returns a copy of the custom attributes map.
func (m Metadata) CustomAttributes() map[string]string {
	if m.customAttributes == nil {
		return nil
	}
	cp := make(map[string]string, len(m.customAttributes))
	for k, v := range m.customAttributes {
		cp[k] = v
	}
	return cp
}

// Clone creates a deep defensive copy of Metadata.
func (m Metadata) Clone() Metadata {
	return NewMetadata(
		m.name,
		m.description,
		m.publisher,
		m.license,
		m.homepage,
		m.Tags(),
		m.CustomAttributes(),
	)
}
