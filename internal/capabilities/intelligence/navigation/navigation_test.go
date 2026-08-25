package navigation_test

import (
	"sync"
	"testing"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/intelligence/crossrepo"
	"github.com/unhield/limoxel/internal/capabilities/intelligence/navigation"
	"github.com/unhield/limoxel/internal/capabilities/intelligence/semantic"
	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
	"github.com/unhield/limoxel/internal/capabilities/repository/xref"
)

func createTestSymbol(id, name, pkgPath, filePath string, kind symbol.SymbolKind, sig string, pos *symbol.SourcePosition, fields ...string) *symbol.Symbol {
	return symbol.NewSymbol(
		id,
		kind,
		name,
		"pkg",
		pkgPath,
		filePath,
		"",
		false,
		sig,
		"",
		false,
		nil,
		fields,
		pos,
		nil,
	)
}

func createTestMethodSymbol(id, name, recvType, pkgPath, filePath string, sig string, pos *symbol.SourcePosition) *symbol.Symbol {
	return symbol.NewSymbol(
		id,
		symbol.SymbolKindMethod,
		name,
		"pkg",
		pkgPath,
		filePath,
		recvType,
		true,
		sig,
		"",
		false,
		nil,
		nil,
		pos,
		nil,
	)
}

func TestDefinitionNavigation(t *testing.T) {
	pos1 := symbol.NewSourcePosition("pkg/auth/auth.go", 10, 1, 100)
	pos2 := symbol.NewSourcePosition("pkg/auth/service.go", 20, 1, 200)

	sym1 := createTestSymbol("sym:Authenticator", "Authenticator", "pkg/auth", "pkg/auth/auth.go", symbol.SymbolKindInterface, "type Authenticator interface", pos1, "Authenticate")
	sym2 := createTestSymbol("sym:AuthService", "AuthService", "pkg/auth", "pkg/auth/service.go", symbol.SymbolKindStruct, "type AuthService struct", pos2)
	sym3 := createTestMethodSymbol("sym:AuthService.Authenticate", "Authenticate", "AuthService", "pkg/auth", "pkg/auth/service.go", "func (s *AuthService) Authenticate() bool", pos2)

	symDB := symbol.NewSymbolDatabase([]*symbol.Symbol{sym1, sym2, sym3})
	nav := navigation.NewDefinitionNavigator(symDB, nil, nil, nil)

	// 1. Go to Definition
	defRes, err := nav.GoToDefinition("sym:AuthService")
	if err != nil || defRes.Target() == nil {
		t.Fatalf("GoToDefinition failed: %v", err)
	}
	if defRes.Target().SymbolID() != "sym:AuthService" || defRes.Target().FilePath() != "pkg/auth/service.go" {
		t.Fatalf("Unexpected definition target: %+v", defRes.Target())
	}

	// 2. Go to Declaration
	declRes, err := nav.GoToDeclaration("sym:AuthService.Authenticate")
	if err != nil || declRes.Target() == nil {
		t.Fatalf("GoToDeclaration failed: %v", err)
	}
	if declRes.Target().Name() != "Authenticate" || declRes.Target().NavKind() != navigation.NavKindDeclaration {
		t.Fatalf("Unexpected declaration target: %+v", declRes.Target())
	}

	// 3. Go to Implementation
	impls, err := nav.GoToImplementation("sym:Authenticator")
	if err != nil || len(impls) == 0 {
		t.Fatalf("GoToImplementation failed: err=%v, count=%d", err, len(impls))
	}
	if impls[0].Name() != "AuthService" {
		t.Fatalf("Expected AuthService to implement Authenticator, got %s", impls[0].Name())
	}

	// 4. Go to Package
	pkgTgt, err := nav.GoToPackage("sym:AuthService")
	if err != nil || pkgTgt == nil {
		t.Fatalf("GoToPackage failed: %v", err)
	}
	if pkgTgt.PackagePath() != "pkg/auth" {
		t.Fatalf("Unexpected package path: %s", pkgTgt.PackagePath())
	}

	// 5. Go to Module
	modTgt, err := nav.GoToModule("pkg/auth")
	if err != nil || modTgt == nil {
		t.Fatalf("GoToModule failed: %v", err)
	}
	if modTgt.NavKind() != navigation.NavKindModule {
		t.Fatalf("Expected NavKindModule, got %s", modTgt.NavKind())
	}
}

