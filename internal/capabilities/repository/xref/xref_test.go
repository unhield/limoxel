package xref_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/repository/dependency"
	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
	"github.com/unhield/limoxel/internal/capabilities/repository/xref"
	langreg "github.com/unhield/limoxel/internal/language"
)

func setupTestLanguageRegistry(t *testing.T) *langreg.Registry {
	t.Helper()
	reg := langreg.NewRegistry()

	goLang, _ := langreg.New("go", "Go", []string{".go"}, nil, []string{"golang"})
	_ = reg.Register(goLang)

	return reg
}

func setupTestPipeline(t *testing.T) (*discovery.Discoverer, *symbol.Engine, *dependency.Analyzer, *xref.Engine) {
	t.Helper()
	reg := setupTestLanguageRegistry(t)
	disc, err := discovery.New(reg)
	if err != nil {
		t.Fatalf("failed creating discoverer: %v", err)
	}

	depAnalyzer, _ := dependency.New(disc)
	symEngine, _ := symbol.New(disc)

	eng, err := xref.New(disc, symEngine, depAnalyzer)
	if err != nil {
		t.Fatalf("xref.New failed: %v", err)
	}

	return disc, symEngine, depAnalyzer, eng
}

func TestXRef_ConstructorAndValidation(t *testing.T) {
	_, symEngine, depAnalyzer, _ := setupTestPipeline(t)

	t.Run("nil discoverer returns error", func(t *testing.T) {
		eng, err := xref.New(nil, symEngine, depAnalyzer)
		if err != xref.ErrNilDiscoverer || eng != nil {
			t.Errorf("expected ErrNilDiscoverer, got %v", err)
		}
	})

	t.Run("nil receiver methods return safe errors", func(t *testing.T) {
		var nilEng *xref.Engine
		if _, err := nilEng.Analyze(nil, nil, nil); err != xref.ErrNilEngine {
			t.Errorf("expected ErrNilEngine, got %v", err)
		}
		if _, err := nilEng.AnalyzeRepository(nil); err != xref.ErrNilEngine {
			t.Errorf("expected ErrNilEngine, got %v", err)
		}
		if _, err := nilEng.AnalyzePath(""); err != xref.ErrNilEngine {
			t.Errorf("expected ErrNilEngine, got %v", err)
		}
	})

	t.Run("empty path returns ErrPathEmpty", func(t *testing.T) {
		_, _, _, eng := setupTestPipeline(t)
		if _, err := eng.AnalyzePath("   "); err != xref.ErrPathEmpty {
			t.Errorf("expected ErrPathEmpty, got %v", err)
		}
	})
}

