# CLI User & Developer Guide

Project   : Limoxel  
Category  : CLI  
Document  : CLI User & Developer Guide  
Version   : 1.0  
Author    : Raj Joshi  

---

# Purpose

This document provides the canonical, practical user and developer guide for operating the Limoxel Command Line Interface (CLI).

It explains how to install, configure, execute, automate, and troubleshoot the Limoxel CLI across local developer environments and automated continuous integration pipelines.

The underlying capability specifications, formal invariants, and architectural contracts remain defined in the canonical specification documents:

- CLI Core Framework: [01_Command_Line_Interface.md](01_Command_Line_Interface.md)
- Output & Reporting Engine: [02_Output_and_Reporting.md](02_Output_and_Reporting.md)
- Configuration System: [03_Configuration_System.md](03_Configuration_System.md)
- Logging & Diagnostics: [04_Logging_and_Diagnostics.md](04_Logging_and_Diagnostics.md)

---

# Audience

This guide is intended for:

1. **Software Engineers & Developers**: Using Limoxel locally to inspect codebases, analyze architecture, navigate symbols, query dependency graphs, and diagnose regressions.
2. **DevOps & Platform Engineers**: Integrating Limoxel into build pipelines, CI workflows, automated quality gates, and scheduled health checks.
3. **Software Architects & Leads**: Generating architectural scorecards, visual dependency graphs, structural boundary reports, and multi-hop change impact assessments.

---

# Prerequisites

Limoxel is a native, self-contained binary built in Go.

### System & Toolchain Requirements
- **Go Toolchain** (when compiling from repository source): Go 1.26.5+ (as declared in `go.mod`).
- **Operating Systems & Architectures** (official release targets):
  - Linux (`x86_64` / `amd64`)
  - Windows (`x86_64` / `amd64`)
  - macOS (`x86_64` / `amd64`)
- **Version Control** (optional): Git on system PATH for branch, commit, and VCS metadata detection.
- **External Graph Tools** (optional): External Graphviz tools (`dot`) are only required if you choose to render exported DOT files using third-party tools outside of Limoxel's native Mermaid, SVG, PNG, and JSON exporters.

---

# Installation & Setup

### Building from Source

To compile the `limoxel` binary directly from repository source:

```bash
# Clone the repository
git clone https://github.com/unhield/limoxel.git
cd limoxel

# Build binary into local bin directory
go build -o bin/limoxel ./cmd/limoxel
```

### Windows (PowerShell / Command Prompt)

```powershell
go build -o bin\limoxel.exe .\cmd\limoxel
```

### Adding to System PATH

Ensure the compiled binary directory is accessible in your environment PATH:

```bash
# Linux / macOS
export PATH="$PWD/bin:$PATH"

# Windows PowerShell
$env:Path += ";$PWD\bin"
```

### Verifying Installation

Verify that the CLI is installed and responsive:

```bash
limoxel version
```

Expected output:
```text
limoxel version 1.0.0
```

---

# Quick Start

Limoxel commands operate statelessly per process invocation by default, or statefully inside an interactive REPL shell.

### Option A: Direct Command Execution (One-Shot)

Run complete repository analysis and generate an executive summary in a single command:

```bash
# Analyze target workspace and print executive scorecard
limoxel report summary --repo /path/to/my-project

# Search for symbols across the target repository
limoxel search symbol HandleRequest --repo /path/to/my-project

# Run operational health check
limoxel health --repo /path/to/my-project
```

### Option B: Stateful Interactive REPL Session

Launch the interactive shell for stateful exploration:

```bash
limoxel interactive
```

Inside the interactive shell:
```text
limoxel> open /path/to/my-project
limoxel> scan
limoxel> analyze
limoxel> intel explain
limoxel> report summary
limoxel> exit
```

---

# CLI Overview

Limoxel organizes functionality into a unified hierarchy consisting of:
- **8 Command Families**: Logical categories grouping related capabilities.
- **53 Canonical Operational Endpoints**: The complete inventory of user-facing commands and subcommands.
- **9 Top-Level Repository Shortcuts**: `init`, `open`, `scan`, `analyze`, `validate`, `reload`, `close`, `info`, `stats` registered directly on the root command for rapid access.
- **Command Aliases**: Registered shorthand names for families (`cfg`, `settings`, `intelligence`, `kg`, `repository`) and subcommands (`stats`, `pkg`, `mod`, `dep`, `documentation`, `configuration`, `deps`, `nav`, `recommend`, `rec`, `sym`, `executive`, `show`, `ls`, `rm`, `delete`, `check`, `logs`, `diagnostics`, `repl`, `shell`).

