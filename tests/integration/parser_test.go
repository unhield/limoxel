package integration_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/unhield/limoxel/internal/filesystem"
	"github.com/unhield/limoxel/internal/language"
	"github.com/unhield/limoxel/internal/parser"
	"github.com/unhield/limoxel/internal/project"
	"github.com/unhield/limoxel/internal/repository"
	"github.com/unhield/limoxel/internal/workspace"
)

func TestParserIntegration(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Setup real production language & parser registries
	langReg := language.NewRegistry()
	parserReg := parser.NewRegistry()

	goLang, _ := language.New("go", "Go", []string{".go"}, nil, []string{"golang"})
	pyLang, _ := language.New("python", "Python", []string{".py"}, nil, []string{"py"})
	_ = langReg.Register(goLang)
	_ = langReg.Register(pyLang)

	goParser, err := parser.NewDescriptor("go-parser", "Go Tree-Sitter Parser", "go", "1.0.0")
	if err != nil {
		t.Fatalf("parser.NewDescriptor for go failed: %v", err)
	}

	pyParser, err := parser.NewDescriptor("python-parser", "Python Tree-Sitter Parser", "python", "2.1.0")
	if err != nil {
		t.Fatalf("parser.NewDescriptor for python failed: %v", err)
	}

	if err := parserReg.Register(goParser); err != nil {
		t.Fatalf("Register goParser failed: %v", err)
	}
	if err := parserReg.Register(pyParser); err != nil {
		t.Fatalf("Register pyParser failed: %v", err)
	}

	t.Run("parser registration, activation, and lookup flow", func(t *testing.T) {
		if parserReg.Count() != 2 {
			t.Errorf("got count %d, want 2", parserReg.Count())
		}

		p1, err := parserReg.Get("go-parser")
		if err != nil || p1.Name() != "Go Tree-Sitter Parser" {
			t.Fatalf("Get(go-parser) failed: %v", err)
		}
		if p1.LanguageID() != "go" {
			t.Errorf("LanguageID got %q, want go", p1.LanguageID())
		}

		// Activate parser lifecycle
		if err := parserReg.Activate("go-parser"); err != nil {
			t.Fatalf("Activate go-parser failed: %v", err)
		}
		if !parserReg.IsActive("go-parser") {
			t.Error("expected go-parser to be active")
		}

		st, err := parserReg.State("go-parser")
		if err != nil || st != parser.StateActive {
			t.Errorf("got state %v, want StateActive", st)
		}
	})

	t.Run("repository parser resolution via file language inspection", func(t *testing.T) {
		osFs := filesystem.NewOSFilesystem()
		fileSer, err := filesystem.NewFileService(osFs)
		if err != nil {
			t.Fatalf("filesystem.NewFileService failed: %v", err)
		}

		wsDir := filepath.Join(tempDir, "parser_workspace")
		projDir := filepath.Join(wsDir, "app_project")
		repoDir := filepath.Join(projDir, "core_repo")

		_ = fileSer.EnsureDirectory(repoDir)

		ws, _ := workspace.New("parser-ws", wsDir)
		proj, _ := project.New("parser-proj", ws, "app_project")
		repo, _ := repository.New("parser-repo", proj, "src_repo")
		_ = fileSer.EnsureDirectory(repo.Root())

		// Write source files
		goFile := filepath.Join(repo.Root(), "server.go")
		pyFile := filepath.Join(repo.Root(), "app.py")
		txtFile := filepath.Join(repo.Root(), "README.txt")

		_ = fileSer.WriteFile(goFile, []byte("package main\nfunc main(){}"), 0644)
		_ = fileSer.WriteFile(pyFile, []byte("print('hello')"), 0644)
		_ = fileSer.WriteFile(txtFile, []byte("documentation"), 0644)

		disc, _ := filesystem.NewDiscoverer(repo.Root(), filesystem.NewIgnorer())
		discRes, _ := disc.Discover()

		resolvedParsers := make(map[string]string)

		for _, entry := range discRes.Entries() {
			if entry.IsDir() {
				continue
			}
			fileName := filepath.Base(entry.Path())
			lang, err := langReg.DiscoverByFilename(fileName)
			if err != nil {
				continue
			}

			// Find matching registered parser for detected language
			for _, desc := range parserReg.List() {
				if desc.LanguageID() == lang.ID() {
					resolvedParsers[fileName] = desc.ID()
					break
				}
			}
		}

		if resolvedParsers["server.go"] != "go-parser" {
			t.Errorf("server.go parser got %q, want go-parser", resolvedParsers["server.go"])
		}
		if resolvedParsers["app.py"] != "python-parser" {
			t.Errorf("app.py parser got %q, want python-parser", resolvedParsers["app.py"])
		}
		if _, exists := resolvedParsers["README.txt"]; exists {
			t.Error("README.txt should not resolve a parser")
		}
	})

	t.Run("duplicate parser registration error handling", func(t *testing.T) {
		err := parserReg.Register(goParser)
		if !errors.Is(err, parser.ErrDuplicateParser) {
			t.Errorf("got %v, want ErrDuplicateParser", err)
		}
	})

	t.Run("unsupported and missing parser error handling", func(t *testing.T) {
		_, err := parserReg.Get("missing-parser")
		if !errors.Is(err, parser.ErrParserNotFound) {
			t.Errorf("got %v, want ErrParserNotFound", err)
		}

		if err := parserReg.Activate("missing-parser"); !errors.Is(err, parser.ErrParserNotFound) {
			t.Errorf("got %v, want ErrParserNotFound", err)
		}
	})

	t.Run("parser removal and resource cleanup", func(t *testing.T) {
		tempParser, _ := parser.NewDescriptor("temp-parser", "Temp Parser", "go", "1.0")
		_ = parserReg.Register(tempParser)

		if !parserReg.Has("temp-parser") {
			t.Error("expected temp-parser to exist")
		}

		if err := parserReg.Remove("temp-parser"); err != nil {
			t.Fatalf("Remove temp-parser failed: %v", err)
		}

		if parserReg.Has("temp-parser") {
			t.Error("expected temp-parser to be removed")
		}
	})
}
