package discovery_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
	"github.com/unhield/limoxel/internal/language"
	"github.com/unhield/limoxel/internal/project"
	"github.com/unhield/limoxel/internal/repository"
	"github.com/unhield/limoxel/internal/workspace"
)

func setupTestLanguageRegistry(t *testing.T) *language.Registry {
	t.Helper()
	reg := language.NewRegistry()

	goLang, err := language.New("go", "Go", []string{".go"}, nil, []string{"golang"})
	if err != nil {
		t.Fatalf("failed creating Go language: %v", err)
	}

	pyLang, err := language.New("python", "Python", []string{".py"}, nil, []string{"py"})
	if err != nil {
		t.Fatalf("failed creating Python language: %v", err)
	}

	tsLang, err := language.New("typescript", "TypeScript", []string{".ts", ".tsx"}, nil, []string{"ts"})
	if err != nil {
		t.Fatalf("failed creating TypeScript language: %v", err)
	}

	dockerLang, err := language.New("dockerfile", "Dockerfile", nil, []string{"Dockerfile"}, []string{"docker"})
	if err != nil {
		t.Fatalf("failed creating Dockerfile language: %v", err)
	}

	_ = reg.Register(goLang)
	_ = reg.Register(pyLang)
	_ = reg.Register(tsLang)
	_ = reg.Register(dockerLang)

	return reg
}

