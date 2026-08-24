package dependency_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/repository/dependency"
	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
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
	rsLang, _ := langreg.New("rust", "Rust", []string{".rs"}, nil, []string{"rs"})
	javaLang, _ := langreg.New("java", "Java", []string{".java"}, nil, []string{"java"})

	_ = reg.Register(goLang)
	_ = reg.Register(pyLang)
	_ = reg.Register(tsLang)
	_ = reg.Register(jsLang)
	_ = reg.Register(rsLang)
	_ = reg.Register(javaLang)

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

func TestAnalyzer_New(t *testing.T) {
	t.Run("nil discoverer returns ErrNilDiscoverer", func(t *testing.T) {
		a, err := dependency.New(nil)
		if err != dependency.ErrNilDiscoverer {
			t.Errorf("got %v, want ErrNilDiscoverer", err)
		}
		if a != nil {
			t.Errorf("got %v, want nil", a)
		}
	})

	t.Run("valid discoverer returns operational analyzer", func(t *testing.T) {
		disc := setupTestDiscoverer(t)
		a, err := dependency.New(disc)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if a == nil || a.Discoverer() != disc {
			t.Fatalf("invalid analyzer state")
		}
	})
}

func TestAnalyzer_InputValidation(t *testing.T) {
	disc := setupTestDiscoverer(t)
	analyzer, _ := dependency.New(disc)

	t.Run("nil analyzer receiver methods return safe errors", func(t *testing.T) {
		var nilA *dependency.Analyzer
		if _, err := nilA.Analyze(nil); err != dependency.ErrNilAnalyzer {
			t.Errorf("got %v, want ErrNilAnalyzer", err)
		}
		if _, err := nilA.AnalyzeRepository(nil); err != dependency.ErrNilAnalyzer {
			t.Errorf("got %v, want ErrNilAnalyzer", err)
		}
		if _, err := nilA.AnalyzePath("some/path"); err != dependency.ErrNilAnalyzer {
			t.Errorf("got %v, want ErrNilAnalyzer", err)
		}
		if _, err := nilA.AnalyzeStructure(nil, nil); err != dependency.ErrNilAnalyzer {
			t.Errorf("got %v, want ErrNilAnalyzer", err)
		}
		if nilA.Discoverer() != nil {
			t.Errorf("expected nil discoverer on nil analyzer")
		}
	})

	t.Run("nil discovery result returns ErrNilDiscoveryResult", func(t *testing.T) {
		_, err := analyzer.Analyze(nil)
		if err != dependency.ErrNilDiscoveryResult {
			t.Errorf("got %v, want ErrNilDiscoveryResult", err)
		}
		if _, err := analyzer.AnalyzeStructure(nil, nil); err != dependency.ErrNilDiscoveryResult {
			t.Errorf("got %v, want ErrNilDiscoveryResult", err)
		}
	})

	t.Run("nil repository returns ErrNilRepository", func(t *testing.T) {
		_, err := analyzer.AnalyzeRepository(nil)
		if err != dependency.ErrNilRepository {
			t.Errorf("got %v, want ErrNilRepository", err)
		}
	})

	t.Run("empty path returns ErrPathEmpty", func(t *testing.T) {
		_, err := analyzer.AnalyzePath("   ")
		if err != dependency.ErrPathEmpty {
			t.Errorf("got %v, want ErrPathEmpty", err)
		}
	})

	t.Run("file path instead of directory returns ErrNotDirectory", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "file.txt")
		_ = os.WriteFile(filePath, []byte("hello"), 0644)

		_, err := analyzer.AnalyzePath(filePath)
		if err == nil {
			t.Fatal("expected ErrNotDirectory, got nil")
		}
	})
}

