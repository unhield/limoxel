# Project Structure

Project    : Limoxel  
Category   : Repository  
Document   : Project Structure  
Version    : 1.0  
Author     : Raj Joshi

---

# Purpose

This document defines the architecture and engineering contract for deterministic project structure analysis within Limoxel.

Project Structure describes how a repository is organized internally.

It identifies directory hierarchy, packages, modules, workspaces, build systems, configuration assets, and engineering documentation without modifying repository contents.

The capability consumes the repository information established by Repository Discovery and extends it with structural knowledge.

It does not perform source-code semantic analysis, dependency resolution, AST construction, symbol analysis, or knowledge-graph construction.

---

# Scope

Project Structure is responsible for:

- Directory hierarchy discovery.
- Nested directory discovery.
- Package discovery.
- Module discovery.
- Workspace discovery.
- Vendor detection.
- Module descriptor detection.
- Build-system detection.
- Configuration asset discovery.
- Engineering documentation discovery.

The capability produces:

- Repository structure model.
- Module inventory and module graph.
- Build configuration model and build-system graph.
- Configuration inventory.
- Documentation index.

---

# Architectural Position

Project Structure is implemented within the repository capability boundary:

    internal/
    └── capabilities/
        └── repository/

The capability consumes existing Limoxel foundation services and Repository Discovery results.

The dependency direction is:

    Existing Limoxel Foundation
            |
            v
    Repository Discovery
            |
            v
    Project Structure
            |
            v
    Future Repository Analysis

Existing foundation packages must not depend on Project Structure.

Project Structure must not replace existing Workspace, Project, Repository, Filesystem, Language, or configuration abstractions.

---

# Relationship with Repository Discovery

Repository Discovery establishes the repository boundary and provides the deterministic file inventory required for structural analysis.

Project Structure consumes that information rather than performing an independent repository-wide filesystem scan.

The preferred flow is:

    Repository
        |
        v
    Repository Discovery
        |
        v
    File Inventory
        |
        v
    Project Structure
        |
        +-- Directory Graph
        +-- Module Inventory
        +-- Build Configuration
        +-- Configuration Inventory
        +-- Documentation Index
        |
        v
    Repository Structure Model

Project Structure must reuse repository-relative paths and file identities already established by Repository Discovery.

---

# Directory Graph

The directory graph represents the hierarchical organization of the repository.

It must support:

- Repository root.
- Child directories.
- Nested directories.
- Relative paths.
- Directory relationships.
- Package locations where detectable.
- Module locations where detectable.
- Workspace locations where detectable.
- Vendor locations where detectable.

The graph must preserve the actual repository hierarchy.

Directory relationships must not be inferred from naming conventions when filesystem evidence is available.

---

# Directory Hierarchy

Directory hierarchy must be derived from the repository file inventory.

The implementation must:

- Preserve repository-relative paths.
- Preserve parent-child relationships.
- Maintain deterministic ordering.
- Represent empty directories only when they are observable through the repository discovery model.
- Distinguish directories from files and symbolic links.
- Respect repository boundaries established during discovery.

Directory hierarchy must not be reconstructed through an unrelated second filesystem traversal unless the existing discovery result cannot provide the required information.

---

# Package Discovery

Project Structure may identify packages where package boundaries can be deterministically established from repository contents and supported project conventions.

Package discovery must remain structural.

It must not attempt to resolve package dependencies or semantic relationships.

Package identification must use established language and project conventions where available.

The capability must not introduce a new language-specific package model when the existing Limoxel architecture already provides the required abstraction.

---

# Module Discovery

Project Structure identifies project modules through deterministic module descriptors and repository structure.

The initial module detection set includes:

- `go.mod`
- `package.json`
- `Cargo.toml`
- `pom.xml`
- Gradle configuration
- `requirements.txt`
- `pyproject.toml`
- `composer.json`

Detection must be based on repository evidence.

A module must not be considered present merely because its language is detected.

Module detection and language detection are related but separate responsibilities.

---

# Module Inventory

Each detected module must provide sufficient information for later repository analysis.

The module inventory should support:

- Module type.
- Module descriptor.
- Repository-relative location.
- Associated package or directory where deterministically available.
- Associated language where deterministically available.
- Associated build system where deterministically available.

