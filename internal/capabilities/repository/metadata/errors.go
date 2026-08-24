package metadata

import "errors"

var (
	// ErrNilCollector indicates an operation was attempted on a nil Collector instance.
	ErrNilCollector = errors.New("metadata: collector instance is nil")

	// ErrNilDiscoverer indicates a nil Discoverer instance was supplied.
	ErrNilDiscoverer = errors.New("metadata: discoverer instance is nil")

	// ErrNilDiscoveryResult indicates a nil Discovery Result was supplied.
	ErrNilDiscoveryResult = errors.New("metadata: discovery result is nil")

	// ErrNilRepository indicates a nil Repository instance was supplied.
	ErrNilRepository = errors.New("metadata: repository is nil")

	// ErrPathEmpty indicates an empty path string was supplied.
	ErrPathEmpty = errors.New("metadata: path is empty")

	// ErrPathNotFound indicates the target repository root path does not exist.
	ErrPathNotFound = errors.New("metadata: repository path does not exist")

	// ErrNotDirectory indicates the target repository path exists but is not a directory.
	ErrNotDirectory = errors.New("metadata: repository path is not a directory")

	// ErrMetadataCollectionFailed indicates an unrecoverable failure during metadata collection.
	ErrMetadataCollectionFailed = errors.New("metadata: repository metadata collection failed")
)
