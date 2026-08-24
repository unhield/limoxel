package symbol_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
	langreg "github.com/unhield/limoxel/internal/language"
	"github.com/unhield/limoxel/internal/project"
	"github.com/unhield/limoxel/internal/repository"
	"github.com/unhield/limoxel/internal/workspace"
)

func setupTestLanguageRegistry(t *testing.T) *langreg.Registry {
	t.Helper()
	reg := langreg.NewRegistry()

	goLang, _ := langreg.New("go", "Go", []string{".go"}, nil, []string{"golang"})
	_ = reg.Register(goLang)

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

func TestEngine_New(t *testing.T) {
	t.Run("nil discoverer returns ErrNilDiscoverer", func(t *testing.T) {
		eng, err := symbol.New(nil)
		if err != symbol.ErrNilDiscoverer {
			t.Errorf("got %v, want ErrNilDiscoverer", err)
		}
		if eng != nil {
			t.Errorf("got %v, want nil", eng)
		}

		eng2, err2 := symbol.NewWithWorkers(nil, 4)
		if err2 != symbol.ErrNilDiscoverer || eng2 != nil {
			t.Errorf("expected ErrNilDiscoverer on NewWithWorkers")
		}
	})

	t.Run("valid discoverer returns operational engine", func(t *testing.T) {
		disc := setupTestDiscoverer(t)
		eng, err := symbol.New(disc)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if eng == nil || eng.Discoverer() != disc {
			t.Fatalf("invalid engine state")
		}

		eng2, err2 := symbol.NewWithWorkers(disc, 4)
		if err2 != nil || eng2 == nil {
			t.Fatalf("NewWithWorkers failed: %v", err2)
		}
	})
}

func TestEngine_InputValidation(t *testing.T) {
	disc := setupTestDiscoverer(t)
	eng, _ := symbol.New(disc)

	t.Run("nil engine receiver methods return safe errors", func(t *testing.T) {
		var nilEng *symbol.Engine
		if _, err := nilEng.Parse(nil); err != symbol.ErrNilEngine {
			t.Errorf("got %v, want ErrNilEngine", err)
		}
		if _, err := nilEng.ParseIncremental(nil, nil); err != symbol.ErrNilEngine {
			t.Errorf("got %v, want ErrNilEngine", err)
		}
		if _, err := nilEng.ParseRepository(nil); err != symbol.ErrNilEngine {
			t.Errorf("got %v, want ErrNilEngine", err)
		}
		if _, err := nilEng.ParsePath("some/path"); err != symbol.ErrNilEngine {
			t.Errorf("got %v, want ErrNilEngine", err)
		}
		if nilEng.Discoverer() != nil {
			t.Errorf("expected nil discoverer on nil engine")
		}
	})

	t.Run("nil discovery result returns ErrNilDiscoveryResult", func(t *testing.T) {
		_, err := eng.Parse(nil)
		if err != symbol.ErrNilDiscoveryResult {
			t.Errorf("got %v, want ErrNilDiscoveryResult", err)
		}
	})

	t.Run("nil repository returns ErrNilRepository", func(t *testing.T) {
		_, err := eng.ParseRepository(nil)
		if err != symbol.ErrNilRepository {
			t.Errorf("got %v, want ErrNilRepository", err)
		}
	})

	t.Run("empty path returns ErrPathEmpty", func(t *testing.T) {
		_, err := eng.ParsePath("   ")
		if err != symbol.ErrPathEmpty {
			t.Errorf("got %v, want ErrPathEmpty", err)
		}
	})
}

func TestEngine_ASTAndSymbolExtraction(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	eng, _ := symbol.New(disc)

	repoRoot := filepath.Join(tempDir, "symbol_repo")
	pkgDir := filepath.Join(repoRoot, "pkg", "service")
	_ = os.MkdirAll(pkgDir, 0755)

	src := `// Package service provides core service abstractions.
package service

import "fmt"

// Config represents service configuration.
type Config struct {
	Port int
	Host string
}

// Handler defines request handler behavior.
type Handler interface {
	Handle(req string) (string, error)
}

// Server implements Handler and embeds Config.
type Server struct {
	Config
	name string
}

// Handle processes incoming request.
func (s *Server) Handle(req string) (string, error) {
	return fmt.Sprintf("Handled: %s", req), nil
}

// NewServer constructs a new server instance.
func NewServer[T any, C comparable](cfg Config) *Server {
	return &Server{Config: cfg, name: "demo"}
}

// GlobalStatus is a package-level status variable.
var GlobalStatus = "ready"

// MaxRetries defines maximum retry attempts.
const MaxRetries = 3

// StringAlias aliases standard string.
type StringAlias = string

// CustomInt defines a new type based on int.
type CustomInt int
`
	_ = os.WriteFile(filepath.Join(pkgDir, "service.go"), []byte(src), 0644)

	model, err := eng.ParsePath(repoRoot)
	if err != nil {
		t.Fatalf("ParsePath failed: %v", err)
	}

	symDB := model.Symbols()
	if symDB == nil {
		t.Fatal("symbol database is nil")
	}

	// 1. Package symbol
	pkgSym := symDB.SymbolByID("pkg/service")
	if pkgSym == nil || pkgSym.Kind() != symbol.SymbolKindPackage || pkgSym.Name() != "service" {
		t.Errorf("package symbol invalid: %+v", pkgSym)
	}

	// 2. Struct symbols
	cfgSym := symDB.SymbolByID("pkg/service.Config")
	if cfgSym == nil || cfgSym.Kind() != symbol.SymbolKindStruct || len(cfgSym.Fields()) != 2 {
		t.Errorf("Config struct symbol invalid: %+v", cfgSym)
	}

	srvSym := symDB.SymbolByID("pkg/service.Server")
	if srvSym == nil || srvSym.Kind() != symbol.SymbolKindStruct {
		t.Errorf("Server struct symbol invalid: %+v", srvSym)
	}

	// 3. Interface symbol
	handlerSym := symDB.SymbolByID("pkg/service.Handler")
	if handlerSym == nil || handlerSym.Kind() != symbol.SymbolKindInterface || len(handlerSym.Fields()) != 1 {
		t.Errorf("Handler interface symbol invalid: %+v", handlerSym)
	}

	// 4. Function symbol with generics
	newSrvSym := symDB.SymbolByID("pkg/service.NewServer")
	if newSrvSym == nil || newSrvSym.Kind() != symbol.SymbolKindFunction || len(newSrvSym.Generics()) != 2 {
		t.Errorf("NewServer function symbol invalid: %+v", newSrvSym)
	}

	// 5. Method symbol with pointer receiver
	handleMethod := symDB.SymbolByID("pkg/service.(*Server).Handle")
	if handleMethod == nil || handleMethod.Kind() != symbol.SymbolKindMethod || !handleMethod.IsPointerReceiver() || handleMethod.ReceiverType() != "*Server" {
		t.Errorf("Handle method symbol invalid: %+v", handleMethod)
	}

	// 6. Variable and Constant symbols
	varSym := symDB.SymbolByID("pkg/service.GlobalStatus")
	if varSym == nil || varSym.Kind() != symbol.SymbolKindVariable {
		t.Errorf("GlobalStatus variable symbol invalid: %+v", varSym)
	}

	constSym := symDB.SymbolByID("pkg/service.MaxRetries")
	if constSym == nil || constSym.Kind() != symbol.SymbolKindConstant {
		t.Errorf("MaxRetries constant symbol invalid: %+v", constSym)
	}

	// 7. Alias vs Defined Type
	aliasSym := symDB.SymbolByID("pkg/service.StringAlias")
	if aliasSym == nil || aliasSym.Kind() != symbol.SymbolKindAlias || !aliasSym.IsAlias() {
		t.Errorf("StringAlias alias symbol invalid: %+v", aliasSym)
	}

	customIntSym := symDB.SymbolByID("pkg/service.CustomInt")
	if customIntSym == nil || customIntSym.Kind() != symbol.SymbolKindType || customIntSym.IsAlias() {
		t.Errorf("CustomInt type symbol invalid: %+v", customIntSym)
	}
}

func TestEngine_DocumentationAndComments(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	eng, _ := symbol.New(disc)

	repoRoot := filepath.Join(tempDir, "doc_repo")
	_ = os.MkdirAll(repoRoot, 0755)

	src := `// Package main implements application entrypoint.
package main

// TODO: Refactor initialize logic for v2
// FIXME: Handle memory leak in buffer pool

// App runs the main application loop.
func App() {}
`
	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte(src), 0644)

	model, err := eng.ParsePath(repoRoot)
	if err != nil {
		t.Fatalf("ParsePath failed: %v", err)
	}

	docDB := model.Docs()
	if docDB == nil {
		t.Fatal("documentation database is nil")
	}

	todos := docDB.TODOs()
	if len(todos) != 1 {
		t.Errorf("expected 1 TODO entry, got %d", len(todos))
	} else if todos[0].Kind() != symbol.DocKindTODO || todos[0].Position() == nil {
		t.Errorf("invalid TODO entry: %+v", todos[0])
	}

	fixmes := docDB.FIXMEs()
	if len(fixmes) != 1 {
		t.Errorf("expected 1 FIXME entry, got %d", len(fixmes))
	} else if fixmes[0].Kind() != symbol.DocKindFIXME || fixmes[0].Position() == nil {
		t.Errorf("invalid FIXME entry: %+v", fixmes[0])
	}

	allDocs := docDB.AllDocs()
	if len(allDocs) < 3 {
		t.Errorf("expected at least 3 doc entries (package, TODO, FIXME, func), got %d", len(allDocs))
	}
}

