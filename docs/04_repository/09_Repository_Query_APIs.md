# Repository Query APIs

Project  : Limoxel
Category : Repository
Document : Repository Query APIs
Version  : 1.0
Author   : Raj Joshi

---

# Purpose

This document defines the architecture and engineering contract for the internal repository query APIs of Limoxel.

The API layer provides stable interfaces through which repository capabilities can be consumed by future engineering intelligence, SDK, CLI, plugin, and IDE integrations.

The API layer exposes established repository information without replacing or duplicating the systems responsible for producing that information.

The API layer is deterministic infrastructure.

---

# Scope

The Repository Query API capability is responsible for:

- Repository loading API.
- Repository metadata API.
- Repository statistics API.
- Repository lifecycle management.
- Repository error handling.
- Symbol lookup.
- Symbol listing.
- Symbol lookup by type.
- Symbol lookup by package.
- Symbol lookup by name.
- Graph querying.
- Graph traversal.
- Dependency lookup.
- Call graph lookup.
- Relationship lookup.
- Symbol search.
- File search.
- Package search.
- Documentation search.
- Configuration search.
- Fuzzy search.
- API interface validation.
- Version compatibility validation.
- Error validation.
- API performance validation.
- API documentation.

The capability provides stable internal interfaces over established repository intelligence.

---

# Architectural Position

The Repository Query API layer is an additive capability.

Its implementation belongs within the repository capability boundary:

    internal/
    └── capabilities/

The API layer consumes established repository capabilities.

The conceptual relationship is:

    Repository Infrastructure
            |
            +-- Repository Discovery
            +-- Project Structure
            +-- Dependency Analysis
            +-- Source Code Indexing
            +-- AST & Symbol Engine
            +-- Cross-Reference Engine
            +-- Knowledge Graph
            |
            v
    Repository Query APIs
            |
            +-- Repository Service
            +-- Symbol APIs
            +-- Graph APIs
            +-- Search Engine
            |
            v
    Future Internal Consumers

The API layer must not become an alternative implementation of upstream capabilities.

---

# API Design Principles

The internal API layer follows these principles:

- Stable contracts.
- Deterministic behavior.
- Explicit inputs.
- Explicit outputs.
- Read-oriented access.
- Controlled lifecycle operations.
- Strong error semantics.
- No hidden repository mutation.
- No duplicate analysis engines.
- No AI dependency.
- Minimal coupling.
- Long-term compatibility.
- Clear ownership of authoritative data.

---

# Repository Service Layer

The Repository Service Layer provides controlled access to repository lifecycle and repository-level information.

It is responsible for:

- Repository loading.
- Repository metadata.
- Repository statistics.
- Repository lifecycle management.
- Error handling.

The service layer must use established repository infrastructure.

It must not create a second repository abstraction that competes with the established repository model.

---

# Repository Loading API

The repository loading API provides access to an analyzed repository.

Loading must use the established repository discovery and workspace mechanisms.

The API must not:

- Reimplement repository discovery.
- Independently scan repository structure.
- Modify repository files.
- Execute repository source code.
- Install dependencies.

Repository loading must produce a controlled repository representation suitable for downstream queries.

---

# Repository Metadata API

The metadata API exposes established repository metadata.

Possible metadata includes:

- Repository identity.
- Repository root.
- Repository-relative structure.
- Supported analysis information.
- Analysis state.
- Capability availability.

Metadata must originate from authoritative repository services.

The API must not manufacture metadata from unrelated filesystem assumptions.

---

# Repository Statistics API

The statistics API exposes deterministic repository measurements.

Possible statistics include:

- File count.
- Directory count.
- Package count.
- Symbol count.
- Dependency count.
- Relationship count.
- Documentation count.
- Configuration count.

Statistics must represent the current repository analysis state.

The API must clearly distinguish unavailable statistics from zero-valued statistics.

---

# Repository Lifecycle Management

The repository service may expose controlled lifecycle operations required by repository consumers.

Lifecycle operations must respect established repository lifecycle semantics.

The API must not expose uncontrolled mutation of repository infrastructure.

Lifecycle transitions must be explicit and deterministic.

Invalid lifecycle transitions must return defined errors.

---

# Repository Error Handling

Repository APIs must use structured error semantics.

Errors should distinguish categories such as:

- Invalid input.
- Repository not found.
- Repository unavailable.
- Repository not loaded.
- Analysis unavailable.
- Unsupported capability.
- Invalid lifecycle state.
- Internal repository failure.

Errors must provide sufficient information for consumers to handle failures without exposing unnecessary internal implementation details.

---

# Symbol APIs

The Symbol APIs expose established symbol information from the AST and Symbol Engine.

The Symbol API surface includes:

- Find symbol.
- List symbols.
- Lookup by type.
- Lookup by package.
- Lookup by name.