func TestXRef_ReferencesAndCallGraph(t *testing.T) {
	tempDir := t.TempDir()
	_, _, _, eng := setupTestPipeline(t)

	repoRoot := filepath.Join(tempDir, "call_repo")
	_ = os.MkdirAll(repoRoot, 0755)

	src := `package main

type Service struct {
	Name string
}

func (s *Service) Process() string {
	return Helper()
}

func Helper() string {
	return "ok"
}

func DirectRecurse(n int) int {
	if n <= 0 {
		return 0
	}
	return DirectRecurse(n - 1)
}

func MutualA(n int) int {
	if n <= 0 {
		return 0
	}
	return MutualB(n - 1)
}

func MutualB(n int) int {
	return MutualA(n)
}

func UnreachableFunc() {
	println("never called")
}

func main() {
	srv := &Service{Name: "app"}
	srv.Process()
	DirectRecurse(5)
	MutualA(3)
}
`
	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte(src), 0644)

	model, err := eng.AnalyzePath(repoRoot)
	if err != nil {
		t.Fatalf("AnalyzePath failed: %v", err)
	}

	refDB := model.References()
	if refDB.TotalCount() == 0 {
		t.Fatal("expected references, got 0")
	}

	callGraph := model.CallGraph()
	if callGraph.TotalEdges() == 0 {
		t.Fatal("expected call edges, got 0")
	}

	// 1. Verify Helper callers includes (*Service).Process
	callersOfHelper := callGraph.Callers("Helper")
	if len(callersOfHelper) == 0 || callersOfHelper[0] != "(*Service).Process" {
		t.Errorf("expected (*Service).Process caller for Helper, got: %v", callersOfHelper)
	}

	// 2. Verify direct recursion detected
	cycles := callGraph.RecursiveCycles()
	var hasDirect, hasMutual bool
	for _, cyc := range cycles {
		if len(cyc) == 2 && cyc[0] == "DirectRecurse" && cyc[1] == "DirectRecurse" {
			hasDirect = true
		}
		if len(cyc) >= 3 && (cyc[0] == "MutualA" || cyc[0] == "MutualB") {
			hasMutual = true
		}
	}
	if !hasDirect {
		t.Errorf("expected direct recursion for DirectRecurse, got cycles: %v", cycles)
	}
	if !hasMutual {
		t.Errorf("expected mutual recursion between MutualA and MutualB, got cycles: %v", cycles)
	}

	// 3. Verify entry points
	entries := callGraph.EntryPoints()
	var hasMain bool
	for _, e := range entries {
		if e == "main" || strings.HasSuffix(e, ".main") {
			hasMain = true
		}
	}
	if !hasMain {
		t.Errorf("expected main entry point, got: %v", entries)
	}

	// 4. Verify dead function reachability
	if callGraph.Reachability("UnreachableFunc") != xref.UnreachableConfirmed {
		t.Errorf("expected UnreachableConfirmed for UnreachableFunc, got: %v", callGraph.Reachability("UnreachableFunc"))
	}
}

func TestXRef_NavigationEngine(t *testing.T) {
	tempDir := t.TempDir()
	_, _, _, eng := setupTestPipeline(t)

	repoRoot := filepath.Join(tempDir, "nav_repo")
	pkgPath := filepath.Join(repoRoot, "pkg", "auth")
	_ = os.MkdirAll(pkgPath, 0755)

	src := `package auth

type Authenticator interface {
	Authenticate(token string) bool
}

type TokenAuth struct{}

func (ta *TokenAuth) Authenticate(token string) bool {
	return token != ""
}

func Verify(a Authenticator, token string) bool {
	return a.Authenticate(token)
}
`
	_ = os.WriteFile(filepath.Join(pkgPath, "auth.go"), []byte(src), 0644)

	model, err := eng.AnalyzePath(repoRoot)
	if err != nil {
		t.Fatalf("AnalyzePath failed: %v", err)
	}

	nav := model.Navigation()

	// 1. Go to definition
	defResult, err := nav.GoToDefinition("pkg/auth.Authenticator")
	if err != nil || defResult.TargetSymbol() == nil || defResult.TargetSymbol().Name() != "Authenticator" {
		t.Fatalf("GoToDefinition for Authenticator failed: %v, result: %+v", err, defResult)
	}

	// 2. Find implementations
	impls := nav.FindImplementations("pkg/auth.Authenticator")
	if len(impls) == 0 || impls[0].Name() != "TokenAuth" {
		t.Fatalf("FindImplementations for Authenticator failed, got: %+v", impls)
	}

	// 3. Find references
	refs := nav.FindReferences("pkg/auth.Authenticator")
	if len(refs) == 0 {
		t.Fatalf("expected references targeting Authenticator, got 0")
	}

	// 4. Package navigation
	pNav := nav.PackageNavigation("pkg/auth")
	if pNav == nil || len(pNav.Symbols()) == 0 {
		t.Fatalf("PackageNavigation failed: %+v", pNav)
	}
}

