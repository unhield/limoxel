# Repository Discovery

Project  : Limoxel  
Category : Repository Capabilities  
Document : Repository Discovery  
Version  : 1.0  
Author   : Raj Joshi

---

# Purpose

This document defines the architecture and engineering contract for deterministic repository discovery within Limoxel.

Repository Discovery establishes the first repository-level capability built on top of the existing Limoxel foundation.

Its responsibility is to identify, validate, traverse, classify, and describe repository contents without modifying the existing core architecture.

Repository Discovery consumes existing Limoxel abstractions for workspace, project, repository, filesystem, ignore handling, language registration, and error handling.

The capability exists to transform an existing repository into a deterministic repository representation that can be consumed by subsequent repository capabilities.

---

# Scope

Repository Discovery is responsible for:

- Repository loading
- Repository root resolution
- Repository boundary validation
- Nested repository detection
- Repository file discovery
- Ignore-rule evaluation
- Deterministic filesystem traversal
- Language identification
- Repository metadata collection
- Discovery diagnostics
- Immutable discovery results

Repository Discovery provides the repository-level information required by subsequent capabilities such as project structure analysis, dependency analysis, source indexing, AST analysis, symbol analysis, cross-reference analysis, knowledge graph construction, and search.

Repository Discovery does not perform those later analyses.

---

# Architectural Position

Repository Discovery is an extension capability located under:

    internal/capabilities/repository

The capability layer consumes the existing Limoxel core.

The existing core does not depend on the capability layer.

The dependency direction is therefore:

    Limoxel Core
        |
        v
    Repository Capabilities
        |
        v
    Future Repository Analysis

Repository Discovery must remain downstream of the existing core architecture.

No dependency may be introduced from existing core packages into:

    internal/capabilities/repository

The capability must not create a second architecture for responsibilities already established by the core.

---

# Existing Abstractions

Repository Discovery reuses the existing Limoxel abstractions wherever they already provide the required responsibility.

## Workspace

The existing Workspace abstraction remains authoritative for workspace identity, root validation, and workspace ownership.

Repository Discovery must not introduce an alternative workspace model.

## Project

The existing Project abstraction remains authoritative for project identity, root validation, and workspace ownership.

Repository Discovery must not introduce an alternative project model.

## Repository

The existing Repository abstraction remains authoritative for repository identity, repository root representation, and project ownership.

Repository Discovery discovers repository information through this existing abstraction rather than replacing it.

## Filesystem

The existing filesystem abstraction remains authoritative for filesystem operations.

Repository Discovery must use the established filesystem facilities rather than introducing an independent operating-system filesystem abstraction.

## Ignore

The existing ignore facilities remain authoritative for repository exclusion behavior.

Repository Discovery may coordinate ignore evaluation but must not create a competing ignore engine when the existing infrastructure already provides the required behavior.

## Language

The existing language registry and language descriptors remain authoritative for language identification.

Repository Discovery must consume the existing language catalog rather than introducing a duplicate language registry.

## Error Handling

Repository Discovery must integrate with the existing Limoxel error system.

Capability-specific errors may be introduced only where an existing error category cannot accurately represent a discovery condition.

---

# Repository Discovery Model

Repository Discovery follows a deterministic processing model:

    Repository Path
          |
          v
    Path Validation
          |
          v
    Repository Resolution
          |
          v
    Repository Context
          |
          v
    Ignore Evaluation
          |
          v
    File Discovery
          |
          v
    Language Identification
          |
          v
    Metadata Aggregation
          |
          v
    Immutable Discovery Result

Each operation must have a defined responsibility and deterministic result.

Repository Discovery must not execute repository source code as part of this process.

---

# Repository Loading

Repository loading accepts a filesystem path and resolves it into an existing Limoxel repository representation.

The loader must:

- Validate that the supplied path exists.
- Resolve the path into a canonical representation.
- Reject unsupported path types.
- Determine the applicable repository boundary.
- Validate that traversal can remain within the permitted repository boundary.
- Preserve the existing Workspace, Project, and Repository relationships.
- Support repositories contained within larger workspaces.
- Support monorepository layouts.
- Detect nested repository boundaries.
- Handle empty repositories deterministically.
- Return structured errors for invalid repository inputs.

Repository loading must be read-only.

It must not modify repository files, Git state, branches, tags, configuration, or history.

---

# Repository Root Resolution

Repository root resolution determines the filesystem boundary used by discovery.

The resolved root must:

- Be canonical.
- Be deterministic.
- Be represented consistently across supported platforms.
- Remain within the permitted filesystem boundary.
- Provide a stable reference for repository-relative paths.

Repository-relative paths are the canonical paths exposed by repository discovery results.

Absolute filesystem paths may be used internally where required by existing filesystem abstractions, but they must not become the repository identity exposed by discovery results.