The Symbol APIs must not independently parse source files or construct a competing symbol database.

---

# Find Symbol

The find-symbol operation resolves a symbol using supported identity or lookup criteria.

A symbol lookup must use established symbol identity.

Name-only lookup must not silently select an arbitrary symbol when multiple symbols match.

Ambiguous results must remain explicitly distinguishable.

---

# List Symbols

The list-symbol operation returns symbols available within a defined repository scope.

Supported scopes may include:

- Repository.
- Module.
- Package.
- File.

Results must be deterministic.

Result ordering must be explicitly defined and must not depend on map iteration order or filesystem enumeration order.

---

# Symbol Lookup by Type

The symbol API must support filtering symbols by established symbol type.

Examples include:

- Function.
- Method.
- Interface.
- Struct.
- Constant.
- Variable.
- Type.

The available symbol types must remain aligned with the authoritative Symbol Engine.

---

# Symbol Lookup by Package

The symbol API must support retrieving symbols associated with an established package.

Package identity must use the authoritative package model.

The API must not infer package identity solely from a directory name where authoritative package information exists.

---

# Symbol Lookup by Name

The symbol API may provide name-based lookup.

Name matching must define:

- Matching semantics.
- Scope.
- Case behavior where applicable.
- Ambiguity behavior.

A name must not be treated as a globally unique identifier.

---

# Graph APIs

The Graph APIs expose the established Knowledge Graph through controlled query interfaces.

The Graph API surface includes:

- Graph query.
- Graph traversal.
- Dependency lookup.
- Call graph lookup.
- Relationship lookup.

The Graph API must consume the authoritative Knowledge Graph.

It must not construct a second graph implementation.

---

# Graph Query

The graph query API provides structured graph queries.

Supported query dimensions may include:

- Node.
- Relationship.
- Node type.
- Relationship type.
- Repository.
- Module.
- Package.
- Path.
- Depth.

Queries must remain deterministic.

Graph queries must not mutate graph state.

---

# Graph Traversal

The graph traversal API provides controlled traversal of graph relationships.

Traversal must define:

- Starting node.
- Direction.
- Relationship constraints.
- Maximum traversal depth where applicable.

Traversal must protect against unbounded execution.

Cycles must be represented correctly without causing infinite traversal.

---

# Dependency Lookup

Dependency lookup provides access to established dependency relationships.

The API must consume Dependency Analysis and/or the Knowledge Graph.

It must not create an independent dependency resolver.

The API must preserve established dependency direction and identity.

---

# Call Graph Lookup

Call graph lookup provides access to established call relationships.

The API must consume the Cross-Reference Engine and/or Knowledge Graph.

It must not independently reconstruct function-call relationships.

Caller and callee direction must remain explicit.

---

# Relationship Lookup

Relationship lookup provides controlled access to graph relationships.

Lookup may be constrained by:

- Source.
- Target.
- Relationship type.
- Source type.
- Target type.

Results must remain deterministic.

---

# Search Engine

The Repository Search Engine provides deterministic search across repository information.

The initial search domains are:

- Symbols.
- Files.
- Packages.
- Documentation.
- Configuration.

Fuzzy search is also supported as a search capability.

Search must remain distinct from graph traversal.

---

# Symbol Search

Symbol search searches established symbol information.

The search engine must use the authoritative Symbol Engine data.

It must not reparse source code merely to perform ordinary symbol search.

Search results should provide sufficient identity information to distinguish symbols with identical names.

---

# File Search

File search searches indexed repository files.

File search must use established source indexing and repository-relative path semantics.

Search results must remain deterministic.

---

# Package Search

Package search searches established package information.

Package identity must originate from the authoritative repository and project structure models.

Directory names alone must not be treated as package identity when stronger information is available.

---

# Documentation Search

Documentation search provides access to searchable repository documentation.

The search engine must preserve association between documentation and its source artifact.

Documentation search must not generate documentation content.

---

# Configuration Search

Configuration search provides access to searchable configuration information.

Sensitive configuration values must not be exposed merely because they are searchable.

The search layer must respect established repository security and configuration handling rules.

---

# Fuzzy Search

Fuzzy search provides approximate matching over supported repository entities.

Fuzzy matching must remain deterministic.

For identical input and repository state:

- Candidate selection must remain equivalent.
- Ranking must remain equivalent.
- Result ordering must remain equivalent.

Fuzzy search must not use AI-generated similarity judgments.

---

# Search Result Model

Search results must contain sufficient information for consumers to identify the matched repository entity.

Depending on result type, this may include:

- Entity identity.
- Entity type.
- Name.
- Repository-relative path.
- Package.
- Symbol scope.
- Match information.

Search results should avoid duplicating large source artifacts.