The inventory must use deterministic ordering.

A module descriptor must not be parsed as a dependency graph at this point.

Detailed dependency analysis belongs to a separate capability.

---

# Module Graph

The module graph represents structural relationships between detected modules where those relationships can be established deterministically from repository structure.

The graph must distinguish between:

- Module identity.
- Module location.
- Module containment.
- Workspace membership.
- Explicit structural relationships.

It must not claim dependency relationships that require dependency analysis.

A module graph is therefore a structural model, not a dependency graph.

---

# Workspace Discovery

Project Structure may identify workspace structures supported by repository conventions and available project metadata.

Workspace discovery must preserve the existing Limoxel Workspace abstraction.

The capability must not create a replacement workspace model.

Workspace relationships must be represented only when they can be established deterministically.

Ambiguous directory layouts must not be converted into asserted workspace relationships without sufficient evidence.

---

# Monorepository Structure

Project Structure must support repositories containing multiple modules or projects.

A monorepository may contain:

- Multiple applications.
- Multiple libraries.
- Multiple language ecosystems.
- Multiple module descriptors.
- Multiple build systems.
- Multiple documentation areas.

The resulting structure model must preserve these boundaries.

The capability must not flatten a monorepository into a single module merely because all content belongs to one repository.

---

# Vendor Detection

Project Structure must identify vendor directories where deterministic structural evidence exists.

Vendor detection must distinguish vendor content from ordinary project directories.

The capability must not assume that every directory named `vendor` has identical semantics across all supported ecosystems.

Vendor classification must be based on supported deterministic rules.

Vendor detection must not become dependency analysis.

---

# Build-System Detection

Project Structure identifies repository build systems through deterministic repository evidence.

The initial build-system detection set includes:

- Make.
- Taskfile.
- CMake.
- Maven.
- Gradle.
- npm.
- pnpm.
- yarn.
- Cargo.

Build-system detection must identify the presence and repository-relative location of supported build configurations.

Detection must not execute build commands.

Repository-controlled build scripts must never be executed merely to determine whether a build system exists.

---

# Build Configuration Model

The build configuration model represents detected build-system information.

It should support:

- Build-system identity.
- Configuration file.
- Repository-relative location.
- Associated module where deterministically available.
- Associated language or ecosystem where deterministically available.

The model must not attempt to execute or validate builds.

Build correctness, dependency resolution, compilation, packaging, and deployment are outside this capability.

---

# Multiple Build Systems

A repository may contain multiple build systems.

Project Structure must represent multiple detected build systems without selecting one as authoritative unless repository evidence explicitly establishes such a relationship.

Examples include repositories containing:

- Make and CMake.
- Maven and Gradle.
- npm and Make.
- Cargo and Make.

Multiple detected systems must remain distinguishable in the resulting build-system graph.

---

# Configuration Discovery

Project Structure identifies configuration assets contained within the repository.

The initial configuration categories include:

- YAML.
- JSON.
- TOML.
- ENV.
- INI.
- Properties.
- XML.

Configuration discovery is structural.

It identifies configuration assets and their locations.

It does not parse configuration semantics unless explicitly required by a later capability.

---

# Configuration Inventory

The configuration inventory should support:

- Configuration type.
- Repository-relative path.
- Associated module where deterministically available.
- Associated build system where deterministically available.
- Hidden-file classification where applicable.

Configuration files must remain repository data.

Project Structure must not execute configuration-defined commands or interpolate configuration values.

Sensitive configuration contents must not be loaded merely to identify the existence of the configuration asset.

---

# Environment Configuration

Environment configuration files require special handling because they may contain secrets.

Project Structure may identify environment configuration assets structurally.

The capability must not expose secret values merely to populate the configuration inventory.

For example, detection of:

    .env

does not require reading and returning the contents of `.env`.

Secret discovery is not a responsibility of Project Structure.

---

# Documentation Discovery

Project Structure identifies engineering documentation contained within the repository.

The initial documentation set includes:

- README.
- CONTRIBUTING.
- SECURITY.
- LICENSE.
- CHANGELOG.
- ROADMAP.
- ADR.
- Documentation directories.

Documentation discovery is structural.

It identifies documentation assets and their locations.

It does not analyze or summarize documentation contents.