func TestEngine_SymbolRelationships(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	eng, _ := symbol.New(disc)

	repoRoot := filepath.Join(tempDir, "rel_repo")
	pkgDir := filepath.Join(repoRoot, "core")
	_ = os.MkdirAll(pkgDir, 0755)

	src := `package core

type Base struct {
	ID string
}

type Runner interface {
	Run() error
}

type Worker struct {
	Base
}

func (w *Worker) Run() error {
	return nil
}

type WorkerAlias = Worker
`
	_ = os.WriteFile(filepath.Join(pkgDir, "core.go"), []byte(src), 0644)

	model, err := eng.ParsePath(repoRoot)
	if err != nil {
		t.Fatalf("ParsePath failed: %v", err)
	}

	relGraph := model.Relationships()
	if relGraph == nil {
		t.Fatal("relationship graph is nil")
	}

	var (
		hasMethodReceiver bool
		hasEmbedding      bool
		hasInterfaceImpl  bool
		hasTypeAlias      bool
	)

	for _, r := range relGraph.AllRelationships() {
		switch r.Kind() {
		case symbol.RelMethodReceiver:
			hasMethodReceiver = true
		case symbol.RelStructEmbedding:
			hasEmbedding = true
		case symbol.RelInterfaceImplementation:
			hasInterfaceImpl = true
		case symbol.RelTypeAlias:
			hasTypeAlias = true
		}
	}

	if !hasMethodReceiver {
		t.Error("missing RelMethodReceiver relationship")
	}
	if !hasEmbedding {
		t.Error("missing RelStructEmbedding relationship")
	}
	if !hasInterfaceImpl {
		t.Error("missing RelInterfaceImplementation relationship (Worker implements Runner)")
	}
	if !hasTypeAlias {
		t.Error("missing RelTypeAlias relationship")
	}
}

