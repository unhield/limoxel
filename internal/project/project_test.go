package project_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/unhield/limoxel/internal/project"
	"github.com/unhield/limoxel/internal/workspace"
)

func TestProjectConstructorAndGetters(t *testing.T) {
	tempDir := t.TempDir()
	ws, err := workspace.New("ws-1", tempDir)
	if err != nil {
		t.Fatalf("workspace.New failed: %v", err)
	}

	t.Run("valid absolute path project creation", func(t *testing.T) {
		proj, err := project.New("proj-main", ws, tempDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if proj.Name() != "proj-main" {
			t.Errorf("got Name %q, want proj-main", proj.Name())
		}
		if proj.Root() != ws.Root() {
			t.Errorf("got Root %q, want %q", proj.Root(), ws.Root())
		}
		if proj.Workspace() != ws {
			t.Error("workspace instance mismatch")
		}
		expectedStr := "Project<proj-main>(" + ws.Root() + ")"
		if proj.String() != expectedStr {
			t.Errorf("got String %q, want %q", proj.String(), expectedStr)
		}
	})

	t.Run("valid relative path project creation", func(t *testing.T) {
		subDir := filepath.Join(tempDir, "subproj")

		proj, err := project.New("sub-proj", ws, "subproj")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if proj.Name() != "sub-proj" {
			t.Errorf("got Name %q, want sub-proj", proj.Name())
		}
		if proj.Root() != filepath.Clean(subDir) {
			t.Errorf("got Root %q, want %q", proj.Root(), filepath.Clean(subDir))
		}
	})

	t.Run("empty name error", func(t *testing.T) {
		_, err := project.New("   ", ws, tempDir)
		if !errors.Is(err, project.ErrInvalidName) {
			t.Errorf("got %v, want ErrInvalidName", err)
		}
	})

	t.Run("nil workspace error", func(t *testing.T) {
		_, err := project.New("proj-1", nil, tempDir)
		if !errors.Is(err, project.ErrNilWorkspace) {
			t.Errorf("got %v, want ErrNilWorkspace", err)
		}
	})

	t.Run("empty project root path error", func(t *testing.T) {
		_, err := project.New("proj-1", ws, "   ")
		if !errors.Is(err, project.ErrInvalidProjectRoot) {
			t.Errorf("got %v, want ErrInvalidProjectRoot", err)
		}
	})

	t.Run("project root outside workspace error", func(t *testing.T) {
		outsideDir := filepath.Dir(tempDir)
		_, err := project.New("proj-1", ws, outsideDir)
		if !errors.Is(err, project.ErrProjectRootOutsideWorkspace) {
			t.Errorf("got %v, want ErrProjectRootOutsideWorkspace", err)
		}
	})
}

func TestNilProjectSafety(t *testing.T) {
	var proj *project.Project

	if proj.Name() != "" {
		t.Error("expected empty string for nil Name")
	}
	if proj.Root() != "" {
		t.Error("expected empty string for nil Root")
	}
	if proj.Workspace() != nil {
		t.Error("expected nil Workspace for nil Project")
	}
	if proj.String() != "" {
		t.Error("expected empty string for nil String")
	}
}
