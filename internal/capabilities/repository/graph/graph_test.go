package graph_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/repository/dependency"
	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
	"github.com/unhield/limoxel/internal/capabilities/repository/graph"
	"github.com/unhield/limoxel/internal/capabilities/repository/indexing"
	"github.com/unhield/limoxel/internal/capabilities/repository/language"
	"github.com/unhield/limoxel/internal/capabilities/repository/metadata"
	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
	"github.com/unhield/limoxel/internal/capabilities/repository/xref"
	langreg "github.com/unhield/limoxel/internal/language"
)

func setupTestLanguageRegistry(t *testing.T) *langreg.Registry {
	t.Helper()
	reg := langreg.NewRegistry()

	goLang, _ := langreg.New("go", "Go", []string{".go"}, nil, []string{"golang"})
	mdLang, _ := langreg.New("markdown", "Markdown", []string{".md"}, nil, []string{"md"})
	jsonLang, _ := langreg.New("json", "JSON", []string{".json"}, nil, []string{"json"})
	yamlLang, _ := langreg.New("yaml", "YAML", []string{".yaml", ".yml"}, nil, []string{"yaml"})

	_ = reg.Register(goLang)
	_ = reg.Register(mdLang)
	_ = reg.Register(jsonLang)
	_ = reg.Register(yamlLang)

	return reg
}

func setupTestPipeline(t *testing.T) (
	*discovery.Discoverer,
	*metadata.Collector,
	*language.Analyzer,
	*dependency.Analyzer,
	*indexing.Indexer,
	*symbol.Engine,
	*xref.Engine,
	*graph.Engine,
) {
	t.Helper()
	reg := setupTestLanguageRegistry(t)
	disc, err := discovery.New(reg)
	if err != nil {
		t.Fatalf("failed creating discoverer: %v", err)
	}

	metaCollector, _ := metadata.New(disc)
	langAnalyzer, _ := language.New(disc)
	depAnalyzer, _ := dependency.New(disc)
	indexer, _ := indexing.New(disc)
	symEngine, _ := symbol.New(disc)
	xrefEngine, _ := xref.New(disc, symEngine, depAnalyzer)

	graphEngine, err := graph.New(
		disc,
		metaCollector,
		langAnalyzer,
		depAnalyzer,
		indexer,
		symEngine,
		xrefEngine,
	)
	if err != nil {
		t.Fatalf("graph.New failed: %v", err)
	}

	return disc, metaCollector, langAnalyzer, depAnalyzer, indexer, symEngine, xrefEngine, graphEngine
}

// 1. Constructor & Nil Handling
func TestGraph_ConstructorAndNilGuards(t *testing.T) {
	_, metaCollector, langAnalyzer, depAnalyzer, indexer, symEngine, xrefEngine, eng := setupTestPipeline(t)

	t.Run("nil discoverer returns ErrNilDiscoverer", func(t *testing.T) {
		g, err := graph.New(nil, metaCollector, langAnalyzer, depAnalyzer, indexer, symEngine, xrefEngine)
		if err != graph.ErrNilDiscoverer || g != nil {
			t.Errorf("expected ErrNilDiscoverer, got %v", err)
		}
		g2, err2 := graph.NewWithWorkers(nil, metaCollector, langAnalyzer, depAnalyzer, indexer, symEngine, xrefEngine, 2)
		if err2 != graph.ErrNilDiscoverer || g2 != nil {
			t.Errorf("expected ErrNilDiscoverer, got %v", err2)
		}
	})

	t.Run("nil engine receiver methods return safe errors", func(t *testing.T) {
		var nilEng *graph.Engine
		if _, err := nilEng.BuildGraph(nil, nil, nil, nil, nil, nil, nil); err != graph.ErrNilEngine {
			t.Errorf("expected ErrNilEngine, got %v", err)
		}
		if _, err := nilEng.BuildGraphFromPath("some/path"); err != graph.ErrNilEngine {
			t.Errorf("expected ErrNilEngine, got %v", err)
		}
		if _, err := nilEng.BuildGraphFromRepository(nil); err != graph.ErrNilEngine {
			t.Errorf("expected ErrNilEngine, got %v", err)
		}
	})

	t.Run("empty path returns ErrPathEmpty", func(t *testing.T) {
		if _, err := eng.BuildGraphFromPath("   "); err != graph.ErrPathEmpty {
			t.Errorf("expected ErrPathEmpty, got %v", err)
		}
	})

	t.Run("nil repository returns ErrNilRepository", func(t *testing.T) {
		if _, err := eng.BuildGraphFromRepository(nil); err != graph.ErrNilRepository {
			t.Errorf("expected ErrNilRepository, got %v", err)
		}
	})

	t.Run("nil discovery result returns ErrNilDiscoveryResult", func(t *testing.T) {
		if _, err := eng.BuildGraph(nil, nil, nil, nil, nil, nil, nil); err != graph.ErrNilDiscoveryResult {
			t.Errorf("expected ErrNilDiscoveryResult, got %v", err)
		}
	})
}

