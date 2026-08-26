# Changelog

All notable changes to Limoxel are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and Limoxel follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [1.3.0] - 2026-08-26

### Added

#### Command-Line Interface

- Added a comprehensive command-line interface for interacting with Limoxel repository capabilities.
- Added repository lifecycle commands for initializing, opening, scanning, analyzing, validating, reloading, inspecting, reporting statistics, and closing repository contexts.
- Added repository search commands covering symbols, files, packages, modules, dependencies, documentation, configuration, and related repository information.
- Added intelligence commands for inspection, explanation, dependencies, navigation, health, impact analysis, and engineering recommendations.
- Added knowledge graph commands for repository, package, module, dependency, call, and symbol relationships.
- Added reporting commands for repository summaries, repository analysis, architecture, dependencies, and health.
- Added export commands for repository graphs and diagrams.
- Added configuration commands for initialization, listing, reading, setting, unsetting, validation, and profile management.
- Added logging and diagnostic commands for operational inspection and troubleshooting.
- Added health and runtime diagnostic commands.
- Added debugging commands for execution tracing and diagnostic state inspection.
- Added profiling commands for runtime statistics.
- Added an interactive shell for stateful command-line repository workflows.

#### Output and Export

- Added structured output support for supported CLI operations.
- Added JSON output.
- Added YAML output.
- Added TOML output.
- Added XML output.
- Added CSV output.
- Added Markdown output.
- Added HTML output.
- Added PDF output.
- Added Mermaid graph and diagram output.
- Added Graphviz/DOT graph and diagram output.
- Added SVG output.
- Added PNG output.
- Added interactive output where supported by the selected operation.
- Added command output redirection to files where supported.

#### Global CLI Options

- Added repository selection through `--repo`.
- Added output-format selection through `--format`.
- Added direct output-file selection through `--output`.
- Added configuration selection through `--config`.
- Added configuration-profile selection through `--profile`.
- Added log-level selection through `--log-level`.
- Added log-format selection through `--log-format`.
- Added log-file selection through `--log-file`.
- Added verbose execution through `--verbose`.
- Added debug execution through `--debug`.
- Added tracing support through `--trace`.
- Added profiling support through `--profile-cpu`.
- Added memory profiling support through `--profile-mem`.
- Added help and version options.
- Added interactive execution control.

#### Command-Specific Options

- Added depth controls for applicable repository and intelligence operations.
- Added result limits for applicable search operations.
- Added category filtering.
- Added severity filtering.
- Added redaction controls.
- Added force controls for applicable configuration operations.
- Added analysis-level controls for applicable logging and analysis operations.

#### Configuration

- Added structured CLI configuration management.
- Added configuration initialization.
- Added configuration listing.
- Added configuration value retrieval.
- Added configuration value updates.
- Added configuration value removal.
- Added configuration validation.
- Added named configuration profiles.
- Added support for repository, output, logging, analysis, and related configuration settings exposed through the CLI.

#### Diagnostics and Operations

- Added structured operational logging.
- Added configurable log levels.
- Added text and JSON log output.
- Added log-file output.
- Added diagnostic collection.
- Added health reporting.
- Added execution tracing.
- Added runtime diagnostic inspection.
- Added CPU profiling support.
- Added memory profiling support.
- Added runtime statistics.

#### Documentation

- Added a comprehensive CLI user and developer guide.
- Added command reference documentation.
- Added CLI output and reporting documentation.
- Added CLI configuration documentation.
- Added CLI logging and diagnostics documentation.
- Added CLI workflow examples.
- Added command-line troubleshooting guidance.
- Added executable usage examples for supported commands and options.

### Changed

- Extended Limoxel with a unified user-facing command-line experience across repository analysis, search, intelligence, graph operations, reporting, configuration, and diagnostics.
- Extended command output handling to support human-readable and machine-readable workflows.
- Extended repository operations with consistent command-line access.
- Extended engineering analysis capabilities with command-line inspection and reporting workflows.
- Extended repository graph capabilities with command-line traversal and export workflows.
- Extended configuration management with command-line administration and named profiles.
- Extended operational tooling with command-line logging, health, debugging, and profiling workflows.
- Improved consistency of command structure, option handling, output behavior, and command documentation across the CLI.
- Added an interactive workflow for users who prefer stateful repository exploration instead of independent one-shot commands.