```text
limoxel [global-options] <command> <subcommand> [arguments] [options]
│
├── 1. Repository Lifecycle (Category: Repository Commands)
│   ├── init [path]          # Initialize new repository workspace context (Shortcut: init)
│   ├── open [path]          # Open and inspect repository workspace (Shortcut: open)
│   ├── scan [path]          # Enumerate filesystem entities and packages (Shortcut: scan)
│   ├── analyze [path]       # Run indexing and semantic symbol extraction (Shortcut: analyze)
│   ├── validate [path]      # Validate repository structure and consistency (Shortcut: validate)
│   ├── reload [path]        # Invalidate cache and reload repository state (Shortcut: reload)
│   ├── close                # Release active workspace resources (Shortcut: close)
│   ├── info [path]          # Display repository metadata, VCS, and languages (Shortcut: info)
│   └── statistics [path]    # Quantitative metrics and breakdown (Shortcut: stats, Alias: stats)
│
├── 2. Engineering Search (Category: Search Commands)
│   ├── search <query>       # Unified multi-domain search
│   ├── search symbol <q>    # Search symbols, types, and functions
│   ├── search package <q>   # Search packages (Alias: search pkg)
│   ├── search module <q>    # Search modules (Alias: search mod)
│   ├── search file <q>      # Search files by path or extension
│   ├── search dependency <q># Search dependencies (Alias: search dep)
│   ├── search doc <q>       # Search comments and docstrings (Alias: search documentation)
│   └── search config <q>    # Search configuration files (Alias: search configuration)
│
├── 3. Engineering Intelligence (Category: Intelligence Commands)
│   ├── intel inspect <sym>  # Detailed symbol semantics, scope, and relationships
│   ├── intel explain [comp] # Architecture, layers, and boundary explanation
│   ├── intel dependencies   # Package dependency and cycle analysis (Alias: intel deps)
│   ├── intel health [path]  # Repository quality scorecard and risk breakdown
│   ├── intel impact <target># Multi-hop change impact analysis
│   ├── intel navigate <sym> # Inbound callers, definitions, and references (Alias: intel nav)
│   └── intel recommendations# Prioritized actionable recommendations (Alias: intel rec, recommend)
│
├── 4. Knowledge Graph (Category: Graph Commands)
│   ├── graph repo [path]    # Repository-level entity metrics (Alias: kg repo, graph repository)
│   ├── graph package [pkg]  # Inter-package dependency graph (Alias: graph pkg)
│   ├── graph dependency [d] # Dependency graph edges (Alias: graph dep)
│   ├── graph call <symbol>  # Caller and callee call graph
│   ├── graph module [mod]   # Module hierarchy graph (Alias: graph mod)
│   └── graph symbol <sym>   # Symbol hierarchy and cross-references (Alias: graph sym)
│
├── 5. Reporting & Export (Category: Reporting & Export Commands)
│   ├── report summary       # Executive scorecard (Alias: report executive)
│   ├── report repository    # Detailed repository characteristics (Alias: report repo)
│   ├── report architecture  # Structural boundary report (Alias: report arch)
│   ├── report dependency    # Dependency inventory report (Alias: report deps)
│   ├── report health        # Risk and quality report
│   ├── export graph         # Export knowledge graph (Mermaid, DOT, SVG, PNG, JSON, YAML)
│   └── export diagram <type># Export specialized architecture/call/package diagrams
│
├── 6. Configuration & Profiles (Category: Configuration Commands, Alias: cfg, settings)
│   ├── config list          # List effective configuration entries (Alias: config show, config ls)
│   ├── config get <key>     # Retrieve effective value for a specific key
│   ├── config set <k> <v>   # Persist configuration key-value pair
│   ├── config unset <key>   # Remove configuration key from file (Alias: config rm, delete)
│   ├── config validate [f]  # Validate configuration file syntax and bounds (Alias: config check)
│   ├── config init          # Create initial configuration template (.limoxel.yaml)
│   └── config profile <act> # List, create, or delete named profiles
│
├── 7. Diagnostics & Observability (Category: Diagnostics & Health Commands)
│   ├── log                  # Inspect recent log records from ring buffer (Alias: logs)
│   ├── diag                 # Run system, repository, and toolchain diagnostics (Alias: diagnostics)
│   ├── health               # Operational readiness check of Limoxel runtime components
│   ├── debug [trace|dump]   # Operational state dump and execution trace spans
│   └── profile [heap|stats] # Runtime memory heap profiling and resource statistics
│
└── 8. System & Utility (Category: General Commands)
    ├── help [command...]    # Contextual help documentation
    ├── version              # Executable version and build metadata
    ├── completion <shell>   # Shell autocompletion script (bash, zsh, fish, powershell)
    └── interactive          # Interactive REPL session (Alias: repl, shell)
```

---

# Global Options & Flag Semantics

Limoxel distinguishes between **Application-Wide Options** (effective across all commands), **Serialization & Output Options** (format and file redirection), and **Command-Specific Options** (scoped to specific handlers).

### 1. Application-Wide Options

| Flag | Short | Default | Accepted Values | Scope & Purpose |
|---|---|---|---|---|
| `--repo` | `-r` | `.` | Directory path | Sets the root repository workspace path across all repository, search, intelligence, graph, reporting, and diagnostic commands. |
| `--config` | `-c` | `""` | File path | Explicit configuration file path override (bypasses auto-discovery). |
| `--profile` | | `""` | Profile name | Activates a named profile overlay from the configuration file (e.g. `ci`). |
| `--log-level` | | `info` | `debug`, `info`, `warn`, `error`, `critical` | Sets minimum operational log severity emitted to stderr. |
| `--log-format` | | `text` | `text`, `json` | Sets operational log serialization format. |
| `--log-file` | | `""` | File path | Redirects operational log stream to a persistent file on disk. |
| `--verbose` | `-v` | `false` | Boolean flag | Enables verbose operational logging. |
| `--debug` | | `false` | Boolean flag | Enables debug mode and trace collection. |
| `--trace` | | `false` | Boolean flag | Enables execution span tracing. |
| `--profile-cpu`| | `""` | File path | Starts CPU profiling and writes pprof profile to the specified path on exit. |
| `--profile-mem`| | `""` | File path | Writes heap allocation pprof profile to the specified path on exit. |
| `--help` | `-h` | `false` | Boolean flag | Displays contextual help for the invoked command and exits. |
| `--version` | | `false` | Boolean flag | Displays application version and exits. |
| `--interactive`| `-i` | `false` | Boolean flag | Launches the interactive REPL shell. |

