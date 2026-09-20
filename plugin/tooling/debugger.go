package tooling

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/unhield/limoxel/plugin"
)

// DebugOptions configures a local plugin diagnostics session.
type DebugOptions struct {
	PluginDir string        `json:"plugin_dir"`
	Verbose   bool          `json:"verbose"`
	Timeout   time.Duration `json:"timeout,omitempty"`
}

// DiagnosticItem represents a specific finding or status in a debug session.
type DiagnosticItem struct {
	Level   string `json:"level"`
	Scope   string `json:"scope"`
	Message string `json:"message"`
}

// DebugSession encapsulates diagnostic observations and runtime validation.
type DebugSession struct {
	SessionID   string           `json:"session_id"`
	PluginDir   string           `json:"plugin_dir"`
	Manifest    *plugin.Manifest `json:"manifest,omitempty"`
	Status      string           `json:"status"`
	Diagnostics []DiagnosticItem `json:"diagnostics"`
	Duration    time.Duration    `json:"duration"`
}

// Debugger provides local inspection, validation, and diagnostics for plugin developers.
type Debugger struct{}

// NewDebugger constructs a Debugger.
func NewDebugger() *Debugger {
	return &Debugger{}
}

// Diagnose evaluates a plugin project directory for structural, manifest, and lifecycle correctness.
func (d *Debugger) Diagnose(ctx context.Context, opts DebugOptions) (*DebugSession, error) {
	start := time.Now()
	pDir := filepath.Clean(opts.PluginDir)
	if pDir == "" || pDir == "." {
		pDir = "."
	}

	sessionID := fmt.Sprintf("debug-%d", start.UnixNano())
	session := &DebugSession{
		SessionID:   sessionID,
		PluginDir:   pDir,
		Status:      "PASS",
		Diagnostics: make([]DiagnosticItem, 0),
	}

	addDiag := func(level, scope, msg string) {
		session.Diagnostics = append(session.Diagnostics, DiagnosticItem{
			Level:   level,
			Scope:   scope,
			Message: msg,
		})
		if level == "ERROR" {
			session.Status = "FAIL"
		}
	}

	// 1. Directory inspection
	stat, err := os.Stat(pDir)
	if err != nil {
		addDiag("ERROR", "filesystem", fmt.Sprintf("failed to access plugin directory %s: %v", pDir, err))
		session.Duration = time.Since(start)
		return session, nil
	}
	if !stat.IsDir() {
		addDiag("ERROR", "filesystem", fmt.Sprintf("path %s is not a directory", pDir))
		session.Duration = time.Since(start)
		return session, nil
	}
	addDiag("INFO", "filesystem", fmt.Sprintf("plugin directory verified at %s", pDir))

	// 2. Manifest validation
	manifestPath := filepath.Join(pDir, "plugin.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		addDiag("ERROR", "manifest", fmt.Sprintf("missing plugin.json: %v", err))
	} else {
		var manifest plugin.Manifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			addDiag("ERROR", "manifest", fmt.Sprintf("malformed plugin.json: %v", err))
		} else {
			session.Manifest = &manifest
			addDiag("INFO", "manifest", fmt.Sprintf("parsed manifest ID: %s, Version: %s", manifest.ID, manifest.Version))

			if strings.TrimSpace(manifest.ID) == "" {
				addDiag("ERROR", "manifest", "manifest 'id' field cannot be empty")
			}
			if strings.TrimSpace(manifest.Name) == "" {
				addDiag("WARN", "manifest", "manifest 'name' field is empty")
			}
			if strings.TrimSpace(manifest.Version) == "" {
				addDiag("ERROR", "manifest", "manifest 'version' field cannot be empty")
			}
			if len(manifest.Capabilities) == 0 {
				addDiag("WARN", "manifest", "no capabilities declared in manifest")
			}

			// 3. Entrypoint check
			if manifest.Entrypoint != "" {
				epClean := filepath.Clean(manifest.Entrypoint)
				if filepath.IsAbs(epClean) || filepath.VolumeName(epClean) != "" || strings.HasPrefix(manifest.Entrypoint, "/") || strings.HasPrefix(manifest.Entrypoint, "\\") {
					addDiag("ERROR", "entrypoint", fmt.Sprintf("entrypoint cannot be an absolute or volume path: %s", manifest.Entrypoint))
				} else if epClean == ".." || strings.HasPrefix(epClean, ".."+string(filepath.Separator)) || strings.HasPrefix(epClean, "../") {
					addDiag("ERROR", "entrypoint", fmt.Sprintf("entrypoint traverses outside plugin directory: %s", manifest.Entrypoint))
				} else {
					epPath := filepath.Join(pDir, epClean)
					if _, err := os.Stat(epPath); err != nil {
						addDiag("WARN", "entrypoint", fmt.Sprintf("declared entrypoint file not found: %s", manifest.Entrypoint))
					} else {
						addDiag("INFO", "entrypoint", fmt.Sprintf("entrypoint verified: %s", manifest.Entrypoint))
					}
				}
			}
		}
	}

	session.Duration = time.Since(start)
	return session, nil
}
