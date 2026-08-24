package language

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
	"github.com/unhield/limoxel/internal/repository"
)

// Analyzer coordinates deterministic repository project structure and language analysis.
type Analyzer struct {
	discoverer *discovery.Discoverer
}

// New constructs and validates an immutable Analyzer instance.
func New(discoverer *discovery.Discoverer) (*Analyzer, error) {
	if discoverer == nil {
		return nil, ErrNilDiscoverer
	}
	return &Analyzer{
		discoverer: discoverer,
	}, nil
}

// Discoverer returns the underlying repository Discoverer.
func (a *Analyzer) Discoverer() *discovery.Discoverer {
	if a == nil {
		return nil
	}
	return a.discoverer
}

// Analyze compiles a complete StructureModel from an existing discovery result.
func (a *Analyzer) Analyze(discResult *discovery.Result) (*StructureModel, error) {
	if a == nil {
		return nil, ErrNilAnalyzer
	}
	if discResult == nil {
		return nil, ErrNilDiscoveryResult
	}

	root := discResult.Root()
	files := discResult.Files()

	// 1. Build directory graph and map files to directories
	dirMap := make(map[string][]string)
	dirChildMap := make(map[string]map[string]struct{})

	// Ensure root directory node exists
	dirMap["."] = nil
	dirChildMap["."] = make(map[string]struct{})

	for _, f := range files {
		relPath := f.RelPath()
		dir := filepath.ToSlash(filepath.Dir(relPath))
		if dir == "" {
			dir = "."
		}

		dirMap[dir] = append(dirMap[dir], relPath)

		// Register parent-child directory relationships up to root "."
		current := dir
		for current != "." && current != "" {
			parent := filepath.ToSlash(filepath.Dir(current))
			if parent == "" {
				parent = "."
			}
			if _, ok := dirChildMap[parent]; !ok {
				dirChildMap[parent] = make(map[string]struct{})
			}
			dirChildMap[parent][current] = struct{}{}
			if _, ok := dirMap[parent]; !ok {
				dirMap[parent] = nil
			}
			current = parent
		}
	}

	// 2. Discover modules, build configurations, configs, and docs from file inventory
	var (
		modules    []*Module
		buildCfgs  []*BuildConfig
		configs    []*ConfigAsset
		docs       []*DocAsset
		vendorDirs = make(map[string]string)
		pkgMap     = make(map[string]*Package)
	)

	// Check candidate vendor directories at repository root
	commonVendors := []string{"vendor", "node_modules", "bower_components", ".venv", "venv"}
	for _, vName := range commonVendors {
		vPath := filepath.Join(root, vName)
		if info, err := os.Stat(vPath); err == nil && info.IsDir() {
			vendorDirs[vName] = classifyVendorEcosystem(vName)
		}
	}

	for _, f := range files {
		relPath := f.RelPath()
		dir := filepath.ToSlash(filepath.Dir(relPath))
		if dir == "" {
			dir = "."
		}
		baseName := filepath.Base(relPath)
		ext := strings.ToLower(filepath.Ext(baseName))

		// Check vendor directories from discovered file paths
		if isVendorPath(relPath) {
			topVendor := extractVendorRoot(relPath)
			if topVendor != "" {
				vendorDirs[topVendor] = classifyVendorEcosystem(topVendor)
			}
		}

		// A. Module Descriptors
		if mod := classifyModule(baseName, dir, relPath); mod != nil {
			modules = append(modules, mod)
		}

		// B. Build Systems
		if bConfig := classifyBuildSystem(baseName, dir, relPath); bConfig != nil {
			buildCfgs = append(buildCfgs, bConfig)
		}

		// C. Configuration Assets
		if cfgAsset := classifyConfigAsset(baseName, relPath, ext); cfgAsset != nil {
			configs = append(configs, cfgAsset)
		}

		// D. Documentation Assets
		if docAsset := classifyDocAsset(baseName, relPath, ext); docAsset != nil {
			docs = append(docs, docAsset)
		}

		// E. Structural Packages (group by directory and language)
		if f.Language() != nil && f.Language().ID() != "" {
			langID := f.Language().ID()
			pkgKey := dir + ":" + langID
			if existing, ok := pkgMap[pkgKey]; ok {
				existing.files = append(existing.files, relPath)
			} else {
				pkgName := filepath.Base(dir)
				if dir == "." {
					pkgName = filepath.Base(root)
				}
				pkgMap[pkgKey] = &Package{
					name:       pkgName,
					path:       dir,
					languageID: langID,
					files:      []string{relPath},
				}
			}
		}
	}

	// 3. Assemble DirectoryNodes and DirectoryGraph
	var dirNodes []*DirectoryNode
	for dirPath, dirFiles := range dirMap {
		var children []string
		if childSet, ok := dirChildMap[dirPath]; ok {
			for ch := range childSet {
				children = append(children, ch)
			}
		}

		// Check if directory is module, package, or vendor
		var isMod, isPkg, isVend bool
		for _, m := range modules {
			if m.Path() == dirPath {
				isMod = true
				break
			}
		}
		for _, p := range pkgMap {
			if p.Path() == dirPath {
				isPkg = true
				break
			}
		}
		if _, ok := vendorDirs[dirPath]; ok {
			isVend = true
		}

		parent := filepath.ToSlash(filepath.Dir(dirPath))
		if dirPath == "." {
			parent = ""
		} else if parent == "" {
			parent = "."
		}

		node := NewDirectoryNode(dirPath, parent, children, dirFiles, isPkg, isMod, isVend)
		dirNodes = append(dirNodes, node)
	}
	dirGraph := NewDirectoryGraph(root, dirNodes)

	// 4. Assemble Packages
	var packages []*Package
	for _, p := range pkgMap {
		packages = append(packages, NewPackage(p.name, p.path, p.languageID, p.files))
	}

	// 5. Assemble ModuleGraph & WorkspaceStructure
	moduleGraph := NewModuleGraph(modules)
	var rootMods, nestedMods []*Module
	for _, m := range modules {
		if m.Path() == "." || m.Path() == "" {
			rootMods = append(rootMods, m)
		} else {
			nestedMods = append(nestedMods, m)
		}
	}
	workspace := NewWorkspaceStructure(rootMods, nestedMods)

	// 6. Assemble Vendors
	var vendors []*VendorEntry
	for vPath, eco := range vendorDirs {
		vendors = append(vendors, NewVendorEntry(vPath, eco))
	}
	sort.Slice(vendors, func(i, j int) bool {
		return vendors[i].path < vendors[j].path
	})

	// 7. Assemble BuildGraph
	buildGraph := NewBuildGraph(buildCfgs)

	// 8. Construct consolidated StructureModel
	structure := NewStructureModel(
		root,
		dirGraph,
		packages,
		moduleGraph,
		workspace,
		vendors,
		buildGraph,
		configs,
		docs,
		discResult.Languages(),
		discResult.Diagnostics(),
	)

	return structure, nil
}