---

# Repository Boundary

Repository Discovery must enforce the repository boundary during every filesystem operation.

No discovery operation may traverse outside the selected repository boundary unless that behavior is explicitly supported by an existing Limoxel contract.

Boundary enforcement must cover:

- Relative path traversal.
- Symbolic links.
- Nested repository boundaries.
- Canonical path resolution.
- Directory recursion.

A path resolving outside the permitted boundary must never be silently accepted.

---

# Nested Repositories

Nested repositories must be detected explicitly.

A nested repository must not be silently treated as ordinary content belonging to the parent repository.

Nested repository information must preserve:

- Repository identity.
- Repository-relative location.
- Repository boundary.

Discovery must prevent:

- Recursive repository registration.
- Duplicate repository registration.
- Ambiguous repository ownership.
- Uncontrolled traversal between repository boundaries.

Nested repositories may be represented as separate repository contexts where supported by the existing repository model.

---

# Workspace and Project Context

Repository Discovery must preserve the existing hierarchy:

    Workspace
        |
        +-- Project
              |
              +-- Repository

Repository discovery must operate within this hierarchy rather than replacing it.

When repository discovery is initiated from an existing Workspace or Project context, the discovered repository must remain associated with that context according to the existing domain contracts.

No duplicate workspace or project container may be created merely to support discovery.

---

# File Discovery

File discovery recursively inventories repository files.

The inventory must support:

- Repository-relative path.
- File type.
- File extension.
- File size.
- Modification timestamp.
- Permission information where supported.
- Hidden-file classification.
- Symbolic-link classification.
- Language classification where available.

File discovery must use the existing filesystem abstraction.

The implementation must avoid direct operating-system filesystem calls when an existing Limoxel filesystem abstraction already provides the required operation.

---

# Deterministic Traversal

Filesystem enumeration order must not become observable Limoxel behavior.

Operating systems and filesystem implementations may return directory entries in different orders.

Repository Discovery must therefore establish a deterministic traversal order before processing entries.

For identical:

- Repository contents
- Discovery configuration
- Ignore configuration
- Supported environment semantics

the traversal order must remain equivalent.

Deterministic traversal applies to:

- Directories.
- Files.
- Nested repositories.
- Diagnostics.
- Aggregated results.

---

# Hidden Files

Hidden files are valid repository content unless excluded by repository rules or discovery configuration.

Repository Discovery must therefore distinguish between:

- Hidden files.
- Ignored files.
- Unsupported files.
- Symbolic links.
- Regular files.
- Directories.

Hidden status must not itself imply exclusion.

---

# Symbolic Links

Symbolic links require explicit handling because they may:

- Resolve outside the repository.
- Create traversal cycles.
- Produce duplicate filesystem paths.
- Cause uncontrolled repository expansion.

Repository Discovery must detect symbolic links and apply a deterministic traversal policy.

The implementation must:

- Prevent traversal outside the permitted repository boundary.
- Prevent recursive cycles.
- Avoid uncontrolled expansion.
- Preserve deterministic behavior.
- Represent unresolved links through the defined discovery result or diagnostic model.

Symbolic-link behavior must be governed by repository safety rather than convenience.

---

# Traversal Limits

Repository Discovery must protect itself against pathological repository structures.

The discovery implementation must support controlled limits for operations such as:

- Maximum traversal depth.
- Symbolic-link traversal.
- Path expansion.
- File enumeration.

Limit violations must produce structured discovery diagnostics or errors according to severity.

Traversal limits must never cause uncontrolled recursion or process instability.

---

# Ignore Evaluation

Repository Discovery must evaluate repository exclusion rules before performing unnecessary downstream processing.

Supported ignore sources must follow the existing Limoxel ignore architecture.

Where applicable, discovery may consume:

- Repository ignore files.
- Project-level ignore configuration.
- Workspace-level ignore configuration.
- Explicit discovery exclusions.

Ignore evaluation must remain deterministic.

An ignored file must not proceed into language identification or subsequent analysis unless the caller explicitly requests behavior that permits ignored content.

Repository Discovery must not duplicate ignore semantics already defined by the core.

---

# File Inventory

The discovered file inventory forms the filesystem foundation for subsequent repository analysis.

Each inventory entry must provide a stable repository-relative identity.

The inventory must distinguish at minimum:

- Regular files.
- Directories.
- Symbolic links.
- Ignored entries where diagnostics require their representation.
- Unknown or unsupported file types.

The inventory returned by discovery must be immutable from the perspective of consumers.

Later capabilities may derive additional information from the inventory but must not mutate the original discovery result.

---

# Language Identification

Language identification uses the existing Limoxel language registry.

Identification must be deterministic.

The implementation may use the language information already established by the core, including:

- File extensions.
- Recognized filenames.
- Existing language descriptors.
- Existing language matching rules.

Language identification must not introduce a second language catalog.

When a file cannot be identified confidently, the result must remain explicitly unknown.

An unknown file must not be assigned an inaccurate language merely to increase classification coverage.

---

# Language Distribution

Repository Discovery must aggregate language information from the discovered file inventory.

The resulting language distribution may include:

- Language.
- File count.
- Associated extensions.
- Relative distribution.

Aggregation must be derived from the already discovered inventory.

Repository Discovery must not perform an unnecessary second full filesystem scan merely to calculate language statistics.

Ordering of language results must be deterministic.

---

# Repository Metadata

Repository metadata provides a deterministic description of repository state where that information is locally available.

Metadata may include:

- Repository name.
- Repository root.
- Default branch where deterministically available.
- Latest commit.
- Commit statistics.
- Contributors.
- Tags.
- Releases where locally available.
- Repository age where deterministically derivable.

Metadata that cannot be established from the available repository state must remain explicitly unavailable.

Repository Discovery must never fabricate metadata.

Remote hosting information must not become an implicit dependency of local repository discovery.

Any future remote metadata provider must be isolated behind an explicit boundary.

---

# Git Metadata

Git metadata collection is separate from filesystem discovery.

Git metadata collection may provide:

- Git repository root.
- Current branch.
- Default branch where available.
- Latest commit.
- Commit statistics.
- Contributors.
- Tags.
- Locally available release information.

Git metadata collection must be read-only.

Repository Discovery must never:

- Create commits.
- Modify branches.
- Modify tags.
- Modify Git configuration.
- Alter the working tree.
- Execute repository hooks.

Unavailable Git information must be represented explicitly.

---

# Discovery Result

Repository Discovery produces one immutable discovery result.

The result logically contains:

    Discovery Result
        |
        +-- Repository
        +-- Root
        +-- Files
        +-- Nested Repositories
        +-- Languages
        +-- Metadata
        +-- Diagnostics

The result must preserve the distinction between:

- Repository identity.
- Filesystem inventory.
- Language classification.
- Repository metadata.
- Discovery diagnostics.

The discovery result is the primary input for subsequent repository capabilities.

Subsequent capabilities should consume this result rather than unnecessarily repeating repository traversal.

---

# Diagnostics

Discovery diagnostics must distinguish between conditions that prevent discovery and conditions that only reduce the completeness of discovered information.

## Fatal Conditions

Fatal conditions prevent a valid discovery result from being produced.

Examples include:

- Invalid repository path.
- Unusable repository root.
- Unrecoverable filesystem failure.
- Repository-boundary violation.
- Invalid repository state required for discovery.

## Non-Fatal Conditions

Non-fatal conditions allow discovery to continue.

Examples include:

- Inaccessible optional file metadata.
- Unavailable optional Git metadata.
- Unresolved symbolic links.
- Unsupported file types.
- Optional repository metadata unavailable.

Diagnostics must be deterministic and structured.

Sensitive filesystem information must not be exposed unnecessarily.

---

# Security Boundaries

Repository Discovery is a read-only capability.

Repository contents must be treated as untrusted data.

The capability must enforce:

- Repository-boundary validation.
- Path traversal protection.
- Symbolic-link boundary protection.
- Controlled recursion.
- Resource limits.
- Safe handling of malformed paths.
- Safe handling of malformed ignore rules.
- Safe handling of malformed repository metadata.
- No implicit secret extraction.
- No repository modification.

Repository Discovery must never execute arbitrary source code, scripts, repository hooks, or build commands as part of ordinary discovery.

---

# Performance Requirements

Repository Discovery must minimize unnecessary filesystem work.

The preferred processing model is:

    Filesystem Traversal
            |
            v
      File Inventory
            |
            v
    Language Identification
            |
            v
    Metadata Aggregation

Information already obtained during discovery must be reused by later processing within the capability.

The implementation must avoid:

- Repeated complete repository scans.
- Duplicate ignore evaluation.
- Duplicate language identification.
- Unnecessary file reads.
- Unnecessary metadata operations.

Performance optimizations must never compromise deterministic behavior or repository safety.

---

# Package Responsibilities

The capability package may internally separate responsibilities into components such as:

- Repository Loader
- File Discovery
- Ignore Evaluation
- Language Identification
- Metadata Collection
- Discovery Orchestration

These components remain implementation details unless an explicit architectural requirement establishes a public contract.

The orchestration component coordinates the discovery process.

Each specialized component owns one clearly defined responsibility.

No component should duplicate an existing core responsibility.

---

# Dependency Rules

Repository Discovery may depend on approved existing core abstractions.

Repository Discovery must not:

- Modify existing core contracts for convenience.
- Introduce reverse dependencies into the core.
- Duplicate filesystem infrastructure.
- Duplicate language registration.
- Duplicate repository domain models.
- Bypass established error handling.
- Create circular dependencies.
- Introduce unnecessary third-party dependencies.

Every dependency must have a direct architectural justification.

---

# API Boundary

Repository Discovery remains an internal capability.

Its implementation must not prematurely become part of the public SDK.

Internal contracts should remain minimal and purpose-specific.

Public exposure must occur only through an approved public API architecture once the underlying capability contracts are sufficiently stable.

---

# Extensibility

Repository Discovery must provide a stable foundation for future repository capabilities.

Future consumers may include:

- Project Structure Analysis
- Dependency Analysis
- Source Code Indexing
- AST Analysis
- Symbol Analysis
- Cross Reference Analysis
- Knowledge Graph Construction
- Repository Search

Those capabilities must consume repository discovery information rather than requiring Repository Discovery to absorb their responsibilities.

Repository Discovery therefore establishes repository knowledge without becoming a general-purpose analysis engine.

---

# Determinism

Deterministic behavior is a mandatory architectural property.

For equivalent repository state and equivalent discovery configuration, Repository Discovery must produce equivalent results.

Determinism applies to:

- Repository resolution.
- Repository boundaries.
- Traversal order.
- Ignore decisions.
- File identities.
- Language classification.
- Metadata aggregation.
- Diagnostics.
- Result ordering.

Filesystem enumeration order, map iteration order, or platform-specific incidental behavior must never become an uncontrolled source of result variation.

---

# Compatibility

Repository Discovery must preserve the existing Limoxel core architecture.

Future changes must prefer:

1. Additive capability-layer behavior.
2. New internal capability abstractions.
3. Adapters around existing contracts.
4. Explicit extension points.

Modification of existing core architecture is not part of normal Repository Discovery development.

A requirement that cannot be satisfied without modifying the existing core must trigger architectural review before implementation continues.

---

# Acceptance Criteria

Repository Discovery is considered architecturally complete when it provides:

- Deterministic repository loading.
- Repository root resolution.
- Repository-boundary enforcement.
- Nested repository detection.
- Workspace and Project context preservation.
- Recursive file discovery.
- Hidden-file handling.
- Symbolic-link safety.
- Traversal limits.
- Repository-relative file identities.
- File metadata.
- Deterministic ignore evaluation.
- Deterministic language identification.
- Language distribution.
- Repository metadata.
- Git metadata where locally available.
- Immutable discovery results.
- Structured diagnostics.
- Read-only behavior.
- Repository security boundaries.
- Deterministic result ordering.
- Efficient reuse of discovery information.
- Reuse of existing Limoxel core abstractions.
- No unnecessary dependencies.
- No reverse dependency into the core.
- No replacement of existing domain models.

---

# Architectural Guardrails

Implementation must stop and return to architectural review if any proposed implementation requires:

- Modification of an existing core package merely for convenience.
- Replacement of an existing filesystem abstraction.
- Replacement of the existing language registry.
- Replacement of the existing Workspace, Project, or Repository model.
- A dependency from the core into the capability layer.
- Uncontrolled filesystem traversal.
- Repository-boundary escape.
- Uncontrolled symbolic-link traversal.
- Non-deterministic result ordering.
- Execution of repository-controlled code.
- Unjustified third-party dependencies.
- Mutation of repository state.

These conditions are architectural violations, not implementation details.

---

# Authority

This document defines the repository discovery architecture and engineering boundaries for the Repository Capabilities layer.

The existing Limoxel Core Foundation documentation remains authoritative for all core contracts, domain models, filesystem abstractions, language abstractions, error handling, dependency direction, and architectural principles.

Where this document conflicts with an established Core Foundation contract, the existing core contract takes precedence and this document must be revised before implementation proceeds.

---

# Applicability

The principles and contracts defined in this document apply to all repository discovery implementations and all components that consume its discovery results.

They govern:

- Repository loading
- Filesystem discovery
- Ignore evaluation
- Language identification
- Metadata collection
- Discovery orchestration
- Repository discovery consumers
- Future extensions of repository discovery

All implementations must remain consistent with the established Limoxel architectural principles of determinism, minimal coupling, explicit boundaries, maintainability, and extension without unnecessary modification of existing foundations.

---

# Change Policy

Repository Discovery should evolve through additive capability-layer changes whenever possible.

Changes must preserve:

- Existing core contracts.
- Existing domain models.
- Deterministic behavior.
- Repository-boundary guarantees.
- Read-only behavior.
- Compatibility with existing capability consumers.

Changes that require modification of the existing core architecture require explicit architectural review before implementation.

The capability must not use feature growth as justification for weakening the established core boundaries.