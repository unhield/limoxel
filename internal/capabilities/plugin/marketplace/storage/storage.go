package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	pubmarket "github.com/unhield/limoxel/plugin/marketplace"
)

var (
	// ErrNotFound indicates the requested entity does not exist in storage.
	ErrNotFound = errors.New("marketplace storage: entity not found")
	// ErrDigestMismatch indicates the written artifact does not match declared digest.
	ErrDigestMismatch = errors.New("marketplace storage: artifact digest mismatch")
	// ErrInvalidInput indicates bad or empty parameter.
	ErrInvalidInput = errors.New("marketplace storage: invalid input")
)

// Storage abstracts persistent marketplace data operations.
type Storage interface {
	SavePluginDetail(detail pubmarket.PluginDetail) error
	GetPluginDetail(id string) (*pubmarket.PluginDetail, error)
	ListPluginDetails() ([]pubmarket.PluginDetail, error)
	DeletePlugin(id string) error

	SavePublisher(pub pubmarket.PublisherProfile) error
	GetPublisher(id string) (*pubmarket.PublisherProfile, error)
	ListPublishers() ([]pubmarket.PublisherProfile, error)

	SaveArtifact(expectedDigest string, r io.Reader) (string, int64, error)
	GetArtifact(digest string) (io.ReadCloser, int64, error)
	ArtifactExists(digest string) bool

	SaveReviews(pluginID string, reviews []pubmarket.Review) error
	GetReviews(pluginID string) ([]pubmarket.Review, error)

	SaveRatings(pluginID string, ratings map[string]int) error
	GetRatings(pluginID string) (map[string]int, error)
}

// FileStorage implements Storage using local filesystem persistence with atomic writes.
type FileStorage struct {
	mu      sync.RWMutex
	baseDir string
}

// NewFileStorage constructs an initialized FileStorage rooted at baseDir.
func NewFileStorage(baseDir string) (*FileStorage, error) {
	cleanDir := filepath.Clean(baseDir)
	if cleanDir == "" || cleanDir == "." {
		cleanDir = "marketplace_data"
	}

	dirs := []string{
		filepath.Join(cleanDir, "metadata", "plugins"),
		filepath.Join(cleanDir, "metadata", "publishers"),
		filepath.Join(cleanDir, "artifacts"),
		filepath.Join(cleanDir, "community", "reviews"),
		filepath.Join(cleanDir, "community", "ratings"),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0750); err != nil {
			return nil, fmt.Errorf("failed to create storage directory %s: %w", d, err)
		}
	}

	return &FileStorage{
		baseDir: cleanDir,
	}, nil
}

// BaseDir returns the root storage directory path.
func (s *FileStorage) BaseDir() string {
	return s.baseDir
}

func validateIdentifier(id string) error {
	if id == "" {
		return fmt.Errorf("%w: identifier cannot be empty", ErrInvalidInput)
	}
	if len(id) > 128 {
		return fmt.Errorf("%w: identifier exceeds maximum length of 128", ErrInvalidInput)
	}

	// Must start with an alphanumeric character
	first := id[0]
	if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || (first >= '0' && first <= '9')) {
		return fmt.Errorf("%w: identifier must start with an alphanumeric character", ErrInvalidInput)
	}

	// Must not end with a dot (prevents Windows trailing dot stripping)
	if id[len(id)-1] == '.' {
		return fmt.Errorf("%w: identifier cannot end with a dot", ErrInvalidInput)
	}

	// Must not contain consecutive dots
	if strings.Contains(id, "..") {
		return fmt.Errorf("%w: identifier cannot contain '..'", ErrInvalidInput)
	}

	// Check each character against allowed set
	for i := 0; i < len(id); i++ {
		c := id[i]
		switch {
		case (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9'):
		case c == '-' || c == '_' || c == '.':
		default:
			return fmt.Errorf("%w: identifier contains invalid character %q", ErrInvalidInput, c)
		}
	}

	// Lexical locality and containment check
	if !filepath.IsLocal(id) {
		return fmt.Errorf("%w: identifier is not a local path component", ErrInvalidInput)
	}
	if filepath.Base(id) != id {
		return fmt.Errorf("%w: identifier cannot contain path separators", ErrInvalidInput)
	}

	return nil
}