func TestEngine_SyntaxErrorsAndRecovery(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	eng, _ := symbol.New(disc)

	repoRoot := filepath.Join(tempDir, "error_repo")
	_ = os.MkdirAll(repoRoot, 0755)

	// Valid Go file
	_ = os.WriteFile(filepath.Join(repoRoot, "valid.go"), []byte("package main\nfunc ValidFunc() {}\n"), 0644)

	// Malformed Go file (syntax error: missing brace)
	_ = os.WriteFile(filepath.Join(repoRoot, "invalid.go"), []byte("package main\nfunc BrokenFunc( {\n"), 0644)

	model, err := eng.ParsePath(repoRoot)
	if err != nil {
		t.Fatalf("ParsePath should not fail overall on syntax error: %v", err)
	}

	if len(model.Diagnostics()) == 0 {
		t.Error("expected syntax error diagnostic for invalid.go")
	}

	// Valid symbol must still be extracted
	validSym := model.Symbols().SymbolByID("ValidFunc")
	if validSym == nil {
		validSym = model.Symbols().SymbolByID(".ValidFunc")
	}
	if validSym == nil && len(model.Symbols().SymbolsByName("ValidFunc")) > 0 {
		validSym = model.Symbols().SymbolsByName("ValidFunc")[0]
	}
	if validSym == nil {
		t.Error("valid symbol ValidFunc was not extracted despite error in invalid.go")
	}
}

