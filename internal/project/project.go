package project

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/unhield/limoxel/internal/workspace"
)

// Project represents an immutable logical project located within a Workspace.
type Project struct {
	name string
	root string
	ws   *workspace.Workspace
}

// New constructs and validates a new immutable Project instance.
// It resolves the project root relative to the provided Workspace and ensures it resides within the workspace root.
func New(name string, ws *workspace.Workspace, projectRootPath string) (*Project, error) {
	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		return nil, ErrInvalidName
	}

	if ws == nil {
		return nil, ErrNilWorkspace
	}

	cleanPath := strings.TrimSpace(projectRootPath)
	if cleanPath == "" {
		return nil, ErrInvalidProjectRoot
	}

	var absProjectRoot string
	if filepath.IsAbs(cleanPath) {
		absProjectRoot = filepath.Clean(cleanPath)
	} else {
		absProjectRoot = filepath.Clean(filepath.Join(ws.Root(), cleanPath))
	}

	rel, relErr := filepath.Rel(ws.Root(), absProjectRoot)
	if relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || strings.HasPrefix(rel, "../") || strings.HasPrefix(rel, "..\\") {
		return nil, fmt.Errorf("%w: path %s is outside workspace root %s", ErrProjectRootOutsideWorkspace, absProjectRoot, ws.Root())
	}

	return &Project{
		name: cleanName,
		root: absProjectRoot,
		ws:   ws,
	}, nil
}

// Name returns the project's identity name.
func (p *Project) Name() string {
	if p == nil {
		return ""
	}
	return p.name
}

// Root returns the cleaned, absolute path to the project root directory.
func (p *Project) Root() string {
	if p == nil {
		return ""
	}
	return p.root
}

// Workspace returns the owning Workspace instance.
func (p *Project) Workspace() *workspace.Workspace {
	if p == nil {
		return nil
	}
	return p.ws
}

// String returns the formatted string representation of the Project.
func (p *Project) String() string {
	if p == nil {
		return ""
	}
	return fmt.Sprintf("Project<%s>(%s)", p.name, p.root)
}