func validateDigest(digest string) error {
	if len(digest) != 64 {
		return fmt.Errorf("%w: artifact digest must be exactly 64 hex characters", ErrInvalidInput)
	}
	for i := 0; i < len(digest); i++ {
		c := digest[i]
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			continue
		}
		return fmt.Errorf("%w: artifact digest contains non-hexadecimal character %q", ErrInvalidInput, c)
	}
	if !filepath.IsLocal(digest) {
		return fmt.Errorf("%w: artifact digest is not a local path component", ErrInvalidInput)
	}
	if filepath.Base(digest) != digest {
		return fmt.Errorf("%w: artifact digest cannot contain path separators", ErrInvalidInput)
	}
	return nil
}

func (s *FileStorage) resolveStoragePath(subDir, filename string) (string, string, error) {
	relPath := filepath.Join(subDir, filename)
	if !filepath.IsLocal(relPath) {
		return "", "", fmt.Errorf("%w: storage path %q is not local", ErrInvalidInput, relPath)
	}
	targetPath := filepath.Join(s.baseDir, relPath)
	return relPath, targetPath, nil
}

// SavePluginDetail atomically writes plugin detail JSON to disk.
func (s *FileStorage) SavePluginDetail(detail pubmarket.PluginDetail) error {
	if err := validateIdentifier(detail.ID); err != nil {
		return fmt.Errorf("%w: invalid plugin ID: %v", ErrInvalidInput, err)
	}

	relPath, _, err := s.resolveStoragePath(filepath.Join("metadata", "plugins"), fmt.Sprintf("%s.json", detail.ID))
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.atomicWriteJSON(relPath, detail)
}

// GetPluginDetail reads and deserializes a plugin detail file.
func (s *FileStorage) GetPluginDetail(id string) (*pubmarket.PluginDetail, error) {
	if err := validateIdentifier(id); err != nil {
		return nil, fmt.Errorf("%w: invalid plugin ID: %v", ErrInvalidInput, err)
	}

	_, target, err := s.resolveStoragePath(filepath.Join("metadata", "plugins"), fmt.Sprintf("%s.json", id))
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(target)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	var detail pubmarket.PluginDetail
	if err := json.Unmarshal(data, &detail); err != nil {
		return nil, fmt.Errorf("failed to parse plugin detail for %s: %w", id, err)
	}

	return &detail, nil
}

// ListPluginDetails returns all stored plugin details.
func (s *FileStorage) ListPluginDetails() ([]pubmarket.PluginDetail, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pluginsDir := filepath.Join(s.baseDir, "metadata", "plugins")
	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		return nil, err
	}

	var results []pubmarket.PluginDetail
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		if !filepath.IsLocal(e.Name()) {
			continue
		}
		filePath := filepath.Join(pluginsDir, e.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		var detail pubmarket.PluginDetail
		if err := json.Unmarshal(data, &detail); err == nil {
			results = append(results, detail)
		}
	}

	return results, nil
}

// DeletePlugin removes metadata, reviews, and ratings for a plugin.
func (s *FileStorage) DeletePlugin(id string) error {
	if err := validateIdentifier(id); err != nil {
		return fmt.Errorf("%w: invalid plugin ID: %v", ErrInvalidInput, err)
	}

	_, pluginTarget, err := s.resolveStoragePath(filepath.Join("metadata", "plugins"), fmt.Sprintf("%s.json", id))
	if err != nil {
		return err
	}
	_, reviewsTarget, err := s.resolveStoragePath(filepath.Join("community", "reviews"), fmt.Sprintf("%s.json", id))
	if err != nil {
		return err
	}
	_, ratingsTarget, err := s.resolveStoragePath(filepath.Join("community", "ratings"), fmt.Sprintf("%s.json", id))
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	_ = os.Remove(pluginTarget)
	_ = os.Remove(reviewsTarget)
	_ = os.Remove(ratingsTarget)
	return nil
}

