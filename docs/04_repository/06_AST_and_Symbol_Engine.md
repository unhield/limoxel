# Symbol & AST Engine

Project    : Limoxel  
Category   : Repository  
Document   : Symbol & AST Engine  
Version    : 1.0  
Author     : Raj Joshi

---

# Purpose

This document defines the architecture and engineering contract for deterministic source parsing, Abstract Syntax Tree construction, symbol extraction, documentation extraction, and symbol relationship construction within Limoxel.

The Symbol & AST Engine transforms indexed source artifacts into structured syntax and symbol information that can be consumed by higher-level repository capabilities.

The engine is deterministic.

It must derive repository facts from source structure and language rules rather than probabilistic interpretation.

---

# Scope

The Symbol & AST Engine is responsible for:

- Parsing Go source code into Abstract Syntax Trees.
- Handling syntax errors.
- Parsing source comments.
- Preserving formatting metadata.
- Supporting incremental parsing.
- Supporting parallel parsing.
- Extracting packages.
- Extracting structs.
- Extracting interfaces.
- Extracting functions.
- Extracting methods.
- Extracting variables.
- Extracting constants.
- Extracting types.
- Extracting generics.
- Extracting aliases.
- Associating documentation with symbols.
- Extracting package comments.
- Extracting struct comments.
- Extracting function comments.
- Extracting interface comments.
- Extracting method comments.
- Extracting TODO comments.
- Extracting FIXME comments.
- Building symbol relationships.
- Recording function ownership.
- Recording method receivers.
- Recording interface implementations.
- Recording struct embedding.
- Recording type aliases.
- Recording generic constraints.
- Validating AST parsing behavior.

The resulting capability provides:

- AST engine.
- Symbol database.
- Documentation database.
- Symbol relationship graph.

---

# Architectural Position

The Symbol & AST Engine is an additive repository capability.

Its implementation belongs within the repository capability boundary:

    internal/
    └── capabilities/

The engine consumes source artifacts made available by established repository indexing capabilities.

The conceptual processing relationship is:

    Repository Source
          |
          v
    Source Index
          |
          v
    AST Engine
          |
          +-- Abstract Syntax Trees
          |
          v
    Symbol Extraction
          |
          +-- Symbol Database
          |
          v
    Documentation Extraction
          |
          +-- Documentation Database
          |
          v
    Symbol Relationships
          |
          +-- Symbol Relationship Graph

The engine must extend established repository infrastructure rather than replace or redesign it.

---

# AST Engine

The AST engine parses supported source files into structured Abstract Syntax Trees.

The initial implementation supports Go source code.

AST construction must preserve the structural information required by downstream symbol and repository analysis.

The AST engine must not perform semantic reasoning beyond what is deterministically represented by the source language and parser.

---

# Go AST Parsing

Go source files must be parsed using deterministic Go parsing rules.

The parser must produce structured AST representations for valid source files.

The implementation must preserve sufficient source-location information to associate AST nodes with repository files and source positions.

AST identity must remain associated with the indexed source artifact from which it was produced.

---

# Syntax Error Handling

Invalid source syntax must be handled explicitly.

A syntax error must not be silently converted into a valid AST interpretation.

The engine must distinguish between:

- Successfully parsed source.
- Partially recoverable source where parser behavior permits recovery.
- Invalid source.
- Unparseable source.

Syntax diagnostics must retain enough information for consumers to understand which source artifact failed and where the failure occurred.

A malformed source file must not silently corrupt unrelated AST or symbol data.

---

# Comment Parsing

The AST engine must preserve source comments where the parser provides them.

Comments are required for documentation extraction and repository documentation analysis.

The engine must preserve the relationship between comments and the source structures with which they are associated.

Comments must not be treated as executable source.

---

# Formatting Metadata

Formatting metadata required by downstream repository analysis must be preserved.

Relevant source-location information may include:

- File position.
- Line.
- Column.
- Source range.

