package graph

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/repository/dependency"
	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
	"github.com/unhield/limoxel/internal/capabilities/repository/indexing"
	"github.com/unhield/limoxel/internal/capabilities/repository/language"
	"github.com/unhield/limoxel/internal/capabilities/repository/metadata"
	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
	"github.com/unhield/limoxel/internal/capabilities/repository/xref"
	"github.com/unhield/limoxel/internal/repository"
)

// Engine orchestrates deterministic knowledge graph construction from upstream repository capabilities.
type Engine struct {
	discoverer    *discovery.Discoverer
	metaCollector *metadata.Collector
	langAnalyzer  *language.Analyzer
	depAnalyzer   *dependency.Analyzer
	indexer       *indexing.Indexer
	symEngine     *symbol.Engine
	xrefEngine    *xref.Engine
	numWorkers    int
}

// New constructs a new Knowledge Graph Engine.
func New(
	discoverer *discovery.Discoverer,
	metaCollector *metadata.Collector,
	langAnalyzer *language.Analyzer,
	depAnalyzer *dependency.Analyzer,
	indexer *indexing.Indexer,
	symEngine *symbol.Engine,
	xrefEngine *xref.Engine,
) (*Engine, error) {
	if discoverer == nil {
		return nil, ErrNilDiscoverer
	}
	return &Engine{
		discoverer:    discoverer,
		metaCollector: metaCollector,
		langAnalyzer:  langAnalyzer,
		depAnalyzer:   depAnalyzer,
		indexer:       indexer,
		symEngine:     symEngine,
		xrefEngine:    xrefEngine,
		numWorkers:    4,
	}, nil
}

// NewWithWorkers constructs an Engine with a specific worker count.
func NewWithWorkers(
	discoverer *discovery.Discoverer,
	metaCollector *metadata.Collector,
	langAnalyzer *language.Analyzer,
	depAnalyzer *dependency.Analyzer,
	indexer *indexing.Indexer,
	symEngine *symbol.Engine,
	xrefEngine *xref.Engine,
	numWorkers int,
) (*Engine, error) {
	if discoverer == nil {
		return nil, ErrNilDiscoverer
	}
	if numWorkers <= 0 {
		numWorkers = 4
	}
	return &Engine{
		discoverer:    discoverer,
		metaCollector: metaCollector,
		langAnalyzer:  langAnalyzer,
		depAnalyzer:   depAnalyzer,
		indexer:       indexer,
		symEngine:     symEngine,
		xrefEngine:    xrefEngine,
		numWorkers:    numWorkers,
	}, nil
}

// NumWorkers returns the configured concurrency worker count.
func (e *Engine) NumWorkers() int {
	if e == nil {
		return 0
	}
	return e.numWorkers
}