// 2. Node Identity & Entity Types
func TestGraph_NodeTypesAndIdentities(t *testing.T) {
	tempDir := t.TempDir()
	_, _, _, _, _, _, _, eng := setupTestPipeline(t)

	repoRoot := filepath.Join(tempDir, "nodes_repo")
	pkgPath := filepath.Join(repoRoot, "pkg", "service")
	_ = os.MkdirAll(pkgPath, 0755)

	_ = os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("# Service\nDocs\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "config.json"), []byte("{\"port\": 9000}\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module mymod\n\ngo 1.26\n"), 0644)

	src := `package service

type Handler interface {
	Serve() string
}

type HTTPHandler struct {
	Port int
}

func (h *HTTPHandler) Serve() string {
	return "ok"
}
`
	_ = os.WriteFile(filepath.Join(pkgPath, "service.go"), []byte(src), 0644)

	kg, err := eng.BuildGraphFromPath(repoRoot)
	if err != nil {
		t.Fatalf("BuildGraphFromPath failed: %v", err)
	}

	// 1. Repository Node
	repos := kg.NodesByType(graph.NodeRepository)
	if len(repos) != 1 {
		t.Fatalf("expected 1 repository node, got %d", len(repos))
	}
	if !strings.HasPrefix(repos[0].ID(), "repo:") {
		t.Errorf("expected repo: prefix for ID, got %s", repos[0].ID())
	}

	// 2. Module Node
	mods := kg.NodesByType(graph.NodeModule)
	if len(mods) < 1 {
		t.Errorf("expected at least 1 module node, got %d", len(mods))
	}

	// 3. Package Node
	pkgs := kg.NodesByType(graph.NodePackage)
	if len(pkgs) < 1 {
		t.Errorf("expected package nodes, got %d", len(pkgs))
	}

	// 4. File Node
	files := kg.NodesByType(graph.NodeFile)
	if len(files) < 1 {
		t.Errorf("expected file nodes, got %d", len(files))
	}

	// 5. Symbol Node
	syms := kg.NodesByType(graph.NodeSymbol)
	if len(syms) < 3 {
		t.Errorf("expected at least 3 symbol nodes (Handler, HTTPHandler, Serve), got %d", len(syms))
	}

	// 6. Documentation Node
	docs := kg.NodesByType(graph.NodeDoc)
	if len(docs) != 1 || docs[0].Type() != graph.NodeDoc {
		t.Errorf("expected 1 documentation node for README.md, got %d", len(docs))
	}

	// 7. Configuration Node
	cfgs := kg.NodesByType(graph.NodeConfig)
	if len(cfgs) != 1 || cfgs[0].Type() != graph.NodeConfig {
		t.Errorf("expected 1 config node for config.json, got %d", len(cfgs))
	}
}

