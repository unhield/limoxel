package dependency

import "errors"

var (
	// ErrNilAnalyzer indicates an operation was attempted on a nil Analyzer instance.
	ErrNilAnalyzer = errors.New("dependency: analyzer instance is nil")

	// ErrNilDiscoverer indicates a nil Discoverer was supplied.
	ErrNilDiscoverer = errors.New("dependency: discoverer instance is nil")

	// ErrNilDiscoveryResult indicates a nil Discovery Result was supplied.
	ErrNilDiscoveryResult = errors.New("dependency: discovery result is nil")

	// ErrNilRepository indicates a nil Repository was supplied.
	ErrNilRepository = errors.New("dependency: repository is nil")

	// ErrPathEmpty indicates an empty path string was supplied.
	ErrPathEmpty = errors.New("dependency: path is empty")

	// ErrPathNotFound indicates the target repository root path does not exist.
	ErrPathNotFound = errors.New("dependency: repository path does not exist")

	// ErrNotDirectory indicates the target repository path exists but is not a directory.
	ErrNotDirectory = errors.New("dependency: repository path is not a directory")

	// ErrAnalysisFailed indicates an unrecoverable failure occurred during dependency analysis.
	ErrAnalysisFailed = errors.New("dependency: repository dependency analysis failed")
)
