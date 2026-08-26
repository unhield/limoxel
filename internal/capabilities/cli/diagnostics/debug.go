package diagnostics

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/cli/config"
)

// Tracer records lightweight hierarchical operational trace spans.
type Tracer struct {
	mu      sync.RWMutex
	spans   []TraceSpan
	enabled bool
	counter int64
}

// NewTracer constructs an initialized Tracer.
func NewTracer(enabled bool) *Tracer {
	return &Tracer{
		spans:   make([]TraceSpan, 0, 100),
		enabled: enabled,
	}
}

// StartSpan begins an operational trace span and records its start timestamp.
func (t *Tracer) StartSpan(name, component string, parentID string, tags map[string]string) *TraceSpan {
	if t == nil || !t.enabled {
		return &TraceSpan{Name: name, Component: component, StartTime: time.Now()}
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	t.counter++
	spanID := fmt.Sprintf("span-%d", t.counter)

	cleanTags := make(map[string]string, len(tags))
	for k, v := range tags {
		if config.IsSecretKey(k) {
			cleanTags[k] = config.MaskedValueConstant
		} else {
			cleanTags[k] = v
		}
	}

	span := TraceSpan{
		ID:        spanID,
		ParentID:  parentID,
		Name:      name,
		Component: component,
		StartTime: time.Now().UTC(),
		Tags:      cleanTags,
	}

	t.spans = append(t.spans, span)
	return &span
}

// EndSpan finalizes a span, calculating its duration in milliseconds.
func (t *Tracer) EndSpan(span *TraceSpan) {
	if t == nil || span == nil || !t.enabled {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	endTime := time.Now().UTC()
	span.EndTime = endTime
	span.DurationMs = float64(endTime.Sub(span.StartTime).Microseconds()) / 1000.0

	// Update in recorded spans list
	for i := range t.spans {
		if t.spans[i].ID == span.ID {
			t.spans[i].EndTime = span.EndTime
			t.spans[i].DurationMs = span.DurationMs
			break
		}
	}
}

// Spans returns a copy of all recorded trace spans.
func (t *Tracer) Spans() []TraceSpan {
	if t == nil {
		return nil
	}
	t.mu.RLock()
	defer t.mu.RUnlock()

	res := make([]TraceSpan, len(t.spans))
	copy(res, t.spans)
	return res
}

// Clear resets recorded trace spans.
func (t *Tracer) Clear() {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.spans = t.spans[:0]
	t.counter = 0
}

// DumpOperationalState creates a comprehensive operational state dump with secret protection.
func DumpOperationalState(logger *Logger, profiler *Profiler, cfg *config.EffectiveConfig, repoPath string) map[string]any {
	now := time.Now().UTC()

	var resStats ResourceStats
	if profiler != nil {
		resStats = profiler.GetResourceStats()
	} else {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		resStats = ResourceStats{
			NumGoroutine: runtime.NumGoroutine(),
			AllocMB:      float64(m.Alloc) / (1024 * 1024),
			TotalAllocMB: float64(m.TotalAlloc) / (1024 * 1024),
			SysMB:        float64(m.Sys) / (1024 * 1024),
			NumGC:        m.NumGC,
			CPUCount:     runtime.NumCPU(),
			GoVersion:    runtime.Version(),
			OS:           runtime.GOOS,
			Arch:         runtime.GOARCH,
		}
	}

	state := map[string]any{
		"timestamp":  now,
		"pid":        os.Getpid(),
		"repo_path":  repoPath,
		"resources":  resStats,
		"goroutines": runtime.NumGoroutine(),
	}

	if cfg != nil {
		allEntries := cfg.AllEntries()
		redactedConfig := make(map[string]any, len(allEntries))
		for _, e := range allEntries {
			if e.IsSecret || config.IsSecretKey(e.Key) {
				redactedConfig[e.Key] = config.MaskedValueConstant
			} else {
				redactedConfig[e.Key] = e.Value
			}
		}
		state["config"] = map[string]any{
			"profile": cfg.Profile(),
			"entries": redactedConfig,
		}
	}

	if logger != nil {
		recentLogs := logger.GetRecentLogs(50)
		state["recent_logs_count"] = len(recentLogs)
		state["recent_logs"] = recentLogs
	}

	return state
}

// FormatSpansText formats trace spans into a clean visual tree.
func FormatSpansText(spans []TraceSpan) string {
	if len(spans) == 0 {
		return "No execution trace spans recorded."
	}

	sort.Slice(spans, func(i, j int) bool {
		return spans[i].StartTime.Before(spans[j].StartTime)
	})

	var sb strings.Builder
	for _, s := range spans {
		prefix := "• "
		if s.ParentID != "" {
			prefix = "  └─ "
		}
		fmt.Fprintf(&sb, "%s[%s] %s (%.2f ms)\n", prefix, s.Component, s.Name, s.DurationMs)
		var tagKeys []string
		for k := range s.Tags {
			tagKeys = append(tagKeys, k)
		}
		sort.Strings(tagKeys)
		for _, k := range tagKeys {
			fmt.Fprintf(&sb, "     tag: %s=%s\n", k, s.Tags[k])
		}
	}

	return sb.String()
}
