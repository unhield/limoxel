# Dependency Analysis

Project    : Limoxel  
Category   : Repository  
Document   : Dependency Analysis  
Version    : 1.0  
Author     : Raj Joshi

---

# Purpose

This document defines the architecture and engineering contract for deterministic dependency analysis within Limoxel.

Dependency Analysis identifies dependency relationships within a repository and between the repository and external dependencies.

The capability consumes repository structure, module information, source indexing information, and other deterministic repository facts as they become available.

It produces structured dependency information that can be consumed by later repository intelligence capabilities.

Dependency Analysis is deterministic engineering infrastructure.

It does not use AI or probabilistic reasoning to establish dependency relationships.

---

# Scope

Dependency Analysis is responsible for:

- Internal dependency analysis.
- External dependency analysis.
- Dependency version detection.
- Semantic version parsing.
- Indirect dependency detection.
- Dependency graph construction.
- Circular dependency detection.
- Orphan package detection.
- Dependency depth analysis.
- License detection.
- Dependency health analysis.
- Dependency inventory.
- Dependency graph representation.
- License inventory.
- Dependency health reporting.

The capability does not perform repository-wide semantic reasoning.

---

# Architectural Position

Dependency Analysis is implemented within the repository capability boundary:

    internal/
    └── capabilities/
        └── repository/

The capability consumes existing Limoxel foundation services and repository capability outputs.

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
    Dependency Analysis
            |
            v
    Future Repository Intelligence

Existing foundation packages must not depend on Dependency Analysis.

Dependency Analysis must not modify the established core architecture.

---

# Relationship with Repository Structure

Dependency Analysis builds upon structural information rather than replacing it.

The preferred processing relationship is:

    Repository
        |
        v
    Repository Discovery
        |
        v
    Project Structure
        |
        v
    Dependency Analysis
        |
        +-- Internal Dependencies
        +-- External Dependencies
        +-- Dependency Graph
        +-- Licenses
        +-- Dependency Health
        |
        v
    Dependency Knowledge

Dependency Analysis must reuse:

- Repository-relative paths.
- Module identities.
- Package locations.
- Workspace relationships.
- Language information.
- Build-system information.
- Repository metadata.

It must not independently reconstruct these concepts when authoritative information already exists.

---

# Internal Dependency Analysis

Internal dependency analysis identifies relationships between packages, modules, and other repository-owned components.

The initial internal dependency model must support:

- Import relationships.
- Package relationships.
- Module relationships where deterministically available.
- Dependency direction.
- Unused package detection.
- Duplicate package detection.

Internal dependency analysis must distinguish structural containment from actual dependency relationships.

A package located inside another directory is not automatically a dependency of that directory.

---

# Import Graph

The import graph represents deterministic import relationships between source-level packages or modules.

Each graph relationship must identify:

- Source component.
- Target component.
- Relationship type.
- Source location where available.
- Target location where available.

Import relationships must be derived from supported source-language representations.

The graph must not infer an import relationship merely because two packages share naming conventions.

Unsupported source syntax must remain explicitly unsupported rather than producing fabricated relationships.

---

# Package Graph

The package graph represents deterministic package-to-package relationships.

The graph must distinguish:

- Package identity.
- Package location.
- Package language.
- Import relationships.
- Module membership.

Package graph construction must remain compatible with the existing language and repository models.

The capability must not create language-specific package models that duplicate established Limoxel abstractions unnecessarily.

---

# Dependency Direction

Dependency Analysis must represent the direction of dependency relationships explicitly.

For a relationship:

    A -> B

A depends on B.

The graph must not reverse this relationship merely because B is located elsewhere in the repository.

Dependency direction must be deterministic and preserved across all graph representations.

---

# Unused Package Detection

Dependency Analysis may identify packages that have no inbound or outbound dependency relationships within the analyzed dependency scope.

Unused package detection must distinguish between:

- Truly isolated packages.
- Entry-point packages.
- Public packages.
- Tooling packages.
- Test-only packages.
- Generated packages.
- Packages outside the analyzed dependency boundary.

A package must not be declared unused solely because no relationship was discovered within an incomplete analysis scope.

---

# Duplicate Package Detection

Dependency Analysis may identify duplicate package identities or structurally conflicting package representations.

Duplicate detection must be based on deterministic package identity rules.

Similar names alone are insufficient evidence of duplication.

The capability must distinguish between:

- Duplicate package identity.
- Multiple versions of an external dependency.
- Separate packages with similar names.
- Vendored copies.
- Generated package copies.

---

# External Dependency Analysis

External dependency analysis identifies dependencies that originate outside the repository.

The initial analysis must support:

- Dependency identity.
- Declared version.
- Resolved version where available.
- Semantic version representation.
- Direct dependency classification.
- Indirect dependency classification.
- Unused dependency detection.
- Outdated dependency detection.

External dependency information must be derived from authoritative repository manifests and supported dependency metadata.

The capability must not fabricate dependency information from filenames or package naming patterns.

---

# Dependency Manifest Sources

Dependency Analysis must use supported dependency manifests detected during Project Structure analysis.

Initial supported module and dependency sources include repository ecosystems represented by:

- Go modules.
- npm packages.
- Cargo packages.
- Maven dependencies.
- Gradle dependencies.
- Python dependency manifests.
- Composer dependencies.

Additional ecosystems may be introduced through additive extensions.

Each dependency source must have explicit parsing and normalization rules.

---

# Direct Dependencies

A direct dependency is a dependency explicitly declared by the repository's supported dependency manifest or equivalent authoritative source.

Direct dependency information must preserve:

- Dependency identity.
- Declared version or constraint.
- Source manifest.
- Repository-relative manifest path.
- Associated module.

The capability must not classify an undeclared dependency as direct merely because it is observed in a resolved dependency graph.

---

# Indirect Dependencies

An indirect dependency is a dependency introduced through another dependency.

Indirect dependency classification must be based on deterministic dependency metadata.

The capability must distinguish:

    Direct Dependency
          |
          v
    Indirect Dependency

Indirect dependency information must not be inferred solely from package popularity or naming relationships.

Where the dependency ecosystem does not expose sufficient information, the relationship must remain unavailable.

---

# Dependency Version Detection

Dependency Analysis must detect dependency versions where the repository's dependency system provides them.

Version information may include:

- Exact version.
- Version constraint.
- Version range.
- Resolved version.
- Pseudo-version or ecosystem-specific equivalent where supported.

The original declared representation must not be destroyed when a normalized version is also available.

---

# Semantic Version Parsing

Semantic version parsing must normalize supported semantic versions into deterministic representations.

The parser must distinguish between:

- Major version.
- Minor version.
- Patch version.
- Pre-release information.
- Build metadata.
- Version constraints.
- Exact versions.

Invalid version strings must not be silently normalized into valid versions.

Ecosystem-specific version schemes that do not conform to semantic versioning must retain their native representation and must not be falsely converted into semantic versions.

---

# Outdated Dependency Detection

Dependency Analysis may identify dependencies that appear outdated when sufficient deterministic version information is available.

Outdated detection requires an authoritative comparison source.

The capability must not claim that a dependency is outdated merely because a newer version number exists elsewhere in an untrusted or incomplete source.

External registry queries must not become mandatory for local dependency analysis.

When current upstream information is unavailable, outdated status must remain unavailable.

---

# Unused Dependency Detection

Dependency Analysis may identify declared dependencies that are not referenced within the analyzed repository dependency model.

Unused dependency detection must distinguish between:

- Unused direct dependencies.
- Dependencies used through generated code.
- Runtime-only dependencies.
- Build-time dependencies.
- Tooling dependencies.
- Test-only dependencies.
- Reflection-based or dynamically resolved dependencies.

A dependency must not be declared unused merely because a static source-level reference was not found.

Detection confidence must remain evidence-based.

---

# Dependency Graph

The dependency graph represents the complete deterministic dependency relationships available to the capability.

Conceptually:

    Dependency Graph
        |
        +-- Repository Components
        |
        +-- Internal Dependencies
        |
        +-- External Dependencies
        |
        +-- Direct Dependencies
        |
        +-- Indirect Dependencies
        |
        +-- Dependency Metadata

Graph nodes must have stable identities.

Graph edges must have explicit direction and relationship type.

---

# Graph Identity

Every dependency graph node must have a deterministic identity.

Identity must distinguish:

- Repository packages.
- Repository modules.
- External dependencies.
- Dependency versions where version-specific identity is required.

Two nodes must not be merged merely because they have identical display names.

Graph identity must remain stable across equivalent repository states.

---

# Circular Dependency Detection

Dependency Analysis must detect circular dependency relationships where the analyzed dependency graph supports such analysis.

A cycle exists when dependency relationships form a path returning to an earlier node.

