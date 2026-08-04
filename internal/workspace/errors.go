package workspace

import "errors"

var (
	// ErrNilWorkspace indicates an operation was attempted on a nil Workspace instance.
	ErrNilWorkspace = errors.New("workspace: instance is nil")

	// ErrInvalidID indicates the workspace identifier is empty or invalid.
	ErrInvalidID = errors.New("workspace: invalid or empty ID")

	// ErrInvalidRoot indicates the workspace root path is empty or invalid.
	ErrInvalidRoot = errors.New("workspace: root path is invalid or empty")

	// ErrRootNotFound indicates the workspace root directory does not exist on the filesystem.
	ErrRootNotFound = errors.New("workspace: root directory does not exist")

	// ErrNotDirectory indicates the workspace root path exists but is not a directory.
	ErrNotDirectory = errors.New("workspace: root path is not a directory")
)
