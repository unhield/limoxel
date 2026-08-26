package diagnostics

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"time"
)

// HealthChecker defines the contract for an individual component health probe.
type HealthChecker interface {
	Name() string
	Category() DiagnosticCategory
	Check() HealthCheckResult
}

// HealthMonitor coordinates and aggregates operational health probes.
type HealthMonitor struct {
	mu       sync.RWMutex
	checkers []HealthChecker
	repoPath string
}

// NewHealthMonitor constructs an initialized HealthMonitor with standard platform checks.
func NewHealthMonitor(repoPath string) *HealthMonitor {
	cleanPath := filepath.Clean(repoPath)
	if cleanPath == "" {
		cleanPath = "."
	}

	m := &HealthMonitor{
		checkers: make([]HealthChecker, 0, 10),
		repoPath: cleanPath,
	}

	// Register default baseline operational checks
	m.RegisterChecker(&SystemHealthChecker{})
	m.RegisterChecker(&RepositoryHealthChecker{repoPath: cleanPath})
	m.RegisterChecker(&RuntimeHealthChecker{})
	m.RegisterChecker(&CacheHealthChecker{repoPath: cleanPath})
	m.RegisterChecker(&IndexHealthChecker{repoPath: cleanPath})

	return m
}

// RegisterChecker adds a health check probe to the monitor.
func (m *HealthMonitor) RegisterChecker(checker HealthChecker) {
	if m == nil || checker == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checkers = append(m.checkers, checker)
}

// RunAllChecks executes all registered health checks and compiles a HealthReport.
func (m *HealthMonitor) RunAllChecks() HealthReport {
	m.mu.RLock()
	checkers := make([]HealthChecker, len(m.checkers))
	copy(checkers, m.checkers)
	m.mu.RUnlock()

	start := time.Now()
	results := make([]HealthCheckResult, 0, len(checkers))

	for _, c := range checkers {
		chkStart := time.Now()
		res := func(checker HealthChecker) (r HealthCheckResult) {
			defer func() {
				if rec := recover(); rec != nil {
					r = HealthCheckResult{
						Name:      checker.Name(),
						Category:  checker.Category(),
						Status:    HealthFailed,
						Message:   fmt.Sprintf("Health check panic recovered: %v", rec),
						Timestamp: time.Now().UTC(),
					}
				}
			}()
			return checker.Check()
		}(c)
		res.LatencyMs = float64(time.Since(chkStart).Microseconds()) / 1000.0
		results = append(results, res)
	}

	// Sort results deterministically by status severity, category, then name
	sort.Slice(results, func(i, j int) bool {
		sI, sJ := statusWeight(results[i].Status), statusWeight(results[j].Status)
		if sI != sJ {
			return sI > sJ
		}
		if results[i].Category != results[j].Category {
			return results[i].Category < results[j].Category
		}
		return results[i].Name < results[j].Name
	})

	overall := HealthHealthy
	for _, r := range results {
		if r.Status == HealthFailed {
			overall = HealthFailed
			break
		} else if r.Status == HealthUnavailable && overall != HealthFailed {
			overall = HealthUnavailable
		} else if r.Status == HealthDegraded && overall == HealthHealthy {
			overall = HealthDegraded
		}
	}

	return HealthReport{
		OverallStatus: overall,
		Timestamp:     time.Now().UTC(),
		DurationMs:    float64(time.Since(start).Microseconds()) / 1000.0,
		Checks:        results,
	}
}

func statusWeight(s HealthStatus) int {
	switch s {
	case HealthFailed:
		return 40
	case HealthUnavailable:
		return 30
	case HealthDegraded:
		return 20
	case HealthHealthy:
		return 10
	default:
		return 0
	}
}

// 1. System Health Checker
type SystemHealthChecker struct{}

func (c *SystemHealthChecker) Name() string                 { return "system_resources" }
func (c *SystemHealthChecker) Category() DiagnosticCategory { return DiagCatSystem }
func (c *SystemHealthChecker) Check() HealthCheckResult {
	now := time.Now().UTC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	allocMB := float64(m.Alloc) / (1024 * 1024)
	status := HealthHealthy
	msg := fmt.Sprintf("System resources nominal (CPU cores: %d, Heap: %.2f MB)", runtime.NumCPU(), allocMB)

	if allocMB > 2048 {
		status = HealthDegraded
		msg = fmt.Sprintf("High memory consumption (Heap: %.2f MB)", allocMB)
	}

	return HealthCheckResult{
		Name:      c.Name(),
		Category:  c.Category(),
		Status:    status,
		Message:   msg,
		Timestamp: now,
		Details: map[string]any{
			"num_cpu":  runtime.NumCPU(),
			"alloc_mb": allocMB,
		},
	}
}