// SavePublisher writes publisher profile JSON to disk.
func (s *FileStorage) SavePublisher(pub pubmarket.PublisherProfile) error {
	if err := validateIdentifier(pub.ID); err != nil {
		return fmt.Errorf("%w: invalid publisher ID: %v", ErrInvalidInput, err)
	}

	relPath, _, err := s.resolveStoragePath(filepath.Join("metadata", "publishers"), fmt.Sprintf("%s.json", pub.ID))
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.atomicWriteJSON(relPath, pub)
}

// GetPublisher retrieves publisher profile by ID.
func (s *FileStorage) GetPublisher(id string) (*pubmarket.PublisherProfile, error) {
	if err := validateIdentifier(id); err != nil {
		return nil, fmt.Errorf("%w: invalid publisher ID: %v", ErrInvalidInput, err)
	}

	_, target, err := s.resolveStoragePath(filepath.Join("metadata", "publishers"), fmt.Sprintf("%s.json", id))
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(target)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	var pub pubmarket.PublisherProfile
	if err := json.Unmarshal(data, &pub); err != nil {
		return nil, err
	}
	return &pub, nil
}

// ListPublishers returns all stored publisher profiles.
func (s *FileStorage) ListPublishers() ([]pubmarket.PublisherProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dir := filepath.Join(s.baseDir, "metadata", "publishers")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var results []pubmarket.PublisherProfile
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		if !filepath.IsLocal(e.Name()) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var pub pubmarket.PublisherProfile
		if err := json.Unmarshal(data, &pub); err == nil {
			results = append(results, pub)
		}
	}
	return results, nil
}

