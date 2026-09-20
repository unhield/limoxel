package tooling_test

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/unhield/limoxel/plugin/templates"
	"github.com/unhield/limoxel/plugin/tooling"
)

func TestGenerator_And_Debugger(t *testing.T) {
	tempDir := t.TempDir()
	outDir := filepath.Join(tempDir, "my-plugin")

	gen := tooling.NewGenerator()
	err := gen.Generate(tooling.GenerateOptions{
		Template:    templates.TemplateBasic,
		PluginID:    "com.example.generator",
		PluginName:  "GeneratorTest",
		Version:     "1.0.0",
		Publisher:   "Test Author",
		Description: "A generated plugin test",
		OutputDir:   outDir,
		Overwrite:   false,
	})
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	// Verify generated files exist
	for _, f := range []string{"plugin.json", "main.go", "README.md"} {
		path := filepath.Join(outDir, f)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected generated file %s to exist", f)
		}
	}

	// Overwrite guard test
	err = gen.Generate(tooling.GenerateOptions{
		Template:   templates.TemplateBasic,
		PluginID:   "com.example.generator",
		PluginName: "GeneratorTest",
		OutputDir:  outDir,
		Overwrite:  false,
	})
	if err == nil {
		t.Errorf("expected error when generating to existing directory without overwrite")
	}

	// Path traversal protection test
	err = gen.Generate(tooling.GenerateOptions{
		Template:  templates.TemplateBasic,
		PluginID:  "com.example.generator",
		OutputDir: tempDir + "/../escaped",
		Overwrite: true,
	})
	if err == nil {
		t.Errorf("expected error on path traversal in OutputDir")
	}

	// Debugger diagnostic test
	dbg := tooling.NewDebugger()
	session, err := dbg.Diagnose(context.Background(), tooling.DebugOptions{
		PluginDir: outDir,
		Verbose:   true,
	})
	if err != nil {
		t.Fatalf("diagnose failed: %v", err)
	}
	if session.Status != "PASS" {
		t.Errorf("expected session status PASS, got %s", session.Status)
	}
	if session.Manifest == nil || session.Manifest.ID != "com.example.generator" {
		t.Errorf("unexpected manifest in session: %+v", session.Manifest)
	}
}

func TestPackager(t *testing.T) {
	tempDir := t.TempDir()
	outDir := filepath.Join(tempDir, "pack-plugin")

	gen := tooling.NewGenerator()
	err := gen.Generate(tooling.GenerateOptions{
		Template:   templates.TemplateCLI,
		PluginID:   "com.example.cli",
		PluginName: "CLITest",
		Version:    "2.0.0",
		OutputDir:  outDir,
	})
	if err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	// Create a dummy binary
	binDir := filepath.Join(outDir, "bin")
	_ = os.MkdirAll(binDir, 0755)
	dummyBin := filepath.Join(binDir, "plugin-binary")
	if err := os.WriteFile(dummyBin, []byte("fake-binary-content"), 0755); err != nil {
		t.Fatalf("failed to write dummy binary: %v", err)
	}

	packager := tooling.NewPackager()
	pkgOut := filepath.Join(tempDir, "dist", "plugin.tar.gz")
	res, err := packager.Package(tooling.PackageOptions{
		ProjectDir: outDir,
		BinaryPath: dummyBin,
		OutputFile: pkgOut,
	})
	if err != nil {
		t.Fatalf("package failed: %v", err)
	}

	if res.ArchivePath != pkgOut {
		t.Errorf("archive path mismatch: %s", res.ArchivePath)
	}
	if res.ChecksumSHA256 == "" || res.Size <= 0 {
		t.Errorf("invalid package metrics: checksum=%s, size=%d", res.ChecksumSHA256, res.Size)
	}

	// Verify tar.gz contents
	f, err := os.Open(pkgOut)
	if err != nil {
		t.Fatalf("failed to open package archive: %v", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	foundFiles := make(map[string]bool)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar read error: %v", err)
		}
		foundFiles[hdr.Name] = true
	}

	if !foundFiles["plugin.json"] {
		t.Errorf("archive missing plugin.json")
	}
	if !foundFiles["README.md"] {
		t.Errorf("archive missing README.md")
	}
	if !foundFiles["bin/plugin-binary"] {
		t.Errorf("archive missing bin/plugin-binary")
	}
}

