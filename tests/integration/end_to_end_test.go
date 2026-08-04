package integration_test

import (
	"path/filepath"
	"testing"

	"github.com/unhield/limoxel/internal/cli"
	"github.com/unhield/limoxel/internal/engine"
	"github.com/unhield/limoxel/internal/extension"
	"github.com/unhield/limoxel/internal/filesystem"
	"github.com/unhield/limoxel/internal/language"
	"github.com/unhield/limoxel/internal/parser"
)

func TestEndToEndProductionWorkflow(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("complete end to end limoxel lifecycle pipeline", func(t *testing.T) {
		// 1. App Configuration & Bootstrap Initialization
		cfg, err := cli.NewConfig("limoxel-e2e", "1.0.0", tempDir)
		if err != nil {
			t.Fatalf("cli.NewConfig failed: %v", err)
		}

		boot, err := cli.NewBootstrap(cfg)
		if err != nil {
			t.Fatalf("cli.NewBootstrap failed: %v", err)
		}

		eng, err := boot.Initialize()
		if err != nil {
			t.Fatalf("boot.Initialize failed: %v", err)
		}

		if eng.State() != engine.StateRunning {
			t.Fatalf("got engine state %v, want StateRunning", eng.State())
		}

		// 2. Validate Engine Subsystem Wiring
		fileSer := eng.Filesystem()
		langReg := eng.LanguageRegistry()
		parserReg := eng.ParserRegistry()
		ws := eng.Workspace()
		repo := eng.Repository()

		if fileSer == nil || langReg == nil || parserReg == nil || ws == nil || repo == nil {
			t.Fatal("engine subsystem wiring incomplete: one or more core subsystems are nil")
		}

		// 3. Register Languages and Parsers
		goLang, err := language.New("go", "Go", []string{".go"}, nil, []string{"golang"})
		if err != nil {
			t.Fatalf("language.New failed: %v", err)
		}
		if err := langReg.Register(goLang); err != nil {
			t.Fatalf("langReg.Register failed: %v", err)
		}

		goParser, err := parser.NewDescriptor("go-parser", "Go Parser", "go", "1.0.0")
		if err != nil {
			t.Fatalf("parser.NewDescriptor failed: %v", err)
		}
		if err := parserReg.Register(goParser); err != nil {
			t.Fatalf("parserReg.Register failed: %v", err)
		}

		// 4. Register and Activate Extensions
		extReg := extension.NewRegistry()
		extDesc, err := extension.NewDescriptor("ext-analyzer", "Go Analyzer", "1.0.0", "Limoxel", "Analyzes Go source", map[string]string{
			"namespace": "org.limoxel.analyzer",
			"target":    "go",
			"parser":    "go-parser",
		})
		if err != nil {
			t.Fatalf("extension.NewDescriptor failed: %v", err)
		}
		if err := extReg.Register(extDesc); err != nil {
			t.Fatalf("extReg.Register failed: %v", err)
		}
		if err := extReg.Activate("ext-analyzer"); err != nil {
			t.Fatalf("extReg.Activate failed: %v", err)
		}

		// 5. Create Source File Structure inside Repository via FileService
		sourceFile := filepath.Join(repo.Root(), "main.go")
		if err := fileSer.WriteFile(sourceFile, []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
			t.Fatalf("fileSer.WriteFile failed: %v", err)
		}

		// 6. Discover and Inspect Source File across Subsystems
		disc, err := filesystem.NewDiscoverer(repo.Root(), filesystem.NewIgnorer())
		if err != nil {
			t.Fatalf("filesystem.NewDiscoverer failed: %v", err)
		}

		discRes, err := disc.Discover()
		if err != nil {
			t.Fatalf("disc.Discover failed: %v", err)
		}

		var discoveredSource bool
		for _, entry := range discRes.Entries() {
			if entry.IsDir() {
				continue
			}
			baseName := filepath.Base(entry.Path())
			if baseName == "main.go" {
				discoveredSource = true

				// Resolve Language
				detectedLang, err := langReg.DiscoverByFilename(baseName)
				if err != nil || detectedLang.ID() != "go" {
					t.Fatalf("language detection failed for main.go: %v", err)
				}

				// Resolve Parser
				boundParser, err := parserReg.Get("go-parser")
				if err != nil || boundParser.LanguageID() != detectedLang.ID() {
					t.Fatalf("parser binding failed for main.go: %v", err)
				}

				// Resolve Active Extension
				ext, err := extReg.Get("ext-analyzer")
				if err != nil || !extReg.IsActive("ext-analyzer") {
					t.Fatalf("extension retrieval failed: %v", err)
				}
				if ext.Metadata()["target"] != detectedLang.ID() || ext.Metadata()["parser"] != boundParser.ID() {
					t.Errorf("extension metadata binding mismatch: %v", ext.Metadata())
				}
			}
		}

		if !discoveredSource {
			t.Error("expected main.go to be discovered in repository root")
		}

		// 7. Graceful Engine Shutdown and Resource Cleanup
		if err := eng.Stop(); err != nil {
			t.Fatalf("eng.Stop failed: %v", err)
		}
		if eng.State() != engine.StateStopped {
			t.Errorf("got engine state %v after Stop(), want StateStopped", eng.State())
		}

		if err := eng.Terminate(); err != nil {
			t.Fatalf("eng.Terminate failed: %v", err)
		}
		if eng.State() != engine.StateTerminated {
			t.Errorf("got engine state %v after Terminate(), want StateTerminated", eng.State())
		}
	})
}
