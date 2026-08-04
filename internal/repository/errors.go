package repository

import "errors"

var (
	// ErrNilRepository indicates an operation was attempted on a nil Repository instance.
	ErrNilRepository = errors.New("repository: instance is nil")

	// ErrNilProject indicates an attempt to create a Repository with a nil Project.
	ErrNilProject = errors.New("repository: project instance is nil")

	// ErrInvalidName indicates the specified repository name is empty or invalid.
	ErrInvalidName = errors.New("repository: name is invalid or empty")

	// ErrInvalidRepositoryRoot indicates the specified repository root path is empty or invalid.
	ErrInvalidRepositoryRoot = errors.New("repository: repository root path is invalid or empty")

	// ErrRepositoryRootOutsideProject indicates the repository root is located outside its owning Project root.
	ErrRepositoryRootOutsideProject = errors.New("repository: repository root is outside owning project")
)
