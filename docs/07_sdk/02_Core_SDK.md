# Core SDK

Project  : Limoxel  
Category : SDK & Public API  
Document : Core SDK  
Version  : 1.0  
Author   : Raj Joshi

---

# Overview

The Limoxel Core SDK provides stable public access to the fundamental repository capabilities exposed by Limoxel.

It gives external software a consistent interface for working with repository information, workspaces, files, packages, symbols, and search without requiring consumers to depend upon internal implementation details.

The Core SDK represents the foundational repository-facing public API of Limoxel.

It is composed of the following capability areas:

- Repository
- File
- Package
- Symbol
- Search

Each capability provides a focused public interface while following the common principles established by the SDK Foundation.

---

# Purpose

The Core SDK provides programmatic access to repository-level information and operations through stable public APIs.

It allows SDK consumers to:

- Work with repositories
- Work with repository workspaces
- Access repository metadata
- Access repository statistics
- Observe repository lifecycle state
- Discover files
- Access file metadata
- Locate files
- Work with file indexing information
- Access file relationships
- Discover packages
- Locate packages
- Access package statistics
- Navigate package hierarchies
- Access package relationships
- Locate symbols
- Navigate symbol hierarchies
- Access symbol references
- Access symbol documentation
- Access symbol ownership
- Search repository content
- Search symbols
- Search packages
- Search documentation
- Search configuration

The Core SDK exposes these capabilities through public contracts rather than exposing the underlying implementation.

---

# Repository SDK

The Repository SDK provides access to repository-level concepts exposed by Limoxel.

A repository represents a software repository that Limoxel can work with through its supported repository capabilities.

The Repository SDK provides a public interface for repository management and repository information.

## Repository Management

Repository management provides access to supported repository operations and repository state.

The public interface may provide:

- Repository identification
- Repository location
- Repository state
- Repository availability
- Repository lifecycle information

Repository management must provide predictable behavior for supported repository states.

---

## Workspace Management

The Repository SDK provides access to supported workspace information associated with a repository.

Workspace information represents the working context in which repository operations are performed.

The public contract defines the supported workspace behavior without exposing internal workspace implementation.

---

## Repository Metadata

Repository metadata provides descriptive information about the repository.

Metadata may include information such as:

- Repository identity
- Repository location
- Repository characteristics
- Repository state
- Other supported repository attributes

The exact metadata exposed is determined by the public contract.

---

## Repository Statistics

The Repository SDK may provide statistics describing the repository.

Statistics may include supported measurements such as:

- File counts
- Package counts
- Symbol counts
- Other repository-level measurements

Statistics represent the repository state available through Limoxel at the time of the operation.

---

## Repository Lifecycle

The Repository SDK exposes supported repository lifecycle behavior.

Lifecycle operations and state transitions must have defined public semantics.

Consumers must be able to determine the supported state of a repository through the public contract rather than depending upon internal lifecycle implementation.

---

# File SDK

The File SDK provides access to supported file-level concepts within a repository.

A file represents a repository file exposed through Limoxel's repository model.

The File SDK provides stable public access to file discovery, metadata, lookup, indexing information, and relationships.

---

## File Discovery

File discovery provides access to files known to the repository model.

Consumers may use file discovery to locate files according to supported repository and search criteria.

Discovery results must use stable public representations.

---

## File Metadata

File metadata provides descriptive information associated with a file.

Metadata may include:

- Repository-relative path
- File identity
- File type
- File state
- File size
- Other supported file attributes

Only documented metadata constitutes a public guarantee.

---

## File Lookup

File lookup provides a direct way to locate a supported file using its public identity or repository-relative location.

Lookup behavior must define:

- Successful lookup behavior
- Missing-file behavior
- Invalid input behavior
- Error behavior

---

## File Indexing

The File SDK may expose supported information about the indexing state of a file.

Indexing information allows consumers to understand whether a file has been processed within the repository model where such state is part of the public contract.

Internal indexing mechanisms are not part of the public File SDK contract.

---

## File Relationships

The File SDK may expose supported relationships between files and other repository entities.

Relationships may represent connections such as:

- File-to-package relationships
- File-to-symbol relationships
- Other supported repository relationships

The meaning of each exposed relationship is defined by the applicable public contract.

---

# Package SDK

The Package SDK provides access to package-level concepts represented within a repository.

