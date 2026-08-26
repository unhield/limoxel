package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/cli/diagnostics"
)

// RegisterDiagnosticCommands binds all operational logging, diagnostics, health, and debug commands.
func RegisterDiagnosticCommands(app *App) {
	if app == nil {
		return
	}

	// 1. limoxel log / logs
	logCmd := NewCommand(
		"log",
		"Inspect operational log stream and recent events",
		"limoxel log [options]",
		CategoryDiagnostics,
		handleLogCommand,
	).AddAlias("logs").
		AddOption("limit", "l", "Maximum number of recent log entries to retrieve", "50").
		AddOption("level", "", "Filter log entries by minimum level (debug, info, warn, error, critical)", "").
		AddOption("format", "f", "Output format (text, json, yaml, toml, xml)", "text")
	app.RegisterCommand(logCmd)

	// 2. limoxel diag / diagnostics
	diagCmd := NewCommand(
		"diag",
		"Run operational system, repository, config, dependency, and performance diagnostics",
		"limoxel diag [options]",
		CategoryDiagnostics,
		handleDiagCommand,
	).AddAlias("diagnostics").
		AddOption("category", "", "Filter by diagnostic category (system, repository, configuration, dependency, performance, runtime)", "").
		AddOption("severity", "", "Filter by minimum severity (info, warn, error, critical)", "").
		AddOption("repo", "r", "Target repository directory path", ".").
		AddOption("format", "f", "Output format (text, json, yaml, toml, xml)", "text")
	app.RegisterCommand(diagCmd)

	// 3. limoxel health
	healthCmd := NewCommand(
		"health",
		"Check operational health and readiness of Limoxel runtime components",
		"limoxel health [options]",
		CategoryDiagnostics,
		handleHealthCommand,
	).AddOption("repo", "r", "Target repository directory path", ".").
		AddOption("format", "f", "Output format (text, json, yaml, toml, xml)", "text")
	app.RegisterCommand(healthCmd)

	// 4. limoxel debug
	debugCmd := NewCommand(
		"debug",
		"Operational debug tools, execution tracing, and state dumps",
		"limoxel debug [trace|dump] [options]",
		CategoryDiagnostics,
		handleDebugCommand,
	).AddOption("repo", "r", "Target repository directory path", ".").
		AddOption("format", "f", "Output format (text, json, yaml, toml, xml)", "text")
	app.RegisterCommand(debugCmd)

	// 5. limoxel profile
	profileCmd := NewCommand(
		"profile",
		"Runtime profiling and resource observation tools",
		"limoxel profile [heap|stats] [target] [options]",
		CategoryDiagnostics,
		handleProfileCommand,
	).AddOption("format", "f", "Output format (text, json, yaml, toml, xml)", "text")
	app.RegisterCommand(profileCmd)
}

// 1. handleLogCommand
func handleLogCommand(ctx *Context, flags *Flags) error {
	dm, err := ctx.EnsureDiagnosticsManager()
	if err != nil {
		return err
	}

	limit := flags.Int("limit", 50)
	recent := dm.Logger().GetRecentLogs(limit)

	if lvlStr := flags.String("level", ""); lvlStr != "" {
		if targetLvl, err := diagnostics.ParseLogLevel(lvlStr); err == nil {
			var filtered []diagnostics.LogEvent
			for _, ev := range recent {
				if ev.Level >= targetLvl {
					filtered = append(filtered, ev)
				}
			}
			recent = filtered
		}
	}

	format := flags.Format()
	if format == FormatJSON || format == FormatYAML || format == FormatTOML || format == FormatXML || format == FormatCSV {
		return ctx.Formatter().RenderStructured(recent)
	}

	ctx.Formatter().RenderSection("Operational Log Buffer")
	if len(recent) == 0 {
		fmt.Fprintln(ctx.Formatter().Stdout(), "(No operational log events recorded in memory)")
		return nil
	}

	headers := []string{"Timestamp", "Level", "Component", "Message"}
	var rows [][]string
	for _, ev := range recent {
		rows = append(rows, []string{
			ev.Timestamp.Format("15:04:05.000"),
			ev.LevelName,
			ev.Component,
			ev.Message,
		})
	}
	return ctx.Formatter().RenderTable(headers, rows)
}

// 2. handleDiagCommand
func handleDiagCommand(ctx *Context, flags *Flags) error {
	dm, err := ctx.EnsureDiagnosticsManager()
	if err != nil {
		return err
	}

	category := diagnostics.DiagnosticCategory(strings.ToLower(flags.String("category", "")))
	minSev := diagnostics.DiagnosticSeverity(strings.ToLower(flags.String("severity", "")))

	entries := dm.RunDiagnostics(diagnostics.DiagnosticOptions{
		RepoPath:    flags.RepoRoot(),
		Category:    category,
		MinSeverity: minSev,
	})

	format := flags.Format()
	if format == FormatJSON || format == FormatYAML || format == FormatTOML || format == FormatXML || format == FormatCSV {
		return ctx.Formatter().RenderStructured(entries)
	}

	ctx.Formatter().RenderSection("Operational Diagnostics")
	if len(entries) == 0 {
		fmt.Fprintln(ctx.Formatter().Stdout(), "All diagnostic checks nominal. No issues or observations found.")
		return nil
	}

	headers := []string{"Severity", "Category", "Component", "Observation"}
	var rows [][]string
	for _, e := range entries {
		msg := e.Message
		if e.Remediation != "" {
			msg += fmt.Sprintf(" (Remediation: %s)", e.Remediation)
		}
		rows = append(rows, []string{
			strings.ToUpper(string(e.Severity)),
			string(e.Category),
			e.Component,
			msg,
		})
	}
	return ctx.Formatter().RenderTable(headers, rows)
}