---

# Search Ordering

Search result ordering must be explicitly defined.

Ordering must not depend on:

- Filesystem traversal order.
- Map iteration order.
- Thread scheduling.
- Unstable external state.

Where ranking is used, the ranking algorithm must be deterministic.

Ties must have deterministic secondary ordering.

---

# API Contract Stability

Internal API contracts must be explicit.

Each API contract should define:

- Operation.
- Inputs.
- Output.
- Errors.
- Preconditions.
- Postconditions.
- Determinism requirements.
- Lifecycle requirements where applicable.

Internal implementation details must not unnecessarily leak through API contracts.

---

# Input Validation

All API inputs must be validated before processing.

Validation must detect:

- Missing required inputs.
- Invalid identifiers.
- Invalid scopes.
- Invalid traversal parameters.
- Invalid filters.
- Invalid lifecycle operations.
- Unsupported query combinations.

Invalid inputs must produce structured errors.

---

# Output Validation

API implementations must ensure that returned data satisfies the defined contract.

Outputs must not contain:

- Invalid identities.
- Broken references.
- Unexpected nil values where prohibited.
- Duplicate results where uniqueness is required.
- Unstable ordering.

---

# Version Compatibility

Internal API evolution must preserve compatibility wherever practical.

Compatibility must consider:

- Method signatures.
- Input semantics.
- Output semantics.
- Error contracts.
- Identifier semantics.
- Result ordering where consumers depend upon it.

Breaking changes must be explicit.

Silent behavioral changes are prohibited.

---

# API Versioning

API versioning must remain separate from repository release versioning.

An internal API version identifies the compatibility contract of the interface.

Repository release numbers identify the overall software release.

The two concepts must not be conflated.

---

# Error Contract

Errors are part of the API contract.

Consumers must be able to distinguish expected operational failures from unexpected internal failures.

Errors must not require consumers to parse human-readable messages to determine their category.

Human-readable messages may accompany structured error information.

---

# Read-Only Boundary

Repository query APIs should remain read-oriented.

Query operations must not:

- Modify source files.
- Modify repository structure.
- Modify Git state.
- Modify ASTs.
- Modify symbol databases.
- Modify dependency databases.
- Modify graph state.

Controlled repository lifecycle operations are governed separately by the Repository Service Layer.

---

# Data Ownership

Each API must respect the ownership of the data it exposes.

The ownership model is:

    Repository Discovery
        -> Repository identity

    Project Structure
        -> Structural information

    Dependency Analysis
        -> Dependency information

    Source Code Indexing
        -> Indexed source information

    AST & Symbol Engine
        -> AST and symbol information

    Cross-Reference Engine
        -> Reference and call relationships

    Knowledge Graph
        -> Unified graph representation

    Search Engine
        -> Search representation and ranking

The API layer exposes these capabilities.

It does not become the authoritative source for their underlying facts.

---

# API Composition

Higher-level API operations may compose multiple authoritative capabilities.

For example:

    Repository Query
        |
        +-- Repository metadata
        +-- Symbol information
        +-- Graph information
        +-- Dependency information

Composition must preserve the ownership and semantics of each underlying capability.

A composed API must not silently alter authoritative data.

---

# Determinism

The API layer must produce deterministic results.

For equivalent repository state and equivalent inputs:

- API outputs must be equivalent.
- Search results must be equivalent.
- Ordering must be equivalent.
- Errors must be equivalent in category.
- Graph traversal results must be equivalent.

Concurrency must not introduce externally observable nondeterminism.

---

# Performance

API performance must remain suitable for large repositories.

Performance-sensitive operations include:

- Repository metadata retrieval.
- Symbol lookup.
- Symbol listing.
- Graph queries.
- Graph traversal.
- Dependency lookup.
- Call graph lookup.
- Search.
- Fuzzy search.

APIs should avoid unnecessary reconstruction of upstream data.

Caching may be introduced where justified by measurable performance requirements and must preserve deterministic behavior.

---

# Resource Management

API operations must respect established resource and lifecycle boundaries.

Operations must not leak:

- File handles.
- Memory.
- Goroutines.
- Repository resources.
- Search indexes.
- Graph resources.

Long-running operations must have controlled cancellation or lifecycle semantics where required by the underlying infrastructure.

---

# Concurrency

API implementations may support concurrent access where the underlying capability permits it.

Concurrency must not compromise:

- Determinism.
- Data integrity.
- Lifecycle correctness.
- Resource ownership.

Shared mutable state must be controlled.

Read-only operations should prefer safe concurrent access over unnecessary serialization when performance requirements justify it.

---

# Security

Repository APIs must not expose sensitive repository information beyond the established security boundary.

Security considerations include:

- Repository path exposure.
- Configuration secrets.
- Credential values.
- Environment data.
- Sensitive metadata.
- Unauthorized repository access.

API consumers must not receive capabilities beyond their intended scope.

---

# API Boundary

These APIs are internal repository infrastructure.

They provide stable interfaces for Limoxel's internal consumers.

They are not automatically public SDK contracts.

Public API exposure belongs to the SDK architecture and must be explicitly designed and documented there.

---

# Extensibility

The API layer must support additive extensions.

New operations should:

- Preserve existing contracts.
- Reuse authoritative capabilities.
- Avoid breaking existing consumers.
- Define explicit input and output contracts.
- Define deterministic behavior.
- Define error semantics.

New repository capabilities should be exposed through APIs without requiring modification of unrelated API contracts.

---

# Separation from Core

The Repository Query API implementation must remain outside the established core implementation boundary wherever possible.

The API layer must not modify the core engine merely to simplify repository capability implementation.

If an existing extension point is sufficient, it must be used.

If an extension point is insufficient, architectural review is required before modifying established core contracts.

---

# Separation from Intelligence

The API layer is deterministic infrastructure.

It must not:

- Generate semantic explanations.
- Infer repository intent.
- Use AI reasoning.
- Depend on probabilistic model output.
- Generate natural-language answers as an authoritative repository fact.

Future intelligence systems may consume these APIs.

The APIs themselves remain deterministic.

---

# Documentation Contract

Every stable internal API must have corresponding documentation defining:

- Purpose.
- Inputs.
- Outputs.
- Errors.
- Determinism.
- Lifecycle behavior.
- Compatibility expectations.

Documentation must remain synchronized with the implementation contract.

Implementation must not introduce undocumented public-facing internal behavior.

---

# Acceptance Criteria

The Repository Query API capability is considered complete when it provides:

- Repository loading API.
- Repository metadata API.
- Repository statistics API.
- Repository lifecycle management.
- Structured repository errors.
- Symbol lookup.
- Symbol listing.
- Symbol lookup by type.
- Symbol lookup by package.
- Symbol lookup by name.
- Graph querying.
- Graph traversal.
- Dependency lookup.
- Call graph lookup.
- Relationship lookup.
- Symbol search.
- File search.
- Package search.
- Documentation search.
- Configuration search.
- Deterministic fuzzy search.
- Stable interfaces.
- Explicit error contracts.
- Version compatibility rules.
- Deterministic result ordering.
- Input validation.
- Output validation.
- Performance validation.
- No duplicate repository analysis engines.
- No modification of established core architecture merely for convenience.
- No AI dependency.

---

# Architectural Guardrails

Implementation must stop and return to architectural review if any proposal requires:

- Reimplementing repository discovery.
- Reimplementing dependency analysis.
- Reimplementing source indexing.
- Reimplementing AST parsing.
- Reimplementing symbol extraction.
- Reimplementing cross-reference analysis.
- Reimplementing the Knowledge Graph.
- Making API consumers mutate authoritative data directly.
- Introducing unstable result ordering.
- Introducing hidden lifecycle transitions.
- Exposing secrets through search.
- Making internal APIs depend on AI.
- Breaking established core contracts without architectural justification.
- Introducing unnecessary dependencies.

These are architectural violations rather than implementation details.

---

# Authority

This document defines the internal Repository Query API capability.

The underlying repository capabilities remain authoritative for the facts they produce.

Repository Discovery is authoritative for repository identity and discovery.

Project Structure is authoritative for repository structure.

Dependency Analysis is authoritative for dependency information.

Source Code Indexing is authoritative for indexed source information.

The AST & Symbol Engine is authoritative for AST and symbol information.

The Cross-Reference Engine is authoritative for references and call relationships.

The Knowledge Graph is authoritative for unified graph representation.

The Search Engine is authoritative for search execution and ranking within its defined search boundary.

The API layer exposes these capabilities through stable interfaces.

Where this document conflicts with an established authoritative capability contract, the authoritative contract takes precedence and this document must be revised before implementation proceeds.

---

# Applicability

The principles and contracts defined in this document apply to:

- Repository service interfaces.
- Symbol APIs.
- Graph APIs.
- Search APIs.
- Internal API consumers.
- API validation.
- API compatibility management.
- Future internal integrations.

All implementations must remain consistent with deterministic behavior, explicit contracts, separation of responsibilities, minimal coupling, security, testability, and long-term maintainability.

---

# Change Policy

Changes to established API contracts must be deliberate and documented.

Changes should prefer additive evolution.

Existing consumers must not be broken without explicit compatibility review.

Any breaking change must define:

- Reason.
- Affected contracts.
- Migration requirements.
- Compatibility impact.
- Documentation changes.
- Validation requirements.

API changes that require modification of established core architecture require explicit architectural review before implementation.

---