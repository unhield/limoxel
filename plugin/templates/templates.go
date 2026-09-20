package templates

import (
	"encoding/json"
	"fmt"
	"strings"
)

// TemplateType identifies a canonical plugin template scaffold.
type TemplateType string

const (
	TemplateBasic        TemplateType = "basic"
	TemplateCLI          TemplateType = "cli"
	TemplateRepository   TemplateType = "repository"
	TemplateIntelligence TemplateType = "intelligence"
	TemplateEnterprise   TemplateType = "enterprise"
)

// TemplateDescriptor provides metadata describing an available template.
type TemplateDescriptor struct {
	Type                 TemplateType `json:"type"`
	Name                 string       `json:"name"`
	Description          string       `json:"description"`
	RequiredCapabilities []string     `json:"required_capabilities"`
}

// TemplateVariables configures parameters interpolated into generated template files.
type TemplateVariables struct {
	PluginID    string `json:"plugin_id"`
	PluginName  string `json:"plugin_name"`
	Version     string `json:"version"`
	Publisher   string `json:"publisher"`
	Description string `json:"description"`
}

// Normalize applies defaults to template variables.
func (v *TemplateVariables) Normalize() {
	if strings.TrimSpace(v.PluginID) == "" {
		v.PluginID = "com.example.myplugin"
	}
	if strings.TrimSpace(v.PluginName) == "" {
		v.PluginName = "MyPlugin"
	}
	if strings.TrimSpace(v.Version) == "" {
		v.Version = "1.0.0"
	}
	if strings.TrimSpace(v.Publisher) == "" {
		v.Publisher = "Plugin Author"
	}
	if strings.TrimSpace(v.Description) == "" {
		v.Description = "A Limoxel plugin built with the Plugin SDK."
	}
}

// ListTemplates enumerates all available development template descriptors.
func ListTemplates() []TemplateDescriptor {
	return []TemplateDescriptor{
		{
			Type:                 TemplateBasic,
			Name:                 "Basic Plugin",
			Description:          "Minimal canonical starter plugin demonstrating lifecycle, metadata, and clean shutdown.",
			RequiredCapabilities: []string{"plugin.lifecycle"},
		},
		{
			Type:                 TemplateCLI,
			Name:                 "CLI Plugin",
			Description:          "Plugin exposing command-line utilities and tool commands via the Plugin SDK.",
			RequiredCapabilities: []string{"cli.commands", "logging.structured"},
		},
		{
			Type:                 TemplateRepository,
			Name:                 "Repository Plugin",
			Description:          "Plugin demonstrating workspace file inspection, metadata querying, and repository events.",
			RequiredCapabilities: []string{"repository.metadata", "repository.files", "events.pubsub"},
		},
		{
			Type:                 TemplateIntelligence,
			Name:                 "Intelligence Plugin",
			Description:          "Plugin consuming symbols, multi-domain search, Knowledge Graph, and engineering analysis.",
			RequiredCapabilities: []string{"symbols.api", "search.api", "graph.api", "analysis.api"},
		},
		{
			Type:                 TemplateEnterprise,
			Name:                 "Enterprise Developer Scaffold",
			Description:          "Developer project scaffold demonstrating structured configuration, logging, and health checks.",
			RequiredCapabilities: []string{"logging.structured", "configuration.audit"},
		},
	}
}

// RenderTemplate generates a map of relative file paths to file contents for the requested template.
func RenderTemplate(t TemplateType, vars TemplateVariables) (map[string]string, error) {
	vars.Normalize()

	switch t {
	case TemplateBasic:
		return renderBasic(vars), nil
	case TemplateCLI:
		return renderCLI(vars), nil
	case TemplateRepository:
		return renderRepository(vars), nil
	case TemplateIntelligence:
		return renderIntelligence(vars), nil
	case TemplateEnterprise:
		return renderEnterprise(vars), nil
	default:
		return nil, fmt.Errorf("unknown template type: %q", t)
	}
}

