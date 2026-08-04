package engine_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/unhield/limoxel/internal/engine"
	"github.com/unhield/limoxel/internal/filesystem"
	"github.com/unhield/limoxel/internal/language"
	"github.com/unhield/limoxel/internal/parser"
	"github.com/unhield/limoxel/internal/platform/context"
	"github.com/unhield/limoxel/internal/project"
	"github.com/unhield/limoxel/internal/repository"
	"github.com/unhield/limoxel/internal/workspace"
)

func helperCreateEngine(t *testing.T) (*engine.Engine, *engine.Config) {
	t.Helper()
	cfg, err := engine.NewConfig("Limoxel-Engine", "Limoxel Core Engine")
	if err != nil {
		t.Fatalf("NewConfig failed: %v", err)
	}
	eng, err := engine.New(cfg)
	if err != nil {
		t.Fatalf("engine.New failed: %v", err)
	}
	return eng, cfg
}

func helperAttachSubsystems(t *testing.T, eng *engine.Engine) {
	t.Helper()
	tempDir := t.TempDir()

	fsSer, _ := filesystem.NewFileService(filesystem.NewOSFilesystem())
	langReg := language.NewRegistry()
	parserReg := parser.NewRegistry()
	ws, _ := workspace.New("ws-1", tempDir)
	proj, _ := project.New("proj-1", ws, tempDir)
	repo, _ := repository.New("repo-1", proj, tempDir)

	_ = eng.WithFilesystem(fsSer)
	_ = eng.WithLanguageRegistry(langReg)
	_ = eng.WithParserRegistry(parserReg)
	_ = eng.WithWorkspace(ws)
	_ = eng.WithRepository(repo)
}

func TestConfigConstructorAndGetters(t *testing.T) {
	t.Run("valid config creation", func(t *testing.T) {
		cfg, err := engine.NewConfig("Engine-01", "Core Engine")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.ID() != "engine-01" {
			t.Errorf("got ID %q, want engine-01", cfg.ID())
		}
		if cfg.Name() != "Core Engine" {
			t.Errorf("got Name %q, want Core Engine", cfg.Name())
		}
	})

	t.Run("empty ID error", func(t *testing.T) {
		_, err := engine.NewConfig("   ", "Core Engine")
		if !errors.Is(err, engine.ErrInvalidConfig) {
			t.Errorf("got %v, want ErrInvalidConfig", err)
		}
	})

	t.Run("spaces in ID error", func(t *testing.T) {
		_, err := engine.NewConfig("engine 01", "Core Engine")
		if err == nil || !errors.Is(err, engine.ErrInvalidConfig) {
			t.Errorf("got %v, want ErrInvalidConfig for spaces", err)
		}
	})

	t.Run("empty Name error", func(t *testing.T) {
		_, err := engine.NewConfig("engine-01", "   ")
		if !errors.Is(err, engine.ErrInvalidConfig) {
			t.Errorf("got %v, want ErrInvalidConfig", err)
		}
	})

	t.Run("nil config getters", func(t *testing.T) {
		var cfg *engine.Config
		if cfg.ID() != "" || cfg.Name() != "" {
			t.Error("expected empty string getters on nil config")
		}
	})
}

