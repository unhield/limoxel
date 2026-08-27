package cli_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/cli"
	"github.com/unhield/limoxel/internal/version"
)

func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create a sample Go repository structure
	mainGo := `package main

import (
	"fmt"
	"github.com/sample/pkg/user"
)

func main() {
	fmt.Println(user.GetName())
}
`
	userGo := `package user

// GetName returns user name.
func GetName() string {
	return "Alice"
}
`
	goMod := `module github.com/sample

go 1.26.5
`
	configJSON := `{"app": "sample", "port": 8080}`

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(configJSON), 0644); err != nil {
		t.Fatalf("failed to write config.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}
	pkgDir := filepath.Join(dir, "pkg", "user")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatalf("failed to mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "user.go"), []byte(userGo), 0644); err != nil {
		t.Fatalf("failed to write user.go: %v", err)
	}

	return dir
}

func executeCommand(app *cli.App, args ...string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	app.SetStreams(&stdout, &stderr, strings.NewReader(""))
	code := app.Run(args)
	return code, stdout.String(), stderr.String()
}

func TestCLIFramework(t *testing.T) {
	app := cli.NewApp()

	t.Run("flags parsing", func(t *testing.T) {
		flags, err := cli.ParseFlags([]string{
			"--repo=/tmp/myrepo",
			"--format=json",
			"-v",
			"--depth", "5",
			"search", "symbol", "GetName",
		})
		if err != nil {
			t.Fatalf("ParseFlags failed: %v", err)
		}

		if flags.String("repo", "") != "/tmp/myrepo" {
			t.Errorf("got repo %q, want /tmp/myrepo", flags.String("repo", ""))
		}
		if flags.Format() != cli.FormatJSON {
			t.Errorf("got format %v, want FormatJSON", flags.Format())
		}
		if !flags.Bool("verbose") {
			t.Error("expected verbose flag to be true")
		}
		if flags.Int("depth", 0) != 5 {
			t.Errorf("got depth %d, want 5", flags.Int("depth", 0))
		}
		if flags.NArg() != 3 {
			t.Fatalf("got %d args, want 3", flags.NArg())
		}
		if flags.Arg(0) != "search" || flags.Arg(1) != "symbol" || flags.Arg(2) != "GetName" {
			t.Errorf("got args %v, want [search symbol GetName]", flags.Args())
		}
	})

	t.Run("version command", func(t *testing.T) {
		code, out, _ := executeCommand(app, "version")
		if code != 0 {
			t.Errorf("got exit code %d, want 0", code)
		}
		if !strings.Contains(out, fmt.Sprintf("limoxel version %s", version.Version)) {
			t.Errorf("unexpected version output: %s", out)
		}

		// JSON version
		code, out, _ = executeCommand(app, "version", "--json")
		if code != 0 {
			t.Errorf("got exit code %d, want 0", code)
		}
		var vMap map[string]string
		if err := json.Unmarshal([]byte(out), &vMap); err != nil {
			t.Fatalf("invalid json version output: %v (raw: %s)", err, out)
		}
		if vMap["version"] != version.Version {
			t.Errorf("got json version %q, want %s", vMap["version"], version.Version)
		}
	})

	t.Run("root help command", func(t *testing.T) {
		code, out, _ := executeCommand(app, "--help")
		if code != 0 {
			t.Errorf("got exit code %d, want 0", code)
		}
		if !strings.Contains(out, "Repository Commands:") || !strings.Contains(out, "Search Commands:") {
			t.Errorf("help missing categories: %s", out)
		}
	})

	t.Run("command-specific help", func(t *testing.T) {
		code, out, _ := executeCommand(app, "help", "repo")
		if code != 0 {
			t.Errorf("got exit code %d, want 0", code)
		}
		if !strings.Contains(out, "Available Subcommands:") || !strings.Contains(out, "init") {
			t.Errorf("repo help missing subcommands: %s", out)
		}
	})

	t.Run("unknown command error", func(t *testing.T) {
		code, _, errOut := executeCommand(app, "nonexistent-cmd")
		if code != 2 {
			t.Errorf("got exit code %d, want 2 (ExitUsage)", code)
		}
		if !strings.Contains(errOut, "unknown command") {
			t.Errorf("expected unknown command error, got: %s", errOut)
		}
	})

	t.Run("shell completions", func(t *testing.T) {
		for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
			code, out, _ := executeCommand(app, "completion", shell)
			if code != 0 {
				t.Errorf("completion for %s failed with code %d", shell, code)
			}
			if len(out) == 0 {
				t.Errorf("completion for %s produced empty output", shell)
			}
		}

		// Invalid shell
		code, _, errOut := executeCommand(app, "completion", "invalid-shell")
		if code == 0 {
			t.Error("expected error for invalid shell")
		}
		if !strings.Contains(errOut, "unsupported shell") {
			t.Errorf("unexpected error message: %s", errOut)
		}
	})
}

