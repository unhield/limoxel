package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
	"github.com/unhield/limoxel/plugin"
)

const (
	// ManifestSchemaVersion defines the canonical supported manifest format version.
	ManifestSchemaVersion = "1.0.0"
)

var (
	// ErrUnsupportedSchemaVersion indicates that the manifest schema version is unrecognized.
	ErrUnsupportedSchemaVersion = errors.New("plugin manifest: unsupported schema version")

	// ErrInvalidManifest indicates a structural or semantic violation in the manifest.
	ErrInvalidManifest = errors.New("plugin manifest: invalid manifest")
)

// CompatibilityRequirement specifies host environment compatibility rules.
type CompatibilityRequirement struct {
	MinHostVersion     *version.SemVer `json:"min_host_version,omitempty"`
	MaxHostVersion     *version.SemVer `json:"max_host_version,omitempty"`
	SupportedPlatforms []string        `json:"supported_platforms,omitempty"`
}

// Manifest represents the complete, validated internal model of a plugin manifest.
type Manifest struct {
	SchemaVersion        string                   `json:"schema_version"`
	ID                   Identity                 `json:"id"`
	Name                 string                   `json:"name"`
	Version              version.SemVer           `json:"version"`
	Publisher            string                   `json:"publisher,omitempty"`
	Description          string                   `json:"description,omitempty"`
	Entrypoint           string                   `json:"entrypoint,omitempty"`
	Capabilities         []Capability             `json:"capabilities,omitempty"`
	RequiredCapabilities []RequiredCapability     `json:"required_capabilities,omitempty"`
	Dependencies         []Dependency             `json:"dependencies,omitempty"`
	Compatibility        CompatibilityRequirement `json:"compatibility,omitempty"`
	Metadata             Metadata                 `json:"metadata,omitempty"`
}

type rawManifest struct {
	SchemaVersion string `json:"schema_version"`
	ID            string `json:"id"`
	Name          string `json:"name"`
	Version       string `json:"version"`
	Publisher     string `json:"publisher,omitempty"`
	Description   string `json:"description,omitempty"`
	Entrypoint    string `json:"entrypoint,omitempty"`
	Capabilities  []struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Description string `json:"description,omitempty"`
	} `json:"capabilities,omitempty"`
	RequiredCapabilities []struct {
		Name       string `json:"name"`
		MinVersion string `json:"min_version,omitempty"`
		MaxVersion string `json:"max_version,omitempty"`
		Optional   bool   `json:"optional,omitempty"`
	} `json:"required_capabilities,omitempty"`
	Dependencies []struct {
		ID         string `json:"id"`
		Constraint string `json:"constraint,omitempty"`
		Optional   bool   `json:"optional,omitempty"`
	} `json:"dependencies,omitempty"`
	Compatibility struct {
		MinHostVersion     string   `json:"min_host_version,omitempty"`
		MaxHostVersion     string   `json:"max_host_version,omitempty"`
		SupportedPlatforms []string `json:"supported_platforms,omitempty"`
	} `json:"compatibility,omitempty"`
	Metadata struct {
		License          string            `json:"license,omitempty"`
		Homepage         string            `json:"homepage,omitempty"`
		Tags             []string          `json:"tags,omitempty"`
		CustomAttributes map[string]string `json:"custom_attributes,omitempty"`
	} `json:"metadata,omitempty"`
}

