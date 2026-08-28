# SDK Foundation

Project  : Limoxel  
Category : SDK & Public API  
Document : SDK Foundation  
Version  : 1.0  
Author   : Raj Joshi

---

# Overview

The Limoxel SDK Foundation defines the common foundation through which Limoxel capabilities are made available to external software.

It establishes a stable and consistent public interface for consuming Limoxel capabilities without requiring consumers to depend upon Limoxel's internal implementation.

The foundation provides the common rules and guarantees shared by Limoxel SDKs, including public API boundaries, API contracts, naming, documentation, errors, compatibility, lifecycle, versioning, and evolution.

It is the common foundation upon which Limoxel's capability-specific SDKs are exposed.

---

# What the SDK Foundation Provides

The SDK Foundation provides a consistent public model for interacting with Limoxel.

It defines:

- A public API surface
- Stable public types and operations
- Consistent API contracts
- Consistent error behavior
- Consistent naming conventions
- Consistent documentation expectations
- API lifecycle management
- Semantic versioning
- Backward compatibility expectations
- Deprecation rules
- Release and migration expectations

These properties apply across Limoxel SDKs unless a capability-specific specification explicitly defines additional behavior.

---

# Public API

The Limoxel SDK provides capabilities through public interfaces intended for external consumers.

A public API represents supported Limoxel behavior.

Public APIs are distinct from implementation details.

Consumers should be able to use supported Limoxel capabilities without depending upon:

- Internal implementation structures
- Internal package organization
- Private data representations
- Internal algorithms
- Internal lifecycle mechanisms
- Temporary implementation details

A change to internal implementation does not constitute a public API change when the supported external contract remains unchanged.

---

# Public and Internal Boundaries

Limoxel maintains a distinction between capabilities intended for external consumption and implementation details used internally.

The public SDK exposes supported behavior through stable contracts.

Internal implementation remains free to evolve within the boundaries of those contracts.

Public consumers must not be required to depend upon internal implementation details to use a supported SDK capability.

The public boundary therefore provides abstraction between:

- What Limoxel exposes
- How Limoxel implements it

This separation allows Limoxel's implementation to evolve without unnecessarily breaking external consumers.

---

# SDK Capabilities

Limoxel SDKs expose capabilities according to their public responsibility.

The SDK Foundation provides the common model used by capability-specific SDKs.

Capability-specific SDKs may expose functionality related to:

- Repository information
- Files
- Packages
- Symbols
- Search
- Knowledge graphs
- Engineering analysis
- Navigation
- Reasoning
- Events

Each capability remains responsible for defining its own domain-specific behavior.

The SDK Foundation defines the common public rules under which those capabilities are exposed.

---

# Public Contracts

A Limoxel SDK contract defines the behavior that consumers can rely upon.

A public contract may define:

- Available operations
- Accepted inputs
- Returned values
- Public types
- Error conditions
- Lifecycle behavior
- Ordering guarantees
- Empty-result behavior
- Compatibility expectations
- Other externally observable behavior

Only behavior that forms part of the supported public contract should be relied upon by consumers.

Implementation behavior that is not part of the public contract is not considered a public guarantee.

---

# Stable Types

Public SDK types represent concepts that are intended to be consumed by external software.

Public types must provide predictable meaning and behavior.

A public type should represent a stable concept rather than expose an implementation-specific structure.

Changes to internal representations must not unnecessarily require consumers to change their code.

Where internal and public representations differ, the public representation defines the supported external contract.

---

# Repository Contract

The Repository SDK provides access to repository-level concepts exposed by Limoxel.

Repository-facing behavior may include:

- Repository identification
- Repository metadata
- Workspace information
- Repository statistics
- Repository lifecycle
- Repository state

The exact operations and representations are defined by the Repository SDK.

The SDK Foundation requires those operations to follow the common public contract, lifecycle, error, compatibility, and documentation principles defined here.

---

# Symbol Contract