func TestDiscoverer_New(t *testing.T) {
	t.Run("nil registry returns ErrNilRegistry", func(t *testing.T) {
		d, err := discovery.New(nil)
		if d != nil {
			t.Errorf("expected nil discoverer, got %v", d)
		}
		if !errors.Is(err, discovery.ErrNilRegistry) {
			t.Errorf("expected ErrNilRegistry, got %v", err)
		}
	})

	t.Run("valid registry returns operational discoverer with default options", func(t *testing.T) {
		reg := setupTestLanguageRegistry(t)
		d, err := discovery.New(reg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if d.LanguageRegistry() != reg {
			t.Errorf("registry mismatch")
		}
		opts := d.Options()
		if opts.FollowSymlinks != false || opts.IncludeHidden != true {
			t.Errorf("unexpected default options: %+v", opts)
		}
	})
}

func TestDiscoverer_Discover_Validation(t *testing.T) {
	reg := setupTestLanguageRegistry(t)
	d, err := discovery.New(reg)
	if err != nil {
		t.Fatalf("failed creating discoverer: %v", err)
	}

	t.Run("nil discoverer returns ErrNilDiscoverer", func(t *testing.T) {
		var nilD *discovery.Discoverer
		res, err := nilD.Discover(nil)
		if res != nil || !errors.Is(err, discovery.ErrNilDiscoverer) {
			t.Errorf("expected ErrNilDiscoverer, got res=%v, err=%v", res, err)
		}

		res, err = nilD.DiscoverPath("some/path")
		if res != nil || !errors.Is(err, discovery.ErrNilDiscoverer) {
			t.Errorf("expected ErrNilDiscoverer, got res=%v, err=%v", res, err)
		}
	})

	t.Run("nil repository returns ErrNilRepository", func(t *testing.T) {
		res, err := d.Discover(nil)
		if res != nil || !errors.Is(err, discovery.ErrNilRepository) {
			t.Errorf("expected ErrNilRepository, got res=%v, err=%v", res, err)
		}
	})

	t.Run("empty path returns ErrPathEmpty", func(t *testing.T) {
		res, err := d.DiscoverPath("")
		if res != nil || !errors.Is(err, discovery.ErrPathEmpty) {
			t.Errorf("expected ErrPathEmpty, got res=%v, err=%v", res, err)
		}

		res, err = d.DiscoverPath("   ")
		if res != nil || !errors.Is(err, discovery.ErrPathEmpty) {
			t.Errorf("expected ErrPathEmpty, got res=%v, err=%v", res, err)
		}
	})

	t.Run("nonexistent path returns ErrPathNotFound", func(t *testing.T) {
		res, err := d.DiscoverPath(filepath.Join(t.TempDir(), "nonexistent_dir"))
		if res != nil || !errors.Is(err, discovery.ErrPathNotFound) {
			t.Errorf("expected ErrPathNotFound, got res=%v, err=%v", res, err)
		}
	})

	t.Run("file path instead of directory returns ErrNotDirectory", func(t *testing.T) {
		tempFile := filepath.Join(t.TempDir(), "sample.txt")
		if err := os.WriteFile(tempFile, []byte("hello"), 0644); err != nil {
			t.Fatalf("failed creating temp file: %v", err)
		}

		res, err := d.DiscoverPath(tempFile)
		if res != nil || !errors.Is(err, discovery.ErrNotDirectory) {
			t.Errorf("expected ErrNotDirectory, got res=%v, err=%v", res, err)
		}
	})
}

func TestDiscoverer_Discover_BasicRepository(t *testing.T) {
	tempDir := t.TempDir()
	reg := setupTestLanguageRegistry(t)

	// Create test repository tree
	repoRoot := filepath.Join(tempDir, "sample_repo")
	_ = os.MkdirAll(filepath.Join(repoRoot, "src", "pkg"), 0755)
	_ = os.MkdirAll(filepath.Join(repoRoot, "scripts"), 0755)
	_ = os.MkdirAll(filepath.Join(repoRoot, "node_modules", "lib"), 0755)
	_ = os.MkdirAll(filepath.Join(repoRoot, "vendor", "dep"), 0755)

	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte("package main\nfunc main(){}"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "src", "pkg", "util.go"), []byte("package pkg"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "scripts", "build.py"), []byte("print('building')"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "Dockerfile"), []byte("FROM golang:1.26"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("# Sample Repo"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, ".gitignore"), []byte("*.log"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "node_modules", "lib", "ignored.js"), []byte("ignored"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "vendor", "dep", "vendor.go"), []byte("package dep"), 0644)

	d, err := discovery.New(reg)
	if err != nil {
		t.Fatalf("failed creating discoverer: %v", err)
	}

	res, err := d.DiscoverPath(repoRoot)
	if err != nil {
		t.Fatalf("DiscoverPath failed: %v", err)
	}

	if res == nil {
		t.Fatal("expected non-nil result")
	}

	if res.Root() != filepath.Clean(repoRoot) {
		t.Errorf("got root %q, want %q", res.Root(), repoRoot)
	}

	if res.Repository() == nil {
		t.Error("expected valid repository instance in result")
	}

	// Discovered files should be 6: .gitignore, Dockerfile, README.md, main.go, scripts/build.py, src/pkg/util.go
	// node_modules and vendor are ignored by DefaultIgnoreRules
	if res.FileCount() != 6 {
		t.Errorf("got file count %d, want 6", res.FileCount())
		for _, f := range res.Files() {
			t.Logf("discovered: %s", f.RelPath())
		}
	}

	// Verify specific file lookup
	mainGo, exists := res.File("main.go")
	if !exists || mainGo == nil {
		t.Fatal("expected main.go to exist in result")
	}
	if mainGo.LanguageID() != "go" || mainGo.LanguageName() != "Go" {
		t.Errorf("main.go language got id=%s, name=%s", mainGo.LanguageID(), mainGo.LanguageName())
	}
	if mainGo.IsHidden() {
		t.Error("main.go should not be hidden")
	}
	if mainGo.Size() == 0 {
		t.Error("main.go size should be > 0")
	}

	// Verify hidden file (.gitignore)
	gitIgnore, exists := res.File(".gitignore")
	if !exists || gitIgnore == nil {
		t.Fatal("expected .gitignore in result")
	}
	if !gitIgnore.IsHidden() {
		t.Error(".gitignore should be marked as hidden")
	}

	// Verify Dockerfile
	dockerfile, exists := res.File("Dockerfile")
	if !exists || dockerfile == nil {
		t.Fatal("expected Dockerfile in result")
	}
	if dockerfile.LanguageID() != "dockerfile" {
		t.Errorf("Dockerfile language got %s, want dockerfile", dockerfile.LanguageID())
	}

	// Verify unknown language (README.md)
	readme, exists := res.File("README.md")
	if !exists || readme == nil {
		t.Fatal("expected README.md in result")
	}
	if readme.LanguageID() != "unknown" || readme.Language() != nil {
		t.Errorf("README.md language got %s, want unknown", readme.LanguageID())
	}

	// Verify Language Distribution
	langs := res.Languages()
	if len(langs) == 0 {
		t.Fatal("expected languages in result")
	}

	// Go should have 2 files (main.go, src/pkg/util.go)
	goDist, exists := res.Language("go")
	if !exists || goDist == nil {
		t.Fatal("expected Go in language distribution")
	}
	if goDist.FileCount() != 2 {
		t.Errorf("Go file count got %d, want 2", goDist.FileCount())
	}

	// Metadata
	meta := res.Metadata()
	if meta == nil {
		t.Fatal("expected metadata in result")
	}
	if meta.TotalFiles() != 6 {
		t.Errorf("meta TotalFiles got %d, want 6", meta.TotalFiles())
	}
}

func TestDiscoverer_Discover_DomainRepository(t *testing.T) {
	tempDir := t.TempDir()
	reg := setupTestLanguageRegistry(t)

	ws, _ := workspace.New("test-ws", tempDir)
	proj, _ := project.New("test-proj", ws, "my_app")
	repoRoot := filepath.Join(tempDir, "my_app", "my_repo")
	_ = os.MkdirAll(repoRoot, 0755)
	repo, err := repository.New("my_repo", proj, repoRoot)
	if err != nil {
		t.Fatalf("repository.New failed: %v", err)
	}

	_ = os.WriteFile(filepath.Join(repo.Root(), "app.go"), []byte("package app"), 0644)

	d, err := discovery.New(reg)
	if err != nil {
		t.Fatalf("discovery.New failed: %v", err)
	}

	res, err := d.Discover(repo)
	if err != nil {
		t.Fatalf("Discover failed: %v", err)
	}

	if res.Repository() != repo {
		t.Errorf("expected original repository instance to be preserved")
	}
	if res.FileCount() != 1 {
		t.Errorf("got file count %d, want 1", res.FileCount())
	}
}

func TestDiscoverer_GitMetadata(t *testing.T) {
	tempDir := t.TempDir()
	reg := setupTestLanguageRegistry(t)

	repoRoot := filepath.Join(tempDir, "git_repo")
	gitDir := filepath.Join(repoRoot, ".git")
	_ = os.MkdirAll(filepath.Join(gitDir, "refs", "heads", "feature"), 0755)
	_ = os.MkdirAll(filepath.Join(gitDir, "refs", "remotes", "origin"), 0755)

	// Mock git HEAD pointing to refs/heads/feature/capability
	_ = os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/feature/capability\n"), 0644)
	_ = os.WriteFile(filepath.Join(gitDir, "refs", "heads", "feature", "capability"), []byte("a1b2c3d4e5f60718293a4b5c6d7e8f9012345678\n"), 0644)
	_ = os.WriteFile(filepath.Join(gitDir, "refs", "remotes", "origin", "HEAD"), []byte("ref: refs/remotes/origin/main\n"), 0644)

	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte("package main"), 0644)

	d, err := discovery.New(reg)
	if err != nil {
		t.Fatalf("discovery.New failed: %v", err)
	}

	res, err := d.DiscoverPath(repoRoot)
	if err != nil {
		t.Fatalf("DiscoverPath failed: %v", err)
	}

	meta := res.Metadata()
	if !meta.IsGit() {
		t.Error("expected isGit=true")
	}
	if meta.CurrentBranch() != "feature/capability" {
		t.Errorf("got branch %q, want feature/capability", meta.CurrentBranch())
	}
	if meta.DefaultBranch() != "main" {
		t.Errorf("got default branch %q, want main", meta.DefaultBranch())
	}
	if meta.LatestCommit() != "a1b2c3d4e5f60718293a4b5c6d7e8f9012345678" {
		t.Errorf("got latest commit %q", meta.LatestCommit())
	}
}

