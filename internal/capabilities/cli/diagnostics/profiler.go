package diagnostics

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"strings"
	"sync"
	"time"
)

// Profiler manages CPU, Memory (heap), Execution Trace profiling sessions and runtime resource statistics.
type Profiler struct {
	mu           sync.Mutex
	cpuActive    bool
	cpuFile      *os.File
	cpuPath      string
	cpuStartTime time.Time
	traceActive  bool
	traceFile    *os.File
	tracePath    string
	traceStart   time.Time
}

// NewProfiler constructs an initialized Profiler instance.
func NewProfiler() *Profiler {
	return &Profiler{}
}

// StartCPUProfile initiates CPU profiling to the specified targetPath.
func (p *Profiler) StartCPUProfile(targetPath string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cpuActive {
		return fmt.Errorf("CPU profiling is already running on %q", p.cpuPath)
	}

	cleanPath := filepath.Clean(strings.TrimSpace(targetPath))
	if cleanPath == "" || cleanPath == "." {
		cleanPath = "cpu.pprof"
	}

	dir := filepath.Dir(cleanPath)
	if dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}

	f, err := os.Create(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to create CPU profile file %q: %w", cleanPath, err)
	}

	if err := pprof.StartCPUProfile(f); err != nil {
		_ = f.Close()
		_ = os.Remove(cleanPath)
		return fmt.Errorf("failed to start CPU profile: %w", err)
	}

	p.cpuActive = true
	p.cpuFile = f
	p.cpuPath = cleanPath
	p.cpuStartTime = time.Now()
	return nil
}

// StopCPUProfile stops the running CPU profiling session and flushes profile data.
func (p *Profiler) StopCPUProfile() (*ProfileResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.cpuActive {
		return nil, fmt.Errorf("no active CPU profiling session")
	}

	pprof.StopCPUProfile()
	_ = p.cpuFile.Close()

	dur := time.Since(p.cpuStartTime)
	var size int64
	if stat, err := os.Stat(p.cpuPath); err == nil {
		size = stat.Size()
	}

	res := &ProfileResult{
		Type:       ProfileCPU,
		FilePath:   p.cpuPath,
		DurationMs: float64(dur.Microseconds()) / 1000.0,
		Size:       size,
		Metadata: map[string]any{
			"num_cpu": runtime.NumCPU(),
		},
	}

	p.cpuActive = false
	p.cpuFile = nil
	p.cpuPath = ""
	return res, nil
}

// WriteHeapProfile writes a current memory heap profile snapshot to targetPath.
func (p *Profiler) WriteHeapProfile(targetPath string) (*ProfileResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	cleanPath := filepath.Clean(strings.TrimSpace(targetPath))
	if cleanPath == "" || cleanPath == "." {
		cleanPath = "mem.pprof"
	}

	dir := filepath.Dir(cleanPath)
	if dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}

	f, err := os.Create(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create heap profile file %q: %w", cleanPath, err)
	}
	defer f.Close()

	runtime.GC() // Request GC to get up-to-date heap metrics
	start := time.Now()
	if err := pprof.WriteHeapProfile(f); err != nil {
		return nil, fmt.Errorf("failed to write heap profile: %w", err)
	}
	dur := time.Since(start)

	var size int64
	if stat, err := os.Stat(cleanPath); err == nil {
		size = stat.Size()
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return &ProfileResult{
		Type:       ProfileMemory,
		FilePath:   cleanPath,
		DurationMs: float64(dur.Microseconds()) / 1000.0,
		Size:       size,
		Metadata: map[string]any{
			"heap_alloc_mb": float64(m.HeapAlloc) / (1024 * 1024),
			"heap_sys_mb":   float64(m.HeapSys) / (1024 * 1024),
		},
	}, nil
}

// StartTrace starts execution tracing to targetPath.
func (p *Profiler) StartTrace(targetPath string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.traceActive {
		return fmt.Errorf("execution trace is already running on %q", p.tracePath)
	}

	cleanPath := filepath.Clean(strings.TrimSpace(targetPath))
	if cleanPath == "" || cleanPath == "." {
		cleanPath = "trace.out"
	}

	dir := filepath.Dir(cleanPath)
	if dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}

	f, err := os.Create(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to create trace file %q: %w", cleanPath, err)
	}

	if err := trace.Start(f); err != nil {
		_ = f.Close()
		_ = os.Remove(cleanPath)
		return fmt.Errorf("failed to start execution trace: %w", err)
	}

	p.traceActive = true
	p.traceFile = f
	p.tracePath = cleanPath
	p.traceStart = time.Now()
	return nil
}

// StopTrace stops the execution trace and flushes the trace data.
func (p *Profiler) StopTrace() (*ProfileResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.traceActive {
		return nil, fmt.Errorf("no active execution trace session")
	}

	trace.Stop()
	_ = p.traceFile.Close()

	dur := time.Since(p.traceStart)
	var size int64
	if stat, err := os.Stat(p.tracePath); err == nil {
		size = stat.Size()
	}

	res := &ProfileResult{
		Type:       ProfileTrace,
		FilePath:   p.tracePath,
		DurationMs: float64(dur.Microseconds()) / 1000.0,
		Size:       size,
	}

	p.traceActive = false
	p.traceFile = nil
	p.tracePath = ""
	return res, nil
}

// MeasureOperation runs an operation, measures its execution duration, and returns the elapsed time.
func (p *Profiler) MeasureOperation(name string, fn func() error) (time.Duration, error) {
	start := time.Now()
	err := fn()
	elapsed := time.Since(start)
	return elapsed, err
}

// GetResourceStats captures a point-in-time snapshot of runtime metrics.
func (p *Profiler) GetResourceStats() ResourceStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return ResourceStats{
		NumGoroutine: runtime.NumGoroutine(),
		AllocMB:      float64(m.Alloc) / (1024 * 1024),
		TotalAllocMB: float64(m.TotalAlloc) / (1024 * 1024),
		SysMB:        float64(m.Sys) / (1024 * 1024),
		NumGC:        m.NumGC,
		PauseTotalMs: float64(m.PauseTotalNs) / 1000000.0,
		CPUCount:     runtime.NumCPU(),
		GoVersion:    runtime.Version(),
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
	}
}

// Close ensures any running profiling sessions are stopped gracefully.
func (p *Profiler) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cpuActive {
		pprof.StopCPUProfile()
		if p.cpuFile != nil {
			_ = p.cpuFile.Close()
		}
		p.cpuActive = false
	}
	if p.traceActive {
		trace.Stop()
		if p.traceFile != nil {
			_ = p.traceFile.Close()
		}
		p.traceActive = false
	}
}
