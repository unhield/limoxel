package indexing_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
	"github.com/unhield/limoxel/internal/capabilities/repository/indexing"
	langreg "github.com/unhield/limoxel/internal/language"
	"github.com/unhield/limoxel/internal/project"
	"github.com/unhield/limoxel/internal/repository"
	"github.com/unhield/limoxel/internal/workspace"
)

func setupTestLanguageRegistry(t *testing.T) *langreg.Registry {
	t.Helper()
	reg := langreg.NewRegistry()

	goLang, _ := langreg.New("go", "Go", []string{".go"}, nil, []string{"golang"})
	pyLang, _ := langreg.New("python", "Python", []string{".py"}, nil, []string{"py"})
	tsLang, _ := langreg.New("typescript", "TypeScript", []string{".ts", ".tsx"}, nil, []string{"ts"})
	jsLang, _ := langreg.New("javascript", "JavaScript", []string{".js", ".jsx"}, nil, []string{"js"})

	_ = reg.Register(goLang)
	_ = reg.Register(pyLang)
	_ = reg.Register(tsLang)
	_ = reg.Register(jsLang)

	return reg
}

func setupTestDiscoverer(t *testing.T) *discovery.Discoverer {
	t.Helper()
	reg := setupTestLanguageRegistry(t)
	disc, err := discovery.New(reg)
	if err != nil {
		t.Fatalf("failed creating discoverer: %v", err)
	}
	return disc
}

func TestIndexer_New(t *testing.T) {
	t.Run("nil discoverer returns ErrNilDiscoverer", func(t *testing.T) {
		idx, err := indexing.New(nil)
		if err != indexing.ErrNilDiscoverer {
			t.Errorf("got %v, want ErrNilDiscoverer", err)
		}
		if idx != nil {
			t.Errorf("got %v, want nil", idx)
		}
	})

	t.Run("valid discoverer returns operational indexer", func(t *testing.T) {
		disc := setupTestDiscoverer(t)
		idx, err := indexing.New(disc)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if idx == nil || idx.Discoverer() != disc {
			t.Fatalf("invalid indexer state")
		}
	})
}

func TestIndexer_InputValidation(t *testing.T) {
	disc := setupTestDiscoverer(t)
	indexer, _ := indexing.New(disc)

	t.Run("nil indexer receiver methods return safe errors", func(t *testing.T) {
		var nilIdx *indexing.Indexer
		if _, err := nilIdx.Index(nil); err != indexing.ErrNilIndexer {
			t.Errorf("got %v, want ErrNilIndexer", err)
		}
		if _, err := nilIdx.IndexIncremental(nil, nil); err != indexing.ErrNilIndexer {
			t.Errorf("got %v, want ErrNilIndexer", err)
		}
		if _, err := nilIdx.IndexRepository(nil); err != indexing.ErrNilIndexer {
			t.Errorf("got %v, want ErrNilIndexer", err)
		}
		if _, err := nilIdx.IndexPath("some/path"); err != indexing.ErrNilIndexer {
			t.Errorf("got %v, want ErrNilIndexer", err)
		}
		if nilIdx.Discoverer() != nil {
			t.Errorf("expected nil discoverer on nil indexer")
		}
	})

	t.Run("nil discovery result returns ErrNilDiscoveryResult", func(t *testing.T) {
		_, err := indexer.Index(nil)
		if err != indexing.ErrNilDiscoveryResult {
			t.Errorf("got %v, want ErrNilDiscoveryResult", err)
		}
	})

	t.Run("nil repository returns ErrNilRepository", func(t *testing.T) {
		_, err := indexer.IndexRepository(nil)
		if err != indexing.ErrNilRepository {
			t.Errorf("got %v, want ErrNilRepository", err)
		}
	})

	t.Run("empty path returns ErrPathEmpty", func(t *testing.T) {
		_, err := indexer.IndexPath("   ")
		if err != indexing.ErrPathEmpty {
			t.Errorf("got %v, want ErrPathEmpty", err)
		}
	})

	t.Run("file path instead of directory returns ErrNotDirectory", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "file.txt")
		_ = os.WriteFile(filePath, []byte("hello"), 0644)

		_, err := indexer.IndexPath(filePath)
		if err == nil {
			t.Fatal("expected ErrNotDirectory, got nil")
		}
	})
}