func TestXRef_ChangeImpactAnalyzer(t *testing.T) {
	tempDir := t.TempDir()
	_, _, _, eng := setupTestPipeline(t)

	repoRoot := filepath.Join(tempDir, "impact_repo")
	_ = os.MkdirAll(repoRoot, 0755)

	srcA := `package main
func CoreUtil() string {
	return "core"
}
`
	srcB := `package main
func Feature() string {
	return CoreUtil()
}
`
	_ = os.WriteFile(filepath.Join(repoRoot, "core.go"), []byte(srcA), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "feature.go"), []byte(srcB), 0644)

	model, err := eng.AnalyzePath(repoRoot)
	if err != nil {
		t.Fatalf("AnalyzePath failed: %v", err)
	}

	impact := model.Impact()

	// 1. File impact
	fileImpact := impact.AnalyzeFileImpact("core.go")
	if len(fileImpact.ImpactedSymbols()) == 0 {
		t.Fatalf("expected impacted symbols for core.go, got 0")
	}

	// 2. Symbol impact
	symImpact := impact.AnalyzeSymbolImpact("CoreUtil")
	if len(symImpact.DirectlyImpactedSymbols()) == 0 || symImpact.DirectlyImpactedSymbols()[0] != "Feature" {
		t.Errorf("expected Feature directly impacted by CoreUtil, got: %+v", symImpact)
	}
}

func TestXRef_BreakingChangeDetection(t *testing.T) {
	tempDir := t.TempDir()
	disc, symEngine, _, eng := setupTestPipeline(t)

	repoRoot := filepath.Join(tempDir, "break_repo")
	_ = os.MkdirAll(repoRoot, 0755)

	fileA := filepath.Join(repoRoot, "a.go")
	fileB := filepath.Join(repoRoot, "b.go")

	_ = os.WriteFile(fileA, []byte("package main\nfunc LegacyAPI() string { return \"v1\" }\n"), 0644)
	_ = os.WriteFile(fileB, []byte("package main\nfunc Consumer() string { return LegacyAPI() }\n"), 0644)

	disc1, _ := disc.DiscoverPath(repoRoot)
	sym1, _ := symEngine.Parse(disc1)
	m1, _ := eng.Analyze(disc1, sym1, nil)

	// Remove LegacyAPI in state 2
	_ = os.WriteFile(fileA, []byte("package main\nfunc NewAPI() string { return \"v2\" }\n"), 0644)

	disc2, _ := disc.DiscoverPath(repoRoot)
	sym2, _ := symEngine.Parse(disc2)

	breaking := m1.Impact().DetectBreakingChanges(sym1, sym2)
	if breaking.ConfirmedCount() == 0 {
		t.Fatalf("expected confirmed breaking changes for removed LegacyAPI, got: %+v", breaking)
	}
}

func TestXRef_ValidationEngine(t *testing.T) {
	tempDir := t.TempDir()
	_, _, _, eng := setupTestPipeline(t)

	repoRoot := filepath.Join(tempDir, "val_repo")
	_ = os.MkdirAll(repoRoot, 0755)

	src := `package main

func Recurse1() { Recurse2() }
func Recurse2() { Recurse1() }

func main() {
	Recurse1()
}
`
	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte(src), 0644)

	model, err := eng.AnalyzePath(repoRoot)
	if err != nil {
		t.Fatalf("AnalyzePath failed: %v", err)
	}

	val := model.Validation()
	if val.TotalIssues() == 0 {
		t.Fatal("expected circular recursion validation issue, got 0")
	}

	var hasCircular bool
	for _, issue := range val.CircularReferences() {
		if issue.Severity() == xref.ValidationCircularRef {
			hasCircular = true
		}
	}
	if !hasCircular {
		t.Errorf("expected circular reference validation issue, got: %+v", val.Issues())
	}
}

