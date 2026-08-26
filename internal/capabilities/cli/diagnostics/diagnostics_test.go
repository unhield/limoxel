package diagnostics_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/cli/config"
	"github.com/unhield/limoxel/internal/capabilities/cli/diagnostics"
)

func TestLogging(t *testing.T) {
	t.Run("log levels and text formatting", func(t *testing.T) {
		var buf bytes.Buffer
		logger, err := diagnostics.NewLogger(diagnostics.LoggerOptions{
			Level:  diagnostics.LevelDebug,
			Format: diagnostics.LogFormatText,
			Output: &buf,
		})
		if err != nil {
			t.Fatalf("failed to create logger: %v", err)
		}

		logger.Debug("debugging operation", map[string]any{"op_id": 101})
		logger.Info("informational event")
		logger.Warn("warning condition")
		logger.Error("error condition")
		logger.Critical("critical failure")

		out := buf.String()
		if !strings.Contains(out, "[DEBUG]") || !strings.Contains(out, "debugging operation") {
			t.Errorf("missing debug log in output: %s", out)
		}
		if !strings.Contains(out, "[INFO]") || !strings.Contains(out, "[WARN]") || !strings.Contains(out, "[ERROR]") || !strings.Contains(out, "[CRITICAL]") {
			t.Errorf("missing standard log levels in output: %s", out)
		}
	})

	t.Run("log level filtering", func(t *testing.T) {
		var buf bytes.Buffer
		logger, _ := diagnostics.NewLogger(diagnostics.LoggerOptions{
			Level:  diagnostics.LevelWarn,
			Format: diagnostics.LogFormatText,
			Output: &buf,
		})

		logger.Debug("should not appear")
		logger.Info("should not appear")
		logger.Warn("should appear")

		out := buf.String()
		if strings.Contains(out, "should not appear") {
			t.Errorf("filtered log appeared in output: %s", out)
		}
		if !strings.Contains(out, "should appear") {
			t.Errorf("expected warning log in output: %s", out)
		}
	})

	t.Run("json formatting", func(t *testing.T) {
		var buf bytes.Buffer
		logger, _ := diagnostics.NewLogger(diagnostics.LoggerOptions{
			Level:  diagnostics.LevelInfo,
			Format: diagnostics.LogFormatJSON,
			Output: &buf,
		})

		logger.Info("json message", map[string]any{"count": 42})
		out := buf.String()
		if !strings.Contains(out, "\"level_name\":\"INFO\"") || !strings.Contains(out, "\"message\":\"json message\"") {
			t.Errorf("invalid json log output: %s", out)
		}
	})

	t.Run("file logging", func(t *testing.T) {
		tmpDir := t.TempDir()
		logFile := filepath.Join(tmpDir, "test.log")

		logger, err := diagnostics.NewLogger(diagnostics.LoggerOptions{
			Level:    diagnostics.LevelInfo,
			FilePath: logFile,
		})
		if err != nil {
			t.Fatalf("failed to create file logger: %v", err)
		}

		logger.Info("file message line 1")
		logger.Info("file message line 2")
		_ = logger.Close()

		data, err := os.ReadFile(logFile)
		if err != nil {
			t.Fatalf("failed to read log file: %v", err)
		}
		if !strings.Contains(string(data), "file message line 1") || !strings.Contains(string(data), "file message line 2") {
			t.Errorf("file missing log content: %s", string(data))
		}
	})

	t.Run("ring buffer inspection", func(t *testing.T) {
		logger, _ := diagnostics.NewLogger(diagnostics.LoggerOptions{
			Level:        diagnostics.LevelDebug,
			RingCapacity: 5,
		})

		for i := 1; i <= 10; i++ {
			logger.Info(string(rune('A' + i - 1)))
		}

		recent := logger.GetRecentLogs(3)
		if len(recent) != 3 {
			t.Fatalf("expected 3 recent logs, got %d", len(recent))
		}
		if recent[2].Message != "J" {
			t.Errorf("expected last log to be 'J', got %q", recent[2].Message)
		}
	})
}

