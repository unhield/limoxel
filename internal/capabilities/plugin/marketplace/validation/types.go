package validation

import (
	"time"

	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

// CheckResult represents the outcome of an individual validation rule or scanner.
type CheckResult struct {
	Name     string `json:"name"`
	Passed   bool   `json:"passed"`
	Details  string `json:"details"`
	Critical bool   `json:"critical"`
}

// ValidationResult records the complete outcome of the validation pipeline for a package.
type ValidationResult struct {
	Passed         bool           `json:"passed"`
	PluginID       string         `json:"plugin_id"`
	Version        version.SemVer `json:"version"`
	ArtifactDigest string         `json:"artifact_digest"`
	ManifestDigest string         `json:"manifest_digest"`
	ValidatedAt    time.Time      `json:"validated_at"`
	Checks         []CheckResult  `json:"checks"`
	Errors         []string       `json:"errors,omitempty"`
	Warnings       []string       `json:"warnings,omitempty"`
}

// ValidationPolicy configures strictness and security checks for the pipeline.
type ValidationPolicy struct {
	RequireSignature bool
	AllowBinaries    bool
	MaxUncompressed  int64
	MaxFileCount     int
	CurrentHostVer   version.SemVer
}

// DefaultPolicy returns standard safe production defaults for package validation.
func DefaultPolicy() ValidationPolicy {
	return ValidationPolicy{
		RequireSignature: false,            // Set to true when trust keys are mandated
		AllowBinaries:    false,            // Default to script/source plugins; reject raw executables
		MaxUncompressed:  50 * 1024 * 1024, // 50MB maximum unpacked
		MaxFileCount:     5000,
		CurrentHostVer:   version.SemVer{Major: 1, Minor: 0, Patch: 0},
	}
}
