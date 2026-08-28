package distribution

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrChecksumMismatch = errors.New("distribution: checksum mismatch detected")
	ErrFileNotFound     = errors.New("distribution: artifact file not found")
)

// ChecksumEntry represents a file and its SHA-256 hash.
type ChecksumEntry struct {
	FilePath string `json:"file_path"`
	SHA256   string `json:"sha256"`
}

// ComputeSHA256 calculates the hex-encoded SHA-256 hash of a file.
func ComputeSHA256(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrFileNotFound, err)
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", fmt.Errorf("failed to read file for hashing: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// GenerateChecksumManifest generates SHA256SUMS formatted content for a list of files.
func GenerateChecksumManifest(baseDir string, filePaths []string) (string, []ChecksumEntry, error) {
	var sb strings.Builder
	var entries []ChecksumEntry

	for _, relPath := range filePaths {
		fullPath := filepath.Join(baseDir, relPath)
		hash, err := ComputeSHA256(fullPath)
		if err != nil {
			return "", nil, err
		}

		cleanRel := filepath.ToSlash(relPath)
		sb.WriteString(fmt.Sprintf("%s  %s\n", hash, cleanRel))
		entries = append(entries, ChecksumEntry{
			FilePath: cleanRel,
			SHA256:   hash,
		})
	}

	return sb.String(), entries, nil
}

// VerifyChecksumManifest verifies all files in baseDir against a SHA256SUMS manifest.
func VerifyChecksumManifest(baseDir string, manifestContent string) error {
	lines := strings.Split(manifestContent, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		expectedHash := parts[0]
		relPath := parts[1]
		fullPath := filepath.Join(baseDir, filepath.FromSlash(relPath))

		actualHash, err := ComputeSHA256(fullPath)
		if err != nil {
			return fmt.Errorf("verification error for %s: %w", relPath, err)
		}

		if !strings.EqualFold(actualHash, expectedHash) {
			return fmt.Errorf("%w: file %s (expected %s, got %s)", ErrChecksumMismatch, relPath, expectedHash, actualHash)
		}
	}
	return nil
}