func renderBasic(v TemplateVariables) map[string]string {
	manifest := map[string]any{
		"schema_version": "1.0.0",
		"id":             v.PluginID,
		"name":           v.PluginName,
		"version":        v.Version,
		"publisher":      v.Publisher,
		"description":    v.Description,
		"entrypoint":     "main.go",
		"capabilities": []map[string]string{
			{"name": "plugin.lifecycle", "version": "1.0.0"},
		},
	}
	manifestJSON, _ := json.MarshalIndent(manifest, "", "  ")

	mainGo := fmt.Sprintf(`package main

import (
	"context"
	"fmt"

	"github.com/unhield/limoxel/plugin"
)

type %sPlugin struct {
	*plugin.BasePlugin
}

func NewPlugin() plugin.Plugin {
	manifest := plugin.Manifest{
		ID:      "%s",
		Name:    "%s",
		Version: "%s",
	}
	return &%sPlugin{
		BasePlugin: plugin.NewBasePlugin("%s", manifest),
	}
}

func (p *%sPlugin) Start(ctx context.Context) error {
	if err := p.BasePlugin.Start(ctx); err != nil {
		return err
	}
	p.GetContext().Logger().Info("%s started successfully")
	return nil
}

func (p *%sPlugin) Stop(ctx context.Context) error {
	p.GetContext().Logger().Info("%s stopping")
	return p.BasePlugin.Stop(ctx)
}

func main() {
	fmt.Println("%s plugin loaded")
}
`, toPascal(v.PluginName), v.PluginID, v.PluginName, v.Version, toPascal(v.PluginName), v.PluginID, toPascal(v.PluginName), v.PluginName, toPascal(v.PluginName), v.PluginName, v.PluginName)

	readme := fmt.Sprintf(`# %s

%s

## Overview
Built with the Limoxel Plugin SDK (Basic Template).

## Requirements
- Go 1.22+
- Limoxel Host
`, v.PluginName, v.Description)

	return map[string]string{
		"plugin.json": string(manifestJSON),
		"main.go":     mainGo,
		"README.md":   readme,
	}
}

func renderCLI(v TemplateVariables) map[string]string {
	manifest := map[string]any{
		"schema_version": "1.0.0",
		"id":             v.PluginID,
		"name":           v.PluginName,
		"version":        v.Version,
		"publisher":      v.Publisher,
		"description":    v.Description,
		"entrypoint":     "main.go",
		"capabilities": []map[string]string{
			{"name": "cli.commands", "version": "1.0.0"},
			{"name": "logging.structured", "version": "1.0.0"},
		},
	}
	manifestJSON, _ := json.MarshalIndent(manifest, "", "  ")

	mainGo := fmt.Sprintf(`package main

import (
	"context"
	"fmt"

	"github.com/unhield/limoxel/plugin"
)

type %sCLIPlugin struct {
	*plugin.BasePlugin
}

func NewPlugin() plugin.Plugin {
	manifest := plugin.Manifest{
		ID:      "%s",
		Name:    "%s",
		Version: "%s",
	}
	return &%sCLIPlugin{
		BasePlugin: plugin.NewBasePlugin("%s", manifest),
	}
}

func (p *%sCLIPlugin) ExecuteCommand(ctx context.Context, cmd string, args []string) (string, error) {
	p.GetContext().Logger().Info("executing cli command", "cmd", cmd)
	switch cmd {
	case "status":
		return "operational", nil
	case "version":
		return "%s", nil
	default:
		return "", fmt.Errorf("unknown command: %%s", cmd)
	}
}

func main() {
	fmt.Println("%s CLI tool initialized")
}
`, toPascal(v.PluginName), v.PluginID, v.PluginName, v.Version, toPascal(v.PluginName), v.PluginID, toPascal(v.PluginName), v.Version, v.PluginName)

	readme := fmt.Sprintf(`# %s (CLI Plugin)

%s
`, v.PluginName, v.Description)

	return map[string]string{
		"plugin.json": string(manifestJSON),
		"main.go":     mainGo,
		"README.md":   readme,
	}
}

