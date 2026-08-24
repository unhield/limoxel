# Cross-Reference Engine

Project    : Limoxel  
Category   : Repository  
Document   : Cross-Reference Engine  
Version    : 1.0  
Author     : Raj Joshi

---

# Purpose

This document defines the architecture and engineering contract for deterministic cross-reference analysis within Limoxel.

The Cross-Reference Engine transforms repository symbols and structural information into explicit reference relationships, call relationships, navigation information, and change-impact information.

The engine provides deterministic repository relationship analysis.

It must derive relationships from verified repository structure and source information rather than probabilistic interpretation.

---

# Scope

The Cross-Reference Engine is responsible for:

- Function reference analysis.
- Method reference analysis.
- Interface reference analysis.
- Struct reference analysis.
- Constant reference analysis.
- Variable reference analysis.
- Type reference analysis.
- Call graph construction.
- Caller mapping.
- Callee mapping.
- Recursive call detection.
- Dead function identification.
- Entry-point identification.
- Exit-point identification.
- Definition navigation.
- Reference navigation.
- Implementation navigation.
- Package navigation.
- Reverse dependency lookup.
- Change impact analysis.
- Modified-file impact analysis.
- Package impact analysis.
- Symbol impact analysis.
- Dependency impact analysis.
- Breaking-change detection.
- Relationship validation.
- Broken-reference detection.
- Missing-symbol detection.
- Duplicate-symbol detection.
- Invalid-import detection.
- Circular-reference detection.

The resulting capability provides:

- Reference database.
- Call graph.
- Navigation engine.
- Change-impact analyzer.
- Verified cross-reference relationships.

---

# Architectural Position

The Cross-Reference Engine is an additive repository capability.

Its implementation belongs within the repository capability boundary:

    internal/
    └── capabilities/

The engine consumes structured information produced by established repository capabilities.

The conceptual processing relationship is:

    Repository Source
          |
          v
    Source Index
          |
          v
    AST & Symbol Information
          |
          v
    Cross-Reference Analysis
          |
          +-- Reference Database
          |
          +-- Call Graph
          |
          +-- Navigation Engine
          |
          +-- Change Impact Analyzer
          |
          v
    Verified Repository Relationships

The Cross-Reference Engine must extend established repository infrastructure rather than replace or redesign it.

---

# Relationship Analysis

The engine determines how repository symbols reference and depend upon one another.

Reference analysis must operate on structured repository information.

Textual name matching alone must not be treated as authoritative evidence of a reference.

Every recorded relationship should have sufficient source evidence to explain why the relationship exists.

---

# Reference Database

The Reference Database stores deterministic relationships between repository symbols.

The database must support references involving:

- Functions.
- Methods.
- Interfaces.
- Structs.
- Constants.
- Variables.
- Types.

Each reference must preserve enough information to identify:

- Referencing symbol.
- Referenced symbol.
- Relationship type.
- Source file.
- Source location where available.

Reference identity must be deterministic.

---

# Function References

Function references represent source-level references from one function or supported repository construct to another function.

The engine must distinguish an actual function reference from:

- A textual name match.
- A comment mention.
- A documentation mention.
- An unrelated identifier with the same name.

Reference resolution must use available structural and symbol information.

---

# Method References

Method references must represent references to methods using available receiver and method information.

Method identity must not depend solely on the method name.

Receiver type information must be considered where required to distinguish methods with identical names.

---

# Interface References

Interface references must represent relationships involving interface symbols and their usage within repository source.

The engine must preserve the distinction between:

- A reference to an interface type.
- A method associated with an interface.
- An implementation relationship.
- A textual mention of an interface name.

Interface implementation relationships originate from the symbol relationship model and must not be confused with ordinary references.

---

# Struct References

Struct references must represent source-level usage of struct types or declarations.

The engine must distinguish:

- Struct type references.
- Struct declarations.
- Struct embedding.
- Field access where deterministically resolvable.