func TestDiscoverer_GitPackedRefsFallback(t *testing.T) {
	tempDir := t.TempDir()
	reg := setupTestLanguageRegistry(t)

	repoRoot := filepath.Join(tempDir, "git_packed_repo")
	gitDir := filepath.Join(repoRoot, ".git")
	_ = os.MkdirAll(gitDir, 0755)

	_ = os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main\n"), 0644)
	packedContent := "# pack-refs with: peeled-tags\n" +
		"f0e1d2c3b4a5968778695a4b3c2d1e0f98765432 refs/heads/main\n"
	_ = os.WriteFile(filepath.Join(gitDir, "packed-refs"), []byte(packedContent), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte("package main"), 0644)

	d, err := discovery.New(reg)
	if err != nil {
		t.Fatalf("discovery.New failed: %v", err)
	}

	res, err := d.DiscoverPath(repoRoot)
	if err != nil {
		t.Fatalf("DiscoverPath failed: %v", err)
	}

	meta := res.Metadata()
	if meta.LatestCommit() != "f0e1d2c3b4a5968778695a4b3c2d1e0f98765432" {
		t.Errorf("got packed latest commit %q", meta.LatestCommit())
	}
}

func TestDiscoverer_GitPackedRefs_NotFound(t *testing.T) {
	tempDir := t.TempDir()
	reg := setupTestLanguageRegistry(t)

	repoRoot := filepath.Join(tempDir, "git_packed_notfound")
	gitDir := filepath.Join(repoRoot, ".git")
	_ = os.MkdirAll(gitDir, 0755)

	_ = os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/feature/unpacked\n"), 0644)
	packedContent := "# pack-refs with: peeled-tags\n" +
		"f0e1d2c3b4a5968778695a4b3c2d1e0f98765432 refs/heads/main\n"
	_ = os.WriteFile(filepath.Join(gitDir, "packed-refs"), []byte(packedContent), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte("package main"), 0644)

	d, err := discovery.New(reg)
	if err != nil {
		t.Fatalf("discovery.New failed: %v", err)
	}

	res, err := d.DiscoverPath(repoRoot)
	if err != nil {
		t.Fatalf("DiscoverPath failed: %v", err)
	}

	meta := res.Metadata()
	if meta.LatestCommit() != "" {
		t.Errorf("expected empty commit for missing ref, got %q", meta.LatestCommit())
	}

	// Ensure no error diagnostics recorded for normal not-found condition
	for _, diag := range res.Diagnostics() {
		if diag.Code() == "GIT_PACKED_REFS_READ_ERROR" {
			t.Errorf("unexpected error diagnostic for missing ref: %v", diag)
		}
	}
}

