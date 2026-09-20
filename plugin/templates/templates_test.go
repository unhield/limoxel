package templates_test

import (
	"encoding/json"
	"testing"

	"github.com/unhield/limoxel/plugin"
	"github.com/unhield/limoxel/plugin/templates"
)

func TestListTemplates(t *testing.T) {
	all := templates.ListTemplates()
	if len(all) != 5 {
		t.Fatalf("expected 5 templates, got %d", len(all))
	}

	expectedTypes := map[templates.TemplateType]bool{
		templates.TemplateBasic:        false,
		templates.TemplateCLI:          false,
		templates.TemplateRepository:   false,
		templates.TemplateIntelligence: false,
		templates.TemplateEnterprise:   false,
	}

	for _, d := range all {
		expectedTypes[d.Type] = true
		if d.Name == "" || d.Description == "" || len(d.RequiredCapabilities) == 0 {
			t.Errorf("incomplete descriptor for %s: %+v", d.Type, d)
		}
	}

	for k, found := range expectedTypes {
		if !found {
			t.Errorf("missing template type in list: %s", k)
		}
	}
}

func TestRenderAllTemplates(t *testing.T) {
	vars := templates.TemplateVariables{
		PluginID:    "com.test.sample",
		PluginName:  "SamplePlugin",
		Version:     "1.2.3",
		Publisher:   "Test Author",
		Description: "A unit test plugin",
	}

	all := templates.ListTemplates()
	for _, d := range all {
		t.Run(string(d.Type), func(t *testing.T) {
			files, err := templates.RenderTemplate(d.Type, vars)
			if err != nil {
				t.Fatalf("render template %s failed: %v", d.Type, err)
			}

			// Must contain plugin.json, main.go, and README.md
			for _, required := range []string{"plugin.json", "main.go", "README.md"} {
				content, ok := files[required]
				if !ok || len(content) == 0 {
					t.Errorf("template %s missing non-empty file %s", d.Type, required)
				}
			}

			// Validate plugin.json schema
			var manifest plugin.Manifest
			if err := json.Unmarshal([]byte(files["plugin.json"]), &manifest); err != nil {
				t.Errorf("template %s has invalid plugin.json: %v", d.Type, err)
			}
			if manifest.ID != vars.PluginID {
				t.Errorf("template %s manifest ID mismatch: expected %s, got %s", d.Type, vars.PluginID, manifest.ID)
			}
			if len(manifest.Capabilities) == 0 {
				t.Errorf("template %s manifest has no capabilities declared", d.Type)
			}
		})
	}
}

func TestRenderUnknownTemplate(t *testing.T) {
	_, err := templates.RenderTemplate("nonexistent", templates.TemplateVariables{})
	if err == nil {
		t.Errorf("expected error for unknown template")
	}
}
