# Knowledge Graph

Project    : Limoxel  
Category   : Repository  
Document   : Knowledge Graph  
Version    : 1.0  
Author     : Raj Joshi

---

# Purpose

This document defines the architecture and engineering contract for the deterministic engineering knowledge graph within Limoxel.

The Knowledge Graph unifies repository entities and their verified relationships into a single structured graph.

The graph provides a common representation through which downstream repository capabilities can query, traverse, validate, and export engineering knowledge.

The Knowledge Graph is deterministic infrastructure.

It does not generate repository facts through AI or probabilistic reasoning.

---

# Scope

The Knowledge Graph is responsible for:

- Repository node modeling.
- Module node modeling.
- Package node modeling.
- File node modeling.
- Symbol node modeling.
- Documentation node modeling.
- Configuration node modeling.
- Relationship construction.
- Repository graph construction.
- Graph queries.
- Node lookup.
- Relationship lookup.
- Path traversal.
- Reverse traversal.
- Neighbor search.
- Graph filtering.
- Graph export.
- JSON export.
- DOT export.
- GraphML export.
- Mermaid export.
- Internal API export.
- Graph validation.
- Missing-node detection.
- Invalid-edge detection.
- Duplicate-edge detection.
- Orphan-node detection.
- Graph performance validation.

The capability produces:

- Knowledge graph schema.
- Repository knowledge graph.
- Graph query engine.
- Graph export engine.
- Validated repository graph.

---

# Architectural Position

The Knowledge Graph is an additive repository capability.

Its implementation belongs within the repository capability boundary:

    internal/
    └── capabilities/

The graph consumes established repository information produced by upstream capabilities.

The conceptual processing relationship is:

    Repository Discovery
            |
            v
    Project Structure
            |
            v
    Dependency Analysis
            |
            v
    Source Code Index
            |
            v
    AST & Symbol Engine
            |
            v
    Cross-Reference Engine
            |
            v
    Knowledge Graph
            |
            +-- Knowledge Model
            +-- Relationship Builder
            +-- Query Engine
            +-- Export Engine
            +-- Validation Engine

The Knowledge Graph must extend established repository capabilities rather than replace them.

---

# Knowledge Model

The Knowledge Model defines the entities represented within the engineering knowledge graph.

The initial node categories are:

- Repository.
- Module.
- Package.
- File.
- Symbol.
- Documentation.
- Configuration.

Each node must have a deterministic identity.

Node identity must remain stable for equivalent repository state.

---

# Repository Nodes

A repository node represents the analyzed repository.

The repository node must use the established repository identity.

It may reference repository metadata already established by the repository metadata capability.

The graph must not create an independent repository identity system.

---

# Module Nodes

A module node represents a detected repository module.

Module identity must reuse the established module representation.

Module nodes may contain relationships to:

- Repository.
- Packages.
- Files.
- Documentation.
- Configuration.
- Build-system information.

The Knowledge Graph must not redefine module detection.

---

# Package Nodes

A package node represents an established repository package.

Package identity must reuse the established package model.

Package nodes may participate in:

- Containment.
- Import relationships.
- Dependency relationships.
- Symbol relationships.
- Documentation relationships.

Package nodes must not be created solely from directory names when authoritative package information is available.

---

# File Nodes

A file node represents an indexed repository file.

File identity must reuse the established source index and repository-relative path semantics.

File nodes may represent:

- Source files.
- Test files.
- Generated files.
- Configuration files.
- Documentation files.

File classification must remain consistent with upstream indexing capabilities.

---

# Symbol Nodes

A symbol node represents a symbol extracted by the AST and Symbol Engine.

The graph must preserve established symbol identity.

Symbol nodes may represent:

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

The Knowledge Graph must not independently extract symbols.

---

# Documentation Nodes

A documentation node represents documentation discovered by repository capabilities.

Documentation nodes may represent:

- README files.
- Package documentation.
- Symbol documentation.
- Engineering documentation.
- Other supported documentation artifacts.

Documentation identity must remain associated with its authoritative source artifact.

The graph must not generate documentation content.

---

# Configuration Nodes

A configuration node represents a configuration artifact discovered by repository capabilities.

