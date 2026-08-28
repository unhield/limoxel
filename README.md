<div align="center">

# Limoxel

### Engineering Knowledge Infrastructure

**A production-grade platform for understanding, analyzing, navigating, and reporting on software repositories.**

<br>

![Release](https://img.shields.io/github/v/release/unhield/limoxel?style=for-the-badge)
![Go Version](https://img.shields.io/github/go-mod/go-version/unhield/limoxel?style=for-the-badge)
![License](https://img.shields.io/github/license/unhield/limoxel.svg?style=for-the-badge)
![CI](https://img.shields.io/github/actions/workflow/status/unhield/limoxel/ci.yml?branch=main&style=for-the-badge&label=CI)
![Documentation](https://img.shields.io/badge/Documentation-Complete-success?style=for-the-badge)

<br>

**Engineering First. Intelligence Through Foundation.**

</div>

---

> [!IMPORTANT]
>
> **Limoxel is an Engineering Knowledge Infrastructure (EKI).**
>
> Limoxel helps developers, architects, and engineering teams understand software repositories through deterministic repository analysis, structured repository knowledge, engineering intelligence, and a practical command-line interface.

---

<details>
<summary><strong>Table of Contents</strong></summary>

- [Overview](#overview)
- [What Limoxel Does](#what-limoxel-does)
- [Who It Is For](#who-it-is-for)
- [Architecture at a Glance](#architecture-at-a-glance)
- [Repository Analysis](#repository-analysis)
- [Engineering Intelligence](#engineering-intelligence)
- [SDK & Public API](#sdk--public-api)
- [Command-Line Interface](#command-line-interface)
- [Output and Reporting](#output-and-reporting)
- [Configuration](#configuration)
- [Logging and Diagnostics](#logging-and-diagnostics)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Common Workflows](#common-workflows)
- [Validation](#validation)
- [Project Documentation](#project-documentation)
- [Repository Organization](#repository-organization)
- [Current Release](#current-release)
- [Contributing](#contributing)
- [License](#license)
- [Acknowledgements](#acknowledgements)

</details>

---

# Overview

Software repositories are engineering systems made up of source code, files, packages, modules, dependencies, configuration, documentation, symbols, and relationships.

Understanding a large repository requires more than searching files. Developers often need to answer questions such as:

- What is inside this repository?
- Which packages and modules depend on each other?
- Where is a symbol defined?
- Where is a symbol used?
- Which components depend on a changed package?
- How are modules and packages connected?
- What are the important architectural relationships?
- Where are potential dependency, quality, or structural problems?
- What could be affected by a proposed change?
- How can the resulting information be exported or shared?

Limoxel provides a unified platform for answering these questions from structured repository information.

It combines repository discovery, indexing, dependency analysis, source and symbol analysis, cross-reference analysis, search, knowledge graphs, engineering analysis, deterministic reasoning, reporting, and a developer-oriented CLI.

---

# What Limoxel Does

Limoxel works from the repository itself and builds progressively richer information about the software system.

At a high level:

```text
Repository
    │
    ▼
Discovery
    │
    ▼
Structure & Metadata
    │
    ▼
Indexing & Source Analysis
    │
    ▼
Dependencies & Cross-References
    │
    ▼
Repository Knowledge
    │
    ├── Search & Navigation
    ├── Knowledge Graphs
    ├── Engineering Analysis
    ├── Change Impact
    ├── Recommendations
    └── Reports & Exports
```

The result is a structured view of the repository that can be queried, explored, analyzed, and exported.

---

# Who It Is For

Limoxel is designed primarily for people who need to understand software systems rather than merely edit individual files.

### Developers

Use Limoxel to:

- Explore unfamiliar repositories
- Find symbols, packages, modules, and files
- Trace references and dependencies
- Navigate callers and callees
- Inspect repository structure
- Investigate configuration and documentation
- Understand potential change impact
- Diagnose repository or runtime issues

### Software Architects

Use Limoxel to:

- Examine package and module relationships
- Explore dependency graphs
- Inspect architectural boundaries
- Analyze coupling and structure
- Generate architecture-oriented reports
- Investigate repository-wide relationships

### Engineering Leads

Use Limoxel to:

- Produce repository health information
- Review structural and dependency risks
- Generate engineering reports
- Examine change impact
- Support technical decision-making with repository evidence

### DevOps and Platform Engineers

Use Limoxel to:

- Run repository analysis in automated workflows
- Produce machine-readable output
- Validate repository characteristics
- Integrate analysis into development pipelines
- Collect operational diagnostics

---

# Architecture at a Glance

Limoxel is organized around clear responsibilities.

```text
                           User
                             │
                             ▼
                    Command-Line Interface
                             │
                             ▼
                    Engine Coordination
                             │
                             ▼
                  Platform Infrastructure
                             │
            ┌────────────────┼────────────────┐
            ▼                ▼                ▼
       Workspace         Repository        Language
       & Project        & Filesystem       & Parser
            │                │                │
            └────────────────┼────────────────┘
                             ▼
                  Repository Capabilities
                             │
       ┌────────────┬────────┼────────┬────────────┐
       ▼            ▼        ▼        ▼            ▼
   Discovery     Structure Indexing Analysis     Search
       │            │        │        │            │
       └────────────┴────────┼────────┴────────────┘
                             ▼
                   Repository Knowledge
                             │
                             ▼
                  Engineering Intelligence
                             │
       ┌────────────┬───────┼────────┬────────────┐
       ▼            ▼       ▼        ▼            ▼
    Semantic    Navigation Analysis Reasoning   Insights
```

The command-line layer provides the user-facing interface, while the underlying platform and repository capabilities provide the data and services used by higher-level analysis.

---

# Repository Analysis

Limoxel provides structured repository-analysis capabilities for discovering and understanding software systems.

### Repository Discovery

- Repository boundary detection
- File discovery
- Exclusion handling
- Language identification
- Repository metadata collection
- Deterministic file enumeration

### Repository Structure

- Directory structure
- Packages
- Modules
- Configuration
- Documentation
- Repository organization
- Repository statistics

### Source Analysis

- Source indexing
- AST analysis
- Symbol extraction
- Symbol relationships
- Type and interface information
- Cross-reference analysis

### Dependency Analysis

- Internal dependencies
- External dependencies
- Dependency relationships
- Dependency graphs
- Circular dependency detection
- Dependency-chain traversal

### Repository Search

Search across supported repository information including:

- Symbols
- Types
- Functions
- Packages
- Modules
- Files
- Dependencies
- Documentation
- Configuration

---

# Engineering Intelligence

Limoxel extends repository analysis with deterministic engineering intelligence.

The intelligence capabilities operate on structured repository information to provide deeper understanding without requiring probabilistic AI to interpret the repository.

## Semantic Understanding

Limoxel can represent and analyze:

- Repositories
- Packages
- Modules
- Symbols
- Types
- Functions
- Interfaces
- Variables
- Scope
- Ownership
- Visibility
- Semantic relationships

## Repository Relationships

Limoxel can analyze relationships across:

- Files
- Packages
- Modules
- Workspaces
- Dependencies
- Configuration
- APIs
- Repository history and evolution

## Engineering Navigation

Navigation capabilities include:

- Definitions
- Declarations
- Implementations
- References
- Usages
- Reverse dependencies
- Symbol hierarchies
- Type hierarchies
- Interface hierarchies
- Package hierarchies
- Call hierarchies
- Dependency chains

## Engineering Analysis

Limoxel can analyze repository characteristics including:

- Code quality
- Dead code
- Unused imports and exports
- Duplicate logic
- Large files
- Large functions
- Dependencies
- Circular dependencies
- Layer relationships
- Coupling
- Architecture
- Module boundaries
- Package cohesion
- Configuration
- Repository health

## Change Impact and Recommendations

Limoxel can reason about potential engineering consequences of changes, including:

- Symbol impact
- Package impact
- Module impact
- Repository impact
- Dependency impact
- API changes
- Breaking changes
- Refactoring opportunities
- Compatibility considerations
- Dependency recommendations
- Architecture recommendations
- Performance recommendations
- Repository organization recommendations

Results are based on available repository evidence and are designed to remain deterministic and explainable.

---

# SDK & Public API

Limoxel provides a public Go SDK for applications and developer tooling that need to consume repository analysis and engineering intelligence programmatically.

The canonical public package is:

```go
import "github.com/unhield/limoxel/sdk"
```

The SDK exposes public contracts for repository and intelligence capabilities without requiring consumers to depend directly on Limoxel's internal implementation packages.

## Core SDK

The Core SDK provides public access to:

- Repository management and lifecycle
- Repository metadata and statistics
- File discovery, metadata, indexing status, and relationships
- Package discovery, hierarchy, statistics, and relationships
- Symbol lookup, references, hierarchy, documentation, and ownership
- Multi-domain repository search

## Intelligence SDK

The Intelligence SDK provides public contracts for:

- Knowledge graph access and traversal
- Graph filtering and export
- Architecture, dependency, quality, and health analysis
- Definition, reference, symbol, and call navigation
- Deterministic impact analysis and engineering reasoning
- Engineering recommendations and insights
- Repository and SDK lifecycle events
- Unified request-based intelligence access

## Compatibility

The public SDK includes compatibility utilities for:

- Semantic Versioning
- API-change evaluation
- Upgrade validation
- Migration guidance
- API deprecation tracking

## Examples and Templates

Runnable SDK examples are available under `sdk/examples/`, covering:

- Basic SDK usage
- Repository analysis
- Knowledge graphs
- Code navigation
- Intelligence and reasoning
- Event streaming

Starter templates for common integration styles are available under `sdk/templates/`.

## SDK Documentation

The SDK documentation is available under:

```text
docs/07_sdk/
├── 01_SDK_Foundation.md
├── 02_Core_SDK.md
├── 03_Intelligence_SDK.md
└── 04_SDK_and_Public_API_User_Manual.md
```

For integration work, start with:

`docs/07_sdk/04_SDK_and_Public_API_User_Manual.md`

A minimal client can be initialized with:

```go
package main

import (
	"context"
	"log"

	"github.com/unhield/limoxel/sdk"
)

func main() {
	ctx := context.Background()

	client, err := sdk.OpenWorkspace(ctx, ".")
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	stats, err := client.Repository().Statistics(ctx)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("files=%d packages=%d symbols=%d",
		stats.TotalFiles,
		stats.TotalPackages,
		stats.TotalSymbols,
	)
}
```

The SDK requires local filesystem access to the workspace being analyzed.

---

# Command-Line Interface

Limoxel provides a native command-line interface for interactive repository exploration, automation, analysis, reporting, and diagnostics.

The general form is:

```text
limoxel [global-options] <command> <subcommand> [arguments] [options]
```

The CLI is organized around practical workflows rather than requiring users to interact directly with internal packages.

## Repository Operations

Use repository commands to establish and inspect the working repository context.

Examples:

```bash
limoxel init
limoxel open /path/to/project
limoxel scan
limoxel analyze
limoxel validate
limoxel reload
limoxel info
limoxel statistics
limoxel close
```

## Search

Search repository information from the command line:

```bash
limoxel search symbol HandleRequest
limoxel search package api
limoxel search module internal
limoxel search file "*.go"
limoxel search dependency
limoxel search doc authentication
limoxel search config
```

## Engineering Analysis

Inspect repository structure and relationships:

```bash
limoxel intel inspect HandleRequest
limoxel intel explain
limoxel intel dependencies
limoxel intel health
limoxel intel impact HandleRequest
limoxel intel navigate HandleRequest
limoxel intel recommendations
```

## Knowledge Graphs

Explore repository relationships as graphs:

```bash
limoxel graph repo
limoxel graph package
limoxel graph dependency
limoxel graph call HandleRequest
limoxel graph module
limoxel graph symbol HandleRequest
```

## Reports

Generate higher-level repository reports:

```bash
limoxel report summary
limoxel report repository
limoxel report architecture
limoxel report dependency
limoxel report health
```

## Graph and Diagram Export

Export repository relationships for use outside the terminal:

```bash
limoxel export graph
limoxel export diagram architecture
limoxel export diagram call
limoxel export diagram package
```

Supported export formats include Mermaid, Graphviz/DOT, SVG, PNG, JSON, YAML, and interactive output where supported by the selected export operation.

## Configuration

Manage Limoxel configuration from the CLI:

```bash
limoxel config list
limoxel config get repository.root
limoxel config set output.format json
limoxel config unset output.format
limoxel config validate
limoxel config init
limoxel config profile list
```

Configuration can be supplied through the supported configuration file and command-line options.

## Diagnostics and Health

Inspect operational information when troubleshooting:

```bash
limoxel log
limoxel diag
limoxel health
limoxel debug trace
limoxel debug dump
limoxel profile stats
```

## Interactive Shell

Limoxel also provides a stateful interactive shell:

```bash
limoxel interactive
```

Example:

```text
limoxel> open /path/to/project
limoxel> scan
limoxel> analyze
limoxel> search symbol HandleRequest
limoxel> intel explain
limoxel> report summary
limoxel> exit
```

For the complete command reference, option semantics, configuration behavior, logging, diagnostics, workflows, troubleshooting, and developer guidance, see:

`docs/06_cli/05_CLI_User_and_Developer_Guide.md`

The detailed CLI specification documents are also available under:

```text
docs/06_cli/
├── 01_Command_Line_Interface.md
├── 02_Output_and_Reporting.md
├── 03_Configuration_System.md
├── 04_Logging_and_Diagnostics.md
└── 05_CLI_User_and_Developer_Guide.md
```

---

# Output and Reporting

Limoxel supports both human-readable and machine-readable output.

Depending on the command, output can be represented as:

- Text
- JSON
- YAML
- TOML
- XML
- CSV
- Markdown
- HTML
- PDF
- Mermaid
- Graphviz/DOT
- SVG
- PNG
- Interactive output

Global output controls include:

```bash
limoxel --format json <command>
limoxel --format yaml <command>
limoxel --json <command>
limoxel --yaml <command>
```

For commands that support direct file output:

```bash
limoxel report summary --format markdown --output report.md
limoxel export graph --format mermaid --output graph.mmd
```

For commands that write to standard output, normal shell redirection can also be used:

```bash
limoxel search symbol HandleRequest --format json > result.json
```

---

# Configuration

Limoxel supports configuration for repository analysis, output behavior, logging, diagnostics, and performance.

Configuration can cover areas such as:

- Repository root
- Repository indexing behavior
- Excluded paths
- Analysis depth
- Analysis strictness
- Output format
- Output behavior
- Logging level
- Logging format
- Log file destination
- Worker count
- Operation timeouts
- Named profiles

A configuration file can be initialized with:

```bash
limoxel config init
```

The detailed configuration behavior is documented in:

`docs/06_cli/03_Configuration_System.md`

---

# Logging and Diagnostics

Limoxel includes operational logging and diagnostic facilities intended for both local development and automated environments.

Supported functionality includes:

- Structured log records
- Multiple log levels
- Text and JSON log output
- Log-file output
- Diagnostic collection
- Runtime health checks
- Debug state information
- Execution tracing
- CPU profiling
- Memory profiling
- Resource statistics

Common diagnostic commands include:

```bash
limoxel log
limoxel diag
limoxel health
limoxel debug trace
limoxel debug dump
limoxel profile stats
```

Detailed behavior is documented in:

`docs/06_cli/04_Logging_and_Diagnostics.md`

---

# Installation

Limoxel is implemented in Go and can be built directly from source.

## Prerequisites

| Requirement | Version |
|-------------|---------|
| Go | 1.26.5 or later |
| Git | Current stable release recommended |
| Operating System | Linux, macOS, or Windows |

The official release builds target supported desktop/server platforms and architectures provided by the project release process.

## Clone the Repository

```bash
git clone https://github.com/unhield/limoxel.git
cd limoxel
```

## Build from Source

```bash
go build -o bin/limoxel ./cmd/limoxel
```

On Windows:

```powershell
go build -o bin\limoxel.exe .\cmd\limoxel
```

## Verify the Build

```bash
limoxel version
```

You can also run the executable directly from the repository:

```bash
go run ./cmd/limoxel version
```

---

# Quick Start

Once Limoxel is available on your PATH, point it at a repository and begin with discovery.

## 1. Inspect a Repository

```bash
limoxel info --repo /path/to/project
```

## 2. Scan and Analyze

```bash
limoxel scan --repo /path/to/project
limoxel analyze --repo /path/to/project
```

## 3. Search the Repository

```bash
limoxel search symbol MyFunction --repo /path/to/project
```

## 4. Inspect Relationships

```bash
limoxel intel dependencies --repo /path/to/project
limoxel graph package --repo /path/to/project
```

## 5. Generate a Report

```bash
limoxel report summary --repo /path/to/project
```

## 6. Produce Machine-Readable Output

```bash
limoxel search symbol MyFunction \
  --repo /path/to/project \
  --format json
```

---

# Common Workflows

## Explore an Unfamiliar Repository

```bash
limoxel open /path/to/project
limoxel scan
limoxel analyze
limoxel info
limoxel statistics
```

Then search and navigate:

```bash
limoxel search symbol MyFunction
limoxel intel navigate MyFunction
limoxel intel dependencies
```

## Investigate Architecture

```bash
limoxel intel explain
limoxel graph package
limoxel graph module
limoxel report architecture
```

## Investigate Dependencies

```bash
limoxel intel dependencies
limoxel graph dependency
limoxel report dependency
```

## Investigate a Potential Change

```bash
limoxel intel impact MyFunction
limoxel intel navigate MyFunction
limoxel intel recommendations
```

## Generate Reports for Automation

```bash
limoxel report summary --format json --output summary.json
limoxel report health --format json --output health.json
```

## Diagnose Operational Problems

```bash
limoxel health
limoxel diag
limoxel log
limoxel debug dump
```

For automated environments, prefer structured output formats such as JSON or YAML where machine processing is required.

---

# Validation

Limoxel is validated through multiple layers of automated verification.

Validation includes:

- Unit testing
- Integration testing
- Build verification
- Static analysis
- Dependency verification
- Runtime behavior
- Deterministic execution
- Repository integration
- Cross-package integration
- Cross-module integration
- API integration
- Graph traversal
- Search behavior
- Input validation
- Error handling
- Documentation integrity
- Performance testing

The standard Go validation commands are:

```bash
go test ./...
go vet ./...
go build ./...
go mod verify
```

Formatting can be checked with:

```bash
gofmt -l .
```

Repository workflows also provide automated CI, dependency review, documentation checks, security analysis, and release automation where applicable.

---

# Project Documentation

Limoxel documentation is organized by subject so users and developers can find the appropriate level of detail without having to read the entire project.

| Documentation | Purpose |
|---------------|---------|
| **Foundation** | Project principles, goals, engineering philosophy, repository organization, and contribution guidance |
| **Architecture** | Component boundaries, dependency rules, interfaces, communication, extensions, and system structure |
| **Engineering** | Runtime, package structure, contracts, bootstrap, and implementation guidance |
| **Repository** | Discovery, metadata, language detection, dependencies, indexing, symbols, cross-references, graphs, queries, and search |
| **Intelligence** | Semantic analysis, cross-repository analysis, navigation, engineering analysis, knowledge graphs, reasoning, and impact analysis |
| **CLI** | Command usage, output, configuration, logging, diagnostics, workflows, troubleshooting, and developer guidance |
| **SDK** | Public SDK foundation, core SDK, intelligence SDK, compatibility, examples, templates, and public API usage |

Documentation is available under:

```text
docs/
├── 01_foundation/
├── 02_architecture/
├── 03_engineering/
├── 04_repository/
├── 05_intelligence/
├── 06_cli/
└── 07_sdk/
```

For users working primarily with the command line, start with:

`docs/06_cli/05_CLI_User_and_Developer_Guide.md`

For SDK and Public API integration, start with:

`docs/07_sdk/04_SDK_and_Public_API_User_Manual.md`

For the underlying CLI behavior and configuration details:

```text
docs/06_cli/
├── 01_Command_Line_Interface.md
├── 02_Output_and_Reporting.md
├── 03_Configuration_System.md
└── 04_Logging_and_Diagnostics.md
```

---

# Repository Organization

The repository follows a responsibility-oriented structure.

```text
limoxel/
│
├── .github/                    # Repository automation and workflows
├── cmd/                        # Executable entry points
├── docs/                       # Project documentation
├── internal/
│   ├── capabilities/           # Repository and higher-level capabilities
│   ├── cli/                    # CLI foundation
│   ├── engine/                 # Execution engine
│   ├── extension/              # Extension infrastructure
│   ├── filesystem/             # Filesystem services
│   ├── language/               # Language management
│   ├── parser/                 # Parser infrastructure
│   ├── platform/               # Shared platform services
│   ├── project/                # Project management
│   ├── repository/              # Repository foundation
│   └── workspace/               # Workspace management
│
├── sdk/                        # Public Go SDK, examples, templates, and supporting tooling
│
├── tests/                      # Integration testing
│
├── CHANGELOG.md                # Release history
├── CODEOWNERS                  # Repository ownership
├── CODE_OF_CONDUCT.md          # Community guidelines
├── CONTRIBUTING.md             # Contribution guide
├── LICENSE                     # MIT License
├── README.md                   # Project overview
├── SECURITY.md                 # Security policy
├── go.mod                      # Go module definition
└── go.sum                      # Dependency checksums
```

---

# Current Release

Limoxel 1.4.0 provides a production-oriented platform combining repository analysis, engineering intelligence, reporting, a comprehensive developer CLI, and a public Go SDK.

| Capability | Status |
|------------|:------:|
| Engineering Foundation | ✅ |
| Platform Infrastructure | ✅ |
| Repository Discovery | ✅ |
| Repository Structure Analysis | ✅ |
| Source Indexing | ✅ |
| AST & Symbol Analysis | ✅ |
| Dependency Analysis | ✅ |
| Cross-Reference Analysis | ✅ |
| Repository Knowledge Graph | ✅ |
| Repository Search | ✅ |
| Semantic Analysis | ✅ |
| Cross-Repository Analysis | ✅ |
| Engineering Navigation | ✅ |
| Engineering Analysis | ✅ |
| Change-Impact Analysis | ✅ |
| Engineering Recommendations | ✅ |
| Public SDK & API Contracts | ✅ |
| SDK Compatibility & Versioning | ✅ |
| SDK Examples & Templates | ✅ |
| Command-Line Interface | ✅ |
| Configuration | ✅ |
| Reporting & Export | ✅ |
| Logging & Diagnostics | ✅ |
| Interactive Shell | ✅ |
| Engineering Documentation | ✅ |
| Automated Validation | ✅ |

For release history and version-specific changes, see `CHANGELOG.md`.

---

# Contributing

Contributions are welcome and appreciated.

Limoxel follows a maintainer-approval model to preserve engineering quality, architectural consistency, repository organization, and long-term maintainability.

Every contribution is reviewed according to the project's engineering principles, implementation standards, documentation quality, and long-term direction.

Submitting an Issue, Pull Request, or proposed enhancement does not guarantee acceptance.

Before contributing, please read:

- `CONTRIBUTING.md`
- `CODE_OF_CONDUCT.md`

For substantial features, architectural changes, or repository-wide improvements, please review the relevant documentation before beginning implementation.

If you are unsure whether a proposal aligns with the direction of Limoxel, contact:

**hello.limoxel@gmail.com**

before investing significant implementation effort.

---

## ❤️ Support Limoxel

Limoxel is developed and maintained as an independent open-source project.

If Limoxel saves you time, supports your research, or becomes part of your engineering workflow, consider supporting its continued development.

[![Sponsor](https://img.shields.io/badge/Sponsor-Buy%20Me%20A%20Coffee-FFDD00?logo=buymeacoffee&logoColor=000000)](https://buymeacoffee.com/limoxel)

---

# License

Limoxel is released under the **MIT License**.

See the `LICENSE` file for the complete license text.

---

# Acknowledgements

Limoxel is a long-term engineering initiative focused on building a stable and extensible foundation for understanding software repositories.

The project emphasizes:

- Engineering quality
- Clear architecture
- Deterministic analysis
- Structured repository knowledge
- Practical developer tooling
- Documentation-first development
- Maintainability
- Explainable engineering analysis

<div align="center">

---

### Engineering First. Intelligence Through Foundation.

*Building structured engineering intelligence on a permanent engineering foundation.*

</div>