### 2. Format & Output Redirection Semantics

- `--format`, `-f <format>`: Controls serialization. While parsed globally, individual command families render specific formats:
  - **Structured & Query Commands** (`repo`, `search`, `intel`, `graph`, `config`, `diag`, `health`, `debug`, `profile`, `version`): Support `text` (human-readable table/key-value), `json`, `yaml`, `toml`, `xml`, `csv`.
  - **Report Commands** (`report`): Support `text`, `markdown`, `html`, `pdf`, `json`, `yaml`, `toml`, `xml`, `csv`.
  - **Export Commands** (`export`): Support `mermaid`, `graphviz`, `svg`, `png`, `interactive`, `json`, `yaml`.
- `--json`: Shorthand for `--format json`.
- `--yaml`: Shorthand for `--format yaml`.
- `--output`, `-o <path>`: Redirects output directly to a target file on disk. Consumed specifically by `report` and `export` commands. For other commands, standard shell stdout redirection (`> output.json`) is used.

### 3. Command-Specific Options

- `--depth`, `-d <n>`: Consumed by `intel impact` (Default: `10`), `export graph` (Default: `3`), and `export diagram` (Default: `3`).
- `--limit`, `-l <n>`: Consumed by `search` commands (Default: `20`) and `log` (Default: `50`).
- `--category <cat>`: Consumed by `diag` to filter observations by category (`system`, `repository`, `configuration`, `dependency`, `performance`, `runtime`).
- `--severity <sev>`: Consumed by `diag` to filter by minimum severity (`info`, `warn`, `error`, `critical`).
- `--force`: Consumed by `config init` to overwrite an existing configuration file.
- `--level <lvl>`: Consumed by `log` to filter log records by minimum level.

---

# Command Reference

## 1. Repository Commands

### `limoxel repo init [path]` (Shortcut: `limoxel init`)
- **Purpose**: Initializes a new repository workspace context and prepares engine structures.
- **Syntax**: `limoxel repo init [path] [options]`
- **Options**: `--repo`, `-r <path>`
- **Example**: `limoxel repo init ./my-project`
- **Output**: Confirmation of initialized workspace path.

### `limoxel repo open [path]` (Shortcut: `limoxel open`)
- **Purpose**: Opens an existing repository workspace, reads VCS status, and displays workspace summary.
- **Syntax**: `limoxel repo open [path] [options]`
- **Options**: `--repo`, `-r <path>`, `--format`, `-f <format>`
- **Example**: `limoxel open --json`
- **Output**: Key-value table or JSON object with name, root path, languages, branch, and lifecycle state.

### `limoxel repo scan [path]` (Shortcut: `limoxel scan`)
- **Purpose**: Traverses the filesystem, discovers source files, and builds the file/package inventory according to exclusion patterns.
- **Syntax**: `limoxel repo scan [path] [options]`
- **Options**: `--repo`, `-r <path>`, `--format`, `-f <format>`
- **Example**: `limoxel scan -r ./backend`
- **Output**: Summary of total files, directories, and discovered packages.

### `limoxel repo analyze [path]` (Shortcut: `limoxel analyze`)
- **Purpose**: Executes AST parsers, extracts symbols, functions, types, and builds local cross-reference tables.
- **Syntax**: `limoxel repo analyze [path] [options]`
- **Options**: `--repo`, `-r <path>`, `--format`, `-f <format>`
- **Example**: `limoxel analyze`
- **Output**: Breakdown of parsed symbols, packages, and index duration.

### `limoxel repo validate [path]` (Shortcut: `limoxel validate`)
- **Purpose**: Validates repository structure, syntax, and referential consistency without modifying files.
- **Syntax**: `limoxel repo validate [path] [options]`
- **Options**: `--repo`, `-r <path>`, `--format`, `-f <format>`
- **Example**: `limoxel validate`
- **Output**: Validation report detailing syntax errors or reference inconsistencies.

### `limoxel repo reload [path]` (Shortcut: `limoxel reload`)
- **Purpose**: Evicts in-memory caches and reloads repository state from disk.
- **Syntax**: `limoxel repo reload [path] [options]`
- **Options**: `--repo`, `-r <path>`
- **Example**: `limoxel reload`
- **Output**: Confirmation of reloaded state.

### `limoxel repo close` (Shortcut: `limoxel close`)
- **Purpose**: Closes the active repository session and releases memory resources.
- **Syntax**: `limoxel repo close`
- **Example**: `limoxel close`
- **Output**: Confirmation of closed session.

### `limoxel repo info [path]` (Shortcut: `limoxel info`)
- **Purpose**: Displays detailed VCS branch, commit hash, file counts, detected languages, and capabilities.
- **Syntax**: `limoxel repo info [path] [options]`
- **Options**: `--repo`, `-r <path>`, `--format`, `-f <format>`
- **Example**: `limoxel info --format yaml`
- **Output**: Structured or formatted metadata summary.

### `limoxel repo statistics [path]` (Alias: `stats`, Shortcut: `limoxel stats`)
- **Purpose**: Displays quantitative metrics including line counts, symbol density, and language distribution.
- **Syntax**: `limoxel repo statistics [path] [options]`
- **Options**: `--repo`, `-r <path>`, `--format`, `-f <format>`
- **Example**: `limoxel stats --json`
- **Output**: Statistical metrics table or JSON data.