func TestEngineConstructionAndSubsystems(t *testing.T) {
	t.Run("nil config error", func(t *testing.T) {
		_, err := engine.New(nil)
		if !errors.Is(err, engine.ErrNilConfig) {
			t.Errorf("got %v, want ErrNilConfig", err)
		}
	})

	t.Run("subsystem attachments and getters", func(t *testing.T) {
		eng, cfg := helperCreateEngine(t)
		if eng.Config() != cfg {
			t.Error("config mismatch")
		}
		if eng.State() != engine.StateCreated {
			t.Errorf("initial state got %v, want StateCreated", eng.State())
		}

		tempDir := t.TempDir()
		fsSer, _ := filesystem.NewFileService(filesystem.NewOSFilesystem())
		langReg := language.NewRegistry()
		parserReg := parser.NewRegistry()
		ws, _ := workspace.New("ws-1", tempDir)
		proj, _ := project.New("proj-1", ws, tempDir)
		repo, _ := repository.New("repo-1", proj, tempDir)

		if err := eng.WithFilesystem(fsSer); err != nil {
			t.Errorf("WithFilesystem failed: %v", err)
		}
		if err := eng.WithLanguageRegistry(langReg); err != nil {
			t.Errorf("WithLanguageRegistry failed: %v", err)
		}
		if err := eng.WithParserRegistry(parserReg); err != nil {
			t.Errorf("WithParserRegistry failed: %v", err)
		}
		if err := eng.WithWorkspace(ws); err != nil {
			t.Errorf("WithWorkspace failed: %v", err)
		}
		if err := eng.WithRepository(repo); err != nil {
			t.Errorf("WithRepository failed: %v", err)
		}

		if eng.Filesystem() != fsSer || eng.LanguageRegistry() != langReg || eng.ParserRegistry() != parserReg || eng.Workspace() != ws || eng.Repository() != repo {
			t.Error("subsystem getters mismatch")
		}
	})

	t.Run("nil subsystem error", func(t *testing.T) {
		eng, _ := helperCreateEngine(t)
		if err := eng.WithFilesystem(nil); !errors.Is(err, engine.ErrNilSubsystem) {
			t.Errorf("got %v, want ErrNilSubsystem", err)
		}
	})
}

func TestEngineLifecycleTransitions(t *testing.T) {
	eng, _ := helperCreateEngine(t)
	helperAttachSubsystems(t, eng)

	// 1. Configure
	if err := eng.Configure(); err != nil {
		t.Fatalf("Configure failed: %v", err)
	}
	if eng.State() != engine.StateConfigured {
		t.Errorf("got state %v, want StateConfigured", eng.State())
	}
	// Configure idempotency
	if err := eng.Configure(); err != nil {
		t.Errorf("idempotent Configure failed: %v", err)
	}

	// Cannot attach subsystem after StateCreated
	fsSer, _ := filesystem.NewFileService(filesystem.NewOSFilesystem())
	if err := eng.WithFilesystem(fsSer); !errors.Is(err, engine.ErrEngineState) {
		t.Errorf("got %v, want ErrEngineState", err)
	}

	// 2. Prepare
	if err := eng.Prepare(); err != nil {
		t.Fatalf("Prepare failed: %v", err)
	}
	if eng.State() != engine.StateReady {
		t.Errorf("got state %v, want StateReady", eng.State())
	}
	// Prepare idempotency
	if err := eng.Prepare(); err != nil {
		t.Errorf("idempotent Prepare failed: %v", err)
	}

	// 3. Start
	if err := eng.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if eng.State() != engine.StateRunning {
		t.Errorf("got state %v, want StateRunning", eng.State())
	}
	// Start idempotency
	if err := eng.Start(); err != nil {
		t.Errorf("idempotent Start failed: %v", err)
	}

	// 4. Stop
	if err := eng.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	if eng.State() != engine.StateStopped {
		t.Errorf("got state %v, want StateStopped", eng.State())
	}
	// Stop idempotency
	if err := eng.Stop(); err != nil {
		t.Errorf("idempotent Stop failed: %v", err)
	}

	// Re-start from StateStopped
	if err := eng.Start(); err != nil {
		t.Errorf("Re-start failed: %v", err)
	}

	// 5. Terminate
	if err := eng.Terminate(); err != nil {
		t.Fatalf("Terminate failed: %v", err)
	}
	if eng.State() != engine.StateTerminated {
		t.Errorf("got state %v, want StateTerminated", eng.State())
	}
	// Terminate idempotency
	if err := eng.Terminate(); err != nil {
		t.Errorf("idempotent Terminate failed: %v", err)
	}

	// All subsystem references cleared upon termination
	if eng.Filesystem() != nil || eng.LanguageRegistry() != nil || eng.ParserRegistry() != nil || eng.Workspace() != nil || eng.Repository() != nil {
		t.Error("subsystem references should be nil after Terminate")
	}
}