// BuildGraph constructs the complete KnowledgeGraph from established upstream models.
func (e *Engine) BuildGraph(
	discResult *discovery.Result,
	profile *metadata.Profile,
	structModel *language.StructureModel,
	depModel *dependency.DependencyModel,
	indexModel *indexing.IndexModel,
	symModel *symbol.SymbolModel,
	xrefModel *xref.XRefModel,
) (*KnowledgeGraph, error) {
	if e == nil {
		return nil, ErrNilEngine
	}
	if discResult == nil {
		return nil, ErrNilDiscoveryResult
	}

	repoRoot := ""
	repoName := "repository"
	if discResult.Repository() != nil {
		repoRoot = discResult.Repository().Root()
		if discResult.Repository().Name() != "" {
			repoName = discResult.Repository().Name()
		}
	}

	var nodes []*Node
	var relationships []*Relationship
	nodeMap := make(map[string]*Node)

	addNode := func(n *Node) {
		if n != nil && n.ID() != "" {
			if _, exists := nodeMap[n.ID()]; !exists {
				nodeMap[n.ID()] = n
				nodes = append(nodes, n)
			}
		}
	}

	addRel := func(r *Relationship) {
		if r != nil && r.ID() != "" {
			relationships = append(relationships, r)
		}
	}

	// 1. Repository Node
	repoNodeID := "repo:" + repoName
	repoMeta := make(map[string]string)
	if profile != nil {
		if profile.DefaultBranch() != "" {
			repoMeta["default_branch"] = profile.DefaultBranch()
		}
	}
	repoNode := NewNode(repoNodeID, NodeRepository, repoName, "", "", "", repoMeta)
	addNode(repoNode)

	// 2. Module Nodes from language.StructureModel
	modNodeMap := make(map[string]string) // moduleName -> nodeID
	if structModel != nil && structModel.ModuleGraph() != nil {
		for _, mod := range structModel.ModuleGraph().Modules() {
			modID := "mod:" + mod.Name()
			modNodeMap[mod.Name()] = modID
			modMeta := make(map[string]string)
			modMeta["build_system"] = string(mod.BuildSystem())
			modMeta["path"] = mod.Path()
			modNode := NewNode(modID, NodeModule, mod.Name(), mod.Path(), mod.Name(), "", modMeta)
			addNode(modNode)

			// Repo -> Module (Contains)
			addRel(NewRelationship(
				RelContains,
				repoNodeID,
				modID,
				[]ProvenanceSource{ProvLanguage},
				nil,
			))
		}
	}

	// Default module if none detected
	defaultModID := repoNodeID
	if len(modNodeMap) == 1 {
		for _, mID := range modNodeMap {
			defaultModID = mID
		}
	}

	// 3. Package Nodes from language.StructureModel or indexing.IndexModel
	pkgNodeMap := make(map[string]string) // pkgPath -> nodeID
	if structModel != nil {
		for _, pkg := range structModel.Packages() {
			pID := "pkg:" + pkg.Path()
			pkgNodeMap[pkg.Path()] = pID
			pkgMeta := map[string]string{"package_name": pkg.Name()}
			pkgNode := NewNode(pID, NodePackage, pkg.Name(), pkg.Path(), "", pkg.Name(), pkgMeta)
			addNode(pkgNode)

			// Module / Repo -> Package (Contains)
			addRel(NewRelationship(
				RelContains,
				defaultModID,
				pID,
				[]ProvenanceSource{ProvLanguage},
				nil,
			))
		}
	} else if indexModel != nil {
		for _, pkg := range indexModel.Packages() {
			pID := "pkg:" + pkg.Path()
			pkgNodeMap[pkg.Path()] = pID
			pkgMeta := map[string]string{"package_name": pkg.Name()}
			pkgNode := NewNode(pID, NodePackage, pkg.Name(), pkg.Path(), "", pkg.Name(), pkgMeta)
			addNode(pkgNode)

			// Module / Repo -> Package (Contains)
			addRel(NewRelationship(
				RelContains,
				defaultModID,
				pID,
				[]ProvenanceSource{ProvIndexing},
				nil,
			))
		}
	}

	// 4. File, Doc, Config Nodes
	fileNodeMap := make(map[string]string) // relPath -> nodeID
	for _, f := range discResult.Files() {
		relPath := f.RelPath()
		cleanRel := filepath.ToSlash(filepath.Clean(relPath))
		ext := strings.ToLower(filepath.Ext(cleanRel))
		baseName := filepath.Base(cleanRel)

		var fNode *Node
		var fID string
		var prov ProvenanceSource = ProvDiscovery

		if ext == ".md" || strings.HasPrefix(strings.ToUpper(baseName), "README") {
			fID = "doc:" + cleanRel
			fNode = NewNode(fID, NodeDoc, baseName, cleanRel, "", "", map[string]string{"type": "markdown"})
			addNode(fNode)
			fileNodeMap[cleanRel] = fID

			// Documents -> Repo / Package
			addRel(NewRelationship(
				RelDocuments,
				fID,
				repoNodeID,
				[]ProvenanceSource{ProvDiscovery},
				nil,
			))
		} else if ext == ".json" || ext == ".yaml" || ext == ".yml" || ext == ".toml" || ext == ".xml" {
			fID = "cfg:" + cleanRel
			fNode = NewNode(fID, NodeConfig, baseName, cleanRel, "", "", map[string]string{"format": ext})
			addNode(fNode)
			fileNodeMap[cleanRel] = fID

			// Configures -> Repo
			addRel(NewRelationship(
				RelConfigures,
				fID,
				repoNodeID,
				[]ProvenanceSource{ProvDiscovery},
				nil,
			))
		} else {
			fID = "file:" + cleanRel
			fNode = NewNode(fID, NodeFile, baseName, cleanRel, "", "", map[string]string{"extension": ext})
			addNode(fNode)
			fileNodeMap[cleanRel] = fID

			// Package -> File (Contains)
			dir := filepath.ToSlash(filepath.Dir(cleanRel))
			if dir == "." {
				dir = "."
			}
			parentPkgID, hasPkg := pkgNodeMap[dir]
			if !hasPkg {
				parentPkgID = repoNodeID
			}
			addRel(NewRelationship(
				RelContains,
				parentPkgID,
				fID,
				[]ProvenanceSource{prov},
				nil,
			))
		}
	}

	// 5. Symbol Nodes
	symNodeMap := make(map[string]string) // symID -> nodeID
	if symModel != nil && symModel.Symbols() != nil {
		for _, s := range symModel.Symbols().AllSymbols() {
			sID := "sym:" + s.ID()
			symNodeMap[s.ID()] = sID
			sMeta := map[string]string{
				"kind":         string(s.Kind()),
				"exported":     fmt.Sprintf("%t", s.IsExported()),
				"package_name": s.PackageName(),
			}
			sNode := NewNode(sID, NodeSymbol, s.Name(), s.FilePath(), "", s.PackageName(), sMeta)
			addNode(sNode)

			// File -> Symbol (Contains)
			parentFileID, hasFile := fileNodeMap[s.FilePath()]
			if !hasFile {
				parentFileID = repoNodeID
			}
			addRel(NewRelationship(
				RelContains,
				parentFileID,
				sID,
				[]ProvenanceSource{ProvSymbol},
				nil,
			))
		}

		// 6. Implements Relationships from Symbol Relationships
		if symModel.Relationships() != nil {
			for _, rel := range symModel.Relationships().AllRelationships() {
				if rel.Kind() == symbol.RelInterfaceImplementation {
					srcID := "sym:" + rel.SourceID()
					tgtID := "sym:" + rel.TargetID()
					if _, ok1 := nodeMap[srcID]; ok1 {
						if _, ok2 := nodeMap[tgtID]; ok2 {
							addRel(NewRelationship(
								RelImplements,
								srcID,
								tgtID,
								[]ProvenanceSource{ProvSymbol},
								map[string]string{"kind": string(rel.Kind())},
							))
						}
					}
				}
			}
		}
	}

	// 7. Imports & DependsOn from DependencyModel
	if depModel != nil {
		if depModel.Graph() != nil {
			for _, edge := range depModel.Graph().Edges() {
				srcPkgID, ok1 := pkgNodeMap[edge.SourceID()]
				tgtPkgID, ok2 := pkgNodeMap[edge.TargetID()]
				if ok1 && ok2 && srcPkgID != tgtPkgID {
					addRel(NewRelationship(
						RelImports,
						srcPkgID,
						tgtPkgID,
						[]ProvenanceSource{ProvDependency},
						nil,
					))
				}
			}
		}

		if depModel.Inventory() != nil {
			for _, dep := range depModel.Inventory().AllDependencies() {
				depNodeID := "dep:" + dep.Name()
				depNode := NewNode(depNodeID, NodePackage, dep.Name(), "", "", dep.Name(), map[string]string{"version": dep.Version().String(), "direct": fmt.Sprintf("%t", dep.IsDirect())})
				addNode(depNode)

				addRel(NewRelationship(
					RelDependsOn,
					repoNodeID,
					depNodeID,
					[]ProvenanceSource{ProvDependency},
					map[string]string{"direct": fmt.Sprintf("%t", dep.IsDirect())},
				))
			}
		}
	}

	// 8. Calls & References from XRefModel
	if xrefModel != nil {
		if xrefModel.CallGraph() != nil {
			for _, edge := range xrefModel.CallGraph().AllEdges() {
				srcID := "sym:" + edge.CallerID()
				tgtID := "sym:" + edge.CalleeID()
				if _, ok1 := nodeMap[srcID]; ok1 {
					if _, ok2 := nodeMap[tgtID]; ok2 {
						addRel(NewRelationship(
							RelCalls,
							srcID,
							tgtID,
							[]ProvenanceSource{ProvXRef},
							map[string]string{"call_kind": string(edge.Kind())},
						))
					}
				}
			}
		}

		if xrefModel.References() != nil {
			for _, ref := range xrefModel.References().AllReferences() {
				srcID := "sym:" + ref.SourceSymbolID()
				tgtID := "sym:" + ref.TargetSymbolID()
				if _, ok1 := nodeMap[srcID]; ok1 {
					if _, ok2 := nodeMap[tgtID]; ok2 {
						addRel(NewRelationship(
							RelReferences,
							srcID,
							tgtID,
							[]ProvenanceSource{ProvXRef},
							map[string]string{"ref_kind": string(ref.Kind()), "evidence": ref.Evidence()},
						))
					}
				}
			}
		}
	}

	// Sort nodes and relationships deterministically before creating KnowledgeGraph
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ID() < nodes[j].ID()
	})
	sort.Slice(relationships, func(i, j int) bool {
		return relationships[i].ID() < relationships[j].ID()
	})

	return NewKnowledgeGraph(repoRoot, nodes, relationships), nil
}

