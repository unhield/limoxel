package semantic

import (
	"sync"
	"testing"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
	"github.com/unhield/limoxel/internal/capabilities/repository/xref"
)

func TestSemanticModel_EntitiesAndGetters(t *testing.T) {
	pos := symbol.NewSourcePosition("pkg/foo.go", 10, 2, 100)
	doc := symbol.NewDocEntry("doc1", "sym1", symbol.DocKindFunction, "Test doc", "// Test doc", pos)

	// Task 1: Semantic Symbol
	sym := NewSemanticSymbol(
		"sym1",
		"CalculateTotal",
		symbol.SymbolKindFunction,
		"pkg/billing",
		"pkg/billing/calc.go",
		10,
		true,
		VisibilityPublic,
		"pkg/billing",
		"scope:pkg:billing",
		"type:func",
		"func(int) int",
		doc.Content(),
		[]string{"ref1"},
		[]string{"func:callee1"},
		[]string{"func:caller1"},
	)

	if sym.ID() != "sym1" || sym.Name() != "CalculateTotal" || !sym.IsExported() {
		t.Fatalf("Unexpected symbol properties: %+v", sym)
	}
	if sym.Visibility() != VisibilityPublic || sym.Ownership() != "pkg/billing" {
		t.Fatalf("Unexpected symbol visibility/ownership: %+v", sym)
	}
	if len(sym.References()) != 1 || len(sym.Calls()) != 1 || len(sym.CalledBy()) != 1 {
		t.Fatalf("Unexpected references or call slices: %+v", sym)
	}

	// Defensive copy check for symbol slices
	refs := sym.References()
	refs[0] = "mutated"
	if sym.References()[0] == "mutated" {
		t.Fatal("Symbol References() leaked internal slice")
	}

	// Task 1: Semantic Variable
	v := NewSemanticVariable(
		"var:count",
		"count",
		"pkg/billing",
		"pkg/billing/calc.go",
		ScopeLocal,
		"scope:local:1",
		"int",
		"type:int",
		false,
		VisibilityLocal,
		12,
	)
	if v.Name() != "count" || v.ScopeKind() != ScopeLocal || v.TypeExpression() != "int" {
		t.Fatalf("Unexpected variable properties: %+v", v)
	}

	// Task 1: Semantic Generic
	gen := NewSemanticGeneric(
		"gen:1",
		"sym1",
		[]string{"T", "K"},
		map[string]string{"T": "any", "K": "comparable"},
		map[string]string{"T": "string"},
		StateResolved,
	)
	if len(gen.TypeParameters()) != 2 || gen.Constraints()["T"] != "any" || gen.TypeArguments()["T"] != "string" {
		t.Fatalf("Unexpected generic properties: %+v", gen)
	}

	// Task 1: Semantic Function
	fn := NewSemanticFunction(
		"func:sym1",
		"CalculateTotal",
		"pkg/billing",
		"pkg/billing/calc.go",
		"",
		false,
		[]*SemanticVariable{v},
		[]string{"int"},
		true,
		VisibilityPublic,
		"func(int) int",
		"scope:func:1",
		[]string{"func:callee1"},
		[]string{"func:caller1"},
		gen,
	)
	if fn.Name() != "CalculateTotal" || len(fn.Parameters()) != 1 || len(fn.ReturnTypes()) != 1 {
		t.Fatalf("Unexpected function properties: %+v", fn)
	}

	// Task 1: Semantic Type
	typ := NewSemanticType(
		"type:Account",
		"Account",
		TypeCustom,
		"pkg/billing",
		"pkg/billing/types.go",
		"struct",
		false,
		"",
		true,
		[]*SemanticVariable{v},
		[]*SemanticFunction{fn},
		[]string{"type:Base"},
		[]string{"iface:Auditable"},
		gen,
		StateResolved,
	)
	if typ.Name() != "Account" || typ.Kind() != TypeCustom || len(typ.Methods()) != 1 {
		t.Fatalf("Unexpected type properties: %+v", typ)
	}

	// Task 1: Semantic Interface
	iface := NewSemanticInterface(
		"iface:Auditable",
		"Auditable",
		"pkg/billing",
		"pkg/billing/types.go",
		[]*SemanticFunction{fn},
		[]string{"iface:Stringer"},
		[]string{"type:Account"},
		true,
	)
	if iface.Name() != "Auditable" || len(iface.Methods()) != 1 || len(iface.Implementors()) != 1 {
		t.Fatalf("Unexpected interface properties: %+v", iface)
	}

	// Task 1: Semantic Package
	pkg := NewSemanticPackage(
		"billing",
		"pkg/billing",
		[]*SemanticSymbol{sym},
		[]*SemanticType{typ},
		[]*SemanticInterface{iface},
		[]*SemanticFunction{fn},
		[]*SemanticVariable{v},
		[]string{"pkg/auth"},
	)
	if pkg.Name() != "billing" || len(pkg.Symbols()) != 1 || len(pkg.Types()) != 1 {
		t.Fatalf("Unexpected package properties: %+v", pkg)
	}

	// Task 1: Semantic Repository
	repo := NewSemanticRepository(
		"myrepo",
		"/workspace/myrepo",
		[]*SemanticPackage{pkg},
		1, 1, 1, 1, 1,
		time.Now(),
	)
	if repo.Name() != "myrepo" || len(repo.Packages()) != 1 || repo.TotalSymbols() != 1 {
		t.Fatalf("Unexpected repository properties: %+v", repo)
	}

	// Task 1: Semantic Relationship
	rel := NewSemanticRelationship(
		"rel:1",
		RelSemanticOwnership,
		"pkg/billing",
		"sym1",
		"package declaration",
		"discovery",
		map[string]string{"weight": "1.0"},
	)
	if rel.Kind() != RelSemanticOwnership || rel.SourceID() != "pkg/billing" || rel.Metadata()["weight"] != "1.0" {
		t.Fatalf("Unexpected relationship properties: %+v", rel)
	}

	// Task 1: Semantic Model
	model := NewSemanticModel(
		repo,
		map[string]*SemanticSymbol{sym.ID(): sym},
		map[string]*SemanticType{typ.ID(): typ},
		map[string]*SemanticInterface{iface.ID(): iface},
		map[string]*SemanticFunction{fn.ID(): fn},
		map[string]*SemanticVariable{v.ID(): v},
		map[string]*SemanticGeneric{gen.ID(): gen},
		[]*SemanticRelationship{rel},
		nil,
		nil,
		time.Now(),
	)

	if model.SymbolByID("sym1") == nil || model.TypeByID("type:Account") == nil || model.InterfaceByID("iface:Auditable") == nil {
		t.Fatal("SemanticModel ID lookups failed")
	}
	if len(model.AllSymbols()) != 1 || len(model.AllTypes()) != 1 || len(model.AllInterfaces()) != 1 {
		t.Fatal("SemanticModel collection counts mismatch")
	}
}