The Symbol SDK provides access to supported symbol-level concepts exposed by Limoxel.

Symbol-facing behavior may include:

- Symbol identification
- Symbol lookup
- Symbol hierarchy
- Symbol references
- Symbol documentation
- Symbol ownership

The public representation of a symbol is defined by the SDK contract rather than by any internal representation used to obtain it.

---

# Graph Contract

The Graph SDK provides access to supported knowledge-graph concepts.

Graph-facing behavior may include:

- Knowledge graph access
- Node traversal
- Relationship traversal
- Graph filtering
- Graph export

Graph consumers interact with the public graph model rather than depending upon the internal graph implementation.

---

# Search Contract

The Search SDK provides a consistent public interface for supported Limoxel search capabilities.

Search behavior may include:

- Repository search
- Symbol search
- Package search
- Documentation search
- Configuration search

A search contract defines the meaning of the query, result, and error behavior exposed to consumers.

Search implementation details remain outside the public contract unless explicitly documented as supported behavior.

---

# Intelligence Contract

The Intelligence SDK provides public access to supported engineering-intelligence capabilities.

The public intelligence contract defines the externally observable capabilities and representations available to consumers.

Consumers interact with stable public intelligence concepts rather than depending upon internal intelligence implementation.

The public contract may evolve independently of the internal algorithms used to produce the exposed results, provided that supported compatibility guarantees are preserved.

---

# Error Contract

Errors are part of the public SDK contract.

SDK errors must provide predictable information about unsuccessful operations.

Where programmatic distinction is required, errors should expose stable semantic categories rather than requiring consumers to interpret arbitrary human-readable messages.

Error behavior includes, where applicable:

- Invalid input
- Missing resources
- Unsupported operations
- Invalid state
- Lifecycle violations
- Internal processing failures

Human-readable error information may provide additional context but must not be the sole mechanism for determining a stable semantic error category where such distinction is required by the contract.

---

# Naming

Public SDK names must communicate the concept or capability they represent.

Names should be:

- Clear
- Consistent
- Predictable
- Stable
- Responsibility-oriented

Public names should not depend upon temporary implementation terminology.

Once a public name becomes part of a supported contract, changing it may constitute a compatibility change.

---

# Documentation

Public SDK behavior must be documented sufficiently for consumers to understand and use the supported interface correctly.

Documentation should describe:

- What an API provides
- Required inputs
- Returned information
- Errors
- Lifecycle requirements
- Important behavioral guarantees
- Compatibility considerations

Documentation describes supported behavior.

Implementation details that are not part of the public contract should not be presented as public guarantees.

---

# API Lifecycle

Every public API has a lifecycle.

An API may progress through:

1. Introduction
2. Stabilization
3. Supported operation
4. Deprecation, when required
5. Removal, when permitted by the compatibility policy

A public API must not be removed or changed incompatibly without applying the appropriate versioning and compatibility rules.

API lifecycle status must be understandable from the public documentation.

---

# Backward Compatibility

Backward compatibility means that existing consumers can continue using supported public APIs according to the guarantees of their applicable SDK version.

Compatibility applies to externally observable contracts, including:

- Public API availability
- Public names
- Public types
- Function and method signatures
- Supported behavior
- Error semantics
- Documented guarantees

Internal implementation changes are compatible when they preserve the supported public contract.

Compatibility is evaluated from the consumer's perspective.

---

# Semantic Versioning

Limoxel SDK releases use Semantic Versioning.

The version format is:

    MAJOR.MINOR.PATCH

Version changes communicate the compatibility impact of public API changes.

## Major Version

A major version indicates incompatible public API changes.

Examples include:

- Removing a supported public API
- Incompatibly changing a public contract
- Removing required supported behavior
- Changing public semantics in a way that requires consumer changes

## Minor Version

A minor version indicates backward-compatible functionality additions.

Examples include:

- Adding a new public API
- Adding a new optional capability
- Adding new supported functionality without breaking existing consumers

## Patch Version