func TestDiagnostics(t *testing.T) {
	engine := diagnostics.NewDiagnosticEngine()
	tmpDir := t.TempDir()

	t.Run("collect system diagnostics", func(t *testing.T) {
		entries := engine.CollectSystem()
		if len(entries) == 0 {
			t.Fatal("expected system diagnostics entries")
		}
	})

	t.Run("collect repository diagnostics", func(t *testing.T) {
		entries := engine.CollectRepository(tmpDir)
		if len(entries) == 0 {
			t.Fatal("expected repository diagnostics entries")
		}

		nonExistent := filepath.Join(tmpDir, "does_not_exist")
		errEntries := engine.CollectRepository(nonExistent)
		if len(errEntries) == 0 || errEntries[0].Severity != diagnostics.DiagError {
			t.Fatal("expected error severity for non-existent repository")
		}
	})

	t.Run("collect configuration diagnostics", func(t *testing.T) {
		mgr, _ := config.NewManager(func(o *config.ManagerOptions) {
			o.WorkspaceDir = tmpDir
		})
		entries := engine.CollectConfiguration(mgr.Effective())
		if len(entries) == 0 {
			t.Fatal("expected configuration diagnostics entries")
		}
	})

	t.Run("collect performance and runtime", func(t *testing.T) {
		entries := engine.CollectPerformance()
		if len(entries) == 0 {
			t.Fatal("expected performance diagnostics entries")
		}

		rtEntries := engine.CollectRuntime()
		if len(rtEntries) == 0 {
			t.Fatal("expected runtime diagnostics entries")
		}
	})

	t.Run("filter diagnostics", func(t *testing.T) {
		all := engine.CollectAll(diagnostics.DiagnosticOptions{
			RepoPath: tmpDir,
		})
		filtered := engine.Filter(all, diagnostics.DiagCatSystem, diagnostics.DiagInfo)
		for _, f := range filtered {
			if f.Category != diagnostics.DiagCatSystem {
				t.Errorf("unexpected category %s in filtered result", f.Category)
			}
		}
	})
}

