package filesystem

import "errors"

var (
	// ErrNilDiscoverer indicates an operation was attempted on a nil Discoverer instance.
	ErrNilDiscoverer = errors.New("filesystem: discoverer instance is nil")

	// ErrRootNotFound indicates the target root directory does not exist on the filesystem.
	ErrRootNotFound = errors.New("filesystem: root directory does not exist")

	// ErrNotDirectory indicates the target root path exists but is not a directory.
	ErrNotDirectory = errors.New("filesystem: root path is not a directory")

	// ErrDiscoveryFailed indicates an unrecoverable failure during filesystem traversal.
	ErrDiscoveryFailed = errors.New("filesystem: discovery traversal failed")

	// ErrNilIgnorer indicates an operation was attempted on a nil Ignorer instance.
	ErrNilIgnorer = errors.New("filesystem: ignorer instance is nil")

	// ErrInvalidRule indicates an empty or invalid ignore rule string was provided.
	ErrInvalidRule = errors.New("filesystem: invalid ignore rule")

	// ErrNilFilesystem indicates an operation was attempted on a nil Filesystem instance.
	ErrNilFilesystem = errors.New("filesystem: instance is nil")

	// ErrPathEmpty indicates a path argument was empty.
	ErrPathEmpty = errors.New("filesystem: path is empty")

	// ErrFileNotFound indicates the specified file or directory was not found.
	ErrFileNotFound = errors.New("filesystem: file or directory not found")

	// ErrNilService indicates an operation was attempted on a nil FileService instance.
	ErrNilService = errors.New("filesystem: file service instance is nil")

	// ErrSourceNotFound indicates the source file for a copy operation was not found.
	ErrSourceNotFound = errors.New("filesystem: source file not found")
)