func renderRepository(v TemplateVariables) map[string]string {
	manifest := map[string]any{
		"schema_version": "1.0.0",
		"id":             v.PluginID,
		"name":           v.PluginName,
		"version":        v.Version,
		"publisher":      v.Publisher,
		"description":    v.Description,
		"entrypoint":     "main.go",
		"capabilities": []map[string]string{
			{"name": "repository.metadata", "version": "1.0.0"},
			{"name": "repository.files", "version": "1.0.0"},
			{"name": "events.pubsub", "version": "1.0.0"},
		},
	}
	manifestJSON, _ := json.MarshalIndent(manifest, "", "  ")

	mainGo := fmt.Sprintf(`package main

import (
	"context"
	"fmt"

	"github.com/unhield/limoxel/plugin"
	"github.com/unhield/limoxel/plugin/api"
)

type %sRepoPlugin struct {
	*plugin.BasePlugin
	sub api.Subscription
}

func NewPlugin() plugin.Plugin {
	manifest := plugin.Manifest{
		ID:      "%s",
		Name:    "%s",
		Version: "%s",
	}
	return &%sRepoPlugin{
		BasePlugin: plugin.NewBasePlugin("%s", manifest),
	}
}

func (p *%sRepoPlugin) Start(ctx context.Context) error {
	if err := p.BasePlugin.Start(ctx); err != nil {
		return err
	}
	repo := p.GetContext().Repository()
	meta, err := repo.Metadata(ctx)
	if err == nil && meta != nil {
		p.GetContext().Logger().Info("repository inspect", "name", meta.Name)
	}

	sub, err := p.GetContext().Events().Subscribe(ctx, "repository.*", func(c context.Context, evt api.Event) error {
		p.GetContext().Logger().Info("repository event received", "type", evt.Type)
		return nil
	})
	if err == nil {
		p.sub = sub
	}
	return nil
}

func (p *%sRepoPlugin) Stop(ctx context.Context) error {
	if p.sub != nil {
		_ = p.sub.Unsubscribe()
	}
	return p.BasePlugin.Stop(ctx)
}

func main() {
	fmt.Println("%s repository plugin loaded")
}
`, toPascal(v.PluginName), v.PluginID, v.PluginName, v.Version, toPascal(v.PluginName), v.PluginID, toPascal(v.PluginName), toPascal(v.PluginName), v.PluginName)

	readme := fmt.Sprintf(`# %s (Repository Plugin)

%s
`, v.PluginName, v.Description)

	return map[string]string{
		"plugin.json": string(manifestJSON),
		"main.go":     mainGo,
		"README.md":   readme,
	}
}

func renderIntelligence(v TemplateVariables) map[string]string {
	manifest := map[string]any{
		"schema_version": "1.0.0",
		"id":             v.PluginID,
		"name":           v.PluginName,
		"version":        v.Version,
		"publisher":      v.Publisher,
		"description":    v.Description,
		"entrypoint":     "main.go",
		"capabilities": []map[string]string{
			{"name": "symbols.api", "version": "1.0.0"},
			{"name": "search.api", "version": "1.0.0"},
			{"name": "graph.api", "version": "1.0.0"},
			{"name": "analysis.api", "version": "1.0.0"},
		},
	}
	manifestJSON, _ := json.MarshalIndent(manifest, "", "  ")

	mainGo := fmt.Sprintf(`package main

import (
	"context"
	"fmt"

	"github.com/unhield/limoxel/plugin"
	"github.com/unhield/limoxel/plugin/api"
)

type %sIntelPlugin struct {
	*plugin.BasePlugin
}

func NewPlugin() plugin.Plugin {
	manifest := plugin.Manifest{
		ID:      "%s",
		Name:    "%s",
		Version: "%s",
	}
	return &%sIntelPlugin{
		BasePlugin: plugin.NewBasePlugin("%s", manifest),
	}
}

func (p *%sIntelPlugin) InspectIntelligence(ctx context.Context) error {
	pCtx := p.GetContext()

	// 1. Search Symbols
	res, err := pCtx.Search().SearchSymbols(ctx, "Plugin", api.PaginationOptions{Limit: 5})
	if err == nil && res != nil {
		pCtx.Logger().Info("symbols found", "count", res.TotalMatches)
	}

	// 2. Query Knowledge Graph
	nodes, edges, err := pCtx.Graph().GraphInfo(ctx)
	if err == nil {
		pCtx.Logger().Info("knowledge graph overview", "nodes", nodes, "edges", edges)
	}

	// 3. Health Analysis
	health, err := pCtx.Analysis().RepositoryHealth(ctx)
	if err == nil && health != nil {
		pCtx.Logger().Info("repository health", "score", health.OverallScore, "grade", health.Grade)
	}

	return nil
}

func main() {
	fmt.Println("%s intelligence plugin loaded")
}
`, toPascal(v.PluginName), v.PluginID, v.PluginName, v.Version, toPascal(v.PluginName), v.PluginID, toPascal(v.PluginName), v.PluginName)

	readme := fmt.Sprintf(`# %s (Intelligence Plugin)

%s
`, v.PluginName, v.Description)

	return map[string]string{
		"plugin.json": string(manifestJSON),
		"main.go":     mainGo,
		"README.md":   readme,
	}
}