func TestScopeResolution(t *testing.T) {
	// Task 2: Scope Resolution (Package, File, Local, Block)
	varLocal := NewSemanticVariable("var:x:local", "x", "pkg/core", "pkg/core/a.go", ScopeLocal, "scope:func:1", "int", "type:int", false, VisibilityLocal, 15)
	varBlock := NewSemanticVariable("var:x:block", "x", "pkg/core", "pkg/core/a.go", ScopeBlock, "scope:block:1", "string", "type:string", false, VisibilityLocal, 20)

	symPkg := NewSemanticSymbol("sym:x:pkg", "x", symbol.SymbolKindVariable, "pkg/core", "pkg/core/a.go", 5, true, VisibilityPublic, "pkg/core", "scope:pkg:core", "type:int", "var x int", "", nil, nil, nil)

	scopePkg := NewSemanticScope("scope:pkg:core", ScopePackage, "core", "pkg/core", "", "", nil, []string{symPkg.ID()}, nil, 1, 100)
	scopeFile := NewSemanticScope("scope:file:a", ScopeFile, "a.go", "pkg/core", "pkg/core/a.go", scopePkg.ID(), nil, nil, nil, 1, 100)
	scopeFunc := NewSemanticScope("scope:func:1", ScopeLocal, "MyFunc", "pkg/core", "pkg/core/a.go", scopeFile.ID(), nil, nil, []*SemanticVariable{varLocal}, 10, 50)
	scopeBlock := NewSemanticScope("scope:block:1", ScopeBlock, "if_block", "pkg/core", "pkg/core/a.go", scopeFunc.ID(), nil, nil, []*SemanticVariable{varBlock}, 18, 25)

	scopes := map[string]*SemanticScope{
		scopePkg.ID():   scopePkg,
		scopeFile.ID():  scopeFile,
		scopeFunc.ID():  scopeFunc,
		scopeBlock.ID(): scopeBlock,
	}
	syms := map[string]*SemanticSymbol{
		symPkg.ID(): symPkg,
	}

	resolver := NewScopeResolver(scopes, syms)

	// 1. Resolve from block scope (should find block-scoped 'x')
	resBlock := resolver.ResolveInScope(scopeBlock.ID(), "x")
	if resBlock.ResolutionState != StateResolved || resBlock.Variable == nil || resBlock.Variable.ID() != varBlock.ID() {
		t.Fatalf("Block scope resolution expected block var, got: %+v", resBlock)
	}

	// 2. Resolve from function scope (should find local 'x')
	resFunc := resolver.ResolveInScope(scopeFunc.ID(), "x")
	if resFunc.ResolutionState != StateResolved || resFunc.Variable == nil || resFunc.Variable.ID() != varLocal.ID() {
		t.Fatalf("Function scope resolution expected local var, got: %+v", resFunc)
	}

	// 3. Resolve from file scope (should find package symbol 'x')
	resFile := resolver.ResolveInScope(scopeFile.ID(), "x")
	if resFile.ResolutionState != StateResolved || resFile.Symbol == nil || resFile.Symbol.ID() != symPkg.ID() {
		t.Fatalf("File scope resolution expected package symbol, got: %+v", resFile)
	}

	// 4. Resolve non-existent name
	resNotFound := resolver.ResolveInScope(scopeBlock.ID(), "nonExistentName")
	if resNotFound.ResolutionState != StateUnresolved {
		t.Fatalf("Expected unresolved for non-existent name, got: %+v", resNotFound)
	}

	// 5. Find enclosing scope
	foundScope := resolver.FindEnclosingScope("pkg/core/a.go", 22)
	if foundScope == nil || foundScope.ID() != scopeBlock.ID() {
		t.Fatalf("FindEnclosingScope expected scopeBlock, got: %+v", foundScope)
	}
}