---

## 2. Engineering Search Commands

### `limoxel search <query>`
- **Purpose**: Performs a unified search across all domains (symbols, packages, modules, files, dependencies, docs, config).
- **Syntax**: `limoxel search <query> [options]`
- **Options**: `--limit`, `-l <n>`, `--repo`, `-r <path>`, `--format`, `-f <format>`
- **Example**: `limoxel search Engine --limit 10`
- **Output**: Unified search results grouped by domain with match locations.

### `limoxel search symbol <query>`
- **Purpose**: Searches functions, methods, types, interfaces, and constants.
- **Syntax**: `limoxel search symbol <query> [options]`
- **Options**: `--limit`, `-l <n>`, `--repo`, `-r <path>`, `--format`, `-f <format>`
- **Example**: `limoxel search symbol "New*"`
- **Output**: Symbol identifiers, types, containing files, and line numbers.

### `limoxel search package <query>` (Alias: `search pkg`)
- **Purpose**: Searches declared packages and namespaces.
- **Syntax**: `limoxel search package <query> [options]`
- **Options**: `--limit`, `-l <n>`, `--repo`, `-r <path>`, `--format`, `-f <format>`
- **Example**: `limoxel search pkg logging`
- **Output**: Package names, import paths, and containing directory paths.

### `limoxel search module <query>` (Alias: `search mod`)
- **Purpose**: Searches defined modules across the repository.
- **Syntax**: `limoxel search module <query> [options]`
- **Options**: `--limit`, `-l <n>`, `--repo`, `-r <path>`, `--format`, `-f <format>`
- **Example**: `limoxel search mod limoxel`
- **Output**: Module names, root directories, and manifest files.

### `limoxel search file <query>`
- **Purpose**: Searches files by name, relative path, or extension pattern.
- **Syntax**: `limoxel search file <query> [options]`
- **Options**: `--limit`, `-l <n>`, `--repo`, `-r <path>`, `--format`, `-f <format>`
- **Example**: `limoxel search file ".json"`
- **Output**: Relative file paths, sizes, and language classifications.

### `limoxel search dependency <query>` (Alias: `search dep`)
- **Purpose**: Searches direct and indirect external dependencies.
- **Syntax**: `limoxel search dependency <query> [options]`
- **Options**: `--limit`, `-l <n>`, `--repo`, `-r <path>`, `--format`, `-f <format>`
- **Example**: `limoxel search dep yaml`
- **Output**: Dependency module names, versions, and referencing packages.

### `limoxel search doc <query>` (Alias: `search documentation`)
- **Purpose**: Searches documentation comments, docstrings, and markdown files.
- **Syntax**: `limoxel search doc <query> [options]`
- **Options**: `--limit`, `-l <n>`, `--repo`, `-r <path>`, `--format`, `-f <format>`
- **Example**: `limoxel search doc "thread-safe"`
- **Output**: Matching documentation snippets and associated symbol references.

### `limoxel search config <query>` (Alias: `search configuration`)
- **Purpose**: Searches configuration files, settings, and keys.
- **Syntax**: `limoxel search config <query> [options]`
- **Options**: `--limit`, `-l <n>`, `--repo`, `-r <path>`, `--format`, `-f <format>`
- **Example**: `limoxel search config "workers"`
- **Output**: Configuration file paths, matching keys, and assigned values.

---

## 3. Engineering Intelligence Commands

### `limoxel intel inspect <symbol>`
- **Purpose**: Inspects detailed semantics, signatures, scope, visibility, and direct relationships for a symbol.
- **Syntax**: `limoxel intel inspect <symbol-id|name> [options]`
- **Options**: `--repo`, `-r <path>`, `--format`, `-f <format>`
- **Example**: `limoxel intel inspect "NewLogger"`
- **Output**: Symbol specification card detailing type, package, file location, parameters, and callers.

### `limoxel intel explain [component]`
- **Purpose**: Generates an architectural explanation of structural layers, boundaries, and components.
- **Syntax**: `limoxel intel explain [component] [options]`
- **Options**: `--repo`, `-r <path>`, `--format`, `-f <format>`
- **Example**: `limoxel intel explain cli`
- **Output**: Prose explanation of component architecture, responsibilities, and inbound/outbound contracts.

### `limoxel intel dependencies [package]` (Alias: `intel deps`)
- **Purpose**: Analyzes package dependencies, fan-in, fan-out, coupling metrics, and circular dependency risks.
- **Syntax**: `limoxel intel dependencies [package] [options]`
- **Options**: `--repo`, `-r <path>`, `--format`, `-f <format>`
- **Example**: `limoxel intel deps internal/capabilities/cli`
- **Output**: Coupling analysis table and circular dependency warnings.

### `limoxel intel health [path]`
- **Purpose**: Computes repository health scorecard, maintainability index, and risk metrics.
- **Syntax**: `limoxel intel health [path] [options]`
- **Options**: `--repo`, `-r <path>`, `--format`, `-f <format>`
- **Example**: `limoxel intel health --json`
- **Output**: Health scorecard with numerical scores and risk breakdown.

### `limoxel intel impact <symbol|package>`
- **Purpose**: Performs deterministic multi-hop change impact analysis for a symbol or package.
- **Syntax**: `limoxel intel impact <symbol-id|package-path> [options]`
- **Options**: `--depth`, `-d <n>`, `--repo`, `-r <path>`, `--format`, `-f <format>`
- **Example**: `limoxel intel impact ParseFlags --depth 5`
- **Output**: Affected downstream symbols, caller paths, and impacted packages.