func TestXRef_IncrementalEquivalence(t *testing.T) {
	tempDir := t.TempDir()
	disc, symEngine, depAnalyzer, eng := setupTestPipeline(t)

	repoRoot := filepath.Join(tempDir, "incr_repo")
	_ = os.MkdirAll(repoRoot, 0755)

	f1 := filepath.Join(repoRoot, "f1.go")
	f2 := filepath.Join(repoRoot, "f2.go")

	_ = os.WriteFile(f1, []byte("package main\nfunc Calc() int { return 10 }\n"), 0644)
	_ = os.WriteFile(f2, []byte("package main\nfunc Run() int { return Calc() }\n"), 0644)

	disc1, _ := disc.DiscoverPath(repoRoot)
	sym1, _ := symEngine.Parse(disc1)
	dep1, _ := depAnalyzer.Analyze(disc1)
	m1, _ := eng.Analyze(disc1, sym1, dep1)

	// Modify f1.go
	_ = os.WriteFile(f1, []byte("package main\nfunc Calc() int { return 20 }\n"), 0644)

	disc2, _ := disc.DiscoverPath(repoRoot)
	sym2, _ := symEngine.Parse(disc2)
	dep2, _ := depAnalyzer.Analyze(disc2)

	mIncr, err := eng.AnalyzeIncremental(disc2, sym2, dep2, m1)
	if err != nil {
		t.Fatalf("incremental analysis failed: %v", err)
	}
	mFull, err := eng.Analyze(disc2, sym2, dep2)
	if err != nil {
		t.Fatalf("full analysis failed: %v", err)
	}

	if mIncr.References().TotalCount() != mFull.References().TotalCount() {
		t.Errorf("reference count mismatch: incr=%d vs full=%d", mIncr.References().TotalCount(), mFull.References().TotalCount())
	}
	if mIncr.CallGraph().TotalEdges() != mFull.CallGraph().TotalEdges() {
		t.Errorf("call edge count mismatch: incr=%d vs full=%d", mIncr.CallGraph().TotalEdges(), mFull.CallGraph().TotalEdges())
	}
}

func TestXRef_ResultImmutability(t *testing.T) {
	tempDir := t.TempDir()
	_, _, _, eng := setupTestPipeline(t)

	repoRoot := filepath.Join(tempDir, "immut_repo")
	_ = os.MkdirAll(repoRoot, 0755)

	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte("package main\nfunc Foo() {}\nfunc main() { Foo() }\n"), 0644)

	model, err := eng.AnalyzePath(repoRoot)
	if err != nil {
		t.Fatalf("AnalyzePath failed: %v", err)
	}

	allRefs := model.References().AllReferences()
	initialCount := len(allRefs)
	if initialCount == 0 {
		t.Fatal("expected references, got 0")
	}

	allRefs[0] = nil
	if model.References().AllReferences()[0] == nil {
		t.Errorf("mutation of returned slice altered internal ReferenceDatabase")
	}

	edges := model.CallGraph().AllEdges()
	if len(edges) > 0 {
		edges[0] = nil
		if model.CallGraph().AllEdges()[0] == nil {
			t.Errorf("mutation of returned slice altered internal CallGraph")
		}
	}
}

func TestXRef_AdversarialCrossPackageAndAmbiguity(t *testing.T) {
	tempDir := t.TempDir()
	_, _, _, eng := setupTestPipeline(t)

	repoRoot := filepath.Join(tempDir, "ambig_repo")
	pkgA := filepath.Join(repoRoot, "pkg", "auth")
	pkgB := filepath.Join(repoRoot, "pkg", "billing")
	_ = os.MkdirAll(pkgA, 0755)
	_ = os.MkdirAll(pkgB, 0755)

	_ = os.WriteFile(filepath.Join(pkgA, "service.go"), []byte("package auth\ntype Service struct{}\nfunc (s *Service) Execute() {}\n"), 0644)
	_ = os.WriteFile(filepath.Join(pkgB, "service.go"), []byte("package billing\ntype Service struct{}\nfunc (s *Service) Execute() {}\n"), 0644)

	mainSrc := `package main

import (
	"pkg/auth"
	"pkg/billing"
)

func main() {
	s1 := &auth.Service{}
	s1.Execute()
	s2 := &billing.Service{}
	s2.Execute()
}
`
	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte(mainSrc), 0644)

	model, err := eng.AnalyzePath(repoRoot)
	if err != nil {
		t.Fatalf("AnalyzePath failed: %v", err)
	}

	nav := model.Navigation()

	// 1. Ambiguous query by common name "Service" returns ErrAmbiguousDefinition and multiple candidates
	defRes, err := nav.GoToDefinition("Service")
	if err != xref.ErrAmbiguousDefinition || defRes.State() != xref.StateAmbiguous {
		t.Errorf("expected ErrAmbiguousDefinition and StateAmbiguous for 'Service', got err=%v, res=%+v", err, defRes)
	}
	if len(defRes.Candidates()) != 2 {
		t.Errorf("expected 2 candidates for 'Service', got %d", len(defRes.Candidates()))
	}

	// 2. Exact qualified ID resolves uniquely
	exactA, err := nav.GoToDefinition("pkg/auth.Service")
	if err != nil || exactA.State() != xref.StateResolved || exactA.TargetSymbol() == nil {
		t.Errorf("expected unique resolution for pkg/auth.Service, got err=%v, res=%+v", err, exactA)
	}
}