// BuildGraphFromRepository builds the knowledge graph for a domain repository.
func (e *Engine) BuildGraphFromRepository(repo *repository.Repository) (*KnowledgeGraph, error) {
	if e == nil {
		return nil, ErrNilEngine
	}
	if repo == nil {
		return nil, ErrNilRepository
	}

	discResult, err := e.discoverer.Discover(repo)
	if err != nil {
		return nil, err
	}

	var profile *metadata.Profile
	if e.metaCollector != nil {
		profile, _ = e.metaCollector.Collect(discResult)
	}

	var structModel *language.StructureModel
	if e.langAnalyzer != nil {
		structModel, _ = e.langAnalyzer.Analyze(discResult)
	}

	var depModel *dependency.DependencyModel
	if e.depAnalyzer != nil {
		depModel, _ = e.depAnalyzer.Analyze(discResult)
	}

	var indexModel *indexing.IndexModel
	if e.indexer != nil {
		indexModel, _ = e.indexer.Index(discResult)
	}

	var symModel *symbol.SymbolModel
	if e.symEngine != nil {
		symModel, _ = e.symEngine.Parse(discResult)
	}

	var xrefModel *xref.XRefModel
	if e.xrefEngine != nil && symModel != nil {
		xrefModel, _ = e.xrefEngine.Analyze(discResult, symModel, depModel)
	}

	return e.BuildGraph(discResult, profile, structModel, depModel, indexModel, symModel, xrefModel)
}

