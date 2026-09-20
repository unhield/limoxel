package marketplace

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
	pubmarket "github.com/unhield/limoxel/plugin/marketplace"
)

var (
	// ErrExtractionSecurity indicates path traversal or unsafe file in archive.
	ErrExtractionSecurity = errors.New("marketplace distribution: unsafe path in archive extraction")
	// ErrDigestMismatch indicates downloaded artifact digest did not match expected digest.
	ErrDigestMismatch = errors.New("marketplace distribution: artifact digest mismatch during download")
)

// LocalClient provides an in-process implementation of the public marketplace Client interface.
type LocalClient struct {
	service *Service
}

// NewLocalClient creates a client bound to a local marketplace service.
func NewLocalClient(service *Service) *LocalClient {
	return &LocalClient{
		service: service,
	}
}

// Ensure LocalClient implements pubmarket.Client at compile-time.
var _ pubmarket.Client = (*LocalClient)(nil)

// Search delegates to the service search.
func (c *LocalClient) Search(ctx context.Context, query pubmarket.SearchQuery) (*pubmarket.SearchResult, error) {
	return c.service.Search(ctx, query)
}

// GetPlugin retrieves detail for a plugin.
func (c *LocalClient) GetPlugin(ctx context.Context, pluginID string) (*pubmarket.PluginDetail, error) {
	return c.service.GetPlugin(ctx, pluginID)
}

// GetVersions returns all versions for a plugin.
func (c *LocalClient) GetVersions(ctx context.Context, pluginID string) ([]pubmarket.VersionInfo, error) {
	return c.service.GetVersions(ctx, pluginID)
}

// GetFeatured retrieves featured plugins.
func (c *LocalClient) GetFeatured(ctx context.Context, limit int) ([]pubmarket.PluginSummary, error) {
	return c.service.GetFeatured(ctx, limit)
}

// GetTrending retrieves trending plugins.
func (c *LocalClient) GetTrending(ctx context.Context, limit int) ([]pubmarket.PluginSummary, error) {
	return c.service.GetTrending(ctx, limit)
}

// GetCategories returns official marketplace categories.
func (c *LocalClient) GetCategories(ctx context.Context) ([]pubmarket.Category, error) {
	return c.service.GetCategories(ctx)
}

// GetPublisher retrieves a publisher profile.
func (c *LocalClient) GetPublisher(ctx context.Context, publisherID string) (*pubmarket.PublisherProfile, error) {
	return c.service.GetPublisher(ctx, publisherID)
}

// CheckUpdates compares installed versions against published versions.
func (c *LocalClient) CheckUpdates(ctx context.Context, installed []pubmarket.UpdateCheckItem) ([]pubmarket.UpdateAvailableInfo, error) {
	return c.service.CheckUpdates(ctx, installed)
}

// DownloadArtifact downloads and verifies the artifact for a plugin version, writing it to destPath.
func (c *LocalClient) DownloadArtifact(
	ctx context.Context,
	pluginID string,
	ver version.SemVer,
	destPath string,
) (*pubmarket.VersionInfo, error) {
	stream, vInfo, err := c.service.DownloadArtifact(ctx, pluginID, ver)
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	// Ensure destination directory exists
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Write to temporary file while computing sha256
	tmpFile := destPath + ".tmp"
	f, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0640)
	if err != nil {
		return nil, fmt.Errorf("failed to create download temp file: %w", err)
	}

	hasher := sha256.New()
	mw := io.MultiWriter(f, hasher)

	if _, err := io.Copy(mw, stream); err != nil {
		f.Close()
		_ = os.Remove(tmpFile)
		return nil, fmt.Errorf("failed downloading artifact stream: %w", err)
	}
	f.Close()

	computedDigest := hex.EncodeToString(hasher.Sum(nil))
	if computedDigest != vInfo.ArtifactDigest {
		_ = os.Remove(tmpFile)
		return nil, fmt.Errorf("%w: expected %s, got %s", ErrDigestMismatch, vInfo.ArtifactDigest, computedDigest)
	}

	// Atomically move to final destination
	if err := os.Rename(tmpFile, destPath); err != nil {
		_ = os.Remove(tmpFile)
		return nil, fmt.Errorf("failed moving artifact to final path: %w", err)
	}

	return vInfo, nil
}

// SubmitReview posts a review for a plugin.
func (c *LocalClient) SubmitReview(ctx context.Context, review pubmarket.Review) error {
	return c.service.SubmitReview(ctx, review)
}

// SubmitRating posts a star rating.
func (c *LocalClient) SubmitRating(ctx context.Context, pluginID, authorID string, stars int) (*pubmarket.RatingSummary, error) {
	return c.service.SubmitRating(ctx, pluginID, authorID, stars)
}

// ResolveDependencies computes an installation plan resolving all required dependencies from the catalog.
func (c *LocalClient) ResolveDependencies(ctx context.Context, pluginID string, ver version.SemVer) (*pubmarket.DependencyResolutionPlan, error) {
	return c.service.ResolveDependencies(ctx, pluginID, ver)
}

