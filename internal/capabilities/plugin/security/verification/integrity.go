package verification

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var (
	// ErrDigestMismatch indicates the computed hash did not match the expected hash.
	ErrDigestMismatch = errors.New("integrity verification failed: digest mismatch")
	// ErrFileNotFound indicates a referenced file was missing during integrity calculation.
	ErrFileNotFound = errors.New("integrity verification failed: file not found")
)

// ComputeBytesDigest computes the lowercase hex-encoded SHA-256 digest of an in-memory byte slice.
func ComputeBytesDigest(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// ComputeFileDigest computes the lowercase hex-encoded SHA-256 digest of a file on disk.
func ComputeFileDigest(filePath string) (string, error) {
	cleanPath := filepath.Clean(filePath)
	f, err := os.Open(cleanPath)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrFileNotFound, err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("failed to read file for digest calculation: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// VerifyFileDigest checks if a file's SHA-256 digest matches the expected hex string using constant-time comparison.
func VerifyFileDigest(filePath, expectedDigest string) error {
	actual, err := ComputeFileDigest(filePath)
	if err != nil {
		return err
	}

	expectedNorm := strings.ToLower(strings.TrimSpace(expectedDigest))
	actualNorm := strings.ToLower(strings.TrimSpace(actual))

	if subtle.ConstantTimeCompare([]byte(expectedNorm), []byte(actualNorm)) != 1 {
		return fmt.Errorf("%w: expected %s, got %s for file %s", ErrDigestMismatch, expectedDigest, actual, filepath.Base(filePath))
	}

	return nil
}

// ChecksumManifest represents a map of relative file paths to their expected SHA-256 digests.
type ChecksumManifest map[string]string

// ParseChecksums parses standard sha256sum formatted text (e.g. "<hash>  <filename>").
func ParseChecksums(content string) ChecksumManifest {
	manifest := make(ChecksumManifest)
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Split by whitespace: could be two spaces or one space + asterisk
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			hash := strings.ToLower(parts[0])
			path := strings.TrimPrefix(parts[1], "*")
			path = filepath.Clean(filepath.ToSlash(path))
			manifest[path] = hash
		}
	}
	return manifest
}

// VerifyPackageIntegrity validates all files listed in the ChecksumManifest against the target directory.
func VerifyPackageIntegrity(dir string, manifest ChecksumManifest) error {
	if len(manifest) == 0 {
		return errors.New("integrity verification failed: empty checksum manifest")
	}

	cleanDir := filepath.Clean(dir)
	for relPath, expectedHash := range manifest {
		// Prevent path traversal escape in relative path
		if strings.HasPrefix(relPath, "..") || filepath.IsAbs(relPath) {
			return fmt.Errorf("illegal relative path in checksum manifest: %s", relPath)
		}

		fullPath := filepath.Join(cleanDir, filepath.FromSlash(relPath))
		if err := VerifyFileDigest(fullPath, expectedHash); err != nil {
			return err
		}
	}

	return nil
}