Formatting metadata must remain associated with the corresponding source artifact and AST structure.

The engine must not rewrite or normalize source files as part of parsing.

---

# Incremental Parsing

The engine must support incremental parsing where practical.

Incremental parsing should allow unchanged source artifacts to avoid unnecessary parsing.

Existing source indexes and content identity information should be used where available to determine whether reparsing is required.

Incremental parsing must produce results equivalent to a clean parse of the current repository state.

A stale AST must never be treated as current source truth.

---

# Parallel Parsing

The engine must support parallel parsing where practical.

Parallel parsing must preserve deterministic results.

The ordering in which files are parsed must not affect:

- AST contents.
- Symbol identity.
- Symbol relationships.
- Documentation associations.
- Final database ordering.

Concurrency must not introduce nondeterministic repository state.

---

# AST Identity

Each AST must remain associated with a stable source artifact identity.

The identity must be derived from established repository indexing information rather than independently redefining file identity.

Source path, file identity, and content identity must remain distinct concepts.

---

# Symbol Database

The Symbol Database provides structured representations of engineering symbols extracted from parsed source code.

The database must support the symbol categories defined by the repository specification:

- Packages.
- Structs.
- Interfaces.
- Functions.
- Methods.
- Variables.
- Constants.
- Types.
- Generics.
- Aliases.

Each symbol must retain sufficient information to identify its source location and structural context.

---

# Package Symbols

Package declarations must be represented as symbols where applicable.

Package symbols must preserve their relationship to:

- Source files.
- Package identity.
- Repository location.

Package identity must use established package information rather than being independently inferred from arbitrary naming conventions.

---

# Struct Symbols

Struct declarations must be extracted as structured symbols.

A struct symbol should preserve its relationship to:

- Declaring package.
- Source file.
- Source position.
- Declared fields.
- Embedding information where represented by the language structure.

Detailed symbol relationships are represented separately in the symbol relationship graph.

---

# Interface Symbols

Interface declarations must be extracted as structured symbols.

The symbol representation must preserve:

- Declaring package.
- Source file.
- Source position.
- Interface methods.
- Relevant type information.

Interface implementation relationships are represented separately.

---

# Function Symbols

Function declarations must be extracted as symbols.

Each function symbol must preserve its source identity and structural location.

The symbol representation must distinguish functions from methods.

Call relationships are not established by this capability.

---

# Method Symbols

Methods must be extracted separately from standalone functions.

Method symbols must preserve receiver information.

Receiver relationships are represented explicitly in the symbol relationship graph.

---

# Variable Symbols

Declared variables must be represented as symbols where they are part of the supported source structure.

Variable symbols must preserve their declaration context and source location.

---

# Constant Symbols

Declared constants must be represented as symbols.

Constant symbols must remain distinguishable from variables.

The engine must preserve their declaration context and source location.

---

# Type Symbols

Type declarations must be extracted as symbols.

The representation must preserve the relationship between the declared type and its source location.

Where the language syntax provides a distinct type form, that form should remain identifiable.

---

# Generic Symbols

Generic declarations and generic type information must be represented where supported by the Go language syntax.

The engine must preserve generic structure rather than flattening generic declarations into ordinary non-generic symbols.

Generic constraints must remain available for symbol relationship construction.

---

# Alias Symbols

Type aliases must be distinguishable from new type declarations.

The symbol representation must preserve that the declaration is an alias.

Alias relationships must remain explicit rather than being represented as independent unrelated types.

---

# Symbol Identity

Symbol identity must be deterministic.

The same source state must produce the same symbol identity.

Symbol identity must be sufficiently qualified to distinguish symbols with identical names in different scopes or packages.

A symbol name alone must not be assumed to be globally unique.

---

# Symbol Location

Each symbol should retain source-location information sufficient for deterministic navigation by downstream capabilities.

Location information should identify:

- Source file.
- Declaration position.
- Relevant source range where available.

The engine must use established repository-relative file identity.