Struct embedding relationships originate from the symbol relationship model and must remain semantically distinct from ordinary references.

---

# Constant References

References to declared constants must be represented in the Reference Database.

Constant identity must be resolved through established symbol information.

A constant name appearing in unrelated text must not be recorded as a reference.

---

# Variable References

References to declared variables must be represented where deterministically resolvable.

Variable identity must remain associated with its declaration context.

A variable name alone must not be treated as globally unique.

---

# Type References

Type references must represent source-level references to declared types.

The engine must preserve the distinction between:

- Type declarations.
- Type references.
- Type aliases.
- Interface types.
- Struct types.
- Generic types.

Type resolution must use established symbol identity.

---

# Reference Identity

Reference identity must be deterministic.

The same repository state must produce the same reference relationships.

A reference must identify the source and target symbols explicitly.

Where multiple syntactic references connect the same symbols, each source occurrence may remain independently represented when source-level detail is required.

Aggregated relationship views must not lose the underlying source evidence.

---

# Call Graph

The Call Graph represents deterministic function and method invocation relationships.

The graph must support:

- Caller mapping.
- Callee mapping.
- Recursive call detection.
- Entry-point identification.
- Exit-point identification.
- Dead-function analysis.

The Call Graph must remain separate from the general Reference Database.

A function can reference another symbol without necessarily creating a function call relationship.

---

# Caller Mapping

Caller mapping identifies symbols that invoke a given function or method.

Each caller relationship must be based on resolvable source evidence.

Caller mappings must preserve the source relationship between the calling symbol and the invoked symbol.

---

# Callee Mapping

Callee mapping identifies functions or methods invoked by a given symbol.

The mapping must distinguish direct invocation relationships from unrelated symbol references.

---

# Recursive Calls

The engine must detect recursive call relationships.

Recursive relationships may include:

- Direct recursion.
- Mutual recursion.

Recursive detection must be based on the Call Graph rather than textual heuristics.

---

# Entry Points

The engine should identify repository entry points where they can be determined from supported language and repository structure.

Entry-point detection must use explicit structural evidence.

The engine must not label arbitrary functions as entry points merely because they appear to be externally accessible.

---

# Exit Points

The engine should identify relevant terminal or externally observable call paths where deterministically supported.

Exit-point semantics must be explicitly defined by the implementation contract.

The engine must not infer business-level meaning from function names.

---

# Dead Functions

Dead-function analysis identifies functions that have no reachable callers from known repository entry points or other defined roots.

Dead-function results are analytical findings rather than source modifications.

The engine must not delete or modify functions based on dead-function analysis.

Dead-function analysis must clearly distinguish:

- Confirmed unreachable functions.
- Functions whose reachability cannot be determined.

Unknown reachability must not be classified as dead.

---

# Navigation Engine

The Navigation Engine provides deterministic navigation relationships over repository symbols.

The defined navigation capabilities include:

- Go-to definition.
- Find references.
- Find implementations.
- Package navigation.
- Reverse dependency lookup.

Navigation must consume verified repository relationships.

Navigation results must not be generated through semantic guessing.

---

# Go-to Definition

Go-to-definition resolves a reference to its corresponding symbol declaration.

The result must identify the authoritative declaration.

If multiple possible declarations exist and the engine cannot deterministically resolve the reference, the engine must report ambiguity rather than selecting an arbitrary result.

---

# Find References

Find-references returns source locations that reference a selected symbol.

Results must originate from the Reference Database.

The engine must not include comments, documentation, or unrelated textual matches unless they are explicitly represented as separate searchable artifacts.

---

# Find Implementations

Find-implementations resolves implementation relationships for supported interfaces and applicable types.

Implementation results must be based on deterministic structural analysis.

Textual naming similarity is insufficient evidence.

---

# Package Navigation

Package navigation provides deterministic traversal between package-level repository entities.

Package navigation must rely on established package identity and repository structure.

The engine must not independently redefine package identity.

---

# Reverse Dependency Lookup

