package repository_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/unhield/limoxel/internal/project"
	"github.com/unhield/limoxel/internal/repository"
	"github.com/unhield/limoxel/internal/workspace"
)

func TestRepositoryConstructorAndGetters(t *testing.T) {
	tempDir := t.TempDir()
	ws, err := workspace.New("ws-1", tempDir)
	if err != nil {
		t.Fatalf("workspace.New failed: %v", err)
	}

	proj, err := project.New("proj-1", ws, tempDir)
	if err != nil {
		t.Fatalf("project.New failed: %v", err)
	}

	t.Run("valid absolute path repository creation", func(t *testing.T) {
		repo, err := repository.New("repo-main", proj, tempDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if repo.Name() != "repo-main" {
			t.Errorf("got Name %q, want repo-main", repo.Name())
		}
		if repo.Root() != proj.Root() {
			t.Errorf("got Root %q, want %q", repo.Root(), proj.Root())
		}
		if repo.Project() != proj {
			t.Error("project instance mismatch")
		}
		expectedStr := "Repository<repo-main>(" + proj.Root() + ")"
		if repo.String() != expectedStr {
			t.Errorf("got String %q, want %q", repo.String(), expectedStr)
		}
	})

	t.Run("valid relative path repository creation", func(t *testing.T) {
		subDir := filepath.Join(tempDir, "subrepo")
		repo, err := repository.New("sub-repo", proj, "subrepo")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if repo.Name() != "sub-repo" {
			t.Errorf("got Name %q, want sub-repo", repo.Name())
		}
		if repo.Root() != filepath.Clean(subDir) {
			t.Errorf("got Root %q, want %q", repo.Root(), filepath.Clean(subDir))
		}
	})

	t.Run("empty name error", func(t *testing.T) {
		_, err := repository.New("   ", proj, tempDir)
		if !errors.Is(err, repository.ErrInvalidName) {
			t.Errorf("got %v, want ErrInvalidName", err)
		}
	})

	t.Run("nil project error", func(t *testing.T) {
		_, err := repository.New("repo-1", nil, tempDir)
		if !errors.Is(err, repository.ErrNilProject) {
			t.Errorf("got %v, want ErrNilProject", err)
		}
	})

	t.Run("empty repository root path error", func(t *testing.T) {
		_, err := repository.New("repo-1", proj, "   ")
		if !errors.Is(err, repository.ErrInvalidRepositoryRoot) {
			t.Errorf("got %v, want ErrInvalidRepositoryRoot", err)
		}
	})

	t.Run("repository root outside project error", func(t *testing.T) {
		outsideDir := filepath.Dir(tempDir)
		_, err := repository.New("repo-1", proj, outsideDir)
		if !errors.Is(err, repository.ErrRepositoryRootOutsideProject) {
			t.Errorf("got %v, want ErrRepositoryRootOutsideProject", err)
		}
	})
}

func TestNilRepositorySafety(t *testing.T) {
	var repo *repository.Repository

	if repo.Name() != "" {
		t.Error("expected empty string for nil Name")
	}
	if repo.Root() != "" {
		t.Error("expected empty string for nil Root")
	}
	if repo.Project() != nil {
		t.Error("expected nil Project for nil Repository")
	}
	if repo.String() != "" {
		t.Error("expected empty string for nil String")
	}
}