---

# Documentation Database

The Documentation Database associates source documentation with the symbols to which that documentation belongs.

Documentation extraction must remain source-derived.

The engine must not generate or rewrite documentation.

The database must preserve the relationship between:

- Documentation text.
- Symbol.
- Source location.
- Documentation type.

---

# Package Comments

Package-level comments must be associated with the appropriate package symbol where the source structure permits deterministic association.

The original comment content must be preserved.

---

# Struct Comments

Comments associated with struct declarations must be associated with the corresponding struct symbol.

The association must be based on source structure rather than textual similarity.

---

# Function Comments

Comments associated with function declarations must be associated with the corresponding function symbol.

The engine must preserve the relationship between documentation and declaration.

---

# Interface Comments

Comments associated with interface declarations must be associated with the corresponding interface symbol.

The association must remain deterministic.

---

# Method Comments

Comments associated with method declarations must be associated with the corresponding method symbol.

Receiver information must remain available so that methods with identical names can be distinguished.

---

# TODO Comments

TODO comments must be detectable and represented as documentation metadata.

TODO entries should retain their source location.

TODO extraction is observational.

The engine must not automatically convert TODO comments into tasks or modify repository tracking systems.

---

# FIXME Comments

FIXME comments must be detectable and represented as documentation metadata.

FIXME entries should retain their source location.

FIXME extraction is observational.

The engine must not automatically modify source code or issue tracking systems.

---

# Symbol Relationship Graph

The Symbol Relationship Graph represents deterministic structural relationships between extracted symbols.

The defined relationship categories include:

- Function ownership.
- Method receivers.
- Interface implementations.
- Struct embedding.
- Type aliases.
- Generic constraints.

Relationships must have explicit types.

Relationships must connect stable symbol identities.

---

# Function Ownership

Function ownership represents the structural ownership of functions by their declaring package or applicable source structure.

Ownership must be derived from source declarations.

Ownership must not be inferred from naming conventions alone.

---

# Method Receivers

Method receiver relationships must connect each method to its receiver type.

The relationship must preserve the receiver information represented by the source declaration.

Receiver relationships must distinguish:

- Value receivers.
- Pointer receivers.

Where the language structure makes such distinctions available.

---

# Interface Implementations

Interface implementation relationships must be established according to deterministic language rules and available type information.

The engine must not claim an implementation relationship without sufficient structural evidence.

Interface implementation analysis must remain separate from general text matching.

---

# Struct Embedding

Struct embedding relationships must represent embedded fields and types according to Go language structure.

Embedding must remain distinct from ordinary field declaration.

The relationship graph must preserve the source-defined embedding relationship.

---

# Type Aliases

Type alias relationships must connect the alias symbol to the symbol or type it aliases where deterministically resolvable.

An alias must not be represented as an unrelated type.

---

# Generic Constraints

Generic constraints must be represented as relationships between generic declarations and their applicable constraints.

The relationship must preserve the source-defined generic structure.

Constraint extraction must remain deterministic.

---

# Relationship Identity

Every relationship must identify:

- Source symbol.
- Target symbol where applicable.
- Relationship type.
- Source location where available.

Relationship identity must be deterministic.

Duplicate relationships must not be created solely because the same relationship is discovered through multiple traversal paths.

---

# Separation from Source Indexing

The Symbol & AST Engine consumes source indexing information.

It must not replace the source index.

Source indexing remains responsible for:

- Source file identity.
- File metadata.
- File hashes.
- Encoding.
- Line endings.
- Package indexing.
- Repository statistics.
- Persistent source indexes.

The AST engine is responsible for interpreting supported source syntax into AST structures and symbols.

---

# Separation from Dependency Analysis

Dependency Analysis remains authoritative for repository dependency relationships.

The Symbol & AST Engine may expose import or type information needed by downstream analysis, but it must not redefine the dependency model.

Dependency semantics must not be duplicated inside the symbol database.

---