func TestReferenceNavigation(t *testing.T) {
	pos1 := symbol.NewSourcePosition("pkg/user/user.go", 10, 1, 100)
	pos2 := symbol.NewSourcePosition("cmd/server/main.go", 30, 5, 300)

	sym1 := createTestSymbol("sym:User", "User", "pkg/user", "pkg/user/user.go", symbol.SymbolKindStruct, "type User struct", pos1)
	symDB := symbol.NewSymbolDatabase([]*symbol.Symbol{sym1})

	ref := xref.NewReference("sym:Main", "sym:User", xref.RefStruct, "cmd/server/main.go", pos2, xref.StateResolved, "var u User")
	xrefDB := xref.NewReferenceDatabase([]*xref.Reference{ref})
	xrefModel := xref.NewXRefModel("repo", xrefDB, nil, nil, nil, nil, nil)

	nav := navigation.NewReferenceNavigator(symDB, xrefModel, nil, nil)

	// 1. Find References
	refs, err := nav.FindReferences("sym:User")
	if err != nil || refs.TotalCount() != 1 {
		t.Fatalf("FindReferences failed: err=%v, count=%d", err, refs.TotalCount())
	}
	if refs.References()[0].FilePath() != "cmd/server/main.go" {
		t.Fatalf("Unexpected reference file: %s", refs.References()[0].FilePath())
	}

	// 2. Find Usages
	usages, err := nav.FindUsages("sym:User")
	if err != nil || len(usages) != 1 {
		t.Fatalf("FindUsages failed: err=%v, count=%d", err, len(usages))
	}
	if usages[0].Kind() != navigation.UsageKindType {
		t.Fatalf("Expected UsageKindType, got %s", usages[0].Kind())
	}

	// 3. Reverse Lookup
	revs, err := nav.ReverseLookup("sym:User")
	if err != nil || len(revs) != 1 {
		t.Fatalf("ReverseLookup failed: err=%v, count=%d", err, len(revs))
	}
	if revs[0].RelKind() != navigation.RelKindReferences {
		t.Fatalf("Expected RelKindReferences, got %s", revs[0].RelKind())
	}

	// 4. Relationship Lookup
	rels, err := nav.RelationshipLookup("sym:User")
	if err != nil || len(rels) < 2 {
		t.Fatalf("RelationshipLookup failed: err=%v, count=%d", err, len(rels))
	}
}

func TestSymbolHierarchy(t *testing.T) {
	pos1 := symbol.NewSourcePosition("pkg/math/vector.go", 10, 1, 100)
	pos2 := symbol.NewSourcePosition("pkg/math/vector.go", 20, 1, 200)

	sym1 := createTestSymbol("sym:Vector", "Vector", "pkg/math", "pkg/math/vector.go", symbol.SymbolKindStruct, "type Vector struct", pos1, "X", "Y")
	sym2 := createTestMethodSymbol("sym:Vector.Norm", "Norm", "Vector", "pkg/math", "pkg/math/vector.go", "func (v *Vector) Norm() float64", pos2)

	symDB := symbol.NewSymbolDatabase([]*symbol.Symbol{sym1, sym2})
	nav := navigation.NewHierarchyNavigator(symDB, nil, nil)

	// 1. Parent Symbols
	parents, err := nav.GetParentSymbols("sym:Vector.Norm")
	if err != nil || len(parents) == 0 {
		t.Fatalf("GetParentSymbols failed: %v", err)
	}
	foundParent := false
	for _, p := range parents {
		if p.SymbolID() == "sym:Vector" {
			foundParent = true
			break
		}
	}
	if !foundParent {
		t.Fatalf("Expected parent Vector in parents, got %+v", parents)
	}

	// 2. Child Symbols
	children, err := nav.GetChildSymbols("sym:Vector")
	if err != nil || len(children) != 3 { // Method Norm, Field X, Field Y
		t.Fatalf("GetChildSymbols failed: err=%v, count=%d", err, len(children))
	}

	// 3. Package Hierarchy
	pkgNode, err := nav.GetPackageHierarchy("pkg/math")
	if err != nil || pkgNode == nil {
		t.Fatalf("GetPackageHierarchy failed: %v", err)
	}
	if len(pkgNode.ContainedFiles()) == 0 {
		t.Fatal("Expected contained files in package hierarchy node")
	}
}

