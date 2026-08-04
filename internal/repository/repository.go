package repository

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/unhield/limoxel/internal/project"
)

// Repository represents an immutable logical source repository located within a Project.
type Repository struct {
	name string
	root string
	proj *project.Project
}

// New constructs and validates a new immutable Repository instance.
// It resolves the repository root relative to the provided Project and ensures it resides within the project root.
func New(name string, proj *project.Project, repoRootPath string) (*Repository, error) {
	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		return nil, ErrInvalidName
	}

	if proj == nil {
		return nil, ErrNilProject
	}

	cleanPath := strings.TrimSpace(repoRootPath)
	if cleanPath == "" {
		return nil, ErrInvalidRepositoryRoot
	}

	var absRepoRoot string
	if filepath.IsAbs(cleanPath) {
		absRepoRoot = filepath.Clean(cleanPath)
	} else {
		absRepoRoot = filepath.Clean(filepath.Join(proj.Root(), cleanPath))
	}

	rel, err := filepath.Rel(proj.Root(), absRepoRoot)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || strings.HasPrefix(rel, "../") || strings.HasPrefix(rel, "..\\") {
		return nil, fmt.Errorf("%w: path %s is outside project root %s", ErrRepositoryRootOutsideProject, absRepoRoot, proj.Root())
	}

	return &Repository{
		name: cleanName,
		root: absRepoRoot,
		proj: proj,
	}, nil
}

// Name returns the repository's identity name.
func (r *Repository) Name() string {
	if r == nil {
		return ""
	}
	return r.name
}

// Root returns the cleaned, absolute path to the repository root directory.
func (r *Repository) Root() string {
	if r == nil {
		return ""
	}
	return r.root
}

// Project returns the owning Project instance.
func (r *Repository) Project() *project.Project {
	if r == nil {
		return nil
	}
	return r.proj
}

// String returns the formatted string representation of the Repository.
func (r *Repository) String() string {
	if r == nil {
		return ""
	}
	return fmt.Sprintf("Repository<%s>(%s)", r.name, r.root)
}