// 2. Repository Health Checker
type RepositoryHealthChecker struct {
	repoPath string
}

func (c *RepositoryHealthChecker) Name() string                 { return "workspace_repository" }
func (c *RepositoryHealthChecker) Category() DiagnosticCategory { return DiagCatRepository }
func (c *RepositoryHealthChecker) Check() HealthCheckResult {
	now := time.Now().UTC()
	info, err := os.Stat(c.repoPath)
	if err != nil {
		return HealthCheckResult{
			Name:      c.Name(),
			Category:  c.Category(),
			Status:    HealthUnavailable,
			Message:   fmt.Sprintf("Repository path %q is unavailable: %v", c.repoPath, err),
			Timestamp: now,
		}
	}

	if !info.IsDir() {
		return HealthCheckResult{
			Name:      c.Name(),
			Category:  c.Category(),
			Status:    HealthFailed,
			Message:   fmt.Sprintf("Repository path %q is a regular file, expected directory", c.repoPath),
			Timestamp: now,
		}
	}

	return HealthCheckResult{
		Name:      c.Name(),
		Category:  c.Category(),
		Status:    HealthHealthy,
		Message:   fmt.Sprintf("Workspace directory %q is accessible and valid", c.repoPath),
		Timestamp: now,
	}
}

// 3. Runtime Health Checker
type RuntimeHealthChecker struct{}

func (c *RuntimeHealthChecker) Name() string                 { return "go_runtime" }
func (c *RuntimeHealthChecker) Category() DiagnosticCategory { return DiagCatRuntime }
func (c *RuntimeHealthChecker) Check() HealthCheckResult {
	now := time.Now().UTC()
	goroutines := runtime.NumGoroutine()

	status := HealthHealthy
	msg := fmt.Sprintf("Go runtime scheduler nominal (%d active goroutines)", goroutines)
	if goroutines > 10000 {
		status = HealthDegraded
		msg = fmt.Sprintf("High goroutine count detected (%d goroutines)", goroutines)
	}

	return HealthCheckResult{
		Name:      c.Name(),
		Category:  c.Category(),
		Status:    status,
		Message:   msg,
		Timestamp: now,
		Details: map[string]any{
			"num_goroutine": goroutines,
			"version":       runtime.Version(),
		},
	}
}

// 4. Cache Health Checker
type CacheHealthChecker struct {
	repoPath string
}

func (c *CacheHealthChecker) Name() string                 { return "cache_subsystem" }
func (c *CacheHealthChecker) Category() DiagnosticCategory { return DiagCatSystem }
func (c *CacheHealthChecker) Check() HealthCheckResult {
	now := time.Now().UTC()
	cacheDir := filepath.Join(c.repoPath, ".limoxel", "cache")

	if info, err := os.Stat(cacheDir); err == nil && info.IsDir() {
		return HealthCheckResult{
			Name:      c.Name(),
			Category:  c.Category(),
			Status:    HealthHealthy,
			Message:   fmt.Sprintf("Cache directory %q is online and accessible", cacheDir),
			Timestamp: now,
		}
	}

	return HealthCheckResult{
		Name:      c.Name(),
		Category:  c.Category(),
		Status:    HealthHealthy,
		Message:   "Cache directory not yet initialized (nominal on first run)",
		Timestamp: now,
	}
}

// 5. Index Health Checker
type IndexHealthChecker struct {
	repoPath string
}

func (c *IndexHealthChecker) Name() string                 { return "index_database" }
func (c *IndexHealthChecker) Category() DiagnosticCategory { return DiagCatRepository }
func (c *IndexHealthChecker) Check() HealthCheckResult {
	now := time.Now().UTC()
	indexDir := filepath.Join(c.repoPath, ".limoxel", "index")

	if info, err := os.Stat(indexDir); err == nil && info.IsDir() {
		return HealthCheckResult{
			Name:      c.Name(),
			Category:  c.Category(),
			Status:    HealthHealthy,
			Message:   fmt.Sprintf("Index storage directory %q is accessible", indexDir),
			Timestamp: now,
		}
	}

	return HealthCheckResult{
		Name:      c.Name(),
		Category:  c.Category(),
		Status:    HealthHealthy,
		Message:   "Index storage nominal (ephemeral in-memory or uninitialized)",
		Timestamp: now,
	}
}
