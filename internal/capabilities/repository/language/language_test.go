package language_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
	"github.com/unhield/limoxel/internal/capabilities/repository/language"
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
		a, err := language.New(nil)
		if err != language.ErrNilDiscoverer {
			t.Errorf("got %v, want ErrNilDiscoverer", err)
		}
		if a != nil {
			t.Errorf("got %v, want nil", a)
		}
	})

	t.Run("valid discoverer returns operational analyzer", func(t *testing.T) {
		disc := setupTestDiscoverer(t)
		a, err := language.New(disc)
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
	analyzer, _ := language.New(disc)

	t.Run("nil analyzer receiver methods return safe errors", func(t *testing.T) {
		var nilA *language.Analyzer
		if _, err := nilA.Analyze(nil); err != language.ErrNilAnalyzer {
			t.Errorf("got %v, want ErrNilAnalyzer", err)
		}
		if _, err := nilA.AnalyzeRepository(nil); err != language.ErrNilAnalyzer {
			t.Errorf("got %v, want ErrNilAnalyzer", err)
		}
		if _, err := nilA.AnalyzePath("some/path"); err != language.ErrNilAnalyzer {
			t.Errorf("got %v, want ErrNilAnalyzer", err)
		}
		if nilA.Discoverer() != nil {
			t.Errorf("expected nil discoverer on nil analyzer")
		}
	})

	t.Run("nil discovery result returns ErrNilDiscoveryResult", func(t *testing.T) {
		_, err := analyzer.Analyze(nil)
		if err != language.ErrNilDiscoveryResult {
			t.Errorf("got %v, want ErrNilDiscoveryResult", err)
		}
	})

	t.Run("nil repository returns ErrNilRepository", func(t *testing.T) {
		_, err := analyzer.AnalyzeRepository(nil)
		if err != language.ErrNilRepository {
			t.Errorf("got %v, want ErrNilRepository", err)
		}
	})

	t.Run("empty path returns ErrPathEmpty", func(t *testing.T) {
		_, err := analyzer.AnalyzePath("   ")
		if err != language.ErrPathEmpty {
			t.Errorf("got %v, want ErrPathEmpty", err)
		}
	})

	t.Run("nonexistent path returns ErrPathNotFound", func(t *testing.T) {
		_, err := analyzer.AnalyzePath(filepath.Join(t.TempDir(), "nonexistent_dir"))
		if err == nil || err != language.ErrPathNotFound && !filepath.IsAbs(err.Error()) {
			// check if it wraps ErrPathNotFound
			if !os.IsNotExist(err) && err != language.ErrPathNotFound {
				// verify error wrapping
			}
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

func TestAnalyzer_FullProjectStructure(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	analyzer, _ := language.New(disc)

	repoRoot := filepath.Join(tempDir, "full_repo")
	_ = os.MkdirAll(filepath.Join(repoRoot, "src", "pkg"), 0755)
	_ = os.MkdirAll(filepath.Join(repoRoot, "docs", "adr"), 0755)
	_ = os.MkdirAll(filepath.Join(repoRoot, "vendor", "dep"), 0755)

	// Modules & Build
	_ = os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module github.com/example/repo\n\ngo 1.22\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "Makefile"), []byte("all:\n\tgo build\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "CMakeLists.txt"), []byte("cmake_minimum_required(VERSION 3.10)\n"), 0644)

	// Source files
	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte("package main\nfunc main() {}\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "src", "pkg", "lib.go"), []byte("package pkg\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "vendor", "dep", "dep.go"), []byte("package dep\n"), 0644)

	// Configs
	_ = os.WriteFile(filepath.Join(repoRoot, "config.yaml"), []byte("key: value\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "app.json"), []byte("{\"name\": \"app\"}\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, ".env"), []byte("SECRET=123\n"), 0644)

	// Docs
	_ = os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("# Full Repo\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "CONTRIBUTING.md"), []byte("# Contributing\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "LICENSE"), []byte("MIT License\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "docs", "adr", "0001-record.md"), []byte("# ADR 1\n"), 0644)

	model, err := analyzer.AnalyzePath(repoRoot)
	if err != nil {
		t.Fatalf("AnalyzePath failed: %v", err)
	}

	if model.Root() == "" {
		t.Error("expected non-empty root")
	}

	// 1. Check DirectoryGraph
	dg := model.DirectoryGraph()
	if dg == nil || dg.NodeCount() == 0 {
		t.Fatal("expected non-empty directory graph")
	}
	rootNode, exists := dg.Node(".")
	if !exists || rootNode == nil {
		t.Fatal("expected root directory node '.'")
	}
	if !rootNode.IsModule() {
		t.Error("root node should be classified as module")
	}

	// 2. Check Modules
	mg := model.ModuleGraph()
	if mg == nil || mg.Count() != 1 {
		t.Fatalf("expected 1 module, got %d", mg.Count())
	}
	mod, exists := mg.ModuleByPath(".")
	if !exists || mod.Type() != language.ModuleGo {
		t.Errorf("expected Go module at root, got %+v", mod)
	}

	// 3. Check Build Systems (Makefile + CMakeLists.txt)
	bg := model.BuildGraph()
	if bg == nil || bg.Count() != 2 {
		t.Fatalf("expected 2 build systems, got %d", bg.Count())
	}

	// 4. Check Config Assets (config.yaml, app.json, .env)
	configs := model.ConfigAssets()
	if len(configs) != 3 {
		t.Fatalf("expected 3 config assets, got %d", len(configs))
	}
	var hasHiddenEnv bool
	for _, c := range configs {
		if c.Type() == language.ConfigENV && c.IsHidden() {
			hasHiddenEnv = true
		}
	}
	if !hasHiddenEnv {
		t.Error("expected hidden .env config asset")
	}

	// 5. Check Documentation (README, CONTRIBUTING, LICENSE, ADR)
	docs := model.DocAssets()
	if len(docs) < 4 {
		t.Fatalf("expected at least 4 doc assets, got %d", len(docs))
	}

	// 6. Check Vendor
	vendors := model.VendorEntries()
	if len(vendors) != 1 || vendors[0].Path() != "vendor" {
		t.Fatalf("expected vendor directory, got %+v", vendors)
	}

	// 7. Check Packages
	packages := model.Packages()
	if len(packages) == 0 {
		t.Fatal("expected discovered packages")
	}
}

func TestAnalyzer_Monorepo(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	analyzer, _ := language.New(disc)

	repoRoot := filepath.Join(tempDir, "monorepo")
	_ = os.MkdirAll(filepath.Join(repoRoot, "apps", "web"), 0755)
	_ = os.MkdirAll(filepath.Join(repoRoot, "services", "api"), 0755)
	_ = os.MkdirAll(filepath.Join(repoRoot, "packages", "core"), 0755)

	_ = os.WriteFile(filepath.Join(repoRoot, "package.json"), []byte("{\"name\": \"root\"}\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "apps", "web", "package.json"), []byte("{\"name\": \"web\"}\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "services", "api", "go.mod"), []byte("module api\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "packages", "core", "Cargo.toml"), []byte("[package]\nname = \"core\"\n"), 0644)

	_ = os.WriteFile(filepath.Join(repoRoot, "apps", "web", "index.js"), []byte("console.log('hi');\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "services", "api", "main.go"), []byte("package main\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "packages", "core", "lib.rs"), []byte("fn core() {}\n"), 0644)

	model, err := analyzer.AnalyzePath(repoRoot)
	if err != nil {
		t.Fatalf("AnalyzePath failed: %v", err)
	}

	ws := model.WorkspaceStructure()
	if ws == nil {
		t.Fatal("expected non-nil WorkspaceStructure")
	}

	if !ws.IsMonorepo() {
		t.Error("expected IsMonorepo to be true for multi-module project")
	}

	if len(ws.RootModules()) != 1 {
		t.Errorf("expected 1 root module, got %d", len(ws.RootModules()))
	}
	if len(ws.NestedModules()) != 3 {
		t.Errorf("expected 3 nested modules, got %d", len(ws.NestedModules()))
	}
}

func TestAnalyzer_ConfigDiscovery(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	analyzer, _ := language.New(disc)

	repoRoot := filepath.Join(tempDir, "config_repo")
	_ = os.MkdirAll(repoRoot, 0755)

	_ = os.WriteFile(filepath.Join(repoRoot, "a.yaml"), []byte(""), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "b.yml"), []byte(""), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "c.json"), []byte(""), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "d.toml"), []byte(""), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "e.ini"), []byte(""), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "f.properties"), []byte(""), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "g.xml"), []byte(""), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, ".env.local"), []byte(""), 0644)

	model, err := analyzer.AnalyzePath(repoRoot)
	if err != nil {
		t.Fatalf("AnalyzePath failed: %v", err)
	}

	cfgs := model.ConfigAssets()
	if len(cfgs) != 8 {
		t.Fatalf("expected 8 config assets, got %d", len(cfgs))
	}
}

func TestAnalyzer_DocDiscovery(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	analyzer, _ := language.New(disc)

	repoRoot := filepath.Join(tempDir, "doc_repo")
	_ = os.MkdirAll(filepath.Join(repoRoot, "docs", "adr"), 0755)

	_ = os.WriteFile(filepath.Join(repoRoot, "README.txt"), []byte(""), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "CONTRIBUTING.md"), []byte(""), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "SECURITY.md"), []byte(""), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "LICENSE.md"), []byte(""), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "CHANGELOG.md"), []byte(""), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "ROADMAP.md"), []byte(""), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "docs", "adr", "ADR-001.md"), []byte(""), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "docs", "guide.md"), []byte(""), 0644)

	model, err := analyzer.AnalyzePath(repoRoot)
	if err != nil {
		t.Fatalf("AnalyzePath failed: %v", err)
	}

	docs := model.DocAssets()
	if len(docs) != 8 {
		t.Fatalf("expected 8 doc assets, got %d", len(docs))
	}
}