### `limoxel intel navigate <symbol>` (Alias: `intel nav`)
- **Purpose**: Navigates symbol hierarchies, definitions, inbound callers, and references.
- **Syntax**: `limoxel intel navigate <symbol-id|name> [options]`
- **Options**: `--repo`, `-r <path>`, `--format`, `-f <format>`
- **Example**: `limoxel intel nav RunCommand`
- **Output**: Navigation tree showing definitions, caller trees, and references.

### `limoxel intel recommendations [path]` (Aliases: `intel recommend`, `intel rec`)
- **Purpose**: Derives prioritized actionable recommendations across architecture, dependencies, and testing.
- **Syntax**: `limoxel intel recommendations [path] [options]`
- **Options**: `--repo`, `-r <path>`, `--format`, `-f <format>`
- **Example**: `limoxel intel rec --format markdown`
- **Output**: Categorized recommendation list ranked by priority.

---

## 4. Knowledge Graph Commands

### `limoxel graph repo [path]` (Aliases: `kg repo`, `graph repository`)
- **Purpose**: Queries repository-level knowledge graph entity counts and connectivity metrics.
- **Syntax**: `limoxel graph repo [path] [options]`
- **Options**: `--repo`, `-r <path>`, `--format`, `-f <format>`
- **Example**: `limoxel graph repo --format json`
- **Output**: Entity counts, relationship counts, and insight metrics.

### `limoxel graph package [pkg]` (Alias: `graph pkg`)
- **Purpose**: Queries package nodes and inter-package dependency relationships.
- **Syntax**: `limoxel graph package [package-name] [options]`
- **Options**: `--repo`, `-r <path>`, `--format`, `-f <format>`
- **Example**: `limoxel graph pkg internal/platform/logging`
- **Output**: List of incoming and outgoing package dependency edges.

### `limoxel graph dependency [dep]` (Alias: `graph dep`)
- **Purpose**: Queries external and internal dependency graph edges.
- **Syntax**: `limoxel graph dependency [dependency-name] [options]`
- **Options**: `--repo`, `-r <path>`, `--format`, `-f <format>`
- **Example**: `limoxel graph dep`
- **Output**: Dependency nodes and dependency-to-package edges.

### `limoxel graph call <symbol>`
- **Purpose**: Queries direct and indirect call graph relationships (callers and callees).
- **Syntax**: `limoxel graph call <symbol-id|name> [options]`
- **Options**: `--repo`, `-r <path>`, `--format`, `-f <format>`
- **Example**: `limoxel graph call HandleRequest`
- **Output**: Direct callers and callees with edge weights.

### `limoxel graph module [module]` (Alias: `graph mod`)
- **Purpose**: Queries module-level graph entities and containment relationships.
- **Syntax**: `limoxel graph module [module-name] [options]`
- **Options**: `--repo`, `-r <path>`, `--format`, `-f <format>`
- **Example**: `limoxel graph mod limoxel`
- **Output**: Module boundaries and contained package nodes.

### `limoxel graph symbol <symbol>` (Alias: `graph sym`)
- **Purpose**: Queries symbol relationships (definitions, references, implementations).
- **Syntax**: `limoxel graph symbol <symbol-id|name> [options]`
- **Options**: `--repo`, `-r <path>`, `--format`, `-f <format>`
- **Example**: `limoxel graph sym Logger`
- **Output**: Symbol entity node and associated relationship edges.

---

## 5. Reporting & Export Commands

### `limoxel report summary [path]` (Alias: `report executive`)
- **Purpose**: Generates an executive engineering summary scorecard.
- **Syntax**: `limoxel report summary [path] [options]`
- **Options**: `--format`, `-f <format>`, `--output`, `-o <path>`, `--repo`, `-r <path>`
- **Example**: `limoxel report summary -f markdown -o summary.md`
- **Output**: Rendered scorecard in text, markdown, HTML, or structured formats.

### `limoxel report repository [path]` (Alias: `report repo`)
- **Purpose**: Generates a complete repository characteristics report.
- **Syntax**: `limoxel report repository [path] [options]`
- **Options**: `--format`, `-f <format>`, `--output`, `-o <path>`, `--repo`, `-r <path>`
- **Example**: `limoxel report repo -f html -o repo.html`
- **Output**: Comprehensive repository report.

### `limoxel report architecture [path]` (Alias: `report arch`)
- **Purpose**: Generates an architectural analysis and structural boundary report.
- **Syntax**: `limoxel report architecture [path] [options]`
- **Options**: `--format`, `-f <format>`, `--output`, `-o <path>`, `--repo`, `-r <path>`
- **Example**: `limoxel report arch -f html -o arch.html`
- **Output**: Architecture analysis report.

### `limoxel report dependency [path]` (Alias: `report deps`)
- **Purpose**: Generates a comprehensive dependency and ecosystem inventory report.
- **Syntax**: `limoxel report dependency [path] [options]`
- **Options**: `--format`, `-f <format>`, `--output`, `-o <path>`, `--repo`, `-r <path>`
- **Example**: `limoxel report deps -f csv -o deps.csv`
- **Output**: Dependency inventory report.

### `limoxel report health [path]`
- **Purpose**: Generates a repository quality, security, and risk health report.
- **Syntax**: `limoxel report health [path] [options]`
- **Options**: `--format`, `-f <format>`, `--output`, `-o <path>`, `--repo`, `-r <path>`
- **Example**: `limoxel report health -f pdf -o health.pdf`
- **Output**: Risk and quality report.

