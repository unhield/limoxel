package diagnostics

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/cli/config"
)

// DiagnosticOptions configures a diagnostic collection pass.
type DiagnosticOptions struct {
	RepoPath    string
	Config      *config.EffectiveConfig
	Category    DiagnosticCategory
	MinSeverity DiagnosticSeverity
}

// DiagnosticEngine orchestrates and collects system, repository, configuration, dependency, and performance diagnostics.
type DiagnosticEngine struct {
	mu sync.RWMutex
}

// NewDiagnosticEngine constructs an initialized DiagnosticEngine.
func NewDiagnosticEngine() *DiagnosticEngine {
	return &DiagnosticEngine{}
}

// CollectAll executes diagnostic collectors and returns filtered, sorted entries.
func (e *DiagnosticEngine) CollectAll(opts DiagnosticOptions) []DiagnosticEntry {
	var entries []DiagnosticEntry

	entries = append(entries, e.CollectSystem()...)
	if opts.RepoPath != "" {
		entries = append(entries, e.CollectRepository(opts.RepoPath)...)
		entries = append(entries, e.CollectDependency(opts.RepoPath)...)
	}
	if opts.Config != nil {
		entries = append(entries, e.CollectConfiguration(opts.Config)...)
	}
	entries = append(entries, e.CollectPerformance()...)
	entries = append(entries, e.CollectRuntime()...)

	return e.Filter(entries, opts.Category, opts.MinSeverity)
}

// CollectSystem captures host environment, OS, architecture, and Go runtime conditions.
func (e *DiagnosticEngine) CollectSystem() []DiagnosticEntry {
	now := time.Now().UTC()
	var entries []DiagnosticEntry

	// OS and architecture observation
	entries = append(entries, DiagnosticEntry{
		ID:        "sys-platform-01",
		Timestamp: now,
		Severity:  DiagInfo,
		Category:  DiagCatSystem,
		Component: "host",
		Message:   fmt.Sprintf("Host platform: %s/%s (%d CPUs)", runtime.GOOS, runtime.GOARCH, runtime.NumCPU()),
		Details: map[string]any{
			"os":       runtime.GOOS,
			"arch":     runtime.GOARCH,
			"num_cpu":  runtime.NumCPU(),
			"compiler": runtime.Compiler,
		},
	})

	// Go runtime version
	entries = append(entries, DiagnosticEntry{
		ID:        "sys-go-version-01",
		Timestamp: now,
		Severity:  DiagInfo,
		Category:  DiagCatSystem,
		Component: "go_runtime",
		Message:   fmt.Sprintf("Go runtime version: %s", runtime.Version()),
		Details: map[string]any{
			"version": runtime.Version(),
		},
	})

	return entries
}

// CollectRepository inspects directory accessibility, permissions, and git workspace metadata.
func (e *DiagnosticEngine) CollectRepository(repoPath string) []DiagnosticEntry {
	now := time.Now().UTC()
	var entries []DiagnosticEntry
	cleanPath := filepath.Clean(repoPath)

	info, err := os.Stat(cleanPath)
	if err != nil {
		entries = append(entries, DiagnosticEntry{
			ID:          "repo-access-01",
			Timestamp:   now,
			Severity:    DiagError,
			Category:    DiagCatRepository,
			Component:   "filesystem",
			Location:    cleanPath,
			Message:     fmt.Sprintf("Repository path %q is inaccessible or does not exist", cleanPath),
			Error:       err.Error(),
			Remediation: "Verify the directory path and filesystem permissions.",
		})
		return entries
	}

	if !info.IsDir() {
		entries = append(entries, DiagnosticEntry{
			ID:          "repo-type-01",
			Timestamp:   now,
			Severity:    DiagError,
			Category:    DiagCatRepository,
			Component:   "filesystem",
			Location:    cleanPath,
			Message:     fmt.Sprintf("Target repository path %q is a file, not a directory", cleanPath),
			Remediation: "Specify a valid workspace directory root.",
		})
		return entries
	}

	// Check directory read permissions
	f, errOpen := os.Open(cleanPath)
	if errOpen != nil {
		entries = append(entries, DiagnosticEntry{
			ID:          "repo-perm-01",
			Timestamp:   now,
			Severity:    DiagError,
			Category:    DiagCatRepository,
			Component:   "permissions",
			Location:    cleanPath,
			Message:     fmt.Sprintf("Read permission denied on workspace directory %q", cleanPath),
			Error:       errOpen.Error(),
			Remediation: "Ensure current process user has read permissions on workspace.",
		})
	} else {
		_ = f.Close()
		entries = append(entries, DiagnosticEntry{
			ID:        "repo-perm-ok",
			Timestamp: now,
			Severity:  DiagInfo,
			Category:  DiagCatRepository,
			Component: "permissions",
			Location:  cleanPath,
			Message:   fmt.Sprintf("Workspace directory %q is accessible and readable", cleanPath),
		})
	}

	// Git metadata check
	gitDir := filepath.Join(cleanPath, ".git")
	if gitInfo, errGit := os.Stat(gitDir); errGit == nil && gitInfo.IsDir() {
		entries = append(entries, DiagnosticEntry{
			ID:        "repo-git-01",
			Timestamp: now,
			Severity:  DiagInfo,
			Category:  DiagCatRepository,
			Component: "vcs",
			Location:  gitDir,
			Message:   "Git repository metadata directory (.git) detected",
		})
	}

	return entries
}

