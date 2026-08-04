package parser_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/unhield/limoxel/internal/parser"
)

func TestDescriptorConstructorAndGetters(t *testing.T) {
	t.Run("valid descriptor creation", func(t *testing.T) {
		desc, err := parser.NewDescriptor("Go-Parser", "Go AST Parser", "GO", "2.1.0")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if desc.ID() != "go-parser" {
			t.Errorf("got ID %q, want go-parser", desc.ID())
		}
		if desc.Name() != "Go AST Parser" {
			t.Errorf("got Name %q, want Go AST Parser", desc.Name())
		}
		if desc.LanguageID() != "go" {
			t.Errorf("got LanguageID %q, want go", desc.LanguageID())
		}
		if desc.Version() != "2.1.0" {
			t.Errorf("got Version %q, want 2.1.0", desc.Version())
		}
	})

	t.Run("default version fallback", func(t *testing.T) {
		desc, err := parser.NewDescriptor("py-parser", "Python Parser", "python", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if desc.Version() != "1.0.0" {
			t.Errorf("got Version %q, want 1.0.0", desc.Version())
		}
	})

	t.Run("invalid ID errors", func(t *testing.T) {
		_, err := parser.NewDescriptor("  ", "Parser", "go", "1.0")
		if !errors.Is(err, parser.ErrInvalidID) {
			t.Errorf("got %v, want ErrInvalidID", err)
		}

		_, err = parser.NewDescriptor("go parser", "Parser", "go", "1.0")
		if err == nil || !errors.Is(err, parser.ErrInvalidID) {
			t.Errorf("got %v, want ErrInvalidID for spaces", err)
		}
	})

	t.Run("invalid Name error", func(t *testing.T) {
		_, err := parser.NewDescriptor("go-parser", "  ", "go", "1.0")
		if !errors.Is(err, parser.ErrInvalidName) {
			t.Errorf("got %v, want ErrInvalidName", err)
		}
	})

	t.Run("invalid LanguageID error", func(t *testing.T) {
		_, err := parser.NewDescriptor("go-parser", "Go Parser", "   ", "1.0")
		if !errors.Is(err, parser.ErrInvalidLanguageID) {
			t.Errorf("got %v, want ErrInvalidLanguageID", err)
		}
	})

	t.Run("nil descriptor getters", func(t *testing.T) {
		var d *parser.Descriptor
		if d.ID() != "" || d.Name() != "" || d.LanguageID() != "" || d.Version() != "" {
			t.Error("expected empty string getters on nil descriptor")
		}
	})
}

func TestRegistryRegistrationAndLifecycle(t *testing.T) {
	reg := parser.NewRegistry()
	d1, _ := parser.NewDescriptor("go-parser", "Go Parser", "go", "1.0")
	d2, _ := parser.NewDescriptor("py-parser", "Py Parser", "python", "1.0")

	// Register
	if err := reg.Register(d1); err != nil {
		t.Fatalf("Register d1 failed: %v", err)
	}
	if err := reg.Register(d2); err != nil {
		t.Fatalf("Register d2 failed: %v", err)
	}

	if reg.Count() != 2 {
		t.Errorf("got count %d, want 2", reg.Count())
	}
	if !reg.Has("go-parser") || !reg.Has("py-parser") {
		t.Error("expected Has to return true")
	}

	// Duplicate registration error
	err := reg.Register(d1)
	if !errors.Is(err, parser.ErrDuplicateParser) {
		t.Errorf("got %v, want ErrDuplicateParser", err)
	}

	// State transitions: initial state is StateRegistered
	st, err := reg.State("go-parser")
	if err != nil || st != parser.StateRegistered {
		t.Errorf("got state %v, %v; want StateRegistered", st, err)
	}
	if reg.IsActive("go-parser") {
		t.Error("should not be active yet")
	}

	// Activate
	if err := reg.Activate("go-parser"); err != nil {
		t.Fatalf("Activate failed: %v", err)
	}
	if !reg.IsActive("go-parser") {
		t.Error("expected IsActive to be true")
	}
	// Double activate error
	if err := reg.Activate("go-parser"); !errors.Is(err, parser.ErrAlreadyActive) {
		t.Errorf("got %v, want ErrAlreadyActive", err)
	}

	// Deactivate
	if err := reg.Deactivate("go-parser"); err != nil {
		t.Fatalf("Deactivate failed: %v", err)
	}
	if reg.IsActive("go-parser") {
		t.Error("should be inactive")
	}
	// Double deactivate error
	if err := reg.Deactivate("go-parser"); !errors.Is(err, parser.ErrAlreadyInactive) {
		t.Errorf("got %v, want ErrAlreadyInactive", err)
	}

	// Re-activate
	if err := reg.Activate("go-parser"); err != nil {
		t.Errorf("Re-activate failed: %v", err)
	}

	// List ordering
	list := reg.List()
	if len(list) != 2 || list[0].ID() != "go-parser" || list[1].ID() != "py-parser" {
		t.Errorf("List() ordering mismatch: %v", list)
	}

	// Remove
	if err := reg.Remove("go-parser"); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
	if reg.Has("go-parser") {
		t.Error("go-parser should be removed")
	}
	if reg.Count() != 1 {
		t.Errorf("got count %d, want 1", reg.Count())
	}
}

func TestNilRegistrySafety(t *testing.T) {
	var reg *parser.Registry
	d, _ := parser.NewDescriptor("d", "D", "l", "1.0")

	if err := reg.Register(d); !errors.Is(err, parser.ErrNilRegistry) {
		t.Errorf("got %v, want ErrNilRegistry", err)
	}
	if err := reg.Activate("d"); !errors.Is(err, parser.ErrNilRegistry) {
		t.Errorf("got %v, want ErrNilRegistry", err)
	}
	if err := reg.Deactivate("d"); !errors.Is(err, parser.ErrNilRegistry) {
		t.Errorf("got %v, want ErrNilRegistry", err)
	}
	if _, err := reg.State("d"); !errors.Is(err, parser.ErrNilRegistry) {
		t.Errorf("got %v, want ErrNilRegistry", err)
	}
	if reg.IsActive("d") {
		t.Error("expected false for nil IsActive")
	}
	if err := reg.Remove("d"); !errors.Is(err, parser.ErrNilRegistry) {
		t.Errorf("got %v, want ErrNilRegistry", err)
	}
	if _, err := reg.Get("d"); !errors.Is(err, parser.ErrNilRegistry) {
		t.Errorf("got %v, want ErrNilRegistry", err)
	}
	if reg.Has("d") {
		t.Error("expected false for nil Has")
	}
	if reg.Count() != 0 {
		t.Error("expected 0 for nil Count")
	}
	if reg.List() != nil {
		t.Error("expected nil for nil List")
	}
}

func TestPipeline(t *testing.T) {
	desc, _ := parser.NewDescriptor("go-p", "Go P", "go", "1.0")

	t.Run("default stages", func(t *testing.T) {
		p, err := parser.NewPipeline(desc)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.Descriptor() != desc {
			t.Error("descriptor mismatch")
		}
		if p.StageCount() != 3 {
			t.Errorf("got stage count %d, want 3", p.StageCount())
		}
		stages := p.Stages()
		if len(stages) != 3 || stages[0] != parser.StagePrepare || stages[1] != parser.StageProcess || stages[2] != parser.StageFinalize {
			t.Errorf("unexpected stages: %v", stages)
		}
	})

	t.Run("custom stages and defensive copy", func(t *testing.T) {
		custom := []parser.PipelineStage{parser.StagePrepare, parser.StageProcess}
		p, err := parser.NewPipeline(desc, custom...)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		custom[0] = parser.StageFinalize
		if p.Stages()[0] != parser.StagePrepare {
			t.Error("input slice mutation leaked into Pipeline")
		}
	})

	t.Run("invalid stage sequence", func(t *testing.T) {
		// Non-monotonic ordering
		_, err := parser.NewPipeline(desc, parser.StageProcess, parser.StagePrepare)
		if !errors.Is(err, parser.ErrInvalidPipelineSequence) {
			t.Errorf("got %v, want ErrInvalidPipelineSequence", err)
		}
	})

	t.Run("nil pipeline receiver", func(t *testing.T) {
		var p *parser.Pipeline
		if p.Descriptor() != nil || p.Stages() != nil || p.StageCount() != 0 {
			t.Error("expected zero values for nil Pipeline getters")
		}
	})
}

func TestResult(t *testing.T) {
	desc, _ := parser.NewDescriptor("go-p", "Go P", "go", "1.0")
	pipe, _ := parser.NewPipeline(desc)
	now := time.Now().UTC()
	meta := map[string]string{"tokens": "100"}

	res, err := parser.NewResult(desc, pipe, parser.StatusSuccess, now, meta)
	if err != nil {
		t.Fatalf("NewResult failed: %v", err)
	}

	if res.Descriptor() != desc {
		t.Error("descriptor mismatch")
	}
	if res.Pipeline() != pipe {
		t.Error("pipeline mismatch")
	}
	if res.Status() != parser.StatusSuccess || !res.Successful() {
		t.Error("expected StatusSuccess and Successful == true")
	}
	if res.Timestamp() != now {
		t.Errorf("got timestamp %v, want %v", res.Timestamp(), now)
	}

	// Defensive copy of metadata
	meta["tokens"] = "999"
	if res.Metadata()["tokens"] != "100" {
		t.Error("metadata mutation leaked into Result")
	}

	// Nil result receiver
	var nilRes *parser.Result
	if nilRes.Descriptor() != nil || nilRes.Pipeline() != nil || nilRes.Status() != parser.StatusFailure || !nilRes.Timestamp().IsZero() || nilRes.Metadata() != nil || nilRes.Successful() {
		t.Error("expected zero/safe values for nil Result getters")
	}

	// Invalid status
	_, err = parser.NewResult(desc, pipe, parser.Status(99), now, nil)
	if !errors.Is(err, parser.ErrInvalidStatus) {
		t.Errorf("got %v, want ErrInvalidStatus", err)
	}
}

func TestEnumsAndStrings(t *testing.T) {
	stages := map[parser.PipelineStage]string{
		parser.StagePrepare:         "PREPARE",
		parser.StageProcess:         "PROCESS",
		parser.StageFinalize:        "FINALIZE",
		parser.PipelineStage(99):    "UNKNOWN_STAGE(99)",
	}
	for st, str := range stages {
		if st.String() != str {
			t.Errorf("got %q, want %q", st.String(), str)
		}
	}

	statuses := map[parser.Status]string{
		parser.StatusSuccess:     "SUCCESS",
		parser.StatusFailure:     "FAILURE",
		parser.StatusPartial:     "PARTIAL",
		parser.Status(99):        "UNKNOWN_STATUS(99)",
	}
	for st, str := range statuses {
		if st.String() != str {
			t.Errorf("got %q, want %q", st.String(), str)
		}
	}

	states := map[parser.State]string{
		parser.StateRegistered: "REGISTERED",
		parser.StateActive:     "ACTIVE",
		parser.StateInactive:   "INACTIVE",
		parser.State(99):       "UNKNOWN(99)",
	}
	for st, str := range states {
		if st.String() != str {
			t.Errorf("got %q, want %q", st.String(), str)
		}
	}
}

func TestConcurrentRegistryReads(t *testing.T) {
	reg := parser.NewRegistry()
	desc, _ := parser.NewDescriptor("go-p", "Go P", "go", "1.0")
	_ = reg.Register(desc)
	_ = reg.Activate("go-p")

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_, _ = reg.Get("go-p")
				_, _ = reg.State("go-p")
				_ = reg.IsActive("go-p")
				_ = reg.Has("go-p")
				_ = reg.Count()
				_ = reg.List()
			}
		}()
	}
	wg.Wait()
}
