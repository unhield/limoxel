package model_test

import (
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
)

const sampleValidManifest = `{
  "schema_version": "1.0.0",
  "id": "org.limoxel.sample",
  "name": "Sample Plugin",
  "version": "1.2.3",
  "publisher": "Limoxel Core",
  "description": "A sample plugin demonstrating manifest parsing",
  "entrypoint": "main.go",
  "capabilities": [
    {
      "name": "linter.custom",
      "version": "1.0.0",
      "description": "Custom code linter"
    }
  ],
  "required_capabilities": [
    {
      "name": "repository.files",
      "min_version": "1.0.0",
      "optional": false
    }
  ],
  "dependencies": [
    {
      "id": "org.limoxel.common",
      "constraint": ">= 1.0.0, < 2.0.0",
      "optional": true
    }
  ],
  "compatibility": {
    "min_host_version": "1.0.0",
    "supported_platforms": ["windows", "linux", "darwin"]
  },
  "metadata": {
    "license": "MIT",
    "homepage": "https://example.com/sample",
    "tags": ["linter", "sample"],
    "custom_attributes": {
      "tier": "enterprise"
    }
  }
}`

func TestParseManifestJSON_Valid(t *testing.T) {
	m, err := model.ParseManifestJSON([]byte(sampleValidManifest))
	if err != nil {
		t.Fatalf("ParseManifestJSON failed unexpectedly: %v", err)
	}

	if m.SchemaVersion != "1.0.0" {
		t.Errorf("got schema_version %q, want 1.0.0", m.SchemaVersion)
	}
	if string(m.ID) != "org.limoxel.sample" {
		t.Errorf("got id %q, want org.limoxel.sample", m.ID)
	}
	if m.Version.String() != "1.2.3" {
		t.Errorf("got version %q, want 1.2.3", m.Version)
	}
	if len(m.Capabilities) != 1 || m.Capabilities[0].Name != "linter.custom" {
		t.Errorf("unexpected capabilities: %+v", m.Capabilities)
	}
	if len(m.RequiredCapabilities) != 1 || m.RequiredCapabilities[0].Name != "repository.files" {
		t.Errorf("unexpected required capabilities: %+v", m.RequiredCapabilities)
	}
	if len(m.Dependencies) != 1 || string(m.Dependencies[0].ID) != "org.limoxel.common" {
		t.Errorf("unexpected dependencies: %+v", m.Dependencies)
	}
	if m.Metadata.License() != "MIT" {
		t.Errorf("got license %q, want MIT", m.Metadata.License())
	}
	if m.Metadata.CustomAttributes()["tier"] != "enterprise" {
		t.Errorf("unexpected custom attribute: %+v", m.Metadata.CustomAttributes())
	}

	// Test ToPublic
	pub := m.ToPublic()
	if pub.ID != "org.limoxel.sample" || pub.Version != "1.2.3" {
		t.Errorf("unexpected public manifest: %+v", pub)
	}
}

func TestParseManifestJSON_Invalid(t *testing.T) {
	invalidCases := []struct {
		name string
		json string
	}{
		{"empty", ""},
		{"malformed json", "{not-valid-json}"},
		{"unsupported schema", `{"schema_version":"2.0.0","id":"a.b","name":"A","version":"1.0.0"}`},
		{"missing id", `{"schema_version":"1.0.0","name":"A","version":"1.0.0"}`},
		{"invalid id syntax", `{"schema_version":"1.0.0","id":"Invalid ID","name":"A","version":"1.0.0"}`},
		{"missing name", `{"schema_version":"1.0.0","id":"a.b","version":"1.0.0"}`},
		{"invalid semver", `{"schema_version":"1.0.0","id":"a.b","name":"A","version":"invalid-ver"}`},
		{"self dependency", `{"schema_version":"1.0.0","id":"a.b","name":"A","version":"1.0.0","dependencies":[{"id":"a.b"}]}`},
		{"invalid capability semver", `{"schema_version":"1.0.0","id":"a.b","name":"A","version":"1.0.0","capabilities":[{"name":"c","version":"bad"}]}`},
	}

	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := model.ParseManifestJSON([]byte(tc.json))
			if err == nil {
				t.Errorf("expected error for invalid case %s, got nil", tc.name)
			}
		})
	}
}
