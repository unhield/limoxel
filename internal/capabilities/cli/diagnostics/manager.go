package diagnostics

import (
	"io"
	"os"
	"strings"
	"sync"

	"github.com/unhield/limoxel/internal/capabilities/cli/config"
)

// ManagerOptions configures the master operational diagnostics coordinator.
type ManagerOptions struct {
	WorkspaceDir string
	Config       *config.EffectiveConfig
	LogLevelStr  string
	LogFormatStr string
	LogFilePath  string
	Verbose      bool
	Debug        bool
	Trace        bool
	ProfileCPU   string
	ProfileMem   string
	Output       io.Writer
}

// Manager is the master coordinator for Logging, Diagnostics, Profiling, Tracing, and Health Monitoring.
type Manager struct {
	mu           sync.RWMutex
	options      ManagerOptions
	logger       *Logger
	engine       *DiagnosticEngine
	profiler     *Profiler
	tracer       *Tracer
	health       *HealthMonitor
	workspaceDir string
}

// NewManager constructs and initializes the master Diagnostics Manager.
func NewManager(opts ManagerOptions) (*Manager, error) {
	wsDir := strings.TrimSpace(opts.WorkspaceDir)
	if wsDir == "" {
		wsDir = "."
	}

	// 1. Resolve Log Level (Precedence: Flags/Options > Configuration > Default Info)
	level := LevelInfo
	if opts.Debug || opts.Verbose {
		level = LevelDebug
	} else if opts.LogLevelStr != "" {
		if l, err := ParseLogLevel(opts.LogLevelStr); err == nil {
			level = l
		}
	} else if opts.Config != nil {
		lvlStr := opts.Config.GetString("logging.level", "info")
		if l, err := ParseLogLevel(lvlStr); err == nil {
			level = l
		}
	}

	// 2. Resolve Log Format (Precedence: Flags > Configuration > Default Text)
	format := LogFormatText
	if strings.ToLower(strings.TrimSpace(opts.LogFormatStr)) == "json" {
		format = LogFormatJSON
	} else if opts.Config != nil {
		fmtStr := opts.Config.GetString("logging.format", "text")
		if strings.ToLower(strings.TrimSpace(fmtStr)) == "json" {
			format = LogFormatJSON
		}
	}

	// 3. Resolve Log File Path (Precedence: Flags > Configuration > Default None)
	filePath := opts.LogFilePath
	if filePath == "" && opts.Config != nil {
		filePath = opts.Config.GetString("logging.file", "")
	}

	out := opts.Output
	if out == nil {
		out = os.Stderr
	}

	// Initialize structured Logger
	logger, err := NewLogger(LoggerOptions{
		Level:    level,
		Format:   format,
		Output:   out,
		FilePath: filePath,
	})
	if err != nil {
		return nil, err
	}

	// Initialize Profiler
	profiler := NewProfiler()
	if opts.ProfileCPU != "" {
		_ = profiler.StartCPUProfile(opts.ProfileCPU)
	}

	// Initialize Tracer
	tracer := NewTracer(opts.Trace || opts.Debug)

	// Initialize Diagnostic Engine & Health Monitor
	engine := NewDiagnosticEngine()
	health := NewHealthMonitor(wsDir)

	m := &Manager{
		options:      opts,
		logger:       logger,
		engine:       engine,
		profiler:     profiler,
		tracer:       tracer,
		health:       health,
		workspaceDir: wsDir,
	}

	return m, nil
}

// Logger returns the active structured Logger.
func (m *Manager) Logger() *Logger {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.logger
}

// Engine returns the diagnostic collection engine.
func (m *Manager) Engine() *DiagnosticEngine {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.engine
}

// Profiler returns the runtime profiler.
func (m *Manager) Profiler() *Profiler {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.profiler
}

// Tracer returns the execution tracer.
func (m *Manager) Tracer() *Tracer {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tracer
}

// Health returns the operational health monitor.
func (m *Manager) Health() *HealthMonitor {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.health
}

// RunDiagnostics collects diagnostic entries using active workspace and configuration context.
func (m *Manager) RunDiagnostics(opts DiagnosticOptions) []DiagnosticEntry {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()

	if opts.RepoPath == "" {
		opts.RepoPath = m.workspaceDir
	}
	if opts.Config == nil {
		opts.Config = m.options.Config
	}

	return m.engine.CollectAll(opts)
}

// RunHealth executes all operational health checks.
func (m *Manager) RunHealth() HealthReport {
	if m == nil {
		return HealthReport{OverallStatus: HealthUnavailable}
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.health.RunAllChecks()
}

// DumpState compiles an operational state dump.
func (m *Manager) DumpState() map[string]any {
	if m == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return DumpOperationalState(m.logger, m.profiler, m.options.Config, m.workspaceDir)
}

// Close flushes logs, stops active CPU profiles, and writes memory profiles if requested.
func (m *Manager) Close() {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.options.ProfileMem != "" && m.profiler != nil {
		_, _ = m.profiler.WriteHeapProfile(m.options.ProfileMem)
	}

	if m.profiler != nil {
		m.profiler.Close()
	}

	if m.logger != nil {
		_ = m.logger.Close()
	}
}