func TestTypeResolution(t *testing.T) {
	// Task 3: Type Resolution (Primitives, Custom, Interfaces, Generics, Aliases)
	fnM1 := NewSemanticFunction("func:Reader.Read", "Read", "pkg/io", "pkg/io/io.go", "Reader", false, nil, []string{"int", "error"}, true, VisibilityPublic, "func(p []byte) (n int, err error)", "", nil, nil, nil)

	ifaceReader := NewSemanticInterface("iface:Reader", "Reader", "pkg/io", "pkg/io/io.go", []*SemanticFunction{fnM1}, nil, nil, true)
	structBuffer := NewSemanticType("type:Buffer", "Buffer", TypeCustom, "pkg/io", "pkg/io/buf.go", "struct", false, "", true, nil, []*SemanticFunction{fnM1}, nil, nil, nil, StateResolved)
	aliasBuf := NewSemanticType("type:MyBuffer", "MyBuffer", TypeAlias, "pkg/io", "pkg/io/alias.go", "Buffer", true, "type:Buffer", true, nil, nil, nil, nil, nil, StateResolved)
	cyclicA := NewSemanticType("type:CyclicA", "CyclicA", TypeAlias, "pkg/io", "pkg/io/cyc.go", "CyclicB", true, "type:CyclicB", true, nil, nil, nil, nil, nil, StateResolved)
	cyclicB := NewSemanticType("type:CyclicB", "CyclicB", TypeAlias, "pkg/io", "pkg/io/cyc.go", "CyclicA", true, "type:CyclicA", true, nil, nil, nil, nil, nil, StateResolved)

	types := map[string]*SemanticType{
		structBuffer.ID(): structBuffer,
		aliasBuf.ID():     aliasBuf,
		cyclicA.ID():      cyclicA,
		cyclicB.ID():      cyclicB,
	}
	ifaces := map[string]*SemanticInterface{
		ifaceReader.ID(): ifaceReader,
	}

	resolver := NewTypeResolver(types, ifaces)

	// 1. Primitive types
	prims := []string{"int", "string", "bool", "float64", "byte", "error", "any", "comparable"}
	for _, p := range prims {
		res := resolver.ResolveType(p, "pkg/io")
		if res.ResolutionState != StateResolved || !res.IsPrimitive || res.TypeKind != TypePrimitive {
			t.Fatalf("Primitive resolution failed for %s: %+v", p, res)
		}
	}

	// 2. Pointer and Slice types
	resPtr := resolver.ResolveType("*Buffer", "pkg/io")
	if resPtr.ResolutionState != StateResolved || resPtr.TypeKind != TypePointer {
		t.Fatalf("Pointer resolution failed: %+v", resPtr)
	}

	resSlice := resolver.ResolveType("[]string", "pkg/io")
	if resSlice.ResolutionState != StateResolved || resSlice.TypeKind != TypeSlice {
		t.Fatalf("Slice resolution failed: %+v", resSlice)
	}

	// 3. Custom type
	resCustom := resolver.ResolveType("Buffer", "pkg/io")
	if resCustom.ResolutionState != StateResolved || resCustom.ResolvedType == nil || resCustom.ResolvedType.ID() != structBuffer.ID() {
		t.Fatalf("Custom type resolution failed: %+v", resCustom)
	}

	// 4. Alias type resolution
	resAlias := resolver.ResolveType("MyBuffer", "pkg/io")
	if resAlias.ResolutionState != StateResolved || !resAlias.IsAlias || resAlias.ResolvedType.ID() != structBuffer.ID() {
		t.Fatalf("Alias type resolution failed: %+v", resAlias)
	}

	// 5. Cyclic alias detection
	resCyclic := resolver.ResolveType("CyclicA", "pkg/io")
	if resCyclic.ResolutionState != StateInvalid {
		t.Fatalf("Expected StateInvalid for cyclic alias, got: %+v", resCyclic)
	}

	// 6. Interface satisfaction check
	if !resolver.CheckInterfaceSatisfaction(structBuffer.ID(), ifaceReader.ID()) {
		t.Fatal("Expected Buffer to satisfy Reader interface")
	}
}

