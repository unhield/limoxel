package config

// SchemaRegistry defines the canonical schema and baseline defaults for all Limoxel configuration properties.
var SchemaRegistry = []SchemaProperty{
	// 1. General & Core
	{
		Key:         "general.version",
		Type:        TypeString,
		Default:     "1.0.0",
		Description: "Limoxel configuration specification version.",
		Required:    true,
	},
	{
		Key:         "general.active_profile",
		Type:        TypeString,
		Default:     "default",
		Description: "Active configuration profile name.",
		Required:    true,
	},

	// 2. Repository Configuration
	{
		Key:         "repository.root",
		Type:        TypeString,
		Default:     ".",
		Description: "Root workspace directory for repository indexing and analysis.",
		Required:    true,
	},
	{
		Key:         "repository.max_file_size_mb",
		Type:        TypeInt,
		Default:     10,
		Description: "Maximum individual source file size limit in megabytes.",
		Required:    false,
		MinInt:      intPtr(1),
		MaxInt:      intPtr(100),
	},
	{
		Key:         "repository.indexing_mode",
		Type:        TypeString,
		Default:     "standard",
		Description: "Indexing depth and rigor (standard, shallow, full).",
		EnumValues:  []string{"standard", "shallow", "full"},
		Required:    false,
	},
	{
		Key:         "repository.exclude_patterns",
		Type:        TypeSlice,
		Default:     []string{".git", "vendor", "node_modules", ".idea", ".vscode", "bin", "dist"},
		Description: "Glob patterns for paths excluded during scanning.",
		Required:    false,
	},

	// 3. Analysis & Intelligence
	{
		Key:         "analysis.strict_mode",
		Type:        TypeBool,
		Default:     false,
		Description: "Enables strict boundary and architectural checking rules.",
		Required:    false,
	},
	{
		Key:         "analysis.max_depth",
		Type:        TypeInt,
		Default:     15,
		Description: "Maximum traversal depth for dependency and call graph analysis.",
		Required:    false,
		MinInt:      intPtr(1),
		MaxInt:      intPtr(100),
	},
	{
		Key:         "analysis.rule_severity_threshold",
		Type:        TypeString,
		Default:     "info",
		Description: "Minimum finding severity reported (info, low, medium, high, critical).",
		EnumValues:  []string{"info", "low", "medium", "high", "critical"},
		Required:    false,
	},

	// 4. Output & Reporting
	{
		Key:         "output.format",
		Type:        TypeString,
		Default:     "text",
		Description: "Default CLI output format across commands.",
		EnumValues:  []string{"text", "json", "yaml", "toml", "xml", "csv", "markdown", "html", "pdf", "mermaid", "dot", "svg", "png", "interactive"},
		Required:    true,
	},
	{
		Key:         "output.color",
		Type:        TypeBool,
		Default:     true,
		Description: "Enable ANSI terminal coloring in interactive sessions.",
		Required:    false,
	},
	{
		Key:         "output.theme",
		Type:        TypeString,
		Default:     "dark",
		Description: "Color palette theme (dark, light, plain).",
		EnumValues:  []string{"dark", "light", "plain"},
		Required:    false,
	},
	{
		Key:         "output.file_overwrite",
		Type:        TypeBool,
		Default:     false,
		Description: "Allow overwriting existing output report files automatically.",
		Required:    false,
	},

	// 5. Logging & Diagnostics
	{
		Key:         "logging.level",
		Type:        TypeString,
		Default:     "info",
		Description: "Platform logging level (trace, debug, info, warn, error, fatal).",
		EnumValues:  []string{"trace", "debug", "info", "warn", "error", "fatal"},
		Required:    false,
	},
	{
		Key:         "logging.format",
		Type:        TypeString,
		Default:     "text",
		Description: "Log output serialization format (text, json).",
		EnumValues:  []string{"text", "json"},
		Required:    false,
	},
	{
		Key:         "logging.file",
		Type:        TypeString,
		Default:     "",
		Description: "Log file path (empty for stderr).",
		Required:    false,
	},

	// 6. Performance & Concurrency
	{
		Key:         "performance.workers",
		Type:        TypeInt,
		Default:     4,
		Description: "Number of concurrent worker goroutines for repository scanning.",
		Required:    false,
		MinInt:      intPtr(1),
		MaxInt:      intPtr(64),
	},
	{
		Key:         "performance.timeout_seconds",
		Type:        TypeInt,
		Default:     60,
		Description: "Default timeout in seconds for long-running operations.",
		Required:    false,
		MinInt:      intPtr(1),
		MaxInt:      intPtr(3600),
	},

	// 7. Deprecated legacy keys for backward compatibility checking
	{
		Key:            "general.legacy_output",
		Type:           TypeString,
		Default:        "",
		Description:    "Deprecated: Use output.format instead.",
		Deprecated:     true,
		DeprecationMsg: "general.legacy_output is deprecated; please use output.format.",
	},
}

func intPtr(i int) *int {
	return &i
}

// GetDefaultEntries returns a map of all baseline default entries populated from the SchemaRegistry.
func GetDefaultEntries() map[string]ConfigEntry {
	res := make(map[string]ConfigEntry, len(SchemaRegistry))
	for _, prop := range SchemaRegistry {
		if prop.Deprecated {
			continue
		}
		res[prop.Key] = ConfigEntry{
			Key:         prop.Key,
			Value:       prop.Default,
			Type:        prop.Type,
			Source:      SourceDefault,
			Precedence:  PrecedenceDefault,
			IsSecret:    prop.IsSecret || IsSecretKey(prop.Key),
			IsDefault:   true,
			Description: prop.Description,
		}
	}
	return res
}