A package represents a supported logical grouping of repository content.

The Package SDK provides public access to package discovery, lookup, statistics, hierarchy, and relationships.

---

## Package Discovery

Package discovery provides access to packages represented by Limoxel.

Consumers may discover packages using supported repository information and filtering criteria.

---

## Package Lookup

Package lookup provides a direct way to locate a supported package through its public identity.

Lookup behavior includes defined handling for:

- Existing packages
- Missing packages
- Invalid lookup input
- Other supported failure conditions

---

## Package Statistics

The Package SDK may expose statistics describing a package.

Statistics may include supported information such as:

- Number of files
- Number of symbols
- Package-level relationships
- Other package measurements

Only documented statistics are considered part of the public contract.

---

## Package Hierarchy

The Package SDK provides access to supported package hierarchy information.

Hierarchy allows consumers to understand relationships between packages where such relationships are represented by the repository model.

Hierarchy traversal must provide predictable public behavior.

---

## Package Relationships

The Package SDK may expose relationships between packages and other supported repository entities.

Relationships must have defined semantics and stable public representations.

---

# Symbol SDK

The Symbol SDK provides access to symbols represented within a repository.

A symbol represents a supported program entity exposed through Limoxel's repository model.

The Symbol SDK provides public access to symbol lookup, hierarchy, references, documentation, and ownership.

---

## Symbol Lookup

Symbol lookup provides a way to locate supported symbols through their public identity or supported lookup criteria.

Lookup behavior must define successful results and applicable failure conditions.

---

## Symbol Hierarchy

The Symbol SDK provides access to supported relationships between symbols within a hierarchy.

Hierarchy information may represent relationships such as:

- Parent and child symbols
- Nested symbols
- Other supported structural relationships

The public meaning of each relationship is defined by the applicable symbol contract.

---

## Symbol References

The Symbol SDK provides access to supported references to symbols.

References allow consumers to determine where a symbol is referenced within the repository model.

Reference behavior must provide stable public representations.

---

## Symbol Documentation

The Symbol SDK may provide documentation associated with a symbol where such documentation is available through Limoxel.

Documentation may include supported descriptive information associated with the symbol.

The absence of documentation must have predictable public behavior.

---

## Symbol Ownership

The Symbol SDK provides access to supported ownership information for symbols.

Ownership identifies the repository entity responsible for or containing a symbol according to the public repository model.

---

# Search SDK

The Search SDK provides a unified public interface for searching supported Limoxel repository information.

Search allows consumers to locate information without requiring knowledge of the underlying indexing or search implementation.

The Search SDK supports the following search areas:

- Repository
- Symbol
- Package
- Documentation
- Configuration

---

## Repository Search

Repository search provides access to supported search over repository content and repository information.

Results must provide stable public representations and predictable behavior.

---

## Symbol Search

Symbol search allows consumers to locate symbols using supported search criteria.

Search results must identify matching symbols through stable public representations.

---

## Package Search

Package search allows consumers to locate supported packages using defined search criteria.

Results must follow the applicable package and search contracts.

---

## Documentation Search

Documentation search provides access to supported searchable documentation associated with repository content.

Results must distinguish supported documentation information from unrelated repository content where required by the public contract.

---

## Configuration Search

Configuration search provides access to supported configuration information represented within the repository.

Search behavior must respect the applicable repository and configuration semantics.

---

# Search Results

Search results are public representations.

A result may contain supported information such as:

- Matching entity
- Entity identity
- Repository-relative location
- Match information
- Relevance information
- Other documented result attributes

The exact result structure and ordering guarantees are defined by the applicable Search SDK contract.

Consumers must not depend upon undocumented internal search structures.

---

# Common Behavior

Core SDK capabilities follow the common SDK Foundation principles.

These include:

- Stable public contracts
- Predictable behavior
- Explicit errors
- Consistent naming
- Documented public behavior
- Backward compatibility
- Semantic versioning
- Defined lifecycle
- Separation between public interfaces and internal implementation

Capability-specific behavior may extend these principles where required by the nature of the capability.

---

# Entity Identity

Repository entities exposed through the Core SDK must have stable public identities appropriate to their respective type.

Identity allows consumers to:

- Locate entities
- Compare entities
- Follow supported relationships
- Reuse references across compatible operations

Internal identifiers are not automatically public identifiers.

Only identities defined by the applicable public contract should be used by external consumers.

