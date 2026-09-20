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

// SavePluginDetail atomically writes plugin detail JSON to disk.
func (s *FileStorage) SavePluginDetail(detail pubmarket.PluginDetail) error {
	if detail.ID == "" {
		return fmt.Errorf("%w: missing plugin ID", ErrInvalidInput)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	target := filepath.Join(s.baseDir, "metadata", "plugins", fmt.Sprintf("%s.json", detail.ID))
	return atomicWriteJSON(target, detail)
}

// GetPluginDetail reads and deserializes a plugin detail file.
func (s *FileStorage) GetPluginDetail(id string) (*pubmarket.PluginDetail, error) {
	if id == "" {
		return nil, ErrInvalidInput
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	target := filepath.Join(s.baseDir, "metadata", "plugins", fmt.Sprintf("%s.json", id))
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
	s.mu.Lock()
	defer s.mu.Unlock()

	_ = os.Remove(filepath.Join(s.baseDir, "metadata", "plugins", fmt.Sprintf("%s.json", id)))
	_ = os.Remove(filepath.Join(s.baseDir, "community", "reviews", fmt.Sprintf("%s.json", id)))
	_ = os.Remove(filepath.Join(s.baseDir, "community", "ratings", fmt.Sprintf("%s.json", id)))
	return nil
}

// SavePublisher writes publisher profile JSON to disk.
func (s *FileStorage) SavePublisher(pub pubmarket.PublisherProfile) error {
	if pub.ID == "" {
		return fmt.Errorf("%w: missing publisher ID", ErrInvalidInput)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	target := filepath.Join(s.baseDir, "metadata", "publishers", fmt.Sprintf("%s.json", pub.ID))
	return atomicWriteJSON(target, pub)
}

// GetPublisher retrieves publisher profile by ID.
func (s *FileStorage) GetPublisher(id string) (*pubmarket.PublisherProfile, error) {
	if id == "" {
		return nil, ErrInvalidInput
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	target := filepath.Join(s.baseDir, "metadata", "publishers", fmt.Sprintf("%s.json", id))
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
	s.mu.Lock()
	defer s.mu.Unlock()

	tmpFile, err := os.CreateTemp(filepath.Join(s.baseDir, "artifacts"), "upload-*.tmp")
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
	if expectedDigest != "" && actualDigest != expectedDigest {
		_ = os.Remove(tmpPath)
		return "", 0, fmt.Errorf("%w: declared %s, computed %s", ErrDigestMismatch, expectedDigest, actualDigest)
	}

	finalPath := filepath.Join(s.baseDir, "artifacts", fmt.Sprintf("%s.tar.gz", actualDigest))
	lockErr := WithLock(finalPath, func() error {
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
	if digest == "" {
		return nil, 0, ErrInvalidInput
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	target := filepath.Join(s.baseDir, "artifacts", fmt.Sprintf("%s.tar.gz", digest))
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
	if digest == "" {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	target := filepath.Join(s.baseDir, "artifacts", fmt.Sprintf("%s.tar.gz", digest))
	_, err := os.Stat(target)
	return err == nil
}

// SaveReviews writes the list of reviews for a plugin.
func (s *FileStorage) SaveReviews(pluginID string, reviews []pubmarket.Review) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	target := filepath.Join(s.baseDir, "community", "reviews", fmt.Sprintf("%s.json", pluginID))
	return atomicWriteJSON(target, reviews)
}

// GetReviews retrieves stored reviews for a plugin.
func (s *FileStorage) GetReviews(pluginID string) ([]pubmarket.Review, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	target := filepath.Join(s.baseDir, "community", "reviews", fmt.Sprintf("%s.json", pluginID))
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
	s.mu.Lock()
	defer s.mu.Unlock()

	target := filepath.Join(s.baseDir, "community", "ratings", fmt.Sprintf("%s.json", pluginID))
	return atomicWriteJSON(target, ratings)
}

// GetRatings retrieves stored user ratings for a plugin.
func (s *FileStorage) GetRatings(pluginID string) (map[string]int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	target := filepath.Join(s.baseDir, "community", "ratings", fmt.Sprintf("%s.json", pluginID))
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

func atomicWriteJSON(targetPath string, v any) error {
	return WithLock(targetPath, func() error {
		data, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}

		dir := filepath.Dir(targetPath)
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
