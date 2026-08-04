package context_test

import (
	"errors"
	"testing"

	platCtx "github.com/unhield/limoxel/internal/platform/context"
)

func TestKeyFormattingAndValidation(t *testing.T) {
	k1 := platCtx.NewKey("port")
	if k1.Name() != "port" || k1.Category() != "" || k1.String() != "port" || k1.IsEmpty() {
		t.Errorf("unexpected key k1: %v", k1)
	}

	k2 := platCtx.NewCategorizedKey("app", "host")
	if k2.Name() != "host" || k2.Category() != "app" || k2.String() != "app.host" || k2.IsEmpty() {
		t.Errorf("unexpected key k2: %v", k2)
	}

	emptyKey := platCtx.NewKey("")
	if err := platCtx.ValidateKey(emptyKey); !errors.Is(err, platCtx.ErrKeyEmpty) {
		t.Errorf("got %v, want ErrKeyEmpty", err)
	}
}

func TestValueConversions(t *testing.T) {
	t.Run("String conversion", func(t *testing.T) {
		v := platCtx.NewValue("val")
		s, err := v.String()
		if err != nil || s != "val" {
			t.Errorf("got %q, %v", s, err)
		}
	})

	t.Run("Int conversion", func(t *testing.T) {
		v := platCtx.NewValue("42")
		i, err := v.Int()
		if err != nil || i != 42 {
			t.Errorf("got %d, %v", i, err)
		}
	})

	t.Run("Bool conversion", func(t *testing.T) {
		v := platCtx.NewValue(true)
		b, err := v.Bool()
		if err != nil || !b {
			t.Errorf("got %v, %v", b, err)
		}
	})

	t.Run("Type mismatch error", func(t *testing.T) {
		v := platCtx.NewValue([]int{1, 2})
		_, err := v.Int()
		if !errors.Is(err, platCtx.ErrTypeMismatch) {
			t.Errorf("got %v, want ErrTypeMismatch", err)
		}
	})
}

func TestPlatformContextChainAndImmutability(t *testing.T) {
	keyApp := platCtx.NewCategorizedKey("system", "app")
	keyPort := platCtx.NewCategorizedKey("system", "port")
	keyEnv := platCtx.NewCategorizedKey("env", "mode")

	parent := platCtx.New().With(keyApp, "limoxel").With(keyPort, 8080)
	child := platCtx.NewWithParent(parent).With(keyEnv, "production")

	// Child inherits parent keys
	appStr, err := child.GetString(keyApp)
	if err != nil || appStr != "limoxel" {
		t.Errorf("child GetString(keyApp) got %q, %v", appStr, err)
	}

	portInt, err := child.GetInt(keyPort)
	if err != nil || portInt != 8080 {
		t.Errorf("child GetInt(keyPort) got %d, %v", portInt, err)
	}

	envStr, err := child.GetString(keyEnv)
	if err != nil || envStr != "production" {
		t.Errorf("child GetString(keyEnv) got %q, %v", envStr, err)
	}

	// Keys() should return sorted unique keys across parent and child
	keys := child.Keys()
	if len(keys) != 3 {
		t.Errorf("got %d keys, want 3", len(keys))
	}

	// Overriding key in child doesn't mutate parent
	childOverride := child.With(keyApp, "limoxel-override")
	parentApp, _ := parent.GetString(keyApp)
	childApp, _ := childOverride.GetString(keyApp)

	if parentApp != "limoxel" || childApp != "limoxel-override" {
		t.Errorf("immutability check failed: parentApp=%q, childApp=%q", parentApp, childApp)
	}
}

func TestPlatformContextMergeAndClone(t *testing.T) {
	k1 := platCtx.NewKey("k1")
	k2 := platCtx.NewKey("k2")

	ctx1 := platCtx.New().With(k1, "v1")
	ctx2 := platCtx.New().With(k2, "v2")

	merged := ctx1.Merge(ctx2)
	if !merged.Has(k1) || !merged.Has(k2) {
		t.Error("merged context missing keys")
	}

	cloned := ctx1.Clone()
	if !cloned.Has(k1) {
		t.Error("cloned context missing k1")
	}
}

func TestNilPlatformContextSafety(t *testing.T) {
	var c *platCtx.PlatformContext
	k := platCtx.NewKey("k")

	if c.Parent() != nil {
		t.Error("expected nil parent")
	}
	if _, exists := c.Get(k); exists {
		t.Error("expected false for nil Get")
	}
	if _, err := c.GetString(k); !errors.Is(err, platCtx.ErrKeyNotFound) {
		t.Errorf("got %v, want ErrKeyNotFound", err)
	}
	if c.Has(k) {
		t.Error("expected false for nil Has")
	}
	if len(c.Keys()) != 0 {
		t.Error("expected empty keys slice")
	}
}
