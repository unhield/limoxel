package language

import "errors"

var (
	// ErrNilAnalyzer indicates an operation was attempted on a nil Analyzer instance.
	ErrNilAnalyzer = errors.New("language: analyzer instance is nil")

	// ErrNilDiscoverer indicates a nil Discoverer was supplied.
	ErrNilDiscoverer = errors.New("language: discoverer instance is nil")

	// ErrNilDiscoveryResult indicates a nil Discovery Result was supplied.
	ErrNilDiscoveryResult = errors.New("language: discovery result is nil")

	// ErrNilRepository indicates a nil Repository was supplied.
	ErrNilRepository = errors.New("language: repository is nil")

	// ErrPathEmpty indicates an empty path string was supplied.
	ErrPathEmpty = errors.New("language: path is empty")

	// ErrPathNotFound indicates the target repository root path does not exist.
	ErrPathNotFound = errors.New("language: repository path does not exist")

	// ErrNotDirectory indicates the target repository path exists but is not a directory.
	ErrNotDirectory = errors.New("language: repository path is not a directory")

	// ErrAnalysisFailed indicates an unrecoverable failure occurred during structural analysis.
	ErrAnalysisFailed = errors.New("language: repository structural analysis failed")
)