### `limoxel export graph [target]`
- **Purpose**: Exports knowledge graph entities and relationships in visual diagram formats or structured data.
- **Syntax**: `limoxel export graph [target] [options]`
- **Options**: `--format`, `-f <format>`, `--depth`, `-d <n>`, `--output`, `-o <path>`, `--repo`, `-r <path>`
- **Accepted Formats**: `mermaid`, `graphviz`, `svg`, `png`, `interactive`, `json`, `yaml`
- **Example**: `limoxel export graph -f mermaid -o graph.mmd`
- **Output**: Serialized graph diagram or data written to file or stdout.

### `limoxel export diagram <type> [target]`
- **Purpose**: Exports specialized subsystem diagrams (`dependency`, `call`, `package`, `module`, `symbol`, `architecture`).
- **Syntax**: `limoxel export diagram <type> [target] [options]`
- **Options**: `--format`, `-f <format>`, `--depth`, `-d <n>`, `--output`, `-o <path>`, `--repo`, `-r <path>`
- **Example**: `limoxel export diagram package -f mermaid -o packages.mmd`
- **Output**: Formatted diagram text or rendered image.

---

## 6. Configuration & Profile Commands

*(Note: Configuration commands automatically inherit application-wide `--config`, `--profile`, and `--repo` flags).*

### `limoxel config list` (Aliases: `config show`, `config ls`)
- **Purpose**: Lists all effective configuration keys, values, provenance source, and precedence levels. Sensitive secret values are automatically masked by default.
- **Syntax**: `limoxel config list [options]`
- **Options**: `--format`, `-f <format>`
- **Example**: `limoxel config list --format json`
- **Output**: Table or JSON array of configuration entries.

### `limoxel config get <key>`
- **Purpose**: Retrieves the effective resolved value for a specific configuration key.
- **Syntax**: `limoxel config get <key> [options]`
- **Options**: `--format`, `-f <format>`
- **Example**: `limoxel config get output.format`
- **Output**: Key value and provenance information.

### `limoxel config set <key> <value>`
- **Purpose**: Sets and persists a configuration key-value pair in the active configuration file.
- **Syntax**: `limoxel config set <key> <value> [options]`
- **Example**: `limoxel config set output.format json`
- **Output**: Confirmation of persisted key.

### `limoxel config unset <key>` (Aliases: `config rm`, `config delete`)
- **Purpose**: Removes a configuration key from the configuration file, reverting to lower-precedence default.
- **Syntax**: `limoxel config unset <key> [options]`
- **Example**: `limoxel config unset output.format`
- **Output**: Confirmation of removed key.

### `limoxel config validate [file]` (Alias: `config check`)
- **Purpose**: Validates configuration syntax, field types, bounds, and enum constraints.
- **Syntax**: `limoxel config validate [file] [options]`
- **Options**: `--format`, `-f <format>`
- **Example**: `limoxel config validate`
- **Output**: Validation result indicating success or specific line errors.

### `limoxel config init`
- **Purpose**: Creates an initial `.limoxel.yaml` configuration file populated with default values.
- **Syntax**: `limoxel config init [options]`
- **Options**: `--format`, `-f <format>`, `--force`
- **Example**: `limoxel config init --format yaml`
- **Output**: Confirmation of created configuration file path.

### `limoxel config profile <list|create|delete> [name]`
- **Purpose**: Manages named configuration profile overlays.
- **Syntax**: `limoxel config profile <list|create|delete> [name] [options]`
- **Options**: `--format`, `-f <format>`
- **Example**: `limoxel config profile create ci`
- **Output**: Profile listing or confirmation of profile creation/deletion.

---

## 7. Diagnostics & Observability Commands

### `limoxel log` (Alias: `logs`)
- **Purpose**: Inspects operational log events recorded in the in-memory circular buffer.
- **Syntax**: `limoxel log [options]`
- **Options**: `--limit`, `-l <n>`, `--level <lvl>`, `--format`, `-f <format>`
- **Example**: `limoxel log --limit 25 --level warn`
- **Output**: Log events table with timestamps, levels, components, and messages.

### `limoxel diag` (Alias: `diagnostics`)
- **Purpose**: Runs operational diagnostics across system host resources, repository accessibility, configuration, toolchains, performance, and runtime.
- **Syntax**: `limoxel diag [options]`
- **Options**: `--category <cat>`, `--severity <sev>`, `--format`, `-f <format>`
- **Example**: `limoxel diag --category dependency`
- **Output**: Diagnostic observations table with severity and actionable remediations.

### `limoxel health`
- **Purpose**: Performs operational readiness health checks across Limoxel runtime components (`system_resources`, `workspace_repository`, `go_runtime`, `cache_subsystem`, `index_database`).
- **Syntax**: `limoxel health [options]`
- **Options**: `--format`, `-f <format>`
- **Example**: `limoxel health --format json`
- **Output**: Overall health status (`HEALTHY`, `DEGRADED`, `UNAVAILABLE`, `FAILED`) and probe latencies.

### `limoxel debug [trace|dump]`
- **Purpose**: Generates an operational state dump or formats recorded execution trace spans.
- **Syntax**: `limoxel debug [trace|dump] [options]`
- **Options**: `--format`, `-f <format>`
- **Example**: `limoxel debug trace`
- **Output**: Formatted trace tree or process state snapshot.