func TestDiscoverer_GitPackedRefs_ScannerError(t *testing.T) {
	tempDir := t.TempDir()
	reg := setupTestLanguageRegistry(t)

	repoRoot := filepath.Join(tempDir, "git_packed_scanner_err")
	gitDir := filepath.Join(repoRoot, ".git")
	_ = os.MkdirAll(gitDir, 0755)

	_ = os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main\n"), 0644)

	// Create a corrupted packed-refs file with a single line exceeding 64KB (bufio.MaxScanTokenSize) without newline
	longLine := make([]byte, 70000)
	for i := range longLine {
		longLine[i] = 'a'
	}
	_ = os.WriteFile(filepath.Join(gitDir, "packed-refs"), longLine, 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte("package main"), 0644)

	d, err := discovery.New(reg)
	if err != nil {
		t.Fatalf("discovery.New failed: %v", err)
	}

	res, err := d.DiscoverPath(repoRoot)
	if err != nil {
		t.Fatalf("DiscoverPath failed: %v", err)
	}

	// Should record a diagnostic for scanner error
	var foundScannerDiag bool
	for _, diag := range res.Diagnostics() {
		if diag.Code() == "GIT_PACKED_REFS_READ_ERROR" {
			foundScannerDiag = true
			break
		}
	}

	if !foundScannerDiag {
		t.Error("expected GIT_PACKED_REFS_READ_ERROR diagnostic when scanner fails")
	}
}