// 3. handleHealthCommand
func handleHealthCommand(ctx *Context, flags *Flags) error {
	dm, err := ctx.EnsureDiagnosticsManager()
	if err != nil {
		return err
	}

	report := dm.RunHealth()

	format := flags.Format()
	if format == FormatJSON || format == FormatYAML || format == FormatTOML || format == FormatXML || format == FormatCSV {
		return ctx.Formatter().RenderStructured(report)
	}

	ctx.Formatter().RenderSection(fmt.Sprintf("Operational Health Report (Status: %s)", strings.ToUpper(string(report.OverallStatus))))

	headers := []string{"Check", "Status", "Latency", "Message"}
	var rows [][]string
	for _, chk := range report.Checks {
		rows = append(rows, []string{
			chk.Name,
			strings.ToUpper(string(chk.Status)),
			fmt.Sprintf("%.2f ms", chk.LatencyMs),
			chk.Message,
		})
	}

	if err := ctx.Formatter().RenderTable(headers, rows); err != nil {
		return err
	}

	if report.OverallStatus == diagnostics.HealthFailed || report.OverallStatus == diagnostics.HealthUnavailable {
		return fmt.Errorf("operational health check reported %s status", report.OverallStatus)
	}
	return nil
}

// 4. handleDebugCommand
func handleDebugCommand(ctx *Context, flags *Flags) error {
	dm, err := ctx.EnsureDiagnosticsManager()
	if err != nil {
		return err
	}

	sub := flags.Arg(0)
	if sub == "trace" {
		spans := dm.Tracer().Spans()
		format := flags.Format()
		if format == FormatJSON || format == FormatYAML || format == FormatTOML || format == FormatXML {
			return ctx.Formatter().RenderStructured(spans)
		}

		ctx.Formatter().RenderSection("Execution Trace Spans")
		fmt.Fprintln(ctx.Formatter().Stdout(), diagnostics.FormatSpansText(spans))
		return nil
	}

	dump := dm.DumpState()
	format := flags.Format()
	if format == FormatJSON || format == FormatYAML || format == FormatTOML || format == FormatXML {
		return ctx.Formatter().RenderStructured(dump)
	}

	ctx.Formatter().RenderSection("Operational State Dump")
	w := ctx.Formatter().Stdout()
	fmt.Fprintf(w, "Timestamp:    %v\n", dump["timestamp"])
	fmt.Fprintf(w, "Process PID:  %v\n", dump["pid"])
	fmt.Fprintf(w, "Workspace:    %v\n", dump["repo_path"])
	fmt.Fprintf(w, "Goroutines:   %v\n", dump["goroutines"])
	if res, ok := dump["resources"].(diagnostics.ResourceStats); ok {
		fmt.Fprintf(w, "Memory:       Alloc=%.2fMB, Sys=%.2fMB, GCs=%d\n", res.AllocMB, res.SysMB, res.NumGC)
		fmt.Fprintf(w, "Environment:  %s/%s (Go %s, %d CPUs)\n", res.OS, res.Arch, res.GoVersion, res.CPUCount)
	}
	return nil
}

// 5. handleProfileCommand
func handleProfileCommand(ctx *Context, flags *Flags) error {
	dm, err := ctx.EnsureDiagnosticsManager()
	if err != nil {
		return err
	}

	sub := flags.Arg(0)
	switch sub {
	case "heap", "mem":
		target := flags.Arg(1)
		if target == "" {
			target = "heap.pprof"
		}
		res, err := dm.Profiler().WriteHeapProfile(target)
		if err != nil {
			return err
		}
		format := flags.Format()
		if format == FormatJSON || format == FormatYAML || format == FormatTOML || format == FormatXML {
			return ctx.Formatter().RenderStructured(res)
		}
		fmt.Fprintf(ctx.Formatter().Stdout(), "[SUCCESS] Heap profile written to %s (size: %d bytes, %.2f ms)\n", res.FilePath, res.Size, res.DurationMs)
		return nil

	case "stats", "resources":
		stats := dm.Profiler().GetResourceStats()
		format := flags.Format()
		if format == FormatJSON || format == FormatYAML || format == FormatTOML || format == FormatXML {
			return ctx.Formatter().RenderStructured(stats)
		}
		ctx.Formatter().RenderSection("Runtime Resource Statistics")
		w := ctx.Formatter().Stdout()
		fmt.Fprintf(w, "Goroutines:       %d\n", stats.NumGoroutine)
		fmt.Fprintf(w, "Heap Allocated:   %.2f MB\n", stats.AllocMB)
		fmt.Fprintf(w, "Total Allocated:  %.2f MB\n", stats.TotalAllocMB)
		fmt.Fprintf(w, "Sys Memory:       %.2f MB\n", stats.SysMB)
		fmt.Fprintf(w, "GC Cycles:        %d\n", stats.NumGC)
		fmt.Fprintf(w, "GC Pause Total:   %.2f ms\n", stats.PauseTotalMs)
		fmt.Fprintf(w, "Host Environment: %s/%s (%d CPUs, %s)\n", stats.OS, stats.Arch, stats.CPUCount, stats.GoVersion)
		return nil

	default:
		dur, _ := dm.Profiler().MeasureOperation("resource_sampling", func() error {
			time.Sleep(10 * time.Millisecond)
			return nil
		})
		fmt.Fprintf(ctx.Formatter().Stdout(), "Profiling subsystem online. Baseline sampling latency: %v\nUse 'limoxel profile heap <path>' or 'limoxel profile stats' for specific operations.\n", dur)
		return nil
	}
}
