package project

import "errors"

var (
	// ErrNilProject indicates an operation was attempted on a nil Project instance.
	ErrNilProject = errors.New("project: instance is nil")

	// ErrNilWorkspace indicates an attempt to create a Project with a nil Workspace.
	ErrNilWorkspace = errors.New("project: workspace instance is nil")

	// ErrInvalidName indicates the specified project name is empty or invalid.
	ErrInvalidName = errors.New("project: name is invalid or empty")

	// ErrInvalidProjectRoot indicates the specified project root path is empty or invalid.
	ErrInvalidProjectRoot = errors.New("project: project root path is invalid or empty")

	// ErrProjectRootOutsideWorkspace indicates the project root is located outside its owning Workspace root.
	ErrProjectRootOutsideWorkspace = errors.New("project: project root is outside owning workspace")
)
