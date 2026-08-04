package integration_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/unhield/limoxel/internal/extension"
	"github.com/unhield/limoxel/internal/filesystem"
	"github.com/unhield/limoxel/internal/language"
	"github.com/unhield/limoxel/internal/parser"
	"github.com/unhield/limoxel/internal/project"
	"github.com/unhield/limoxel/internal/repository"
	"github.com/unhield/limoxel/internal/workspace"
)

func TestExtensionIntegration(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Setup filesystem and directory structure for discovery
	osFs := filesystem.NewOSFilesystem()
	fileSer, err := filesystem.NewFileService(osFs)
	if err != nil {
		t.Fatalf("filesystem.NewFileService failed: %v", err)
	}

	wsDir := filepath.Join(tempDir, "ext_workspace")
	projDir := filepath.Join(wsDir, "ext_project")
	repoDir := filepath.Join(projDir, "ext_repo")
	extRootDir := filepath.Join(repoDir, ".limoxel", "extensions")

	extPath1 := filepath.Join(extRootDir, "ext-linter")
	extPath2 := filepath.Join(extRootDir, "ext-formatter")

	_ = fileSer.EnsureDirectory(extPath1)
	_ = fileSer.EnsureDirectory(extPath2)

	ws, _ := workspace.New("ext-ws", wsDir)
	proj, _ := project.New("ext-proj", ws, "ext_project")
	repo, _ := repository.New("ext-repo", proj, "ext_repo")

	// Setup language and parser registries
	langReg := language.NewRegistry()
	parserReg := parser.NewRegistry()

	goLang, _ := language.New("go", "Go", []string{".go"}, nil, []string{"golang"})
	_ = langReg.Register(goLang)

	goParser, _ := parser.NewDescriptor("go-parser", "Go Parser", "go", "1.0.0")
	_ = parserReg.Register(goParser)

	// 2. Discover extensions from filesystem directory inside repository
	discoverer, err := extension.NewDiscoverer(extPath1, extPath2)
	if err != nil {
		t.Fatalf("extension.NewDiscoverer failed: %v", err)
	}

	discRes, err := discoverer.Discover()
	if err != nil {
		t.Fatalf("discoverer.Discover failed: %v", err)
	}

	if discRes.Count() != 2 {
		t.Errorf("got discovered extensions count %d, want 2", discRes.Count())
	}

	t.Run("extension registration, activation, and metadata association", func(t *testing.T) {
		extReg := extension.NewRegistry()

		descLinter, err := extension.NewDescriptor("ext-linter", "Go Linter", "1.0.0", "Author A", "Lints Go code", map[string]string{
			"namespace": "org.limoxel.linter",
			"target":    "go",
			"parser":    "go-parser",
		})
		if err != nil {
			t.Fatalf("NewDescriptor ext-linter failed: %v", err)
		}

		descFormatter, err := extension.NewDescriptor("ext-formatter", "Go Formatter", "1.1.0", "Author B", "Formats Go code", map[string]string{
			"namespace": "org.limoxel.formatter",
			"target":    "go",
			"parser":    "go-parser",
		})
		if err != nil {
			t.Fatalf("NewDescriptor ext-formatter failed: %v", err)
		}

		if err := extReg.Register(descLinter); err != nil {
			t.Fatalf("Register descLinter failed: %v", err)
		}
		if err := extReg.Register(descFormatter); err != nil {
			t.Fatalf("Register descFormatter failed: %v", err)
		}

		if extReg.Count() != 2 {
			t.Errorf("got count %d, want 2", extReg.Count())
		}

		// Activate extensions
		if err := extReg.Activate("ext-linter"); err != nil {
			t.Fatalf("Activate ext-linter failed: %v", err)
		}
		if err := extReg.Activate("ext-formatter"); err != nil {
			t.Fatalf("Activate ext-formatter failed: %v", err)
		}

		if !extReg.IsActive("ext-linter") || !extReg.IsActive("ext-formatter") {
			t.Error("expected registered extensions to be active")
		}

		// Verify target language & parser existence in linked subsystems
		extObj, _ := extReg.Get("ext-linter")
		targetLangID := extObj.Metadata()["target"]
		targetParserID := extObj.Metadata()["parser"]

		if !langReg.Has(targetLangID) {
			t.Errorf("extension target language %q not found in language.Registry", targetLangID)
		}
		if !parserReg.Has(targetParserID) {
			t.Errorf("extension target parser %q not found in parser.Registry", targetParserID)
		}
	})

	t.Run("extension isolation and architectural boundary enforcement", func(t *testing.T) {
		validator := extension.NewIsolationValidator()
		d1, _ := extension.NewDescriptor("ext-linter", "Go Linter", "1.0.0", "Author", "Desc", map[string]string{"namespace": "org.limoxel.linter"})
		d2Colliding, _ := extension.NewDescriptor("ext-other", "Other Linter", "1.0.0", "Author", "Desc", map[string]string{"namespace": "org.limoxel.linter"})

		existing := []*extension.Descriptor{d1}

		err := validator.ValidateIsolation(d2Colliding, existing)
		if err == nil || !errors.Is(err, extension.ErrIsolationViolation) {
			t.Errorf("got error %v, want ErrIsolationViolation for namespace collision", err)
		}
	})

	t.Run("duplicate extension registration error handling", func(t *testing.T) {
		extReg := extension.NewRegistry()
		d1, _ := extension.NewDescriptor("ext-linter", "Go Linter", "1.0.0", "Author", "Desc", nil)
		_ = extReg.Register(d1)

		err := extReg.Register(d1)
		if !errors.Is(err, extension.ErrDuplicateExtension) {
			t.Errorf("got %v, want ErrDuplicateExtension", err)
		}
	})

	t.Run("deactivation, removal, and resource cleanup", func(t *testing.T) {
		extReg := extension.NewRegistry()
		d1, _ := extension.NewDescriptor("temp-ext", "Temp Ext", "1.0.0", "Author", "Desc", nil)
		_ = extReg.Register(d1)
		_ = extReg.Activate("temp-ext")

		if err := extReg.Deactivate("temp-ext"); err != nil {
			t.Fatalf("Deactivate temp-ext failed: %v", err)
		}
		if extReg.IsActive("temp-ext") {
			t.Error("expected temp-ext to be inactive")
		}

		if err := extReg.Remove("temp-ext"); err != nil {
			t.Fatalf("Remove temp-ext failed: %v", err)
		}

		if extReg.Has("temp-ext") {
			t.Error("expected temp-ext to be removed from registry")
		}

		// Verify workspace and repository references remain clean
		if !fileSer.Exists(repo.Root()) {
			t.Error("expected repository directory to remain intact after extension cleanup")
		}
	})
}
