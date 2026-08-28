package contracts

import (
	"context"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/sdk/lifecycle"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

// FileInfo represents public information about a repository file.
type FileInfo struct {
	Path      string `json:"path"`
	Name      string `json:"name"`
	Extension string `json:"extension,omitempty"`
	Size      int64  `json:"size"`
	Language  string `json:"language,omitempty"`
	Package   string `json:"package,omitempty"`
	Lines     int    `json:"lines,omitempty"`
	Hash      string `json:"hash,omitempty"`
}

// FileMetadata represents detailed metadata and classification attributes of a file.
type FileMetadata struct {
	File         FileInfo          `json:"file"`
	IsTest       bool              `json:"is_test"`
	IsGenerated  bool              `json:"is_generated"`
	IsVendor     bool              `json:"is_vendor"`
	LastModified time.Time         `json:"last_modified,omitempty"`
	Properties   map[string]string `json:"properties,omitempty"`
}

// FileIndexStatus represents the indexing and processing state of a file.
type FileIndexStatus struct {
	Path        string    `json:"path"`
	IsIndexed   bool      `json:"is_indexed"`
	IndexedAt   time.Time `json:"indexed_at,omitempty"`
	SymbolCount int       `json:"symbol_count"`
	Error       string    `json:"error,omitempty"`
}

// FileRelationship represents a semantic connection between two repository files.
type FileRelationship struct {
	SourcePath string `json:"source_path"`
	TargetPath string `json:"target_path"`
	Kind       string `json:"kind"`
	Weight     int    `json:"weight,omitempty"`
}

// FileFilter provides criteria for filtering file discovery results.
type FileFilter struct {
	Language     string `json:"language,omitempty"`
	Package      string `json:"package,omitempty"`
	Pattern      string `json:"pattern,omitempty"`
	IncludeTests bool   `json:"include_tests"`
}

// FileContract defines the public contract for file discovery, lookup, metadata, indexing status, and relationships.
type FileContract interface {
	Contract
	DiscoverFiles(ctx context.Context, filter FileFilter, opts PaginationOptions) ([]FileInfo, error)
	LookupFile(ctx context.Context, path string) (*FileInfo, error)
	GetFileMetadata(ctx context.Context, path string) (*FileMetadata, error)
	GetFileIndexStatus(ctx context.Context, path string) (*FileIndexStatus, error)
	GetFileRelationships(ctx context.Context, path string) ([]FileRelationship, error)
}

// DefaultFileContractMetadata returns default contract descriptor for File operations.
func DefaultFileContractMetadata() BaseContract {
	return NewBaseContract(
		"FileContract",
		lifecycle.CapabilityRepository,
		version.SemVer{Major: 1, Minor: 0, Patch: 0},
		lifecycle.StateSupported,
		"Provides public file discovery, metadata, lookup, indexing status, and relationship queries.",
	)
}