A patch version indicates backward-compatible corrections.

Examples include:

- Bug fixes
- Correctness improvements
- Security corrections
- Documentation corrections
- Internal improvements that preserve the public contract

Version numbers describe public compatibility impact rather than the size of an internal implementation change.

---

# Deprecation

A public API may be deprecated when it remains supported but is no longer the preferred interface.

A deprecated API should:

- Remain available for the supported compatibility period
- Be clearly identified as deprecated
- Provide a replacement where applicable
- Preserve its supported behavior during the applicable period
- Provide migration guidance when required

Deprecation provides consumers with a controlled transition path rather than requiring an immediate incompatible change.

---

# Release Compatibility

A released SDK represents a supported public contract.

Consumers should be able to determine the compatibility expectations of a release from its version and documentation.

A compatible release must preserve previously supported public behavior within the guarantees of its version.

New functionality may be introduced without invalidating existing consumers when the addition remains backward compatible.

---

# Migration

When a public API cannot remain compatible, a supported migration path should be provided where practical.

Migration information should identify:

- The affected API
- The replacement API
- Relevant behavioral differences
- Required consumer changes
- Applicable version boundary
- Deprecation status where applicable

Migration guidance must describe actual public behavior rather than internal implementation changes.

---

# SDK Consistency

All Limoxel SDKs share the common principles defined by the SDK Foundation.

Consumers should encounter consistent behavior for:

- Naming
- Documentation
- Errors
- Lifecycle
- Versioning
- Compatibility
- Deprecation

Capability-specific SDKs may define additional domain-specific behavior, but should not unnecessarily contradict the common SDK model.

---

# Deterministic Public Behavior

Public SDK behavior must be predictable and explicitly defined.

Equivalent supported inputs should produce behavior consistent with the applicable contract.

Public APIs must not depend upon undocumented internal behavior.

Where ordering, identity, lifecycle, or result semantics matter to consumers, those properties should be explicitly defined by the relevant public contract.

---

# Evolution

The SDK Foundation is designed to allow Limoxel's public API surface to grow without requiring consumers to understand the internal evolution of Limoxel.

New capabilities may be added through additional public interfaces.

Existing capabilities may evolve while preserving their supported contracts.

Internal implementation may change independently when the public contract remains valid.

The public SDK therefore provides a stable boundary between Limoxel's evolving implementation and external consumers.

---

# Non-Goals

The SDK Foundation does not define:

- Internal implementation algorithms
- Repository analysis implementation
- Parsing implementation
- Graph construction algorithms
- Intelligence algorithms
- CLI behavior
- Filesystem implementation
- Language-specific implementation
- Plugin implementation
- Enterprise deployment behavior
- Individual SDK feature implementations

Those concerns belong to their respective capability specifications.

---

# Authority

The SDK Foundation defines the common public principles and guarantees applicable to Limoxel SDKs.

Capability-specific SDK specifications define the detailed behavior of individual SDK capabilities while remaining consistent with this foundation.

Where a capability-specific specification defines behavior unique to that capability, the capability-specific specification governs that behavior.

---

# Applicability

This specification applies to Limoxel SDKs and their public APIs.

It applies to:

- Public SDK interfaces
- Public SDK types
- Public API contracts
- SDK naming
- SDK documentation
- SDK errors
- SDK lifecycle
- SDK versioning
- SDK compatibility
- SDK deprecation
- SDK migration behavior

It applies to existing and future public SDK capabilities unless explicitly superseded by an approved specification.

---

# Change Policy

Changes to the SDK Foundation must preserve the fundamental principles of:

- Stable public contracts
- Clear public and internal boundaries
- Predictable behavior
- Backward compatibility
- Semantic versioning
- Consistent API lifecycle
- Clear documentation
- Stable error semantics
- Consistent public API conventions

Changes that alter the meaning of public compatibility, lifecycle, versioning, or the fundamental public API boundary require explicit architectural approval.

This specification remains authoritative until an approved revision supersedes it.

---