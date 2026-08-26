package cli

// OutputFormat defines the presentation format for command output and exports.
type OutputFormat string

const (
	// Human-readable console format
	FormatText OutputFormat = "text"

	// Machine-oriented structured formats
	FormatJSON OutputFormat = "json"
	FormatYAML OutputFormat = "yaml"
	FormatTOML OutputFormat = "toml"
	FormatXML  OutputFormat = "xml"
	FormatCSV  OutputFormat = "csv"

	// Document formats
	FormatMarkdown OutputFormat = "markdown"
	FormatHTML     OutputFormat = "html"
	FormatPDF      OutputFormat = "pdf"

	// Visualization diagram formats
	FormatMermaid     OutputFormat = "mermaid"
	FormatGraphviz    OutputFormat = "graphviz"
	FormatSVG         OutputFormat = "svg"
	FormatPNG         OutputFormat = "png"
	FormatInteractive OutputFormat = "interactive"
)

// ExitCode represents standard POSIX process exit status codes.
type ExitCode int

const (
	// ExitSuccess indicates successful command completion.
	ExitSuccess ExitCode = 0

	// ExitFailure indicates a general command or capability failure.
	ExitFailure ExitCode = 1

	// ExitUsage indicates invalid command usage, arguments, or options.
	ExitUsage ExitCode = 2
)

// CommandCategory represents organizational categories for grouping CLI commands.
type CommandCategory string

const (
	// CategoryRepository groups repository lifecycle and inspection commands.
	CategoryRepository CommandCategory = "Repository Commands"

	// CategorySearch groups engineering search commands.
	CategorySearch CommandCategory = "Search Commands"

	// CategoryIntelligence groups engineering intelligence commands.
	CategoryIntelligence CommandCategory = "Intelligence Commands"

	// CategoryGraph groups knowledge graph commands.
	CategoryGraph CommandCategory = "Graph Commands"

	// CategoryReporting groups reporting and visualization export commands.
	CategoryReporting CommandCategory = "Reporting & Export Commands"

	// CategoryConfiguration groups configuration and profile management commands.
	CategoryConfiguration CommandCategory = "Configuration Commands"

	// CategoryDiagnostics groups operational diagnostics, logging, profiling, and health commands.
	CategoryDiagnostics CommandCategory = "Diagnostics & Health Commands"

	// CategoryGeneral groups system, help, version, and utility commands.
	CategoryGeneral CommandCategory = "General Commands"
)