func TestSymbolResolution(t *testing.T) {
	// Task 4: Symbol Resolution (Ownership, References, Visibility)
	symPub := NewSemanticSymbol("sym:pkgA.ExportedFunc", "ExportedFunc", symbol.SymbolKindFunction, "pkg/a", "pkg/a/a.go", 10, true, VisibilityPublic, "pkg/a", "scope:file:a", "type:func", "func()", "", nil, nil, nil)
	symPriv := NewSemanticSymbol("sym:pkgA.privateFunc", "privateFunc", symbol.SymbolKindFunction, "pkg/a", "pkg/a/a.go", 20, false, VisibilityPackagePrivate, "pkg/a", "scope:file:a", "type:func", "func()", "", nil, nil, nil)

	syms := map[string]*SemanticSymbol{
		symPub.ID():  symPub,
		symPriv.ID(): symPriv,
	}

	resolver := NewSymbolResolver(syms, nil)

	// 1. Direct and Qualified lookups
	resPub := resolver.ResolveSymbol("pkgA.ExportedFunc", nil)
	if resPub.ResolutionState != StateResolved || resPub.ResolvedSymbol.ID() != symPub.ID() {
		t.Fatalf("Symbol lookup failed for ExportedFunc: %+v", resPub)
	}

	// 2. Ownership
	if resolver.GetSymbolOwnership(symPub.ID()) != "pkg/a" {
		t.Fatalf("Unexpected symbol ownership: %s", resolver.GetSymbolOwnership(symPub.ID()))
	}

	// 3. Visibility checking
	// Public symbol is visible from outside package
	if !resolver.CheckVisibility(symPub.ID(), "pkg/b") {
		t.Fatal("Expected ExportedFunc to be visible from pkg/b")
	}

	// Private symbol is NOT visible from outside package
	if resolver.CheckVisibility(symPriv.ID(), "pkg/b") {
		t.Fatal("Expected privateFunc to NOT be visible from pkg/b")
	}

	// Private symbol IS visible from same package
	if !resolver.CheckVisibility(symPriv.ID(), "pkg/a") {
		t.Fatal("Expected privateFunc to be visible from pkg/a")
	}
}