### `limoxel profile [heap|stats] [target]`
- **Purpose**: Inspects runtime memory statistics or writes a heap allocation pprof profile.
- **Syntax**: `limoxel profile [heap|stats] [target] [options]`
- **Options**: `--format`, `-f <format>`
- **Example**: `limoxel profile stats`
- **Output**: Runtime memory metrics table (Allocated heap, Sys memory, Goroutine count, GC pauses).

---

## 8. General & System Commands

### `limoxel help [command...]`
- **Purpose**: Displays contextual help and usage information for any command or subcommand.
- **Syntax**: `limoxel help [command...]`
- **Example**: `limoxel help intel impact`
- **Output**: Usage description, argument requirements, options, and aliases.

### `limoxel version`
- **Purpose**: Displays executable name, version, and build information.
- **Syntax**: `limoxel version [options]`
- **Options**: `--format`, `-f <format>`, `--json`
- **Example**: `limoxel version --json`
- **Output**: Version string or JSON object.

### `limoxel completion <shell>`
- **Purpose**: Generates shell autocompletion scripts for Bash, Zsh, Fish, or PowerShell.
- **Syntax**: `limoxel completion <bash|zsh|fish|powershell>`
- **Example**: `source <(limoxel completion bash)`
- **Output**: Shell completion script.

### `limoxel interactive` (Aliases: `repl`, `shell`)
- **Purpose**: Starts an interactive REPL terminal session.
- **Syntax**: `limoxel interactive`
- **Example**: `limoxel interactive`
- **Output**: Interactive command prompt.

---

# Configuration Usage

### Precedence Hierarchy (Highest to Lowest)

1. **Runtime Overrides** (`PrecedenceRuntime`: `50`): Programmatic overrides or command-line runtime flags.
2. **Environment Variables** (`PrecedenceEnv`: `40`): Variables prefixed with `LIMOXEL_*`.
3. **Named Configuration Profiles** (`PrecedenceProfile`: `30`): Active profile overlay selected via `--profile` or `active_profile`.
4. **Configuration Files** (`PrecedenceFile`: `20`): Workspace configuration overlaying user-home configuration.
5. **Built-in Canonical Defaults** (`PrecedenceDefault`: `10`): Factory baseline defaults.

### Configuration File Discovery Order

Limoxel searches for configuration files in the following deterministic sequence:

1. **Explicit File**: Specified via `--config <path>` (bypasses auto-discovery).
2. **User Home Directory**:
   - `~/.limoxel/config.yaml`
   - `~/.limoxel/config.json`
   - `~/.limoxel/config.toml`
   - `~/.limoxelrc`
3. **Workspace Directory**:
   - `<workspace>/.limoxel.yaml`
   - `<workspace>/.limoxel.yml`
   - `<workspace>/.limoxel.json`
   - `<workspace>/.limoxel.toml`
   - `<workspace>/.limoxel/config.yaml`
   - `<workspace>/.limoxel/config.json`
   - `<workspace>/.limoxel/config.toml`
   - `<workspace>/limoxel.config.yaml`

Workspace configuration files override user-home configuration files.

### Authoritative `.limoxel.yaml` Example

```yaml
version: "1.0.0"
active_profile: "default"

repository:
  root: "."
  max_file_size_mb: 10
  indexing_mode: "standard"
  exclude_patterns:
    - ".git"
    - "vendor"
    - "node_modules"
    - "bin"
    - "dist"

analysis:
  strict_mode: false
  max_depth: 15
  rule_severity_threshold: "info"

output:
  format: "text"
  color: true
  theme: "dark"
  file_overwrite: false

logging:
  level: "info"
  format: "text"
  file: ""

performance:
  workers: 4
  timeout_seconds: 60

profiles:
  ci:
    name: "ci"
    description: "Continuous integration profile with strict analysis and JSON output"
    values:
      output.format: "json"
      output.color: false
      analysis.strict_mode: true
      logging.level: "warn"
```

### Environment Variables

Configuration properties can be set via environment variables using the `LIMOXEL_` prefix and dot-to-underscore mapping:

```bash
export LIMOXEL_OUTPUT_FORMAT="json"
export LIMOXEL_LOGGING_LEVEL="debug"
export LIMOXEL_ANALYSIS_STRICT_MODE="true"
export LIMOXEL_PERFORMANCE_WORKERS="8"
```

---

# Scripting & CI Automation

### GitHub Actions Workflow Example

```yaml
name: Limoxel Architecture & Quality Gate

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main, develop]

jobs:
  limoxel-audit:
    name: Architecture & Health Audit
    runs-on: ubuntu-latest

    steps:
      - name: Checkout Code
        uses: actions/checkout@v4

      - name: Set Up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.26.5"
          cache: true

      - name: Build Limoxel CLI
        run: |
          go build -o /usr/local/bin/limoxel ./cmd/limoxel

      - name: Operational Health Check
        run: |
          limoxel health --format json

      - name: Validate Repository Structure
        run: |
          limoxel validate

      - name: Run Architecture Health Audit
        run: |
          limoxel intel health --format json > health_audit.json

      - name: Export Dependency Diagram
        run: |
          limoxel export diagram package -f mermaid -o package_deps.mmd
```

### Processing Output with `jq`

```bash
# Assert zero critical diagnostic findings
CRITICAL_COUNT=$(limoxel diag --format json | jq '[.[] | select(.severity == "critical")] | length')
if [ "$CRITICAL_COUNT" -gt 0 ]; then
  echo "Audit failed: $CRITICAL_COUNT critical diagnostic findings detected."
  exit 1
fi
```