func TestEnginePrepareMissingSubsystem(t *testing.T) {
	eng, _ := helperCreateEngine(t)
	_ = eng.Configure()

	// Missing subsystems error
	err := eng.Prepare()
	if !errors.Is(err, engine.ErrMissingSubsystem) {
		t.Errorf("got %v, want ErrMissingSubsystem", err)
	}
}

func TestNilEngineSafety(t *testing.T) {
	var eng *engine.Engine

	if eng.Config() != nil {
		t.Error("expected nil Config")
	}
	if eng.State() != engine.StateTerminated {
		t.Errorf("got state %v, want StateTerminated for nil Engine", eng.State())
	}
	if err := eng.Configure(); !errors.Is(err, engine.ErrNilEngine) {
		t.Errorf("got %v, want ErrNilEngine", err)
	}
	if err := eng.Prepare(); !errors.Is(err, engine.ErrNilEngine) {
		t.Errorf("got %v, want ErrNilEngine", err)
	}
	if err := eng.Start(); !errors.Is(err, engine.ErrNilEngine) {
		t.Errorf("got %v, want ErrNilEngine", err)
	}
	if err := eng.Stop(); !errors.Is(err, engine.ErrNilEngine) {
		t.Errorf("got %v, want ErrNilEngine", err)
	}
	if err := eng.Terminate(); !errors.Is(err, engine.ErrNilEngine) {
		t.Errorf("got %v, want ErrNilEngine", err)
	}
	if eng.Filesystem() != nil || eng.LanguageRegistry() != nil || eng.ParserRegistry() != nil || eng.Workspace() != nil || eng.Repository() != nil {
		t.Error("expected nil subsystem getters for nil engine")
	}
}

func TestPipeline(t *testing.T) {
	eng, _ := helperCreateEngine(t)

	t.Run("default stages", func(t *testing.T) {
		p, err := engine.NewPipeline(eng)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.Engine() != eng {
			t.Error("engine mismatch")
		}
		if p.StageCount() != 4 {
			t.Errorf("got stage count %d, want 4", p.StageCount())
		}
		stages := p.Stages()
		if len(stages) != 4 || stages[0] != engine.StageDiscovery || stages[3] != engine.StageFinalization {
			t.Errorf("unexpected stages: %v", stages)
		}
	})

	t.Run("invalid stage sequence", func(t *testing.T) {
		_, err := engine.NewPipeline(eng, engine.StageExecution, engine.StageDiscovery)
		if !errors.Is(err, engine.ErrInvalidPipelineSequence) {
			t.Errorf("got %v, want ErrInvalidPipelineSequence", err)
		}
	})

	t.Run("nil pipeline getters", func(t *testing.T) {
		var p *engine.Pipeline
		if p.Engine() != nil || p.Stages() != nil || p.StageCount() != 0 {
			t.Error("expected zero values for nil Pipeline getters")
		}
	})
}

func TestExecutorAndRequest(t *testing.T) {
	eng, _ := helperCreateEngine(t)
	helperAttachSubsystems(t, eng)
	_ = eng.Configure()
	_ = eng.Prepare()
	_ = eng.Start()

	pipe, _ := engine.NewPipeline(eng)
	ctx := context.New()

	req, err := engine.NewRequest(pipe, ctx)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}
	if req.Pipeline() != pipe || req.Context() != ctx {
		t.Error("request getters mismatch")
	}

	exec := engine.NewExecutor()
	res, err := exec.Execute(req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !res.Success() || res.FailedStage() != engine.StageNone {
		t.Errorf("execution should be successful, got %v", res)
	}
	if len(res.CompletedStages()) != 4 {
		t.Errorf("got %d completed stages, want 4", len(res.CompletedStages()))
	}

	// Execution when Engine is not in StateRunning
	_ = eng.Stop()
	_, err = exec.Execute(req)
	if !errors.Is(err, engine.ErrEngineState) {
		t.Errorf("got %v, want ErrEngineState", err)
	}

	// Nil receiver and argument safety
	var nilExec *engine.Executor
	if _, err := nilExec.Execute(req); !errors.Is(err, engine.ErrNilExecutor) {
		t.Errorf("got %v, want ErrNilExecutor", err)
	}
	if _, err := exec.Execute(nil); !errors.Is(err, engine.ErrNilRequest) {
		t.Errorf("got %v, want ErrNilRequest", err)
	}
	var nilReq *engine.Request
	if nilReq.Pipeline() != nil || nilReq.Context() != nil {
		t.Error("expected nil getters for nil Request")
	}
}