// SafeExtractor safely extracts plugin archive archives to a target directory with path sanitization.
type SafeExtractor struct {
	MaxFiles int
	MaxSize  int64
}

// NewSafeExtractor creates a safe archive extractor.
func NewSafeExtractor() *SafeExtractor {
	return &SafeExtractor{
		MaxFiles: 5000,
		MaxSize:  50 * 1024 * 1024, // 50MB
	}
}

// Extract extracts an in-memory or streamed tar.gz package into destDir safely.
func (e *SafeExtractor) Extract(r io.Reader, destDir string) error {
	cleanDest := filepath.Clean(destDir)
	if err := os.MkdirAll(cleanDest, 0750); err != nil {
		return fmt.Errorf("failed to create target extraction directory: %w", err)
	}

	gzReader, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("invalid gzip stream: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	var totalFiles int
	var totalBytes int64

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading archive entry: %w", err)
		}

		totalFiles++
		if totalFiles > e.MaxFiles {
			return errors.New("safe extractor: exceeded maximum file count limit")
		}

		// Security: reject symlinks, hardlinks, device nodes, and fifos
		if header.Typeflag == tar.TypeSymlink || header.Typeflag == tar.TypeLink {
			return fmt.Errorf("%w: symlinks and hardlinks are not permitted: %s", ErrExtractionSecurity, header.Name)
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeDir {
			return fmt.Errorf("%w: special entries and devices are not permitted: %s", ErrExtractionSecurity, header.Name)
		}

		// Security: sanitize path against traversal, ADS, UNC, and reserved names
		if err := validateArchivePath(header.Name); err != nil {
			return fmt.Errorf("%w: %v", ErrExtractionSecurity, err)
		}

		cleanName := filepath.ToSlash(filepath.Clean(header.Name))
		targetPath := filepath.Join(cleanDest, filepath.FromSlash(cleanName))

		// Ensure target path is strictly within cleanDest
		rel, err := filepath.Rel(cleanDest, targetPath)
		if err != nil || strings.HasPrefix(rel, "..") || rel == "." && header.Typeflag != tar.TypeDir {
			return fmt.Errorf("%w: path escapes target directory: %s", ErrExtractionSecurity, header.Name)
		}
		if !strings.HasPrefix(targetPath, cleanDest+string(filepath.Separator)) && targetPath != cleanDest {
			return fmt.Errorf("%w: path escapes target directory: %s", ErrExtractionSecurity, header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			// Ensure target does not collide with an existing regular file
			if fi, err := os.Stat(targetPath); err == nil && !fi.IsDir() {
				return fmt.Errorf("%w: directory collides with existing file %s", ErrExtractionSecurity, targetPath)
			}
			if err := os.MkdirAll(targetPath, 0750); err != nil {
				return fmt.Errorf("failed creating directory %s: %w", targetPath, err)
			}
		case tar.TypeReg:
			totalBytes += header.Size
			if totalBytes > e.MaxSize {
				return errors.New("safe extractor: exceeded maximum uncompressed size limit")
			}

			// Ensure target parent directory exists
			parentDir := filepath.Dir(targetPath)
			if err := os.MkdirAll(parentDir, 0750); err != nil {
				return fmt.Errorf("failed creating parent directory for %s: %w", targetPath, err)
			}

			// Ensure target does not collide with an existing directory
			if fi, err := os.Stat(targetPath); err == nil && fi.IsDir() {
				return fmt.Errorf("%w: file collides with existing directory %s", ErrExtractionSecurity, targetPath)
			}

			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0640)
			if err != nil {
				return fmt.Errorf("failed creating extracted file %s: %w", targetPath, err)
			}

			if _, err := io.CopyN(outFile, tarReader, header.Size); err != nil && err != io.EOF {
				outFile.Close()
				return fmt.Errorf("failed writing extracted file %s: %w", targetPath, err)
			}
			outFile.Close()
		}
	}

	return nil
}

// validateArchivePath validates an archive entry path against traversal, ADS, UNC, and device names.
func validateArchivePath(rawName string) error {
	clean := filepath.ToSlash(filepath.Clean(rawName))

	// Reject absolute paths, UNC paths, and traversal sequences
	if strings.HasPrefix(clean, "/") || strings.HasPrefix(clean, "\\") ||
		strings.HasPrefix(clean, "//") || strings.HasPrefix(rawName, "\\\\") ||
		strings.Contains(clean, "../") || strings.Contains(clean, "..\\") ||
		clean == ".." || strings.Contains(clean, ":") {
		return fmt.Errorf("path traversal, alternate data stream, or absolute path detected: %s", rawName)
	}

	// Reject excessive depth and length
	parts := strings.Split(clean, "/")
	if len(parts) > 32 {
		return fmt.Errorf("path depth exceeds limit: %s", rawName)
	}
	if len(clean) > 260 {
		return fmt.Errorf("path length exceeds limit: %s", rawName)
	}

	// Reject Windows reserved device names
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