func TestIndexer_GoSourceAndPackageIndexing(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	indexer, _ := indexing.New(disc)

	repoRoot := filepath.Join(tempDir, "go_repo")
	pkgDir := filepath.Join(repoRoot, "pkg", "mathutil")
	_ = os.MkdirAll(pkgDir, 0755)

	mathGo := `// Package mathutil provides mathematical utility functions.
package mathutil

import (
	"fmt"
	"math"
)

// Add performs integer addition.
func Add(a, b int) int {
	return a + b
}

type Calculator struct {
	Base float64
}

const Pi = 3.14159
`
	_ = os.WriteFile(filepath.Join(pkgDir, "math.go"), []byte(mathGo), 0644)

	model, err := indexer.IndexPath(repoRoot)
	if err != nil {
		t.Fatalf("IndexPath failed: %v", err)
	}

	if len(model.Files()) != 1 {
		t.Fatalf("expected 1 file, got %d", len(model.Files()))
	}

	f := model.Files()[0]
	if f.LanguageID() != "go" || f.FileType() != indexing.FileTypeSource || f.IsTest() {
		t.Errorf("unexpected file attributes: %+v", f)
	}
	if f.LineCount() < 10 || f.CodeLineCount() == 0 {
		t.Errorf("unexpected line counts: total=%d, code=%d", f.LineCount(), f.CodeLineCount())
	}

	if len(model.Packages()) != 1 {
		t.Fatalf("expected 1 package, got %d", len(model.Packages()))
	}

	p := model.Packages()[0]
	if p.Name() != "mathutil" {
		t.Errorf("expected package name mathutil, got %s", p.Name())
	}
	if p.Doc() != "Package mathutil provides mathematical utility functions." {
		t.Errorf("unexpected package doc: %s", p.Doc())
	}
	if len(p.Imports()) != 2 {
		t.Errorf("expected 2 imports, got %v", p.Imports())
	}
	if len(p.Exports()) != 3 {
		t.Errorf("expected 3 exports (Add, Calculator, Pi), got %v", p.Exports())
	}
}

func TestIndexer_TestAndGeneratedFiles(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	indexer, _ := indexing.New(disc)

	repoRoot := filepath.Join(tempDir, "gen_repo")
	_ = os.MkdirAll(repoRoot, 0755)

	mainGo := `package main

func main() {}
`
	testGo := `package main

import "testing"

func TestMain(t *testing.T) {}
`
	protoGo := `// Code generated by protoc-gen-go. DO NOT EDIT.
package main

type Message struct{}
`

	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte(mainGo), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "main_test.go"), []byte(testGo), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "msg.pb.go"), []byte(protoGo), 0644)

	model, err := indexer.IndexPath(repoRoot)
	if err != nil {
		t.Fatalf("IndexPath failed: %v", err)
	}

	testF := model.FileByPath("main_test.go")
	if testF == nil || !testF.IsTest() || testF.FileType() != indexing.FileTypeTest {
		t.Errorf("test file classification failed: %+v", testF)
	}

	protoF := model.FileByPath("msg.pb.go")
	if protoF == nil || protoF.GenerationStatus() != indexing.GenerationStatusGenerated {
		t.Errorf("generated file classification failed: %+v", protoF)
	}

	handF := model.FileByPath("main.go")
	if handF == nil || handF.GenerationStatus() != indexing.GenerationStatusHandwritten {
		t.Errorf("handwritten file classification failed: %+v", handF)
	}
}

func TestIndexer_EncodingAndLineEndings(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	indexer, _ := indexing.New(disc)

	repoRoot := filepath.Join(tempDir, "enc_repo")
	_ = os.MkdirAll(repoRoot, 0755)

	lfContent := "line1\nline2\nline3\n"
	crlfContent := "line1\r\nline2\r\nline3\r\n"
	mixedContent := "line1\r\nline2\nline3\r\n"

	_ = os.WriteFile(filepath.Join(repoRoot, "lf.go"), []byte(lfContent), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "crlf.go"), []byte(crlfContent), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "mixed.go"), []byte(mixedContent), 0644)

	model, err := indexer.IndexPath(repoRoot)
	if err != nil {
		t.Fatalf("IndexPath failed: %v", err)
	}

	lfF := model.FileByPath("lf.go")
	if lfF == nil || lfF.LineEnding() != indexing.LineEndingLF {
		t.Errorf("expected LF, got %+v", lfF)
	}

	crlfF := model.FileByPath("crlf.go")
	if crlfF == nil || crlfF.LineEnding() != indexing.LineEndingCRLF {
		t.Errorf("expected CRLF, got %+v", crlfF)
	}

	mixedF := model.FileByPath("mixed.go")
	if mixedF == nil || mixedF.LineEnding() != indexing.LineEndingMixed {
		t.Errorf("expected Mixed, got %+v", mixedF)
	}
}