func TestRepositoryCommands(t *testing.T) {
	repoDir := setupTestRepo(t)
	app := cli.NewApp()

	t.Run("repo init", func(t *testing.T) {
		code, out, _ := executeCommand(app, "repo", "init", repoDir)
		if code != 0 {
			t.Fatalf("repo init failed with code %d", code)
		}
		if !strings.Contains(out, "Initialized repository workspace") {
			t.Errorf("unexpected init output: %s", out)
		}
	})

	t.Run("repo open", func(t *testing.T) {
		code, out, _ := executeCommand(app, "repo", "open", repoDir)
		if code != 0 {
			t.Fatalf("repo open failed with code %d", code)
		}
		if !strings.Contains(out, "Opened repository:") {
			t.Errorf("unexpected open output: %s", out)
		}
	})

	t.Run("repo scan", func(t *testing.T) {
		code, out, _ := executeCommand(app, "repo", "scan", repoDir)
		if code != 0 {
			t.Fatalf("repo scan failed with code %d", code)
		}
		if !strings.Contains(out, "Scanned") || !strings.Contains(out, "main.go") {
			t.Errorf("unexpected scan output: %s", out)
		}
	})

	t.Run("repo analyze", func(t *testing.T) {
		code, out, _ := executeCommand(app, "repo", "analyze", repoDir)
		if code != 0 {
			t.Fatalf("repo analyze failed with code %d", code)
		}
		if !strings.Contains(out, "Repository analysis complete") {
			t.Errorf("unexpected analyze output: %s", out)
		}
	})

	t.Run("repo validate", func(t *testing.T) {
		code, out, _ := executeCommand(app, "repo", "validate", repoDir)
		if code != 0 {
			t.Fatalf("repo validate failed with code %d", code)
		}
		if !strings.Contains(out, "Repository validation PASSED") {
			t.Errorf("unexpected validate output: %s", out)
		}
	})

	t.Run("repo info", func(t *testing.T) {
		code, out, _ := executeCommand(app, "repo", "info", repoDir)
		if code != 0 {
			t.Fatalf("repo info failed with code %d", code)
		}
		if !strings.Contains(out, "REPOSITORY INFORMATION") {
			t.Errorf("unexpected info output: %s", out)
		}
	})

	t.Run("repo statistics", func(t *testing.T) {
		code, out, _ := executeCommand(app, "repo", "statistics", repoDir)
		if code != 0 {
			t.Fatalf("repo statistics failed with code %d", code)
		}
		if !strings.Contains(out, "REPOSITORY STATISTICS") {
			t.Errorf("unexpected stats output: %s", out)
		}
	})

	t.Run("repo reload and close", func(t *testing.T) {
		code, out, _ := executeCommand(app, "repo", "reload", repoDir)
		if code != 0 {
			t.Fatalf("repo reload failed: %s", out)
		}
		code, out, _ = executeCommand(app, "repo", "close")
		if code != 0 {
			t.Fatalf("repo close failed: %s", out)
		}
	})
}