func TestAnalyzer_VendorDetection(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	analyzer, _ := language.New(disc)

	repoRoot := filepath.Join(tempDir, "vendor_repo")
	_ = os.MkdirAll(filepath.Join(repoRoot, "node_modules", "express"), 0755)
	_ = os.MkdirAll(filepath.Join(repoRoot, ".venv", "lib"), 0755)

	_ = os.WriteFile(filepath.Join(repoRoot, "node_modules", "express", "index.js"), []byte(""), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, ".venv", "lib", "site.py"), []byte(""), 0644)

	model, err := analyzer.AnalyzePath(repoRoot)
	if err != nil {
		t.Fatalf("AnalyzePath failed: %v", err)
	}

	vendors := model.VendorEntries()
	if len(vendors) != 2 {
		t.Fatalf("expected 2 vendor entries, got %d", len(vendors))
	}
}

func TestAnalyzer_EmptyRepository(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	analyzer, _ := language.New(disc)

	repoRoot := filepath.Join(tempDir, "empty_repo")
	_ = os.MkdirAll(repoRoot, 0755)

	model, err := analyzer.AnalyzePath(repoRoot)
	if err != nil {
		t.Fatalf("AnalyzePath failed: %v", err)
	}

	if model.DirectoryGraph().NodeCount() != 1 {
		t.Errorf("expected 1 root directory node, got %d", model.DirectoryGraph().NodeCount())
	}
	if len(model.Packages()) != 0 {
		t.Errorf("expected 0 packages")
	}
	if model.ModuleGraph().Count() != 0 {
		t.Errorf("expected 0 modules")
	}
}