Reverse dependency lookup identifies repository entities that depend upon a selected entity.

The result must be derived from verified repository relationships.

Reverse dependency analysis must distinguish direct relationships from transitive relationships.

---

# Change Impact Analysis

The Change Impact Analyzer determines the repository entities potentially affected by a source change.

Impact analysis must remain deterministic.

The engine should distinguish between:

- Direct impact.
- Transitive impact.
- Structural impact.
- Dependency impact.

Impact analysis predicts repository consequences.

It must not modify source code.

---

# Modified File Impact

A modified file must be analyzed against known repository relationships.

The engine must identify directly related:

- Files.
- Packages.
- Symbols.
- Dependencies.

The analysis must preserve the distinction between confirmed relationships and unresolved relationships.

---

# Package Impact

Package impact analysis identifies packages affected by changes within a package or its symbols.

The analysis must consider established dependency and reference relationships.

Package impact must not be inferred solely from directory proximity.

---

# Symbol Impact

Symbol impact analysis identifies symbols directly or transitively affected by a changed symbol.

Impact relationships must originate from the Reference Database, Call Graph, symbol relationships, and applicable dependency information.

---

# Dependency Impact

Dependency impact analysis identifies downstream repository entities affected by a change in an upstream dependency.

The engine must distinguish:

- Direct dependency impact.
- Transitive dependency impact.

Dependency relationships must remain consistent with the established Dependency Analysis capability.

---

# Breaking Change Detection

The engine must identify structurally detectable breaking changes where sufficient repository information exists.

Examples may include:

- Removed symbols with known references.
- Changed declarations that invalidate known references.
- Removed methods required by known implementations.
- Changed type relationships that invalidate known consumers.

The engine must not claim a breaking change when available evidence is insufficient.

A result should distinguish:

- Confirmed breaking change.
- Potential breaking change.
- Unable to determine.

---

# Relationship Validation

The engine must validate cross-reference integrity.

Validation must identify:

- Broken references.
- Missing symbols.
- Duplicate symbols.
- Invalid imports.
- Circular references.

Validation must operate on structured repository data.

---

# Broken References

A broken reference occurs when a recorded reference cannot be resolved to a valid target symbol.

Broken references must retain sufficient information to identify:

- Referencing source.
- Expected target.
- Source location.
- Failure reason where available.

---

# Missing Symbols

The engine must detect references to symbols that cannot be resolved within the applicable repository scope.

Missing symbols must not automatically be classified as errors when the reference legitimately resolves outside the repository boundary.

External dependencies must remain distinguishable from unresolved internal symbols.

---

# Duplicate Symbols

Duplicate symbol detection must identify cases where repository identity rules indicate that two declarations incorrectly occupy the same symbol identity.

The engine must respect language scope and package boundaries.

Symbols with identical names in valid independent scopes must not automatically be classified as duplicates.

---

# Invalid Imports

Invalid import relationships must be detected using established dependency and package information.

The engine must distinguish:

- Missing external dependencies.
- Invalid internal imports.
- Unresolved imports.
- Structurally invalid import relationships.

---

# Circular References

The engine must identify circular relationships where the applicable relationship graph permits such analysis.

Circularity must be reported with the actual relationship path.

The engine must distinguish legitimate graph cycles from invalid cycles according to the semantics of the analyzed relationship type.

The presence of a cycle alone must not automatically be classified as a defect unless the applicable repository rule prohibits it.

---

# Relationship Evidence

Every significant cross-reference relationship should preserve evidence sufficient for deterministic verification.

Evidence may include:

- Source file.
- Source position.
- Source symbol.
- Target symbol.
- Relationship type.
- Resolution state.

The engine must not create opaque relationships whose origin cannot be verified.

---

# Determinism

The Cross-Reference Engine must produce deterministic results.

For equivalent repository source and equivalent analysis configuration:

- Reference relationships must be equivalent.
- Call graph relationships must be equivalent.
- Navigation results must be equivalent.
- Impact results must be equivalent.
- Validation results must be equivalent.

