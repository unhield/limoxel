package validation

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

var (
	// ErrArchiveTraversal indicates an illegal path attempting directory traversal.
	ErrArchiveTraversal = errors.New("validation: archive contains illegal path traversal")
	// ErrArchiveTooLarge indicates unpacked size exceeded safe limits.
	ErrArchiveTooLarge = errors.New("validation: archive unpacked size exceeds safe limit")
	// ErrTooManyFiles indicates file count exceeded safe limits.
	ErrTooManyFiles = errors.New("validation: archive contains too many files")
	// ErrMissingManifest indicates plugin manifest was not found in archive.
	ErrMissingManifest = errors.New("validation: missing plugin manifest in package")
	// ErrInvalidManifest indicates invalid manifest syntax or missing required fields.
	ErrInvalidManifest = errors.New("validation: invalid plugin manifest")

	pluginIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9\-_\.]{2,62}[a-z0-9]$`)
)

// ManifestSchema represents the core fields expected in a plugin.json file.
type ManifestSchema struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Version        string            `json:"version"`
	Publisher      string            `json:"publisher"`
	Description    string            `json:"description,omitempty"`
	Entrypoint     string            `json:"entrypoint,omitempty"`
	MinHostVersion string            `json:"min_host_version,omitempty"`
	Categories     []string          `json:"categories,omitempty"`
	Capabilities   []string          `json:"capabilities,omitempty"`
	Permissions    []string          `json:"permissions,omitempty"`
	Dependencies   map[string]string `json:"dependencies,omitempty"`
}

// ExtractedPackage holds unpacked files and parsed manifest.
type ExtractedPackage struct {
	Manifest       ManifestSchema
	ParsedVersion  version.SemVer
	ManifestRaw    []byte
	ManifestDigest string
	Files          map[string][]byte
}

// InspectAndExtract inspects an in-memory package archive with traversal protection and size limits.
func InspectAndExtract(r io.Reader, policy ValidationPolicy) (*ExtractedPackage, []CheckResult, error) {
	var checks []CheckResult

	gzReader, err := gzip.NewReader(r)
	if err != nil {
		return nil, []CheckResult{{
			Name:     "GzipDecompress",
			Passed:   false,
			Details:  fmt.Sprintf("Corrupt or invalid gzip stream: %v", err),
			Critical: true,
		}}, err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	extracted := &ExtractedPackage{
		Files: make(map[string][]byte),
	}

	var totalUnpacked int64
	var fileCount int
	manifestFound := false

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, checks, fmt.Errorf("error reading tar entry: %w", err)
		}

		fileCount++
		if fileCount > policy.MaxFileCount {
			return nil, append(checks, CheckResult{
				Name:     "FileCountLimit",
				Passed:   false,
				Details:  fmt.Sprintf("Exceeded max file count of %d", policy.MaxFileCount),
				Critical: true,
			}), ErrTooManyFiles
		}

		// Security: reject symlinks, hardlinks, and special files
		if header.Typeflag == tar.TypeSymlink || header.Typeflag == tar.TypeLink {
			return nil, append(checks, CheckResult{
				Name:     "SafeArchivePaths",
				Passed:   false,
				Details:  fmt.Sprintf("Symlinks and hardlinks are forbidden: %s", header.Name),
				Critical: true,
			}), ErrArchiveTraversal
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeDir {
			return nil, append(checks, CheckResult{
				Name:     "SafeArchivePaths",
				Passed:   false,
				Details:  fmt.Sprintf("Special entry type forbidden: %s", header.Name),
				Critical: true,
			}), ErrArchiveTraversal
		}

		// Clean and validate path
		if err := validatePackageEntryPath(header.Name); err != nil {
			return nil, append(checks, CheckResult{
				Name:     "SafeArchivePaths",
				Passed:   false,
				Details:  fmt.Sprintf("Path traversal or unsafe name detected: %v", err),
				Critical: true,
			}), ErrArchiveTraversal
		}

		cleanName := filepath.ToSlash(filepath.Clean(header.Name))

		// Directories don't consume file content
		if header.Typeflag == tar.TypeDir {
			continue
		}

		// Single file limit check
		if header.Size > 20*1024*1024 {
			return nil, checks, fmt.Errorf("single file %s exceeds 20MB limit", cleanName)
		}

		totalUnpacked += header.Size
		if totalUnpacked > policy.MaxUncompressed {
			return nil, append(checks, CheckResult{
				Name:     "UncompressedSizeLimit",
				Passed:   false,
				Details:  fmt.Sprintf("Exceeded max unpacked size of %d bytes", policy.MaxUncompressed),
				Critical: true,
			}), ErrArchiveTooLarge
		}

		// Read file bytes safely with limit
		var buf bytes.Buffer
		n, err := io.CopyN(&buf, tarReader, header.Size)
		if err != nil && err != io.EOF {
			return nil, checks, fmt.Errorf("failed reading file %s: %w", cleanName, err)
		}
		if n != header.Size {
			return nil, checks, fmt.Errorf("file size mismatch for %s: header %d, read %d", cleanName, header.Size, n)
		}

		data := buf.Bytes()
		extracted.Files[cleanName] = data

		// Check for manifest: plugin.json or manifest.json in root or first subdir
		baseName := filepath.Base(cleanName)
		if (baseName == "plugin.json" || baseName == "manifest.json") && !manifestFound {
			manifestFound = true
			extracted.ManifestRaw = data

			h := sha256.Sum256(data)
			extracted.ManifestDigest = hex.EncodeToString(h[:])

			var mf ManifestSchema
			if err := json.Unmarshal(data, &mf); err != nil {
				return nil, append(checks, CheckResult{
					Name:     "ManifestJSONValid",
					Passed:   false,
					Details:  fmt.Sprintf("Manifest JSON decode error: %v", err),
					Critical: true,
				}), fmt.Errorf("%w: %v", ErrInvalidManifest, err)
			}
			extracted.Manifest = mf
		}
	}

	checks = append(checks, CheckResult{
		Name:     "SafeArchivePaths",
		Passed:   true,
		Details:  fmt.Sprintf("Inspected %d files (%d bytes unpacked) without traversal", fileCount, totalUnpacked),
		Critical: true,
	})

	if !manifestFound {
		return nil, append(checks, CheckResult{
			Name:     "ManifestPresence",
			Passed:   false,
			Details:  "No plugin.json or manifest.json found in package",
			Critical: true,
		}), ErrMissingManifest
	}

	// Validate manifest fields
	id := strings.TrimSpace(extracted.Manifest.ID)
	if !pluginIDPattern.MatchString(id) {
		return nil, append(checks, CheckResult{
			Name:     "ManifestIDFormat",
			Passed:   false,
			Details:  fmt.Sprintf("Invalid plugin ID '%s': must be 4-64 alphanumeric chars", id),
			Critical: true,
		}), fmt.Errorf("%w: invalid plugin ID '%s'", ErrInvalidManifest, id)
	}

	if strings.TrimSpace(extracted.Manifest.Name) == "" {
		return nil, append(checks, CheckResult{
			Name:     "ManifestNamePresence",
			Passed:   false,
			Details:  "Plugin name must not be empty",
			Critical: true,
		}), fmt.Errorf("%w: missing plugin name", ErrInvalidManifest)
	}

	parsedVer, err := version.ParseSemVer(extracted.Manifest.Version)
	if err != nil {
		return nil, append(checks, CheckResult{
			Name:     "ManifestVersionSemVer",
			Passed:   false,
			Details:  fmt.Sprintf("Invalid semver '%s': %v", extracted.Manifest.Version, err),
			Critical: true,
		}), fmt.Errorf("%w: invalid semver: %v", ErrInvalidManifest, err)
	}
	extracted.ParsedVersion = parsedVer

	checks = append(checks, CheckResult{
		Name:     "ManifestSchemaValid",
		Passed:   true,
		Details:  fmt.Sprintf("Plugin ID: %s, Version: %s, Publisher: %s", id, parsedVer.String(), extracted.Manifest.Publisher),
		Critical: true,
	})

	return extracted, checks, nil
}

func validatePackageEntryPath(rawName string) error {
	clean := filepath.ToSlash(filepath.Clean(rawName))

	// Reject absolute paths, UNC paths, and traversal sequences
	if strings.HasPrefix(clean, "/") || strings.HasPrefix(clean, "\\") ||
		strings.HasPrefix(clean, "//") || strings.HasPrefix(rawName, "\\\\") ||
		strings.Contains(clean, "../") || strings.Contains(clean, "..\\") ||
		clean == ".." || strings.Contains(clean, ":") {
		return fmt.Errorf("path traversal, alternate data stream, or absolute path detected: %s", rawName)
	}

	parts := strings.Split(clean, "/")
	if len(parts) > 32 {
		return fmt.Errorf("path depth exceeds limit: %s", rawName)
	}
	if len(clean) > 260 {
		return fmt.Errorf("path length exceeds limit: %s", rawName)
	}

	for _, part := range parts {
		base := strings.ToUpper(part)
		if idx := strings.IndexByte(base, '.'); idx != -1 {
			base = base[:idx]
		}
		switch base {
		case "CON", "PRN", "AUX", "NUL",
			"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
			"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
			return fmt.Errorf("reserved device name detected: %s", part)
		}
	}

	return nil
}