For example:

    A -> B
    B -> C
    C -> A

must be represented as a dependency cycle.

Cycle detection must be deterministic.

The capability must report the participating nodes and relationships without modifying the repository.

---

# Orphan Package Detection

Dependency Analysis may identify packages that are disconnected from the relevant dependency graph.

Orphan detection must distinguish between:

- Genuine isolated packages.
- Repository entry points.
- Public APIs.
- Tooling packages.
- Test packages.
- Generated packages.
- Packages excluded from the analysis scope.

An orphan classification must therefore include sufficient context for consumers to interpret the result correctly.

---

# Dependency Depth

Dependency depth measures the number of dependency relationships between a selected graph root and reachable dependency nodes.

Depth analysis must define:

- Root node.
- Traversal direction.
- Maximum depth.
- Cycle behavior.

Cycles must not cause unbounded traversal.

Depth calculation must remain deterministic.

---

# Graph Export

The dependency graph may provide deterministic export representations for downstream consumers.

Export formats must preserve:

- Node identity.
- Edge direction.
- Relationship type.
- Version information where applicable.

Graph export must not become coupled to a visualization framework.

Visualization is a separate consumer responsibility.

---

# License Detection

Dependency Analysis must identify licenses associated with dependencies where authoritative license information is available.

The initial license categories include:

- MIT.
- Apache.
- BSD.
- GPL.
- LGPL.
- MPL.
- Unknown.

License detection must distinguish between:

- Known license.
- Multiple licenses.
- License expression.
- Unknown license.
- Unavailable license metadata.

The capability must not guess a license from package names.

---

# License Inventory

The license inventory should provide:

- Dependency identity.
- License identity.
- License source.
- License expression where available.
- Repository-relative source where applicable.
- Availability state.

License ordering must be deterministic.

Unknown licenses must remain explicitly represented.

---

# License Ambiguity

A dependency may expose more than one license or an ambiguous license declaration.

The capability must preserve the source representation rather than selecting one license without evidence.

Where a license expression can be parsed deterministically, the normalized expression may be stored alongside the original representation.

Where it cannot be parsed safely, the original representation must remain available.

---

# Dependency Health

Dependency health evaluates the observable maintenance condition of dependencies.

The initial health criteria include:

- Archived projects.
- Deprecated packages.
- Abandoned dependencies.
- Active maintenance detection.
- Repository health scoring.

Health evaluation must remain evidence-based.

A dependency must not be classified as unhealthy merely because it has not received a recent release unless the available evidence and defined policy establish that conclusion.

---

# Archived Projects

An archived dependency may be identified when authoritative repository metadata explicitly indicates that the dependency repository is archived.

Archive status must not be inferred from inactivity alone.

An archived dependency may still be functional and must therefore remain distinct from a broken dependency.

---

# Deprecated Packages

Deprecated status must be represented when authoritative dependency metadata explicitly marks a package as deprecated.

Deprecation must not be inferred from:

- Low download counts.
- Old release dates.
- Low commit activity.
- Package age.
- Repository popularity.

Those signals may contribute to separate health analysis but do not establish deprecation.

---

# Abandoned Dependencies

A dependency may be considered potentially abandoned when sufficient deterministic evidence indicates prolonged lack of maintenance.

Abandonment detection requires explicit policy and measurable evidence.

The capability must distinguish:

- No recent activity.
- Potential abandonment.
- Explicit archival.
- Explicit deprecation.

These conditions must not be collapsed into a single status.

---

# Active Maintenance Detection

Active maintenance may be represented when available evidence demonstrates recent and ongoing repository activity.

Possible evidence may include:

- Recent commits.
- Recent releases.
- Recent version updates.
- Maintainer activity.

The capability must not equate activity with quality.

Active maintenance is an observed state, not a guarantee of engineering quality.

---

# Dependency Health Score

A dependency health score may aggregate supported health signals into a deterministic score.

The scoring model must be:

- Explicit.
- Documented.
- Deterministic.
- Reproducible.
- Versioned when its semantics change.

A score must never hide the underlying evidence.

Consumers must be able to inspect the individual health signals that contributed to the score.

---

# External Information

Dependency health and outdated dependency analysis may require information that is not contained within the local repository.

Remote metadata must remain optional.

When remote information is unavailable:

- Local dependency analysis must continue.
- Health fields dependent on remote evidence must remain unavailable.
- The capability must not fabricate current upstream status.

Remote service integration belongs behind an explicit provider boundary.