func TestCallHierarchyAndCycles(t *testing.T) {
	pos := symbol.NewSourcePosition("pkg/service/eval.go", 10, 1, 100)

	symA := createTestSymbol("sym:FuncA", "FuncA", "pkg/service", "pkg/service/eval.go", symbol.SymbolKindFunction, "func FuncA()", pos)
	symB := createTestSymbol("sym:FuncB", "FuncB", "pkg/service", "pkg/service/eval.go", symbol.SymbolKindFunction, "func FuncB()", pos)
	symC := createTestSymbol("sym:FuncC", "FuncC", "pkg/service", "pkg/service/eval.go", symbol.SymbolKindFunction, "func FuncC()", pos)
	symDB := symbol.NewSymbolDatabase([]*symbol.Symbol{symA, symB, symC})

	// Mutual cycle: FuncA -> FuncB -> FuncC -> FuncA
	edge1 := xref.NewCallEdge("sym:FuncA", "sym:FuncB", xref.CallDirect, "pkg/service/eval.go", pos)
	edge2 := xref.NewCallEdge("sym:FuncB", "sym:FuncC", xref.CallDirect, "pkg/service/eval.go", pos)
	edge3 := xref.NewCallEdge("sym:FuncC", "sym:FuncA", xref.CallDirect, "pkg/service/eval.go", pos)

	callGraph := xref.NewCallGraph([]*xref.CallEdge{edge1, edge2, edge3}, nil, nil, nil, nil, nil)
	xrefModel := xref.NewXRefModel("repo", nil, callGraph, nil, nil, nil, nil)

	nav := navigation.NewCallHierarchyNavigator(symDB, xrefModel)

	// 1. Incoming Calls
	in, err := nav.GetIncomingCalls("sym:FuncB")
	if err != nil || len(in) != 1 || in[0].SymbolID() != "sym:FuncA" {
		t.Fatalf("GetIncomingCalls failed: err=%v, count=%d", err, len(in))
	}

	// 2. Outgoing Calls
	out, err := nav.GetOutgoingCalls("sym:FuncB")
	if err != nil || len(out) != 1 || out[0].SymbolID() != "sym:FuncC" {
		t.Fatalf("GetOutgoingCalls failed: err=%v, count=%d", err, len(out))
	}

	// 3. Recursive Paths
	cycles, err := nav.GetRecursivePaths("sym:FuncA")
	if err != nil || len(cycles) == 0 {
		t.Fatalf("GetRecursivePaths failed to detect cycle: err=%v, count=%d", err, len(cycles))
	}

	// 4. Dependency Chains
	chains, err := nav.GetDependencyChains("sym:FuncA", "sym:FuncC", 5)
	if err != nil || len(chains) == 0 {
		t.Fatalf("GetDependencyChains failed: err=%v, count=%d", err, len(chains))
	}
	if chains[0].TotalLength() != 3 { // FuncA -> FuncB -> FuncC
		t.Fatalf("Expected chain length 3, got %d", chains[0].TotalLength())
	}

	// 5. Call Depth
	depth, err := nav.CalculateCallDepth("sym:FuncA", 10)
	if err != nil || depth < 2 {
		t.Fatalf("CalculateCallDepth unexpected depth: %d, err=%v", depth, err)
	}
}

func TestNavigationEngineEndToEndAndConcurrency(t *testing.T) {
	pos := symbol.NewSourcePosition("pkg/user/user.go", 10, 1, 100)
	sym := createTestSymbol("sym:User", "User", "pkg/user", "pkg/user/user.go", symbol.SymbolKindStruct, "type User struct", pos)
	symDB := symbol.NewSymbolDatabase([]*symbol.Symbol{sym})

	engine := navigation.New()
	model, err := engine.Analyze(navigation.AnalysisParams{
		SymbolDB: symDB,
	})

	if err != nil || model == nil {
		t.Fatalf("Engine Analyze failed: %v", err)
	}
	if model.Definition("sym:User") == nil {
		t.Fatal("Expected definition for sym:User in NavigationModel")
	}

	// Concurrency test across 20 goroutines
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = engine.GoToDefinition("sym:User")
			_, _ = engine.GoToPackage("sym:User")
			_, _ = engine.GetChildSymbols("sym:User")
			_ = engine.Model().ValidationReport()
		}()
	}
	wg.Wait()
}