func TestXRef_AdversarialCommentsAndFalsePositives(t *testing.T) {
	tempDir := t.TempDir()
	_, _, _, eng := setupTestPipeline(t)

	repoRoot := filepath.Join(tempDir, "comments_repo")
	_ = os.MkdirAll(repoRoot, 0755)

	src := `package main

// TargetFunction is mentioned in a comment
/* TargetFunction in block comment */
// Also mentioning AnotherFunction() in prose.

func ActualFunction() string {
	msg := "TargetFunction in a string literal"
	_ = msg
	return "ok"
}

func TargetFunction() {}
`
	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte(src), 0644)

	model, err := eng.AnalyzePath(repoRoot)
	if err != nil {
		t.Fatalf("AnalyzePath failed: %v", err)
	}

	// Ensure ActualFunction does NOT have a reference or call edge to TargetFunction merely because it appeared in comments/strings
	for _, ref := range model.References().AllReferences() {
		if ref.SourceSymbolID() == "ActualFunction" && ref.TargetSymbolID() == "TargetFunction" {
			t.Errorf("false positive reference detected from ActualFunction to TargetFunction via comment/string")
		}
	}
	for _, edge := range model.CallGraph().AllEdges() {
		if edge.CallerID() == "ActualFunction" && edge.CalleeID() == "TargetFunction" {
			t.Errorf("false positive call edge detected from ActualFunction to TargetFunction via comment/string")
		}
	}
}

func TestXRef_AdversarialParallelismAndDeterminism(t *testing.T) {
	tempDir := t.TempDir()
	disc, symEngine, depAnalyzer, _ := setupTestPipeline(t)

	repoRoot := filepath.Join(tempDir, "det_repo")
	_ = os.MkdirAll(repoRoot, 0755)

	for i := 1; i <= 8; i++ {
		src := "package main\nimport \"fmt\"\nfunc Func" + string(rune('A'+i-1)) + "() { fmt.Println(\"ok\") }\n"
		_ = os.WriteFile(filepath.Join(repoRoot, "file_"+string(rune('A'+i-1))+".go"), []byte(src), 0644)
	}

	discRes, _ := disc.DiscoverPath(repoRoot)
	symModel, _ := symEngine.Parse(discRes)
	depRes, _ := depAnalyzer.Analyze(discRes)

	eng1, _ := xref.NewWithWorkers(disc, symEngine, depAnalyzer, 1)
	m1, err := eng1.Analyze(discRes, symModel, depRes)
	if err != nil {
		t.Fatalf("1-worker run failed: %v", err)
	}

	eng8, _ := xref.NewWithWorkers(disc, symEngine, depAnalyzer, 8)
	m8, err := eng8.Analyze(discRes, symModel, depRes)
	if err != nil {
		t.Fatalf("8-worker run failed: %v", err)
	}

	if m1.References().TotalCount() != m8.References().TotalCount() {
		t.Errorf("count mismatch between 1 and 8 workers: %d vs %d", m1.References().TotalCount(), m8.References().TotalCount())
	}

	for i := range m1.References().AllReferences() {
		r1 := m1.References().AllReferences()[i]
		r8 := m8.References().AllReferences()[i]
		if r1.ID() != r8.ID() {
			t.Errorf("reference ordering mismatch at index %d: %s vs %s", i, r1.ID(), r8.ID())
		}
	}
}