// AnalyzeRepository executes discovery on an existing Repository instance and analyzes its structure.
func (a *Analyzer) AnalyzeRepository(repo *repository.Repository) (*StructureModel, error) {
	if a == nil {
		return nil, ErrNilAnalyzer
	}
	if repo == nil {
		return nil, ErrNilRepository
	}

	discResult, err := a.discoverer.Discover(repo)
	if err != nil {
		return nil, fmt.Errorf("%w: discovery failed: %v", ErrAnalysisFailed, err)
	}

	return a.Analyze(discResult)
}

// AnalyzePath discovers and analyzes the repository structure for a target directory path.
func (a *Analyzer) AnalyzePath(path string) (*StructureModel, error) {
	if a == nil {
		return nil, ErrNilAnalyzer
	}

	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return nil, ErrPathEmpty
	}

	absPath, err := filepath.Abs(filepath.Clean(cleanPath))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAnalysisFailed, err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrPathNotFound, absPath)
		}
		return nil, fmt.Errorf("%w: %v", ErrAnalysisFailed, err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("%w: %s", ErrNotDirectory, absPath)
	}

	discResult, err := a.discoverer.DiscoverPath(absPath)
	if err != nil {
		return nil, fmt.Errorf("%w: discovery failed: %v", ErrAnalysisFailed, err)
	}

	return a.Analyze(discResult)
}

func classifyModule(baseName, dir, relPath string) *Module {
	switch baseName {
	case "go.mod":
		return NewModule(ModuleGo, filepath.Base(dir), dir, relPath, "go", BuildUnknown)
	case "package.json":
		return NewModule(ModuleNpm, filepath.Base(dir), dir, relPath, "javascript", BuildNpm)
	case "Cargo.toml":
		return NewModule(ModuleCargo, filepath.Base(dir), dir, relPath, "rust", BuildCargo)
	case "pom.xml":
		return NewModule(ModuleMaven, filepath.Base(dir), dir, relPath, "java", BuildMaven)
	case "build.gradle", "build.gradle.kts":
		return NewModule(ModuleGradle, filepath.Base(dir), dir, relPath, "java", BuildGradle)
	case "requirements.txt", "pyproject.toml", "setup.py":
		return NewModule(ModulePython, filepath.Base(dir), dir, relPath, "python", BuildUnknown)
	case "composer.json":
		return NewModule(ModuleComposer, filepath.Base(dir), dir, relPath, "php", BuildUnknown)
	}
	return nil
}