// SaveArtifact streams package contents into an immutable content-addressed artifact file.
func (s *FileStorage) SaveArtifact(expectedDigest string, r io.Reader) (string, int64, error) {
	if expectedDigest != "" {
		if err := validateDigest(expectedDigest); err != nil {
			return "", 0, fmt.Errorf("%w: invalid expected digest: %v", ErrInvalidInput, err)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	artifactsDir := filepath.Join(s.baseDir, "artifacts")
	tmpFile, err := os.CreateTemp(artifactsDir, "upload-*.tmp")
	if err != nil {
		return "", 0, fmt.Errorf("failed to create temporary upload file: %w", err)
	}
	tmpPath := tmpFile.Name()

	hasher := sha256.New()
	writer := io.MultiWriter(tmpFile, hasher)

	written, copyErr := io.Copy(writer, r)
	closeErr := tmpFile.Close()

	if copyErr != nil {
		_ = os.Remove(tmpPath)
		return "", 0, fmt.Errorf("failed during artifact streaming: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return "", 0, fmt.Errorf("failed to close temporary upload file: %w", closeErr)
	}

	actualDigest := hex.EncodeToString(hasher.Sum(nil))
	if err := validateDigest(actualDigest); err != nil {
		_ = os.Remove(tmpPath)
		return "", 0, fmt.Errorf("%w: computed digest invalid: %v", ErrInvalidInput, err)
	}

	if expectedDigest != "" && actualDigest != expectedDigest {
		_ = os.Remove(tmpPath)
		return "", 0, fmt.Errorf("%w: declared %s, computed %s", ErrDigestMismatch, expectedDigest, actualDigest)
	}

	relPath, finalPath, err := s.resolveStoragePath("artifacts", fmt.Sprintf("%s.tar.gz", actualDigest))
	if err != nil {
		_ = os.Remove(tmpPath)
		return "", 0, err
	}

	lockErr := WithStorageLock(s.baseDir, relPath, func() error {
		if _, statErr := os.Stat(finalPath); statErr == nil {
			// Artifact already present in immutable storage
			_ = os.Remove(tmpPath)
			return nil
		}
		return os.Rename(tmpPath, finalPath)
	})
	if lockErr != nil {
		_ = os.Remove(tmpPath)
		return "", 0, fmt.Errorf("failed to finalize artifact: %w", lockErr)
	}

	return actualDigest, written, nil
}

// GetArtifact opens an artifact for reading.
func (s *FileStorage) GetArtifact(digest string) (io.ReadCloser, int64, error) {
	if err := validateDigest(digest); err != nil {
		return nil, 0, fmt.Errorf("%w: invalid digest: %v", ErrInvalidInput, err)
	}

	_, target, err := s.resolveStoragePath("artifacts", fmt.Sprintf("%s.tar.gz", digest))
	if err != nil {
		return nil, 0, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	info, err := os.Stat(target)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, ErrNotFound
		}
		return nil, 0, err
	}

	f, err := os.Open(target)
	if err != nil {
		return nil, 0, err
	}

	return f, info.Size(), nil
}

// ArtifactExists checks if an artifact with digest exists on disk.
func (s *FileStorage) ArtifactExists(digest string) bool {
	if err := validateDigest(digest); err != nil {
		return false
	}

	_, target, err := s.resolveStoragePath("artifacts", fmt.Sprintf("%s.tar.gz", digest))
	if err != nil {
		return false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	_, err = os.Stat(target)
	return err == nil
}

// SaveReviews writes the list of reviews for a plugin.
func (s *FileStorage) SaveReviews(pluginID string, reviews []pubmarket.Review) error {
	if err := validateIdentifier(pluginID); err != nil {
		return fmt.Errorf("%w: invalid plugin ID: %v", ErrInvalidInput, err)
	}

	relPath, _, err := s.resolveStoragePath(filepath.Join("community", "reviews"), fmt.Sprintf("%s.json", pluginID))
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.atomicWriteJSON(relPath, reviews)
}

// GetReviews retrieves stored reviews for a plugin.
func (s *FileStorage) GetReviews(pluginID string) ([]pubmarket.Review, error) {
	if err := validateIdentifier(pluginID); err != nil {
		return nil, fmt.Errorf("%w: invalid plugin ID: %v", ErrInvalidInput, err)
	}

	_, target, err := s.resolveStoragePath(filepath.Join("community", "reviews"), fmt.Sprintf("%s.json", pluginID))
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(target)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No reviews yet
		}
		return nil, err
	}

	var reviews []pubmarket.Review
	if err := json.Unmarshal(data, &reviews); err != nil {
		return nil, err
	}
	return reviews, nil
}

// SaveRatings writes the user rating map for a plugin.
func (s *FileStorage) SaveRatings(pluginID string, ratings map[string]int) error {
	if err := validateIdentifier(pluginID); err != nil {
		return fmt.Errorf("%w: invalid plugin ID: %v", ErrInvalidInput, err)
	}

	relPath, _, err := s.resolveStoragePath(filepath.Join("community", "ratings"), fmt.Sprintf("%s.json", pluginID))
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return s.atomicWriteJSON(relPath, ratings)
}

// GetRatings retrieves stored user ratings for a plugin.
func (s *FileStorage) GetRatings(pluginID string) (map[string]int, error) {
	if err := validateIdentifier(pluginID); err != nil {
		return nil, fmt.Errorf("%w: invalid plugin ID: %v", ErrInvalidInput, err)
	}

	_, target, err := s.resolveStoragePath(filepath.Join("community", "ratings"), fmt.Sprintf("%s.json", pluginID))
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(target)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]int), nil
		}
		return nil, err
	}

	var ratings map[string]int
	if err := json.Unmarshal(data, &ratings); err != nil {
		return nil, err
	}
	return ratings, nil
}

func (s *FileStorage) atomicWriteJSON(relPath string, v any) error {
	if !filepath.IsLocal(relPath) {
		return fmt.Errorf("%w: storage path %q is not local", ErrInvalidInput, relPath)
	}

	targetPath := filepath.Join(s.baseDir, relPath)
	dir := filepath.Dir(targetPath)

	return WithStorageLock(s.baseDir, relPath, func() error {
		data, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}

		tmpFile, err := os.CreateTemp(dir, "write-*.tmp")
		if err != nil {
			return fmt.Errorf("failed to create temporary file: %w", err)
		}
		tmpPath := tmpFile.Name()

		if _, err := tmpFile.Write(data); err != nil {
			_ = tmpFile.Close()
			_ = os.Remove(tmpPath)
			return fmt.Errorf("failed to write data: %w", err)
		}
		if err := tmpFile.Close(); err != nil {
			_ = os.Remove(tmpPath)
			return err
		}

		if err := os.Rename(tmpPath, targetPath); err != nil {
			_ = os.Remove(tmpPath)
			return fmt.Errorf("failed to atomic rename to %s: %w", targetPath, err)
		}
		return nil
	})
}