### Fixed

- Improved command-line error handling and user-facing diagnostic behavior.
- Improved consistency of command validation and option handling.
- Improved handling of structured command output.
- Improved configuration validation and configuration command behavior.
- Improved operational diagnostics for troubleshooting command execution.
- Improved command documentation coverage and alignment with supported CLI behavior.

### Security

- Sensitive values exposed through supported diagnostic, logging, configuration, and output workflows are subject to the CLI's redaction behavior.
- Repository analysis remains focused on the repository supplied to Limoxel and does not require modification of repository source files for analysis operations.
- Configuration and diagnostic output are designed to avoid unnecessarily exposing sensitive values through supported redaction mechanisms.

### Documentation

- Added the complete CLI documentation set under `docs/06_cli/`.
- Added practical command examples, workflows, configuration guidance, diagnostics guidance, and troubleshooting information.
- Expanded the project README with CLI usage, installation, quick-start workflows, reporting, configuration, and diagnostics guidance.

---

## [1.2.1] - 2026-08-25

### Fixed

- Updated the release workflow to support GitHub immutable releases by creating releases as drafts before uploading generated platform binaries, then publishing the completed release.
- Improved release publication reliability without changing Limoxel runtime behavior or public functionality.

---

## [1.2.0] - 2026-08-25

### Added

#### Semantic Intelligence

- Added structured semantic understanding of repository entities.
- Added semantic models for repositories, packages, modules, symbols, types, functions, interfaces, and variables.
- Added type and interface resolution.
- Added scope and ownership resolution.
- Added symbol visibility and semantic validation.

#### Cross-Repository Intelligence

- Added analysis across files, packages, modules, repositories, and workspace relationships.
- Added shared dependency analysis.
- Added shared configuration analysis.
- Added package contract analysis.
- Added internal and public API relationship analysis.
- Added repository evolution analysis.

#### Engineering Navigation

- Added definition and declaration navigation.
- Added implementation navigation.
- Added reference and usage lookup.
- Added reverse dependency lookup.
- Added symbol, type, interface, package, and module hierarchies.
- Added call hierarchy traversal.
- Added dependency-chain traversal.

#### Engineering Analysis

- Added code quality analysis.
- Added dead-code detection.
- Added unused import and export analysis.
- Added duplicate logic detection.
- Added large-file and large-function analysis.
- Added dependency analysis.
- Added circular dependency detection.
- Added layer violation detection.
- Added coupling analysis.
- Added architecture analysis.
- Added module boundary analysis.
- Added layer consistency analysis.
- Added package cohesion analysis.
- Added configuration analysis.
- Added repository health analysis.

#### Knowledge Graph Intelligence

- Added semantic relationships to the repository knowledge graph.
- Added ownership relationships.
- Added dependency relationships.
- Added documentation relationships.
- Added configuration relationships.
- Added relationship inference.
- Added dependency inference.
- Added ownership inference.
- Added architecture inference.
- Added repository context generation.
- Added knowledge consistency validation.
- Added relationship validation.
- Added graph completeness validation.
- Added repository, package, symbol, module, and architecture context generation.
- Added engineering insights for complexity, dependencies, architecture, repository growth, and engineering risk.

#### Deterministic Reasoning

- Added deterministic change-impact analysis.
- Added symbol impact analysis.
- Added package impact analysis.
- Added module impact analysis.
- Added repository impact analysis.
- Added dependency impact analysis.
- Added refactoring intelligence.
- Added safe rename analysis.
- Added safe move analysis.
- Added safe extraction analysis.
- Added safe deletion analysis.
- Added refactoring risk assessment.
- Added breaking-change detection.
- Added API change analysis.
- Added package change analysis.
- Added symbol removal analysis.
- Added interface change analysis.
- Added version compatibility analysis.
- Added deterministic engineering recommendations.
- Added dependency recommendations.
- Added architecture recommendations.
- Added performance recommendations.
- Added repository organization recommendations.
- Added engineering recommendations.
- Added structured handling of insufficient repository evidence.