// CollectConfiguration inspects active configuration profile, schema, and overridden values.
func (e *DiagnosticEngine) CollectConfiguration(cfg *config.EffectiveConfig) []DiagnosticEntry {
	now := time.Now().UTC()
	var entries []DiagnosticEntry
	if cfg == nil {
		return entries
	}

	entries = append(entries, DiagnosticEntry{
		ID:        "cfg-profile-01",
		Timestamp: now,
		Severity:  DiagInfo,
		Category:  DiagCatConfiguration,
		Component: "config",
		Message:   fmt.Sprintf("Active configuration profile: %q", cfg.Profile()),
		Details: map[string]any{
			"profile": cfg.Profile(),
		},
	})

	allEntries := cfg.AllEntries()
	overriddenCount := 0
	for _, entry := range allEntries {
		if entry.Precedence > config.PrecedenceDefault {
			overriddenCount++
		}
	}

	entries = append(entries, DiagnosticEntry{
		ID:        "cfg-stats-01",
		Timestamp: now,
		Severity:  DiagInfo,
		Category:  DiagCatConfiguration,
		Component: "config",
		Message:   fmt.Sprintf("Total configuration properties: %d (%d explicitly overridden from defaults)", len(allEntries), overriddenCount),
		Details: map[string]any{
			"total_keys":      len(allEntries),
			"overridden_keys": overriddenCount,
		},
	})

	return entries
}

// CollectDependency inspects toolchain and external binary prerequisites (Go, Git).
func (e *DiagnosticEngine) CollectDependency(repoPath string) []DiagnosticEntry {
	now := time.Now().UTC()
	var entries []DiagnosticEntry

	// Check git executable
	gitPath, err := exec.LookPath("git")
	if err != nil {
		entries = append(entries, DiagnosticEntry{
			ID:          "dep-git-missing",
			Timestamp:   now,
			Severity:    DiagWarn,
			Category:    DiagCatDependency,
			Component:   "toolchain",
			Message:     "Git executable was not found on system PATH",
			Remediation: "Install Git and ensure it is available in system PATH.",
		})
	} else {
		entries = append(entries, DiagnosticEntry{
			ID:        "dep-git-ok",
			Timestamp: now,
			Severity:  DiagInfo,
			Category:  DiagCatDependency,
			Component: "toolchain",
			Message:   fmt.Sprintf("Git binary found at %s", gitPath),
		})
	}

	// Check Go compiler
	goPath, err := exec.LookPath("go")
	if err != nil {
		entries = append(entries, DiagnosticEntry{
			ID:        "dep-go-missing",
			Timestamp: now,
			Severity:  DiagInfo,
			Category:  DiagCatDependency,
			Component: "toolchain",
			Message:   "Go compiler was not found on system PATH (non-fatal)",
		})
	} else {
		entries = append(entries, DiagnosticEntry{
			ID:        "dep-go-ok",
			Timestamp: now,
			Severity:  DiagInfo,
			Category:  DiagCatDependency,
			Component: "toolchain",
			Message:   fmt.Sprintf("Go compiler found at %s", goPath),
		})
	}

	// Check module file if repoPath provided
	if repoPath != "" {
		goMod := filepath.Join(repoPath, "go.mod")
		if _, errMod := os.Stat(goMod); errMod == nil {
			entries = append(entries, DiagnosticEntry{
				ID:        "dep-gomod-ok",
				Timestamp: now,
				Severity:  DiagInfo,
				Category:  DiagCatDependency,
				Component: "manifest",
				Location:  goMod,
				Message:   "Go module manifest (go.mod) present",
			})
		}
	}

	return entries
}