Filesystem ordering, traversal order, or concurrency must not alter semantic results.

---

# Ambiguity Handling

The engine must not resolve ambiguous relationships through arbitrary selection.

Where multiple valid candidates exist and deterministic resolution is unavailable, the engine must preserve the ambiguity.

An unresolved relationship must be distinguishable from a confirmed relationship.

This is essential for trustworthy repository intelligence.

---

# Unknown State

The engine must support an explicit unknown or unresolved state where repository evidence is insufficient.

Unknown must not be converted into:

- False.
- True.
- Dead.
- Broken.
- Safe.
- Breaking.

This prevents incomplete analysis from becoming false repository knowledge.

---

# Separation from AST and Symbol Analysis

The Cross-Reference Engine consumes AST and symbol information.

It must not replace the AST engine or symbol database.

The following responsibilities remain owned by the Symbol & AST Engine:

- AST construction.
- Symbol extraction.
- Documentation extraction.
- Function ownership.
- Method receiver relationships.
- Struct embedding.
- Type aliases.
- Generic constraints.

The Cross-Reference Engine consumes those relationships when required for reference and impact analysis.

---

# Separation from Dependency Analysis

Dependency Analysis remains authoritative for repository dependency relationships.

The Cross-Reference Engine may consume dependency information for:

- Reverse dependency lookup.
- Dependency impact.
- Import validation.
- Change impact analysis.

It must not create a competing dependency model.

---

# Separation from Knowledge Graph

The Cross-Reference Engine produces verified relationships that may be consumed by the repository knowledge graph.

It must not become the repository-wide knowledge graph.

The following responsibilities remain outside this capability:

- Unified repository node modeling.
- Graph export.
- General graph queries.
- Graph filtering.
- Graph serialization.
- Repository-wide graph validation.

---

# Separation from Search

The Cross-Reference Engine provides relationship data.

It does not define the complete repository search engine.

Search capabilities may consume reference and navigation information but remain independently responsible for:

- Search indexing.
- Query interpretation.
- Result ranking.
- Search filtering.

Search ranking must not modify the underlying deterministic relationship model.

---

# Security

The Cross-Reference Engine must treat repository source as untrusted input.

Analysis must not execute source code.

It must not:

- Execute repository binaries.
- Execute build scripts.
- Execute tests.
- Install dependencies.
- Execute package-manager lifecycle scripts.
- Modify source files.
- Modify repository configuration.

Cross-reference analysis must remain observational.

---

# Performance

The engine must support efficient relationship analysis across repositories of varying size.

Performance-sensitive operations should use:

- Existing symbol indexes.
- Existing source indexes.
- Efficient relationship indexes.
- Incremental analysis where supported.
- Appropriate graph traversal strategies.

Performance optimization must not compromise deterministic correctness.

---

# Incremental Analysis

Where supported, changes should be analyzed incrementally.

Unchanged relationships should not require unnecessary reconstruction.

Incremental results must be equivalent to analysis performed from the current complete repository state.

Stale relationships must never be presented as current repository truth.

---

# Extensibility

The architecture must permit additional relationship types through additive extensions.

New language support must be able to provide language-specific reference resolution without changing unrelated repository contracts.

New relationship types must:

- Have explicit semantics.
- Have deterministic resolution rules.
- Preserve source evidence.
- Remain distinguishable from existing relationship types.

---

# Internal API Boundary

The Cross-Reference Engine remains an internal repository capability.

Its internal implementation structures must not automatically become public API.

Stable internal contracts should expose only information required by consuming repository capabilities.

Public SDK exposure belongs to the established API and SDK architecture.

---

# Acceptance Criteria

The Cross-Reference Engine is considered complete when it provides:

- Function reference analysis.
- Method reference analysis.
- Interface reference analysis.
- Struct reference analysis.
- Constant reference analysis.
- Variable reference analysis.
- Type reference analysis.
- Caller mapping.
- Callee mapping.
- Recursive call detection.
- Entry-point identification.
- Exit-point identification.
- Dead-function analysis.
- Go-to-definition navigation.
- Find-references navigation.
- Find-implementations navigation.
- Package navigation.
- Reverse dependency lookup.
- Modified-file impact analysis.
- Package impact analysis.
- Symbol impact analysis.
- Dependency impact analysis.
- Breaking-change detection.
- Broken-reference detection.
- Missing-symbol detection.
- Duplicate-symbol detection.
- Invalid-import detection.
- Circular-reference detection.
- Deterministic relationship resolution.
- Explicit ambiguity handling.
- Explicit unknown-state handling.
- No source execution.
- No repository mutation.
- No competing dependency model.
- No unnecessary modification of established core architecture.

---

# Architectural Guardrails

Implementation must stop and return to architectural review if any proposal requires:

- Modification of established core architecture merely for convenience.
- Replacement of the established symbol model.
- Replacement of the established dependency model.
- AI-based reference resolution.
- Probabilistic symbol resolution.
- Arbitrary selection among ambiguous symbols.
- Treating textual similarity as authoritative reference evidence.
- Treating unresolved relationships as confirmed relationships.
- Executing repository source during analysis.
- Executing build or test processes during analysis.
- Mixing repository-wide knowledge graph responsibilities into the cross-reference layer.
- Mixing search-ranking logic into the relationship model.
- Unnecessary third-party dependencies.
- Nondeterministic graph results.
- Silent modification of repository source.

These conditions represent architectural violations rather than implementation details.

---

# Architectural Stability

The Cross-Reference Engine is an additive capability built on established repository analysis infrastructure.

Its responsibility is intentionally limited to:

> Determine, validate, and expose deterministic relationships between repository symbols and analyze their navigational and change-impact consequences.

The engine must remain focused on relationships.

Higher-level capabilities may consume its outputs for knowledge graph construction, repository APIs, search, developer experience, SDK functionality, and future intelligence.

Those capabilities must not require the Cross-Reference Engine to absorb responsibilities outside its defined boundary.

---

# Authority

This document defines the Cross-Reference Engine capability and its engineering boundaries.

The established Limoxel Core Foundation remains authoritative for:

- Runtime.
- Configuration.
- Core contracts.
- Repository infrastructure.
- Storage.
- Error handling.
- Extension mechanisms.
- Engineering standards.

Source Code Indexing remains authoritative for source artifact identity and source indexing information.

Dependency Analysis remains authoritative for repository dependency relationships.

The Symbol & AST Engine remains authoritative for AST construction, symbol extraction, documentation extraction, and structural symbol relationships.

The Knowledge Graph capability remains authoritative for repository-wide graph modeling and graph operations.

Where this document conflicts with an established contract, the established contract takes precedence and this document must be revised before implementation proceeds.

---

# Applicability

The principles and contracts defined in this document apply to:

- Reference analysis.
- Call graph construction.
- Navigation.
- Change impact analysis.
- Relationship validation.
- Reference database consumers.
- Call graph consumers.
- Navigation consumers.
- Change-impact consumers.

All implementations must remain consistent with Limoxel's principles of deterministic behavior, explicit source evidence, separation of responsibilities, minimal coupling, long-term maintainability, and additive extension of established architecture.

---

# Change Policy

The Cross-Reference Engine should evolve through additive capability-layer changes whenever possible.

Changes must preserve:

- Existing core contracts.
- Source indexing contracts.
- Symbol identity.
- Dependency semantics.
- Reference semantics.
- Call graph semantics.
- Navigation semantics.
- Impact-analysis semantics.
- Deterministic resolution.
- Explicit ambiguity handling.
- Separation from knowledge graph construction.
- Separation from search.

Additional relationship types should be introduced without changing the semantics of existing relationship types.

Changes requiring modification of established core architecture require explicit architectural review before implementation.

---