func TestProfiling(t *testing.T) {
	profiler := diagnostics.NewProfiler()
	tmpDir := t.TempDir()

	t.Run("resource stats", func(t *testing.T) {
		stats := profiler.GetResourceStats()
		if stats.CPUCount <= 0 || stats.GoVersion == "" || stats.OS == "" {
			t.Errorf("invalid resource stats: %+v", stats)
		}
	})

	t.Run("measure operation", func(t *testing.T) {
		dur, err := profiler.MeasureOperation("test_op", func() error {
			time.Sleep(10 * time.Millisecond)
			return nil
		})
		if err != nil || dur < 5*time.Millisecond {
			t.Errorf("measure operation failed: dur=%v, err=%v", dur, err)
		}
	})

	t.Run("heap profiling", func(t *testing.T) {
		memFile := filepath.Join(tmpDir, "mem.pprof")
		res, err := profiler.WriteHeapProfile(memFile)
		if err != nil {
			t.Fatalf("failed to write heap profile: %v", err)
		}
		if res.Type != diagnostics.ProfileMemory || res.FilePath != memFile {
			t.Errorf("unexpected profile result: %+v", res)
		}
		if _, err := os.Stat(memFile); err != nil {
			t.Errorf("heap profile file not created: %v", err)
		}
	})

	t.Run("cpu profiling", func(t *testing.T) {
		cpuFile := filepath.Join(tmpDir, "cpu.pprof")
		if err := profiler.StartCPUProfile(cpuFile); err != nil {
			t.Fatalf("failed to start CPU profile: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
		res, err := profiler.StopCPUProfile()
		if err != nil {
			t.Fatalf("failed to stop CPU profile: %v", err)
		}
		if res.Type != diagnostics.ProfileCPU || res.FilePath != cpuFile {
			t.Errorf("unexpected CPU profile result: %+v", res)
		}
	})
}

func TestDebugAndTracer(t *testing.T) {
	tracer := diagnostics.NewTracer(true)

	span1 := tracer.StartSpan("root_task", "engine", "", map[string]string{"env": "test"})
	span2 := tracer.StartSpan("sub_task", "parser", span1.ID, map[string]string{"secret_token": "secret123"})
	time.Sleep(5 * time.Millisecond)
	tracer.EndSpan(span2)
	tracer.EndSpan(span1)

	spans := tracer.Spans()
	if len(spans) != 2 {
		t.Fatalf("expected 2 spans, got %d", len(spans))
	}
	if spans[1].Tags["secret_token"] != "***REDACTED***" {
		t.Errorf("expected secret tag to be redacted, got %q", spans[1].Tags["secret_token"])
	}

	formatted := diagnostics.FormatSpansText(spans)
	if !strings.Contains(formatted, "root_task") || !strings.Contains(formatted, "sub_task") {
		t.Errorf("formatted spans text missing content: %s", formatted)
	}

	t.Run("state dump", func(t *testing.T) {
		dump := diagnostics.DumpOperationalState(nil, nil, nil, "/tmp/repo")
		if dump["repo_path"] != "/tmp/repo" || dump["pid"] == nil {
			t.Errorf("invalid state dump: %+v", dump)
		}
	})
}

func TestHealthMonitoring(t *testing.T) {
	tmpDir := t.TempDir()
	monitor := diagnostics.NewHealthMonitor(tmpDir)

	report := monitor.RunAllChecks()
	if report.OverallStatus != diagnostics.HealthHealthy {
		t.Errorf("expected overall status healthy on fresh temp dir, got %s", report.OverallStatus)
	}
	if len(report.Checks) < 5 {
		t.Errorf("expected at least 5 default health checks, got %d", len(report.Checks))
	}

	t.Run("panic recovery in custom health checker", func(t *testing.T) {
		monitor.RegisterChecker(&panicChecker{})
		panicReport := monitor.RunAllChecks()
		if panicReport.OverallStatus != diagnostics.HealthFailed {
			t.Errorf("expected overall status failed on panicking checker, got %s", panicReport.OverallStatus)
		}
	})
}

type panicChecker struct{}

func (p *panicChecker) Name() string                             { return "panicking_probe" }
func (p *panicChecker) Category() diagnostics.DiagnosticCategory { return diagnostics.DiagCatRuntime }
func (p *panicChecker) Check() diagnostics.HealthCheckResult     { panic("simulated probe failure") }

func TestSecretRedaction(t *testing.T) {
	var buf bytes.Buffer
	logger, _ := diagnostics.NewLogger(diagnostics.LoggerOptions{
		Level:  diagnostics.LevelDebug,
		Format: diagnostics.LogFormatJSON,
		Output: &buf,
		Redact: true,
	})

	logger.Info("authentication attempt", map[string]any{
		"api_key":   "secret_key_12345",
		"password":  "super_secret_pw",
		"safe_user": "alice",
	})

	out := buf.String()
	if strings.Contains(out, "secret_key_12345") || strings.Contains(out, "super_secret_pw") {
		t.Fatalf("secret leaked in log output: %s", out)
	}
	if !strings.Contains(out, "***REDACTED***") || !strings.Contains(out, "alice") {
		t.Fatalf("expected redacted value and safe values in output: %s", out)
	}
}

func TestManagerLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := diagnostics.NewManager(diagnostics.ManagerOptions{
		WorkspaceDir: tmpDir,
		LogLevelStr:  "debug",
		LogFormatStr: "text",
		Trace:        true,
	})
	if err != nil {
		t.Fatalf("failed to initialize diagnostics manager: %v", err)
	}
	defer mgr.Close()

	if mgr.Logger().Level() != diagnostics.LevelDebug {
		t.Errorf("expected debug log level, got %s", mgr.Logger().Level())
	}

	diags := mgr.RunDiagnostics(diagnostics.DiagnosticOptions{})
	if len(diags) == 0 {
		t.Fatal("expected diagnostics from manager")
	}

	health := mgr.RunHealth()
	if health.OverallStatus != diagnostics.HealthHealthy {
		t.Errorf("expected healthy report, got %s", health.OverallStatus)
	}

	state := mgr.DumpState()
	if state["repo_path"] != tmpDir {
		t.Errorf("expected repo_path %q in state dump, got %v", tmpDir, state["repo_path"])
	}
}

func TestConcurrencyAndRace(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := diagnostics.NewManager(diagnostics.ManagerOptions{
		WorkspaceDir: tmpDir,
		Trace:        true,
	})
	if err != nil {
		t.Fatalf("failed to initialize manager: %v", err)
	}
	defer mgr.Close()

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			mgr.Logger().Info("concurrent log message", map[string]any{"worker": idx})
			_ = mgr.RunDiagnostics(diagnostics.DiagnosticOptions{})
			_ = mgr.RunHealth()
			span := mgr.Tracer().StartSpan("concurrent_span", "test", "", nil)
			time.Sleep(time.Millisecond)
			mgr.Tracer().EndSpan(span)
			_ = mgr.DumpState()
		}(i)
	}
	wg.Wait()
}
