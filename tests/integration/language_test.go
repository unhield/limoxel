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

func TestLanguageIntegration(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Initialize real production language registry & populate standard languages
	langReg := language.NewRegistry()

	goLang, err := language.New("go", "Go", []string{".go"}, nil, []string{"golang"})
	if err != nil {
		t.Fatalf("failed creating Go language: %v", err)
	}

	pyLang, err := language.New("python", "Python", []string{".py"}, nil, []string{"py"})
	if err != nil {
		t.Fatalf("failed creating Python language: %v", err)
	}

	dockerLang, err := language.New("dockerfile", "Dockerfile", nil, []string{"Dockerfile"}, []string{"docker"})
	if err != nil {
		t.Fatalf("failed creating Dockerfile language: %v", err)
	}

	if err := langReg.Register(goLang); err != nil {
		t.Fatalf("Register goLang failed: %v", err)
	}
	if err := langReg.Register(pyLang); err != nil {
		t.Fatalf("Register pyLang failed: %v", err)
	}
	if err := langReg.Register(dockerLang); err != nil {
		t.Fatalf("Register dockerLang failed: %v", err)
	}

	t.Run("language registration and O(1) discovery lookups", func(t *testing.T) {
		if langReg.Count() != 3 {
			t.Errorf("got count %d, want 3", langReg.Count())
		}

		// Lookup by ID
		lGo, err := langReg.Get("go")
		if err != nil || lGo.Name() != "Go" {
			t.Errorf("Get(go) failed: %v", err)
		}

		// Discovery by Extension
		lPy, err := langReg.DiscoverByExtension(".py")
		if err != nil || lPy.ID() != "python" {
			t.Errorf("DiscoverByExtension(.py) failed: %v", err)
		}

		// Discovery by Alias
		lAlias, err := langReg.DiscoverByAlias("golang")
		if err != nil || lAlias.ID() != "go" {
			t.Errorf("DiscoverByAlias(golang) failed: %v", err)
		}

		// Discovery by exact Filename
		lDocker, err := langReg.DiscoverByFilename("Dockerfile")
		if err != nil || lDocker.ID() != "dockerfile" {
			t.Errorf("DiscoverByFilename(Dockerfile) failed: %v", err)
		}
	})

	t.Run("parser association with registered languages", func(t *testing.T) {
		parserReg := parser.NewRegistry()
		goParserDesc, err := parser.NewDescriptor("go-parser", "Go Parser", "go", "1.0.0")
		if err != nil {
			t.Fatalf("parser.NewDescriptor failed: %v", err)
		}

		if err := parserReg.Register(goParserDesc); err != nil {
			t.Fatalf("parserReg.Register failed: %v", err)
		}

		if !parserReg.Has("go-parser") {
			t.Error("expected parserReg to have go-parser")
		}

		retrievedParser, err := parserReg.Get("go-parser")
		if err != nil || retrievedParser.ID() != "go-parser" {
			t.Fatalf("parserReg.Get failed: %v", err)
		}
		if retrievedParser.LanguageID() != "go" {
			t.Errorf("target language mismatch: %q, want go", retrievedParser.LanguageID())
		}
	})

	t.Run("repository and project file language detection via filesystem inspection", func(t *testing.T) {
		osFs := filesystem.NewOSFilesystem()
		fileSer, err := filesystem.NewFileService(osFs)
		if err != nil {
			t.Fatalf("filesystem.NewFileService failed: %v", err)
		}

		wsDir := filepath.Join(tempDir, "lang_workspace")
		projDir := filepath.Join(wsDir, "proj_app")
		repoDir := filepath.Join(projDir, "src_repo")

		_ = fileSer.EnsureDirectory(repoDir)

		ws, _ := workspace.New("lang-ws", wsDir)
		proj, _ := project.New("lang-proj", ws, "proj_app")
		repo, err := repository.New("lang-repo", proj, "src_repo")
		if err != nil {
			t.Fatalf("repository.New failed: %v", err)
		}

		// Create files in repository
		_ = fileSer.WriteFile(filepath.Join(repo.Root(), "main.go"), []byte("package main"), 0644)
		_ = fileSer.WriteFile(filepath.Join(repo.Root(), "script.py"), []byte("print('hi')"), 0644)
		_ = fileSer.WriteFile(filepath.Join(repo.Root(), "Dockerfile"), []byte("FROM alpine"), 0644)
		_ = fileSer.WriteFile(filepath.Join(repo.Root(), "unknown.xyz"), []byte("data"), 0644)

		disc, err := filesystem.NewDiscoverer(repo.Root(), filesystem.NewIgnorer())
		if err != nil {
			t.Fatalf("filesystem.NewDiscoverer failed: %v", err)
		}

		discRes, err := disc.Discover()
		if err != nil {
			t.Fatalf("disc.Discover failed: %v", err)
		}

		detectedLangs := make(map[string]string)
		var unknownCount int

		for _, entry := range discRes.Entries() {
			if entry.IsDir() {
				continue
			}
			baseName := filepath.Base(entry.Path())
			lang, err := langReg.DiscoverByFilename(baseName)
			if err != nil {
				if errors.Is(err, language.ErrLanguageNotFound) {
					unknownCount++
				}
				continue
			}
			detectedLangs[baseName] = lang.ID()
		}

		if detectedLangs["main.go"] != "go" {
			t.Errorf("main.go language got %q, want go", detectedLangs["main.go"])
		}
		if detectedLangs["script.py"] != "python" {
			t.Errorf("script.py language got %q, want python", detectedLangs["script.py"])
		}
		if detectedLangs["Dockerfile"] != "dockerfile" {
			t.Errorf("Dockerfile language got %q, want dockerfile", detectedLangs["Dockerfile"])
		}
		if unknownCount < 1 {
			t.Error("expected unknown.xyz to trigger ErrLanguageNotFound")
		}
	})

	t.Run("duplicate language registration protection", func(t *testing.T) {
		err := langReg.Register(goLang)
		if !errors.Is(err, language.ErrDuplicateLanguage) {
			t.Errorf("got %v, want ErrDuplicateLanguage", err)
		}
	})

	t.Run("unsupported and unknown language error propagation", func(t *testing.T) {
		_, err := langReg.Get("unknown_lang")
		if !errors.Is(err, language.ErrLanguageNotFound) {
			t.Errorf("got %v, want ErrLanguageNotFound", err)
		}

		_, err = langReg.DiscoverByExtension(".unsupported")
		if !errors.Is(err, language.ErrLanguageNotFound) {
			t.Errorf("got %v, want ErrLanguageNotFound", err)
		}
	})

	t.Run("lifecycle termination cleanup", func(t *testing.T) {
		termReg := language.NewRegistry()
		_ = termReg.Register(goLang)

		if err := termReg.Close(); err != nil {
			t.Fatalf("termReg.Close failed: %v", err)
		}

		// Operations post termination should fail with ErrRegistryTerminated
		_, err := termReg.Get("go")
		if !errors.Is(err, language.ErrRegistryTerminated) {
			t.Errorf("got %v, want ErrRegistryTerminated", err)
		}
	})
}