func TestEngine_IncrementalEquivalence(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	eng, _ := symbol.New(disc)

	repoRoot := filepath.Join(tempDir, "incr_repo")
	_ = os.MkdirAll(repoRoot, 0755)
	_ = os.WriteFile(filepath.Join(repoRoot, "fileA.go"), []byte("package main\nfunc A() {}\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "fileB.go"), []byte("package main\nfunc B() {}\n"), 0644)

	disc1, _ := disc.DiscoverPath(repoRoot)
	fullModel1, err := eng.Parse(disc1)
	if err != nil {
		t.Fatalf("full parse failed: %v", err)
	}

	// Incremental parse without file modifications
	incrModel1, err := eng.ParseIncremental(disc1, fullModel1)
	if err != nil {
		t.Fatalf("incremental parse failed: %v", err)
	}

	if incrModel1.Symbols().TotalCount() != fullModel1.Symbols().TotalCount() {
		t.Errorf("symbol count mismatch: %d vs %d", incrModel1.Symbols().TotalCount(), fullModel1.Symbols().TotalCount())
	}

	// Add fileC.go
	_ = os.WriteFile(filepath.Join(repoRoot, "fileC.go"), []byte("package main\nfunc C() {}\n"), 0644)

	disc2, _ := disc.DiscoverPath(repoRoot)
	fullModel2, _ := eng.Parse(disc2)
	incrModel2, _ := eng.ParseIncremental(disc2, incrModel1)

	if incrModel2.Symbols().TotalCount() != fullModel2.Symbols().TotalCount() {
		t.Errorf("symbol count mismatch after file addition: %d vs %d", incrModel2.Symbols().TotalCount(), fullModel2.Symbols().TotalCount())
	}
}

func TestEngine_ParallelismAndDeterminism(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)

	repoRoot := filepath.Join(tempDir, "det_repo")
	_ = os.MkdirAll(repoRoot, 0755)
	_ = os.WriteFile(filepath.Join(repoRoot, "z.go"), []byte("package main\nfunc Z() {}\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "a.go"), []byte("package main\nfunc A() {}\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "m.go"), []byte("package main\nfunc M() {}\n"), 0644)

	eng1, _ := symbol.NewWithWorkers(disc, 1)
	eng8, _ := symbol.NewWithWorkers(disc, 8)

	m1, err := eng1.ParsePath(repoRoot)
	if err != nil {
		t.Fatalf("eng1 failed: %v", err)
	}

	m8, err := eng8.ParsePath(repoRoot)
	if err != nil {
		t.Fatalf("eng8 failed: %v", err)
	}

	if m1.Symbols().TotalCount() != m8.Symbols().TotalCount() {
		t.Fatalf("count mismatch: %d vs %d", m1.Symbols().TotalCount(), m8.Symbols().TotalCount())
	}

	syms1 := m1.Symbols().AllSymbols()
	syms8 := m8.Symbols().AllSymbols()

	for i := range syms1 {
		if syms1[i].ID() != syms8[i].ID() {
			t.Errorf("ordering mismatch at index %d: %s vs %s", i, syms1[i].ID(), syms8[i].ID())
		}
	}
}

func TestEngine_ResultImmutability(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	eng, _ := symbol.New(disc)

	repoRoot := filepath.Join(tempDir, "immut_repo")
	_ = os.MkdirAll(repoRoot, 0755)
	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte("package main\nfunc Main() {}\n"), 0644)

	model, err := eng.ParsePath(repoRoot)
	if err != nil {
		t.Fatalf("ParsePath failed: %v", err)
	}

	// 1. Mutate AllSymbols slice
	syms := model.Symbols().AllSymbols()
	if len(syms) > 0 {
		syms[0] = nil
		if model.Symbols().AllSymbols()[0] == nil {
			t.Error("mutation of returned AllSymbols modified internal database state")
		}
	}

	// 2. Mutate AllDocs slice
	docs := model.Docs().AllDocs()
	if len(docs) > 0 {
		docs[0] = nil
		if model.Docs().AllDocs()[0] == nil {
			t.Error("mutation of returned AllDocs modified internal database state")
		}
	}

	// 3. Mutate AllRelationships slice
	rels := model.Relationships().AllRelationships()
	if len(rels) > 0 {
		rels[0] = nil
		if model.Relationships().AllRelationships()[0] == nil {
			t.Error("mutation of returned AllRelationships modified internal state")
		}
	}
}