// ParseManifestJSON parses and semantically validates a manifest from JSON bytes.
func ParseManifestJSON(data []byte) (*Manifest, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("%w: manifest content cannot be empty", ErrInvalidManifest)
	}

	var raw rawManifest
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%w: failed to decode JSON: %v", ErrInvalidManifest, err)
	}

	schemaVer := strings.TrimSpace(raw.SchemaVersion)
	if schemaVer == "" {
		return nil, fmt.Errorf("%w: schema_version is required", ErrInvalidManifest)
	}
	if schemaVer != ManifestSchemaVersion {
		return nil, fmt.Errorf("%w: got %q, supported %q", ErrUnsupportedSchemaVersion, schemaVer, ManifestSchemaVersion)
	}

	id, err := ParseIdentity(raw.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid identity: %v", ErrInvalidManifest, err)
	}

	name := strings.TrimSpace(raw.Name)
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrInvalidManifest)
	}

	ver, err := version.ParseSemVer(raw.Version)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid version %q: %v", ErrInvalidManifest, raw.Version, err)
	}

	// Parse capabilities
	caps := make([]Capability, 0, len(raw.Capabilities))
	for _, c := range raw.Capabilities {
		cVer, err := version.ParseSemVer(c.Version)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid version for capability %q: %v", ErrInvalidManifest, c.Name, err)
		}
		capObj, err := NewCapability(c.Name, cVer, c.Description)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidManifest, err)
		}
		caps = append(caps, capObj)
	}

	// Parse required capabilities
	reqCaps := make([]RequiredCapability, 0, len(raw.RequiredCapabilities))
	for _, rc := range raw.RequiredCapabilities {
		var minV, maxV *version.SemVer
		if rc.MinVersion != "" {
			parsed, err := version.ParseSemVer(rc.MinVersion)
			if err != nil {
				return nil, fmt.Errorf("%w: invalid min_version %q for requirement %s: %v", ErrInvalidManifest, rc.MinVersion, rc.Name, err)
			}
			minV = &parsed
		}
		if rc.MaxVersion != "" {
			parsed, err := version.ParseSemVer(rc.MaxVersion)
			if err != nil {
				return nil, fmt.Errorf("%w: invalid max_version %q for requirement %s: %v", ErrInvalidManifest, rc.MaxVersion, rc.Name, err)
			}
			maxV = &parsed
		}
		reqCapObj, err := NewRequiredCapability(rc.Name, minV, maxV, rc.Optional)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidManifest, err)
		}
		reqCaps = append(reqCaps, reqCapObj)
	}

	// Parse dependencies
	deps := make([]Dependency, 0, len(raw.Dependencies))
	for _, d := range raw.Dependencies {
		depID, err := ParseIdentity(d.ID)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid dependency ID %q: %v", ErrInvalidManifest, d.ID, err)
		}
		if depID == id {
			return nil, fmt.Errorf("%w: plugin cannot depend on itself (%s)", ErrInvalidManifest, id)
		}
		depObj, err := NewDependency(depID, d.Constraint, d.Optional)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidManifest, err)
		}
		deps = append(deps, depObj)
	}

	// Parse compatibility
	var compat CompatibilityRequirement
	if raw.Compatibility.MinHostVersion != "" {
		minH, err := version.ParseSemVer(raw.Compatibility.MinHostVersion)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid min_host_version %q: %v", ErrInvalidManifest, raw.Compatibility.MinHostVersion, err)
		}
		compat.MinHostVersion = &minH
	}
	if raw.Compatibility.MaxHostVersion != "" {
		maxH, err := version.ParseSemVer(raw.Compatibility.MaxHostVersion)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid max_host_version %q: %v", ErrInvalidManifest, raw.Compatibility.MaxHostVersion, err)
		}
		compat.MaxHostVersion = &maxH
	}
	if compat.MinHostVersion != nil && compat.MaxHostVersion != nil && compat.MinHostVersion.Compare(*compat.MaxHostVersion) > 0 {
		return nil, fmt.Errorf("%w: min_host_version cannot exceed max_host_version", ErrInvalidManifest)
	}
	compat.SupportedPlatforms = raw.Compatibility.SupportedPlatforms

	meta := NewMetadata(name, raw.Description, raw.Publisher, raw.Metadata.License, raw.Metadata.Homepage, raw.Metadata.Tags, raw.Metadata.CustomAttributes)

	return &Manifest{
		SchemaVersion:        schemaVer,
		ID:                   id,
		Name:                 name,
		Version:              ver,
		Publisher:            strings.TrimSpace(raw.Publisher),
		Description:          strings.TrimSpace(raw.Description),
		Entrypoint:           strings.TrimSpace(raw.Entrypoint),
		Capabilities:         caps,
		RequiredCapabilities: reqCaps,
		Dependencies:         deps,
		Compatibility:        compat,
		Metadata:             meta,
	}, nil
}

