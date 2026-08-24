# Source Code Indexing

Project    : Limoxel  
Category   : Repository  
Document   : Source Code Indexing  
Version    : 1.0  
Author     : Raj Joshi

---

# Purpose

This document defines the architecture and engineering contract for deterministic source code indexing within Limoxel.

Source Code Indexing catalogs engineering artifacts discovered within a repository and transforms repository files and their structural relationships into deterministic indexes.

The resulting indexes provide structured repository information for downstream capabilities.

Source Code Indexing is deterministic engineering infrastructure.

It does not use AI or probabilistic reasoning to establish repository facts.

---

# Scope

Source Code Indexing is responsible for:

- Source file indexing.
- Test file indexing.
- Generated file indexing.
- Unsupported file detection.
- Duplicate filename detection.
- Relative path tracking.
- File metadata.
- File hashing.
- Encoding detection.
- Line-ending detection.
- Package indexing.
- Package names.
- Package paths.
- Package imports.
- Package exports.
- Package documentation.
- Package ownership.
- Package statistics.
- File relationship indexing.
- Import relationships.
- Package relationships.
- Parent-child relationships.
- Test-to-source relationships.
- Configuration-to-source relationships.
- Documentation-to-module relationships.
- Repository statistics.
- Index persistence.
- Fast lookup.
- Incremental updates.
- Cache invalidation.
- Serialization.

Source Code Indexing does not perform AST parsing or symbol extraction.

---

# Architectural Position

Source Code Indexing is implemented within the repository capability boundary:

    internal/
    └── capabilities/
        └── repository/

The capability consumes repository and structural information provided by established Limoxel capabilities.

The preferred processing relationship is:

    Repository Discovery
            |
            v
    Project Structure
            |
            v
    Dependency Information
            |
            v
    Source Code Indexing
            |
            +-- Source File Index
            +-- Package Index
            +-- File Relationship Index
            +-- Repository Statistics
            +-- Persistent Index
            |
            v
    Future Repository Intelligence

The capability must not require changes to the established core architecture.

---

# Source File Index

The source file index provides a deterministic record for files discovered within the repository.

Each indexed file should retain sufficient information to identify and classify the file without requiring consumers to rescan the repository.

The source file index must support:

- File identity.
- Relative path.
- File type.
- Language where known.
- Test classification.
- Generated-file classification.
- File metadata.
- File hash.
- Encoding.
- Line-ending format.

---

# Supported Source Files

The initial source indexing capability must support Go source files.

The architecture must allow additional source languages to be added through additive extensions.

Language-specific indexing behavior must not require changes to unrelated repository indexing logic.

Unsupported source formats must remain explicitly represented as unsupported where detection is possible.

---

# Test File Indexing

Test files must be represented as source artifacts with explicit test classification.

The index should preserve:

- Test file identity.
- Relative path.
- Associated package where deterministically known.
- Source relationship where deterministically known.
- File metadata.

Test files must not be treated as ordinary source files without classification.

---

# Generated File Indexing

Generated files must be indexable and explicitly classified.

Generated status must be established using deterministic repository or language-specific evidence.

The capability must distinguish:

- Generated files.
- Handwritten source files.
- Unknown generation status.

A file must not be classified as generated solely because its filename appears unusual.

---

# Unsupported File Detection

Unsupported files must be identified without attempting unsafe or undefined parsing.

Unsupported status must distinguish between:

- Known unsupported format.
- Unknown format.
- Binary content.
- Unrecognized source format.

Unsupported files must not cause unrelated repository indexing to fail.

---

# Duplicate Filename Detection

Duplicate filename detection must operate using repository-relative identity and defined filename semantics.

Two files with identical basenames but different paths are not automatically duplicates.

For example:

    package_a/config.go
    package_b/config.go

represent distinct files.

Duplicate detection must therefore distinguish:

- Same filename.
- Same relative path.
- Same content.
- Same logical artifact.

These concepts must not be conflated.

---

# Relative Paths

Repository-relative paths are the canonical path representation within the source index.

Absolute filesystem paths must not become persistent repository identity.

Relative paths must use the established repository path conventions.

Path normalization must be deterministic.

Equivalent paths must resolve to a single canonical representation.

---

# File Metadata

File metadata may include:

- Relative path.
- File size.
- Modification information where appropriate.
- File type.
- Language classification.
- Test classification.
- Generated classification.
- Encoding.
- Line-ending format.