func TestIndexer_FileRelationships(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	indexer, _ := indexing.New(disc)

	repoRoot := filepath.Join(tempDir, "rel_repo")
	pkgDir := filepath.Join(repoRoot, "pkg", "service")
	_ = os.MkdirAll(pkgDir, 0755)

	_ = os.WriteFile(filepath.Join(pkgDir, "service.go"), []byte("package service\nfunc Run() {}\n"), 0644)
	_ = os.WriteFile(filepath.Join(pkgDir, "service_test.go"), []byte("package service\nfunc TestRun() {}\n"), 0644)
	_ = os.WriteFile(filepath.Join(pkgDir, "config.json"), []byte("{\"port\": 8080}"), 0644)
	_ = os.WriteFile(filepath.Join(pkgDir, "README.md"), []byte("# Service"), 0644)

	model, err := indexer.IndexPath(repoRoot)
	if err != nil {
		t.Fatalf("IndexPath failed: %v", err)
	}

	rels := model.Relationships()
	if len(rels) == 0 {
		t.Fatal("expected discovered relationships")
	}

	var hasTestToSrc, hasDocToMod, hasCfgToSrc, hasPkgMembership bool
	for _, r := range rels {
		switch r.Type() {
		case indexing.RelTestToSource:
			hasTestToSrc = true
		case indexing.RelDocToModule:
			hasDocToMod = true
		case indexing.RelConfigToSource:
			hasCfgToSrc = true
		case indexing.RelPackageMembership:
			hasPkgMembership = true
		}
	}

	if !hasTestToSrc {
		t.Error("missing RelTestToSource relationship")
	}
	if !hasDocToMod {
		t.Error("missing RelDocToModule relationship")
	}
	if !hasCfgToSrc {
		t.Error("missing RelConfigToSource relationship")
	}
	if !hasPkgMembership {
		t.Error("missing RelPackageMembership relationship")
	}
}

