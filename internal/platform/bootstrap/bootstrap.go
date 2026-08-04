package bootstrap

import (
	"context"
	"fmt"

	"github.com/unhield/limoxel/internal/platform/runtime"
)

// Bootstrapper coordinates the construction, initialization, and control transfer of the Limoxel platform.
type Bootstrapper struct {
	validators []PrerequisiteValidator
}

// NewBootstrapper creates a new Bootstrapper instance configured with the given functional options.
func NewBootstrapper(opts ...Option) *Bootstrapper {
	b := &Bootstrapper{
		validators: make([]PrerequisiteValidator, 0),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(b)
		}
	}

	return b
}

// Bootstrap constructs, initializes, and starts the Runtime, returning a fully operational Runtime instance.
// If any stage fails, Bootstrap fails fast, performs cleanup, and returns a descriptive error.
func (b *Bootstrapper) Bootstrap(ctx context.Context) (*runtime.Runtime, error) {
	if b == nil {
		return nil, ErrBootstrapperNil
	}

	if ctx == nil {
		return nil, ErrNilContext
	}

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNilContext, err)
	}

	// Step 1: Prerequisite and environment validation
	if err := b.validatePrerequisites(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPrerequisitesFailed, err)
	}

	// Step 2: Runtime construction
	rt, err := runtime.New()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRuntimeCreationFailed, err)
	}

	// Step 3: Runtime pre-startup validation
	if err := rt.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRuntimeValidationFailed, err)
	}

	// Step 4: Runtime initialization phase
	if err := rt.Initialize(ctx); err != nil {
		_ = rt.Shutdown(ctx)
		return nil, fmt.Errorf("%w: %v", ErrRuntimeInitializationFailed, err)
	}

	// Step 5: Runtime preparation phase
	if err := rt.Prepare(ctx); err != nil {
		_ = rt.Shutdown(ctx)
		return nil, fmt.Errorf("%w: %v", ErrRuntimePreparationFailed, err)
	}

	// Step 6: Runtime startup phase
	if err := rt.Start(ctx); err != nil {
		_ = rt.Shutdown(ctx)
		return nil, fmt.Errorf("%w: %v", ErrRuntimeStartupFailed, err)
	}

	// Step 7: Post-startup Runtime verification
	if !rt.IsRunning() || rt.State() != runtime.StateRunning {
		_ = rt.Shutdown(ctx)
		return nil, fmt.Errorf("%w: runtime state is %s, expected %s", ErrRuntimeVerificationFailed, rt.State(), runtime.StateRunning)
	}

	// Step 8: Control transfer complete; return operational Runtime
	return rt, nil
}

// validatePrerequisites runs registered prerequisite validators prior to Runtime construction.
func (b *Bootstrapper) validatePrerequisites() error {
	for _, validator := range b.validators {
		if err := validator(); err != nil {
			return err
		}
	}
	return nil
}

// Run is a convenience function that initializes a default Bootstrapper and executes Bootstrap.
func Run(ctx context.Context, opts ...Option) (*runtime.Runtime, error) {
	bootstrapper := NewBootstrapper(opts...)
	return bootstrapper.Bootstrap(ctx)
}