// BuildGraphFromPath builds the knowledge graph for a repository path on disk.
func (e *Engine) BuildGraphFromPath(path string) (*KnowledgeGraph, error) {
	if e == nil {
		return nil, ErrNilEngine
	}
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return nil, ErrPathEmpty
	}

	discResult, err := e.discoverer.DiscoverPath(cleanPath)
	if err != nil {
		return nil, err
	}

	var profile *metadata.Profile
	if e.metaCollector != nil {
		profile, _ = e.metaCollector.Collect(discResult)
	}

	var structModel *language.StructureModel
	if e.langAnalyzer != nil {
		structModel, _ = e.langAnalyzer.Analyze(discResult)
	}

	var depModel *dependency.DependencyModel
	if e.depAnalyzer != nil {
		depModel, _ = e.depAnalyzer.Analyze(discResult)
	}

	var indexModel *indexing.IndexModel
	if e.indexer != nil {
		indexModel, _ = e.indexer.Index(discResult)
	}

	var symModel *symbol.SymbolModel
	if e.symEngine != nil {
		symModel, _ = e.symEngine.Parse(discResult)
	}

	var xrefModel *xref.XRefModel
	if e.xrefEngine != nil && symModel != nil {
		xrefModel, _ = e.xrefEngine.Analyze(discResult, symModel, depModel)
	}

	return e.BuildGraph(discResult, profile, structModel, depModel, indexModel, symModel, xrefModel)
}