func TestAnalyzer_GoModuleDependencies(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	analyzer, _ := dependency.New(disc)

	repoRoot := filepath.Join(tempDir, "go_repo")
	_ = os.MkdirAll(repoRoot, 0755)

	goModContent := `module github.com/example/app

go 1.22

require (
	github.com/google/uuid v1.6.0
	golang.org/x/sync v0.7.0 // indirect
)
`
	_ = os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte(goModContent), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte("package main\nimport \"github.com/google/uuid\"\n"), 0644)

	model, err := analyzer.AnalyzePath(repoRoot)
	if err != nil {
		t.Fatalf("AnalyzePath failed: %v", err)
	}

	inv := model.Inventory()
	if inv.TotalCount() == 0 {
		t.Fatal("expected discovered dependencies")
	}

	directs := inv.DirectDependencies()
	if len(directs) != 1 || directs[0].Name() != "github.com/google/uuid" {
		t.Errorf("expected direct dependency github.com/google/uuid, got %+v", directs)
	}

	indirects := inv.IndirectDependencies()
	if len(indirects) != 1 || indirects[0].Name() != "golang.org/x/sync" {
		t.Errorf("expected indirect dependency golang.org/x/sync, got %+v", indirects)
	}
}

func TestAnalyzer_NpmDependencies(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	analyzer, _ := dependency.New(disc)

	repoRoot := filepath.Join(tempDir, "npm_repo")
	_ = os.MkdirAll(repoRoot, 0755)

	pkgJSON := `{
  "name": "my-app",
  "version": "1.0.0",
  "license": "MIT",
  "dependencies": {
    "express": "^4.19.2"
  },
  "devDependencies": {
    "typescript": "~5.4.5"
  }
}`
	_ = os.WriteFile(filepath.Join(repoRoot, "package.json"), []byte(pkgJSON), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "index.js"), []byte("const express = require('express');"), 0644)

	model, err := analyzer.AnalyzePath(repoRoot)
	if err != nil {
		t.Fatalf("AnalyzePath failed: %v", err)
	}

	inv := model.Inventory()
	if inv.TotalCount() != 2 {
		t.Fatalf("expected 2 dependencies, got %d", inv.TotalCount())
	}

	lics := model.Licenses()
	if lics.Count() != 1 || lics.Licenses()[0].Type() != dependency.LicenseMIT {
		t.Errorf("expected MIT license, got %+v", lics.Licenses())
	}
}

func TestAnalyzer_CargoDependencies(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	analyzer, _ := dependency.New(disc)

	repoRoot := filepath.Join(tempDir, "cargo_repo")
	_ = os.MkdirAll(repoRoot, 0755)

	cargoToml := `[package]
name = "my_crate"
version = "0.1.0"

[dependencies]
serde = "1.0.200"
tokio = { version = "1.38", features = ["full"] }

[dev-dependencies]
tempfile = "3.10"
`
	_ = os.WriteFile(filepath.Join(repoRoot, "Cargo.toml"), []byte(cargoToml), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "main.rs"), []byte("fn main() {}"), 0644)

	model, err := analyzer.AnalyzePath(repoRoot)
	if err != nil {
		t.Fatalf("AnalyzePath failed: %v", err)
	}

	inv := model.Inventory()
	if inv.TotalCount() != 3 {
		t.Fatalf("expected 3 cargo dependencies, got %d", inv.TotalCount())
	}
}

func TestAnalyzer_MavenAndGradleDependencies(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	analyzer, _ := dependency.New(disc)

	repoRoot := filepath.Join(tempDir, "java_repo")
	_ = os.MkdirAll(repoRoot, 0755)

	pomXML := `<project>
  <dependencies>
    <dependency>
      <groupId>org.springframework.boot</groupId>
      <artifactId>spring-boot-starter-web</artifactId>
      <version>3.2.5</version>
    </dependency>
  </dependencies>
</project>`
	_ = os.WriteFile(filepath.Join(repoRoot, "pom.xml"), []byte(pomXML), 0644)

	gradle := `dependencies {
    implementation 'com.google.guava:guava:33.1.0-jre'
}`
	_ = os.WriteFile(filepath.Join(repoRoot, "build.gradle"), []byte(gradle), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "App.java"), []byte("public class App {}"), 0644)

	model, err := analyzer.AnalyzePath(repoRoot)
	if err != nil {
		t.Fatalf("AnalyzePath failed: %v", err)
	}

	inv := model.Inventory()
	if inv.TotalCount() != 2 {
		t.Fatalf("expected 2 dependencies (1 maven, 1 gradle), got %d", inv.TotalCount())
	}
}