// 3. Relationships Verification
func TestGraph_RelationshipTypes(t *testing.T) {
	tempDir := t.TempDir()
	_, _, _, _, _, _, _, eng := setupTestPipeline(t)

	repoRoot := filepath.Join(tempDir, "rel_repo")
	pkgA := filepath.Join(repoRoot, "pkg", "a")
	pkgB := filepath.Join(repoRoot, "pkg", "b")
	_ = os.MkdirAll(pkgA, 0755)
	_ = os.MkdirAll(pkgB, 0755)

	_ = os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("# Overview\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "settings.yaml"), []byte("env: dev\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module example\n\ngo 1.26\n"), 0644)

	srcA := `package a

type Greeter interface {
	Greet(name string) string
}

type HumanGreeter struct{}

func (h *HumanGreeter) Greet(name string) string {
	return "Hello " + name
}
`
	_ = os.WriteFile(filepath.Join(pkgA, "greeter.go"), []byte(srcA), 0644)

	srcB := `package b

import "pkg/a"

func Execute() {
	var g a.Greeter = &a.HumanGreeter{}
	_ = g.Greet("World")
}
`
	_ = os.WriteFile(filepath.Join(pkgB, "human.go"), []byte(srcB), 0644)

	kg, err := eng.BuildGraphFromPath(repoRoot)
	if err != nil {
		t.Fatalf("BuildGraphFromPath failed: %v", err)
	}

	typesFound := make(map[graph.RelationshipType]int)
	for _, r := range kg.AllRelationships() {
		typesFound[r.Type()]++
	}

	requiredTypes := []graph.RelationshipType{
		graph.RelContains,
		graph.RelImplements,
		graph.RelCalls,
		graph.RelReferences,
		graph.RelDocuments,
		graph.RelConfigures,
	}

	for _, rt := range requiredTypes {
		if typesFound[rt] == 0 {
			t.Errorf("missing expected relationship type %s in graph (found: %+v)", rt, typesFound)
		}
	}
}

// 4. Query Engine (Lookup, Neighbors, Traversal, Reverse, Filter, Cycles)
func TestGraph_ComprehensiveQueryEngine(t *testing.T) {
	tempDir := t.TempDir()
	_, _, _, _, _, _, _, eng := setupTestPipeline(t)

	repoRoot := filepath.Join(tempDir, "query_suite_repo")
	_ = os.MkdirAll(repoRoot, 0755)

	src := `package main

func NodeA() { NodeB() }
func NodeB() { NodeC() }
func NodeC() { NodeA() } // Cycle NodeA -> NodeB -> NodeC -> NodeA
func NodeD() { NodeB() }

func main() {
	NodeA()
}
`
	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte(src), 0644)

	kg, err := eng.BuildGraphFromPath(repoRoot)
	if err != nil {
		t.Fatalf("BuildGraphFromPath failed: %v", err)
	}

	qe := kg.Query()

	t.Run("LookupNode deterministic and safe", func(t *testing.T) {
		n, err := qe.LookupNode("sym:NodeA")
		if err != nil || n == nil {
			t.Fatalf("LookupNode failed: %v", err)
		}
		if n.Name() != "NodeA" {
			t.Errorf("expected name NodeA, got %s", n.Name())
		}
		_, errNotFound := qe.LookupNode("sym:DoesNotExist")
		if errNotFound != graph.ErrNodeNotFound {
			t.Errorf("expected ErrNodeNotFound, got %v", errNotFound)
		}
	})

	t.Run("FindNodesByName handles ambiguous and exact names", func(t *testing.T) {
		nodes := qe.FindNodesByName("NodeA")
		if len(nodes) != 1 {
			t.Errorf("expected 1 node for NodeA, got %d", len(nodes))
		}
		nodesMissing := qe.FindNodesByName("Unknown")
		if len(nodesMissing) != 0 {
			t.Errorf("expected 0 nodes for Unknown, got %d", len(nodesMissing))
		}
	})

	t.Run("Neighbors in outbound, inbound, both directions", func(t *testing.T) {
		outN := qe.Neighbors("sym:NodeA", graph.DirOutbound, graph.RelCalls)
		if len(outN) != 1 || outN[0].ID() != "sym:NodeB" {
			t.Errorf("expected NodeB as outbound neighbor of NodeA, got %+v", outN)
		}

		inN := qe.Neighbors("sym:NodeB", graph.DirInbound, graph.RelCalls)
		// NodeA and NodeD both call NodeB
		if len(inN) < 2 {
			t.Errorf("expected at least 2 inbound callers to NodeB, got %d", len(inN))
		}

		bothN := qe.Neighbors("sym:NodeB", graph.DirBoth, graph.RelCalls)
		if len(bothN) < 3 {
			t.Errorf("expected at least 3 bidirectional neighbors for NodeB, got %d", len(bothN))
		}
	})

	t.Run("TraversePath with cycle handling and bounded depth", func(t *testing.T) {
		nodes, rels, err := qe.TraversePath("sym:NodeA", graph.DirOutbound, 10, graph.RelCalls)
		if err != nil {
			t.Fatalf("TraversePath failed: %v", err)
		}
		// Cycle NodeA -> NodeB -> NodeC -> NodeA must not loop infinitely
		if len(nodes) < 3 {
			t.Errorf("expected at least 3 nodes traversed in cycle, got %d", len(nodes))
		}
		if len(rels) < 3 {
			t.Errorf("expected at least 3 relationships traversed in cycle, got %d", len(rels))
		}
	})

	t.Run("TraversePath depth limit guard", func(t *testing.T) {
		_, _, err := qe.TraversePath("sym:NodeA", graph.DirOutbound, 150)
		if err != graph.ErrMaxDepthExceeded {
			t.Errorf("expected ErrMaxDepthExceeded for depth 150, got %v", err)
		}
	})

	t.Run("ReverseTraversal", func(t *testing.T) {
		nodes, _, err := qe.ReverseTraversal("sym:NodeB", 5, graph.RelCalls)
		if err != nil {
			t.Fatalf("ReverseTraversal failed: %v", err)
		}
		var foundD bool
		for _, n := range nodes {
			if n.Name() == "NodeD" {
				foundD = true
			}
		}
		if !foundD {
			t.Errorf("expected NodeD in reverse traversal from NodeB")
		}
	})

	t.Run("FilterNodes and FilterRelationships", func(t *testing.T) {
		symbols := qe.FilterNodes(func(n *graph.Node) bool {
			return n.Type() == graph.NodeSymbol
		})
		if len(symbols) < 5 {
			t.Errorf("expected at least 5 symbols, got %d", len(symbols))
		}

		callRels := qe.FilterRelationships(func(r *graph.Relationship) bool {
			return r.Type() == graph.RelCalls
		})
		if len(callRels) < 4 {
			t.Errorf("expected at least 4 call relationships, got %d", len(callRels))
		}
	})
}