// CollectPerformance inspects memory allocations and GC pause metrics.
func (e *DiagnosticEngine) CollectPerformance() []DiagnosticEntry {
	now := time.Now().UTC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	allocMB := float64(m.Alloc) / (1024 * 1024)
	sysMB := float64(m.Sys) / (1024 * 1024)

	var entries []DiagnosticEntry
	entries = append(entries, DiagnosticEntry{
		ID:        "perf-mem-01",
		Timestamp: now,
		Severity:  DiagInfo,
		Category:  DiagCatPerformance,
		Component: "memory",
		Message:   fmt.Sprintf("Allocated heap: %.2f MB, Sys memory: %.2f MB, GC cycles: %d", allocMB, sysMB, m.NumGC),
		Details: map[string]any{
			"alloc_mb": allocMB,
			"sys_mb":   sysMB,
			"num_gc":   m.NumGC,
		},
	})

	if allocMB > 1024 {
		entries = append(entries, DiagnosticEntry{
			ID:          "perf-mem-high",
			Timestamp:   now,
			Severity:    DiagWarn,
			Category:    DiagCatPerformance,
			Component:   "memory",
			Message:     fmt.Sprintf("High memory usage detected (%.2f MB allocated)", allocMB),
			Remediation: "Consider adjusting repository batch sizes or indexing memory limits.",
		})
	}

	return entries
}

// CollectRuntime inspects active goroutines and runtime execution condition.
func (e *DiagnosticEngine) CollectRuntime() []DiagnosticEntry {
	now := time.Now().UTC()
	goroutines := runtime.NumGoroutine()

	var entries []DiagnosticEntry
	entries = append(entries, DiagnosticEntry{
		ID:        "rt-goroutines-01",
		Timestamp: now,
		Severity:  DiagInfo,
		Category:  DiagCatRuntime,
		Component: "scheduler",
		Message:   fmt.Sprintf("Active goroutines: %d", goroutines),
		Details: map[string]any{
			"num_goroutine": goroutines,
		},
	})

	return entries
}

// Filter filters diagnostic entries by category and minimum severity.
func (e *DiagnosticEngine) Filter(entries []DiagnosticEntry, category DiagnosticCategory, minSev DiagnosticSeverity) []DiagnosticEntry {
	sevWeight := func(s DiagnosticSeverity) int {
		switch s {
		case DiagInfo:
			return 10
		case DiagWarn:
			return 20
		case DiagError:
			return 30
		case DiagCritical:
			return 40
		default:
			return 0
		}
	}

	minWeight := sevWeight(minSev)
	var res []DiagnosticEntry

	for _, it := range entries {
		if category != "" && it.Category != category {
			continue
		}
		if minWeight > 0 && sevWeight(it.Severity) < minWeight {
			continue
		}
		res = append(res, it)
	}

	// Deterministic sort by severity descending, category, then ID
	sort.Slice(res, func(i, j int) bool {
		wI, wJ := sevWeight(res[i].Severity), sevWeight(res[j].Severity)
		if wI != wJ {
			return wI > wJ
		}
		if res[i].Category != res[j].Category {
			return res[i].Category < res[j].Category
		}
		return res[i].ID < res[j].ID
	})

	return res
}