func TestNavigationErrorsAndIsolation(t *testing.T) {
	navErr := navigation.NewNavigationError(navigation.ErrCatInvalidInput, "ERR_CODE", "test error")
	if !navigation.IsCategory(navErr, navigation.ErrCatInvalidInput) {
		t.Fatal("IsCategory returned false for matching category")
	}

	// Ambiguous Symbol Collision
	pos1 := symbol.NewSourcePosition("pkg/a/foo.go", 10, 1, 100)
	pos2 := symbol.NewSourcePosition("pkg/b/foo.go", 10, 1, 100)
	symA := createTestSymbol("sym:pkg/a.Foo", "Foo", "pkg/a", "pkg/a/foo.go", symbol.SymbolKindFunction, "func Foo()", pos1)
	symB := createTestSymbol("sym:pkg/b.Foo", "Foo", "pkg/b", "pkg/b/foo.go", symbol.SymbolKindFunction, "func Foo()", pos2)
	symDB := symbol.NewSymbolDatabase([]*symbol.Symbol{symA, symB})

	nav := navigation.NewDefinitionNavigator(symDB, nil, nil, nil)
	defRes, err := nav.GoToDefinition("Foo")
	if err != navigation.ErrTargetAmbiguous || defRes == nil || defRes.State() != navigation.NavStateAmbiguous {
		t.Fatalf("Expected ErrTargetAmbiguous for name collision, got err=%v, res=%v", err, defRes)
	}
	if len(defRes.Candidates()) != 2 {
		t.Fatalf("Expected 2 candidate targets, got %d", len(defRes.Candidates()))
	}
}

func TestSemanticAndCrossRepoNavigationIntegration(t *testing.T) {
	// Semantic Model setup
	ifaceType := semantic.NewSemanticType("type:Reader", "Reader", semantic.TypeInterface, "pkg/io", "pkg/io/reader.go", "", false, "", true, nil, nil, nil, nil, nil, semantic.StateResolved)
	implType := semantic.NewSemanticType("type:FileReader", "FileReader", semantic.TypeCustom, "pkg/file", "pkg/file/reader.go", "", false, "", true, nil, nil, nil, []string{"Reader"}, nil, semantic.StateResolved)
	typeMap := map[string]*semantic.SemanticType{
		ifaceType.ID(): ifaceType,
		implType.ID():  implType,
	}
	semModel := semantic.NewSemanticModel(nil, nil, typeMap, nil, nil, nil, nil, nil, nil, nil, time.Time{})

	// CrossRepo Model setup
	repo := crossrepo.NewWorkspaceRepository("d:/repo1", "repo1", []string{"github.com/org/repo1"}, []string{"pkg/io", "pkg/file"}, nil)
	ws := crossrepo.NewWorkspaceModel("d:/repo1", []*crossrepo.WorkspaceRepository{repo}, nil, nil, nil, nil)
	dep := crossrepo.NewCrossFileDependency("pkg/file/reader.go", "pkg/io/reader.go", "pkg/io", []string{"Reader"}, true)
	crossModel := crossrepo.NewCrossRepoModel(nil, nil, []*crossrepo.CrossFileDependency{dep}, nil, nil, nil, nil, nil, nil, nil, nil, ws, nil, nil, time.Time{})

	nav := navigation.NewDefinitionNavigator(nil, nil, semModel, crossModel)
	refNav := navigation.NewReferenceNavigator(nil, nil, semModel, crossModel)

	// GoToImplementation via semantic type model
	impls, err := nav.GoToImplementation("type:Reader")
	if err != nil || len(impls) == 0 {
		t.Fatalf("GoToImplementation failed via semantic model: err=%v, count=%d", err, len(impls))
	}
	if impls[0].ID() != "impl:type:FileReader" {
		t.Fatalf("Unexpected implementation target: %s", impls[0].ID())
	}

	// GoToModule via workspace model
	modTgt, err := nav.GoToModule("pkg/file")
	if err != nil || modTgt == nil {
		t.Fatalf("GoToModule failed: %v", err)
	}
	if modTgt.ModulePath() != "github.com/org/repo1" {
		t.Fatalf("Expected module github.com/org/repo1, got %s", modTgt.ModulePath())
	}

	// DependencyLookup via crossrepo dependencies
	deps, err := refNav.DependencyLookup("pkg/file/reader.go", "outbound")
	if err != nil || len(deps) == 0 {
		t.Fatalf("DependencyLookup failed: err=%v, count=%d", err, len(deps))
	}
	if deps[0].TargetID() != "pkg/io/reader.go" {
		t.Fatalf("Unexpected dependency target: %s", deps[0].TargetID())
	}

	// ReverseLookup implementations via semantic types
	revs, err := refNav.ReverseLookup("type:Reader")
	if err != nil || len(revs) == 0 {
		t.Fatalf("ReverseLookup failed: err=%v, count=%d", err, len(revs))
	}
	if revs[0].RelKind() != navigation.RelKindImplements {
		t.Fatalf("Expected RelKindImplements, got %s", revs[0].RelKind())
	}
}