func TestSearchCommands(t *testing.T) {
	repoDir := setupTestRepo(t)
	app := cli.NewApp()

	t.Run("unified search", func(t *testing.T) {
		code, out, _ := executeCommand(app, "search", "main", "--repo", repoDir)
		if code != 0 {
			t.Fatalf("search main failed with code %d", code)
		}
		if !strings.Contains(out, "Found") || !strings.Contains(out, "main") {
			t.Errorf("unexpected search output: %s", out)
		}
	})

	t.Run("search symbol", func(t *testing.T) {
		code, out, _ := executeCommand(app, "search", "symbol", "GetName", "--repo", repoDir)
		if code != 0 {
			t.Fatalf("search symbol failed with code %d", code)
		}
		if !strings.Contains(out, "GetName") {
			t.Errorf("expected GetName symbol in output: %s", out)
		}
	})

	t.Run("search package", func(t *testing.T) {
		code, out, _ := executeCommand(app, "search", "package", "user", "--repo", repoDir)
		if code != 0 {
			t.Fatalf("search package failed with code %d", code)
		}
		if !strings.Contains(out, "user") {
			t.Errorf("expected user package in output: %s", out)
		}
	})

	t.Run("search file", func(t *testing.T) {
		code, out, _ := executeCommand(app, "search", "file", "main.go", "--repo", repoDir)
		if code != 0 {
			t.Fatalf("search file failed with code %d", code)
		}
		if !strings.Contains(out, "main.go") {
			t.Errorf("expected main.go in output: %s", out)
		}
	})

	t.Run("search doc", func(t *testing.T) {
		code, out, _ := executeCommand(app, "search", "doc", "returns user name", "--repo", repoDir)
		if code != 0 {
			t.Fatalf("search doc failed with code %d", code)
		}
		if !strings.Contains(out, "returns user name") {
			t.Errorf("expected doc entry in output: %s", out)
		}
	})

	t.Run("search config", func(t *testing.T) {
		code, out, _ := executeCommand(app, "search", "config", "config", "--repo", repoDir)
		if code != 0 {
			t.Fatalf("search config failed with code %d", code)
		}
		if !strings.Contains(out, "config.json") {
			t.Errorf("expected config.json in output: %s", out)
		}
	})
}

func TestIntelligenceCommands(t *testing.T) {
	repoDir := setupTestRepo(t)
	app := cli.NewApp()

	t.Run("intel explain", func(t *testing.T) {
		code, out, _ := executeCommand(app, "intel", "explain", "--repo", repoDir)
		if code != 0 {
			t.Fatalf("intel explain failed with code %d", code)
		}
		if !strings.Contains(out, "ARCHITECTURE OVERVIEW") {
			t.Errorf("unexpected explain output: %s", out)
		}
	})

	t.Run("intel dependencies", func(t *testing.T) {
		code, out, _ := executeCommand(app, "intel", "dependencies", "--repo", repoDir)
		if code != 0 {
			t.Fatalf("intel dependencies failed with code %d", code)
		}
		if !strings.Contains(out, "DEPENDENCY ANALYSIS") {
			t.Errorf("unexpected dependencies output: %s", out)
		}
	})

	t.Run("intel health", func(t *testing.T) {
		code, out, _ := executeCommand(app, "intel", "health", "--repo", repoDir)
		if code != 0 {
			t.Fatalf("intel health failed with code %d", code)
		}
		if !strings.Contains(out, "REPOSITORY HEALTH & QUALITY") {
			t.Errorf("unexpected health output: %s", out)
		}
	})

	t.Run("intel inspect", func(t *testing.T) {
		code, out, _ := executeCommand(app, "intel", "inspect", "GetName", "--repo", repoDir)
		if code != 0 {
			t.Fatalf("intel inspect failed with code %d", code)
		}
		if !strings.Contains(out, "SYMBOL INSPECTION") || !strings.Contains(out, "GetName") {
			t.Errorf("unexpected inspect output: %s", out)
		}
	})

	t.Run("intel impact", func(t *testing.T) {
		code, out, _ := executeCommand(app, "intel", "impact", "GetName", "--repo", repoDir)
		if code != 0 {
			t.Fatalf("intel impact failed with code %d", code)
		}
		if !strings.Contains(out, "CHANGE IMPACT ANALYSIS") {
			t.Errorf("unexpected impact output: %s", out)
		}
	})

	t.Run("intel recommendations", func(t *testing.T) {
		code, out, _ := executeCommand(app, "intel", "recommendations", "--repo", repoDir)
		if code != 0 {
			t.Fatalf("intel recommendations failed with code %d", code)
		}
		if !strings.Contains(out, "ENGINEERING RECOMMENDATIONS") {
			t.Errorf("unexpected recommendations output: %s", out)
		}
	})
}

