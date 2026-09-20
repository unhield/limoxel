package tooling

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/unhield/limoxel/plugin/templates"
)

var idRegex = regexp.MustCompile(`^[a-zA-Z0-9][-a-zA-Z0-9_.]*[a-zA-Z0-9]$`)

// GenerateOptions configures the generation of a new plugin scaffold.
type GenerateOptions struct {
	Template    templates.TemplateType `json:"template"`
	PluginID    string                 `json:"plugin_id"`
	PluginName  string                 `json:"plugin_name"`
	Version     string                 `json:"version"`
	Publisher   string                 `json:"publisher"`
	Description string                 `json:"description"`
	OutputDir   string                 `json:"output_dir"`
	Overwrite   bool                   `json:"overwrite"`
}

// Validate ensures options are secure and semantically sound.
func (o *GenerateOptions) Validate() error {
	id := strings.TrimSpace(o.PluginID)
	if id == "" {
		return errors.New("plugin ID cannot be empty")
	}
	if !idRegex.MatchString(id) {
		return fmt.Errorf("invalid plugin ID %q: must contain only alphanumeric, dot, dash, or underscore", id)
	}

	outDir := strings.TrimSpace(o.OutputDir)
	if outDir == "" {
		return errors.New("output directory cannot be empty")
	}

	// Path traversal protection
	cleanOut := filepath.Clean(outDir)
	if strings.Contains(o.OutputDir, "..") || cleanOut == ".." || strings.HasPrefix(cleanOut, ".."+string(filepath.Separator)) || strings.HasPrefix(cleanOut, "../") {
		return fmt.Errorf("path traversal attempt detected in output directory: %q", o.OutputDir)
	}
	if strings.HasPrefix(outDir, `\\`) || strings.HasPrefix(outDir, `//`) {
		return fmt.Errorf("UNC network paths prohibited in output directory: %q", o.OutputDir)
	}

	return nil
}

// Generator scaffolds plugin projects using approved templates.
type Generator struct{}

// NewGenerator constructs a Generator.
func NewGenerator() *Generator {
	return &Generator{}
}

// Generate creates the plugin project structure on disk.
func (g *Generator) Generate(opts GenerateOptions) error {
	if err := opts.Validate(); err != nil {
		return fmt.Errorf("generator validation failed: %w", err)
	}

	targetDir := filepath.Clean(opts.OutputDir)
	if stat, err := os.Stat(targetDir); err == nil {
		if !opts.Overwrite {
			// Check if directory is non-empty
			entries, readErr := os.ReadDir(targetDir)
			if readErr == nil && len(entries) > 0 {
				return fmt.Errorf("target directory %s already exists and is not empty (use Overwrite=true)", targetDir)
			}
		}
		if !stat.IsDir() {
			return fmt.Errorf("target path %s exists and is a file", targetDir)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to inspect target directory: %w", err)
	}

	tmplVars := templates.TemplateVariables{
		PluginID:    opts.PluginID,
		PluginName:  opts.PluginName,
		Version:     opts.Version,
		Publisher:   opts.Publisher,
		Description: opts.Description,
	}

	files, err := templates.RenderTemplate(opts.Template, tmplVars)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory %s: %w", targetDir, err)
	}

	for relPath, content := range files {
		cleanRel := filepath.Clean(relPath)
		if strings.HasPrefix(cleanRel, "..") {
			return fmt.Errorf("illegal relative file path in template: %q", relPath)
		}

		fullPath := filepath.Join(targetDir, cleanRel)
		parent := filepath.Dir(fullPath)
		if err := os.MkdirAll(parent, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", parent, err)
		}

		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", fullPath, err)
		}
	}

	return nil
}