Metadata must remain subordinate to established repository identity and filesystem abstractions.

Source Code Indexing must not create an independent filesystem abstraction.

---

# File Hashes

The index must support deterministic file content hashing.

Hashes provide content identity and support:

- Change detection.
- Incremental indexing.
- Cache validation.
- Persistent index consistency.

The selected hashing mechanism must be documented by implementation-level engineering specifications.

Hash values must represent file content rather than filesystem location.

Path identity and content identity must remain separate concepts.

---

# Encoding Detection

The index must detect file encoding where deterministic detection is possible.

The capability must distinguish:

- Supported encoding.
- Detected encoding.
- Unknown encoding.
- Invalid byte sequence.

Encoding detection must not silently reinterpret invalid data.

The original file content must not be modified during encoding detection.

---

# Line Ending Detection

The index must detect line-ending representation.

The initial model should distinguish:

- LF.
- CRLF.
- CR.
- Mixed line endings.
- Unknown.

Line-ending detection is observational.

Source Code Indexing must never normalize or rewrite source files.

---

# Package Index

The package index provides a deterministic representation of packages discovered within the repository.

Each package should preserve:

- Package name.
- Package path.
- Associated files.
- Imports.
- Exports where deterministically available.
- Documentation.
- Ownership.
- Statistics.

Package indexing must build on established repository and language information.

---

# Package Names

Package names must be extracted according to the supported language's deterministic package rules.

Package names must not be inferred solely from directory names when authoritative package declarations are available.

Where package identity cannot be established reliably, the package must remain explicitly unresolved.

---

# Package Paths

Package paths must preserve the relationship between:

- Repository-relative location.
- Module identity.
- Package identity.

Path construction must reuse established module and repository models.

Package paths must remain deterministic across equivalent repository states.

---

# Package Imports

The package index may contain import information required to represent package relationships.

Import information must preserve the source package and referenced package identity.

Source Code Indexing records these relationships for indexing purposes.

Dependency semantics remain the responsibility of Dependency Analysis.

---

# Package Exports

Where the supported language provides deterministic package export information, the package index may record it.

Exports must not be inferred through semantic reasoning.

Source Code Indexing must preserve the distinction between:

- Declared exports.
- Public symbols.
- Internal symbols.

Detailed symbol extraction belongs to the AST and symbol capability.

---

# Package Documentation

Package-level documentation may be associated with the package index where the source language provides deterministic package documentation.

Documentation association must preserve the source relationship.

Source Code Indexing must not become a documentation-generation system.

---

# Package Ownership

Package ownership represents repository-defined ownership information where such information is deterministically available.

Ownership may originate from repository metadata or established ownership conventions.

Ownership must not be guessed from contributor activity.

If ownership cannot be established, the field must remain unavailable.

---

# Package Statistics

Package statistics provide deterministic structural measurements.

Possible measurements include:

- Number of source files.
- Number of test files.
- Number of generated files.
- Number of imports.
- Number of related files.
- File size totals.
- Line counts where reliably measurable.

Statistics must describe observed repository structure.

They must not be presented as semantic quality measurements.

---

# File Relationship Index

The file relationship index represents deterministic relationships between repository artifacts.

Supported relationship categories include:

- Import relationships.
- Package relationships.
- Parent-child relationships.
- Test-to-source relationships.
- Configuration-to-source relationships.
- Documentation-to-module relationships.

Relationships must have explicit types and direction where direction is meaningful.

---

# Import Relationships

Import relationships connect source artifacts through deterministic import information.

For example:

    Source A
        |
        | imports
        v
    Source B

The relationship must preserve:

- Source identity.
- Target identity.
- Relationship type.
- Location information where available.

Import relationship construction must not use semantic guessing.

---

# Package Relationships

Package relationships connect files and packages according to established package membership.

Package membership must be distinguished from dependency direction.

A file belonging to package A does not automatically depend on every file belonging to package B.

---

# Parent-Child Relationships

Parent-child relationships represent structural containment.

Examples include:

- Repository to directory.
- Directory to file.
- Package to source file.
- Module to package.

Containment must remain separate from dependency.

---

# Test-to-Source Mapping

The index should represent deterministic relationships between test artifacts and source artifacts.

Mappings may be established using:

- Language conventions.
- Package membership.
- Explicit test declarations.
- Repository structure.

