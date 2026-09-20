package tooling

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/unhield/limoxel/plugin"
)

// BuildOptions configures the compilation of a plugin project.
type BuildOptions struct {
	ProjectDir string        `json:"project_dir"`
	OutputFile string        `json:"output_file"`
	GoBinary   string        `json:"go_binary,omitempty"`
	Env        []string      `json:"env,omitempty"`
	Timeout    time.Duration `json:"timeout,omitempty"`
}

// BuildResult encapsulates output artifacts and metrics from a build.
type BuildResult struct {
	BinaryPath   string          `json:"binary_path"`
	ManifestPath string          `json:"manifest_path"`
	Manifest     plugin.Manifest `json:"manifest"`
	Duration     time.Duration   `json:"duration"`
	Output       string          `json:"output,omitempty"`
}

// Builder orchestrates validation and compilation of plugin source projects.
type Builder struct{}

// NewBuilder constructs a Builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// Build validates the plugin manifest and compiles the plugin binary.
func (b *Builder) Build(ctx context.Context, opts BuildOptions) (*BuildResult, error) {
	projDir := filepath.Clean(opts.ProjectDir)
	if projDir == "" || projDir == "." {
		projDir = "."
	}

	manifestPath := filepath.Join(projDir, "plugin.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("plugin manifest not found at %s: %w", manifestPath, err)
	}

	var manifest plugin.Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("invalid plugin manifest %s: %w", manifestPath, err)
	}

	if strings.TrimSpace(manifest.ID) == "" {
		return nil, errors.New("plugin manifest must declare a valid non-empty 'id'")
	}

	outPath := opts.OutputFile
	if strings.TrimSpace(outPath) == "" {
		binaryName := manifest.Name
		if binaryName == "" {
			binaryName = "plugin"
		}
		binaryName = strings.ToLower(strings.ReplaceAll(binaryName, " ", "-"))
		if filepath.Ext(binaryName) == "" && os.Getenv("GOOS") == "windows" {
			binaryName += ".exe"
		}
		outPath = filepath.Join(projDir, "bin", binaryName)
	}
	outPath = filepath.Clean(outPath)

	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	goBin := strings.TrimSpace(opts.GoBinary)
	if goBin == "" {
		goBin = "go"
	} else {
		if strings.ContainsAny(goBin, "\r\n\t;&|`$<>") {
			return nil, fmt.Errorf("invalid go_binary specification: %q", goBin)
		}
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	bCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	startTime := time.Now()
	cmd := exec.CommandContext(bCtx, goBin, "build", "-o", outPath, ".")
	cmd.Dir = projDir
	if len(opts.Env) > 0 {
		cmd.Env = append(os.Environ(), opts.Env...)
	}

	combinedOut, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	if err != nil {
		return nil, fmt.Errorf("build failed (%w): %s", err, string(combinedOut))
	}

	return &BuildResult{
		BinaryPath:   outPath,
		ManifestPath: manifestPath,
		Manifest:     manifest,
		Duration:     duration,
		Output:       string(combinedOut),
	}, nil
}
