package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/unhield/limoxel/internal/capabilities/cli/config"
	"github.com/unhield/limoxel/internal/capabilities/cli/diagnostics"
	origcli "github.com/unhield/limoxel/internal/cli"
)

// App is the top-level application coordinator managing CLI command lifecycle and execution.
type App struct {
	mu          sync.RWMutex
	name        string
	version     string
	description string
	commands    map[string]*Command
	order       []string
	registry    *origcli.CommandRegistry
	router      *origcli.Router
	stdout      io.Writer
	stderr      io.Writer
	stdin       io.Reader
}

// NewApp constructs an initialized App with default standard streams and registered commands.
func NewApp() *App {
	app := &App{
		name:        "limoxel",
		version:     "1.3.0",
		description: "Limoxel Developer Command Line Interface — Engineering Intelligence Platform",
		commands:    make(map[string]*Command),
		order:       make([]string, 0),
		registry:    origcli.NewCommandRegistry(),
		stdout:      os.Stdout,
		stderr:      os.Stderr,
		stdin:       os.Stdin,
	}

	router, _ := origcli.NewRouter(app.registry)
	app.router = router

	// Register all command groups
	RegisterRepositoryCommands(app)
	RegisterSearchCommands(app)
	RegisterIntelligenceCommands(app)
	RegisterGraphCommands(app)
	RegisterReportCommands(app)
	RegisterExportCommands(app)
	RegisterConfigCommands(app)
	RegisterDiagnosticCommands(app)
	RegisterSystemCommands(app)

	return app
}

// Name returns the application name.
func (a *App) Name() string {
	if a == nil {
		return "limoxel"
	}
	return a.name
}

// Version returns the application version string.
func (a *App) Version() string {
	if a == nil {
		return "1.3.0"
	}
	return a.version
}

// SetVersion sets the application version.
func (a *App) SetVersion(v string) *App {
	if a == nil {
		return nil
	}
	a.version = v
	return a
}

// SetStreams configures custom standard I/O streams for testing.
func (a *App) SetStreams(stdout, stderr io.Writer, stdin io.Reader) *App {
	if a == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if stdout != nil {
		a.stdout = stdout
	}
	if stderr != nil {
		a.stderr = stderr
	}
	if stdin != nil {
		a.stdin = stdin
	}
	return a
}