#### Intelligence Coordination

- Added unified coordination across semantic analysis, repository context, knowledge graph operations, engineering analysis, and deterministic reasoning.
- Added structured intelligence interfaces for consuming repository knowledge.
- Added evidence-preserving engineering analysis results.

### Changed

- Extended Limoxel from repository analysis into structured engineering intelligence.
- Extended repository knowledge models to support semantic and engineering relationships.
- Extended repository analysis with deterministic navigation, analysis, reasoning, and recommendation capabilities.
- Extended repository validation coverage for intelligence operations.
- Extended repository documentation with the intelligence capability specifications.

### Fixed

- Improved deterministic ordering across intelligence operations.
- Improved graph traversal safety for cyclic and bounded repository relationships.
- Improved handling of incomplete repository evidence.
- Improved structured error handling across intelligence capabilities.
- Improved deterministic identifier generation and reduced unnecessary intermediate allocations.
- Improved validation of intelligence results and repository relationships.

### Security

- Intelligence operations remain repository-local and read-only.
- Intelligence analysis does not modify repository source files.
- Intelligence analysis does not require network access.
- Insufficient repository evidence is represented explicitly rather than converted into unsupported conclusions.

---

## [1.1.0]

### Added

- Production repository analysis capabilities.
- Repository discovery and deterministic file enumeration.
- Repository structure analysis.
- Source indexing.
- AST and symbol analysis.
- Dependency analysis.
- Cross-reference analysis.
- Repository knowledge graph.
- Repository search.
- Stable internal repository APIs.
- Production repository architecture and supporting engineering infrastructure.

### Changed

- Extended the engineering foundation with comprehensive repository-analysis capabilities.
- Established structured repository knowledge for higher-level engineering analysis.

### Documentation

- Expanded engineering documentation for repository analysis.
- Added repository capability specifications and engineering contracts.

---

## [1.0.0] - 2026-08-04

### Added

#### Engineering Foundation

- Established the production-grade engineering foundation.
- Implemented the core execution engine.
- Implemented platform infrastructure.
- Implemented workspace management.
- Implemented repository management.
- Implemented project management.
- Implemented filesystem abstraction.
- Implemented language management.
- Implemented parser infrastructure.
- Implemented extension framework.
- Implemented command-line interface.

#### Platform Infrastructure

- Added bootstrap system.
- Added runtime environment.
- Added configuration management.
- Added lifecycle management.
- Added logging infrastructure.
- Added context management.
- Added event foundation.
- Added service registry.

#### Engineering

- Established enterprise-grade repository organization.
- Established production package structure.
- Established repository-wide validation infrastructure.

#### Documentation

- Added foundation documentation.
- Added architecture documentation.
- Added engineering documentation.
- Added repository structure documentation.
- Added production README.

#### Validation

- Added unit testing.
- Added integration testing.
- Added build validation.
- Added runtime validation.
- Added architecture validation.
- Added performance baselines.
- Added documentation validation.
- Added API validation.
- Added dependency validation.
- Added repository cleanup and consistency checks.

#### Repository

- Added enterprise repository configuration.
- Added community health files.
- Added issue templates.
- Added pull request template.
- Added automated repository workflows.
- Added dependency management configuration.

### Security

- Initial production release.

---

[1.3.0]: https://github.com/unhield/limoxel/releases/tag/v1.3.0
[1.2.1]: https://github.com/unhield/limoxel/releases/tag/v1.2.1
[1.2.0]: https://github.com/unhield/limoxel/releases/tag/v1.2.0
[1.1.0]: https://github.com/unhield/limoxel/releases/tag/v1.1.0
[1.0.0]: https://github.com/unhield/limoxel/releases/tag/v1.0.0