func TestEngine_DomainRepository(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	eng, _ := symbol.New(disc)

	ws, _ := workspace.New("dom-ws", tempDir)
	proj, _ := project.New("dom-proj", ws, "my_app")
	repoRoot := filepath.Join(tempDir, "my_app", "my_repo")
	_ = os.MkdirAll(repoRoot, 0755)
	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte("package main\nfunc Run() {}\n"), 0644)

	repo, err := repository.New("my_repo", proj, repoRoot)
	if err != nil {
		t.Fatalf("repository.New failed: %v", err)
	}

	model, err := eng.ParseRepository(repo)
	if err != nil {
		t.Fatalf("ParseRepository failed: %v", err)
	}

	if model.RepositoryRoot() != filepath.ToSlash(filepath.Clean(repoRoot)) {
		t.Errorf("root mismatch: got %s, want %s", model.RepositoryRoot(), repoRoot)
	}
}

func TestEngine_FastLookups(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	eng, _ := symbol.New(disc)

	repoRoot := filepath.Join(tempDir, "lookup_repo")
	pkgDir := filepath.Join(repoRoot, "core")
	_ = os.MkdirAll(pkgDir, 0755)
	_ = os.WriteFile(filepath.Join(pkgDir, "core.go"), []byte("package core\nfunc Execute() {}\n"), 0644)

	model, err := eng.ParsePath(repoRoot)
	if err != nil {
		t.Fatalf("ParsePath failed: %v", err)
	}

	symDB := model.Symbols()
	byPkg := symDB.SymbolsByPackage("core")
	if len(byPkg) < 2 { // package symbol + Execute function
		t.Errorf("SymbolsByPackage failed: %+v", byPkg)
	}

	byFile := symDB.SymbolsByFile("core/core.go")
	if len(byFile) < 2 {
		t.Errorf("SymbolsByFile failed: %+v", byFile)
	}

	byKind := symDB.SymbolsByKind(symbol.SymbolKindFunction)
	if len(byKind) != 1 || byKind[0].Name() != "Execute" {
		t.Errorf("SymbolsByKind failed: %+v", byKind)
	}

	byName := symDB.SymbolsByName("Execute")
	if len(byName) != 1 {
		t.Errorf("SymbolsByName failed: %+v", byName)
	}
}

func TestModels_MethodsAndNilSafety(t *testing.T) {
	// SourcePosition
	sp := symbol.NewSourcePosition("main.go", 10, 5, 120)
	if sp.File() != "main.go" || sp.Line() != 10 || sp.Column() != 5 || sp.Offset() != 120 || sp.String() != "main.go:10:5" {
		t.Errorf("SourcePosition mismatch: %+v", sp)
	}
	var nilSP *symbol.SourcePosition
	if nilSP.File() != "" || nilSP.Line() != 0 || nilSP.Column() != 0 || nilSP.Offset() != 0 || nilSP.String() != "" {
		t.Error("nil SourcePosition should return zero values")
	}

	// DocEntry
	de := symbol.NewDocEntry("doc1", "sym1", symbol.DocKindFunction, "content", "raw", sp)
	if de.ID() != "doc1" || de.TargetSymbolID() != "sym1" || de.Kind() != symbol.DocKindFunction || de.Content() != "content" || de.RawText() != "raw" || de.Position() != sp {
		t.Errorf("DocEntry mismatch: %+v", de)
	}
	var nilDE *symbol.DocEntry
	if nilDE.ID() != "" || nilDE.TargetSymbolID() != "" || nilDE.Kind() != symbol.DocKindGeneral || nilDE.Content() != "" || nilDE.Position() != nil {
		t.Error("nil DocEntry should return zero values")
	}

	// Symbol
	sym := symbol.NewSymbol("id1", symbol.SymbolKindStruct, "MyStruct", "mypkg", "pkg/mypkg", "pkg/mypkg/a.go", "", false, "struct{}", "struct{}", false, []string{"T"}, []string{"ID: string"}, sp, de)
	if sym.ID() != "id1" || sym.Kind() != symbol.SymbolKindStruct || sym.Name() != "MyStruct" || !sym.IsExported() || len(sym.Generics()) != 1 || len(sym.Fields()) != 1 || sym.Doc() != de || sym.String() == "" {
		t.Errorf("Symbol mismatch: %+v", sym)
	}
	var nilSym *symbol.Symbol
	if nilSym.ID() != "" || nilSym.Kind() != symbol.SymbolKindUnknown || nilSym.IsExported() || nilSym.Generics() != nil || nilSym.Fields() != nil || nilSym.String() != "" {
		t.Error("nil Symbol should return zero values")
	}

	// SymbolRelationship
	rel := symbol.NewSymbolRelationship("src", "tgt", symbol.RelMethodReceiver, "ev", sp)
	if rel.SourceID() != "src" || rel.TargetID() != "tgt" || rel.Kind() != symbol.RelMethodReceiver || rel.Evidence() != "ev" || rel.Position() != sp {
		t.Errorf("SymbolRelationship mismatch: %+v", rel)
	}
	var nilRel *symbol.SymbolRelationship
	if nilRel.SourceID() != "" || nilRel.Kind() != symbol.RelUnknown || nilRel.Position() != nil {
		t.Error("nil SymbolRelationship should return zero values")
	}

	// SymbolModel
	var nilModel *symbol.SymbolModel
	if nilModel.RepositoryRoot() != "" || nilModel.Symbols() != nil || nilModel.Docs() != nil || nilModel.Relationships() != nil || nilModel.Diagnostics() != nil || nilModel.String() != "" {
		t.Error("nil SymbolModel should return zero values")
	}
}