func TestGraphCommands(t *testing.T) {
	repoDir := setupTestRepo(t)
	app := cli.NewApp()

	t.Run("graph repo", func(t *testing.T) {
		code, out, _ := executeCommand(app, "graph", "repo", "--repo", repoDir)
		if code != 0 {
			t.Fatalf("graph repo failed with code %d", code)
		}
		if !strings.Contains(out, "ENGINEERING KNOWLEDGE GRAPH") {
			t.Errorf("unexpected graph repo output: %s", out)
		}
	})

	t.Run("graph package", func(t *testing.T) {
		code, out, _ := executeCommand(app, "graph", "package", "--repo", repoDir)
		if code != 0 {
			t.Fatalf("graph package failed with code %d", code)
		}
		if !strings.Contains(out, "PACKAGE GRAPH NODES") {
			t.Errorf("unexpected graph package output: %s", out)
		}
	})

	t.Run("graph call", func(t *testing.T) {
		code, out, _ := executeCommand(app, "graph", "call", "GetName", "--repo", repoDir)
		if code != 0 {
			t.Fatalf("graph call failed with code %d", code)
		}
		if !strings.Contains(out, "CALL GRAPH FOR \"GETNAME\"") {
			t.Errorf("unexpected graph call output: %s", out)
		}
	})

	t.Run("graph symbol", func(t *testing.T) {
		code, out, _ := executeCommand(app, "graph", "symbol", "GetName", "--repo", repoDir)
		if code != 0 {
			t.Fatalf("graph symbol failed with code %d", code)
		}
		if !strings.Contains(out, "SYMBOL GRAPH: GETNAME") {
			t.Errorf("unexpected graph symbol output: %s", out)
		}
	})
}

func TestInteractiveREPL(t *testing.T) {
	repoDir := setupTestRepo(t)
	app := cli.NewApp()

	input := strings.Join([]string{
		"version",
		"repo info " + repoDir,
		"search symbol GetName --repo " + repoDir,
		"exit",
	}, "\n")

	var stdout, stderr bytes.Buffer
	app.SetStreams(&stdout, &stderr, strings.NewReader(input))
	code := app.Run([]string{"--interactive"})
	if code != 0 {
		t.Fatalf("interactive REPL failed with code %d", code)
	}

	out := stdout.String()
	if !strings.Contains(out, fmt.Sprintf("limoxel version %s", version.Version)) || !strings.Contains(out, "Goodbye!") {
		t.Errorf("unexpected interactive output: %s", out)
	}
}

func TestDeterminismAndConcurrency(t *testing.T) {
	repoDir := setupTestRepo(t)
	app := cli.NewApp()

	t.Run("deterministic JSON output", func(t *testing.T) {
		_, out1, _ := executeCommand(app, "repo", "statistics", repoDir, "--json")
		_, out2, _ := executeCommand(app, "repo", "statistics", repoDir, "--json")
		if out1 != out2 {
			t.Errorf("nondeterministic JSON output:\nRun 1: %s\nRun 2: %s", out1, out2)
		}
	})

	t.Run("concurrent CLI runs", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				subApp := cli.NewApp()
				code, out, errOut := executeCommand(subApp, "search", "symbol", "GetName", "--repo", repoDir, "--json")
				if code != 0 || !strings.Contains(out, "GetName") {
					t.Errorf("concurrent execution failed (code: %d, out: %s, err: %s)", code, out, errOut)
				}
			}()
		}
		wg.Wait()
	})
}