func TestDiscoverer_NestedRepositories(t *testing.T) {
	tempDir := t.TempDir()
	reg := setupTestLanguageRegistry(t)

	repoRoot := filepath.Join(tempDir, "parent_repo")
	_ = os.MkdirAll(filepath.Join(repoRoot, ".git"), 0755)
	_ = os.MkdirAll(filepath.Join(repoRoot, "submodules", "child_repo", ".git"), 0755)

	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte("package main"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "submodules", "child_repo", "child.go"), []byte("package child"), 0644)

	d, err := discovery.New(reg)
	if err != nil {
		t.Fatalf("discovery.New failed: %v", err)
	}

	res, err := d.DiscoverPath(repoRoot)
	if err != nil {
		t.Fatalf("DiscoverPath failed: %v", err)
	}

	nested := res.NestedRepositories()
	if len(nested) != 1 {
		t.Fatalf("expected 1 nested repository, got %d: %v", len(nested), nested)
	}
	if nested[0] != "submodules/child_repo" {
		t.Errorf("nested repo got %q, want submodules/child_repo", nested[0])
	}
}

func TestDiscoverer_CustomIgnoreRules(t *testing.T) {
	tempDir := t.TempDir()
	reg := setupTestLanguageRegistry(t)

	repoRoot := filepath.Join(tempDir, "custom_ignore_repo")
	_ = os.MkdirAll(filepath.Join(repoRoot, "build", "output"), 0755)
	_ = os.MkdirAll(filepath.Join(repoRoot, "temp"), 0755)

	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte("package main"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "build", "output", "app.exe"), []byte("bin"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "temp", "cache.dat"), []byte("cache"), 0644)

	opts := discovery.DefaultOptions()
	opts.AdditionalIgnoreRules = []string{"build", "temp"}

	d, err := discovery.New(reg, opts)
	if err != nil {
		t.Fatalf("discovery.New failed: %v", err)
	}

	res, err := d.DiscoverPath(repoRoot)
	if err != nil {
		t.Fatalf("DiscoverPath failed: %v", err)
	}

	if res.FileCount() != 1 {
		t.Errorf("got file count %d, want 1", res.FileCount())
	}
	if _, exists := res.File("main.go"); !exists {
		t.Error("expected main.go to be present")
	}
}

func TestDiscoverer_TraversalLimits(t *testing.T) {
	tempDir := t.TempDir()
	reg := setupTestLanguageRegistry(t)

	repoRoot := filepath.Join(tempDir, "limits_repo")
	_ = os.MkdirAll(filepath.Join(repoRoot, "a", "b", "c", "d"), 0755)

	_ = os.WriteFile(filepath.Join(repoRoot, "root.go"), []byte("package root"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "a", "level1.go"), []byte("package a"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "a", "b", "level2.go"), []byte("package b"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "a", "b", "c", "level3.go"), []byte("package c"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "a", "b", "c", "d", "level4.go"), []byte("package d"), 0644)

	t.Run("MaxDepth restriction", func(t *testing.T) {
		opts := discovery.DefaultOptions()
		opts.MaxDepth = 2 // Only allow up to depth 2 (root.go and a/level1.go)

		d, err := discovery.New(reg, opts)
		if err != nil {
			t.Fatalf("discovery.New failed: %v", err)
		}

		res, err := d.DiscoverPath(repoRoot)
		if err != nil {
			t.Fatalf("DiscoverPath failed: %v", err)
		}

		if _, exists := res.File("root.go"); !exists {
			t.Error("root.go should exist at depth 1")
		}
		if _, exists := res.File("a/level1.go"); !exists {
			t.Error("a/level1.go should exist at depth 2")
		}
		if _, exists := res.File("a/b/c/d/level4.go"); exists {
			t.Error("a/b/c/d/level4.go should be skipped by MaxDepth")
		}
	})

	t.Run("MaxFiles restriction", func(t *testing.T) {
		opts := discovery.DefaultOptions()
		opts.MaxFiles = 2

		d, err := discovery.New(reg, opts)
		if err != nil {
			t.Fatalf("discovery.New failed: %v", err)
		}

		res, err := d.DiscoverPath(repoRoot)
		if err != nil {
			t.Fatalf("DiscoverPath failed: %v", err)
		}

		if res.FileCount() > 2 {
			t.Errorf("file count %d exceeds MaxFiles (2)", res.FileCount())
		}
		if len(res.Diagnostics()) == 0 {
			t.Error("expected diagnostic for MaxFiles limit reached")
		}
	})
}

