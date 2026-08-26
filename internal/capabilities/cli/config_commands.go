package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/cli/config"
)

// RegisterConfigCommands registers the config command family on app.
func RegisterConfigCommands(app *App) {
	configCmd := NewCommand(
		"config",
		"Inspect, modify, validate, and manage Limoxel configuration and profiles",
		"limoxel config <subcommand> [options]",
		CategoryConfiguration,
		handleConfigList,
	).AddAlias("cfg").AddAlias("settings")

	// 1. config list
	listCmd := NewCommand(
		"list",
		"List all effective configuration entries with source metadata and provenance",
		"limoxel config list [options]",
		CategoryConfiguration,
		handleConfigList,
	).AddAlias("ls").AddAlias("show").
		AddOption("format", "f", "Output format (text, json, yaml, toml, xml, csv)", "text").
		AddOption("redact", "", "Redact sensitive secret values in output", "true")
	configCmd.AddSubcommand(listCmd)

	// 2. config get
	getCmd := NewCommand(
		"get",
		"Get the resolved value for a specific configuration key",
		"limoxel config get <key> [options]",
		CategoryConfiguration,
		handleConfigGet,
	).AddOption("format", "f", "Output format (text, json, yaml, toml, xml)", "text")
	configCmd.AddSubcommand(getCmd)

	// 3. config set
	setCmd := NewCommand(
		"set",
		"Set a configuration key in the active configuration file",
		"limoxel config set <key> <value> [options]",
		CategoryConfiguration,
		handleConfigSet,
	).AddOption("config", "c", "Explicit configuration file to write to", "")
	configCmd.AddSubcommand(setCmd)

	// 4. config unset
	unsetCmd := NewCommand(
		"unset",
		"Unset a configuration key from the active configuration file",
		"limoxel config unset <key> [options]",
		CategoryConfiguration,
		handleConfigUnset,
	).AddAlias("rm").AddAlias("delete").
		AddOption("config", "c", "Explicit configuration file to modify", "")
	configCmd.AddSubcommand(unsetCmd)

	// 5. config validate
	valCmd := NewCommand(
		"validate",
		"Validate active configuration or a specific configuration file against schema rules",
		"limoxel config validate [file] [options]",
		CategoryConfiguration,
		handleConfigValidate,
	).AddAlias("check").
		AddOption("format", "f", "Output format (text, json, yaml, toml, xml)", "text")
	configCmd.AddSubcommand(valCmd)

	// 6. config init
	initCmd := NewCommand(
		"init",
		"Initialize a new baseline configuration file in the workspace",
		"limoxel config init [options]",
		CategoryConfiguration,
		handleConfigInit,
	).AddOption("format", "f", "Config file format (yaml, json, toml)", "yaml").
		AddOption("force", "", "Force overwrite if file exists", "false")
	configCmd.AddSubcommand(initCmd)

	// 7. config profile
	profCmd := NewCommand(
		"profile",
		"List, create, or manage named configuration profiles",
		"limoxel config profile <list|create|delete> [name] [options]",
		CategoryConfiguration,
		handleConfigProfile,
	).AddOption("format", "f", "Output format (text, json, yaml, toml, xml)", "text")
	configCmd.AddSubcommand(profCmd)

	app.RegisterCommand(configCmd)
}

// 1. config list
func handleConfigList(ctx *Context, flags *Flags) error {
	cfgMgr, err := ctx.EnsureConfigManager()
	if err != nil {
		return err
	}

	entries := cfgMgr.Inspect(true)
	format := flags.Format()

	if format == FormatJSON || format == FormatYAML || format == FormatTOML || format == FormatXML || format == FormatCSV {
		return ctx.Formatter().RenderStructured(entries)
	}

	ctx.Formatter().RenderSection(fmt.Sprintf("Effective Configuration (Profile: %s)", cfgMgr.Effective().Profile()))

	headers := []string{"Key", "Value", "Type", "Source", "Precedence"}
	var rows [][]string

	for _, e := range entries {
		valStr := fmt.Sprint(e.Value)
		if len(valStr) > 40 {
			valStr = valStr[:37] + "..."
		}
		rows = append(rows, []string{
			e.Key,
			valStr,
			string(e.Type),
			string(e.Source),
			e.Precedence.String(),
		})
	}

	return ctx.Formatter().RenderTable(headers, rows)
}

