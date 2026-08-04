package workspace_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/unhield/limoxel/internal/workspace"
)

func TestIDValidation(t *testing.T) {
	t.Run("valid ID", func(t *testing.T) {
		id, err := workspace.NewID("ws-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id.Value() != "ws-1" {
			t.Errorf("got %q, want ws-1", id.Value())
		}
		if id.IsEmpty() {
			t.Error("expected non-empty ID")
		}
	})

	t.Run("empty ID", func(t *testing.T) {
		_, err := workspace.NewID("  ")
		if !errors.Is(err, workspace.ErrInvalidID) {
			t.Errorf("got %v, want ErrInvalidID", err)
		}
	})

	t.Run("spaces in ID", func(t *testing.T) {
		_, err := workspace.NewID("ws id")
		if err == nil || !errors.Is(err, workspace.ErrInvalidID) {
			t.Errorf("got %v, want ErrInvalidID error containing spaces constraint", err)
		}
	})
}

func TestResourceIDAndResource(t *testing.T) {
	t.Run("valid resource creation", func(t *testing.T) {
		res, err := workspace.NewResource("res-1", "/src/main.go")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.ID().Value() != "res-1" {
			t.Errorf("got %q, want res-1", res.ID().Value())
		}
		if res.Path() != "/src/main.go" {
			t.Errorf("got path %q, want /src/main.go", res.Path())
		}
	})

	t.Run("invalid resource ID", func(t *testing.T) {
		_, err := workspace.NewResource("  ", "/path")
		if !errors.Is(err, workspace.ErrInvalidResourceID) {
			t.Errorf("got %v, want ErrInvalidResourceID", err)
		}
	})

	t.Run("invalid resource path", func(t *testing.T) {
		_, err := workspace.NewResource("res-1", "   ")
		if !errors.Is(err, workspace.ErrInvalidResourcePath) {
			t.Errorf("got %v, want ErrInvalidResourcePath", err)
		}
	})

	t.Run("nil resource getters", func(t *testing.T) {
		var res *workspace.Resource
		if !res.ID().IsEmpty() {
			t.Error("expected empty ID for nil resource")
		}
		if res.Path() != "" {
			t.Error("expected empty path for nil resource")
		}
	})
}