A test file must not be mapped to a source file solely because their filenames are similar when stronger evidence is unavailable.

---

# Configuration-to-Source Mapping

The index may represent deterministic relationships between configuration artifacts and source artifacts.

Where configuration usage cannot be established statically, the relationship must remain unavailable.

Configuration filename similarity is insufficient evidence of usage.

Source Code Indexing must not execute application code to discover runtime configuration relationships.

---

# Documentation-to-Module Mapping

Documentation artifacts may be associated with modules where deterministic structural evidence exists.

Examples include:

- Module-local documentation.
- Module README files.
- Module-specific documentation directories.

The index must preserve the source of the mapping.

A documentation file must not be associated with a module merely because it exists somewhere in the repository.

---

# Repository Statistics

Repository statistics provide deterministic aggregate measurements of indexed repository artifacts.

The statistics model must support:

- Total files.
- Total packages.
- Total modules.
- Total lines of code.
- Language distribution.
- File-type distribution.
- Documentation coverage.
- Structural test coverage ratio.
- Configuration statistics.

---

# Total Files

Total file count must be derived from the indexed repository file set.

The calculation must define whether excluded files, unsupported files, generated files, and hidden files are included.

The selected policy must remain deterministic and consistent with repository scanning rules.

---

# Total Packages

Total package count must be derived from the package index.

A package must not be counted multiple times because it contains multiple source files.

Package identity must be stable.

---

# Total Modules

Total module count must reuse module information established by Project Structure.

Source Code Indexing must not independently redefine module identity.

---

# Lines of Code

Line counts must be treated as structural metrics.

The implementation must define how it handles:

- Blank lines.
- Comment lines.
- Generated files.
- Test files.
- Unsupported files.
- Invalid encodings.

The capability must not claim that line count represents software complexity or quality.

---

# Language Distribution

Language distribution must be derived from established language detection and indexed files.

The distribution must use consistent classification rules.

Unknown or unsupported languages must remain represented where applicable.

---

# File Type Distribution

File type distribution provides aggregate counts by recognized file type.

Classification must be deterministic.

A file must not belong to multiple mutually exclusive primary file-type categories.

---

# Documentation Coverage

Documentation coverage measures structural relationships between indexed repository artifacts and available documentation artifacts.

The metric must be explicitly defined.

It must not claim that a documented package is necessarily well documented.

---

# Structural Test Coverage Ratio

The repository statistics model may provide a structural test coverage ratio.

This metric is not runtime code coverage.

It must be based only on deterministic repository structure, such as relationships between test artifacts and source artifacts.

The metric must never be described as execution-based test coverage.

---

# Configuration Statistics

Configuration statistics provide aggregate structural information about configuration artifacts.

Possible measurements include:

- Number of configuration files.
- Configuration types.
- Configuration locations.
- Configuration-to-module relationships.

The capability must not evaluate configuration correctness.

---

# Index Storage

Source Code Indexing must support efficient index storage.

The storage architecture must distinguish between:

- In-memory indexes.
- Persistent indexes.
- Lookup structures.
- Serialized representations.
- Cache state.

Index storage must not require modification of the established core storage architecture.

---

# In-Memory Indexes

In-memory indexes provide fast access during repository analysis.

Indexes should support deterministic lookup by stable identifiers such as:

- Relative path.
- File identity.
- Package identity.
- Module identity.

Lookup behavior must remain deterministic.

---

# Persistent Cache

A persistent cache may store repository indexing information across analysis runs.

Persistent caching must not be treated as the authoritative source of repository truth.

The repository filesystem remains authoritative.

Cached information must be validated before reuse.

---

# Cache Invalidation

Cache invalidation must be deterministic.

Invalidation may be triggered by changes to:

- File content.
- File path.
- File metadata relevant to indexing.
- Repository structure.
- Index schema.
- Indexing configuration.

A stale cache must never silently override current repository state.

---

# Incremental Updates

The indexing engine must support incremental updates where practical.

Incremental indexing should identify changed repository artifacts using deterministic signals such as:

- Content hashes.
- File paths.
- File metadata.
- Repository structure changes.

Unchanged artifacts should not require unnecessary reprocessing.

Incremental updates must produce results equivalent to a clean full index of the same repository state.

---

# Serialization

Persistent repository indexes must support deterministic serialization.

Serialized representations must preserve:

- Stable identifiers.
- File metadata.
- Package metadata.
- Relationships.
- Statistics.
- Schema information.

