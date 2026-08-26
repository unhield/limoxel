package config_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/cli/config"
)

// A. Defaults
func TestConfigurationDefaults(t *testing.T) {
	mgr, err := config.NewManager(func(o *config.ManagerOptions) {
		o.DisableFiles = true
		o.DisableEnv = true
	})
	if err != nil {
		t.Fatalf("failed to initialize manager: %v", err)
	}

	eff := mgr.Effective()
	if eff == nil {
		t.Fatal("expected non-nil effective configuration")
	}

	if val := eff.GetString("general.version", ""); val != "1.0.0" {
		t.Errorf("expected general.version=1.0.0, got %q", val)
	}
	if val := eff.GetString("output.format", ""); val != "text" {
		t.Errorf("expected output.format=text, got %q", val)
	}
	if val := eff.GetInt("performance.workers", 0); val != 4 {
		t.Errorf("expected performance.workers=4, got %d", val)
	}
	if val := eff.GetBool("output.color", false); !val {
		t.Errorf("expected output.color=true, got %v", val)
	}
}

// B. Files (YAML, JSON, TOML)
func TestConfigurationFiles(t *testing.T) {
	tempDir := t.TempDir()

	// 1. JSON config
	jsonPath := filepath.Join(tempDir, "config.json")
	jsonContent := `{
		"version": "1.0.0",
		"output": {
			"format": "json",
			"theme": "light"
		},
		"performance": {
			"workers": 8
		}
	}`
	if err := os.WriteFile(jsonPath, []byte(jsonContent), 0644); err != nil {
		t.Fatal(err)
	}

	mgrJSON, err := config.NewManager(func(o *config.ManagerOptions) {
		o.ConfigFile = jsonPath
		o.DisableEnv = true
	})
	if err != nil {
		t.Fatalf("failed to load JSON config: %v", err)
	}
	if val := mgrJSON.Effective().GetString("output.format", ""); val != "json" {
		t.Errorf("expected output.format=json from file, got %q", val)
	}
	if val := mgrJSON.Effective().GetInt("performance.workers", 0); val != 8 {
		t.Errorf("expected performance.workers=8 from file, got %d", val)
	}

	// 2. YAML config
	yamlPath := filepath.Join(tempDir, "config.yaml")
	yamlContent := `
output:
  format: yaml
  theme: plain
performance:
  workers: 16
`
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	mgrYAML, err := config.NewManager(func(o *config.ManagerOptions) {
		o.ConfigFile = yamlPath
		o.DisableEnv = true
	})
	if err != nil {
		t.Fatalf("failed to load YAML config: %v", err)
	}
	if val := mgrYAML.Effective().GetString("output.format", ""); val != "yaml" {
		t.Errorf("expected output.format=yaml from file, got %q", val)
	}
	if val := mgrYAML.Effective().GetInt("performance.workers", 0); val != 16 {
		t.Errorf("expected performance.workers=16 from file, got %d", val)
	}

	// 3. TOML config
	tomlPath := filepath.Join(tempDir, "config.toml")
	tomlContent := `
[output]
format = "toml"
theme = "dark"

[performance]
workers = 12
`
	if err := os.WriteFile(tomlPath, []byte(tomlContent), 0644); err != nil {
		t.Fatal(err)
	}

	mgrTOML, err := config.NewManager(func(o *config.ManagerOptions) {
		o.ConfigFile = tomlPath
		o.DisableEnv = true
	})
	if err != nil {
		t.Fatalf("failed to load TOML config: %v", err)
	}
	if val := mgrTOML.Effective().GetString("output.format", ""); val != "toml" {
		t.Errorf("expected output.format=toml from file, got %q", val)
	}
	if val := mgrTOML.Effective().GetInt("performance.workers", 0); val != 12 {
		t.Errorf("expected performance.workers=12 from file, got %d", val)
	}
}