func TestResourcesCollection(t *testing.T) {
	r1, _ := workspace.NewResource("r1", "/path/1")
	r2, _ := workspace.NewResource("r2", "/path/2")

	rs, err := workspace.NewResources(r1, r2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if rs.Count() != 2 {
		t.Errorf("got count %d, want 2", rs.Count())
	}
	if !rs.Has("r1") || !rs.Has("r2") {
		t.Error("expected Has to report true for r1 and r2")
	}
	if rs.Has("r3") {
		t.Error("expected Has to report false for r3")
	}

	// Test deterministic List ordering
	list := rs.List()
	if len(list) != 2 || list[0].ID().Value() != "r1" || list[1].ID().Value() != "r2" {
		t.Errorf("unexpected list ordering: %v", list)
	}

	// Test Get
	got, err := rs.Get("r1")
	if err != nil || got.Path() != "/path/1" {
		t.Errorf("Get(r1) got %v, %v", got, err)
	}

	_, err = rs.Get("missing")
	if !errors.Is(err, workspace.ErrResourceNotFound) {
		t.Errorf("got %v, want ErrResourceNotFound", err)
	}

	// Test With immutability
	r3, _ := workspace.NewResource("r3", "/path/3")
	rsWith, err := rs.With(r3)
	if err != nil {
		t.Fatalf("With failed: %v", err)
	}
	if rs.Count() != 2 {
		t.Errorf("original Resources count changed to %d", rs.Count())
	}
	if rsWith.Count() != 3 {
		t.Errorf("new Resources count got %d, want 3", rsWith.Count())
	}

	// Duplicate resource check
	_, err = rs.With(r1)
	if !errors.Is(err, workspace.ErrDuplicateResource) {
		t.Errorf("got %v, want ErrDuplicateResource", err)
	}

	// Nil resource check
	_, err = workspace.NewResources(r1, nil)
	if !errors.Is(err, workspace.ErrNilResource) {
		t.Errorf("got %v, want ErrNilResource", err)
	}
}

func TestNilResourcesSafety(t *testing.T) {
	var rs *workspace.Resources
	if _, err := rs.Get("r1"); !errors.Is(err, workspace.ErrResourceNotFound) {
		t.Errorf("got %v, want ErrResourceNotFound", err)
	}
	if rs.Has("r1") {
		t.Error("expected false for nil Has")
	}
	if rs.Count() != 0 {
		t.Error("expected 0 for nil Count")
	}
	if len(rs.List()) != 0 {
		t.Error("expected empty list for nil List")
	}
}

func TestWorkspaceCreationAndValidation(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("valid workspace", func(t *testing.T) {
		ws, err := workspace.New("ws-main", tempDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if ws.ID().Value() != "ws-main" {
			t.Errorf("got ID %q, want ws-main", ws.ID().Value())
		}

		absTemp, _ := filepath.Abs(filepath.Clean(tempDir))
		if ws.Root() != absTemp {
			t.Errorf("got root %q, want %q", ws.Root(), absTemp)
		}
		if ws.Resources() == nil || ws.Resources().Count() != 0 {
			t.Error("expected empty resources collection")
		}
	})

	t.Run("invalid workspace ID", func(t *testing.T) {
		_, err := workspace.New("", tempDir)
		if !errors.Is(err, workspace.ErrInvalidID) {
			t.Errorf("got %v, want ErrInvalidID", err)
		}
	})

	t.Run("empty root path", func(t *testing.T) {
		_, err := workspace.New("ws-1", "")
		if !errors.Is(err, workspace.ErrInvalidRoot) {
			t.Errorf("got %v, want ErrInvalidRoot", err)
		}
	})

	t.Run("non-existent root path", func(t *testing.T) {
		nonExistent := filepath.Join(tempDir, "does-not-exist")
		_, err := workspace.New("ws-1", nonExistent)
		if !errors.Is(err, workspace.ErrRootNotFound) {
			t.Errorf("got %v, want ErrRootNotFound", err)
		}
	})

	t.Run("root path is a file, not a directory", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "file.txt")
		if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}

		_, err := workspace.New("ws-1", filePath)
		if !errors.Is(err, workspace.ErrNotDirectory) {
			t.Errorf("got %v, want ErrNotDirectory", err)
		}
	})
}

func TestWorkspaceWithResources(t *testing.T) {
	tempDir := t.TempDir()
	ws, _ := workspace.New("ws-1", tempDir)

	r1, _ := workspace.NewResource("r1", "file1.txt")
	newResources, _ := workspace.NewResources(r1)

	updatedWs, err := ws.WithResources(newResources)
	if err != nil {
		t.Fatalf("WithResources failed: %v", err)
	}

	if ws.Resources().Count() != 0 {
		t.Errorf("original workspace resources count changed to %d", ws.Resources().Count())
	}
	if updatedWs.Resources().Count() != 1 {
		t.Errorf("updated workspace resources count got %d, want 1", updatedWs.Resources().Count())
	}

	// Test nil validation
	_, err = ws.WithResources(nil)
	if !errors.Is(err, workspace.ErrNilResource) {
		t.Errorf("got %v, want ErrNilResource", err)
	}
}

func TestNilWorkspaceSafety(t *testing.T) {
	var ws *workspace.Workspace
	if !ws.ID().IsEmpty() {
		t.Error("expected empty ID for nil workspace")
	}
	if ws.Root() != "" {
		t.Error("expected empty root for nil workspace")
	}
	if ws.Resources() != nil {
		t.Error("expected nil resources for nil workspace")
	}
	r1, _ := workspace.NewResource("r1", "file1.txt")
	newRes, _ := workspace.NewResources(r1)
	_, err := ws.WithResources(newRes)
	if !errors.Is(err, workspace.ErrNilWorkspace) {
		t.Errorf("got %v, want ErrNilWorkspace", err)
	}
}