---

# Determinism

Dependency Analysis must produce deterministic results.

For equivalent repository state, dependency metadata, and analysis configuration, the resulting dependency model must be equivalent.

Determinism applies to:

- Dependency identity.
- Dependency versions.
- Direct and indirect classification.
- Graph nodes.
- Graph edges.
- Cycle detection.
- Dependency depth.
- License classification.
- Health signals.
- Health scores.
- Inventory ordering.

Map iteration order and external service response ordering must never become uncontrolled sources of result variation.

---

# Partial Information

Dependency analysis may operate on incomplete information.

Examples include:

- Missing dependency manifests.
- Incomplete source indexing.
- Unavailable lock files.
- Missing remote metadata.
- Unsupported package managers.
- Partial repository history.

Partial information must be represented explicitly.

The capability must never present a partial dependency graph as a complete repository dependency graph without indicating its analysis scope.

---

# Error Handling

Dependency Analysis must integrate with the established Limoxel error architecture.

Errors must distinguish between:

- Invalid repository input.
- Invalid dependency manifests.
- Unsupported dependency ecosystems.
- Invalid version expressions.
- Unavailable optional metadata.
- Unrecoverable graph construction failures.

A malformed dependency manifest should produce a structured diagnostic rather than causing unrelated repository analysis to fail unnecessarily.

---

# Security

Dependency Analysis is a read-only capability.

It must never:

- Modify dependency manifests.
- Modify lock files.
- Install dependencies.
- Execute package-manager commands.
- Execute build scripts.
- Execute dependency lifecycle scripts.
- Download and execute arbitrary dependency code.
- Modify repository state.

Dependency manifests and dependency metadata must be treated as untrusted input.

External metadata providers must remain isolated behind explicit security boundaries.

---

# Performance

Dependency Analysis must reuse outputs from:

- Repository Discovery.
- Project Structure.
- Language Detection.
- Repository Metadata.

The capability must avoid unnecessary repeated repository scans.

Dependency graph construction must use efficient graph algorithms appropriate for large repositories.

Cycle detection and depth analysis must have explicit traversal safeguards.

External metadata requests, if supported, must use bounded and controlled access.

Caching must not be introduced without a defined invalidation strategy.

---

# Large Repository Handling

Dependency Analysis must remain suitable for repositories with:

- Large package counts.
- Large module counts.
- Large dependency graphs.
- Deep dependency chains.
- Multiple dependency ecosystems.

Graph algorithms must avoid unbounded recursion.

Large collections must maintain deterministic ordering.

Performance optimizations must not alter dependency semantics.

---

# API Boundary

Dependency Analysis remains an internal repository capability.

Internal parsers, dependency-source adapters, license detectors, and health evaluators should remain private implementation details.

Consumers should interact with stable dependency models.

Public SDK exposure must occur only through the established public API architecture after the dependency contracts are sufficiently stable.

---

# Extensibility

Dependency Analysis must support additional dependency ecosystems through additive extensions.

A new ecosystem should provide explicit adapters for:

- Manifest detection.
- Dependency extraction.
- Version parsing.
- Direct and indirect classification.
- License metadata.
- Health metadata where supported.

Adding ecosystem support must not require modification of unrelated core architecture.

Ecosystem-specific behavior must remain isolated from the common dependency model.

---

# Compatibility

Dependency Analysis must preserve compatibility with established Limoxel architecture.

It must reuse:

- Repository identity.
- Repository-relative paths.
- File inventory.
- Language information.
- Module information.
- Workspace information.
- Existing filesystem abstractions.
- Existing error handling.

Existing core contracts must not be modified merely to simplify dependency analysis.

If an existing contract is insufficient, an additive capability-layer adapter must be considered first.

---

# Separation from Project Structure

Project Structure establishes:

- Directory relationships.
- Module locations.
- Workspace relationships.
- Build-system presence.
- Configuration locations.

Dependency Analysis establishes:

- Actual dependency relationships.
- Dependency direction.
- Dependency versions.
- Dependency graph.
- Dependency health.

Structural containment must not be treated as dependency.

This boundary must remain explicit.

---

# Separation from Source Indexing

Dependency Analysis may consume source index information when available.

It must not become the source indexing engine.

Source file metadata, hashes, encoding, line endings, generated-file classification, and complete source inventory remain responsibilities of source indexing.

Dependency Analysis consumes the required source information through stable interfaces.

---

# Separation from Semantic Intelligence