// 2. config get <key>
func handleConfigGet(ctx *Context, flags *Flags) error {
	if flags.NArg() == 0 {
		return fmt.Errorf("missing configuration key argument (e.g. 'limoxel config get output.format')")
	}

	key := strings.ToLower(strings.TrimSpace(flags.Arg(0)))
	cfgMgr, err := ctx.EnsureConfigManager()
	if err != nil {
		return err
	}

	entry, exists := cfgMgr.Effective().GetEntry(key)
	if !exists {
		return fmt.Errorf("configuration key %q is not set", key)
	}

	if config.IsSecretKey(key) {
		entry.Value = config.MaskedValueConstant
	}

	format := flags.Format()
	if format == FormatJSON || format == FormatYAML || format == FormatTOML || format == FormatXML {
		return ctx.Formatter().RenderStructured(entry)
	}

	_, err = fmt.Fprintf(ctx.Formatter().Stdout(), "%v\n", entry.Value)
	return err
}

// 3. config set <key> <value>
func handleConfigSet(ctx *Context, flags *Flags) error {
	if flags.NArg() < 2 {
		return fmt.Errorf("usage: limoxel config set <key> <value>")
	}

	key := strings.ToLower(strings.TrimSpace(flags.Arg(0)))
	valStr := strings.TrimSpace(flags.Arg(1))

	cfgMgr, err := ctx.EnsureConfigManager()
	if err != nil {
		return err
	}

	targetConfig := flags.String("config", "")
	if err := cfgMgr.Set(key, valStr, targetConfig); err != nil {
		return err
	}

	return ctx.Formatter().RenderSuccess(fmt.Sprintf("Configuration key %q set to %q", key, config.MaskValue(key, valStr)))
}

// 4. config unset <key>
func handleConfigUnset(ctx *Context, flags *Flags) error {
	if flags.NArg() == 0 {
		return fmt.Errorf("usage: limoxel config unset <key>")
	}

	key := strings.ToLower(strings.TrimSpace(flags.Arg(0)))
	cfgMgr, err := ctx.EnsureConfigManager()
	if err != nil {
		return err
	}

	targetConfig := flags.String("config", "")
	if err := cfgMgr.Unset(key, targetConfig); err != nil {
		return err
	}

	return ctx.Formatter().RenderSuccess(fmt.Sprintf("Configuration key %q removed", key))
}

// 5. config validate [file]
func handleConfigValidate(ctx *Context, flags *Flags) error {
	cfgMgr, err := ctx.EnsureConfigManager()
	if err != nil {
		return err
	}

	var valResult config.ValidationResult
	if flags.NArg() > 0 {
		targetFile := flags.Arg(0)
		valResult = cfgMgr.ValidateFile(targetFile)
	} else {
		entries := cfgMgr.Effective().AllEntries()
		validator := config.NewValidator()
		valResult = validator.Validate(entries)
	}

	format := flags.Format()
	if format == FormatJSON || format == FormatYAML || format == FormatTOML || format == FormatXML {
		return ctx.Formatter().RenderStructured(valResult)
	}

	if !valResult.Valid {
		_ = ctx.Formatter().RenderError(fmt.Sprintf("Configuration validation failed with %d error(s):", len(valResult.Errors)))
		for _, e := range valResult.Errors {
			_ = ctx.Formatter().RenderStatus("ERROR", fmt.Sprintf("[%s] %s: %s", e.Code, e.Key, e.Message))
		}
		return fmt.Errorf("configuration validation failed")
	}

	_ = ctx.Formatter().RenderSuccess("Configuration is valid with 0 errors.")
	if len(valResult.Warnings) > 0 {
		for _, w := range valResult.Warnings {
			_ = ctx.Formatter().RenderStatus("WARN", fmt.Sprintf("%s: %s", w.Key, w.Message))
		}
	}
	return nil
}