// RegisterCommand registers a top-level Command in the application.
func (a *App) RegisterCommand(cmd *Command) *App {
	if a == nil || cmd == nil {
		return a
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	id := cmd.ID()
	a.commands[id] = cmd
	for _, alias := range cmd.Aliases() {
		a.commands[alias] = cmd
	}
	a.order = append(a.order, id)

	// Register in foundation CommandRegistry
	if desc, err := cmd.ToDescriptor(); err == nil {
		_ = a.registry.Register(desc)
	}

	return a
}

// GetCommand retrieves a registered Command by name or alias.
func (a *App) GetCommand(name string) *Command {
	if a == nil {
		return nil
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	clean := strings.ToLower(strings.TrimSpace(name))
	return a.commands[clean]
}

// Commands returns all registered top-level commands.
func (a *App) Commands() []*Command {
	if a == nil {
		return nil
	}
	a.mu.RLock()
	defer a.mu.RUnlock()

	res := make([]*Command, 0, len(a.order))
	seen := make(map[string]bool)
	for _, id := range a.order {
		if cmd, ok := a.commands[id]; ok && !seen[cmd.id] {
			seen[cmd.id] = true
			res = append(res, cmd)
		}
	}
	return res
}

// Run executes the CLI with the provided raw command-line argument slice and returns the exit code.
func (a *App) Run(rawArgs []string) int {
	if a == nil {
		return int(ExitFailure)
	}

	flags, err := ParseFlags(rawArgs)
	if err != nil {
		fmt.Fprintf(a.stderr, "[ERROR] %v\n", err)
		return int(ExitUsage)
	}

	ctx, err := NewContext(a, a.stdout, a.stderr, flags.Format())
	if err != nil {
		fmt.Fprintf(a.stderr, "[ERROR] Failed to initialize CLI context: %v\n", err)
		return int(ExitFailure)
	}

	if flags.RepoRoot() != "" {
		ctx.SetRepoPath(flags.RepoRoot())
	}

	// Initialize configuration subsystem with workspace and flags
	if cfgMgr, errCfg := config.NewManager(func(o *config.ManagerOptions) {
		o.WorkspaceDir = ctx.RepoPath()
		o.ConfigFile = flags.ConfigPath()
		o.ActiveProfile = flags.Profile()
	}); errCfg == nil {
		ctx.SetConfigManager(cfgMgr)
	}

	// Initialize diagnostics subsystem with configuration and flags
	if diagMgr, errDiag := diagnostics.NewManager(diagnostics.ManagerOptions{
		WorkspaceDir: ctx.RepoPath(),
		Config:       ctx.Config(),
		LogLevelStr:  flags.LogLevel(),
		LogFormatStr: flags.LogFormat(),
		LogFilePath:  flags.LogFile(),
		Verbose:      flags.Bool("verbose"),
		Debug:        flags.IsDebug(),
		Trace:        flags.IsTrace(),
		ProfileCPU:   flags.ProfileCPU(),
		ProfileMem:   flags.ProfileMem(),
		Output:       a.stderr,
	}); errDiag == nil {
		ctx.SetDiagnosticsManager(diagMgr)
		defer diagMgr.Close()
	}

	// 1. Check version flag
	if flags.Bool("version") {
		if err := handleVersion(ctx, flags); err != nil {
			_ = ctx.Formatter().RenderError(err.Error())
			return int(ExitFailure)
		}
		return int(ExitSuccess)
	}

	// 2. Check root help
	if flags.NArg() == 0 {
		if flags.Bool("help") {
			a.RenderRootHelp(a.stdout)
			return int(ExitSuccess)
		}
		if flags.Bool("interactive") {
			if err := a.RunInteractive(ctx); err != nil {
				return int(ExitFailure)
			}
			return int(ExitSuccess)
		}
		// Default to root help
		a.RenderRootHelp(a.stdout)
		return int(ExitSuccess)
	}

	// 3. Resolve top-level command
	cmdName := flags.Arg(0)
	cmd := a.GetCommand(cmdName)
	if cmd == nil {
		_ = ctx.Formatter().RenderError(fmt.Sprintf("unknown command %q. Run 'limoxel help' for available commands.", cmdName))
		return int(ExitUsage)
	}

	// Shift arguments for command execution
	cmdFlags := &Flags{
		values: flags.values,
		bools:  flags.bools,
		args:   flags.args[1:],
	}

	// Execute command
	if execErr := cmd.Execute(ctx, cmdFlags); execErr != nil {
		var cliErr *CLIError
		if errors.As(execErr, &cliErr) {
			_ = ctx.Formatter().RenderError(cliErr.Error())
			return int(cliErr.ExitCode)
		}
		_ = ctx.Formatter().RenderError(execErr.Error())
		return int(ExitFailure)
	}

	return int(ExitSuccess)
}

// RunInteractive launches the interactive terminal REPL execution loop.
func (a *App) RunInteractive(ctx *Context) error {
	if a == nil || ctx == nil {
		return ErrNilContext
	}

	ctx.Formatter().RenderSection(fmt.Sprintf("%s Interactive Shell (v%s)", a.name, a.version))
	fmt.Fprintf(a.stdout, "Type 'help' for command list, or 'exit' / 'quit' to close.\n\n")

	scanner := bufio.NewScanner(a.stdin)
	for {
		fmt.Fprintf(a.stdout, "limoxel> ")
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if line == "exit" || line == "quit" || line == "q" {
			_ = ctx.Formatter().RenderSuccess("Goodbye!")
			break
		}

		tokens := tokenizeCommand(line)
		if len(tokens) == 0 {
			continue
		}

		// Execute command in interactive context
		flags, err := ParseFlags(tokens)
		if err != nil {
			_ = ctx.Formatter().RenderError(err.Error())
			continue
		}

		cmdName := flags.Arg(0)
		cmd := a.GetCommand(cmdName)
		if cmd == nil {
			_ = ctx.Formatter().RenderError(fmt.Sprintf("unknown command %q", cmdName))
			continue
		}

		cmdFlags := &Flags{
			values: flags.values,
			bools:  flags.bools,
			args:   flags.args[1:],
		}

		if err := cmd.Execute(ctx, cmdFlags); err != nil {
			var cliErr *CLIError
			if errors.As(err, &cliErr) {
				_ = ctx.Formatter().RenderError(cliErr.Error())
			} else {
				_ = ctx.Formatter().RenderError(err.Error())
			}
		}
		fmt.Fprintln(a.stdout)
	}

	return scanner.Err()
}

func tokenizeCommand(input string) []string {
	var tokens []string
	var current strings.Builder
	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < len(input); i++ {
		b := input[i]
		if (b == '"' || b == '\'') && !inQuote {
			inQuote = true
			quoteChar = b
		} else if inQuote && b == quoteChar {
			inQuote = false
			quoteChar = 0
		} else if (b == ' ' || b == '\t') && !inQuote {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		} else {
			current.WriteByte(b)
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}

// RenderRootHelp formats and prints top-level application help grouped by category.
func (a *App) RenderRootHelp(w io.Writer) {
	if a == nil || w == nil {
		return
	}

	fmt.Fprintf(w, "%s — %s\n\n", a.name, a.description)
	fmt.Fprintf(w, "Usage: limoxel [flags] <command> [subcommand] [args...]\n\n")

	// Global options
	fmt.Fprintf(w, "Global Flags:\n")
	fmt.Fprintf(w, "  -h, --help                 Display contextual help and usage\n")
	fmt.Fprintf(w, "  -v, --version              Display executable version\n")
	fmt.Fprintf(w, "  -f, --format <text|json>   Set output presentation format (default: text)\n")
	fmt.Fprintf(w, "      --json                 Shortcut for --format json\n")
	fmt.Fprintf(w, "  -r, --repo <path>          Set target repository path (default: .)\n")
	fmt.Fprintf(w, "  -i, --interactive          Start interactive REPL terminal session\n\n")

	// Group commands by category
	cmdsByCategory := make(map[CommandCategory][]*Command)
	for _, cmd := range a.Commands() {
		if !cmd.hidden {
			cmdsByCategory[cmd.category] = append(cmdsByCategory[cmd.category], cmd)
		}
	}

	categoryOrder := []CommandCategory{
		CategoryRepository,
		CategorySearch,
		CategoryIntelligence,
		CategoryGraph,
		CategoryGeneral,
	}

	for _, cat := range categoryOrder {
		cmds := cmdsByCategory[cat]
		if len(cmds) == 0 {
			continue
		}

		sort.Slice(cmds, func(i, j int) bool {
			return cmds[i].name < cmds[j].name
		})

		fmt.Fprintf(w, "%s:\n", cat)
		for _, cmd := range cmds {
			fmt.Fprintf(w, "  %-16s %s\n", cmd.name, cmd.description)
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintf(w, "Use \"limoxel <command> --help\" or \"limoxel help <command>\" for more information on a specific command.\n")
}