# Separation from Cross-Reference Analysis

The Symbol & AST Engine establishes symbol relationships defined within its own responsibility.

It does not provide the complete repository cross-reference system.

The following responsibilities remain outside this capability:

- Complete function reference analysis.
- Complete method reference analysis.
- Complete call graph construction.
- Go-to definition.
- Find references.
- Reverse dependency lookup.
- Change impact analysis.
- Breaking change detection.

These belong to the cross-reference capability.

---

# Separation from Knowledge Graph

The Symbol Relationship Graph is an input to higher-level repository graph construction.

The Symbol & AST Engine must not become the complete repository knowledge graph.

Repository-wide knowledge graph modeling, graph queries, traversal, export, and graph validation belong to the dedicated knowledge graph capability.

---

# Determinism

The engine must produce deterministic results.

For equivalent source content and equivalent parser configuration:

- AST structure must be equivalent.
- Symbol identity must be equivalent.
- Documentation associations must be equivalent.
- Symbol relationships must be equivalent.

Filesystem enumeration order, parser scheduling, or concurrency must not alter the resulting repository model.

---

# Error Isolation

Parsing errors must remain isolated to the affected source artifacts whenever possible.

A syntax error in one source file must not silently invalidate unrelated files.

The engine must preserve diagnostics for failed parsing.

Consumers must be able to distinguish incomplete analysis from successfully parsed repository information.

---

# Generated Source

Generated source files must remain identifiable through established source indexing information.

Generated code must still be parsed when supported and required by the repository analysis contract.

The engine must not assume that generated source is invalid or unimportant.

Generated-code validation must verify that generated artifacts can be handled without compromising the correctness of the AST or symbol database.

---

# Large Files

The engine must support large source files without introducing unnecessary memory or processing overhead.

Large-file handling must preserve AST and symbol correctness.

Large files must not be silently excluded merely because they are larger than ordinary source files.

Performance characteristics must be measured through engineering validation.

---

# Nested Packages

The engine must correctly process repositories containing nested package structures where supported by Go repository organization.

Package identity must remain based on source and established repository structure.

Nested directories must not automatically be treated as separate packages without valid package evidence.

---

# Security

The Symbol & AST Engine is a source-analysis capability.

It must not execute source code during parsing or symbol extraction.

It must not:

- Execute repository binaries.
- Execute build scripts.
- Execute test programs.
- Install dependencies.
- Execute package-manager lifecycle scripts.
- Modify source files.
- Modify repository configuration.

Repository source must be treated as untrusted input.

---

# Performance

The engine must support efficient parsing and symbol extraction across repositories of varying size.

The implementation should make use of:

- Incremental parsing.
- Parallel parsing.
- Existing source indexes.
- Existing content identity information.

Parallel execution must not compromise deterministic output.

Performance optimization must not reduce correctness.

---

# Extensibility

The architecture must allow additional source-language support through additive extensions.

Language-specific parsing must remain separated from common repository capability contracts.

The initial Go implementation must not create unnecessary coupling that prevents future language support.

Additional symbol categories must be introduced through explicit extensions.

Additional relationship types must preserve the semantics of existing relationships.

---

# API Boundary

The Symbol & AST Engine remains an internal repository capability.

Internal AST structures and symbol database implementation details must not automatically become public API.

Stable internal contracts should expose only the information required by consuming repository capabilities.

Public SDK exposure belongs to the established public API and SDK architecture.

---

# Acceptance Criteria

The Symbol & AST Engine is considered complete when it provides:

- Go AST parsing.
- Explicit syntax error handling.
- Comment parsing.
- Formatting metadata preservation.
- Incremental parsing support.
- Parallel parsing support.
- Package symbol extraction.
- Struct symbol extraction.
- Interface symbol extraction.
- Function symbol extraction.
- Method symbol extraction.
- Variable symbol extraction.
- Constant symbol extraction.
- Type symbol extraction.
- Generic symbol extraction.
- Alias symbol extraction.
- Package documentation extraction.
- Struct documentation extraction.
- Function documentation extraction.
- Interface documentation extraction.
- Method documentation extraction.
- TODO comment extraction.
- FIXME comment extraction.
- Function ownership relationships.
- Method receiver relationships.
- Interface implementation relationships.
- Struct embedding relationships.
- Type alias relationships.
- Generic constraint relationships.
- Deterministic AST results.
- Deterministic symbol identity.
- Deterministic documentation associations.
- Deterministic relationship construction.
- Invalid syntax handling.
- Large-file handling.
- Nested package handling.
- Generated-code handling.
- Parsing performance validation.
- No source execution.
- No repository mutation.
- No unnecessary modification of established core architecture.

---

# Architectural Guardrails

Implementation must stop and return to architectural review if any proposal requires:

- Modification of established core architecture merely for convenience.
- Replacement of established source indexing contracts.
- Replacement of established repository identity.
- AI-based AST interpretation.
- AI-based symbol extraction.
- Probabilistic symbol identity.
- Source execution during analysis.
- Build execution during parsing.
- Dependency installation during parsing.
- Silent recovery from invalid syntax.
- Treating textual similarity as authoritative symbol relationships.
- Treating symbol relationships as the complete repository cross-reference graph.
- Mixing knowledge graph responsibilities into the AST layer.
- Unnecessary third-party dependencies.
- Nondeterministic output caused by concurrency or traversal order.

These conditions represent architectural violations rather than implementation details.

---

# Architectural Stability

The Symbol & AST Engine is an additive capability built on established repository indexing infrastructure.

Its responsibility is intentionally limited to:

> Parse supported source code into structured ASTs and derive deterministic symbols, documentation associations, and defined symbol relationships.

The engine must remain focused on source structure.

Higher-level capabilities may consume its outputs for cross-reference analysis, knowledge graph construction, repository APIs, search, and future intelligence.

Those capabilities must not require the AST engine to absorb responsibilities outside its defined boundary.

---

# Authority

This document defines the Symbol & AST Engine capability and its engineering boundaries.

The established Limoxel Core Foundation remains authoritative for:

- Runtime.
- Configuration.
- Core contracts.
- Repository infrastructure.
- Storage.
- Error handling.
- Extension mechanisms.
- Engineering standards.

Source Code Indexing remains authoritative for:

- Source file identity.
- File metadata.
- File hashes.
- Encoding.
- Line endings.
- Package index.
- Repository indexing.

Dependency Analysis remains authoritative for dependency relationships.

The Cross-Reference Engine remains authoritative for complete repository reference analysis and navigation.

The Knowledge Graph capability remains authoritative for repository-wide graph modeling and graph operations.

Where this document conflicts with an established contract, the established contract takes precedence and this document must be revised before implementation proceeds.

---

# Applicability

The principles and contracts defined in this document apply to:

- AST construction.
- Symbol extraction.
- Documentation extraction.
- Symbol relationship construction.
- AST persistence where implemented.
- Symbol database consumers.
- Documentation database consumers.
- Symbol relationship consumers.

All implementations must remain consistent with Limoxel's principles of deterministic behavior, explicit source evidence, separation of responsibilities, minimal coupling, long-term maintainability, and additive extension of established architecture.

---

# Change Policy

The Symbol & AST Engine should evolve through additive capability-layer changes whenever possible.

Changes must preserve:

- Existing core contracts.
- Source indexing contracts.
- Repository identity.
- Deterministic symbol identity.
- AST correctness.
- Documentation associations.
- Symbol relationship semantics.
- Separation from cross-reference analysis.
- Separation from knowledge graph construction.

Additional language support should be introduced through explicit language extensions.

Additional symbol types should be introduced without changing the meaning of existing symbol categories.

Additional relationships should be introduced without changing the semantics of existing relationships.

Changes requiring modification of established core architecture require explicit architectural review before implementation.

---