func TestAnalyzer_PythonAndComposerDependencies(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	analyzer, _ := dependency.New(disc)

	repoRoot := filepath.Join(tempDir, "py_php_repo")
	_ = os.MkdirAll(repoRoot, 0755)

	reqTxt := `requests==2.31.0
numpy>=1.26.0
`
	_ = os.WriteFile(filepath.Join(repoRoot, "requirements.txt"), []byte(reqTxt), 0644)

	composerJSON := `{
  "require": {
    "guzzlehttp/guzzle": "^7.8"
  }
}`
	_ = os.WriteFile(filepath.Join(repoRoot, "composer.json"), []byte(composerJSON), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "app.py"), []byte("import requests"), 0644)

	model, err := analyzer.AnalyzePath(repoRoot)
	if err != nil {
		t.Fatalf("AnalyzePath failed: %v", err)
	}

	inv := model.Inventory()
	if inv.TotalCount() != 3 {
		t.Fatalf("expected 3 dependencies (2 python, 1 composer), got %d", inv.TotalCount())
	}
}

func TestAnalyzer_InternalImportsAndGraph(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	analyzer, _ := dependency.New(disc)

	repoRoot := filepath.Join(tempDir, "graph_repo")
	_ = os.MkdirAll(filepath.Join(repoRoot, "pkgA"), 0755)
	_ = os.MkdirAll(filepath.Join(repoRoot, "pkgB"), 0755)
	_ = os.MkdirAll(filepath.Join(repoRoot, "pkgC"), 0755)

	_ = os.WriteFile(filepath.Join(repoRoot, "pkgA", "a.go"), []byte("package pkgA\nimport \"github.com/example/repo/pkgB\"\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "pkgB", "b.go"), []byte("package pkgB\nimport \"github.com/example/repo/pkgC\"\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "pkgC", "c.go"), []byte("package pkgC\n"), 0644)

	model, err := analyzer.AnalyzePath(repoRoot)
	if err != nil {
		t.Fatalf("AnalyzePath failed: %v", err)
	}

	graph := model.Graph()
	if graph == nil || graph.NodeCount() == 0 {
		t.Fatal("expected non-empty graph")
	}

	if model.MaxDepth() < 2 {
		t.Errorf("expected max depth at least 2, got %d", model.MaxDepth())
	}
}

func TestAnalyzer_CircularDependencyDetection(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	analyzer, _ := dependency.New(disc)

	repoRoot := filepath.Join(tempDir, "cycle_repo")
	_ = os.MkdirAll(filepath.Join(repoRoot, "pkgA"), 0755)
	_ = os.MkdirAll(filepath.Join(repoRoot, "pkgB"), 0755)
	_ = os.MkdirAll(filepath.Join(repoRoot, "pkgC"), 0755)

	// Cycle: A -> B -> C -> A
	_ = os.WriteFile(filepath.Join(repoRoot, "pkgA", "a.go"), []byte("package pkgA\nimport \"github.com/example/repo/pkgB\"\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "pkgB", "b.go"), []byte("package pkgB\nimport \"github.com/example/repo/pkgC\"\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "pkgC", "c.go"), []byte("package pkgC\nimport \"github.com/example/repo/pkgA\"\n"), 0644)

	model, err := analyzer.AnalyzePath(repoRoot)
	if err != nil {
		t.Fatalf("AnalyzePath failed: %v", err)
	}

	cycles := model.Cycles()
	if len(cycles) == 0 {
		t.Fatal("expected detected dependency cycle")
	}
}

func TestAnalyzer_OrphanPackageDetection(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	analyzer, _ := dependency.New(disc)

	repoRoot := filepath.Join(tempDir, "orphan_repo")
	_ = os.MkdirAll(filepath.Join(repoRoot, "active"), 0755)
	_ = os.MkdirAll(filepath.Join(repoRoot, "isolated"), 0755)

	_ = os.WriteFile(filepath.Join(repoRoot, "active", "main.go"), []byte("package active\nimport \"fmt\"\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "isolated", "tool.go"), []byte("package isolated\n"), 0644)

	model, err := analyzer.AnalyzePath(repoRoot)
	if err != nil {
		t.Fatalf("AnalyzePath failed: %v", err)
	}

	orphans := model.Orphans()
	var foundIsolated bool
	for _, o := range orphans {
		if o == "isolated" {
			foundIsolated = true
			break
		}
	}
	if !foundIsolated {
		t.Errorf("expected 'isolated' in orphans list, got %v", orphans)
	}
}

func TestAnalyzer_SemanticVersionParsing(t *testing.T) {
	v1 := dependency.ParseSemanticVersion("^1.2.3-alpha.1+build123")
	if !v1.IsValid() || !v1.IsConstraint() || v1.Major() != 1 || v1.Minor() != 2 || v1.Patch() != 3 {
		t.Errorf("v1 parsing failed: %+v", v1)
	}
	if v1.Prerelease() != "alpha.1" || v1.BuildMetadata() != "build123" {
		t.Errorf("v1 prerelease/build failed: %+v", v1)
	}
	if v1.String() == "" {
		t.Error("expected non-empty string")
	}

	v2 := dependency.ParseSemanticVersion("")
	if v2.IsValid() || v2.Raw() != "" {
		t.Errorf("empty version should be invalid: %+v", v2)
	}

	var nilV *dependency.SemanticVersion
	if nilV.Raw() != "" || nilV.Major() != 0 || nilV.IsValid() || nilV.String() != "" {
		t.Error("nil SemanticVersion should return safe zero values")
	}
}

func TestAnalyzer_ResultImmutability(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	analyzer, _ := dependency.New(disc)

	repoRoot := filepath.Join(tempDir, "immut_repo")
	_ = os.MkdirAll(repoRoot, 0755)
	_ = os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module myapp\n\nrequire rsc.io/quote v1.5.2\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte("package main\n"), 0644)

	model, err := analyzer.AnalyzePath(repoRoot)
	if err != nil {
		t.Fatalf("AnalyzePath failed: %v", err)
	}

	// 1. Mutate AllDependencies slice
	deps := model.Inventory().AllDependencies()
	if len(deps) > 0 {
		deps[0] = nil
		if model.Inventory().AllDependencies()[0] == nil {
			t.Error("mutation of returned AllDependencies slice modified internal state")
		}
	}

	// 2. Mutate Graph Nodes slice
	nodes := model.Graph().Nodes()
	if len(nodes) > 0 {
		nodes[0] = nil
		if model.Graph().Nodes()[0] == nil {
			t.Error("mutation of returned Graph Nodes slice modified internal state")
		}
	}

	// 3. Mutate Graph Edges slice
	edges := model.Graph().Edges()
	if len(edges) > 0 {
		edges[0] = nil
		if model.Graph().Edges()[0] == nil {
			t.Error("mutation of returned Graph Edges slice modified internal state")
		}
	}
}

func TestAnalyzer_DeterminismAcrossRepeatedRuns(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	analyzer, _ := dependency.New(disc)

	repoRoot := filepath.Join(tempDir, "det_repo")
	_ = os.MkdirAll(filepath.Join(repoRoot, "pkgA"), 0755)
	_ = os.MkdirAll(filepath.Join(repoRoot, "pkgB"), 0755)

	_ = os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module det\n\nrequire github.com/a/b v1.0.0\nrequire github.com/c/d v2.0.0\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "pkgA", "a.go"), []byte("package pkgA\nimport \"det/pkgB\"\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "pkgB", "b.go"), []byte("package pkgB\n"), 0644)

	m1, err := analyzer.AnalyzePath(repoRoot)
	if err != nil {
		t.Fatalf("run 1 failed: %v", err)
	}

	m2, err := analyzer.AnalyzePath(repoRoot)
	if err != nil {
		t.Fatalf("run 2 failed: %v", err)
	}

	if m1.Inventory().TotalCount() != m2.Inventory().TotalCount() {
		t.Errorf("dependency count mismatch: %d vs %d", m1.Inventory().TotalCount(), m2.Inventory().TotalCount())
	}

	deps1 := m1.Inventory().AllDependencies()
	deps2 := m2.Inventory().AllDependencies()
	for i := range deps1 {
		if deps1[i].Name() != deps2[i].Name() || deps1[i].DeclaredVersion() != deps2[i].DeclaredVersion() {
			t.Errorf("dep[%d] mismatch: %+v vs %+v", i, deps1[i], deps2[i])
		}
	}

	if m1.Graph().NodeCount() != m2.Graph().NodeCount() {
		t.Errorf("node count mismatch: %d vs %d", m1.Graph().NodeCount(), m2.Graph().NodeCount())
	}
	if m1.Graph().EdgeCount() != m2.Graph().EdgeCount() {
		t.Errorf("edge count mismatch: %d vs %d", m1.Graph().EdgeCount(), m2.Graph().EdgeCount())
	}
}

func TestAnalyzer_DomainRepository(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	analyzer, _ := dependency.New(disc)

	ws, _ := workspace.New("dom-ws", tempDir)
	proj, _ := project.New("dom-proj", ws, "my_app")
	repoRoot := filepath.Join(tempDir, "my_app", "my_repo")
	_ = os.MkdirAll(repoRoot, 0755)
	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte("package main"), 0644)

	repo, err := repository.New("my_repo", proj, repoRoot)
	if err != nil {
		t.Fatalf("repository.New failed: %v", err)
	}

	model, err := analyzer.AnalyzeRepository(repo)
	if err != nil {
		t.Fatalf("AnalyzeRepository failed: %v", err)
	}

	if model.Root() != filepath.ToSlash(filepath.Clean(repoRoot)) {
		t.Errorf("root mismatch: got %s, want %s", model.Root(), repoRoot)
	}
}

func TestAnalyzer_ScannerErrors(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	analyzer, _ := dependency.New(disc)

	repoRoot := filepath.Join(tempDir, "corrupt_gomod_repo")
	_ = os.MkdirAll(repoRoot, 0755)

	// Create oversized line in go.mod exceeding 64KB without newline
	longLine := make([]byte, 70000)
	for i := range longLine {
		longLine[i] = 'a'
	}
	_ = os.WriteFile(filepath.Join(repoRoot, "go.mod"), longLine, 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte("package main"), 0644)

	model, err := analyzer.AnalyzePath(repoRoot)
	if err != nil {
		t.Fatalf("AnalyzePath failed: %v", err)
	}

	var foundDiag bool
	for _, d := range model.Diagnostics() {
		if d.Code() == "GOMOD_SCAN_ERROR" {
			foundDiag = true
			break
		}
	}
	if !foundDiag {
		t.Error("expected GOMOD_SCAN_ERROR diagnostic when go.mod scan fails")
	}
}

func TestModels_MethodsAndNilSafety(t *testing.T) {
	// Dependency
	dep := dependency.NewDependency(
		"express",
		"^4.19.0",
		dependency.EcosystemNpm,
		dependency.DependencyDirect,
		true,
		false,
		false,
		true,
		"package.json",
		".",
		dependency.NewLicenseInfo(dependency.LicenseMIT, "MIT", "package.json", true),
		dependency.NewHealthInfo(dependency.HealthActive, false, false, false, true, 1.0, []string{"maintained"}),
	)

	if dep.Name() != "express" || dep.DeclaredVersion() != "^4.19.0" || dep.Ecosystem() != dependency.EcosystemNpm {
		t.Errorf("Dependency fields mismatch")
	}
	if !dep.IsDirect() || dep.IsIndirect() || dep.IsInternal() || !dep.IsExternal() {
		t.Errorf("Dependency boolean flags mismatch")
	}
	if dep.License() == nil || dep.License().Type() != dependency.LicenseMIT {
		t.Errorf("Dependency license mismatch")
	}
	if dep.Health() == nil || !dep.Health().IsActive() || len(dep.Health().Signals()) != 1 {
		t.Errorf("Dependency health mismatch")
	}
	if dep.String() == "" {
		t.Error("expected non-empty dependency string")
	}

	var nilDep *dependency.Dependency
	if nilDep.Name() != "" || nilDep.DeclaredVersion() != "" || nilDep.IsDirect() || nilDep.License() != nil || nilDep.String() != "" {
		t.Error("nil Dependency should return zero values")
	}

	// InternalImport
	imp := dependency.NewInternalImport("pkgA", "pkgB", "pkgA/a.go", "go")
	if imp.SourcePackage() != "pkgA" || imp.TargetPackage() != "pkgB" || imp.SourceFile() != "pkgA/a.go" || imp.LanguageID() != "go" {
		t.Errorf("InternalImport mismatch")
	}
	var nilImp *dependency.InternalImport
	if nilImp.SourcePackage() != "" || nilImp.TargetPackage() != "" {
		t.Error("nil InternalImport should return zero values")
	}

	// GraphNode
	node := dependency.NewGraphNode("n1", "Node1", true, dependency.EcosystemGo)
	if node.ID() != "n1" || node.Name() != "Node1" || !node.IsInternal() || node.Ecosystem() != dependency.EcosystemGo {
		t.Errorf("GraphNode mismatch")
	}
	var nilNode *dependency.GraphNode
	if nilNode.ID() != "" || nilNode.Name() != "" || nilNode.IsInternal() {
		t.Error("nil GraphNode should return zero values")
	}

	// GraphEdge
	edge := dependency.NewGraphEdge("n1", "n2", dependency.DependencyDirect)
	if edge.SourceID() != "n1" || edge.TargetID() != "n2" || edge.RelationshipType() != dependency.DependencyDirect {
		t.Errorf("GraphEdge mismatch")
	}
	var nilEdge *dependency.GraphEdge
	if nilEdge.SourceID() != "" || nilEdge.TargetID() != "" {
		t.Error("nil GraphEdge should return zero values")
	}

	// LicenseInfo
	lic := dependency.NewLicenseInfo(dependency.LicenseApache2, "Apache-2.0", "LICENSE", true)
	if lic.Type() != dependency.LicenseApache2 || lic.Expression() != "Apache-2.0" || lic.Source() != "LICENSE" || !lic.IsAvailable() {
		t.Errorf("LicenseInfo mismatch")
	}
	var nilLic *dependency.LicenseInfo
	if nilLic.Type() != dependency.LicenseUnavailable || nilLic.Expression() != "" || nilLic.IsAvailable() {
		t.Error("nil LicenseInfo should return zero values")
	}

	// HealthInfo
	health := dependency.NewHealthInfo(dependency.HealthDeprecated, false, true, false, false, 0.2, []string{"deprecated"})
	if health.Status() != dependency.HealthDeprecated || !health.IsDeprecated() || health.IsArchived() || health.IsActive() || health.HealthScore() != 0.2 {
		t.Errorf("HealthInfo mismatch")
	}
	var nilHealth *dependency.HealthInfo
	if nilHealth.Status() != dependency.HealthUnknown || nilHealth.IsDeprecated() || nilHealth.HealthScore() != 0.0 {
		t.Error("nil HealthInfo should return zero values")
	}

	// DependencyModel
	var nilModel *dependency.DependencyModel
	if nilModel.Root() != "" || nilModel.Inventory() != nil || nilModel.MaxDepth() != 0 || nilModel.String() != "" {
		t.Error("nil DependencyModel should return zero values")
	}
}