func TestBuilder_Validation(t *testing.T) {
	tempDir := t.TempDir()
	builder := tooling.NewBuilder()

	// Missing manifest
	_, err := builder.Build(context.Background(), tooling.BuildOptions{
		ProjectDir: tempDir,
	})
	if err == nil {
		t.Errorf("expected error when building directory with missing manifest")
	}

	// Invalid manifest
	_ = os.WriteFile(filepath.Join(tempDir, "plugin.json"), []byte("{invalid-json"), 0644)
	_, err = builder.Build(context.Background(), tooling.BuildOptions{
		ProjectDir: tempDir,
	})
	if err == nil {
		t.Errorf("expected error when building directory with malformed manifest")
	}

	// Empty ID
	_ = os.WriteFile(filepath.Join(tempDir, "plugin.json"), []byte(`{"id": "", "version": "1.0.0"}`), 0644)
	_, err = builder.Build(context.Background(), tooling.BuildOptions{
		ProjectDir: tempDir,
	})
	if err == nil {
		t.Errorf("expected error when building manifest with empty ID")
	}

	// Invalid GoBinary characters
	_ = os.WriteFile(filepath.Join(tempDir, "plugin.json"), []byte(`{"id": "valid.id", "version": "1.0.0"}`), 0644)
	_, err = builder.Build(context.Background(), tooling.BuildOptions{
		ProjectDir: tempDir,
		GoBinary:   "go; echo inject",
	})
	if err == nil {
		t.Errorf("expected error when building with injected GoBinary command, got nil")
	}
}

func TestPackager_PartialOutputCleanupOnFailure(t *testing.T) {
	tempDir := t.TempDir()
	outDir := filepath.Join(tempDir, "pack-plugin")
	_ = os.MkdirAll(outDir, 0755)
	_ = os.WriteFile(filepath.Join(outDir, "plugin.json"), []byte(`{"id":"clean.test","version":"1.0.0"}`), 0644)

	packager := tooling.NewPackager()
	pkgOut := filepath.Join(tempDir, "dist", "partial.tar.gz")

	// BinaryPath does not exist -> packaging fails
	_, err := packager.Package(tooling.PackageOptions{
		ProjectDir: outDir,
		BinaryPath: filepath.Join(tempDir, "nonexistent-binary"),
		OutputFile: pkgOut,
	})
	if err == nil {
		t.Fatal("expected package to fail for nonexistent binary, got nil")
	}

	// Verify partial output archive was cleaned up
	if _, statErr := os.Stat(pkgOut); statErr == nil {
		t.Errorf("partial output archive %s was not cleaned up after packaging error", pkgOut)
	}
}

func TestDebugger_EntrypointTraversalDetected(t *testing.T) {
	tempDir := t.TempDir()
	outDir := filepath.Join(tempDir, "traverse-plugin")
	_ = os.MkdirAll(outDir, 0755)

	manifestContent := []byte(`{
		"schema_version": "1.0.0",
		"id": "com.example.traverse",
		"name": "Traverse Test",
		"version": "1.0.0",
		"entrypoint": "../../../etc/passwd"
	}`)
	_ = os.WriteFile(filepath.Join(outDir, "plugin.json"), manifestContent, 0644)

	dbg := tooling.NewDebugger()
	session, err := dbg.Diagnose(context.Background(), tooling.DebugOptions{
		PluginDir: outDir,
	})
	if err != nil {
		t.Fatalf("diagnose failed: %v", err)
	}
	if session.Status != "FAIL" {
		t.Errorf("expected session status FAIL for entrypoint traversal, got %s", session.Status)
	}

	foundDiag := false
	for _, d := range session.Diagnostics {
		if d.Scope == "entrypoint" && d.Level == "ERROR" {
			foundDiag = true
			break
		}
	}
	if !foundDiag {
		t.Errorf("expected ERROR diagnostic for entrypoint traversal")
	}
}