// C. Profiles & Inheritance
func TestConfigurationProfiles(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	content := `{
		"output": {
			"format": "text"
		},
		"profiles": {
			"ci": {
				"name": "ci",
				"values": {
					"output.format": "json",
					"output.color": false,
					"analysis.strict_mode": true
				}
			},
			"ci-verbose": {
				"name": "ci-verbose",
				"inherits": "ci",
				"values": {
					"logging.level": "debug"
				}
			}
		}
	}`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Load CI profile
	mgrCI, err := config.NewManager(func(o *config.ManagerOptions) {
		o.ConfigFile = configPath
		o.ActiveProfile = "ci"
		o.DisableEnv = true
	})
	if err != nil {
		t.Fatalf("failed to load CI profile: %v", err)
	}

	effCI := mgrCI.Effective()
	if val := effCI.GetString("output.format", ""); val != "json" {
		t.Errorf("expected output.format=json in CI profile, got %q", val)
	}
	if val := effCI.GetBool("output.color", true); val {
		t.Errorf("expected output.color=false in CI profile, got %v", val)
	}
	if val := effCI.GetBool("analysis.strict_mode", false); !val {
		t.Errorf("expected analysis.strict_mode=true in CI profile, got %v", val)
	}

	// Load inheriting ci-verbose profile
	mgrVerbose, err := config.NewManager(func(o *config.ManagerOptions) {
		o.ConfigFile = configPath
		o.ActiveProfile = "ci-verbose"
		o.DisableEnv = true
	})
	if err != nil {
		t.Fatalf("failed to load ci-verbose profile: %v", err)
	}

	effVerbose := mgrVerbose.Effective()
	if val := effVerbose.GetString("output.format", ""); val != "json" {
		t.Errorf("expected inherited output.format=json, got %q", val)
	}
	if val := effVerbose.GetString("logging.level", ""); val != "debug" {
		t.Errorf("expected logging.level=debug, got %q", val)
	}
}

// D. Environment Variables & Precedence
func TestConfigurationEnvironment(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	content := `{
		"output": {
			"format": "text"
		}
	}`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	os.Setenv("LIMOXEL_OUTPUT_FORMAT", "markdown")
	os.Setenv("LIMOXEL_PERFORMANCE_WORKERS", "10")
	defer os.Unsetenv("LIMOXEL_OUTPUT_FORMAT")
	defer os.Unsetenv("LIMOXEL_PERFORMANCE_WORKERS")

	mgr, err := config.NewManager(func(o *config.ManagerOptions) {
		o.ConfigFile = configPath
	})
	if err != nil {
		t.Fatalf("failed to initialize manager: %v", err)
	}

	eff := mgr.Effective()
	// Environment (40) overrides File (20)
	if val := eff.GetString("output.format", ""); val != "markdown" {
		t.Errorf("expected env override output.format=markdown, got %q", val)
	}
	if val := eff.GetInt("performance.workers", 0); val != 10 {
		t.Errorf("expected env override performance.workers=10, got %d", val)
	}
}

// E. Runtime Overrides (Highest Precedence)
func TestConfigurationRuntimeOverrides(t *testing.T) {
	os.Setenv("LIMOXEL_OUTPUT_FORMAT", "markdown")
	defer os.Unsetenv("LIMOXEL_OUTPUT_FORMAT")

	mgr, err := config.NewManager(func(o *config.ManagerOptions) {
		o.DisableFiles = true
		o.RuntimeOverrides = map[string]any{
			"output.format": "svg",
		}
	})
	if err != nil {
		t.Fatalf("failed to initialize manager: %v", err)
	}

	// Runtime (50) overrides Environment (40)
	if val := mgr.Effective().GetString("output.format", ""); val != "svg" {
		t.Errorf("expected runtime override output.format=svg, got %q", val)
	}
}

// F. Deterministic Precedence Hierarchy
func TestPrecedenceHierarchy(t *testing.T) {
	// Precedence order: Defaults (10) < Files (20) < Profiles (30) < Env (40) < Runtime (50)
	if config.PrecedenceDefault >= config.PrecedenceFile {
		t.Error("PrecedenceDefault must be < PrecedenceFile")
	}
	if config.PrecedenceFile >= config.PrecedenceProfile {
		t.Error("PrecedenceFile must be < PrecedenceProfile")
	}
	if config.PrecedenceProfile >= config.PrecedenceEnv {
		t.Error("PrecedenceProfile must be < PrecedenceEnv")
	}
	if config.PrecedenceEnv >= config.PrecedenceRuntime {
		t.Error("PrecedenceEnv must be < PrecedenceRuntime")
	}
}

// G. Validation & Error Handling
func TestConfigurationValidation(t *testing.T) {
	// Invalid enum value
	_, err := config.NewManager(func(o *config.ManagerOptions) {
		o.DisableFiles = true
		o.DisableEnv = true
		o.RuntimeOverrides = map[string]any{
			"output.format": "invalid_format_type",
		}
	})
	if err == nil {
		t.Fatal("expected validation error for invalid output.format, got nil")
	}
	if !strings.Contains(err.Error(), "INVALID_VALUE") {
		t.Errorf("expected INVALID_VALUE error code, got %v", err)
	}

	// Invalid integer range
	_, errRange := config.NewManager(func(o *config.ManagerOptions) {
		o.DisableFiles = true
		o.DisableEnv = true
		o.RuntimeOverrides = map[string]any{
			"performance.workers": 9999, // Max allowed is 64
		}
	})
	if errRange == nil {
		t.Fatal("expected validation error for out-of-bounds performance.workers, got nil")
	}
}