Dependency Analysis must not use AI to infer dependency relationships.

Dependency relationships must be established through deterministic:

- Source-language parsing.
- Module metadata.
- Dependency manifests.
- Lock files.
- Repository metadata.
- Explicit dependency rules.

Semantic reasoning may later consume the dependency graph.

It must not be used to construct the authoritative dependency graph.

---

# Acceptance Criteria

Dependency Analysis is considered complete when it provides:

- Internal dependency analysis.
- Import graph.
- Package graph.
- Dependency direction.
- Unused package detection.
- Duplicate package detection.
- External dependency inventory.
- Dependency version detection.
- Semantic version parsing where applicable.
- Direct dependency detection.
- Indirect dependency detection.
- Outdated dependency status where authoritative evidence exists.
- Unused dependency detection where deterministic evidence permits.
- Complete dependency graph within the supported analysis scope.
- Circular dependency detection.
- Orphan package detection.
- Dependency depth analysis.
- Deterministic graph export representation.
- License inventory.
- MIT detection.
- Apache detection.
- BSD detection.
- GPL detection.
- LGPL detection.
- MPL detection.
- Unknown license detection.
- Dependency health analysis.
- Archived-project detection.
- Deprecated-package detection.
- Potential abandonment detection.
- Active-maintenance detection.
- Deterministic health scoring where supported.
- Explicit partial-information semantics.
- Read-only operation.
- No dependency installation.
- No build execution.
- No repository modification.
- Reuse of established capability outputs.
- No unnecessary dependencies.
- No reverse dependency into the existing core.

---

# Architectural Guardrails

Implementation must stop and return to architectural review if any proposed implementation requires:

- Modification of an existing core package merely for convenience.
- Replacement of established Repository, Project, Workspace, or Filesystem abstractions.
- Installation of dependencies during analysis.
- Execution of package-manager lifecycle scripts.
- Execution of repository build commands.
- AI-based dependency inference.
- Fabrication of unavailable dependency metadata.
- Treating structural containment as dependency.
- Treating partial analysis as complete analysis.
- Unbounded graph traversal.
- Unjustified third-party dependencies.
- Mandatory external network access for local dependency analysis.

These conditions represent architectural violations rather than implementation details.

---

# Architectural Stability

Dependency Analysis is an additive repository capability.

Its responsibility is to transform deterministic repository and dependency evidence into structured dependency knowledge.

It must remain focused on dependency relationships and associated dependency metadata.

Future intelligence capabilities may consume the dependency graph without requiring Dependency Analysis to absorb semantic reasoning responsibilities.

The capability should therefore remain focused on one responsibility:

> Provide a deterministic, evidence-based representation of repository and external dependency relationships, associated licenses, and observable dependency health.

---

# Authority

This document defines the Dependency Analysis capability and its engineering boundaries.

The existing Limoxel Core Foundation documentation remains authoritative for Workspace, Project, Repository, Filesystem, Language, Parser, Extension, Runtime, Configuration, platform services, error handling, dependency direction, and established architectural principles.

Repository Discovery remains authoritative for repository loading, repository boundaries, file inventory, path identity, and repository-level discovery semantics.

Project Structure remains authoritative for directory hierarchy, module discovery, workspace structure, build-system detection, configuration discovery, and documentation discovery.

Where this document conflicts with an established foundation or capability contract, the established contract takes precedence and this document must be revised before implementation proceeds.

---

# Applicability

The principles and contracts defined in this document apply to dependency analysis and all consumers of the resulting dependency models.

They govern:

- Internal dependency analysis.
- External dependency analysis.
- Dependency inventories.
- Dependency graphs.
- Version analysis.
- License analysis.
- Dependency health analysis.
- Dependency metadata providers.
- Dependency graph consumers.

All implementations must remain consistent with Limoxel's principles of deterministic behavior, explicit evidence, read-only repository observation, minimal coupling, long-term maintainability, and extension without unnecessary modification of established foundations.

---

# Change Policy

Dependency Analysis should evolve through additive capability-layer changes whenever possible.

Changes must preserve:

- Existing core contracts.
- Repository Discovery contracts.
- Project Structure contracts.
- Deterministic dependency semantics.
- Explicit analysis scope.
- Read-only repository behavior.
- Separation from semantic intelligence.

Additional dependency ecosystems should be introduced through explicit capability-layer adapters.

Changes that require modification of the existing core architecture require explicit architectural review before implementation.