func TestDiscoverer_EmptyAndOnlyIgnoredRepository(t *testing.T) {
	tempDir := t.TempDir()
	reg := setupTestLanguageRegistry(t)
	d, err := discovery.New(reg)
	if err != nil {
		t.Fatalf("discovery.New failed: %v", err)
	}

	t.Run("empty repository", func(t *testing.T) {
		emptyRepo := filepath.Join(tempDir, "empty_repo")
		_ = os.MkdirAll(emptyRepo, 0755)

		res, err := d.DiscoverPath(emptyRepo)
		if err != nil {
			t.Fatalf("DiscoverPath on empty repo failed: %v", err)
		}
		if res.FileCount() != 0 {
			t.Errorf("expected 0 files, got %d", res.FileCount())
		}
		if len(res.Languages()) != 0 {
			t.Errorf("expected 0 languages, got %d", len(res.Languages()))
		}
	})

	t.Run("repository containing only ignored directories and files", func(t *testing.T) {
		ignoredRepo := filepath.Join(tempDir, "ignored_repo")
		_ = os.MkdirAll(filepath.Join(ignoredRepo, "node_modules", "foo"), 0755)
		_ = os.MkdirAll(filepath.Join(ignoredRepo, ".git", "objects"), 0755)
		_ = os.WriteFile(filepath.Join(ignoredRepo, "node_modules", "foo", "index.js"), []byte("var a = 1;"), 0644)

		res, err := d.DiscoverPath(ignoredRepo)
		if err != nil {
			t.Fatalf("DiscoverPath on ignored repo failed: %v", err)
		}
		if res.FileCount() != 0 {
			t.Errorf("expected 0 files, got %d", res.FileCount())
		}
	})
}

func TestDiscoverer_ResultImmutability(t *testing.T) {
	tempDir := t.TempDir()
	reg := setupTestLanguageRegistry(t)

	repoRoot := filepath.Join(tempDir, "immut_repo")
	_ = os.MkdirAll(repoRoot, 0755)
	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte("package main"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "test.py"), []byte("import sys"), 0644)

	d, err := discovery.New(reg)
	if err != nil {
		t.Fatalf("discovery.New failed: %v", err)
	}

	res, err := d.DiscoverPath(repoRoot)
	if err != nil {
		t.Fatalf("DiscoverPath failed: %v", err)
	}

	// Mutate returned files slice
	files1 := res.Files()
	files1[0] = nil
	files2 := res.Files()
	if files2[0] == nil {
		t.Error("mutation of returned files slice affected internal state")
	}

	// Mutate returned languages slice
	langs1 := res.Languages()
	langs1[0] = nil
	langs2 := res.Languages()
	if langs2[0] == nil {
		t.Error("mutation of returned languages slice affected internal state")
	}

	// Mutate returned nested repos slice
	meta := res.Metadata()
	nested1 := meta.NestedRepositories()
	if nested1 != nil {
		nested1[0] = "mutated"
		nested2 := meta.NestedRepositories()
		if nested2[0] == "mutated" {
			t.Error("mutation of returned metadata nested repos slice affected internal state")
		}
	}
}