func TestReportCommands(t *testing.T) {
	repoDir := setupTestRepo(t)
	app := cli.NewApp()

	t.Run("report repository", func(t *testing.T) {
		code, out, errOut := executeCommand(app, "report", "repository", repoDir)
		if code != 0 {
			t.Fatalf("report repository failed (code: %d): %s", code, errOut)
		}
		if !strings.Contains(out, "REPOSITORY ENGINEERING REPORT") {
			t.Errorf("unexpected output: %s", out)
		}
	})

	t.Run("report repository json and yaml", func(t *testing.T) {
		code, out, errOut := executeCommand(app, "report", "repository", repoDir, "--json")
		if code != 0 || !strings.HasPrefix(strings.TrimSpace(out), "{") {
			t.Fatalf("report repository --json failed (code %d): %s, err: %s", code, out, errOut)
		}

		code, out, errOut = executeCommand(app, "report", "repository", repoDir, "--format", "yaml")
		if code != 0 || !strings.Contains(out, "file_count:") {
			t.Fatalf("report repository --format yaml failed (code %d): %s, err: %s", code, out, errOut)
		}
	})

	t.Run("report architecture markdown and html", func(t *testing.T) {
		code, out, errOut := executeCommand(app, "report", "architecture", repoDir, "--format", "markdown")
		if code != 0 || !strings.Contains(out, "# Architecture Analysis Report") {
			t.Fatalf("report architecture markdown failed: %s, err: %s", out, errOut)
		}

		code, out, errOut = executeCommand(app, "report", "architecture", repoDir, "--format", "html")
		if code != 0 || !strings.Contains(out, "<!DOCTYPE html>") {
			t.Fatalf("report architecture html failed: %s, err: %s", out, errOut)
		}
	})

	t.Run("report dependency", func(t *testing.T) {
		code, out, errOut := executeCommand(app, "report", "dependency", repoDir, "--json")
		if code != 0 || !strings.Contains(out, "direct_dependencies") {
			t.Fatalf("report dependency failed: %s, err: %s", out, errOut)
		}
	})

	t.Run("report health and summary", func(t *testing.T) {
		code, out, errOut := executeCommand(app, "report", "health", repoDir)
		if code != 0 || !strings.Contains(out, "HEALTH") {
			t.Fatalf("report health failed: %s, err: %s", out, errOut)
		}

		code, out, errOut = executeCommand(app, "report", "summary", repoDir, "--format", "markdown")
		if code != 0 || !strings.Contains(out, "Executive") {
			t.Fatalf("report summary failed: %s, err: %s", out, errOut)
		}
	})

	t.Run("report file output export", func(t *testing.T) {
		outFile := filepath.Join(t.TempDir(), "reports", "repo_report.pdf")
		code, out, errOut := executeCommand(app, "report", "repository", repoDir, "--output", outFile, "--format", "pdf")
		if code != 0 {
			t.Fatalf("report repository to PDF failed: %s, err: %s", out, errOut)
		}
		if _, err := os.Stat(outFile); err != nil {
			t.Fatalf("expected output PDF file to exist: %v", err)
		}
	})
}

func TestExportCommands(t *testing.T) {
	repoDir := setupTestRepo(t)
	app := cli.NewApp()

	t.Run("export graph mermaid and dot", func(t *testing.T) {
		code, out, errOut := executeCommand(app, "export", "graph", "--repo", repoDir, "--format", "mermaid")
		if code != 0 || !strings.Contains(out, "flowchart TD") {
			t.Fatalf("export graph mermaid failed: %s, err: %s", out, errOut)
		}

		code, out, errOut = executeCommand(app, "export", "graph", "--repo", repoDir, "--format", "graphviz")
		if code != 0 || !strings.Contains(out, "digraph G {") {
			t.Fatalf("export graph graphviz failed: %s, err: %s", out, errOut)
		}
	})

	t.Run("export graph svg and png file", func(t *testing.T) {
		code, out, errOut := executeCommand(app, "export", "graph", "--repo", repoDir, "--format", "svg")
		if code != 0 || !strings.Contains(out, "<svg") {
			t.Fatalf("export graph svg failed: %s, err: %s", out, errOut)
		}

		pngFile := filepath.Join(t.TempDir(), "diagram.png")
		code, out, errOut = executeCommand(app, "export", "graph", "--repo", repoDir, "--format", "png", "--output", pngFile)
		if code != 0 {
			t.Fatalf("export graph png file failed: %s, err: %s", out, errOut)
		}
		if _, err := os.Stat(pngFile); err != nil {
			t.Fatalf("expected output PNG file to exist: %v", err)
		}
	})

	t.Run("export diagram subcommands", func(t *testing.T) {
		code, out, errOut := executeCommand(app, "export", "diagram", "dependency", "--repo", repoDir)
		if code != 0 || !strings.Contains(out, "flowchart TD") {
			t.Fatalf("export diagram dependency failed: %s, err: %s", out, errOut)
		}

		code, out, errOut = executeCommand(app, "export", "diagram", "call", "--repo", repoDir)
		if code != 0 || !strings.Contains(out, "flowchart TD") {
			t.Fatalf("export diagram call failed: %s, err: %s", out, errOut)
		}
	})
}

