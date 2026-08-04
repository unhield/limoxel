package language_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/unhield/limoxel/internal/language"
)

func TestLanguageDescriptorConstructorAndValidation(t *testing.T) {
	t.Run("valid descriptor creation", func(t *testing.T) {
		exts := []string{"go", ".Go"}
		fns := []string{"Makefile", "makefile"}
		aliases := []string{"Golang", "golang"}

		lang, err := language.New("Go-Lang", "Go Programming Language", exts, fns, aliases)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if lang.ID() != "go-lang" {
			t.Errorf("got ID %q, want go-lang", lang.ID())
		}
		if lang.Name() != "Go Programming Language" {
			t.Errorf("got Name %q, want Go Programming Language", lang.Name())
		}

		// Verify extension dot normalization and deduplication
		if len(lang.Extensions()) != 1 || lang.Extensions()[0] != ".go" {
			t.Errorf("got extensions %v, want [.go]", lang.Extensions())
		}

		// Verify filename deduplication (case-insensitive deduplication, preserving first clean case)
		if len(lang.Filenames()) != 1 || lang.Filenames()[0] != "Makefile" {
			t.Errorf("got filenames %v, want [Makefile]", lang.Filenames())
		}

		// Verify alias deduplication
		if len(lang.Aliases()) != 1 || lang.Aliases()[0] != "golang" {
			t.Errorf("got aliases %v, want [golang]", lang.Aliases())
		}

		if lang.String() != "Language<go-lang>(Go Programming Language)" {
			t.Errorf("unexpected String(): %q", lang.String())
		}
	})

	t.Run("invalid ID errors", func(t *testing.T) {
		_, err := language.New("   ", "Go", nil, nil, nil)
		if !errors.Is(err, language.ErrInvalidID) {
			t.Errorf("got %v, want ErrInvalidID", err)
		}

		_, err = language.New("go lang", "Go", nil, nil, nil)
		if err == nil || !errors.Is(err, language.ErrInvalidID) {
			t.Errorf("got %v, want ErrInvalidID for spaces", err)
		}
	})

	t.Run("invalid Name error", func(t *testing.T) {
		_, err := language.New("go", "   ", nil, nil, nil)
		if !errors.Is(err, language.ErrInvalidName) {
			t.Errorf("got %v, want ErrInvalidName", err)
		}
	})
}

func TestLanguageImmutabilityAndNilSafety(t *testing.T) {
	exts := []string{".go"}
	lang, _ := language.New("go", "Go", exts, nil, nil)

	// Mutate input slice
	exts[0] = ".mutated"
	if lang.Extensions()[0] != ".go" {
		t.Error("input slice mutation leaked into Language descriptor")
	}

	// Mutate returned slice
	retExts := lang.Extensions()
	retExts[0] = ".mutated"
	if lang.Extensions()[0] != ".go" {
		t.Error("returned slice mutation leaked into Language descriptor")
	}

	// Nil receiver safety
	var nilLang *language.Language
	if nilLang.ID() != "" {
		t.Error("nil ID should be empty")
	}
	if nilLang.Name() != "" {
		t.Error("nil Name should be empty")
	}
	if nilLang.Extensions() != nil {
		t.Error("nil Extensions should be nil")
	}
	if nilLang.Filenames() != nil {
		t.Error("nil Filenames should be nil")
	}
	if nilLang.Aliases() != nil {
		t.Error("nil Aliases should be nil")
	}
	if nilLang.String() != "" {
		t.Error("nil String should be empty")
	}
}

