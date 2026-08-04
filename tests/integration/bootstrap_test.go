package integration_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/unhield/limoxel/internal/cli"
	"github.com/unhield/limoxel/internal/engine"
)

func TestBootstrapIntegration(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("full successful bootstrap integration flow", func(t *testing.T) {
		cfg, err := cli.NewConfig("limoxel-app", "1.0.0", tempDir)
		if err != nil {
			t.Fatalf("cli.NewConfig failed: %v", err)
		}

		if cfg.AppName() != "limoxel-app" || cfg.Version() != "1.0.0" || cfg.RootDir() != filepath.Clean(tempDir) {
			t.Errorf("unexpected config: appName=%q, ver=%q, rootDir=%q", cfg.AppName(), cfg.Version(), cfg.RootDir())
		}

		boot, err := cli.NewBootstrap(cfg)
		if err != nil {
			t.Fatalf("cli.NewBootstrap failed: %v", err)
		}

		if boot.Initialized() {
			t.Error("expected boot.Initialized() to be false before Initialize()")
		}
		if boot.Engine() != nil {
			t.Error("expected engine to be nil before Initialize()")
		}

		eng, err := boot.Initialize()
		if err != nil {
			t.Fatalf("boot.Initialize() failed: %v", err)
		}

		if !boot.Initialized() {
			t.Error("expected boot.Initialized() to be true after Initialize()")
		}

		if boot.Engine() != eng {
			t.Error("boot.Engine() mismatch with returned engine")
		}

		// Verify Engine wiring and state
		if eng.State() != engine.StateRunning {
			t.Errorf("got engine state %v, want StateRunning", eng.State())
		}

		if eng.Filesystem() == nil {
			t.Error("expected Filesystem service to be wired")
		}
		if eng.LanguageRegistry() == nil {
			t.Error("expected LanguageRegistry to be wired")
		}
		if eng.ParserRegistry() == nil {
			t.Error("expected ParserRegistry to be wired")
		}
		if eng.Workspace() == nil {
			t.Error("expected Workspace to be wired")
		}
		if eng.Repository() == nil {
			t.Error("expected Repository to be wired")
		}

		// Graceful engine shutdown flow
		if err := eng.Stop(); err != nil {
			t.Fatalf("eng.Stop() failed: %v", err)
		}
		if eng.State() != engine.StateStopped {
			t.Errorf("got engine state %v after Stop(), want StateStopped", eng.State())
		}

		if err := eng.Terminate(); err != nil {
			t.Fatalf("eng.Terminate() failed: %v", err)
		}
		if eng.State() != engine.StateTerminated {
			t.Errorf("got engine state %v after Terminate(), want StateTerminated", eng.State())
		}
	})

	t.Run("bootstrap double initialization error handling", func(t *testing.T) {
		cfg, _ := cli.NewConfig("limoxel-app", "1.0.0", tempDir)
		boot, _ := cli.NewBootstrap(cfg)

		_, err := boot.Initialize()
		if err != nil {
			t.Fatalf("first Initialize failed: %v", err)
		}

		_, err = boot.Initialize()
		if !errors.Is(err, cli.ErrAlreadyInitialized) {
			t.Errorf("got error %v, want ErrAlreadyInitialized", err)
		}
	})

	t.Run("bootstrap nil configuration error handling", func(t *testing.T) {
		_, err := cli.NewBootstrap(nil)
		if !errors.Is(err, cli.ErrNilConfig) {
			t.Errorf("got error %v, want ErrNilConfig", err)
		}
	})

	t.Run("bootstrap invalid workspace root directory handling", func(t *testing.T) {
		nonExistentRoot := filepath.Join(tempDir, "non_existent_dir")
		cfg, _ := cli.NewConfig("limoxel-app", "1.0.0", nonExistentRoot)
		boot, _ := cli.NewBootstrap(cfg)

		_, err := boot.Initialize()
		if err == nil || !errors.Is(err, cli.ErrBootstrapFailed) {
			t.Errorf("got error %v, want ErrBootstrapFailed for non-existent root", err)
		}
	})
}