func TestSemanticValidation(t *testing.T) {
	// Task 5: Semantic Validation (Missing symbols, Invalid types, Broken references, Duplicates)
	symA1 := NewSemanticSymbol("sym:dup1", "DuplicateName", symbol.SymbolKindFunction, "pkg/x", "pkg/x/x.go", 10, true, VisibilityPublic, "pkg/x", "scope:pkg:x", "type:func", "func()", "", nil, nil, nil)
	symA2 := NewSemanticSymbol("sym:dup2", "DuplicateName", symbol.SymbolKindFunction, "pkg/x", "pkg/x/x.go", 25, true, VisibilityPublic, "pkg/x", "scope:pkg:x", "type:func", "func()", "", nil, nil, nil)

	// Symbol referencing a missing symbol
	symBrokenRef := NewSemanticSymbol("sym:broken", "BrokenRefFunc", symbol.SymbolKindFunction, "pkg/x", "pkg/x/x.go", 30, true, VisibilityPublic, "pkg/x", "scope:pkg:x", "type:func", "func()", "", []string{"sym:nonExistentTarget"}, nil, nil)

	// Symbol referencing a private symbol from outside package
	symPrivTarget := NewSemanticSymbol("sym:pkgY.privateHelper", "privateHelper", symbol.SymbolKindFunction, "pkg/y", "pkg/y/y.go", 5, false, VisibilityPackagePrivate, "pkg/y", "scope:pkg:y", "type:func", "func()", "", nil, nil, nil)
	symBadCaller := NewSemanticSymbol("sym:pkgX.Caller", "Caller", symbol.SymbolKindFunction, "pkg/x", "pkg/x/x.go", 40, true, VisibilityPublic, "pkg/x", "scope:pkg:x", "type:func", "func()", "", []string{symPrivTarget.ID()}, nil, nil)

	scopePkgX := NewSemanticScope("scope:pkg:x", ScopePackage, "x", "pkg/x", "", "", nil, []string{symA1.ID(), symA2.ID(), symBrokenRef.ID(), symBadCaller.ID()}, nil, 1, 100)

	syms := map[string]*SemanticSymbol{
		symA1.ID():         symA1,
		symA2.ID():         symA2,
		symBrokenRef.ID():  symBrokenRef,
		symPrivTarget.ID(): symPrivTarget,
		symBadCaller.ID():  symBadCaller,
	}
	scopes := map[string]*SemanticScope{
		scopePkgX.ID(): scopePkgX,
	}

	scopeResolver := NewScopeResolver(scopes, syms)
	typeResolver := NewTypeResolver(nil, nil)
	symbolResolver := NewSymbolResolver(syms, scopeResolver)

	validator := NewSemanticValidator(typeResolver, symbolResolver)
	report := validator.Validate(syms, nil, nil, nil, nil, scopes)

	if report.IsValid() {
		t.Fatal("Expected report to be invalid due to duplicate definitions and invalid references")
	}

	if report.TotalErrors() == 0 {
		t.Fatalf("Expected validation errors, got: %+v", report)
	}

	findings := report.Findings()
	if len(findings) == 0 {
		t.Fatal("Expected non-empty findings list")
	}

	// Verify finding categories exist
	var foundDup, foundMissing, foundBadVis bool
	for _, f := range findings {
		switch f.Kind() {
		case FindingDuplicateDefinition:
			foundDup = true
		case FindingMissingSymbol:
			foundMissing = true
		case FindingInvalidReference:
			foundBadVis = true
		}
	}

	if !foundDup {
		t.Error("Expected FindingDuplicateDefinition in findings")
	}
	if !foundMissing {
		t.Error("Expected FindingMissingSymbol in findings")
	}
	if !foundBadVis {
		t.Error("Expected FindingInvalidReference in findings")
	}
}