func TestEngine_AdversarialComplexGenericsAndConstraints(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	eng, _ := symbol.New(disc)

	repoRoot := filepath.Join(tempDir, "gen_repo")
	_ = os.MkdirAll(repoRoot, 0755)

	src := `package main

type Numeric interface {
	~int | ~int64 | ~float64
}

type Matrix[T Numeric] struct {
	data [][]T
}

func Map[T any, R any](items []T, fn func(T) R) []R {
	res := make([]R, len(items))
	for i, v := range items {
		res[i] = fn(v)
	}
	return res
}
`
	_ = os.WriteFile(filepath.Join(repoRoot, "matrix.go"), []byte(src), 0644)

	model, err := eng.ParsePath(repoRoot)
	if err != nil {
		t.Fatalf("ParsePath failed: %v", err)
	}

	symDB := model.Symbols()
	matrixSym := symDB.SymbolByID("Matrix")
	if matrixSym == nil && len(symDB.SymbolsByName("Matrix")) > 0 {
		matrixSym = symDB.SymbolsByName("Matrix")[0]
	}
	if matrixSym == nil || len(matrixSym.Generics()) != 1 {
		t.Errorf("Matrix generic symbol missing or invalid: %+v", matrixSym)
	}

	mapSym := symDB.SymbolByID("Map")
	if mapSym == nil && len(symDB.SymbolsByName("Map")) > 0 {
		mapSym = symDB.SymbolsByName("Map")[0]
	}
	if mapSym == nil || len(mapSym.Generics()) != 2 {
		t.Errorf("Map generic function missing or invalid: %+v", mapSym)
	}
}

func TestEngine_AdversarialNestedPackagesAndDuplicateSymbolNames(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	eng, _ := symbol.New(disc)

	repoRoot := filepath.Join(tempDir, "dup_repo")
	pkgA := filepath.Join(repoRoot, "pkg", "auth")
	pkgB := filepath.Join(repoRoot, "pkg", "storage")
	_ = os.MkdirAll(pkgA, 0755)
	_ = os.MkdirAll(pkgB, 0755)

	_ = os.WriteFile(filepath.Join(pkgA, "service.go"), []byte("package auth\ntype Service struct{}\nfunc (s *Service) Start() {}\n"), 0644)
	_ = os.WriteFile(filepath.Join(pkgB, "service.go"), []byte("package storage\ntype Service struct{}\nfunc (s *Service) Start() {}\n"), 0644)

	model, err := eng.ParsePath(repoRoot)
	if err != nil {
		t.Fatalf("ParsePath failed: %v", err)
	}

	symDB := model.Symbols()
	authService := symDB.SymbolByID("pkg/auth.Service")
	storageService := symDB.SymbolByID("pkg/storage.Service")

	if authService == nil || storageService == nil {
		t.Fatalf("both Service symbols must be uniquely indexed: auth=%+v, storage=%+v", authService, storageService)
	}
	if authService.ID() == storageService.ID() {
		t.Errorf("symbol IDs must not collide: %s vs %s", authService.ID(), storageService.ID())
	}

	authStart := symDB.SymbolByID("pkg/auth.(*Service).Start")
	storageStart := symDB.SymbolByID("pkg/storage.(*Service).Start")

	if authStart == nil || storageStart == nil {
		t.Fatalf("both Start methods must be uniquely indexed: auth=%+v, storage=%+v", authStart, storageStart)
	}
	if authStart.ID() == storageStart.ID() {
		t.Errorf("method symbol IDs must not collide")
	}
}

