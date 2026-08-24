package discovery

import "errors"

var (
	// ErrNilDiscoverer indicates an operation was attempted on a nil Discoverer instance.
	ErrNilDiscoverer = errors.New("discovery: discoverer instance is nil")

	// ErrNilRegistry indicates a nil Language Registry was supplied.
	ErrNilRegistry = errors.New("discovery: language registry is nil")

	// ErrNilRepository indicates a nil Repository instance was supplied.
	ErrNilRepository = errors.New("discovery: repository is nil")

	// ErrPathEmpty indicates an empty path string was supplied.
	ErrPathEmpty = errors.New("discovery: path is empty")

	// ErrPathNotFound indicates the target repository root path does not exist.
	ErrPathNotFound = errors.New("discovery: repository path does not exist")

	// ErrNotDirectory indicates the target repository path exists but is not a directory.
	ErrNotDirectory = errors.New("discovery: repository path is not a directory")

	// ErrBoundaryViolation indicates a filesystem path resolved outside the authorized repository boundary.
	ErrBoundaryViolation = errors.New("discovery: path escapes repository boundary")

	// ErrTraversalLimitExceeded indicates a traversal limit (depth or file count) was exceeded.
	ErrTraversalLimitExceeded = errors.New("discovery: traversal limit exceeded")

	// ErrDiscoveryFailed indicates an unrecoverable failure occurred during repository discovery.
	ErrDiscoveryFailed = errors.New("discovery: repository discovery failed")
)
