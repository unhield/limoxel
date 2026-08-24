package indexing

import "errors"

var (
	// ErrNilIndexer indicates an operation was attempted on a nil Indexer instance.
	ErrNilIndexer = errors.New("indexing: indexer instance is nil")

	// ErrNilDiscoverer indicates a nil Discoverer was supplied.
	ErrNilDiscoverer = errors.New("indexing: discoverer instance is nil")

	// ErrNilDiscoveryResult indicates a nil Discovery Result was supplied.
	ErrNilDiscoveryResult = errors.New("indexing: discovery result is nil")

	// ErrNilRepository indicates a nil Repository was supplied.
	ErrNilRepository = errors.New("indexing: repository is nil")

	// ErrNilIndexModel indicates a nil IndexModel was supplied.
	ErrNilIndexModel = errors.New("indexing: index model is nil")

	// ErrPathEmpty indicates an empty path string was supplied.
	ErrPathEmpty = errors.New("indexing: path is empty")

	// ErrPathNotFound indicates the target repository root path does not exist.
	ErrPathNotFound = errors.New("indexing: repository path does not exist")

	// ErrNotDirectory indicates the target repository path exists but is not a directory.
	ErrNotDirectory = errors.New("indexing: repository path is not a directory")

	// ErrIndexingFailed indicates an unrecoverable failure occurred during source code indexing.
	ErrIndexingFailed = errors.New("indexing: repository indexing failed")

	// ErrIncompatibleSchema indicates the serialized index schema version is incompatible with current engine.
	ErrIncompatibleSchema = errors.New("indexing: incompatible index schema version")

	// ErrCorruptedIndex indicates the serialized index payload is malformed or corrupted.
	ErrCorruptedIndex = errors.New("indexing: corrupted or malformed index data")
)