---

# Documentation Index

The documentation index should support:

- Document type.
- Repository-relative path.
- Document category where deterministically available.
- Associated module or project where deterministically available.

The index must preserve deterministic ordering.

Documentation content remains owned by the repository and must not be modified by structural discovery.

---

# Repository Structure Model

Project Structure produces a consolidated structural model.

Conceptually:

    Repository Structure Model
        |
        +-- Directory Graph
        |
        +-- Package Inventory
        |
        +-- Module Graph
        |
        +-- Workspace Structure
        |
        +-- Vendor Inventory
        |
        +-- Build-System Graph
        |
        +-- Configuration Inventory
        |
        +-- Documentation Index

The model must preserve the distinction between structural facts and analytical relationships.

It must not contain inferred dependency relationships.

---

# Structural Evidence

Every structural classification must be based on deterministic repository evidence.

Evidence may include:

- Repository-relative paths.
- Known descriptor filenames.
- Known configuration filenames.
- Known documentation filenames.
- Existing Repository Discovery metadata.
- Existing language information.
- Existing workspace and project context.

The capability must not infer structure from probabilistic or AI-based reasoning.

---

# Determinism

Project Structure must produce deterministic results.

For equivalent:

- Repository state.
- Repository discovery result.
- Configuration.
- Supported detection rules.

the resulting structure model must be equivalent.

Determinism applies to:

- Directory ordering.
- Package ordering.
- Module ordering.
- Build-system ordering.
- Configuration ordering.
- Documentation ordering.
- Graph node identity.
- Graph relationship ordering.

Filesystem enumeration order and map iteration order must never determine observable output.

---

# Path Identity

All structural entities must use repository-relative paths as their stable repository location.

Path representation must follow the path normalization established by Repository Discovery.

The capability must not create a second path-normalization policy.

Equivalent filesystem paths must not produce duplicate structural entities.

---

# Error Handling

Project Structure must integrate with the established Limoxel error architecture.

Errors must distinguish between:

- Conditions that prevent structural analysis.
- Conditions that prevent classification of an individual asset.
- Optional information that is unavailable.

A single unrecognized configuration file must not invalidate the complete repository structure model.

Unsupported structures must remain representable as unknown or unclassified rather than being assigned inaccurate classifications.

---

# Security

Project Structure is a read-only capability.

It must never:

- Modify repository files.
- Modify Git state.
- Execute build commands.
- Execute scripts.
- Execute repository hooks.
- Execute package-manager lifecycle commands.
- Load secret values unnecessarily.
- Modify configuration.
- Modify documentation.

Repository-controlled files must be treated as untrusted data.

Configuration and documentation discovery must remain structural and bounded.

---

# Performance

Project Structure must reuse the Repository Discovery file inventory.

The implementation must avoid:

- Repeated complete repository traversal.
- Duplicate file classification.
- Duplicate language detection.
- Duplicate metadata collection.

Descriptor detection should operate primarily against already discovered repository-relative paths.

Where file contents are not required for structural classification, they must not be read unnecessarily.

Large repositories must remain practical to analyze.

Performance optimizations must not compromise deterministic output.

---

# API Boundary

Project Structure remains an internal repository capability.

Its structural models must not be prematurely exposed through the public SDK.

Internal detection mechanisms should remain private.

Consumers should depend on stable structural models rather than implementation-specific detectors.

---

# Extensibility

Project Structure must support future repository ecosystems without destabilizing existing structural contracts.

Additional module descriptors, build systems, configuration types, documentation types, and workspace conventions may be added through additive capability-layer changes.

Adding support for a new ecosystem must not require modification of the existing core when an extension-layer implementation can provide the required behavior.

Detection rules must remain explicit and deterministic.

---

# Compatibility

Project Structure must preserve compatibility with the established Limoxel foundation and Repository Discovery contracts.

The capability must reuse:

- Repository identity.
- Repository-relative paths.
- File inventory.
- Language information.
- Workspace context.
- Project context.
- Existing filesystem abstractions.
- Existing configuration abstractions where applicable.
- Existing error conventions.

Existing core contracts must not be modified merely to simplify structural analysis.

If an existing contract appears insufficient, an additive adapter or capability-layer abstraction must be considered first.

---

# Separation from Dependency Analysis

