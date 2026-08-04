package integration_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/unhield/limoxel/internal/filesystem"
	"github.com/unhield/limoxel/internal/project"
	"github.com/unhield/limoxel/internal/repository"
	"github.com/unhield/limoxel/internal/workspace"
)

func TestRepositoryIntegration(t *testing.T) {
	tempDir := t.TempDir()

	// Setup real production filesystem service
	osFs := filesystem.NewOSFilesystem()
	fileSer, err := filesystem.NewFileService(osFs)
	if err != nil {
		t.Fatalf("filesystem.NewFileService failed: %v", err)
	}

	// Setup real workspace directory hierarchy
	wsDir := filepath.Join(tempDir, "workspace_root")
	if err := fileSer.EnsureDirectory(wsDir); err != nil {
		t.Fatalf("EnsureDirectory wsDir failed: %v", err)
	}

	ws, err := workspace.New("ws-prod", wsDir)
	if err != nil {
		t.Fatalf("workspace.New failed: %v", err)
	}

	// Create project directory inside workspace
	projDir := filepath.Join(wsDir, "services")
	if err := fileSer.EnsureDirectory(projDir); err != nil {
		t.Fatalf("EnsureDirectory projDir failed: %v", err)
	}

	proj, err := project.New("services-project", ws, "services")
	if err != nil {
		t.Fatalf("project.New failed: %v", err)
	}

	t.Run("full repository creation and filesystem integration flow", func(t *testing.T) {
		// Create repository directory inside project
		repoDir := filepath.Join(projDir, "auth_service")
		if err := fileSer.EnsureDirectory(repoDir); err != nil {
			t.Fatalf("EnsureDirectory repoDir failed: %v", err)
		}

		// Write source file inside repository using production FileService
		mainFile := filepath.Join(repoDir, "main.go")
		if err := fileSer.WriteFile(mainFile, []byte("package main\n"), 0644); err != nil {
			t.Fatalf("fileSer.WriteFile failed: %v", err)
		}

		// Construct real Repository model
		repo, err := repository.New("auth-repo", proj, "auth_service")
		if err != nil {
			t.Fatalf("repository.New failed: %v", err)
		}

		if repo.Name() != "auth-repo" {
			t.Errorf("got Name %q, want auth-repo", repo.Name())
		}
		if repo.Root() != filepath.Clean(repoDir) {
			t.Errorf("got Root %q, want %q", repo.Root(), filepath.Clean(repoDir))
		}
		if repo.Project() != proj {
			t.Error("project instance mismatch")
		}

		// Discover files inside repository root using production Discoverer
		disc, err := filesystem.NewDiscoverer(repo.Root(), filesystem.NewIgnorer())
		if err != nil {
			t.Fatalf("filesystem.NewDiscoverer failed: %v", err)
		}

		discRes, err := disc.Discover()
		if err != nil {
			t.Fatalf("disc.Discover failed: %v", err)
		}

		if discRes.Count() < 2 { // repoDir and main.go
			t.Errorf("expected at least 2 discovered items, got %d", discRes.Count())
		}
	})

	t.Run("multiple repositories inside same project", func(t *testing.T) {
		repo1Dir := filepath.Join(projDir, "repo_one")
		repo2Dir := filepath.Join(projDir, "repo_two")
		_ = fileSer.EnsureDirectory(repo1Dir)
		_ = fileSer.EnsureDirectory(repo2Dir)

		r1, err1 := repository.New("repo-1", proj, "repo_one")
		r2, err2 := repository.New("repo-2", proj, "repo_two")

		if err1 != nil || err2 != nil {
			t.Fatalf("failed creating multiple repositories: err1=%v, err2=%v", err1, err2)
		}

		if r1.Name() == r2.Name() || r1.Root() == r2.Root() {
			t.Error("expected distinct repository models for r1 and r2")
		}
	})

	t.Run("repository boundary validation error handling", func(t *testing.T) {
		outsideDir := t.TempDir() // Outside project and workspace root

		_, err := repository.New("outside-repo", proj, outsideDir)
		if err == nil || !errors.Is(err, repository.ErrRepositoryRootOutsideProject) {
			t.Errorf("got error %v, want ErrRepositoryRootOutsideProject for outside path", err)
		}
	})

	t.Run("invalid repository configuration error handling", func(t *testing.T) {
		_, errEmptyName := repository.New("  ", proj, "services")
		if !errors.Is(errEmptyName, repository.ErrInvalidName) {
			t.Errorf("got %v, want ErrInvalidName", errEmptyName)
		}

		_, errNilProj := repository.New("repo", nil, "services")
		if !errors.Is(errNilProj, repository.ErrNilProject) {
			t.Errorf("got %v, want ErrNilProject", errNilProj)
		}

		_, errEmptyRoot := repository.New("repo", proj, "   ")
		if !errors.Is(errEmptyRoot, repository.ErrInvalidRepositoryRoot) {
			t.Errorf("got %v, want ErrInvalidRepositoryRoot", errEmptyRoot)
		}
	})

	t.Run("resource cleanup and directory removal integration", func(t *testing.T) {
		tempRepoDir := filepath.Join(projDir, "temp_repo")
		_ = fileSer.EnsureDirectory(tempRepoDir)

		tempRepo, err := repository.New("temp-repo", proj, "temp_repo")
		if err != nil {
			t.Fatalf("failed to create tempRepo: %v", err)
		}

		if !fileSer.Exists(tempRepo.Root()) {
			t.Error("expected tempRepo directory to exist before removal")
		}

		if err := fileSer.RemoveAll(tempRepo.Root()); err != nil {
			t.Fatalf("fileSer.RemoveAll failed: %v", err)
		}

		if fileSer.Exists(tempRepo.Root()) {
			t.Error("expected tempRepo directory to be cleaned up")
		}
	})
}