// 6. config init [--format yaml|json|toml] [--force]
func handleConfigInit(ctx *Context, flags *Flags) error {
	format := flags.String("format", "yaml")
	if format == "" {
		format = "yaml"
	}

	destFileName := ".limoxel.yaml"
	switch strings.ToLower(format) {
	case "json":
		destFileName = ".limoxel.json"
	case "toml":
		destFileName = ".limoxel.toml"
	default:
		destFileName = ".limoxel.yaml"
	}

	destFile := filepath.Join(ctx.RepoPath(), destFileName)
	if _, err := os.Stat(destFile); err == nil && !flags.Bool("force") {
		return fmt.Errorf("configuration file %q already exists (use --force to overwrite)", destFileName)
	}

	model := &config.ConfigFileModel{
		Version:       "1.0.0",
		ActiveProfile: "default",
		Repository: map[string]any{
			"root":             ".",
			"max_file_size_mb": 10,
			"indexing_mode":    "standard",
			"exclude_patterns": []string{".git", "vendor", "node_modules", "dist"},
		},
		Analysis: map[string]any{
			"strict_mode":             false,
			"max_depth":               15,
			"rule_severity_threshold": "info",
		},
		Output: map[string]any{
			"format":         "text",
			"color":          true,
			"theme":          "dark",
			"file_overwrite": false,
		},
		Logging: map[string]any{
			"level":  "info",
			"format": "text",
		},
		Performance: map[string]any{
			"workers":         4,
			"timeout_seconds": 60,
		},
	}

	fileStore := config.NewFileStore()
	if err := fileStore.SaveFile(destFile, model, format); err != nil {
		return err
	}

	return ctx.Formatter().RenderSuccess(fmt.Sprintf("Initialized configuration file %q", destFile))
}

// 7. config profile <list|create|delete> [name]
func handleConfigProfile(ctx *Context, flags *Flags) error {
	cfgMgr, err := ctx.EnsureConfigManager()
	if err != nil {
		return err
	}

	subAction := "list"
	if flags.NArg() > 0 {
		subAction = strings.ToLower(flags.Arg(0))
	}

	switch subAction {
	case "list", "ls":
		profiles := cfgMgr.ListProfiles()
		sort.Strings(profiles)
		format := flags.Format()
		if format == FormatJSON || format == FormatYAML || format == FormatTOML || format == FormatXML {
			return ctx.Formatter().RenderStructured(map[string]any{
				"active_profile": cfgMgr.Effective().Profile(),
				"profiles":       profiles,
			})
		}
		ctx.Formatter().RenderSection("Configuration Profiles")
		for _, p := range profiles {
			prefix := "  "
			if p == cfgMgr.Effective().Profile() {
				prefix = "* "
			}
			_, _ = fmt.Fprintf(ctx.Formatter().Stdout(), "%s%s\n", prefix, p)
		}
		return nil

	case "create", "add":
		if flags.NArg() < 2 {
			return fmt.Errorf("usage: limoxel config profile create <name>")
		}
		profName := flags.Arg(1)
		prof := config.Profile{
			Name:        profName,
			Description: "Custom profile",
			Values:      make(map[string]any),
		}
		targetConfig := flags.String("config", "")
		if err := cfgMgr.CreateProfile(prof, targetConfig); err != nil {
			return err
		}
		return ctx.Formatter().RenderSuccess(fmt.Sprintf("Profile %q created", profName))

	case "delete", "rm":
		if flags.NArg() < 2 {
			return fmt.Errorf("usage: limoxel config profile delete <name>")
		}
		profName := flags.Arg(1)
		targetConfig := flags.String("config", "")
		if err := cfgMgr.DeleteProfile(profName, targetConfig); err != nil {
			return err
		}
		return ctx.Formatter().RenderSuccess(fmt.Sprintf("Profile %q deleted", profName))

	default:
		return fmt.Errorf("unknown profile action %q (use list, create, or delete)", subAction)
	}
}