Configuration nodes may represent supported:

- YAML.
- JSON.
- TOML.
- ENV.
- INI.
- Properties.
- XML.

Sensitive configuration contents must not be embedded into graph metadata merely to establish configuration identity.

---

# Node Identity

Every graph node must have a deterministic identity.

Identity must distinguish nodes with identical names but different repository contexts.

A display name must never be treated as a globally unique graph identity.

Node identity should preserve the relationship between:

- Entity type.
- Repository.
- Module where applicable.
- Package where applicable.
- File where applicable.
- Symbol scope where applicable.

---

# Node Metadata

Node metadata must contain only information required to identify and describe the represented engineering entity.

Metadata should reuse authoritative upstream models.

The Knowledge Graph must not duplicate complete source indexes, dependency manifests, ASTs, or documentation databases inside every node.

Graph metadata should remain concise and queryable.

---

# Relationship Builder

The Relationship Builder connects engineering entities using relationships established by upstream repository capabilities.

The initial relationship categories are:

- Contains.
- Imports.
- Implements.
- Calls.
- References.
- Depends on.
- Documents.
- Configures.

Each relationship must have:

- Source node.
- Target node.
- Relationship type.

Relationships must have deterministic identity.

---

# Contains

The `Contains` relationship represents structural containment.

Examples include:

    Repository -> Module
    Module -> Package
    Package -> File
    File -> Symbol

Containment must represent actual repository structure.

Containment must not be interpreted as dependency.

---

# Imports

The `Imports` relationship represents an established import relationship.

The relationship must originate from verified source or package analysis.

Textual name similarity is insufficient evidence.

Import relationships must preserve direction.

---

# Implements

The `Implements` relationship represents a verified implementation relationship.

It may connect:

- Concrete types to interfaces.
- Other supported implementation entities.

Implementation relationships must originate from deterministic language and symbol analysis.

---

# Calls

The `Calls` relationship represents an established function or method invocation relationship.

It must originate from the Cross-Reference Engine.

The Knowledge Graph must not independently reconstruct the call graph.

---

# References

The `References` relationship represents a verified symbol reference.

It must originate from the Cross-Reference Engine or other authoritative relationship provider.

The graph must preserve the relationship's source and target identities.

---

# Depends On

The `Depends On` relationship represents an established dependency relationship.

It must reuse Dependency Analysis.

The Knowledge Graph must not create a competing dependency model.

Direct and indirect dependency semantics must remain available where supplied by the authoritative dependency model.

---

# Documents

The `Documents` relationship connects documentation to the entity it documents.

Examples include:

    Documentation -> Module
    Documentation -> Package
    Documentation -> Symbol

The relationship must be based on deterministic documentation association.

Documentation content must not be inferred merely from filename similarity.

---

# Configures

The `Configures` relationship connects configuration artifacts to the engineering entity they configure where that relationship can be established deterministically.

Examples may include:

    Configuration -> Module
    Configuration -> Package
    Configuration -> Build System

A configuration file must not be linked to an entity merely because it exists nearby in the filesystem.

---

# Relationship Identity

Relationship identity must be deterministic.

Equivalent repository state must produce equivalent relationship identities.

Duplicate relationships must not be created because the same relationship was discovered through multiple upstream capabilities.

If multiple evidence sources confirm the same relationship, the graph should preserve the relationship once while retaining provenance where required.

---

# Relationship Provenance

Where practical, relationships should retain provenance information identifying the authoritative source capability.

Possible provenance sources include:

- Repository Discovery.
- Project Structure.
- Dependency Analysis.
- Source Code Indexing.
- AST & Symbol Engine.
- Cross-Reference Engine.

Provenance allows graph consumers to distinguish verified relationships from derived graph views.

---

# Graph Construction

The Repository Knowledge Graph is constructed from established nodes and relationships.

Graph construction must not require re-parsing the repository when authoritative upstream data already exists.

The graph builder must consume stable capability outputs.

Graph construction must remain deterministic.

---

# Graph Consistency

The graph must maintain consistency between:

- Node identity.
- Relationship identity.
- Repository structure.
- Symbol identity.
- Dependency identity.

A relationship referencing a node that does not exist in the graph must be considered invalid.