func TestExecutionResult(t *testing.T) {
	eng, _ := helperCreateEngine(t)
	pipe, _ := engine.NewPipeline(eng)
	completed := []engine.PipelineStage{engine.StageDiscovery, engine.StageResolution}

	res := engine.NewExecutionResult(pipe, completed, engine.StageExecution, false, 100*time.Millisecond, errors.New("test error"))

	if res.Pipeline() != pipe {
		t.Error("pipeline mismatch")
	}
	if len(res.CompletedStages()) != 2 {
		t.Errorf("got %d completed stages, want 2", len(res.CompletedStages()))
	}
	// Defensive copy of completed stages
	completed[0] = engine.StageFinalization
	if res.CompletedStages()[0] != engine.StageDiscovery {
		t.Error("input slice mutation leaked into ExecutionResult")
	}

	if res.FailedStage() != engine.StageExecution {
		t.Errorf("got failed stage %v, want StageExecution", res.FailedStage())
	}
	if res.Success() {
		t.Error("expected Success == false")
	}
	if res.Duration() != 100*time.Millisecond {
		t.Errorf("got duration %v", res.Duration())
	}
	if res.Err() == nil {
		t.Error("expected non-nil error")
	}

	var nilRes *engine.ExecutionResult
	if nilRes.Pipeline() != nil || nilRes.CompletedStages() != nil || nilRes.FailedStage() != engine.StageNone || nilRes.Success() || nilRes.Duration() != 0 || nilRes.Err() != nil {
		t.Error("expected zero values for nil ExecutionResult getters")
	}
}

func TestEnumsAndStrings(t *testing.T) {
	states := map[engine.State]string{
		engine.StateCreated:    "CREATED",
		engine.StateConfigured: "CONFIGURED",
		engine.StateReady:      "READY",
		engine.StateRunning:    "RUNNING",
		engine.StateStopped:    "STOPPED",
		engine.StateTerminated: "TERMINATED",
		engine.State(99):       "UNKNOWN_STATE(99)",
	}
	for st, str := range states {
		if st.String() != str {
			t.Errorf("got %q, want %q", st.String(), str)
		}
	}

	stages := map[engine.PipelineStage]string{
		engine.StageNone:         "NONE",
		engine.StageDiscovery:    "DISCOVERY",
		engine.StageResolution:   "RESOLUTION",
		engine.StageExecution:    "EXECUTION",
		engine.StageFinalization: "FINALIZATION",
		engine.PipelineStage(99): "UNKNOWN_STAGE(99)",
	}
	for st, str := range stages {
		if st.String() != str {
			t.Errorf("got %q, want %q", st.String(), str)
		}
	}
}

func TestConcurrentEngineOperations(t *testing.T) {
	eng, _ := helperCreateEngine(t)
	helperAttachSubsystems(t, eng)
	_ = eng.Configure()
	_ = eng.Prepare()
	_ = eng.Start()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = eng.State()
				_ = eng.Filesystem()
				_ = eng.LanguageRegistry()
				_ = eng.ParserRegistry()
				_ = eng.Workspace()
				_ = eng.Repository()
			}
		}()
	}
	wg.Wait()
}