func TestEngine_AdversarialRootPackageReceiverRelationships(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	eng, _ := symbol.New(disc)

	repoRoot := filepath.Join(tempDir, "root_repo")
	_ = os.MkdirAll(repoRoot, 0755)

	src := `package main

type Runner interface {
	Run() error
}

type Server struct {
	Name string
}

func (s *Server) Run() error {
	return nil
}
`
	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte(src), 0644)

	model, err := eng.ParsePath(repoRoot)
	if err != nil {
		t.Fatalf("ParsePath failed: %v", err)
	}

	rels := model.Relationships().AllRelationships()

	var (
		hasMethodReceiver bool
		hasInterfaceImpl  bool
	)

	for _, r := range rels {
		if r.Kind() == symbol.RelMethodReceiver && r.TargetID() == "Server" {
			hasMethodReceiver = true
		}
		if r.Kind() == symbol.RelInterfaceImplementation && r.SourceID() == "Server" && r.TargetID() == "Runner" {
			hasInterfaceImpl = true
		}
	}

	if !hasMethodReceiver {
		t.Errorf("expected method receiver targeting 'Server', got relationships: %v", rels)
	}
	if !hasInterfaceImpl {
		t.Errorf("expected interface implementation 'Server' -> 'Runner', got relationships: %v", rels)
	}
}

func TestEngine_AdversarialIncrementalModificationReparsing(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	eng, _ := symbol.New(disc)

	repoRoot := filepath.Join(tempDir, "incr_mod_repo")
	_ = os.MkdirAll(repoRoot, 0755)

	fileAPath := filepath.Join(repoRoot, "fileA.go")
	_ = os.WriteFile(fileAPath, []byte("package main\nfunc OldFunc() {}\n"), 0644)

	disc1, _ := disc.DiscoverPath(repoRoot)
	m1, err := eng.Parse(disc1)
	if err != nil {
		t.Fatalf("run 1 failed: %v", err)
	}

	if m1.Symbols().SymbolByID("OldFunc") == nil {
		t.Fatal("OldFunc missing in initial parse")
	}

	// Modify fileA.go on disk
	_ = os.WriteFile(fileAPath, []byte("package main\nfunc NewFunc() {}\n"), 0644)

	disc2, _ := disc.DiscoverPath(repoRoot)
	mIncr, err := eng.ParseIncremental(disc2, m1)
	if err != nil {
		t.Fatalf("incremental parse failed: %v", err)
	}
	mFull, err := eng.Parse(disc2)
	if err != nil {
		t.Fatalf("full parse failed: %v", err)
	}

	if mIncr.Symbols().SymbolByID("OldFunc") != nil {
		t.Errorf("stale symbol OldFunc must not survive incremental parse of modified file")
	}
	if mIncr.Symbols().SymbolByID("NewFunc") == nil {
		t.Errorf("new symbol NewFunc must be indexed during incremental parse of modified file")
	}
	if mIncr.Symbols().TotalCount() != mFull.Symbols().TotalCount() {
		t.Errorf("symbol count mismatch: incremental=%d vs full=%d", mIncr.Symbols().TotalCount(), mFull.Symbols().TotalCount())
	}
}

func TestEngine_AdversarialInterfaceMethodWithFuncSubstring(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	eng, _ := symbol.New(disc)

	repoRoot := filepath.Join(tempDir, "func_substring_repo")
	_ = os.MkdirAll(repoRoot, 0755)

	src := `package main

type Processor interface {
	Function() error
	Defunction() error
	GetFunc() bool
	Func() int
}

type CustomProcessor struct{}

func (cp *CustomProcessor) Function() error {
	return nil
}

func (cp *CustomProcessor) Defunction() error {
	return nil
}

func (cp *CustomProcessor) GetFunc() bool {
	return true
}

func (cp *CustomProcessor) Func() int {
	return 42
}
`
	_ = os.WriteFile(filepath.Join(repoRoot, "processor.go"), []byte(src), 0644)

	model, err := eng.ParsePath(repoRoot)
	if err != nil {
		t.Fatalf("ParsePath failed: %v", err)
	}

	procSym := model.Symbols().SymbolByID("Processor")
	if procSym == nil || len(procSym.Fields()) != 4 {
		t.Fatalf("Processor interface symbol invalid: %+v", procSym)
	}

	var hasInterfaceImpl bool
	for _, r := range model.Relationships().AllRelationships() {
		if r.Kind() == symbol.RelInterfaceImplementation && r.SourceID() == "CustomProcessor" && r.TargetID() == "Processor" {
			hasInterfaceImpl = true
		}
	}

	if !hasInterfaceImpl {
		t.Error("expected RelInterfaceImplementation 'CustomProcessor' -> 'Processor' despite methods containing 'func' substring")
	}
}

