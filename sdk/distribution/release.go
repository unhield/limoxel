package distribution

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
	canonversion "github.com/unhield/limoxel/internal/version"
)

// ReleaseManifest encapsulates official metadata for a released SDK version.
type ReleaseManifest struct {
	ProductName string          `json:"product_name"`
	Version     string          `json:"version"`
	SemVer      version.SemVer  `json:"semver"`
	ReleaseKind string          `json:"release_kind"`
	CreatedAt   time.Time       `json:"created_at"`
	Checksums   []ChecksumEntry `json:"checksums,omitempty"`
	Metadata    map[string]any  `json:"metadata,omitempty"`
}

// GenerateReleaseManifest builds a structured ReleaseManifest using the authoritative version.
func GenerateReleaseManifest(checksums []ChecksumEntry) (*ReleaseManifest, error) {
	currentSV := version.Current()

	manifest := &ReleaseManifest{
		ProductName: "Limoxel SDK",
		Version:     canonversion.Version,
		SemVer:      currentSV,
		ReleaseKind: string(version.ReleaseMinor),
		CreatedAt:   time.Now().UTC(),
		Checksums:   checksums,
		Metadata: map[string]any{
			"go_version": "1.26.5",
			"license":    "MIT",
			"module":     "github.com/unhield/limoxel",
		},
	}

	return manifest, nil
}

// MarshalReleaseManifest converts a ReleaseManifest to formatted JSON.
func (m *ReleaseManifest) MarshalReleaseManifest() ([]byte, error) {
	if m == nil {
		return nil, fmt.Errorf("release manifest cannot be nil")
	}
	return json.MarshalIndent(m, "", "  ")
}