func classifyBuildSystem(baseName, dir, relPath string) *BuildConfig {
	lower := strings.ToLower(baseName)
	switch lower {
	case "makefile", "gnumakefile":
		return NewBuildConfig(BuildMake, dir, relPath, dir)
	case "taskfile.yml", "taskfile.yaml":
		return NewBuildConfig(BuildTaskfile, dir, relPath, dir)
	case "cmakelists.txt":
		return NewBuildConfig(BuildCMake, dir, relPath, dir)
	case "pom.xml":
		return NewBuildConfig(BuildMaven, dir, relPath, dir)
	case "build.gradle", "build.gradle.kts":
		return NewBuildConfig(BuildGradle, dir, relPath, dir)
	case "package.json":
		return NewBuildConfig(BuildNpm, dir, relPath, dir)
	case "pnpm-lock.yaml":
		return NewBuildConfig(BuildPnpm, dir, relPath, dir)
	case "yarn.lock":
		return NewBuildConfig(BuildYarn, dir, relPath, dir)
	case "cargo.toml":
		return NewBuildConfig(BuildCargo, dir, relPath, dir)
	}
	return nil
}

func classifyConfigAsset(baseName, relPath, ext string) *ConfigAsset {
	isHidden := strings.HasPrefix(baseName, ".")

	switch ext {
	case ".yaml", ".yml":
		return NewConfigAsset(ConfigYAML, relPath, isHidden)
	case ".json":
		return NewConfigAsset(ConfigJSON, relPath, isHidden)
	case ".toml":
		return NewConfigAsset(ConfigTOML, relPath, isHidden)
	case ".ini":
		return NewConfigAsset(ConfigINI, relPath, isHidden)
	case ".properties":
		return NewConfigAsset(ConfigProperties, relPath, isHidden)
	case ".xml":
		return NewConfigAsset(ConfigXML, relPath, isHidden)
	}

	if strings.HasPrefix(baseName, ".env") || baseName == "env" || strings.HasSuffix(baseName, ".env") {
		return NewConfigAsset(ConfigENV, relPath, isHidden)
	}

	return nil
}

func classifyDocAsset(baseName, relPath, ext string) *DocAsset {
	nameWithoutExt := strings.TrimSuffix(baseName, ext)
	upperName := strings.ToUpper(nameWithoutExt)

	// Check documentation directory
	isDocDir := strings.HasPrefix(relPath, "docs/") ||
		strings.HasPrefix(relPath, "doc/") ||
		strings.HasPrefix(relPath, "documentation/") ||
		strings.Contains(relPath, "/docs/") ||
		strings.Contains(relPath, "/doc/")

	// ADR detection
	if strings.Contains(relPath, "/adr/") || strings.HasPrefix(relPath, "adr/") || strings.HasPrefix(upperName, "ADR") {
		return NewDocAsset(DocADR, relPath, "architecture")
	}

	switch upperName {
	case "README":
		return NewDocAsset(DocReadme, relPath, "general")
	case "CONTRIBUTING":
		return NewDocAsset(DocContributing, relPath, "process")
	case "SECURITY":
		return NewDocAsset(DocSecurity, relPath, "governance")
	case "LICENSE", "COPYING", "UNLICENSE":
		return NewDocAsset(DocLicense, relPath, "legal")
	case "CHANGELOG", "HISTORY", "CHANGES":
		return NewDocAsset(DocChangelog, relPath, "release")
	case "ROADMAP":
		return NewDocAsset(DocRoadmap, relPath, "planning")
	}

	if isDocDir && (ext == ".md" || ext == ".txt" || ext == ".rst" || ext == ".adoc" || ext == ".html" || ext == ".pdf") {
		return NewDocAsset(DocGeneral, relPath, "documentation")
	}

	return nil
}

func isVendorPath(relPath string) bool {
	lower := strings.ToLower(filepath.ToSlash(relPath))
	return strings.HasPrefix(lower, "vendor/") ||
		strings.Contains(lower, "/vendor/") ||
		strings.HasPrefix(lower, "node_modules/") ||
		strings.Contains(lower, "/node_modules/") ||
		strings.HasPrefix(lower, "bower_components/") ||
		strings.Contains(lower, "/bower_components/") ||
		strings.HasPrefix(lower, ".venv/") ||
		strings.Contains(lower, "/.venv/") ||
		strings.HasPrefix(lower, "venv/") ||
		strings.Contains(lower, "/venv/")
}

func extractVendorRoot(relPath string) string {
	slashPath := filepath.ToSlash(relPath)
	segments := strings.Split(slashPath, "/")
	for i, seg := range segments {
		lower := strings.ToLower(seg)
		if lower == "vendor" || lower == "node_modules" || lower == "bower_components" || lower == ".venv" || lower == "venv" {
			return strings.Join(segments[:i+1], "/")
		}
	}
	return ""
}

func classifyVendorEcosystem(vendorPath string) string {
	base := strings.ToLower(filepath.Base(vendorPath))
	switch base {
	case "vendor":
		return "go/php"
	case "node_modules", "bower_components":
		return "npm"
	case ".venv", "venv":
		return "python"
	}
	return "general"
}