// ParseManifestFile reads and parses a manifest file from the local filesystem.
func ParseManifestFile(path string) (*Manifest, error) {
	cleanPath := filepath.Clean(path)
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read manifest file at %s: %v", ErrInvalidManifest, cleanPath, err)
	}
	return ParseManifestJSON(data)
}

// ToPublic converts the internal manifest model to the public plugin.Manifest contract.
func (m *Manifest) ToPublic() plugin.Manifest {
	if m == nil {
		return plugin.Manifest{}
	}

	pubCaps := make([]plugin.Capability, len(m.Capabilities))
	for i, c := range m.Capabilities {
		pubCaps[i] = plugin.Capability{
			Name:        c.Name,
			Version:     c.Version.String(),
			Description: c.Description,
		}
	}

	pubReqs := make([]plugin.RequiredCapability, len(m.RequiredCapabilities))
	for i, r := range m.RequiredCapabilities {
		var minS, maxS string
		if r.MinVersion != nil {
			minS = r.MinVersion.String()
		}
		if r.MaxVersion != nil {
			maxS = r.MaxVersion.String()
		}
		pubReqs[i] = plugin.RequiredCapability{
			Name:       r.Name,
			MinVersion: minS,
			MaxVersion: maxS,
			Optional:   r.Optional,
		}
	}

	pubDeps := make([]plugin.Dependency, len(m.Dependencies))
	for i, d := range m.Dependencies {
		pubDeps[i] = plugin.Dependency{
			ID:         string(d.ID),
			Constraint: d.Constraint,
			Optional:   d.Optional,
		}
	}

	var pubCompat plugin.CompatibilityRequirement
	if m.Compatibility.MinHostVersion != nil {
		pubCompat.MinHostVersion = m.Compatibility.MinHostVersion.String()
	}
	if m.Compatibility.MaxHostVersion != nil {
		pubCompat.MaxHostVersion = m.Compatibility.MaxHostVersion.String()
	}
	pubCompat.SupportedPlatforms = m.Compatibility.SupportedPlatforms

	return plugin.Manifest{
		SchemaVersion:        m.SchemaVersion,
		ID:                   string(m.ID),
		Name:                 m.Name,
		Version:              m.Version.String(),
		Publisher:            m.Publisher,
		Description:          m.Description,
		Entrypoint:           m.Entrypoint,
		Capabilities:         pubCaps,
		RequiredCapabilities: pubReqs,
		Dependencies:         pubDeps,
		Compatibility:        pubCompat,
		Metadata:             m.Metadata.CustomAttributes(),
	}
}

// Clone creates a complete deep defensive copy of Manifest, preventing mutable state leakage.
func (m *Manifest) Clone() *Manifest {
	if m == nil {
		return nil
	}
	cp := *m

	if m.Capabilities != nil {
		cp.Capabilities = make([]Capability, len(m.Capabilities))
		copy(cp.Capabilities, m.Capabilities)
	}

	if m.RequiredCapabilities != nil {
		cp.RequiredCapabilities = make([]RequiredCapability, len(m.RequiredCapabilities))
		for i, rc := range m.RequiredCapabilities {
			rcCp := rc
			if rc.MinVersion != nil {
				v := *rc.MinVersion
				rcCp.MinVersion = &v
			}
			if rc.MaxVersion != nil {
				v := *rc.MaxVersion
				rcCp.MaxVersion = &v
			}
			cp.RequiredCapabilities[i] = rcCp
		}
	}

	if m.Dependencies != nil {
		cp.Dependencies = make([]Dependency, len(m.Dependencies))
		copy(cp.Dependencies, m.Dependencies)
	}

	if m.Compatibility.MinHostVersion != nil {
		v := *m.Compatibility.MinHostVersion
		cp.Compatibility.MinHostVersion = &v
	}
	if m.Compatibility.MaxHostVersion != nil {
		v := *m.Compatibility.MaxHostVersion
		cp.Compatibility.MaxHostVersion = &v
	}
	if m.Compatibility.SupportedPlatforms != nil {
		cp.Compatibility.SupportedPlatforms = make([]string, len(m.Compatibility.SupportedPlatforms))
		copy(cp.Compatibility.SupportedPlatforms, m.Compatibility.SupportedPlatforms)
	}

	cp.Metadata = m.Metadata.Clone()

	return &cp
}