func TestRegistryRegistrationAndLookups(t *testing.T) {
	reg := language.NewRegistry()
	if reg.State() != language.StateCreated {
		t.Errorf("got state %v, want StateCreated", reg.State())
	}

	goLang, _ := language.New("go", "Go", []string{".go"}, []string{"go.mod"}, []string{"golang"})
	pyLang, _ := language.New("python", "Python", []string{".py"}, nil, []string{"py"})

	if err := reg.Register(goLang); err != nil {
		t.Fatalf("Register goLang failed: %v", err)
	}
	if err := reg.Register(pyLang); err != nil {
		t.Fatalf("Register pyLang failed: %v", err)
	}

	if reg.Count() != 2 {
		t.Errorf("got count %d, want 2", reg.Count())
	}
	if !reg.Has("go") || !reg.Has("python") {
		t.Error("expected Has to return true for go and python")
	}

	// 1. Get by ID
	g, err := reg.Get("GO")
	if err != nil || g.ID() != "go" {
		t.Errorf("Get(GO) got %v, %v", g, err)
	}

	// 2. GetByExtension
	gExt, err := reg.GetByExtension("go")
	if err != nil || gExt.ID() != "go" {
		t.Errorf("GetByExtension(go) got %v, %v", gExt, err)
	}

	// 3. GetByAlias
	gAlias, err := reg.GetByAlias("GOLANG")
	if err != nil || gAlias.ID() != "go" {
		t.Errorf("GetByAlias(GOLANG) got %v, %v", gAlias, err)
	}

	// 4. Duplicate registration rejection
	err = reg.Register(goLang)
	if !errors.Is(err, language.ErrDuplicateLanguage) {
		t.Errorf("got %v, want ErrDuplicateLanguage", err)
	}

	// 5. Unknown lookups
	if _, err := reg.Get("java"); !errors.Is(err, language.ErrLanguageNotFound) {
		t.Errorf("got %v, want ErrLanguageNotFound", err)
	}
	if _, err := reg.GetByExtension(".java"); !errors.Is(err, language.ErrLanguageNotFound) {
		t.Errorf("got %v, want ErrLanguageNotFound", err)
	}
	if _, err := reg.GetByAlias("java"); !errors.Is(err, language.ErrLanguageNotFound) {
		t.Errorf("got %v, want ErrLanguageNotFound", err)
	}

	// 6. List deterministic ordering
	list := reg.List()
	if len(list) != 2 || list[0].ID() != "go" || list[1].ID() != "python" {
		t.Errorf("List() ordering mismatch: %v", list)
	}
}

func TestDiscoverySubsystem(t *testing.T) {
	reg := language.NewRegistry()
	makeFileLang, _ := language.New("makefile", "Makefile", []string{".mk"}, []string{"Makefile", "GNUmakefile"}, []string{"make"})
	goLang, _ := language.New("go", "Go", []string{".go"}, nil, nil)

	_ = reg.Register(makeFileLang)
	_ = reg.Register(goLang)

	// DiscoverByFilename: exact filename match
	lang, err := reg.DiscoverByFilename("/path/to/Makefile")
	if err != nil || lang.ID() != "makefile" {
		t.Errorf("DiscoverByFilename(Makefile) got %v, %v", lang, err)
	}

	// DiscoverByFilename: extension fallback match
	lang, err = reg.DiscoverByFilename("/path/to/main.go")
	if err != nil || lang.ID() != "go" {
		t.Errorf("DiscoverByFilename(main.go) got %v, %v", lang, err)
	}

	// DiscoverByFilename: unknown
	_, err = reg.DiscoverByFilename("unknown.xyz")
	if !errors.Is(err, language.ErrLanguageNotFound) {
		t.Errorf("got %v, want ErrLanguageNotFound", err)
	}

	// DiscoverByExtension & DiscoverByAlias delegation
	langExt, err := reg.DiscoverByExtension("mk")
	if err != nil || langExt.ID() != "makefile" {
		t.Errorf("DiscoverByExtension got %v, %v", langExt, err)
	}

	langAlias, err := reg.DiscoverByAlias("make")
	if err != nil || langAlias.ID() != "makefile" {
		t.Errorf("DiscoverByAlias got %v, %v", langAlias, err)
	}
}