---

# Troubleshooting Guide

| Symptom | Probable Cause | Diagnostic Command | Remediation Action |
|---|---|---|---|
| `unknown command "xyz"` | Typo or unrecognized command | `limoxel help` | Verify command name and valid aliases using `limoxel help`. |
| `repository path does not exist` | Invalid `--repo` path provided | `limoxel diag --category repository` | Verify directory path exists and specify valid root directory. |
| `permission denied on workspace` | Insufficient read permissions | `limoxel diag --category repository` | Grant read permissions (POSIX: `chmod -R u+rX <path>`, Windows: check ACLs via `icacls`). |
| `Git executable not found on PATH` | Git not installed or not in PATH | `limoxel diag --category dependency` | Install Git or ensure Git binary directory is added to system PATH. |
| `high memory consumption` | Repository exceeds memory limits | `limoxel profile stats` | Increase exclusion patterns in `.limoxel.yaml` or reduce `performance.workers`. |
| `health check reported failed` | Runtime component unavailable | `limoxel health --format json` | Inspect failing check result in health report output. |
| `invalid configuration: out of bounds` | Configuration value violates schema | `limoxel config validate` | Correct invalid key in `.limoxel.yaml` according to reported line error. |

---

# Error Codes & Exit Statuses

Limoxel adheres to standard POSIX exit status codes for predictable scripting:

| Exit Code | Identifier | Meaning | Conditions |
|---|---|---|---|
| `0` | `ExitSuccess` | Successful execution | Normal completion, successful queries, valid syntax. |
| `1` | `ExitFailure` | Execution / capability failure | Runtime error, inaccessible file, failed health probe assertion. |
| `2` | `ExitUsage` | Command syntax or flag error | Unrecognized flag, missing required argument, invalid option value. |

---

# Frequently Asked Questions (FAQ)

**Q: Does Limoxel modify my source code?**  
A: No. Repository analysis, inspection, search, intelligence, and reporting commands are strictly non-destructive and read-only. File modifications only occur when explicitly executing configuration mutation commands (`config init`, `config set`, `config unset`, `config profile create`, `config profile delete`) or when writing reports/diagrams via `--output` / `-o`.

**Q: What programming languages and asset types are currently supported?**  
A: The current Limoxel CLI pipeline provides full AST-level symbol extraction, call-graph analysis, and cross-referencing for **Go**, alongside multi-format file discovery, structure mapping, configuration analysis, and documentation extraction for **Go**, **Markdown**, **JSON**, and **YAML** assets.

**Q: Can I use Limoxel without creating a `.limoxel.yaml` file?**  
A: Yes. Limoxel operates out-of-the-box using canonical built-in defaults. A configuration file is only needed to customize exclude patterns, profiles, themes, or analysis thresholds.

**Q: How do I export diagrams for pull requests and documentation?**  
A: Use Mermaid format (`limoxel export graph -f mermaid -o diagram.mmd`), which renders natively in GitHub, GitLab, Notion, and Markdown viewers.

**Q: Where are operational logs written?**  
A: By default, operational logs are written to `stderr` to keep `stdout` clean for piped JSON/YAML data. Use `--log-file /path/to/log.txt` to redirect logs to a persistent file.

---

# Documentation Cross-References

For architectural specifications, formal invariants, and capability contracts:

- **Command Line Interface Specification**: [01_Command_Line_Interface.md](01_Command_Line_Interface.md)
- **Output & Reporting Engine Specification**: [02_Output_and_Reporting.md](02_Output_and_Reporting.md)
- **Configuration System Specification**: [03_Configuration_System.md](03_Configuration_System.md)
- **Logging & Diagnostics Specification**: [04_Logging_and_Diagnostics.md](04_Logging_and_Diagnostics.md)

---

# Authority

This document is the canonical practical user and developer guide for the Limoxel Command Line Interface.

It serves as the authoritative operational manual for command syntax, option flags, configuration workflows, and troubleshooting procedures.

The implementation code and the individual capability specification documents ([01_Command_Line_Interface.md](01_Command_Line_Interface.md), [02_Output_and_Reporting.md](02_Output_and_Reporting.md), [03_Configuration_System.md](03_Configuration_System.md), [04_Logging_and_Diagnostics.md](04_Logging_and_Diagnostics.md)) remain authoritative for internal subsystem semantics, mathematical models, and architectural contracts.

---

# Applicability

This document applies to all user-facing operations of the Limoxel CLI, including:

- Command invocation and argument syntax
- Global and command-local option flags
- Presentation, document, and visual export formats
- Configuration files, profiles, and environment overrides
- Operational logging, diagnostic collection, profiling, and health checks
- Interactive REPL execution
- Continuous integration and automated shell scripting

It does not redefine internal capability algorithms or non-public package APIs.

---

# Change Policy

Changes to this document must preserve:

- 100% fidelity with verified, implemented CLI commands and options
- Consistency with the authoritative capability specifications ([01_Command_Line_Interface.md](01_Command_Line_Interface.md), [02_Output_and_Reporting.md](02_Output_and_Reporting.md), [03_Configuration_System.md](03_Configuration_System.md), [04_Logging_and_Diagnostics.md](04_Logging_and_Diagnostics.md))
- Practical usability, clear examples, and deterministic workflows
- Synchronized version and argument descriptions across all documented command families

This document remains the authoritative practical operational guide for the Limoxel CLI until an approved revision supersedes it.