func TestDeterminismAndOrdering(t *testing.T) {
	pos := symbol.NewSourcePosition("pkg/a/a.go", 10, 1, 100)
	symA := createTestSymbol("sym:Alpha", "Alpha", "pkg/a", "pkg/a/a.go", symbol.SymbolKindStruct, "type Alpha struct", pos, "Z", "A", "M")
	symDB := symbol.NewSymbolDatabase([]*symbol.Symbol{symA})

	engine := navigation.New()
	m1, err1 := engine.Analyze(navigation.AnalysisParams{SymbolDB: symDB})
	m2, err2 := engine.Analyze(navigation.AnalysisParams{SymbolDB: symDB})

	if err1 != nil || err2 != nil {
		t.Fatalf("Analyze failed: err1=%v, err2=%v", err1, err2)
	}

	node1 := m1.SymbolHierarchyNode("sym:Alpha")
	node2 := m2.SymbolHierarchyNode("sym:Alpha")

	if len(node1.Children()) != len(node2.Children()) {
		t.Fatalf("Children length mismatch: %d vs %d", len(node1.Children()), len(node2.Children()))
	}
	for i := range node1.Children() {
		if node1.Children()[i].ID() != node2.Children()[i].ID() {
			t.Fatalf("Deterministic child ordering mismatch at %d: %s vs %s", i, node1.Children()[i].ID(), node2.Children()[i].ID())
		}
	}
}

func TestAdversarialEmptyAndMissingEntities(t *testing.T) {
	// 1. Empty Repository / Symbol Database
	emptySymDB := symbol.NewSymbolDatabase([]*symbol.Symbol{})
	nav := navigation.NewDefinitionNavigator(emptySymDB, nil, nil, nil)

	_, err := nav.GoToDefinition("non_existent")
	if err != navigation.ErrSymbolNotFound {
		t.Fatalf("Expected ErrSymbolNotFound for non-existent symbol, got %v", err)
	}

	_, err = nav.GoToDeclaration("")
	if err != navigation.ErrEmptyTarget {
		t.Fatalf("Expected ErrEmptyTarget for empty string, got %v", err)
	}

	impls, err := nav.GoToImplementation("non_existent_interface")
	if err != nil || len(impls) != 0 {
		t.Fatalf("Expected empty implementations for missing interface, got %d, err=%v", len(impls), err)
	}

	// 2. Missing Targets and Broken References in Validation
	pos := symbol.NewSourcePosition("pkg/main.go", 10, 1, 100)
	brokenRef := xref.NewReference("sym:Caller", "sym:MissingCallee", xref.RefFunction, "pkg/main.go", pos, xref.StateBroken, "call MissingCallee()")
	xrefDB := xref.NewReferenceDatabase([]*xref.Reference{brokenRef})
	xrefModel := xref.NewXRefModel("repo", xrefDB, nil, nil, nil, nil, nil)

	engine := navigation.New()
	model, err := engine.Analyze(navigation.AnalysisParams{
		SymbolDB:  emptySymDB,
		XRefModel: xrefModel,
	})
	if err != nil || model == nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	report := model.ValidationReport()
	if report == nil || report.BrokenRefCount() == 0 {
		t.Fatalf("Expected validation report to flag broken reference, got %+v", report)
	}
	if report.IsValid() {
		t.Fatal("Expected report.IsValid() == false due to broken reference")
	}
}

