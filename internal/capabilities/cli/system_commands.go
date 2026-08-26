package cli

import (
	"fmt"
	"strings"
)

// RegisterSystemCommands registers general/system commands on app.
func RegisterSystemCommands(app *App) {
	// 1. help
	helpCmd := NewCommand(
		"help",
		"Display help and usage information for Limoxel commands",
		"limoxel help [command...]",
		CategoryGeneral,
		handleHelp,
	)
	app.RegisterCommand(helpCmd)

	// 2. version
	versionCmd := NewCommand(
		"version",
		"Display Limoxel executable and CLI version information",
		"limoxel version",
		CategoryGeneral,
		handleVersion,
	)
	app.RegisterCommand(versionCmd)

	// 3. completion
	compCmd := NewCommand(
		"completion",
		"Generate shell completion script (bash, zsh, fish, powershell)",
		"limoxel completion <bash|zsh|fish|powershell>",
		CategoryGeneral,
		handleCompletion,
	)
	app.RegisterCommand(compCmd)

	// 4. interactive
	interCmd := NewCommand(
		"interactive",
		"Start interactive Limoxel terminal REPL session",
		"limoxel interactive",
		CategoryGeneral,
		handleInteractive,
	).AddAlias("repl").AddAlias("shell")
	app.RegisterCommand(interCmd)
}

func handleHelp(ctx *Context, flags *Flags) error {
	if flags != nil && flags.NArg() > 0 {
		targetCmd := flags.Arg(0)
		cmd := ctx.App().GetCommand(targetCmd)
		if cmd != nil {
			// Subcommand help
			if flags.NArg() > 1 {
				subName := flags.Arg(1)
				if sub := cmd.GetSubcommand(subName); sub != nil {
					sub.RenderHelp(ctx.Formatter().Stdout())
					return nil
				}
			}
			cmd.RenderHelp(ctx.Formatter().Stdout())
			return nil
		}
		return UsageError("help", fmt.Sprintf("unknown command %q", targetCmd))
	}

	ctx.App().RenderRootHelp(ctx.Formatter().Stdout())
	return nil
}

func handleVersion(ctx *Context, flags *Flags) error {
	version := ctx.App().Version()
	appName := ctx.App().Name()

	if ctx.Formatter().Format() == FormatJSON {
		return ctx.Formatter().RenderJSON(map[string]string{
			"app":     appName,
			"version": version,
		})
	}

	_, err := fmt.Fprintf(ctx.Formatter().Stdout(), "%s version %s\n", appName, version)
	return err
}

func handleCompletion(ctx *Context, flags *Flags) error {
	if flags == nil || flags.NArg() == 0 {
		return UsageError("completion", "shell name required (bash, zsh, fish, powershell). Usage: limoxel completion <shell>")
	}

	shell := strings.ToLower(flags.Arg(0))
	w := ctx.Formatter().Stdout()

	switch shell {
	case "bash":
		fmt.Fprintln(w, `# bash completion for limoxel
_limoxel_completion() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    opts="repo search intel graph init open scan analyze validate reload close info stats version help completion interactive --format --repo --help --version"

    case "${prev}" in
        repo)
            COMPREPLY=( $(compgen -W "init open scan analyze validate reload close info statistics" -- ${cur}) )
            return 0
            ;;
        search)
            COMPREPLY=( $(compgen -W "symbol package module file dependency doc config" -- ${cur}) )
            return 0
            ;;
        intel)
            COMPREPLY=( $(compgen -W "inspect explain dependencies health impact navigate recommendations" -- ${cur}) )
            return 0
            ;;
        graph)
            COMPREPLY=( $(compgen -W "repo package dependency call module symbol" -- ${cur}) )
            return 0
            ;;
    esac

    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
    return 0
}
complete -F _limoxel_completion limoxel`)
	case "zsh":
		fmt.Fprintln(w, `#compdef limoxel
_limoxel() {
    local -a commands
    commands=(
        'repo:Inspect and manage Limoxel repository workspaces'
        'search:Perform repository-aware engineering search'
        'intel:Access engineering intelligence and reasoning'
        'graph:Query and inspect knowledge graph'
        'version:Display version information'
        'help:Display help information'
    )
    _describe 'command' commands
}`)
	case "fish":
		fmt.Fprintln(w, `# fish completion for limoxel
complete -c limoxel -f -a "repo search intel graph version help completion interactive"`)
	case "powershell", "ps":
		fmt.Fprintln(w, `Register-ArgumentCompleter -Native -CommandName limoxel -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)
    $commands = @('repo', 'search', 'intel', 'graph', 'init', 'open', 'scan', 'analyze', 'validate', 'info', 'stats', 'version', 'help')
    $commands | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
        [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
    }
}`)
	default:
		return UsageError("completion", fmt.Sprintf("unsupported shell %q (supported: bash, zsh, fish, powershell)", shell))
	}

	return nil
}

func handleInteractive(ctx *Context, flags *Flags) error {
	ctx.SetInteractive(true)
	return ctx.App().RunInteractive(ctx)
}