// H. Secret Redaction & Introspection
func TestSecretRedaction(t *testing.T) {
	mgr, err := config.NewManager(func(o *config.ManagerOptions) {
		o.DisableFiles = true
		o.DisableEnv = true
		o.RuntimeOverrides = map[string]any{
			"custom.api_key":     "super_secret_token_12345",
			"custom.auth_secret": "top_secret_credential",
		}
	})
	if err != nil {
		t.Fatalf("failed to initialize manager: %v", err)
	}

	entries := mgr.Inspect(true)
	foundSecret := false
	for _, e := range entries {
		if config.IsSecretKey(e.Key) {
			foundSecret = true
			if e.Value != config.MaskedValueConstant {
				t.Errorf("key %q was not redacted! Value: %v", e.Key, e.Value)
			}
		}
	}
	if !foundSecret {
		t.Error("expected to find secret entries in inspection")
	}
}

// I. Safe Persistence & Set/Unset
func TestConfigurationPersistence(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ".limoxel.yaml")

	mgr, err := config.NewManager(func(o *config.ManagerOptions) {
		o.ConfigFile = configPath
		o.WorkspaceDir = tempDir
		o.DisableEnv = true
	})
	if err != nil {
		t.Fatalf("failed to initialize manager: %v", err)
	}

	// Set value
	if err := mgr.Set("output.format", "html", configPath); err != nil {
		t.Fatalf("failed to set output.format: %v", err)
	}

	if val := mgr.Effective().GetString("output.format", ""); val != "html" {
		t.Errorf("expected effective output.format=html after Set, got %q", val)
	}

	// Unset value
	if err := mgr.Unset("output.format", configPath); err != nil {
		t.Fatalf("failed to unset output.format: %v", err)
	}

	// After unset, falls back to default
	if val := mgr.Effective().GetString("output.format", ""); val != "text" {
		t.Errorf("expected fallback output.format=text after Unset, got %q", val)
	}
}

// J. Deterministic Execution
func TestConfigurationDeterminism(t *testing.T) {
	opts := func(o *config.ManagerOptions) {
		o.DisableFiles = true
		o.DisableEnv = true
		o.RuntimeOverrides = map[string]any{
			"output.theme":  "light",
			"logging.level": "warn",
		}
	}

	mgr1, _ := config.NewManager(opts)
	mgr2, _ := config.NewManager(opts)

	entries1 := mgr1.Inspect(true)
	entries2 := mgr2.Inspect(true)

	if len(entries1) != len(entries2) {
		t.Fatalf("mismatched entry lengths: %d vs %d", len(entries1), len(entries2))
	}

	for i := range entries1 {
		if entries1[i].Key != entries2[i].Key || fmt.Sprint(entries1[i].Value) != fmt.Sprint(entries2[i].Value) {
			t.Errorf("determinism mismatch at index %d: %+v vs %+v", i, entries1[i], entries2[i])
		}
	}
}

// K. Typed Accessors
func TestTypedAccessors(t *testing.T) {
	mgr, _ := config.NewManager(func(o *config.ManagerOptions) {
		o.DisableFiles = true
		o.DisableEnv = true
		o.RuntimeOverrides = map[string]any{
			"test.string":   "hello",
			"test.bool":     true,
			"test.int":      42,
			"test.float":    3.1415,
			"test.duration": "500ms",
			"test.slice":    []string{"a", "b", "c"},
		}
	})

	eff := mgr.Effective()
	if v := eff.GetString("test.string", ""); v != "hello" {
		t.Errorf("GetString failed: %q", v)
	}
	if v := eff.GetBool("test.bool", false); !v {
		t.Errorf("GetBool failed: %v", v)
	}
	if v := eff.GetInt("test.int", 0); v != 42 {
		t.Errorf("GetInt failed: %d", v)
	}
	if v := eff.GetFloat64("test.float", 0.0); v < 3.14 {
		t.Errorf("GetFloat64 failed: %f", v)
	}
	if v := eff.GetDuration("test.duration", 0); v != 500*time.Millisecond {
		t.Errorf("GetDuration failed: %v", v)
	}
	if v := eff.GetStringSlice("test.slice", nil); len(v) != 3 || v[0] != "a" {
		t.Errorf("GetStringSlice failed: %v", v)
	}
}