---

# Graph Query Engine

The Graph Query Engine provides deterministic access to graph information.

It must support:

- Node lookup.
- Relationship lookup.
- Path traversal.
- Reverse traversal.
- Neighbor search.
- Graph filtering.

Queries must operate against the graph model without modifying graph state.

---

# Node Lookup

Node lookup must support deterministic lookup by stable graph identity.

Where supported, lookup may additionally use:

- Entity type.
- Repository context.
- Module context.
- Package context.
- Name.

Name-only lookup must not silently return an arbitrary node when multiple nodes match.

---

# Relationship Lookup

Relationship lookup must support queries based on:

- Source node.
- Target node.
- Relationship type.
- Source and target node types.

Results must be deterministically ordered.

---

# Path Traversal

Path traversal must allow deterministic traversal between graph nodes.

Traversal must define:

- Starting node.
- Direction.
- Relationship types.
- Maximum depth where applicable.

Traversal must protect against infinite loops.

Cycles must remain representable without causing unbounded traversal.

---

# Reverse Traversal

Reverse traversal must allow consumers to discover nodes that point toward a selected node.

Examples include:

- Reverse dependencies.
- Callers.
- Referencing symbols.
- Implementations.
- Documentation relationships.

Reverse traversal must preserve relationship semantics.

---

# Neighbor Search

Neighbor search returns nodes directly connected to a selected node.

The query must allow filtering by relationship type where supported.

Neighbor results must remain deterministic.

---

# Graph Filtering

Graph filtering allows consumers to restrict queries by supported criteria.

Possible criteria include:

- Node type.
- Relationship type.
- Repository.
- Module.
- Package.
- Path.
- Depth.

Filtering must operate on graph data rather than altering the graph itself.

---

# Query Safety

Graph queries must be read-only.

Queries must not:

- Modify nodes.
- Modify relationships.
- Modify source files.
- Modify repository state.
- Trigger source execution.
- Trigger build execution.

Expensive graph queries must have explicit traversal boundaries where necessary.

---

# Graph Export

The Graph Export Engine provides deterministic external representations of the repository graph.

The initial export formats are:

- JSON.
- DOT.
- GraphML.
- Mermaid.
- Internal API representation.

Export must preserve graph semantics.

---

# JSON Export

JSON export must represent:

- Nodes.
- Relationships.
- Node identity.
- Relationship identity.
- Relevant metadata.

The serialized structure must be deterministic.

Object and collection ordering must not depend on map iteration order.

---

# DOT Export

DOT export must represent the graph in a deterministic form suitable for graph tooling.

Node and edge identity must remain stable.

Graph semantics must not be altered to satisfy visualization-specific conventions.

---

# GraphML Export

GraphML export must preserve:

- Node identity.
- Node type.
- Relationship identity.
- Relationship type.
- Required graph metadata.

Export must remain deterministic.

---

# Mermaid Export

Mermaid export must provide a deterministic representation suitable for supported Mermaid graph syntax.

The exporter must preserve relationship direction and entity identity.

The graph model must remain authoritative over visualization syntax.

---

# Internal API Export

Internal API export provides structured access to graph information for Limoxel consumers.

The internal representation must remain consistent with the graph model.

Internal API export must not expose mutable graph internals that allow consumers to corrupt graph state.

---

# Graph Validation

The Graph Validation Engine verifies repository graph integrity.

Validation must identify:

- Missing nodes.
- Invalid edges.
- Duplicate edges.
- Orphan nodes.
- Performance issues detectable through defined graph benchmarks.

Validation must be deterministic.

---

# Missing Nodes

A missing node occurs when a graph relationship references an entity that is not represented by a corresponding graph node.

The validator must identify:

- Relationship.
- Missing target or source.
- Relationship type.

Missing nodes must not be silently created merely to suppress validation errors.

---

# Invalid Edges

An invalid edge is a relationship that violates the defined graph model.

Examples include:

- Unsupported source node type.
- Unsupported target node type.
- Invalid relationship type.
- Invalid relationship direction.
- Missing required relationship metadata.

Invalid edges must be reported explicitly.

---

# Duplicate Edges