---

# Relationships

Core SDK entities may expose relationships between repository objects.

Supported relationships provide structured information about how repository entities are connected.

Relationship semantics must be explicitly defined where they form part of the public API.

Consumers must not infer unsupported relationships from internal implementation structures.

---

# Empty Results

An operation that successfully finds no matching entity should be distinguishable from an operation that fails.

Where applicable:

- An empty collection represents a successful operation with no matching results.
- An error represents an unsuccessful operation.

The exact behavior of each operation is defined by its public contract.

---

# Lifecycle Behavior

Core SDK resources that have a lifecycle must expose predictable lifecycle behavior.

Consumers must be able to determine:

- When a resource becomes available
- Which operations are valid for its current state
- Which operations terminate or release its lifecycle
- How invalid lifecycle operations are reported

Lifecycle behavior must not depend upon undocumented internal state.

---

# Error Behavior

Core SDK operations use the common SDK error model.

Errors should allow consumers to distinguish meaningful failure conditions programmatically where such distinction is required.

Typical failure conditions may include:

- Invalid input
- Missing repository entity
- Unsupported operation
- Invalid lifecycle state
- Unavailable resource
- Internal processing failure

Human-readable error information may provide additional context.

---

# Consistency Across Core SDKs

Repository, File, Package, Symbol, and Search APIs should provide consistent public behavior wherever their concepts overlap.

For example:

- Entity identities should remain coherent across related APIs.
- Relationships should use compatible representations.
- Errors should follow the common SDK error model.
- Naming should remain consistent.
- Lifecycle semantics should remain compatible.
- Shared concepts should not be represented inconsistently without a defined reason.

This consistency allows consumers to combine Core SDK capabilities without learning unrelated API conventions for each capability.

---

# Public Stability

The Core SDK represents supported public functionality.

Changes to public interfaces must follow the compatibility and versioning rules established by the SDK Foundation.

Internal implementation may evolve without requiring consumer changes when the public contract remains unchanged.

A public contract must not be changed merely because an internal representation has changed.

---

# Capability Independence

Each Core SDK capability has a focused responsibility.

The Repository SDK is responsible for repository-level concepts.

The File SDK is responsible for file-level concepts.

The Package SDK is responsible for package-level concepts.

The Symbol SDK is responsible for symbol-level concepts.

The Search SDK is responsible for search behavior.

These responsibilities may interact, but overlapping functionality should not create conflicting representations or duplicated public responsibilities.

---

# Non-Goals

The Core SDK does not define:

- Internal repository implementation
- Internal filesystem implementation
- Parser implementation
- Language-specific analysis algorithms
- Knowledge graph implementation
- Engineering-intelligence algorithms
- CLI command behavior
- SDK developer portal behavior
- Plugin architecture
- Enterprise deployment behavior

Those concerns are outside the Core SDK public contract.

---

# Authority

This document defines the public specification of Limoxel's Core SDK.

The SDK Foundation defines the common public API, compatibility, lifecycle, versioning, error, documentation, and engineering principles applicable to the Core SDK.

The individual Core SDK capabilities defined here establish the public meaning of repository, file, package, symbol, and search access.

Implementation details that are not part of these public contracts are not authoritative for SDK consumers.

---

# Applicability

This specification applies to Limoxel's Core SDK and its public interfaces for:

- Repository access
- Workspace access
- Repository metadata
- Repository statistics
- Repository lifecycle
- File discovery
- File metadata
- File lookup
- File indexing
- File relationships
- Package discovery
- Package lookup
- Package statistics
- Package hierarchy
- Package relationships
- Symbol lookup
- Symbol hierarchy
- Symbol references
- Symbol documentation
- Symbol ownership
- Repository search
- Symbol search
- Package search
- Documentation search
- Configuration search

It applies to consumers of the public Core SDK and to compatible versions of its public contracts.

---

# Change Policy

Changes to the Core SDK must preserve the principles established by the SDK Foundation.

Public changes must be evaluated for:

- Contract compatibility
- Behavioral compatibility
- Type compatibility
- Error compatibility
- Lifecycle compatibility
- Versioning impact
- Documentation accuracy

New capabilities may be added when they do not unnecessarily invalidate existing consumers.

Incompatible changes must follow the applicable semantic-versioning and migration policies.

This specification remains authoritative until an approved revision supersedes it.

---