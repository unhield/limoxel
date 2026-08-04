package configuration_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/unhield/limoxel/internal/platform/configuration"
)

func TestValueTypeConversions(t *testing.T) {
	t.Run("String conversion", func(t *testing.T) {
		val := configuration.NewValue("hello")
		str, err := val.String()
		if err != nil || str != "hello" {
			t.Errorf("got %q, %v; want hello, nil", str, err)
		}
	})

	t.Run("Int conversions", func(t *testing.T) {
		val := configuration.NewValue(42)
		i, err := val.Int()
		if err != nil || i != 42 {
			t.Errorf("got %d, %v; want 42, nil", i, err)
		}
		i64, err := val.Int64()
		if err != nil || i64 != 42 {
			t.Errorf("got %d, %v; want 42, nil", i64, err)
		}

		strVal := configuration.NewValue("100")
		parsedInt, err := strVal.Int()
		if err != nil || parsedInt != 100 {
			t.Errorf("got %d, %v; want 100, nil", parsedInt, err)
		}
	})

	t.Run("Float64 conversion", func(t *testing.T) {
		val := configuration.NewValue(3.14)
		f, err := val.Float64()
		if err != nil || f != 3.14 {
			t.Errorf("got %f, %v; want 3.14, nil", f, err)
		}
	})

	t.Run("Bool conversion", func(t *testing.T) {
		val := configuration.NewValue("true")
		b, err := val.Bool()
		if err != nil || !b {
			t.Errorf("got %v, %v; want true, nil", b, err)
		}
	})

	t.Run("Duration conversion", func(t *testing.T) {
		val := configuration.NewValue("5s")
		d, err := val.Duration()
		if err != nil || d != 5*time.Second {
			t.Errorf("got %v, %v; want 5s, nil", d, err)
		}
	})

	t.Run("StringSlice conversion", func(t *testing.T) {
		slice := []string{"a", "b", "c"}
		val := configuration.NewValue(slice)
		res, err := val.StringSlice()
		if err != nil || !reflect.DeepEqual(res, slice) {
			t.Errorf("got %v, %v; want %v, nil", res, err, slice)
		}
	})

	t.Run("Type mismatch error", func(t *testing.T) {
		val := configuration.NewValue("not_an_int")
		_, err := val.Int()
		if !errors.Is(err, configuration.ErrTypeMismatch) {
			t.Errorf("got error %v, want ErrTypeMismatch", err)
		}
	})
}

func TestConfigurationAccessorsAndKeys(t *testing.T) {
	configMap := map[string]configuration.Value{
		"app.name":    configuration.NewValue("limoxel"),
		"app.port":    configuration.NewValue(8080),
		"app.debug":   configuration.NewValue(true),
		"app.timeout": configuration.NewValue("10s"),
		"app.tags":    configuration.NewValue([]string{"core", "cli"}),
	}

	cfg, err := configuration.NewFromMap(configMap)
	if err != nil {
		t.Fatalf("failed to create configuration: %v", err)
	}

	if !cfg.Has("app.name") {
		t.Error("expected cfg.Has('app.name') to be true")
	}

	name, err := cfg.GetString("app.name")
	if err != nil || name != "limoxel" {
		t.Errorf("GetString got %q, %v", name, err)
	}

	port, err := cfg.GetInt("app.port")
	if err != nil || port != 8080 {
		t.Errorf("GetInt got %d, %v", port, err)
	}

	port64, err := cfg.GetInt64("app.port")
	if err != nil || port64 != 8080 {
		t.Errorf("GetInt64 got %d, %v", port64, err)
	}

	debug, err := cfg.GetBool("app.debug")
	if err != nil || !debug {
		t.Errorf("GetBool got %v, %v", debug, err)
	}

	timeout, err := cfg.GetDuration("app.timeout")
	if err != nil || timeout != 10*time.Second {
		t.Errorf("GetDuration got %v, %v", timeout, err)
	}

	tags, err := cfg.GetStringSlice("app.tags")
	if err != nil || len(tags) != 2 {
		t.Errorf("GetStringSlice got %v, %v", tags, err)
	}

	keys := cfg.Keys()
	if len(keys) != 5 {
		t.Errorf("expected 5 keys, got %d", len(keys))
	}
	if keys[0] != "app.debug" {
		t.Errorf("expected sorted keys starting with 'app.debug', got %q", keys[0])
	}
}

func TestConfigurationKeyNotFound(t *testing.T) {
	cfg := configuration.New()
	_, err := cfg.GetString("missing")
	if !errors.Is(err, configuration.ErrKeyNotFound) {
		t.Errorf("got %v, want ErrKeyNotFound", err)
	}
}

func TestConfigurationMergeAndValidate(t *testing.T) {
	cfg1, _ := configuration.NewFromMap(map[string]configuration.Value{
		"k1": configuration.NewValue("v1"),
		"k2": configuration.NewValue("v2"),
	})
	cfg2, _ := configuration.NewFromMap(map[string]configuration.Value{
		"k2": configuration.NewValue("v2_override"),
		"k3": configuration.NewValue("v3"),
	})

	merged, err := cfg1.Merge(cfg2)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	v2, _ := merged.GetString("k2")
	if v2 != "v2_override" {
		t.Errorf("got %q, want v2_override", v2)
	}
	if len(merged.Keys()) != 3 {
		t.Errorf("got %d keys, want 3", len(merged.Keys()))
	}

	// Validate rule test
	err = merged.Validate(func(c *configuration.Configuration) error {
		if !c.Has("k1") {
			return errors.New("k1 required")
		}
		return nil
	})
	if err != nil {
		t.Errorf("Validate rule failed: %v", err)
	}

	err = merged.Validate(func(c *configuration.Configuration) error {
		return errors.New("failing rule")
	})
	if !errors.Is(err, configuration.ErrValidationFailed) {
		t.Errorf("got %v, want ErrValidationFailed", err)
	}
}

func TestBuilderAndMemoryProvider(t *testing.T) {
	memProvider := configuration.NewMemoryProvider("test-memory", map[string]configuration.Value{
		"prov.key": configuration.NewValue("prov_val"),
	})
	if memProvider.Name() != "test-memory" {
		t.Errorf("got name %q, want test-memory", memProvider.Name())
	}

	ctx := context.Background()
	builder := configuration.NewBuilder().
		WithDefault("def.key", "def_val").
		WithProvider(memProvider).
		WithOverride("override.key", "override_val")

	cfg, err := builder.Build(ctx)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	defVal, _ := cfg.GetString("def.key")
	provVal, _ := cfg.GetString("prov.key")
	overVal, _ := cfg.GetString("override.key")

	if defVal != "def_val" || provVal != "prov_val" || overVal != "override_val" {
		t.Errorf("unexpected configuration values: def=%q, prov=%q, over=%q", defVal, provVal, overVal)
	}
}

func TestNilConfigurationSafety(t *testing.T) {
	var cfg *configuration.Configuration
	if _, ok := cfg.Get("k"); ok {
		t.Error("expected false for nil Get")
	}
	if len(cfg.Keys()) != 0 {
		t.Error("expected empty slice for nil Keys")
	}
	if err := cfg.Validate(); !errors.Is(err, configuration.ErrConfigurationNil) {
		t.Errorf("got %v, want ErrConfigurationNil", err)
	}

	var b *configuration.Builder
	if _, err := b.Build(context.Background()); !errors.Is(err, configuration.ErrBuilderNil) {
		t.Errorf("got %v, want ErrBuilderNil", err)
	}
}