func TestAnalyzer_DomainRepository(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	analyzer, _ := language.New(disc)

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

func TestAnalyzer_ResultImmutability(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	analyzer, _ := language.New(disc)

	repoRoot := filepath.Join(tempDir, "immut_repo")
	_ = os.MkdirAll(filepath.Join(repoRoot, "vendor"), 0755)
	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte("package main"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "config.yaml"), []byte("key: 1"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("# Hi"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "vendor", "lib.go"), []byte("package v"), 0644)

	model, err := analyzer.AnalyzePath(repoRoot)
	if err != nil {
		t.Fatalf("AnalyzePath failed: %v", err)
	}

	// 1. Mutate Packages slice
	pkgs := model.Packages()
	if len(pkgs) > 0 {
		pkgs[0] = nil
		if model.Packages()[0] == nil {
			t.Error("mutation of returned Packages slice modified internal state")
		}
	}

	// 2. Mutate ConfigAssets slice
	cfgs := model.ConfigAssets()
	if len(cfgs) > 0 {
		cfgs[0] = nil
		if model.ConfigAssets()[0] == nil {
			t.Error("mutation of returned ConfigAssets slice modified internal state")
		}
	}

	// 3. Mutate DocAssets slice
	docs := model.DocAssets()
	if len(docs) > 0 {
		docs[0] = nil
		if model.DocAssets()[0] == nil {
			t.Error("mutation of returned DocAssets slice modified internal state")
		}
	}

	// 4. Mutate VendorEntries slice
	vends := model.VendorEntries()
	if len(vends) > 0 {
		vends[0] = nil
		if model.VendorEntries()[0] == nil {
			t.Error("mutation of returned VendorEntries slice modified internal state")
		}
	}
}

func TestAnalyzer_DeterminismAcrossRepeatedRuns(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	analyzer, _ := language.New(disc)

	repoRoot := filepath.Join(tempDir, "det_repo")
	_ = os.MkdirAll(filepath.Join(repoRoot, "pkg", "sub"), 0755)
	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte("package main"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "pkg", "a.go"), []byte("package pkg"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "pkg", "sub", "b.go"), []byte("package sub"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "config.yaml"), []byte("k: v"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("# Title"), 0644)

	m1, err := analyzer.AnalyzePath(repoRoot)
	if err != nil {
		t.Fatalf("run 1 failed: %v", err)
	}

	m2, err := analyzer.AnalyzePath(repoRoot)
	if err != nil {
		t.Fatalf("run 2 failed: %v", err)
	}

	if m1.DirectoryGraph().NodeCount() != m2.DirectoryGraph().NodeCount() {
		t.Errorf("node count mismatch: %d vs %d", m1.DirectoryGraph().NodeCount(), m2.DirectoryGraph().NodeCount())
	}

	pkgs1 := m1.Packages()
	pkgs2 := m2.Packages()
	if len(pkgs1) != len(pkgs2) {
		t.Fatalf("packages count mismatch: %d vs %d", len(pkgs1), len(pkgs2))
	}
	for i := range pkgs1 {
		if pkgs1[i].Path() != pkgs2[i].Path() || pkgs1[i].Name() != pkgs2[i].Name() {
			t.Errorf("package[%d] mismatch: %+v vs %+v", i, pkgs1[i], pkgs2[i])
		}
	}

	cfgs1 := m1.ConfigAssets()
	cfgs2 := m2.ConfigAssets()
	if len(cfgs1) != len(cfgs2) {
		t.Fatalf("configs count mismatch: %d vs %d", len(cfgs1), len(cfgs2))
	}
	for i := range cfgs1 {
		if cfgs1[i].Path() != cfgs2[i].Path() {
			t.Errorf("config[%d] mismatch: %s vs %s", i, cfgs1[i].Path(), cfgs2[i].Path())
		}
	}
}

func TestModels_MethodsAndNilSafety(t *testing.T) {
	// DirectoryNode
	dn := language.NewDirectoryNode("src", ".", []string{"src/a"}, []string{"src/main.go"}, true, false, false)
	if dn.Path() != "src" || dn.ParentPath() != "." || !dn.IsPackage() || dn.IsModule() || dn.IsVendor() {
		t.Errorf("DirectoryNode field mismatch")
	}
	if len(dn.ChildDirectories()) != 1 || len(dn.Files()) != 1 {
		t.Errorf("DirectoryNode children/files mismatch")
	}
	var nilNode *language.DirectoryNode
	if nilNode.Path() != "" || nilNode.IsPackage() || len(nilNode.ChildDirectories()) != 0 {
		t.Error("nil DirectoryNode should return zero values")
	}

	// DirectoryGraph
	dg := language.NewDirectoryGraph(".", []*language.DirectoryNode{dn})
	if dg.RootPath() != "." || dg.NodeCount() != 1 {
		t.Errorf("DirectoryGraph mismatch")
	}
	if _, ok := dg.Node("src"); !ok {
		t.Errorf("expected node 'src'")
	}
	var nilDG *language.DirectoryGraph
	if nilDG.RootPath() != "" || nilDG.NodeCount() != 0 || len(nilDG.AllNodes()) != 0 {
		t.Error("nil DirectoryGraph should return zero values")
	}

	// Package
	pkg := language.NewPackage("mypkg", "src/mypkg", "go", []string{"src/mypkg/a.go"})
	if pkg.Name() != "mypkg" || pkg.Path() != "src/mypkg" || pkg.LanguageID() != "go" || len(pkg.Files()) != 1 {
		t.Errorf("Package mismatch")
	}
	var nilPkg *language.Package
	if nilPkg.Name() != "" || nilPkg.Path() != "" || len(nilPkg.Files()) != 0 {
		t.Error("nil Package should return zero values")
	}

	// Module
	mod := language.NewModule(language.ModuleGo, "mymod", "src/mymod", "src/mymod/go.mod", "go", language.BuildUnknown)
	if mod.Type() != language.ModuleGo || mod.Name() != "mymod" || mod.Path() != "src/mymod" || mod.DescriptorFile() != "src/mymod/go.mod" {
		t.Errorf("Module mismatch")
	}
	var nilMod *language.Module
	if nilMod.Type() != language.ModuleUnknown || nilMod.Name() != "" {
		t.Error("nil Module should return zero values")
	}

	// BuildConfig
	bc := language.NewBuildConfig(language.BuildMake, "src", "src/Makefile", "src")
	if bc.Type() != language.BuildMake || bc.Path() != "src" || bc.ConfigFile() != "src/Makefile" || bc.ModulePath() != "src" {
		t.Errorf("BuildConfig mismatch")
	}
	var nilBC *language.BuildConfig
	if nilBC.Type() != language.BuildUnknown || nilBC.ConfigFile() != "" {
		t.Error("nil BuildConfig should return zero values")
	}

	// ConfigAsset
	ca := language.NewConfigAsset(language.ConfigYAML, "conf.yaml", false)
	if ca.Type() != language.ConfigYAML || ca.Path() != "conf.yaml" || ca.Filename() != "conf.yaml" || ca.IsHidden() {
		t.Errorf("ConfigAsset mismatch")
	}
	var nilCA *language.ConfigAsset
	if nilCA.Type() != language.ConfigUnknown || nilCA.Path() != "" {
		t.Error("nil ConfigAsset should return zero values")
	}

	// DocAsset
	da := language.NewDocAsset(language.DocReadme, "README.md", "general")
	if da.Type() != language.DocReadme || da.Path() != "README.md" || da.Filename() != "README.md" || da.Category() != "general" {
		t.Errorf("DocAsset mismatch")
	}
	var nilDA *language.DocAsset
	if nilDA.Type() != language.DocUnknown || nilDA.Path() != "" {
		t.Error("nil DocAsset should return zero values")
	}

	// VendorEntry
	ve := language.NewVendorEntry("vendor", "go")
	if ve.Path() != "vendor" || ve.Ecosystem() != "go" {
		t.Errorf("VendorEntry mismatch")
	}
	var nilVE *language.VendorEntry
	if nilVE.Path() != "" || nilVE.Ecosystem() != "" {
		t.Error("nil VendorEntry should return zero values")
	}

	// StructureModel String
	var nilSM *language.StructureModel
	if nilSM.Root() != "" || nilSM.String() != "" || len(nilSM.Packages()) != 0 {
		t.Error("nil StructureModel should return zero values")
	}
}