// 5. Exporters (JSON, DOT, GraphML, Mermaid, Internal API)
func TestGraph_AllExporters(t *testing.T) {
	tempDir := t.TempDir()
	_, _, _, _, _, _, _, eng := setupTestPipeline(t)

	repoRoot := filepath.Join(tempDir, "exporter_suite_repo")
	_ = os.MkdirAll(repoRoot, 0755)

	_ = os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("# Exporter\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte("package main\nfunc Start() {}\n"), 0644)

	kg, err := eng.BuildGraphFromPath(repoRoot)
	if err != nil {
		t.Fatalf("BuildGraphFromPath failed: %v", err)
	}

	exporter := kg.Export()

	// 1. JSON Export
	t.Run("JSON Export is valid and complete", func(t *testing.T) {
		data, err := exporter.ToJSON()
		if err != nil {
			t.Fatalf("ToJSON failed: %v", err)
		}
		var model graph.InternalGraphModel
		if err := json.Unmarshal(data, &model); err != nil {
			t.Fatalf("json unmarshal failed: %v", err)
		}
		if model.TotalNodes != kg.TotalNodes() {
			t.Errorf("node count mismatch: %d vs %d", model.TotalNodes, kg.TotalNodes())
		}
		if model.TotalRelationships != kg.TotalRelationships() {
			t.Errorf("rel count mismatch: %d vs %d", model.TotalRelationships, kg.TotalRelationships())
		}
	})

	// 2. DOT Export
	t.Run("DOT Export syntax and determinism", func(t *testing.T) {
		dot := exporter.ToDOT()
		if !strings.HasPrefix(dot, "digraph KnowledgeGraph {") {
			t.Errorf("expected digraph header, got:\n%s", dot)
		}
		if !strings.HasSuffix(strings.TrimSpace(dot), "}") {
			t.Errorf("expected closing brace in DOT")
		}
	})

	// 3. GraphML Export
	t.Run("GraphML XML Export", func(t *testing.T) {
		graphml := exporter.ToGraphML()
		if !strings.Contains(graphml, "<graphml") || !strings.Contains(graphml, "</graphml>") {
			t.Errorf("invalid GraphML XML output:\n%s", graphml)
		}
	})

	// 4. Mermaid Export
	t.Run("Mermaid Flowchart Export", func(t *testing.T) {
		mermaid := exporter.ToMermaid()
		if !strings.HasPrefix(mermaid, "flowchart TD") {
			t.Errorf("expected flowchart TD header in Mermaid, got:\n%s", mermaid)
		}
	})

	// 5. Internal API DTO Export
	t.Run("Internal API Model", func(t *testing.T) {
		dto := exporter.ToInternalAPI()
		if dto == nil || dto.SchemaVersion == "" || len(dto.Nodes) == 0 {
			t.Errorf("invalid InternalGraphModel: %+v", dto)
		}
	})
}

// 6. Validation Engine (Integrity, Missing Nodes, Invalid Edges, Orphans, Performance)
func TestGraph_IntegrityValidationSuite(t *testing.T) {
	t.Run("missing node detected", func(t *testing.T) {
		n1 := graph.NewNode("file:a.go", graph.NodeFile, "a.go", "a.go", "", "", nil)
		r1 := graph.NewRelationship(graph.RelContains, "file:a.go", "sym:Missing", nil, nil)
		kg := graph.NewKnowledgeGraph("root", []*graph.Node{n1}, []*graph.Relationship{r1})

		val := kg.Validation()
		if len(val.MissingNodes()) != 1 {
			t.Fatalf("expected 1 missing node issue, got %d", len(val.MissingNodes()))
		}
		if val.MissingNodes()[0].Severity() != graph.ValMissingNode {
			t.Errorf("expected ValMissingNode severity")
		}
	})

	t.Run("invalid edge detected for incompatible types", func(t *testing.T) {
		n1 := graph.NewNode("doc:README.md", graph.NodeDoc, "README.md", "README.md", "", "", nil)
		n2 := graph.NewNode("sym:Foo", graph.NodeSymbol, "Foo", "main.go", "", "", nil)
		// Symbol cannot Document anything
		r1 := graph.NewRelationship(graph.RelDocuments, "sym:Foo", "doc:README.md", nil, nil)
		kg := graph.NewKnowledgeGraph("root", []*graph.Node{n1, n2}, []*graph.Relationship{r1})

		val := kg.Validation()
		if len(val.InvalidEdges()) != 1 {
			t.Fatalf("expected 1 invalid edge issue, got %d", len(val.InvalidEdges()))
		}
	})

	t.Run("orphan node identified", func(t *testing.T) {
		n1 := graph.NewNode("cfg:solo.json", graph.NodeConfig, "solo.json", "solo.json", "", "", nil)
		kg := graph.NewKnowledgeGraph("root", []*graph.Node{n1}, nil)

		val := kg.Validation()
		if len(val.OrphanNodes()) != 1 {
			t.Fatalf("expected 1 orphan node issue, got %d", len(val.OrphanNodes()))
		}
	})

	t.Run("performance benchmark validation", func(t *testing.T) {
		var nodes []*graph.Node
		for i := 0; i < 50; i++ {
			n := graph.NewNode(fmt.Sprintf("sym:Func%d", i), graph.NodeSymbol, fmt.Sprintf("Func%d", i), "main.go", "", "", nil)
			nodes = append(nodes, n)
		}
		kg := graph.NewKnowledgeGraph("root", nodes, nil)
		ve := graph.NewValidationEngine(kg)
		lookupDur, travDur := ve.ValidatePerformance()
		if lookupDur < 0 || travDur < 0 {
			t.Errorf("invalid performance durations: lookup=%v, trav=%v", lookupDur, travDur)
		}
	})
}

// 7. Determinism, Concurrency & Race Safety
func TestGraph_DeterminismAndRaceSafety(t *testing.T) {
	tempDir := t.TempDir()
	disc, metaCollector, langAnalyzer, depAnalyzer, indexer, symEngine, xrefEngine, _ := setupTestPipeline(t)

	repoRoot := filepath.Join(tempDir, "det_race_repo")
	_ = os.MkdirAll(repoRoot, 0755)

	for i := 1; i <= 8; i++ {
		src := fmt.Sprintf("package main\nfunc Action%d() {}\n", i)
		_ = os.WriteFile(filepath.Join(repoRoot, fmt.Sprintf("action_%d.go", i)), []byte(src), 0644)
	}

	discRes, _ := disc.DiscoverPath(repoRoot)
	profile, _ := metaCollector.Collect(discRes)
	structModel, _ := langAnalyzer.Analyze(discRes)
	depModel, _ := depAnalyzer.Analyze(discRes)
	indexModel, _ := indexer.Index(discRes)
	symModel, _ := symEngine.Parse(discRes)
	xrefModel, _ := xrefEngine.Analyze(discRes, symModel, depModel)

	eng1, _ := graph.NewWithWorkers(disc, metaCollector, langAnalyzer, depAnalyzer, indexer, symEngine, xrefEngine, 1)
	g1, err1 := eng1.BuildGraph(discRes, profile, structModel, depModel, indexModel, symModel, xrefModel)
	if err1 != nil {
		t.Fatalf("1-worker build failed: %v", err1)
	}

	eng4, _ := graph.NewWithWorkers(disc, metaCollector, langAnalyzer, depAnalyzer, indexer, symEngine, xrefEngine, 4)
	g4, err4 := eng4.BuildGraph(discRes, profile, structModel, depModel, indexModel, symModel, xrefModel)
	if err4 != nil {
		t.Fatalf("4-worker build failed: %v", err4)
	}

	// Verify exact equivalence of nodes and relationships
	if g1.TotalNodes() != g4.TotalNodes() || g1.TotalRelationships() != g4.TotalRelationships() {
		t.Fatalf("mismatch in counts: nodes %d vs %d, rels %d vs %d", g1.TotalNodes(), g4.TotalNodes(), g1.TotalRelationships(), g4.TotalRelationships())
	}

	for i := range g1.AllNodes() {
		if g1.AllNodes()[i].ID() != g4.AllNodes()[i].ID() {
			t.Fatalf("node ordering mismatch at %d: %s vs %s", i, g1.AllNodes()[i].ID(), g4.AllNodes()[i].ID())
		}
	}

	// Concurrent multi-threaded reads
	var wg sync.WaitGroup
	for w := 0; w < 16; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			qe := g4.Query()
			_ = qe.FindNodesByType(graph.NodeSymbol)
			_ = qe.Neighbors(g4.AllNodes()[0].ID(), graph.DirBoth)
			_, _, _ = qe.TraversePath(g4.AllNodes()[0].ID(), graph.DirOutbound, 5)
			_ = g4.Export().ToDOT()
			_, _ = g4.Export().ToJSON()
			_ = g4.Export().ToMermaid()
			_ = g4.Export().ToGraphML()
			_ = g4.Export().ToInternalAPI()
		}(w)
	}
	wg.Wait()
}