func TestIndexer_RepositoryAndPackageStatistics(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	indexer, _ := indexing.New(disc)

	repoRoot := filepath.Join(tempDir, "stats_repo")
	_ = os.MkdirAll(repoRoot, 0755)

	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte("package main\n\n// comment\nfunc main() {}\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "main_test.go"), []byte("package main\nfunc TestMain() {}\n"), 0644)

	model, err := indexer.IndexPath(repoRoot)
	if err != nil {
		t.Fatalf("IndexPath failed: %v", err)
	}

	stats := model.Stats()
	if stats.TotalFiles() != 2 {
		t.Errorf("expected 2 files, got %d", stats.TotalFiles())
	}
	if stats.TotalPackages() != 1 {
		t.Errorf("expected 1 package, got %d", stats.TotalPackages())
	}
	if stats.TotalLines() == 0 || stats.CodeLines() == 0 {
		t.Errorf("line counts invalid: total=%d, code=%d", stats.TotalLines(), stats.CodeLines())
	}
	if stats.StructuralTestCoverage() != 1.0 {
		t.Errorf("expected 1.0 test coverage, got %f", stats.StructuralTestCoverage())
	}
}

func TestIndexer_SerializationAndDeserialization(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	indexer, _ := indexing.New(disc)

	repoRoot := filepath.Join(tempDir, "ser_repo")
	_ = os.MkdirAll(repoRoot, 0755)
	_ = os.WriteFile(filepath.Join(repoRoot, "app.go"), []byte("package main\nfunc Start() {}\n"), 0644)

	model, err := indexer.IndexPath(repoRoot)
	if err != nil {
		t.Fatalf("IndexPath failed: %v", err)
	}

	bytes, err := indexing.Serialize(model)
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	restored, err := indexing.Deserialize(bytes)
	if err != nil {
		t.Fatalf("Deserialize failed: %v", err)
	}

	if restored.SchemaVersion() != model.SchemaVersion() {
		t.Errorf("schema version mismatch: %s vs %s", restored.SchemaVersion(), model.SchemaVersion())
	}
	if len(restored.Files()) != len(model.Files()) {
		t.Errorf("file count mismatch: %d vs %d", len(restored.Files()), len(model.Files()))
	}
	if len(restored.Packages()) != len(model.Packages()) {
		t.Errorf("package count mismatch: %d vs %d", len(restored.Packages()), len(model.Packages()))
	}

	// Test schema version incompatibility
	incompatibleJSON := `{"schema_version": "99.0.0", "repository_root": "foo"}`
	if _, err := indexing.Deserialize([]byte(incompatibleJSON)); !errors.Is(err, indexing.ErrIncompatibleSchema) {
		t.Errorf("expected ErrIncompatibleSchema, got %v", err)
	}

	// Test corrupted payload
	if _, err := indexing.Deserialize([]byte("{invalid-json")); err == nil {
		t.Error("expected error for corrupted JSON, got nil")
	}
}

func TestIndexer_IncrementalIndexingEquivalence(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	indexer, _ := indexing.New(disc)

	repoRoot := filepath.Join(tempDir, "incr_repo")
	_ = os.MkdirAll(repoRoot, 0755)
	_ = os.WriteFile(filepath.Join(repoRoot, "fileA.go"), []byte("package main\nfunc A() {}\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "fileB.go"), []byte("package main\nfunc B() {}\n"), 0644)

	disc1, _ := disc.DiscoverPath(repoRoot)
	fullModel1, err := indexer.Index(disc1)
	if err != nil {
		t.Fatalf("run 1 failed: %v", err)
	}

	// Run incremental index using previous model without file changes
	incrModel1, err := indexer.IndexIncremental(disc1, fullModel1)
	if err != nil {
		t.Fatalf("incremental 1 failed: %v", err)
	}

	if len(incrModel1.Files()) != len(fullModel1.Files()) {
		t.Errorf("file count mismatch between full and incremental: %d vs %d", len(incrModel1.Files()), len(fullModel1.Files()))
	}

	// Modify one file and add a new file
	_ = os.WriteFile(filepath.Join(repoRoot, "fileB.go"), []byte("package main\nfunc B_Modified() {}\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "fileC.go"), []byte("package main\nfunc C() {}\n"), 0644)

	disc2, _ := disc.DiscoverPath(repoRoot)
	fullModel2, _ := indexer.Index(disc2)
	incrModel2, _ := indexer.IndexIncremental(disc2, incrModel1)

	// Verify CRITICAL ACCEPTANCE PROPERTY: incremental index == clean full index
	if len(incrModel2.Files()) != len(fullModel2.Files()) {
		t.Errorf("file counts differ after changes: %d vs %d", len(incrModel2.Files()), len(fullModel2.Files()))
	}
	for i := range fullModel2.Files() {
		fFull := fullModel2.Files()[i]
		fIncr := incrModel2.Files()[i]
		if fFull.RelPath() != fIncr.RelPath() || fFull.ContentHash() != fIncr.ContentHash() || fFull.LineCount() != fIncr.LineCount() {
			t.Errorf("file[%d] mismatch between full and incremental: %+v vs %+v", i, fFull, fIncr)
		}
	}
	if incrModel2.Stats().TotalLines() != fullModel2.Stats().TotalLines() {
		t.Errorf("stats total lines differ: %d vs %d", incrModel2.Stats().TotalLines(), fullModel2.Stats().TotalLines())
	}
}

func TestIndexer_ResultImmutability(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	indexer, _ := indexing.New(disc)

	repoRoot := filepath.Join(tempDir, "immut_repo")
	_ = os.MkdirAll(repoRoot, 0755)
	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte("package main\n"), 0644)

	model, err := indexer.IndexPath(repoRoot)
	if err != nil {
		t.Fatalf("IndexPath failed: %v", err)
	}

	// 1. Mutate Files slice
	files := model.Files()
	if len(files) > 0 {
		files[0] = nil
		if model.Files()[0] == nil {
			t.Error("mutation of returned Files slice modified internal state")
		}
	}

	// 2. Mutate Packages slice
	pkgs := model.Packages()
	if len(pkgs) > 0 {
		pkgs[0] = nil
		if model.Packages()[0] == nil {
			t.Error("mutation of returned Packages slice modified internal state")
		}
	}

	// 3. Mutate Relationships slice
	rels := model.Relationships()
	if len(rels) > 0 {
		rels[0] = nil
		if model.Relationships()[0] == nil {
			t.Error("mutation of returned Relationships slice modified internal state")
		}
	}
}

func TestIndexer_DeterminismAcrossRepeatedRuns(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	indexer, _ := indexing.New(disc)

	repoRoot := filepath.Join(tempDir, "det_repo")
	_ = os.MkdirAll(repoRoot, 0755)
	_ = os.WriteFile(filepath.Join(repoRoot, "z.go"), []byte("package main\nfunc Z() {}\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "a.go"), []byte("package main\nfunc A() {}\n"), 0644)

	m1, err := indexer.IndexPath(repoRoot)
	if err != nil {
		t.Fatalf("run 1 failed: %v", err)
	}

	m2, err := indexer.IndexPath(repoRoot)
	if err != nil {
		t.Fatalf("run 2 failed: %v", err)
	}

	if len(m1.Files()) != len(m2.Files()) {
		t.Fatalf("file count mismatch: %d vs %d", len(m1.Files()), len(m2.Files()))
	}
	for i := range m1.Files() {
		if m1.Files()[i].RelPath() != m2.Files()[i].RelPath() {
			t.Errorf("file[%d] ordering mismatch: %s vs %s", i, m1.Files()[i].RelPath(), m2.Files()[i].RelPath())
		}
	}
}

func TestIndexer_DomainRepository(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	indexer, _ := indexing.New(disc)

	ws, _ := workspace.New("dom-ws", tempDir)
	proj, _ := project.New("dom-proj", ws, "my_app")
	repoRoot := filepath.Join(tempDir, "my_app", "my_repo")
	_ = os.MkdirAll(repoRoot, 0755)
	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte("package main\n"), 0644)

	repo, err := repository.New("my_repo", proj, repoRoot)
	if err != nil {
		t.Fatalf("repository.New failed: %v", err)
	}

	model, err := indexer.IndexRepository(repo)
	if err != nil {
		t.Fatalf("IndexRepository failed: %v", err)
	}

	if model.RepositoryRoot() != filepath.ToSlash(filepath.Clean(repoRoot)) {
		t.Errorf("root mismatch: got %s, want %s", model.RepositoryRoot(), repoRoot)
	}
}

func TestIndexer_FastLookup(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	indexer, _ := indexing.New(disc)

	repoRoot := filepath.Join(tempDir, "lookup_repo")
	pkgDir := filepath.Join(repoRoot, "core")
	_ = os.MkdirAll(pkgDir, 0755)
	_ = os.WriteFile(filepath.Join(pkgDir, "core.go"), []byte("package core\nfunc Run() {}\n"), 0644)

	model, err := indexer.IndexPath(repoRoot)
	if err != nil {
		t.Fatalf("IndexPath failed: %v", err)
	}

	f := model.FileByPath("core/core.go")
	if f == nil || f.RelPath() != "core/core.go" {
		t.Errorf("FileByPath failed: %+v", f)
	}

	p := model.PackageByPath("core")
	if p == nil || p.Name() != "core" {
		t.Errorf("PackageByPath failed: %+v", p)
	}

	pkgFiles := model.FilesForPackage("core")
	if len(pkgFiles) != 1 || pkgFiles[0].RelPath() != "core/core.go" {
		t.Errorf("FilesForPackage failed: %+v", pkgFiles)
	}

	srcRels := model.RelationshipsForSource("core/core.go")
	if len(srcRels) == 0 {
		t.Errorf("RelationshipsForSource failed: %v", srcRels)
	}

	tgtRels := model.RelationshipsForTarget("core")
	if len(tgtRels) == 0 {
		t.Errorf("RelationshipsForTarget failed: %v", tgtRels)
	}
}

func TestModels_MethodsAndNilSafety(t *testing.T) {
	// IndexedFile
	f := indexing.NewIndexedFile("id1", "main.go", indexing.FileTypeSource, "go", false, indexing.GenerationStatusHandwritten, 100, "hash123", indexing.EncodingUTF8, indexing.LineEndingLF, 10, 2, 1)
	if f.ID() != "id1" || f.RelPath() != "main.go" || f.FileType() != indexing.FileTypeSource || f.LanguageID() != "go" || f.IsTest() {
		t.Errorf("IndexedFile field mismatch")
	}
	if f.GenerationStatus() != indexing.GenerationStatusHandwritten || f.SizeBytes() != 100 || f.ContentHash() != "hash123" {
		t.Errorf("IndexedFile metadata mismatch")
	}
	if f.Encoding() != indexing.EncodingUTF8 || f.LineEnding() != indexing.LineEndingLF || f.LineCount() != 10 || f.BlankLineCount() != 2 || f.CommentLineCount() != 1 || f.CodeLineCount() != 7 {
		t.Errorf("IndexedFile line metrics mismatch")
	}

	var nilF *indexing.IndexedFile
	if nilF.ID() != "" || nilF.RelPath() != "" || nilF.FileType() != indexing.FileTypeUnknown || nilF.LineCount() != 0 || nilF.CodeLineCount() != 0 {
		t.Error("nil IndexedFile should return zero values")
	}

	// PackageStats
	ps := indexing.NewPackageStats(5, 2, 1, 500, 10000)
	if ps.SourceFiles() != 5 || ps.TestFiles() != 2 || ps.GeneratedFiles() != 1 || ps.TotalLines() != 500 || ps.SizeBytes() != 10000 {
		t.Errorf("PackageStats mismatch")
	}
	var nilPS *indexing.PackageStats
	if nilPS.SourceFiles() != 0 || nilPS.TotalLines() != 0 {
		t.Error("nil PackageStats should return zero values")
	}

	// IndexedPackage
	pkg := indexing.NewIndexedPackage("mypkg", "pkg/mypkg", "pkg/mypkg", []string{"pkg/mypkg/a.go"}, []string{"fmt"}, []string{"MyFunc"}, "doc", "owner", ps)
	if pkg.Name() != "mypkg" || pkg.Path() != "pkg/mypkg" || pkg.Doc() != "doc" || pkg.Ownership() != "owner" || pkg.Stats() != ps {
		t.Errorf("IndexedPackage mismatch")
	}
	var nilPkg *indexing.IndexedPackage
	if nilPkg.Name() != "" || nilPkg.Path() != "" || nilPkg.Stats() != nil {
		t.Error("nil IndexedPackage should return zero values")
	}

	// FileRelationship
	rel := indexing.NewFileRelationship("src", "tgt", indexing.RelImport, "ev")
	if rel.SourceID() != "src" || rel.TargetID() != "tgt" || rel.Type() != indexing.RelImport || rel.Evidence() != "ev" {
		t.Errorf("FileRelationship mismatch")
	}
	var nilRel *indexing.FileRelationship
	if nilRel.SourceID() != "" || nilRel.Type() != indexing.RelUnknown {
		t.Error("nil FileRelationship should return zero values")
	}

	// IndexModel
	var nilModel *indexing.IndexModel
	if nilModel.SchemaVersion() != "" || nilModel.RepositoryRoot() != "" || nilModel.Stats() != nil || nilModel.String() != "" {
		t.Error("nil IndexModel should return zero values")
	}
}

func TestIndexer_ScannerErrors(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	indexer, _ := indexing.New(disc)

	repoRoot := filepath.Join(tempDir, "oversized_repo")
	_ = os.MkdirAll(repoRoot, 0755)

	// Create oversized single line > 64KB without newline in Go file
	oversized := make([]byte, 70000)
	for i := range oversized {
		oversized[i] = 'x'
	}
	_ = os.WriteFile(filepath.Join(repoRoot, "oversized.go"), oversized, 0644)

	model, err := indexer.IndexPath(repoRoot)
	if err != nil {
		t.Fatalf("IndexPath should not fail on oversized file: %v", err)
	}

	if len(model.Files()) != 1 {
		t.Errorf("expected 1 indexed file, got %d", len(model.Files()))
	}
}

func TestIndexer_AdversarialDuplicateBasenamesAndSameContent(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	indexer, _ := indexing.New(disc)

	repoRoot := filepath.Join(tempDir, "dup_repo")
	pkgA := filepath.Join(repoRoot, "pkgA")
	pkgB := filepath.Join(repoRoot, "pkgB")
	_ = os.MkdirAll(pkgA, 0755)
	_ = os.MkdirAll(pkgB, 0755)

	// Same basename & identical content in different directories
	sameContent := []byte("package shared\nfunc Helper() {}\n")
	_ = os.WriteFile(filepath.Join(pkgA, "config.go"), sameContent, 0644)
	_ = os.WriteFile(filepath.Join(pkgB, "config.go"), sameContent, 0644)

	model, err := indexer.IndexPath(repoRoot)
	if err != nil {
		t.Fatalf("IndexPath failed: %v", err)
	}

	if len(model.Files()) != 2 {
		t.Fatalf("expected 2 distinct indexed files, got %d", len(model.Files()))
	}

	fA := model.FileByPath("pkgA/config.go")
	fB := model.FileByPath("pkgB/config.go")

	if fA == nil || fB == nil {
		t.Fatalf("both files must be queryable by their relative paths")
	}
	if fA.RelPath() == fB.RelPath() {
		t.Errorf("relative paths must not be conflated")
	}
	if fA.ContentHash() != fB.ContentHash() {
		t.Errorf("identical content must yield identical content hashes: %s vs %s", fA.ContentHash(), fB.ContentHash())
	}
}

func TestIndexer_AdversarialMalformedAndBinaryFiles(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	indexer, _ := indexing.New(disc)

	repoRoot := filepath.Join(tempDir, "mixed_repo")
	_ = os.MkdirAll(repoRoot, 0755)

	// 1. Binary file with null bytes
	binaryData := []byte{0x7f, 'E', 'L', 'F', 0x00, 0x01, 0x02}
	_ = os.WriteFile(filepath.Join(repoRoot, "lib.so"), binaryData, 0644)

	// 2. Empty file
	_ = os.WriteFile(filepath.Join(repoRoot, "empty.txt"), []byte{}, 0644)

	// 3. Invalid UTF-8 sequence
	invalidUTF8 := []byte{0xFF, 0xFE, 0xFD, 0x80, 0x81}
	_ = os.WriteFile(filepath.Join(repoRoot, "corrupted.dat"), invalidUTF8, 0644)

	// 4. Valid Go file alongside malformed files
	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte("package main\nfunc main() {}\n"), 0644)

	model, err := indexer.IndexPath(repoRoot)
	if err != nil {
		t.Fatalf("mixed repository indexing should succeed: %v", err)
	}

	if len(model.Files()) != 4 {
		t.Errorf("expected 4 indexed files, got %d", len(model.Files()))
	}

	binF := model.FileByPath("lib.so")
	if binF == nil || binF.FileType() != indexing.FileTypeBinary {
		t.Errorf("expected binary classification for lib.so, got %+v", binF)
	}

	emptyF := model.FileByPath("empty.txt")
	if emptyF == nil || emptyF.LineCount() != 0 || emptyF.SizeBytes() != 0 {
		t.Errorf("empty file metrics invalid: %+v", emptyF)
	}
}

func TestIndexer_AdversarialGenerationMarkers(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	indexer, _ := indexing.New(disc)

	repoRoot := filepath.Join(tempDir, "marker_repo")
	_ = os.MkdirAll(repoRoot, 0755)

	// False positive comment
	fpContent := `package main
// This algorithm was originally generated by an idea from the author.
func HumanWritten() {}
`
	// True positive comment
	tpContent := `// Code generated by tool. DO NOT EDIT.
package main
func Auto() {}
`
	_ = os.WriteFile(filepath.Join(repoRoot, "human.go"), []byte(fpContent), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "auto.go"), []byte(tpContent), 0644)

	model, err := indexer.IndexPath(repoRoot)
	if err != nil {
		t.Fatalf("IndexPath failed: %v", err)
	}

	humanF := model.FileByPath("human.go")
	if humanF == nil || humanF.GenerationStatus() != indexing.GenerationStatusHandwritten {
		t.Errorf("human.go should be handwritten, got %v", humanF.GenerationStatus())
	}

	autoF := model.FileByPath("auto.go")
	if autoF == nil || autoF.GenerationStatus() != indexing.GenerationStatusGenerated {
		t.Errorf("auto.go should be generated, got %v", autoF.GenerationStatus())
	}
}