func TestXRef_AdversarialMalformedGoSource(t *testing.T) {
	tempDir := t.TempDir()
	_, _, _, eng := setupTestPipeline(t)

	repoRoot := filepath.Join(tempDir, "malformed_repo")
	_ = os.MkdirAll(repoRoot, 0755)

	_ = os.WriteFile(filepath.Join(repoRoot, "valid.go"), []byte("package main\nfunc Valid() {}\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "malformed.go"), []byte("package main\nfunc Broken( {\n"), 0644)

	model, err := eng.AnalyzePath(repoRoot)
	if err != nil {
		t.Fatalf("AnalyzePath should not fail catastrophically on malformed source: %v", err)
	}

	if len(model.Diagnostics()) == 0 {
		t.Error("expected diagnostic for malformed.go syntax error")
	}

	if model.References() == nil {
		t.Error("expected valid reference database despite malformed file")
	}
}

func TestXRef_AdversarialTypesConstantsVariablesAndGenerics(t *testing.T) {
	tempDir := t.TempDir()
	_, _, _, eng := setupTestPipeline(t)

	repoRoot := filepath.Join(tempDir, "types_repo")
	_ = os.MkdirAll(repoRoot, 0755)

	src := `package main

const DefaultTimeout = 30
var GlobalStatus string = "active"

type ID int
type AliasID = ID

type Container[T any] struct {
	Value T
}

func (c *Container[T]) Get() T {
	return c.Value
}

func main() {
	_ = DefaultTimeout
	_ = GlobalStatus
	var id AliasID = 100
	_ = id
	box := &Container[string]{Value: "data"}
	_ = box.Get()
}
`
	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte(src), 0644)

	model, err := eng.AnalyzePath(repoRoot)
	if err != nil {
		t.Fatalf("AnalyzePath failed: %v", err)
	}

	refDB := model.References()
	if refDB.TotalCount() == 0 {
		t.Fatal("expected references for types, constants, variables, got 0")
	}

	var hasTypeRef, hasStructRef bool
	for _, r := range refDB.AllReferences() {
		if r.TargetSymbolID() == "AliasID" || r.TargetSymbolID() == "ID" {
			hasTypeRef = true
		}
		if strings.HasPrefix(r.TargetSymbolID(), "Container") {
			hasStructRef = true
		}
	}

	if !hasTypeRef {
		t.Error("expected reference targeting ID or AliasID")
	}
	if !hasStructRef {
		t.Error("expected struct reference targeting Container")
	}
}

func TestXRef_AdversarialPackageImpactAndReverseDependencies(t *testing.T) {
	tempDir := t.TempDir()
	_, _, _, eng := setupTestPipeline(t)

	repoRoot := filepath.Join(tempDir, "multi_pkg_repo")
	pkgCore := filepath.Join(repoRoot, "pkg", "core")
	pkgApp := filepath.Join(repoRoot, "pkg", "app")
	_ = os.MkdirAll(pkgCore, 0755)
	_ = os.MkdirAll(pkgApp, 0755)

	_ = os.WriteFile(filepath.Join(pkgCore, "config.go"), []byte("package core\ntype Config struct { Port int }\nfunc DefaultConfig() *Config { return &Config{Port: 8080} }\n"), 0644)
	_ = os.WriteFile(filepath.Join(pkgApp, "server.go"), []byte("package app\nimport \"pkg/core\"\nfunc Start() { cfg := core.DefaultConfig(); _ = cfg }\n"), 0644)

	model, err := eng.AnalyzePath(repoRoot)
	if err != nil {
		t.Fatalf("AnalyzePath failed: %v", err)
	}

	nav := model.Navigation()
	reverseDeps := nav.ReverseDependencyLookup("pkg/core")
	if len(reverseDeps) == 0 || reverseDeps[0] != "pkg/app" {
		t.Errorf("expected ReverseDependencyLookup('pkg/core') to return 'pkg/app', got: %v", reverseDeps)
	}

	impact := model.Impact()
	pkgImpact := impact.AnalyzePackageImpact("pkg/core")
	if len(pkgImpact.DownstreamPackages()) == 0 || pkgImpact.DownstreamPackages()[0] != "pkg/app" {
		t.Errorf("expected pkg/app downstream in AnalyzePackageImpact, got: %+v", pkgImpact)
	}
}