// 8. Regression Tests for Corrected Items
func TestGraph_CorrectedItemsRegressions(t *testing.T) {
	disc, metaCollector, langAnalyzer, depAnalyzer, indexer, symEngine, xrefEngine, engDefault := setupTestPipeline(t)

	t.Run("Engine NumWorkers accessor and bounds", func(t *testing.T) {
		if engDefault.NumWorkers() != 4 {
			t.Errorf("expected default 4 workers, got %d", engDefault.NumWorkers())
		}
		engCustom, err := graph.NewWithWorkers(disc, metaCollector, langAnalyzer, depAnalyzer, indexer, symEngine, xrefEngine, 8)
		if err != nil || engCustom.NumWorkers() != 8 {
			t.Errorf("expected 8 workers, got %d (err: %v)", engCustom.NumWorkers(), err)
		}
		engNegative, _ := graph.NewWithWorkers(disc, metaCollector, langAnalyzer, depAnalyzer, indexer, symEngine, xrefEngine, -5)
		if engNegative.NumWorkers() != 4 {
			t.Errorf("expected negative workers to fallback to 4, got %d", engNegative.NumWorkers())
		}
	})

	t.Run("GraphML Export uses canonical relationship IDs for edge elements", func(t *testing.T) {
		n1 := graph.NewNode("sym:A", graph.NodeSymbol, "A", "a.go", "", "", nil)
		n2 := graph.NewNode("sym:B", graph.NodeSymbol, "B", "b.go", "", "", nil)
		r1 := graph.NewRelationship(graph.RelCalls, "sym:A", "sym:B", []graph.ProvenanceSource{graph.ProvXRef}, nil)
		kg := graph.NewKnowledgeGraph("root", []*graph.Node{n1, n2}, []*graph.Relationship{r1})

		xmlOutput := kg.Export().ToGraphML()
		if !strings.Contains(xmlOutput, "<edge id=\"rel:calls:sym:A-&gt;sym:B\"") {
			t.Errorf("expected canonical relationship ID in GraphML edge, got:\n%s", xmlOutput)
		}
	})

	t.Run("TraversePath zero-depth returns only start node with 0 relationships", func(t *testing.T) {
		n1 := graph.NewNode("sym:A", graph.NodeSymbol, "A", "a.go", "", "", nil)
		n2 := graph.NewNode("sym:B", graph.NodeSymbol, "B", "b.go", "", "", nil)
		r1 := graph.NewRelationship(graph.RelCalls, "sym:A", "sym:B", []graph.ProvenanceSource{graph.ProvXRef}, nil)
		kg := graph.NewKnowledgeGraph("root", []*graph.Node{n1, n2}, []*graph.Relationship{r1})

		nodes, rels, err := kg.Query().TraversePath("sym:A", graph.DirOutbound, 0)
		if err != nil {
			t.Fatalf("zero-depth TraversePath failed: %v", err)
		}
		if len(nodes) != 1 || nodes[0].ID() != "sym:A" {
			t.Errorf("expected exactly 1 node (start node), got %d: %v", len(nodes), nodes)
		}
		if len(rels) != 0 {
			t.Errorf("expected 0 traversed relationships at depth 0, got %d", len(rels))
		}
	})

	t.Run("ValidationReport handles duplicate edge issues", func(t *testing.T) {
		issue := graph.NewValidationIssue(
			graph.ValDuplicateEdge,
			"GRAPH_DUPLICATE_EDGE",
			"duplicate relationship",
			"rel:calls:sym:A->sym:B",
			"sym:A",
			"sym:B",
			"",
		)
		report := graph.NewValidationReport([]*graph.ValidationIssue{issue})
		if len(report.DuplicateEdges()) != 1 {
			t.Fatalf("expected 1 duplicate edge in report, got %d", len(report.DuplicateEdges()))
		}
		if report.DuplicateEdges()[0].Code() != "GRAPH_DUPLICATE_EDGE" {
			t.Errorf("expected GRAPH_DUPLICATE_EDGE code")
		}
	})
}