func TestAdversarialMultiCyclesAndComplexPaths(t *testing.T) {
	pos := symbol.NewSourcePosition("pkg/graph/graph.go", 10, 1, 100)

	symA := createTestSymbol("sym:A", "A", "pkg/graph", "pkg/graph/graph.go", symbol.SymbolKindFunction, "func A()", pos)
	symB := createTestSymbol("sym:B", "B", "pkg/graph", "pkg/graph/graph.go", symbol.SymbolKindFunction, "func B()", pos)
	symC := createTestSymbol("sym:C", "C", "pkg/graph", "pkg/graph/graph.go", symbol.SymbolKindFunction, "func C()", pos)
	symD := createTestSymbol("sym:D", "D", "pkg/graph", "pkg/graph/graph.go", symbol.SymbolKindFunction, "func D()", pos)
	symE := createTestSymbol("sym:E", "E", "pkg/graph", "pkg/graph/graph.go", symbol.SymbolKindFunction, "func E()", pos)

	symDB := symbol.NewSymbolDatabase([]*symbol.Symbol{symA, symB, symC, symD, symE})

	// Direct recursion: A -> A
	// 3-node cycle with branch: B -> C -> D -> B and C -> E (acyclic branch)
	edgeAA := xref.NewCallEdge("sym:A", "sym:A", xref.CallDirect, "pkg/graph/graph.go", pos)
	edgeBC := xref.NewCallEdge("sym:B", "sym:C", xref.CallDirect, "pkg/graph/graph.go", pos)
	edgeCD := xref.NewCallEdge("sym:C", "sym:D", xref.CallDirect, "pkg/graph/graph.go", pos)
	edgeDB := xref.NewCallEdge("sym:D", "sym:B", xref.CallDirect, "pkg/graph/graph.go", pos)
	edgeCE := xref.NewCallEdge("sym:C", "sym:E", xref.CallDirect, "pkg/graph/graph.go", pos)

	callGraph := xref.NewCallGraph([]*xref.CallEdge{edgeAA, edgeBC, edgeCD, edgeDB, edgeCE}, nil, nil, nil, nil, nil)
	xrefModel := xref.NewXRefModel("repo", nil, callGraph, nil, nil, nil, nil)

	callNav := navigation.NewCallHierarchyNavigator(symDB, xrefModel)

	// 1. Direct Recursion
	cyclesA, err := callNav.GetRecursivePaths("sym:A")
	if err != nil || len(cyclesA) != 1 || !cyclesA[0].IsDirect() {
		t.Fatalf("Expected 1 direct recursive cycle for A, got count=%d, err=%v", len(cyclesA), err)
	}

	// 2. 3-node Mutual Cycle
	cyclesB, err := callNav.GetRecursivePaths("sym:B")
	if err != nil || len(cyclesB) != 1 || cyclesB[0].IsDirect() || cyclesB[0].Length() != 4 {
		t.Fatalf("Expected 3-node mutual cycle for B, got length=%d, count=%d, err=%v", cyclesB[0].Length(), len(cyclesB), err)
	}

	// 3. Dependency chains along acyclic branch: B -> C -> E
	chains, err := callNav.GetDependencyChains("sym:B", "sym:E", 10)
	if err != nil || len(chains) == 0 {
		t.Fatalf("GetDependencyChains failed: %v", err)
	}
	if chains[0].TotalLength() != 3 { // B -> C -> E
		t.Fatalf("Expected chain length 3, got %d", chains[0].TotalLength())
	}

	// 4. Disconnected graph query
	chainsDisc, err := callNav.GetDependencyChains("sym:A", "sym:E", 10)
	if err != nil || len(chainsDisc) != 0 {
		t.Fatalf("Expected 0 chains for disconnected symbols, got %d, err=%v", len(chainsDisc), err)
	}
}

