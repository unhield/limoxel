package dependency

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
	"github.com/unhield/limoxel/internal/capabilities/repository/language"
	"github.com/unhield/limoxel/internal/repository"
)

// Analyzer coordinates deterministic repository dependency analysis.
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

// Analyze compiles a complete DependencyModel from an existing discovery result.
func (a *Analyzer) Analyze(discResult *discovery.Result) (*DependencyModel, error) {
	if a == nil {
		return nil, ErrNilAnalyzer
	}
	if discResult == nil {
		return nil, ErrNilDiscoveryResult
	}

	root := discResult.Root()
	files := discResult.Files()
	diagnostics := discResult.Diagnostics()

	var (
		allDependencies []*Dependency
		internalImports []*InternalImport
		internalPkgs    = make(map[string]struct{})
		licenses        []*LicenseInfo
	)

	// 1. Identify internal package locations from discovered source files
	for _, f := range files {
		relPath := f.RelPath()
		dir := filepath.ToSlash(filepath.Dir(relPath))
		if dir == "" {
			dir = "."
		}
		if f.Language() != nil && f.Language().ID() != "" {
			internalPkgs[dir] = struct{}{}
		}
	}

	// 2. Discover manifest dependencies and source-level internal imports
	for _, f := range files {
		relPath := f.RelPath()
		absPath := f.AbsPath()
		baseName := filepath.Base(relPath)
		dir := filepath.ToSlash(filepath.Dir(relPath))
		if dir == "" {
			dir = "."
		}

		// A. Parse manifests
		switch baseName {
		case "go.mod":
			deps := parseGoMod(absPath, relPath, dir, &diagnostics)
			allDependencies = append(allDependencies, deps...)
		case "package.json":
			deps := parsePackageJSON(absPath, relPath, dir, &diagnostics)
			allDependencies = append(allDependencies, deps...)
		case "Cargo.toml":
			deps := parseCargoToml(absPath, relPath, dir, &diagnostics)
			allDependencies = append(allDependencies, deps...)
		case "pom.xml":
			deps := parsePomXML(absPath, relPath, dir, &diagnostics)
			allDependencies = append(allDependencies, deps...)
		case "build.gradle", "build.gradle.kts":
			deps := parseGradle(absPath, relPath, dir, &diagnostics)
			allDependencies = append(allDependencies, deps...)
		case "requirements.txt":
			deps := parseRequirementsTxt(absPath, relPath, dir, &diagnostics)
			allDependencies = append(allDependencies, deps...)
		case "composer.json":
			deps := parseComposerJSON(absPath, relPath, dir, &diagnostics)
			allDependencies = append(allDependencies, deps...)
		}

		// B. Parse source-level internal imports
		if f.Language() != nil && f.Language().ID() != "" {
			langID := f.Language().ID()
			if langID == "go" || langID == "javascript" || langID == "typescript" || langID == "python" {
				imps := parseSourceImports(absPath, relPath, langID, dir, &diagnostics)
				internalImports = append(internalImports, imps...)
			}
		}
	}

	// 3. Process internal imports into internal dependencies
	for _, imp := range internalImports {
		targetClean := filepath.ToSlash(filepath.Clean(imp.TargetPackage()))

		// Check if target is an internal package
		isInternal := false
		for pkgPath := range internalPkgs {
			if pkgPath == "." {
				continue
			}
			if targetClean == pkgPath || strings.HasSuffix(targetClean, "/"+pkgPath) || strings.Contains(targetClean, "/"+pkgPath+"/") {
				isInternal = true
				targetClean = pkgPath
				break
			}
		}

		// Also check relative imports e.g. "./pkgB" or "../pkgB"
		if !isInternal && (strings.HasPrefix(targetClean, "./") || strings.HasPrefix(targetClean, "../")) {
			joined := filepath.ToSlash(filepath.Clean(filepath.Join(imp.SourcePackage(), targetClean)))
			if _, ok := internalPkgs[joined]; ok {
				isInternal = true
				targetClean = joined
			}
		}

		if isInternal && imp.SourcePackage() != targetClean {
			allDependencies = append(allDependencies, NewDependency(
				targetClean,
				"",
				EcosystemUnknown,
				DependencyInternal,
				true,
				false,
				true,
				false,
				imp.SourceFile(),
				imp.SourcePackage(),
				NewLicenseInfo(LicenseUnavailable, "", "", false),
				NewHealthInfo(HealthActive, false, false, false, true, 1.0, nil),
			))
		}
	}

	// 4. Deduplicate dependencies
	var dedupedDeps []*Dependency
	seenDeps := make(map[string]struct{})
	for _, d := range allDependencies {
		key := fmt.Sprintf("%s:%s:%s:%s", d.Name(), d.ModulePath(), d.DeclaredVersion(), d.SourceManifest())
		if _, seen := seenDeps[key]; !seen {
			seenDeps[key] = struct{}{}
			dedupedDeps = append(dedupedDeps, d)
		}
	}
	allDependencies = dedupedDeps

	// 5. Collect unique licenses
	seenLicenses := make(map[string]struct{})
	for _, d := range allDependencies {
		if d.license != nil && d.license.IsAvailable() {
			key := string(d.license.Type()) + ":" + d.license.Source()
			if _, exists := seenLicenses[key]; !exists {
				seenLicenses[key] = struct{}{}
				licenses = append(licenses, d.license)
			}
		}
	}

	// 6. Build Graph Nodes & Edges
	nodeMap := make(map[string]*GraphNode)
	var edges []*GraphEdge
	seenEdges := make(map[string]struct{})

	// Ensure all internal packages have graph nodes
	for pkg := range internalPkgs {
		nodeMap[pkg] = NewGraphNode(pkg, pkg, true, EcosystemUnknown)
	}

	// Create nodes and edges for all dependencies
	for _, d := range allDependencies {
		sourceID := d.ModulePath()
		if sourceID == "" {
			sourceID = "."
		}
		targetID := d.Name()

		if _, exists := nodeMap[sourceID]; !exists {
			nodeMap[sourceID] = NewGraphNode(sourceID, sourceID, d.IsInternal(), d.Ecosystem())
		}
		if _, exists := nodeMap[targetID]; !exists {
			nodeMap[targetID] = NewGraphNode(targetID, targetID, d.IsInternal(), d.Ecosystem())
		}

		edgeKey := sourceID + "->" + targetID
		if _, seen := seenEdges[edgeKey]; !seen && sourceID != targetID {
			seenEdges[edgeKey] = struct{}{}
			edges = append(edges, NewGraphEdge(sourceID, targetID, d.Type()))
		}
	}

	var nodes []*GraphNode
	for _, n := range nodeMap {
		nodes = append(nodes, n)
	}

	graph := NewDependencyGraph(nodes, edges)
	cycles := graph.DetectCycles()
	orphans := graph.DetectOrphans()
	maxDepth := graph.CalculateMaxDepth()

	inventory := NewDependencyInventory(allDependencies)
	licenseInventory := NewLicenseInventory(licenses)

	return NewDependencyModel(
		root,
		inventory,
		graph,
		licenseInventory,
		cycles,
		orphans,
		maxDepth,
		diagnostics,
	), nil
}

// AnalyzeRepository executes discovery on an existing Repository instance and analyzes its dependencies.
func (a *Analyzer) AnalyzeRepository(repo *repository.Repository) (*DependencyModel, error) {
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

// AnalyzePath discovers and analyzes dependencies for a target directory path.
func (a *Analyzer) AnalyzePath(path string) (*DependencyModel, error) {
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

// AnalyzeStructure performs dependency analysis combining existing StructureModel and Discovery Result.
func (a *Analyzer) AnalyzeStructure(structModel *language.StructureModel, discResult *discovery.Result) (*DependencyModel, error) {
	if a == nil {
		return nil, ErrNilAnalyzer
	}
	if discResult == nil {
		return nil, ErrNilDiscoveryResult
	}
	return a.Analyze(discResult)
}