func TestEngine_AdversarialMethodSubstringNonMatching(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	eng, _ := symbol.New(disc)

	repoRoot := filepath.Join(tempDir, "non_matching_repo")
	_ = os.MkdirAll(repoRoot, 0755)

	src := `package main

type TargetInterface interface {
	Func2()
	Get()
	Run()
	Read()
}

type PartialStruct struct{}

func (ps *PartialStruct) Func() {}
func (ps *PartialStruct) GetFunc() {}
func (ps *PartialStruct) RunNow() {}
func (ps *PartialStruct) ReadAll() {}
`
	_ = os.WriteFile(filepath.Join(repoRoot, "types.go"), []byte(src), 0644)

	model, err := eng.ParsePath(repoRoot)
	if err != nil {
		t.Fatalf("ParsePath failed: %v", err)
	}

	for _, r := range model.Relationships().AllRelationships() {
		if r.Kind() == symbol.RelInterfaceImplementation && r.SourceID() == "PartialStruct" && r.TargetID() == "TargetInterface" {
			t.Errorf("PartialStruct must NOT be inferred to implement TargetInterface based on substring methods")
		}
	}
}

func TestEngine_AdversarialCompleteIncrementalEquivalence(t *testing.T) {
	tempDir := t.TempDir()
	disc := setupTestDiscoverer(t)
	eng, _ := symbol.New(disc)

	repoRoot := filepath.Join(tempDir, "full_incr_repo")
	_ = os.MkdirAll(repoRoot, 0755)

	f1Path := filepath.Join(repoRoot, "f1.go")
	f2Path := filepath.Join(repoRoot, "f2.go")

	_ = os.WriteFile(f1Path, []byte("package main\nfunc Alpha() int { return 1 }\n"), 0644)
	_ = os.WriteFile(f2Path, []byte("package main\nfunc Beta() int { return 2 }\n"), 0644)

	disc1, _ := disc.DiscoverPath(repoRoot)
	m1, _ := eng.Parse(disc1)

	// Modify f1.go while keeping exact same length (37 bytes)
	_ = os.WriteFile(f1Path, []byte("package main\nfunc Gamma() int { return 3 }\n"), 0644)

	disc2, _ := disc.DiscoverPath(repoRoot)
	mIncr, _ := eng.ParseIncremental(disc2, m1)
	mFull, _ := eng.Parse(disc2)

	// 1. Verify complete symbol set equality
	if mIncr.Symbols().TotalCount() != mFull.Symbols().TotalCount() {
		t.Fatalf("symbol count mismatch: incr=%d vs full=%d", mIncr.Symbols().TotalCount(), mFull.Symbols().TotalCount())
	}
	for i := range mFull.Symbols().AllSymbols() {
		sFull := mFull.Symbols().AllSymbols()[i]
		sIncr := mIncr.Symbols().AllSymbols()[i]
		if sFull.ID() != sIncr.ID() || sFull.Kind() != sIncr.Kind() || sFull.Signature() != sIncr.Signature() {
			t.Errorf("symbol[%d] mismatch: %+v vs %+v", i, sFull, sIncr)
		}
	}

	// 2. Verify complete relationships equality
	if mIncr.Relationships().TotalCount() != mFull.Relationships().TotalCount() {
		t.Fatalf("relationship count mismatch: incr=%d vs full=%d", mIncr.Relationships().TotalCount(), mFull.Relationships().TotalCount())
	}
	for i := range mFull.Relationships().AllRelationships() {
		rFull := mFull.Relationships().AllRelationships()[i]
		rIncr := mIncr.Relationships().AllRelationships()[i]
		if rFull.SourceID() != rIncr.SourceID() || rFull.TargetID() != rIncr.TargetID() || rFull.Kind() != rIncr.Kind() {
			t.Errorf("relationship[%d] mismatch: %+v vs %+v", i, rFull, rIncr)
		}
	}
}