func TestConfigCommands(t *testing.T) {
	repoDir := setupTestRepo(t)
	app := cli.NewApp()

	t.Run("config list", func(t *testing.T) {
		code, out, errOut := executeCommand(app, "config", "list", "--repo", repoDir)
		if code != 0 || !strings.Contains(out, "output.format") {
			t.Fatalf("config list failed: %s, err: %s", out, errOut)
		}

		code, out, errOut = executeCommand(app, "config", "list", "--repo", repoDir, "--format", "json")
		if code != 0 || !strings.Contains(out, "\"key\": \"output.format\"") {
			t.Fatalf("config list json failed: %s, err: %s", out, errOut)
		}
	})

	t.Run("config get", func(t *testing.T) {
		code, out, errOut := executeCommand(app, "config", "get", "output.format", "--repo", repoDir)
		if code != 0 || !strings.Contains(out, "text") {
			t.Fatalf("config get failed: %s, err: %s", out, errOut)
		}

		code, _, _ = executeCommand(app, "config", "get", "nonexistent.key", "--repo", repoDir)
		if code == 0 {
			t.Fatal("expected non-zero exit for non-existent key")
		}
	})

	t.Run("config init and set/unset", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Init
		code, out, errOut := executeCommand(app, "config", "init", "--repo", tmpDir, "--format", "yaml")
		if code != 0 {
			t.Fatalf("config init failed: %s, err: %s", out, errOut)
		}

		// Set
		code, out, errOut = executeCommand(app, "config", "set", "output.format", "markdown", "--repo", tmpDir)
		if code != 0 {
			t.Fatalf("config set failed: %s, err: %s", out, errOut)
		}

		// Get updated
		code, out, errOut = executeCommand(app, "config", "get", "output.format", "--repo", tmpDir)
		if code != 0 || !strings.Contains(out, "markdown") {
			t.Fatalf("config get updated failed: %s, err: %s", out, errOut)
		}

		// Unset
		code, out, errOut = executeCommand(app, "config", "unset", "output.format", "--repo", tmpDir)
		if code != 0 {
			t.Fatalf("config unset failed: %s, err: %s", out, errOut)
		}
	})

	t.Run("config validate", func(t *testing.T) {
		code, out, errOut := executeCommand(app, "config", "validate", "--repo", repoDir)
		if code != 0 || !strings.Contains(out, "Configuration is valid") {
			t.Fatalf("config validate failed: %s, err: %s", out, errOut)
		}
	})

	t.Run("config profile commands", func(t *testing.T) {
		tmpApp := cli.NewApp()
		tmpDir := t.TempDir()
		_, _, _ = executeCommand(tmpApp, "config", "init", "--repo", tmpDir)

		// Create profile
		code, out, errOut := executeCommand(tmpApp, "config", "profile", "create", "production", "--repo", tmpDir)
		if code != 0 {
			t.Fatalf("profile create failed: %s, err: %s", out, errOut)
		}

		// List profiles
		code, out, errOut = executeCommand(tmpApp, "config", "profile", "list", "--repo", tmpDir)
		if code != 0 || !strings.Contains(out, "production") {
			t.Fatalf("profile list failed: %s, err: %s", out, errOut)
		}

		// Delete profile
		code, out, errOut = executeCommand(tmpApp, "config", "profile", "delete", "production", "--repo", tmpDir)
		if code != 0 {
			t.Fatalf("profile delete failed: %s, err: %s", out, errOut)
		}
	})
}