Serialization format must be versionable.

A schema change must not silently reinterpret an older index as a current index.

---

# Index Versioning

Persistent index data must contain sufficient version information to determine compatibility.

The index schema version must be separate from the repository version.

A repository update must not automatically imply an index schema change.

Incompatible index versions must be rejected or rebuilt according to defined policy.

---

# Determinism

Source Code Indexing must produce deterministic results.

For equivalent repository state and equivalent indexing configuration, the resulting indexes must be equivalent.

Determinism applies to:

- File identity.
- Relative paths.
- File classifications.
- Hashes.
- Package identities.
- Relationships.
- Statistics.
- Serialized ordering.

Filesystem enumeration order must never determine index ordering.

---

# Partial Information

Indexing may encounter incomplete information.

Examples include:

- Unsupported file formats.
- Invalid encoding.
- Unresolved package identity.
- Unknown ownership.
- Missing documentation.
- Unavailable configuration relationships.

Unknown information must remain explicitly represented.

The capability must not fabricate repository facts to make the index appear complete.

---

# Error Handling

Source Code Indexing must integrate with the established Limoxel error architecture.

Errors must distinguish between:

- Invalid repository input.
- Invalid file data.
- Unsupported file type.
- Invalid encoding.
- Index persistence failure.
- Cache incompatibility.
- Serialization failure.

A single unsupported or malformed file should not unnecessarily invalidate the entire repository index.

Recoverable indexing failures should produce structured diagnostics where supported.

---

# Security

Source Code Indexing is a read-only capability.

It must never:

- Modify source files.
- Modify configuration files.
- Modify documentation.
- Execute source code.
- Execute build scripts.
- Execute test suites.
- Install dependencies.
- Execute package-manager lifecycle scripts.

Repository contents must be treated as untrusted input.

File parsing and metadata extraction must operate within established security boundaries.

---

# Performance

Source Code Indexing must avoid unnecessary repository rescanning.

It should reuse outputs from:

- Repository Discovery.
- Project Structure.
- Dependency Analysis.
- Language Detection.

The implementation must support efficient:

- File lookup.
- Package lookup.
- Relationship lookup.
- Incremental updates.
- Index persistence.

Large repositories must not require full index reconstruction when deterministic change detection proves that only a subset of artifacts changed.

---

# Large Repository Handling

The index must remain suitable for repositories containing:

- Large file counts.
- Large package counts.
- Large module counts.
- Large relationship graphs.

Large collections must use efficient data structures.

Traversal must avoid unnecessary recursion depth.

Result ordering must remain deterministic.

Memory usage must be considered when building large in-memory indexes.

---

# API Boundary

Source Code Indexing remains an internal repository capability.

Internal indexing structures should remain implementation details.

Consumers should interact through stable index models and capability contracts.

Public SDK exposure belongs to the established public API architecture and must not be introduced merely to support internal indexing.

---

# Extensibility

The indexing architecture must support additive extensions.

Additional language support should be introduced without rewriting common file-indexing infrastructure.

Additional file classifications should be introduced through explicit extensions.

Additional relationship types should be introduced without invalidating existing relationship semantics.

Additional persistent storage mechanisms must remain behind explicit storage boundaries.

---

# Separation from Dependency Analysis

Source Code Indexing consumes dependency information where necessary but does not replace Dependency Analysis.

Dependency Analysis establishes dependency relationships.

Source Code Indexing records and exposes structural indexing information required by downstream capabilities.

The two capabilities must not duplicate authoritative dependency logic.

---

# Separation from AST and Symbol Analysis

Source Code Indexing must not become an AST parser or symbol database.

It may provide source files and package information required by the AST and symbol capability.

The following responsibilities remain outside this capability:

- AST construction.
- Syntax-tree analysis.
- Symbol extraction.
- Symbol relationships.
- Detailed documentation-to-symbol association.

These responsibilities require the dedicated AST and symbol capability.

---

# Separation from Semantic Analysis

Source Code Indexing must not perform semantic interpretation of source code.

It records deterministic repository structure and relationships.

It must not infer:

- Business meaning.
- Architectural intent.
- Developer intent.
- Semantic ownership.
- Code quality.
- Design patterns.

Those concerns belong to higher-level analysis capabilities.

---

# Acceptance Criteria

Source Code Indexing is considered complete when it provides:

- Repository source index.
- Source file indexing.
- Go source indexing.
- Test file indexing.
- Generated file indexing.
- Unsupported file detection.
- Duplicate filename detection.
- Relative path tracking.
- File metadata.
- File hashes.
- Encoding detection.
- Line-ending detection.
- Package index.
- Package names.
- Package paths.
- Package imports.
- Package exports where deterministically available.
- Package documentation.
- Package ownership where deterministically available.
- Package statistics.
- File relationship index.
- Import relationships.
- Package relationships.
- Parent-child relationships.
- Test-to-source mapping.
- Configuration-to-source mapping where deterministically available.
- Documentation-to-module mapping where deterministically available.
- Repository statistics.
- Total file statistics.
- Total package statistics.
- Total module statistics.
- Structural line-count statistics.
- Language distribution.
- File-type distribution.
- Documentation coverage metrics.
- Structural test coverage ratio.
- Configuration statistics.
- In-memory indexes.
- Persistent cache support.
- Fast lookup tables.
- Incremental updates.
- Deterministic cache invalidation.
- Serialization support.
- Index versioning.
- Read-only operation.
- No source execution.
- No build execution.
- No dependency installation.
- No unnecessary modification of established core architecture.

---

# Architectural Guardrails

Implementation must stop and return to architectural review if any proposal requires:

- Modification of established core packages merely for convenience.
- Replacement of established repository or filesystem abstractions.
- Execution of source code.
- Execution of build scripts.
- Execution of package-manager lifecycle scripts.
- AI-based repository classification.
- Semantic inference presented as deterministic repository fact.
- Treating package containment as dependency.
- Treating structural test relationships as runtime coverage.
- Treating line counts as quality measurements.
- Unbounded repository traversal.
- Unjustified third-party dependencies.
- Persistent cache being treated as repository truth.
- Silent use of stale indexes.
- Mixing AST or symbol responsibilities into the indexing layer.

These conditions represent architectural violations rather than implementation details.

---

# Architectural Stability

Source Code Indexing is an additive repository capability.

Its responsibility is to transform deterministic repository artifacts and established structural information into efficient, queryable indexes.

The capability must remain focused on indexing.

Higher-level capabilities may consume the indexes for parsing, symbol analysis, cross-reference analysis, knowledge graph construction, search, and intelligence without requiring the indexing layer to absorb those responsibilities.

The capability should therefore remain focused on one responsibility:

> Provide a deterministic, efficient, and persistent representation of repository files, packages, structural relationships, and repository statistics.

---

# Authority

This document defines the Source Code Indexing capability and its engineering boundaries.

The existing Limoxel Core Foundation documentation remains authoritative for Workspace, Project, Repository, Filesystem, Language, Storage, Configuration, Runtime, error handling, extension mechanisms, and established architectural principles.

Repository Discovery remains authoritative for repository boundaries, repository loading, file discovery, and repository-relative path semantics.

Project Structure remains authoritative for directory hierarchy, module discovery, workspace relationships, build-system detection, configuration discovery, and documentation discovery.

Dependency Analysis remains authoritative for dependency relationships and dependency graph semantics.

The AST and Symbol capability remains authoritative for syntax-tree parsing, symbol extraction, symbol relationships, and symbol-level documentation association.

Where this document conflicts with an established contract, the established contract takes precedence and this document must be revised before implementation proceeds.

---

# Applicability

The principles and contracts defined in this document apply to source code indexing and all consumers of its resulting indexes.

They govern:

- Source file indexing.
- Package indexing.
- File relationships.
- Repository statistics.
- Persistent repository indexes.
- Incremental indexing.
- Index serialization.
- Index consumers.

All implementations must remain consistent with Limoxel's principles of deterministic behavior, explicit evidence, read-only repository observation, minimal coupling, long-term maintainability, and extension without unnecessary modification of established foundations.

---

# Change Policy

Source Code Indexing should evolve through additive capability-layer changes whenever possible.

Changes must preserve:

- Existing core contracts.
- Repository Discovery contracts.
- Project Structure contracts.
- Dependency Analysis contracts.
- Deterministic indexing semantics.
- Repository-relative identity.
- Read-only repository behavior.
- Separation from AST and symbol analysis.

Additional language support should be introduced through explicit indexing extensions.

Additional relationship types should be introduced without changing the meaning of existing relationships.

Changes that require modification of the existing core architecture require explicit architectural review before implementation.