Duplicate edges occur when equivalent relationship identities are represented more than once.

Duplicate detection must use the defined relationship identity semantics.

Two relationships with different valid provenance may still represent the same graph relationship and must not automatically become duplicate graph edges.

---

# Orphan Nodes

An orphan node is a graph node with no applicable relationships.

Orphan detection must be interpreted according to node type.

Some nodes may legitimately have no relationships.

Therefore, orphan status must not automatically imply an engineering defect.

---

# Graph Performance Validation

Graph performance validation must measure defined graph operations against representative graph sizes.

The validation model must consider:

- Node lookup.
- Relationship lookup.
- Neighbor search.
- Path traversal.
- Reverse traversal.
- Filtering.
- Export where applicable.

Performance benchmarks must remain separate from graph semantics.

---

# Determinism

The Knowledge Graph must produce deterministic results.

For equivalent upstream repository information:

- Node identities must be equivalent.
- Relationship identities must be equivalent.
- Query results must be equivalent.
- Export results must be equivalent.
- Validation results must be equivalent.

Traversal order, map iteration order, concurrency, or export implementation details must not alter graph semantics.

---

# Graph Immutability

The graph representation exposed to consumers should be immutable or protected through controlled interfaces.

Consumers must not directly mutate graph nodes or relationships.

Graph modifications, if ever required, must occur through controlled internal construction or update mechanisms.

---

# Incremental Graph Updates

Where supported, the graph may be updated incrementally from changed upstream repository information.

Incremental updates must produce results equivalent to reconstructing the graph from the complete current repository state.

Stale nodes and relationships must be removed or invalidated deterministically.

A stale graph must never be presented as current repository truth.

---

# Graph Versioning

Persistent or externally serialized graph representations must contain sufficient schema information to determine compatibility.

Graph schema versioning must remain separate from repository versioning.

An incompatible graph schema must not be silently interpreted as a current graph.

---

# Security

The Knowledge Graph is a read-only repository intelligence structure.

It must not:

- Execute source code.
- Execute build systems.
- Install dependencies.
- Modify repository files.
- Modify Git state.
- Modify source indexes.
- Modify ASTs.
- Modify dependency manifests.

Graph exports must avoid exposing sensitive configuration values.

Node metadata must contain only information appropriate for graph consumers.

---

# Performance

The graph must support efficient operation over large repositories.

Implementation should use appropriate data structures for:

- Node lookup.
- Relationship lookup.
- Traversal.
- Filtering.

Repeated reconstruction of upstream information must be avoided.

Performance optimizations must not compromise deterministic semantics.

---

# Large Graph Handling

The graph must remain suitable for repositories with:

- Large file counts.
- Large package counts.
- Large symbol counts.
- Large dependency graphs.
- Large relationship counts.

Traversal operations must have explicit safeguards.

Exports must avoid uncontrolled memory growth where practical.

---

# API Boundary

The Knowledge Graph remains an internal repository capability.

The graph query engine may provide stable internal interfaces to later repository capabilities.

Public SDK exposure belongs to the established API and SDK architecture.

Graph internals must not become public API merely because they are queryable internally.

---

# Extensibility

The graph model must support additional node and relationship types through additive extensions.

New node types must define:

- Identity.
- Metadata.
- Valid relationships.

New relationship types must define:

- Source types.
- Target types.
- Direction.
- Identity.
- Validation rules.

Adding a new entity type must not invalidate existing graph semantics.

---

# Separation from Upstream Capabilities

The Knowledge Graph consumes authoritative outputs from upstream capabilities.

It must not replace:

- Repository Discovery.
- Project Structure.
- Dependency Analysis.
- Source Code Indexing.
- AST & Symbol Engine.
- Cross-Reference Engine.

Each upstream capability remains authoritative for the facts it establishes.

The Knowledge Graph provides a unified representation of those facts and relationships.

---

# Separation from Search

The Graph Query Engine is not the repository Search Engine.

Graph queries operate on graph entities and relationships.

Search operates on searchable repository artifacts and query semantics.

The graph must not absorb:

- Search ranking.
- Fuzzy search.
- Full-text indexing.
- Search query interpretation.

---

# Separation from Intelligence