func TestDiagnosticCommands(t *testing.T) {
	repoDir := setupTestRepo(t)
	app := cli.NewApp()

	t.Run("diag command", func(t *testing.T) {
		code, out, errOut := executeCommand(app, "diag", "--repo", repoDir)
		if code != 0 || !strings.Contains(out, "OPERATIONAL DIAGNOSTICS") {
			t.Fatalf("diag command failed: %s, err: %s", out, errOut)
		}

		// JSON format
		code, out, errOut = executeCommand(app, "diag", "--repo", repoDir, "--format", "json")
		if code != 0 || !strings.Contains(out, "\"category\": \"system\"") {
			t.Fatalf("diag command json failed: %s, err: %s", out, errOut)
		}
	})

	t.Run("health command", func(t *testing.T) {
		code, out, errOut := executeCommand(app, "health", "--repo", repoDir)
		if code != 0 || !strings.Contains(out, "OPERATIONAL HEALTH REPORT") {
			t.Fatalf("health command failed: %s, err: %s", out, errOut)
		}

		// JSON format
		code, out, errOut = executeCommand(app, "health", "--repo", repoDir, "--format", "json")
		if code != 0 || !strings.Contains(out, "\"overall_status\": \"healthy\"") {
			t.Fatalf("health command json failed: %s, err: %s", out, errOut)
		}
	})

	t.Run("log command", func(t *testing.T) {
		code, out, errOut := executeCommand(app, "log")
		if code != 0 || !strings.Contains(out, "OPERATIONAL LOG BUFFER") {
			t.Fatalf("log command failed: %s, err: %s", out, errOut)
		}
	})

	t.Run("debug command", func(t *testing.T) {
		code, out, errOut := executeCommand(app, "debug", "--repo", repoDir)
		if code != 0 || !strings.Contains(out, "OPERATIONAL STATE DUMP") {
			t.Fatalf("debug dump failed: %s, err: %s", out, errOut)
		}

		// Debug trace
		code, out, errOut = executeCommand(app, "debug", "trace")
		if code != 0 || !strings.Contains(out, "EXECUTION TRACE SPANS") {
			t.Fatalf("debug trace failed: %s, err: %s", out, errOut)
		}
	})

	t.Run("profile command", func(t *testing.T) {
		code, out, errOut := executeCommand(app, "profile", "stats")
		if code != 0 || !strings.Contains(out, "RUNTIME RESOURCE STATISTICS") {
			t.Fatalf("profile stats failed: %s, err: %s", out, errOut)
		}

		code, out, errOut = executeCommand(app, "profile")
		if code != 0 || !strings.Contains(out, "Profiling subsystem online") {
			t.Fatalf("profile default failed: %s, err: %s", out, errOut)
		}
	})
}

func BenchmarkCLICommandExecution(b *testing.B) {
	repoDir := setupTestRepoBench(b)
	app := cli.NewApp()

	b.ResetTimer()
	for b.Loop() {
		var stdout, stderr bytes.Buffer
		app.SetStreams(&stdout, &stderr, strings.NewReader(""))
		code := app.Run([]string{"search", "symbol", "GetName", "--repo", repoDir, "--json"})
		if code != 0 {
			b.Fatalf("benchmark failed with code %d", code)
		}
	}
}

func setupTestRepoBench(b *testing.B) string {
	b.Helper()
	dir := b.TempDir()
	mainGo := `package main
func main() {}
func GetName() string { return "test" }
`
	goMod := `module github.com/bench
go 1.26.5
`
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0644)
	_ = os.WriteFile(filepath.Join(dir, "main.go"), []byte(mainGo), 0644)
	return dir
}
