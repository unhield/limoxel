package distribution

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	canonversion "github.com/unhield/limoxel/internal/version"
)

var (
	ErrValidationFailed = errors.New("distribution: package validation failed")
)

// PackageValidationResult encapsulates findings from SDK package audit.
type PackageValidationResult struct {
	IsValid          bool     `json:"is_valid"`
	Version          string   `json:"version"`
	RequiredFilesOK  bool     `json:"required_files_ok"`
	MissingFiles     []string `json:"missing_files,omitempty"`
	FoundFiles       []string `json:"found_files,omitempty"`
	ValidationErrors []string `json:"validation_errors,omitempty"`
}

// RequiredPublicFiles defines the essential files that must exist in a public repository package.
var RequiredPublicFiles = []string{
	"README.md",
	"CHANGELOG.md",
	"LICENSE",
	"go.mod",
	"sdk/sdk.go",
}

// ValidatePackage inspects a repository directory to ensure all public SDK packaging invariants hold.
func ValidatePackage(repoRoot string) (*PackageValidationResult, error) {
	result := &PackageValidationResult{
		IsValid:         true,
		Version:         canonversion.Version,
		RequiredFilesOK: true,
	}

	// 1. Check required root files
	for _, req := range RequiredPublicFiles {
		fullPath := filepath.Join(repoRoot, req)
		if _, err := os.Stat(fullPath); err != nil {
			result.IsValid = false
			result.RequiredFilesOK = false
			result.MissingFiles = append(result.MissingFiles, req)
			result.ValidationErrors = append(result.ValidationErrors, fmt.Sprintf("missing required public file: %s", req))
		} else {
			result.FoundFiles = append(result.FoundFiles, req)
		}
	}

	// 2. Validate version consistency
	if result.Version == "" {
		result.IsValid = false
		result.ValidationErrors = append(result.ValidationErrors, "canonical version string is empty")
	}

	if !result.IsValid {
		return result, fmt.Errorf("%w: %d errors found", ErrValidationFailed, len(result.ValidationErrors))
	}

	return result, nil
}