func TestDiscoverer_DeterminismAcrossRepeatedRuns(t *testing.T) {
	tempDir := t.TempDir()
	reg := setupTestLanguageRegistry(t)

	repoRoot := filepath.Join(tempDir, "det_repo")
	_ = os.MkdirAll(filepath.Join(repoRoot, "pkg", "z"), 0755)
	_ = os.MkdirAll(filepath.Join(repoRoot, "pkg", "a"), 0755)
	_ = os.MkdirAll(filepath.Join(repoRoot, "pkg", "m"), 0755)

	_ = os.WriteFile(filepath.Join(repoRoot, "pkg", "z", "z.go"), []byte("package z"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "pkg", "a", "a.go"), []byte("package a"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "pkg", "m", "m.go"), []byte("package m"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "b.py"), []byte("print('b')"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "a.py"), []byte("print('a')"), 0644)

	d, err := discovery.New(reg)
	if err != nil {
		t.Fatalf("discovery.New failed: %v", err)
	}

	res1, err := d.DiscoverPath(repoRoot)
	if err != nil {
		t.Fatalf("run 1 failed: %v", err)
	}

	res2, err := d.DiscoverPath(repoRoot)
	if err != nil {
		t.Fatalf("run 2 failed: %v", err)
	}

	if res1.FileCount() != res2.FileCount() {
		t.Fatalf("file count mismatch: %d vs %d", res1.FileCount(), res2.FileCount())
	}

	files1 := res1.Files()
	files2 := res2.Files()
	for i := range files1 {
		if files1[i].RelPath() != files2[i].RelPath() {
			t.Errorf("file[%d] mismatch: %s vs %s", i, files1[i].RelPath(), files2[i].RelPath())
		}
		if files1[i].LanguageID() != files2[i].LanguageID() {
			t.Errorf("file[%d] lang mismatch: %s vs %s", i, files1[i].LanguageID(), files2[i].LanguageID())
		}
	}

	langs1 := res1.Languages()
	langs2 := res2.Languages()
	if len(langs1) != len(langs2) {
		t.Fatalf("language count mismatch: %d vs %d", len(langs1), len(langs2))
	}
	for i := range langs1 {
		if langs1[i].LanguageID() != langs2[i].LanguageID() || langs1[i].FileCount() != langs2[i].FileCount() {
			t.Errorf("language[%d] mismatch: %+v vs %+v", i, langs1[i], langs2[i])
		}
	}
}

func TestDiscoverer_PathNormalizationAndBoundary(t *testing.T) {
	tempDir := t.TempDir()
	reg := setupTestLanguageRegistry(t)

	repoRoot := filepath.Join(tempDir, "norm_repo")
	_ = os.MkdirAll(filepath.Join(repoRoot, "sub"), 0755)
	_ = os.WriteFile(filepath.Join(repoRoot, "sub", "main.go"), []byte("package main"), 0644)

	d, err := discovery.New(reg)
	if err != nil {
		t.Fatalf("discovery.New failed: %v", err)
	}

	// Test DiscoverPath called from a sub-directory should resolve to root if git present, or handle normalization
	res, err := d.DiscoverPath(filepath.Join(repoRoot, ".", "sub", ".."))
	if err != nil {
		t.Fatalf("DiscoverPath failed: %v", err)
	}

	f, exists := res.File("sub/main.go")
	if !exists || f == nil {
		t.Errorf("expected sub/main.go in normalized result")
	}
	if f.RelPath() != "sub/main.go" {
		t.Errorf("got RelPath %q, want sub/main.go", f.RelPath())
	}
}

func TestFileEntry_Methods(t *testing.T) {
	now := time.Now()
	fe := discovery.NewFileEntry(
		"pkg\\util.go",
		"C:\\repo\\pkg\\util.go",
		false,
		123,
		now,
		".go",
		nil,
		false,
		false,
		false,
	)

	if fe.RelPath() != "pkg/util.go" {
		t.Errorf("RelPath got %q, want pkg/util.go", fe.RelPath())
	}
	if fe.Size() != 123 {
		t.Errorf("Size got %d, want 123", fe.Size())
	}
	if fe.Extension() != ".go" {
		t.Errorf("Extension got %q, want .go", fe.Extension())
	}
	if fe.LanguageID() != "unknown" {
		t.Errorf("LanguageID got %q, want unknown", fe.LanguageID())
	}
	if fe.LanguageName() != "Unknown" {
		t.Errorf("LanguageName got %q, want Unknown", fe.LanguageName())
	}
	if fe.IsDir() || fe.IsHidden() || fe.IsSymlink() || fe.IsIgnored() {
		t.Errorf("unexpected boolean flags")
	}
	if fe.String() == "" {
		t.Errorf("expected non-empty String()")
	}

	var nilFe *discovery.FileEntry
	if nilFe.RelPath() != "" || nilFe.Size() != 0 || nilFe.LanguageID() != "unknown" {
		t.Error("nil FileEntry should return safe default values")
	}
}