func TestAdversarialMultiRepoAndModuleIsolation(t *testing.T) {
	pos := symbol.NewSourcePosition("pkg/service/service.go", 10, 1, 100)

	// Repository 1 / Module 1 / Package service / Symbol Handler
	sym1 := symbol.NewSymbol("sym:repo1/mod1/pkg/service.Handler", symbol.SymbolKindStruct, "Handler", "service", "pkg/service", "repo1/pkg/service/service.go", "", false, "type Handler struct", "", false, nil, nil, pos, nil)
	// Repository 2 / Module 2 / Package service / Symbol Handler
	sym2 := symbol.NewSymbol("sym:repo2/mod2/pkg/service.Handler", symbol.SymbolKindStruct, "Handler", "service", "pkg/service", "repo2/pkg/service/service.go", "", false, "type Handler struct", "", false, nil, nil, pos, nil)

	symDB := symbol.NewSymbolDatabase([]*symbol.Symbol{sym1, sym2})

	repo1 := crossrepo.NewWorkspaceRepository("d:/workspace/repo1", "repo1", []string{"github.com/org/mod1"}, []string{"pkg/service"}, nil)
	repo2 := crossrepo.NewWorkspaceRepository("d:/workspace/repo2", "repo2", []string{"github.com/org/mod2"}, []string{"pkg/service"}, nil)
	ws := crossrepo.NewWorkspaceModel("d:/workspace", []*crossrepo.WorkspaceRepository{repo1, repo2}, nil, nil, nil, nil)
	crossModel := crossrepo.NewCrossRepoModel(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, ws, nil, nil, time.Time{})

	nav := navigation.NewDefinitionNavigator(symDB, nil, nil, crossModel)

	// Exact symbol lookup should resolve without collision
	tgt1, err1 := nav.GoToDefinition("sym:repo1/mod1/pkg/service.Handler")
	if err1 != nil || tgt1.Target().FilePath() != "repo1/pkg/service/service.go" {
		t.Fatalf("Failed exact resolution for repo1 Handler: %+v, err=%v", tgt1, err1)
	}

	tgt2, err2 := nav.GoToDefinition("sym:repo2/mod2/pkg/service.Handler")
	if err2 != nil || tgt2.Target().FilePath() != "repo2/pkg/service/service.go" {
		t.Fatalf("Failed exact resolution for repo2 Handler: %+v, err=%v", tgt2, err2)
	}

	// Ambiguous short name query should fail safely with candidates
	ambTgt, errAmb := nav.GoToDefinition("Handler")
	if errAmb != navigation.ErrTargetAmbiguous || ambTgt.State() != navigation.NavStateAmbiguous || len(ambTgt.Candidates()) != 2 {
		t.Fatalf("Expected ErrTargetAmbiguous with 2 candidates, got err=%v, res=%+v", errAmb, ambTgt)
	}
}

func TestAdversarialEngineReusabilityAndOrdering(t *testing.T) {
	pos := symbol.NewSourcePosition("pkg/core/core.go", 10, 1, 100)

	symA := createTestSymbol("sym:Alpha", "Alpha", "pkg/core", "pkg/core/core.go", symbol.SymbolKindFunction, "func Alpha()", pos)
	symB := createTestSymbol("sym:Beta", "Beta", "pkg/core", "pkg/core/core.go", symbol.SymbolKindFunction, "func Beta()", pos)

	symDB1 := symbol.NewSymbolDatabase([]*symbol.Symbol{symA})
	symDB2 := symbol.NewSymbolDatabase([]*symbol.Symbol{symB})

	engine := navigation.New()

	// Analyze DB 1
	m1, err1 := engine.Analyze(navigation.AnalysisParams{SymbolDB: symDB1})
	if err1 != nil || m1.Definition("sym:Alpha") == nil || m1.Definition("sym:Beta") != nil {
		t.Fatalf("Engine state leak or failure on DB1: %+v", m1)
	}

	// Re-analyze DB 2 with the same engine instance
	m2, err2 := engine.Analyze(navigation.AnalysisParams{SymbolDB: symDB2})
	if err2 != nil || m2.Definition("sym:Beta") == nil || m2.Definition("sym:Alpha") != nil {
		t.Fatalf("Engine state leak or failure on DB2: %+v", m2)
	}
}
