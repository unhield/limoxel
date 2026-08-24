package symbol

import "errors"

var (
	// ErrNilEngine indicates an operation was attempted on a nil Engine instance.
	ErrNilEngine = errors.New("symbol: engine instance is nil")

	// ErrNilDiscoverer indicates a nil Discoverer was supplied.
	ErrNilDiscoverer = errors.New("symbol: discoverer instance is nil")

	// ErrNilDiscoveryResult indicates a nil Discovery Result was supplied.
	ErrNilDiscoveryResult = errors.New("symbol: discovery result is nil")

	// ErrNilIndexModel indicates a nil IndexModel was supplied.
	ErrNilIndexModel = errors.New("symbol: index model is nil")

	// ErrNilRepository indicates a nil Repository was supplied.
	ErrNilRepository = errors.New("symbol: repository is nil")

	// ErrPathEmpty indicates an empty path string was supplied.
	ErrPathEmpty = errors.New("symbol: path is empty")

	// ErrPathNotFound indicates the target repository root path does not exist.
	ErrPathNotFound = errors.New("symbol: repository path does not exist")

	// ErrNotDirectory indicates the target repository path exists but is not a directory.
	ErrNotDirectory = errors.New("symbol: repository path is not a directory")

	// ErrParsingFailed indicates an unrecoverable failure occurred during symbol extraction.
	ErrParsingFailed = errors.New("symbol: repository AST parsing and symbol extraction failed")
)