func TestRegistryLifecycleAndFreeze(t *testing.T) {
	reg := language.NewRegistry()
	goLang, _ := language.New("go", "Go", []string{".go"}, nil, nil)
	pyLang, _ := language.New("python", "Python", []string{".py"}, nil, nil)

	_ = reg.Register(goLang)

	// Freeze registry
	if err := reg.Freeze(); err != nil {
		t.Fatalf("Freeze failed: %v", err)
	}
	if reg.State() != language.StateOperational {
		t.Errorf("got state %v, want StateOperational", reg.State())
	}

	// Freeze is idempotent
	if err := reg.Freeze(); err != nil {
		t.Errorf("idempotent Freeze failed: %v", err)
	}

	// Cannot register in frozen state
	err := reg.Register(pyLang)
	if !errors.Is(err, language.ErrRegistryFrozen) {
		t.Errorf("got %v, want ErrRegistryFrozen", err)
	}

	// Reads still work in frozen state
	g, err := reg.Get("go")
	if err != nil || g.ID() != "go" {
		t.Errorf("Get failed on frozen registry: %v", err)
	}

	// Close registry
	if err := reg.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if reg.State() != language.StateTerminated {
		t.Errorf("got state %v, want StateTerminated", reg.State())
	}

	// Close is idempotent
	if err := reg.Close(); err != nil {
		t.Errorf("idempotent Close failed: %v", err)
	}

	// All calls on terminated registry return ErrRegistryTerminated or zero values
	if _, err := reg.Get("go"); !errors.Is(err, language.ErrRegistryTerminated) {
		t.Errorf("got %v, want ErrRegistryTerminated", err)
	}
	if _, err := reg.GetByExtension(".go"); !errors.Is(err, language.ErrRegistryTerminated) {
		t.Errorf("got %v, want ErrRegistryTerminated", err)
	}
	if _, err := reg.GetByAlias("go"); !errors.Is(err, language.ErrRegistryTerminated) {
		t.Errorf("got %v, want ErrRegistryTerminated", err)
	}
	if _, err := reg.DiscoverByFilename("main.go"); !errors.Is(err, language.ErrRegistryTerminated) {
		t.Errorf("got %v, want ErrRegistryTerminated", err)
	}
	if reg.Has("go") {
		t.Error("expected Has to return false on terminated registry")
	}
	if reg.Count() != 0 {
		t.Error("expected Count 0 on terminated registry")
	}
	if reg.List() != nil {
		t.Error("expected nil List on terminated registry")
	}
}

func TestNilRegistrySafety(t *testing.T) {
	var reg *language.Registry
	goLang, _ := language.New("go", "Go", []string{".go"}, nil, nil)

	if reg.State() != language.StateTerminated {
		t.Errorf("got state %v, want StateTerminated for nil registry", reg.State())
	}
	if err := reg.Freeze(); !errors.Is(err, language.ErrNilRegistry) {
		t.Errorf("got %v, want ErrNilRegistry", err)
	}
	if err := reg.Close(); !errors.Is(err, language.ErrNilRegistry) {
		t.Errorf("got %v, want ErrNilRegistry", err)
	}
	if err := reg.Register(goLang); !errors.Is(err, language.ErrNilRegistry) {
		t.Errorf("got %v, want ErrNilRegistry", err)
	}
	if _, err := reg.Get("go"); !errors.Is(err, language.ErrNilRegistry) {
		t.Errorf("got %v, want ErrNilRegistry", err)
	}
	if _, err := reg.GetByExtension(".go"); !errors.Is(err, language.ErrNilRegistry) {
		t.Errorf("got %v, want ErrNilRegistry", err)
	}
	if _, err := reg.GetByAlias("go"); !errors.Is(err, language.ErrNilRegistry) {
		t.Errorf("got %v, want ErrNilRegistry", err)
	}
	if _, err := reg.DiscoverByFilename("main.go"); !errors.Is(err, language.ErrNilRegistry) {
		t.Errorf("got %v, want ErrNilRegistry", err)
	}
	if reg.Has("go") {
		t.Error("expected false for nil Has")
	}
	if reg.Count() != 0 {
		t.Error("expected 0 for nil Count")
	}
	if reg.List() != nil {
		t.Error("expected nil List for nil registry")
	}
}

func TestConcurrentRegistryReads(t *testing.T) {
	reg := language.NewRegistry()
	goLang, _ := language.New("go", "Go", []string{".go"}, []string{"go.mod"}, []string{"golang"})
	_ = reg.Register(goLang)
	_ = reg.Freeze()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_, _ = reg.Get("go")
				_, _ = reg.GetByExtension(".go")
				_, _ = reg.GetByAlias("golang")
				_, _ = reg.DiscoverByFilename("go.mod")
				_ = reg.Has("go")
				_ = reg.Count()
				_ = reg.List()
			}
		}()
	}
	wg.Wait()
}

func TestStateStringRepresentation(t *testing.T) {
	states := map[language.State]string{
		language.StateCreated:     "CREATED",
		language.StateOperational: "OPERATIONAL",
		language.StateTerminated:  "TERMINATED",
		language.State(99):        "UNKNOWN(99)",
	}

	for st, str := range states {
		if st.String() != str {
			t.Errorf("got %q, want %q", st.String(), str)
		}
	}
}
