package bootstrap_test

import (
	"context"
	"errors"
	"testing"

	"github.com/unhield/limoxel/internal/platform/bootstrap"
	"github.com/unhield/limoxel/internal/platform/runtime"
)

func TestBootstrapperSuccess(t *testing.T) {
	ctx := context.Background()

	prereqRan := false
	opt := bootstrap.WithPrerequisiteValidator(func() error {
		prereqRan = true
		return nil
	})

	b := bootstrap.NewBootstrapper(opt, nil)
	rt, err := b.Bootstrap(ctx)
	if err != nil {
		t.Fatalf("Bootstrap failed: %v", err)
	}
	if !prereqRan {
		t.Error("prerequisite validator did not run")
	}
	if rt == nil || !rt.IsRunning() {
		t.Error("expected running runtime")
	}
	if rt.State() != runtime.StateRunning {
		t.Errorf("got state %v, want StateRunning", rt.State())
	}
}

func TestBootstrapperPrerequisiteFailure(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("missing environment variable")

	b := bootstrap.NewBootstrapper(bootstrap.WithPrerequisiteValidator(func() error {
		return expectedErr
	}))

	rt, err := b.Bootstrap(ctx)
	if err == nil {
		t.Fatal("expected bootstrap error, got nil")
	}
	if !errors.Is(err, bootstrap.ErrPrerequisitesFailed) {
		t.Errorf("got error %v, want ErrPrerequisitesFailed", err)
	}
	if rt != nil {
		t.Error("expected nil runtime on failure")
	}
}

func TestBootstrapperContextErrors(t *testing.T) {
	b := bootstrap.NewBootstrapper()

	t.Run("nil context", func(t *testing.T) {
		_, err := b.Bootstrap(nil)
		if !errors.Is(err, bootstrap.ErrNilContext) {
			t.Errorf("got error %v, want ErrNilContext", err)
		}
	})

	t.Run("canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := b.Bootstrap(ctx)
		if err == nil || !errors.Is(err, bootstrap.ErrNilContext) {
			t.Errorf("got error %v, want ErrNilContext", err)
		}
	})
}

func TestNilBootstrapper(t *testing.T) {
	var b *bootstrap.Bootstrapper
	_, err := b.Bootstrap(context.Background())
	if !errors.Is(err, bootstrap.ErrBootstrapperNil) {
		t.Errorf("got error %v, want ErrBootstrapperNil", err)
	}
}

func TestRunConvenience(t *testing.T) {
	ctx := context.Background()
	rt, err := bootstrap.Run(ctx)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if rt == nil || !rt.IsRunning() {
		t.Error("expected running runtime from Run()")
	}
}