The Knowledge Graph is deterministic infrastructure.

It does not:

- Generate semantic interpretations.
- Infer business intent.
- Use AI reasoning.
- Generate natural-language repository explanations.

Future intelligence capabilities may consume the graph.

The graph itself remains deterministic.

---

# Acceptance Criteria

The Knowledge Graph is considered complete when it provides:

- Repository nodes.
- Module nodes.
- Package nodes.
- File nodes.
- Symbol nodes.
- Documentation nodes.
- Configuration nodes.
- Deterministic node identity.
- Contains relationships.
- Imports relationships.
- Implements relationships.
- Calls relationships.
- References relationships.
- Depends-on relationships.
- Documents relationships.
- Configures relationships.
- Repository knowledge graph.
- Node lookup.
- Relationship lookup.
- Path traversal.
- Reverse traversal.
- Neighbor search.
- Graph filtering.
- JSON export.
- DOT export.
- GraphML export.
- Mermaid export.
- Internal API export.
- Missing-node validation.
- Invalid-edge validation.
- Duplicate-edge validation.
- Orphan-node validation.
- Graph performance validation.
- Deterministic query results.
- Deterministic exports.
- Read-only operation.
- No repository mutation.
- No competing upstream models.
- No unnecessary modification of established core architecture.

---

# Architectural Guardrails

Implementation must stop and return to architectural review if any proposal requires:

- Modification of established core architecture merely for convenience.
- Replacement of an upstream authoritative model.
- Duplicate symbol extraction.
- Duplicate dependency analysis.
- Duplicate cross-reference analysis.
- AI-generated graph facts.
- Probabilistic node identity.
- Arbitrary relationship inference.
- Graph mutation by consumers.
- Source execution.
- Build execution.
- Dependency installation.
- Silent graph corruption.
- Unbounded graph traversal.
- Uncontrolled export memory growth.
- Unjustified third-party dependencies.

These conditions represent architectural violations rather than implementation details.

---

# Architectural Stability

The Knowledge Graph is an additive repository capability.

Its responsibility is intentionally limited to:

> Unify deterministic repository entities and verified relationships into a queryable, exportable, and validated engineering knowledge graph.

The graph must remain a consumer of authoritative repository capabilities.

It must not become a replacement for the systems that establish repository facts.

Future repository APIs, search, developer experience, SDK, plugins, enterprise capabilities, and AI intelligence may consume the graph without requiring changes to its fundamental model.

---

# Authority

This document defines the Knowledge Graph capability and its engineering boundaries.

The established Limoxel Core Foundation remains authoritative for:

- Runtime.
- Configuration.
- Core contracts.
- Repository infrastructure.
- Storage.
- Error handling.
- Extension mechanisms.
- Engineering standards.

Repository Discovery remains authoritative for repository identity and discovery.

Project Structure remains authoritative for structural repository information.

Dependency Analysis remains authoritative for dependency relationships.

Source Code Indexing remains authoritative for indexed source artifacts.

The Symbol & AST Engine remains authoritative for ASTs and symbol relationships.

The Cross-Reference Engine remains authoritative for references, calls, navigation, and impact relationships.

Where this document conflicts with an established capability contract, the established contract takes precedence and this document must be revised before implementation proceeds.

---

# Applicability

The principles and contracts defined in this document apply to:

- Knowledge model construction.
- Relationship construction.
- Graph queries.
- Graph traversal.
- Graph filtering.
- Graph exports.
- Graph validation.
- Graph consumers.

All implementations must remain consistent with Limoxel's principles of deterministic behavior, explicit evidence, separation of responsibilities, minimal coupling, long-term maintainability, and additive extension of established architecture.

---

# Change Policy

The Knowledge Graph should evolve through additive capability-layer changes whenever possible.

Changes must preserve:

- Existing core contracts.
- Upstream capability contracts.
- Node identity semantics.
- Relationship identity semantics.
- Graph query semantics.
- Export semantics.
- Validation semantics.
- Deterministic behavior.
- Read-only graph consumption.
- Separation from search and intelligence.

Additional node and relationship types should be introduced through explicit graph-model extensions.

Changes requiring modification of established core architecture require explicit architectural review before implementation.

---