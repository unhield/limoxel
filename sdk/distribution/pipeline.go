package distribution

import (
	"fmt"
	"os"
	"path/filepath"
)

// PipelineOptions configures the execution of the packaging pipeline.
type PipelineOptions struct {
	RepoRoot   string
	OutputDir  string
	VerifyOnly bool
}

// PipelineResult encapsulates the outcome of a distribution pipeline execution.
type PipelineResult struct {
	Success          bool                     `json:"success"`
	ValidationResult *PackageValidationResult `json:"validation_result"`
	Manifest         *ReleaseManifest         `json:"manifest,omitempty"`
	ChecksumFile     string                   `json:"checksum_file,omitempty"`
	Error            string                   `json:"error,omitempty"`
}

// RunPipeline executes the end-to-end SDK distribution validation and manifest generation.
func RunPipeline(opts PipelineOptions) (*PipelineResult, error) {
	if opts.RepoRoot == "" {
		opts.RepoRoot = "."
	}

	valResult, err := ValidatePackage(opts.RepoRoot)
	if err != nil {
		return &PipelineResult{
			Success:          false,
			ValidationResult: valResult,
			Error:            err.Error(),
		}, err
	}

	if opts.VerifyOnly {
		return &PipelineResult{
			Success:          true,
			ValidationResult: valResult,
		}, nil
	}

	// Generate checksums for key files
	var targetFiles []string
	for _, f := range valResult.FoundFiles {
		targetFiles = append(targetFiles, f)
	}

	checksumContent, entries, err := GenerateChecksumManifest(opts.RepoRoot, targetFiles)
	if err != nil {
		return &PipelineResult{
			Success:          false,
			ValidationResult: valResult,
			Error:            fmt.Sprintf("failed to generate checksums: %v", err),
		}, err
	}

	manifest, err := GenerateReleaseManifest(entries)
	if err != nil {
		return &PipelineResult{
			Success:          false,
			ValidationResult: valResult,
			Error:            fmt.Sprintf("failed to generate manifest: %v", err),
		}, err
	}

	// Write output artifacts if OutputDir is specified
	var checksumFile string
	if opts.OutputDir != "" {
		if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create output dir: %w", err)
		}

		checksumFile = filepath.Join(opts.OutputDir, "SHA256SUMS")
		if err := os.WriteFile(checksumFile, []byte(checksumContent), 0644); err != nil {
			return nil, fmt.Errorf("failed to write SHA256SUMS: %w", err)
		}

		manifestJSON, err := manifest.MarshalReleaseManifest()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal manifest: %w", err)
		}

		manifestFile := filepath.Join(opts.OutputDir, "release-manifest.json")
		if err := os.WriteFile(manifestFile, manifestJSON, 0644); err != nil {
			return nil, fmt.Errorf("failed to write release-manifest.json: %w", err)
		}
	}

	return &PipelineResult{
		Success:          true,
		ValidationResult: valResult,
		Manifest:         manifest,
		ChecksumFile:     checksumFile,
	}, nil
}
