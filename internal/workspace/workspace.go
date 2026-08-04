package workspace

import (
	"fmt"
	"os"
	"path/filepath"
)

// Workspace represents an immutable, validated workspace root location with an identity and resources.
type Workspace struct {
	id        ID
	root      string
	resources *Resources
}

// New creates and validates a new immutable Workspace for the given id string and rootPath.
// It resolves rootPath to an absolute, cleaned path and verifies that it exists and is a directory.
func New(idStr string, rootPath string) (*Workspace, error) {
	wsID, err := NewID(idStr)
	if err != nil {
		return nil, err
	}

	if rootPath == "" {
		return nil, ErrInvalidRoot
	}

	cleanRoot := filepath.Clean(rootPath)
	absRoot, err := filepath.Abs(cleanRoot)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidRoot, err)
	}

	info, err := os.Stat(absRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrRootNotFound, absRoot)
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidRoot, err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("%w: %s", ErrNotDirectory, absRoot)
	}

	emptyResources, err := NewResources()
	if err != nil {
		return nil, err
	}

	return &Workspace{
		id:        wsID,
		root:      absRoot,
		resources: emptyResources,
	}, nil
}

// ID returns the immutable Workspace ID.
func (w *Workspace) ID() ID {
	if w == nil {
		return ID{}
	}
	return w.id
}

// Root returns the cleaned, absolute path to the workspace root directory.
func (w *Workspace) Root() string {
	if w == nil {
		return ""
	}
	return w.root
}

// Resources returns the immutable Resources collection owned by the Workspace.
func (w *Workspace) Resources() *Resources {
	if w == nil {
		return nil
	}
	return w.resources
}

// WithResources returns a new immutable Workspace with updated Resources.
func (w *Workspace) WithResources(resources *Resources) (*Workspace, error) {
	if w == nil {
		return nil, ErrNilWorkspace
	}
	if resources == nil {
		return nil, ErrNilResource
	}
	return &Workspace{
		id:        w.id,
		root:      w.root,
		resources: resources,
	}, nil
}