func renderEnterprise(v TemplateVariables) map[string]string {
	manifest := map[string]any{
		"schema_version": "1.0.0",
		"id":             v.PluginID,
		"name":           v.PluginName,
		"version":        v.Version,
		"publisher":      v.Publisher,
		"description":    v.Description,
		"entrypoint":     "main.go",
		"capabilities": []map[string]string{
			{"name": "logging.structured", "version": "1.0.0"},
			{"name": "configuration.audit", "version": "1.0.0"},
		},
	}
	manifestJSON, _ := json.MarshalIndent(manifest, "", "  ")

	mainGo := fmt.Sprintf(`package main

import (
	"context"
	"fmt"

	"github.com/unhield/limoxel/plugin"
)

// Developer project scaffold demonstrating structured config, structured logging, and health checks.
type %sEnterpriseScaffold struct {
	*plugin.BasePlugin
	teamName string
}

func NewPlugin() plugin.Plugin {
	manifest := plugin.Manifest{
		ID:      "%s",
		Name:    "%s",
		Version: "%s",
	}
	return &%sEnterpriseScaffold{
		BasePlugin: plugin.NewBasePlugin("%s", manifest),
	}
}

func (p *%sEnterpriseScaffold) Init(ctx context.Context, host plugin.Host) error {
	if err := p.BasePlugin.Init(ctx, host); err != nil {
		return err
	}
	cfg := p.GetContext().Config()
	if val, ok := cfg["team_name"]; ok {
		p.teamName = val
	} else {
		p.teamName = "PlatformEngineering"
	}
	p.GetContext().Logger().Info("initialized enterprise scaffold", "team", p.teamName)
	return nil
}

func (p *%sEnterpriseScaffold) HealthCheck() map[string]string {
	return map[string]string{
		"status":  "healthy",
		"team":    p.teamName,
		"version": "%s",
	}
}

func main() {
	fmt.Println("%s enterprise scaffold loaded")
}
`, toPascal(v.PluginName), v.PluginID, v.PluginName, v.Version, toPascal(v.PluginName), v.PluginID, toPascal(v.PluginName), toPascal(v.PluginName), v.Version, v.PluginName)

	readme := fmt.Sprintf(`# %s (Enterprise Developer Scaffold)

%s

Note: This is a developer project scaffold demonstrating configuration parsing, logging, and operational health checks.
`, v.PluginName, v.Description)

	return map[string]string{
		"plugin.json": string(manifestJSON),
		"main.go":     mainGo,
		"README.md":   readme,
	}
}

func toPascal(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == ' ' || r == '-' || r == '_' || r == '.'
	})
	for i := range parts {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	res := strings.Join(parts, "")
	if res == "" {
		return "Plugin"
	}
	return res
}