func TestSemanticEngine_Analyze(t *testing.T) {
	// Full Pipeline Integration Test
	pos1 := symbol.NewSourcePosition("internal/service/user.go", 15, 1, 150)
	doc1 := symbol.NewDocEntry("doc:1", "sym:UserService", symbol.DocKindStruct, "UserService manages users", "// UserService manages users", pos1)
	symStruct := symbol.NewSymbol("sym:UserService", symbol.SymbolKindStruct, "UserService", "service", "internal/service", "internal/service/user.go", "", false, "type UserService struct", "struct", false, nil, []string{"repo string"}, pos1, doc1)

	pos2 := symbol.NewSourcePosition("internal/service/user.go", 30, 1, 300)
	doc2 := symbol.NewDocEntry("doc:2", "sym:GetUser", symbol.DocKindMethod, "GetUser retrieves user by ID", "// GetUser retrieves user by ID", pos2)
	symMethod := symbol.NewSymbol("sym:GetUser", symbol.SymbolKindMethod, "GetUser", "service", "internal/service", "internal/service/user.go", "UserService", false, "func (s *UserService) GetUser(id string) (string, error)", "func", false, nil, nil, pos2, doc2)

	pos3 := symbol.NewSourcePosition("internal/service/user.go", 50, 1, 500)
	symIface := symbol.NewSymbol("sym:UserReader", symbol.SymbolKindInterface, "UserReader", "service", "internal/service", "internal/service/user.go", "", false, "type UserReader interface", "interface", false, nil, []string{"GetUser(id string) (string, error)"}, pos3, nil)

	symDB := symbol.NewSymbolDatabase([]*symbol.Symbol{symStruct, symMethod, symIface})

	symRel := symbol.NewSymbolRelationship(symStruct.ID(), symMethod.ID(), symbol.RelMethodReceiver, "method receiver on UserService", pos2)

	ref := xref.NewReference(symMethod.ID(), symStruct.ID(), xref.RefStruct, "internal/service/user.go", pos2, xref.StateResolved, "receiver usage")
	edge := xref.NewCallEdge(symMethod.ID(), symMethod.ID(), xref.CallRecursiveDirect, "internal/service/user.go", pos2)
	refDB := xref.NewReferenceDatabase([]*xref.Reference{ref})
	cg := xref.NewCallGraph([]*xref.CallEdge{edge}, nil, nil, nil, nil, nil)
	xrefModel := xref.NewXRefModel("d:/limoxel", refDB, cg, nil, nil, nil, nil)

	engine := NewEngine()
	model, err := engine.Analyze(
		"limoxel",
		"d:/limoxel",
		symDB,
		[]*symbol.SymbolRelationship{symRel},
		xrefModel,
		nil,
		nil,
		nil,
		nil,
	)

	if err != nil {
		t.Fatalf("Engine.Analyze failed: %v", err)
	}

	if model == nil || model.Repository() == nil {
		t.Fatal("Expected non-nil SemanticModel and Repository")
	}

	if len(model.AllSymbols()) != 3 {
		t.Fatalf("Expected 3 symbols in model, got: %d", len(model.AllSymbols()))
	}

	if len(model.AllTypes()) == 0 {
		t.Fatal("Expected types in model")
	}

	if len(model.AllInterfaces()) != 1 {
		t.Fatalf("Expected 1 interface in model, got: %d", len(model.AllInterfaces()))
	}

	if len(model.AllFunctions()) != 1 {
		t.Fatalf("Expected 1 function in model, got: %d", len(model.AllFunctions()))
	}

	if len(model.AllRelationships()) == 0 {
		t.Fatal("Expected relationships in model")
	}

	if engine.ScopeResolver() == nil || engine.TypeResolver() == nil || engine.SymbolResolver() == nil || engine.Validator() == nil {
		t.Fatal("Expected all engine sub-resolvers to be initialized")
	}

	// Concurrency test
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_ = engine.Model().AllSymbols()
			_ = engine.Model().AllTypes()
			_ = engine.Model().AllRelationships()
			_ = engine.SymbolResolver().ResolveSymbol("sym:UserService", nil)
			_ = engine.TypeResolver().ResolveType("UserService", "internal/service")
		}(i)
	}
	wg.Wait()
}

func TestSemanticErrors(t *testing.T) {
	err := NewSemanticError(ErrCatInvalidInput, "ERR_TEST", "test error message")
	if err.Category() != ErrCatInvalidInput || err.Code() != "ERR_TEST" {
		t.Fatalf("Unexpected error properties: %v", err)
	}
	if !IsCategory(err, ErrCatInvalidInput) {
		t.Fatal("IsCategory returned false for matching category")
	}
	if IsCategory(err, ErrCatInternal) {
		t.Fatal("IsCategory returned true for non-matching category")
	}

	wrapped := WrapSemanticError(ErrCatValidation, "ERR_WRAP", "wrapper message", err)
	if wrapped.Unwrap() != err {
		t.Fatal("Unwrap failed to return inner error")
	}
}