func TestFileEntry_CrossPlatformNormalization(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"pkg/util.go", "pkg/util.go"},
		{"pkg\\util.go", "pkg/util.go"},
		{"a/b/c/d.go", "a/b/c/d.go"},
		{"a\\b\\c\\d.go", "a/b/c/d.go"},
		{"a/b\\c/d.go", "a/b/c/d.go"},
		{"./pkg/util.go", "pkg/util.go"},
		{".\\pkg\\util.go", "pkg/util.go"},
		{"pkg/../pkg/util.go", "pkg/util.go"},
		{"pkg\\..\\pkg\\util.go", "pkg/util.go"},
		{"/pkg/util.go", "pkg/util.go"},
		{"\\pkg\\util.go", "pkg/util.go"},
	}

	for _, tc := range testCases {
		fe := discovery.NewFileEntry(tc.input, "C:/abs/"+tc.input, false, 10, time.Now(), ".go", nil, false, false, false)
		if fe.RelPath() != tc.expected {
			t.Errorf("input %q: got RelPath %q, want %q", tc.input, fe.RelPath(), tc.expected)
		}
	}
}

func TestLanguageDistribution_Methods(t *testing.T) {
	ld := discovery.NewLanguageDistribution("go", "Go", 10, 2048, 50.0, []string{".go", ".golang"})
	if ld.LanguageID() != "go" || ld.LanguageName() != "Go" {
		t.Errorf("id/name mismatch: %s/%s", ld.LanguageID(), ld.LanguageName())
	}
	if ld.FileCount() != 10 || ld.TotalBytes() != 2048 || ld.Percentage() != 50.0 {
		t.Errorf("stats mismatch")
	}
	if len(ld.Extensions()) != 2 {
		t.Errorf("extensions count got %d, want 2", len(ld.Extensions()))
	}
	if ld.String() == "" {
		t.Error("expected non-empty String()")
	}

	var nilLd *discovery.LanguageDistribution
	if nilLd.LanguageID() != "" || nilLd.FileCount() != 0 || nilLd.Percentage() != 0.0 {
		t.Error("nil LanguageDistribution should return safe defaults")
	}
}

func TestDiagnostic_Methods(t *testing.T) {
	diag := discovery.NewDiagnostic(discovery.SeverityWarning, "WARN_CODE", "warning message", "some/path", false)
	if diag.Severity() != discovery.SeverityWarning {
		t.Errorf("severity got %s, want %s", diag.Severity(), discovery.SeverityWarning)
	}
	if diag.Code() != "WARN_CODE" || diag.Message() != "warning message" || diag.Path() != "some/path" {
		t.Errorf("diagnostic fields mismatch")
	}
	if diag.IsFatal() {
		t.Error("expected isFatal=false")
	}
	if diag.String() == "" {
		t.Error("expected non-empty String()")
	}

	var nilDiag *discovery.Diagnostic
	if nilDiag.Severity() != discovery.SeverityInfo || nilDiag.Code() != "" || nilDiag.IsFatal() {
		t.Error("nil Diagnostic should return safe defaults")
	}
}

func TestMetadata_Methods(t *testing.T) {
	meta := discovery.NewMetadata(
		"test-repo",
		"C:\\test-repo",
		true,
		"main",
		"main",
		"abc1234",
		15,
		4,
		10240,
		[]string{"sub/repo"},
	)

	if meta.Name() != "test-repo" || meta.Root() != "C:\\test-repo" {
		t.Errorf("name/root mismatch")
	}
	if !meta.IsGit() || meta.CurrentBranch() != "main" || meta.DefaultBranch() != "main" || meta.LatestCommit() != "abc1234" {
		t.Errorf("git metadata mismatch")
	}
	if meta.TotalFiles() != 15 || meta.TotalDirectories() != 4 || meta.TotalBytes() != 10240 {
		t.Errorf("counts mismatch")
	}
	if len(meta.NestedRepositories()) != 1 {
		t.Errorf("nested repos mismatch")
	}
	if meta.String() == "" {
		t.Error("expected non-empty String()")
	}

	var nilMeta *discovery.Metadata
	if nilMeta.Name() != "" || nilMeta.TotalFiles() != 0 || nilMeta.IsGit() {
		t.Error("nil Metadata should return safe defaults")
	}
}
