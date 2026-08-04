package cli_test

import (
	"bytes"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/unhield/limoxel/internal/cli"
	"github.com/unhield/limoxel/internal/engine"
)

func TestConfigConstructorAndGetters(t *testing.T) {
	t.Run("valid config creation", func(t *testing.T) {
		cfg, err := cli.NewConfig("my-app", "2.0.0", "/tmp/root")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.AppName() != "my-app" {
			t.Errorf("got AppName %q, want my-app", cfg.AppName())
		}
		if cfg.Version() != "2.0.0" {
			t.Errorf("got Version %q, want 2.0.0", cfg.Version())
		}
		if cfg.RootDir() != "/tmp/root" && cfg.RootDir() != "\\tmp\\root" {
			t.Errorf("got RootDir %q", cfg.RootDir())
		}
	})

	t.Run("default fallbacks", func(t *testing.T) {
		cfg, err := cli.NewConfig("  ", "  ", "  ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.AppName() != "limoxel" {
			t.Errorf("got AppName %q, want limoxel", cfg.AppName())
		}
		if cfg.Version() != "1.0.0" {
			t.Errorf("got Version %q, want 1.0.0", cfg.Version())
		}
		if cfg.RootDir() != "." {
			t.Errorf("got RootDir %q, want .", cfg.RootDir())
		}
	})

	t.Run("nil config getters", func(t *testing.T) {
		var cfg *cli.Config
		if cfg.AppName() != "" || cfg.Version() != "" || cfg.RootDir() != "" {
			t.Error("expected empty string getters on nil config")
		}
	})
}

func TestBootstrapInitialization(t *testing.T) {
	tempDir := t.TempDir()
	cfg, _ := cli.NewConfig("limoxel-cli", "1.0.0", tempDir)

	t.Run("nil config error", func(t *testing.T) {
		_, err := cli.NewBootstrap(nil)
		if !errors.Is(err, cli.ErrNilConfig) {
			t.Errorf("got %v, want ErrNilConfig", err)
		}
	})

	t.Run("successful initialization", func(t *testing.T) {
		boot, err := cli.NewBootstrap(cfg)
		if err != nil {
			t.Fatalf("NewBootstrap failed: %v", err)
		}
		if boot.Config() != cfg {
			t.Error("config mismatch")
		}
		if boot.Initialized() {
			t.Error("should not be initialized yet")
		}
		if boot.Engine() != nil {
			t.Error("engine should be nil before Initialize")
		}

		eng, err := boot.Initialize()
		if err != nil {
			t.Fatalf("Initialize failed: %v", err)
		}
		if !boot.Initialized() {
			t.Error("expected Initialized == true")
		}
		if boot.Engine() != eng {
			t.Error("engine mismatch after Initialize")
		}
		if eng.State() != engine.StateRunning {
			t.Errorf("initialized engine state got %v, want StateRunning", eng.State())
		}

		// Double initialization error
		_, err = boot.Initialize()
		if !errors.Is(err, cli.ErrAlreadyInitialized) {
			t.Errorf("got %v, want ErrAlreadyInitialized", err)
		}
	})

	t.Run("nil bootstrap receiver", func(t *testing.T) {
		var boot *cli.Bootstrap
		if boot.Config() != nil || boot.Initialized() || boot.Engine() != nil {
			t.Error("expected zero values for nil Bootstrap getters")
		}
		if _, err := boot.Initialize(); !errors.Is(err, cli.ErrNilBootstrap) {
			t.Errorf("got %v, want ErrNilBootstrap", err)
		}
	})
}