Project Structure must not perform dependency analysis.

The following are structural responsibilities:

- Directory hierarchy.
- Package locations.
- Module locations.
- Workspace relationships.
- Build-system presence.
- Configuration locations.
- Documentation locations.

The following are not structural responsibilities:

- Import graphs.
- Third-party dependency graphs.
- Dependency versions.
- Dependency risk.
- Circular dependency analysis.
- Dependency direction analysis.

Those responsibilities belong to dependency analysis.

This boundary must remain explicit.

---

# Separation from Source Analysis

Project Structure must not parse source code merely to improve structural classification.

It may use existing deterministic repository and language information.

Source parsing, AST construction, symbol extraction, and semantic analysis belong to later capabilities.

Project Structure must remain useful without requiring source-code semantic interpretation.

---

# Acceptance Criteria

Project Structure is considered complete when it provides:

- Deterministic directory hierarchy.
- Directory graph.
- Package discovery where structurally supported.
- Module detection.
- Module inventory.
- Module graph.
- Workspace discovery where structurally supported.
- Vendor detection.
- Build-system detection.
- Build configuration model.
- Configuration discovery.
- Configuration inventory.
- Documentation discovery.
- Documentation index.
- Repository structure model.
- Deterministic ordering.
- Repository-relative identity.
- Read-only behavior.
- Safe handling of configuration assets.
- No source-code execution.
- No dependency analysis.
- No unnecessary repository rescanning.
- Reuse of Repository Discovery results.
- Reuse of existing Limoxel foundation abstractions.
- No reverse dependency into the existing core.
- No unnecessary dependencies.

---

# Architectural Guardrails

Implementation must stop and return to architectural review if any proposed implementation requires:

- Modification of an existing core package merely for convenience.
- Replacement of the existing Repository Discovery model.
- Replacement of the existing Workspace or Project model.
- Replacement of the existing filesystem abstraction.
- A dependency from the existing core into the capability layer.
- Execution of repository-controlled commands.
- Execution of build systems for detection.
- Reading secret configuration values unnecessarily.
- Probabilistic structural classification.
- Dependency analysis inside the structural model.
- Source-code semantic analysis inside the structural model.
- Unjustified third-party dependencies.

These conditions represent architectural violations rather than implementation details.

---

# Architectural Stability

Project Structure is an additive structural capability.

Its responsibility is to transform deterministic repository evidence into a reusable structural representation.

It must not become a general analysis engine.

The capability should remain focused on one responsibility:

> Provide a deterministic structural model of how a repository is organized, configured, built, and documented.

Future capabilities may consume this model without requiring Project Structure to absorb their responsibilities.

---

# Authority

This document defines the Project Structure capability and its engineering boundaries.

The existing Limoxel Core Foundation documentation remains authoritative for Workspace, Project, Repository, Filesystem, Language, Parser, Extension, Runtime, Configuration, platform services, error handling, dependency direction, and established architectural principles.

The Repository Discovery documentation remains authoritative for repository loading, repository boundaries, file inventory, path identity, ignore handling, and repository-level discovery semantics.

Where this document conflicts with an established foundation or Repository Discovery contract, the established contract takes precedence and this document must be revised before implementation proceeds.

---

# Applicability

The principles and contracts defined in this document apply to repository structural analysis and all consumers of the resulting structural model.

They govern:

- Directory discovery.
- Package discovery.
- Module detection.
- Workspace discovery.
- Vendor detection.
- Build-system detection.
- Configuration discovery.
- Documentation discovery.
- Structural graph construction.
- Repository structure representation.

All implementations must remain consistent with Limoxel's principles of deterministic behavior, explicit evidence, minimal coupling, read-only repository observation, long-term maintainability, and extension without unnecessary modification of established foundations.

---

# Change Policy

Project Structure should evolve through additive capability-layer changes whenever possible.

Changes must preserve:

- Existing core contracts.
- Repository Discovery contracts.
- Repository-relative path semantics.
- Deterministic classification.
- Read-only behavior.
- Structural separation from dependency analysis.
- Structural separation from source-code analysis.

Additional ecosystem support should be introduced through explicit detection rules and capability-layer extensions.

Changes that require modification of the existing core architecture require explicit architectural review before implementation.