func TestCommandDescriptor(t *testing.T) {
	t.Run("valid command descriptor", func(t *testing.T) {
		aliases := []string{"Run", "execute", "execute"}
		desc, err := cli.NewCommandDescriptor("Run", "Run Command", "Executes pipeline", aliases, "Core", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if desc.ID() != "run" {
			t.Errorf("got ID %q, want run", desc.ID())
		}
		if desc.Name() != "Run Command" {
			t.Errorf("got Name %q, want Run Command", desc.Name())
		}
		if desc.Description() != "Executes pipeline" {
			t.Errorf("got Description %q", desc.Description())
		}
		if desc.Category() != "Core" {
			t.Errorf("got Category %q", desc.Category())
		}
		if desc.Hidden() {
			t.Error("expected Hidden == false")
		}

		// Alias deduplication and ID exclusion ("Run" excluded because it equals ID "run")
		if len(desc.Aliases()) != 1 || desc.Aliases()[0] != "execute" {
			t.Errorf("got aliases %v, want [execute]", desc.Aliases())
		}

		// Defensive copy of aliases
		aliases[0] = "mutated"
		if desc.Aliases()[0] == "mutated" {
			t.Error("input slice mutation leaked into CommandDescriptor")
		}
	})

	t.Run("invalid ID errors", func(t *testing.T) {
		_, err := cli.NewCommandDescriptor("  ", "Run", "Desc", nil, "Cat", false)
		if !errors.Is(err, cli.ErrInvalidID) {
			t.Errorf("got %v, want ErrInvalidID", err)
		}
		_, err = cli.NewCommandDescriptor("run cmd", "Run", "Desc", nil, "Cat", false)
		if err == nil || !errors.Is(err, cli.ErrInvalidID) {
			t.Errorf("got %v, want ErrInvalidID for spaces", err)
		}
	})

	t.Run("invalid Name error", func(t *testing.T) {
		_, err := cli.NewCommandDescriptor("run", "  ", "Desc", nil, "Cat", false)
		if !errors.Is(err, cli.ErrInvalidName) {
			t.Errorf("got %v, want ErrInvalidName", err)
		}
	})

	t.Run("nil descriptor getters", func(t *testing.T) {
		var desc *cli.CommandDescriptor
		if desc.ID() != "" || desc.Name() != "" || desc.Description() != "" || desc.Aliases() != nil || desc.Category() != "" || desc.Hidden() {
			t.Error("expected zero values for nil CommandDescriptor getters")
		}
	})
}

func TestCommandRegistry(t *testing.T) {
	reg := cli.NewCommandRegistry()
	c1, _ := cli.NewCommandDescriptor("run", "Run", "Run engine", []string{"r", "exec"}, "core", false)
	c2, _ := cli.NewCommandDescriptor("build", "Build", "Build project", []string{"b"}, "core", false)

	if err := reg.Register(c1); err != nil {
		t.Fatalf("Register c1 failed: %v", err)
	}
	if err := reg.Register(c2); err != nil {
		t.Fatalf("Register c2 failed: %v", err)
	}

	if reg.Count() != 2 {
		t.Errorf("got count %d, want 2", reg.Count())
	}
	if !reg.Has("run") || !reg.Has("r") || !reg.Has("exec") || !reg.Has("build") {
		t.Error("expected Has to return true for IDs and aliases")
	}

	// Lookup by ID & alias
	gotID, err := reg.Get("RUN")
	if err != nil || gotID.ID() != "run" {
		t.Errorf("Get(RUN) got %v, %v", gotID, err)
	}
	gotAlias, err := reg.Get("r")
	if err != nil || gotAlias.ID() != "run" {
		t.Errorf("Get(r) got %v, %v", gotAlias, err)
	}

	// Duplicate registration error
	err = reg.Register(c1)
	if !errors.Is(err, cli.ErrDuplicateCommand) {
		t.Errorf("got %v, want ErrDuplicateCommand", err)
	}

	// Alias collision error
	cColliding, _ := cli.NewCommandDescriptor("exec", "Exec", "Desc", nil, "core", false)
	err = reg.Register(cColliding)
	if !errors.Is(err, cli.ErrDuplicateCommand) {
		t.Errorf("got %v, want ErrDuplicateCommand for alias collision", err)
	}

	// Unknown command
	if _, err := reg.Get("missing"); !errors.Is(err, cli.ErrCommandNotFound) {
		t.Errorf("got %v, want ErrCommandNotFound", err)
	}

	// List ordering
	list := reg.List()
	if len(list) != 2 || list[0].ID() != "run" || list[1].ID() != "build" {
		t.Errorf("List() ordering mismatch: %v", list)
	}

	// Nil registry safety
	var nilReg *cli.CommandRegistry
	if err := nilReg.Register(c1); !errors.Is(err, cli.ErrNilCommandRegistry) {
		t.Errorf("got %v, want ErrNilCommandRegistry", err)
	}
	if _, err := nilReg.Get("run"); !errors.Is(err, cli.ErrNilCommandRegistry) {
		t.Errorf("got %v, want ErrNilCommandRegistry", err)
	}
	if nilReg.Has("run") || nilReg.Count() != 0 || nilReg.List() != nil {
		t.Error("expected zero values for nil CommandRegistry getters")
	}
}

func TestRouter(t *testing.T) {
	reg := cli.NewCommandRegistry()
	c1, _ := cli.NewCommandDescriptor("start", "Start Engine", "Starts engine coordinator", []string{"boot"}, "core", false)
	_ = reg.Register(c1)

	t.Run("nil registry error", func(t *testing.T) {
		_, err := cli.NewRouter(nil)
		if !errors.Is(err, cli.ErrNilCommandRegistry) {
			t.Errorf("got %v, want ErrNilCommandRegistry", err)
		}
	})

	router, err := cli.NewRouter(reg)
	if err != nil {
		t.Fatalf("NewRouter failed: %v", err)
	}
	if router.Registry() != reg {
		t.Error("registry mismatch")
	}

	// Resolve by ID
	desc, err := router.Resolve("START")
	if err != nil || desc.ID() != "start" {
		t.Errorf("Resolve(START) got %v, %v", desc, err)
	}

	// Resolve by alias
	descAlias, err := router.Resolve("boot")
	if err != nil || descAlias.ID() != "start" {
		t.Errorf("Resolve(boot) got %v, %v", descAlias, err)
	}

	// Empty identifier error
	if _, err := router.Resolve("   "); !errors.Is(err, cli.ErrEmptyIdentifier) {
		t.Errorf("got %v, want ErrEmptyIdentifier", err)
	}

	// Unresolved command error
	if _, err := router.Resolve("unknown"); !errors.Is(err, cli.ErrCommandUnresolved) {
		t.Errorf("got %v, want ErrCommandUnresolved", err)
	}

	// Nil router receiver
	var nilRouter *cli.Router
	if nilRouter.Registry() != nil {
		t.Error("expected nil Registry for nil Router")
	}
	if _, err := nilRouter.Resolve("start"); !errors.Is(err, cli.ErrNilRouter) {
		t.Errorf("got %v, want ErrNilRouter", err)
	}
}

func TestTerminalRenderer(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	t.Run("nil writer error", func(t *testing.T) {
		_, err := cli.NewTerminalRenderer(nil, stderr)
		if !errors.Is(err, cli.ErrNilWriter) {
			t.Errorf("got %v, want ErrNilWriter", err)
		}
	})

	rend, err := cli.NewTerminalRenderer(stdout, stderr)
	if err != nil {
		t.Fatalf("NewTerminalRenderer failed: %v", err)
	}
	if rend.Stdout() != stdout || rend.Stderr() != stderr {
		t.Error("writer mismatch")
	}

	// RenderMessage Info -> stdout
	if err := rend.RenderMessage(cli.MessageInfo, "info message"); err != nil {
		t.Fatalf("RenderMessage failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "[INFO] info message") {
		t.Errorf("stdout got %q", stdout.String())
	}
	stdout.Reset()

	// RenderMessage Error -> stderr
	if err := rend.RenderMessage(cli.MessageError, "error message"); err != nil {
		t.Fatalf("RenderMessage failed: %v", err)
	}
	if !strings.Contains(stderr.String(), "[ERROR] error message") {
		t.Errorf("stderr got %q", stderr.String())
	}
	stderr.Reset()

	// RenderCommand
	cmd, _ := cli.NewCommandDescriptor("test", "Test Command", "Test Desc", []string{"t"}, "test", false)
	if err := rend.RenderCommand(cmd); err != nil {
		t.Fatalf("RenderCommand failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "Test Command (test)") {
		t.Errorf("stdout got %q", stdout.String())
	}
	stdout.Reset()

	// RenderCommandList
	if err := rend.RenderCommandList([]*cli.CommandDescriptor{cmd}); err != nil {
		t.Fatalf("RenderCommandList failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "test") || !strings.Contains(stdout.String(), "Test Command") {
		t.Errorf("stdout got %q", stdout.String())
	}
	stdout.Reset()

	// RenderTable
	headers := []string{"COL1", "COL2"}
	rows := [][]string{{"val1", "val2"}}
	if err := rend.RenderTable(headers, rows); err != nil {
		t.Fatalf("RenderTable failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "COL1") || !strings.Contains(stdout.String(), "val1") {
		t.Errorf("stdout got %q", stdout.String())
	}
	stdout.Reset()

	// RenderKeyValue
	pairs := map[string]string{"Key1": "Val1"}
	if err := rend.RenderKeyValue(pairs); err != nil {
		t.Fatalf("RenderKeyValue failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "Key1:") || !strings.Contains(stdout.String(), "Val1") {
		t.Errorf("stdout got %q", stdout.String())
	}
	stdout.Reset()

	// Nil terminal renderer receiver
	var nilRend *cli.TerminalRenderer
	if nilRend.Stdout() != nil || nilRend.Stderr() != nil {
		t.Error("expected nil writers for nil TerminalRenderer")
	}
	if err := nilRend.RenderMessage(cli.MessageInfo, "msg"); !errors.Is(err, cli.ErrNilTerminalRenderer) {
		t.Errorf("got %v, want ErrNilTerminalRenderer", err)
	}
	if err := nilRend.RenderCommand(cmd); !errors.Is(err, cli.ErrNilTerminalRenderer) {
		t.Errorf("got %v, want ErrNilTerminalRenderer", err)
	}
	if err := nilRend.RenderCommandList([]*cli.CommandDescriptor{cmd}); !errors.Is(err, cli.ErrNilTerminalRenderer) {
		t.Errorf("got %v, want ErrNilTerminalRenderer", err)
	}
	if err := nilRend.RenderTable(headers, rows); !errors.Is(err, cli.ErrNilTerminalRenderer) {
		t.Errorf("got %v, want ErrNilTerminalRenderer", err)
	}
	if err := nilRend.RenderKeyValue(pairs); !errors.Is(err, cli.ErrNilTerminalRenderer) {
		t.Errorf("got %v, want ErrNilTerminalRenderer", err)
	}
}

func TestMessageTypeStrings(t *testing.T) {
	types := map[cli.MessageType]string{
		cli.MessageInfo:    "INFO",
		cli.MessageSuccess: "SUCCESS",
		cli.MessageWarning: "WARNING",
		cli.MessageError:   "ERROR",
		cli.MessageType(99): "UNKNOWN(99)",
	}

	for msgType, str := range types {
		if msgType.String() != str {
			t.Errorf("got %q, want %q", msgType.String(), str)
		}
	}
}

func TestConcurrentCommandRegistryReads(t *testing.T) {
	reg := cli.NewCommandRegistry()
	c1, _ := cli.NewCommandDescriptor("run", "Run", "Desc", []string{"r"}, "core", false)
	_ = reg.Register(c1)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_, _ = reg.Get("run")
				_, _ = reg.Get("r")
				_ = reg.Has("run")
				_ = reg.Count()
				_ = reg.List()
			}
		}()
	}
	wg.Wait()
}
