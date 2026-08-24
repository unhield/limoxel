# Repository Metadata

Project    : Limoxel  
Category   : Repository  
Document   : Repository Metadata  
Version    : 1.0  
Author     : Raj Joshi

---

# Purpose

This document defines the deterministic repository metadata capability of Limoxel.

Repository Metadata provides a structured repository profile containing metadata that can be established from the repository and its available local source-control information.

The capability extends the existing repository representation without modifying the established Workspace, Project, Repository, Filesystem, Language, Parser, Extension, Runtime, Configuration, or platform architecture.

Repository Metadata is observational infrastructure.

It does not modify repository state and does not infer unavailable information.

---

# Scope

Repository Metadata is responsible for collecting and representing:

- Repository name.
- Repository owner or namespace where deterministically available.
- Default branch.
- Latest commit.
- Commit statistics.
- Contributors.
- Tags.
- Releases where locally available.
- Repository age.

The capability produces a repository profile that can be consumed by Repository Discovery and subsequent repository-analysis capabilities.

---

# Architectural Position

Repository Metadata is part of the repository capability boundary:

    internal/
    └── capabilities/
        └── repository/

The metadata capability consumes the existing Limoxel foundation.

The dependency direction remains:

    Existing Limoxel Foundation
            |
            v
    Repository Metadata
            |
            v
    Repository Capability Consumers

Existing foundation packages must not depend on Repository Metadata.

Repository Metadata must not replace the existing Repository domain model.

The existing Repository abstraction remains the authoritative representation of repository identity and ownership.

---

# Relationship with Repository Discovery

Repository Discovery establishes the repository context and produces the repository-level representation required by subsequent capabilities.

Repository Metadata provides the structured metadata portion of that representation.

The relationship is:

    Repository
        |
        v
    Repository Discovery
        |
        v
    Repository Metadata
        |
        v
    Repository Profile

Repository Metadata must reuse information already established during discovery wherever possible.

It must not independently repeat complete repository traversal when the required information already exists in the discovery result.

---

# Repository Profile

The repository profile represents the metadata available for a repository at the time of discovery.

Conceptually:

    Repository Profile
        |
        +-- Repository Identity
        +-- Owner
        +-- Default Branch
        +-- Latest Commit
        +-- Commit Statistics
        +-- Contributors
        +-- Tags
        +-- Releases
        +-- Repository Age

Each field must have explicit semantics.

Information that cannot be determined must remain explicitly unavailable.

Unavailable information must never be replaced with fabricated values or assumptions.

---

# Repository Name

Repository Metadata shall provide the repository name where it can be determined deterministically.

The repository name should be derived from the established repository representation or the repository root according to the existing repository conventions.

The metadata capability must not infer a repository name from remote hosting services when a deterministic local representation is already available.

The repository name must remain stable for the same repository context.

---

# Repository Owner

Repository owner or namespace information may be provided when it can be determined deterministically from available repository information.

Possible sources may include locally available repository metadata or explicitly configured repository identity.

Remote hosting ownership must not be silently treated as local repository truth.

If ownership cannot be determined from the available information, the owner must remain unavailable.

The capability must not guess ownership from:

- Directory names.
- Local usernames.
- Remote URLs without an explicitly defined interpretation.
- File authorship.
- Contributor identity.

---

# Default Branch

Repository Metadata shall provide the default branch where it can be determined from available repository information.

The metadata capability must distinguish between:

- Current branch.
- Default branch.

The current branch must not automatically be treated as the default branch unless the repository information explicitly establishes that relationship.

If a default branch cannot be determined reliably, the field must remain unavailable.

Branch metadata collection must be read-only.

---

# Latest Commit

Repository Metadata shall provide information about the latest commit available from the local repository.

The latest commit representation should provide sufficient information for downstream repository capabilities to identify the commit deterministically.

Where supported, the metadata may include:

- Commit identifier.
- Author information.
- Commit timestamp.
- Commit message metadata.
- Parent relationship information required by the repository profile.

The capability must not modify repository history while collecting commit information.

---

# Commit Statistics

Repository Metadata shall provide deterministic commit statistics where the local repository contains sufficient history.

Commit statistics may include:

- Total commit count.
- Earliest available commit.
- Latest commit.
- Commit activity over the available history.
- Time range represented by the available history.

Statistics must be calculated from the repository history that is actually available.

The capability must not imply that a shallow repository contains complete historical information.

If history is incomplete, the metadata must represent the available history accurately rather than presenting partial history as complete repository history.

---

# Contributors

Repository Metadata shall provide contributor information where it can be deterministically derived from available repository history.

Contributor information may include:

- Contributor identity.
- Commit count.
- First contribution timestamp where available.
- Latest contribution timestamp where available.

Contributor identity must be represented according to the available repository metadata.

The capability must not attempt probabilistic identity resolution.

Different author identities must not be merged merely because they appear similar.

Contributor statistics must reflect the available repository history.

---

# Tags

Repository Metadata shall provide repository tags where locally available.

Tag information should preserve:

- Tag name.
- Associated commit identity where deterministically available.
- Tag type where deterministically available.
- Relevant timestamp information where available.

Tags must be represented deterministically.

Tag ordering must not depend on filesystem or map iteration order.

The metadata capability must not create, modify, delete, or retag repository history.

---

# Releases

Repository Metadata may provide release information when releases are represented in locally available repository metadata.

Locally available release information must be distinguished from releases maintained exclusively by an external hosting service.

External release information must not be silently presented as locally established repository truth.

If release metadata is unavailable locally, the release field must remain explicitly unavailable.

Remote release integration belongs behind a separate explicit integration boundary.

---

# Repository Age

Repository age shall be derived from deterministically available repository history.

Where complete repository history is available, repository age may be represented using the earliest available repository commit as the historical starting point.

Where history is incomplete, the result must clearly represent the available historical boundary rather than claiming an exact repository age that cannot be established.

Repository age must therefore be derived from evidence rather than inferred from:

- Directory creation timestamps.
- Filesystem timestamps.
- Remote hosting timestamps.
- Package metadata.
- Documentation dates.

Filesystem timestamps must not be treated as authoritative repository age.

---

# Local Repository Truth

Repository Metadata distinguishes between information that is established locally and information that requires external services.

Local repository truth may include information available through:

- Existing Repository abstractions.
- Local filesystem metadata.
- Local Git repository metadata.
- Local Git history.
- Local tags.
- Locally available release metadata.

External information may include:

- Remote hosting ownership.
- Remote default branch configuration.
- Remote release records.
- Remote contributor information.
- Remote repository creation dates.

External information must not be silently incorporated into the local repository profile.

---

# External Metadata

Repository Metadata does not require external network services to produce a valid local repository profile.

External metadata acquisition, authentication, remote API communication, caching, rate limiting, and remote service failure handling are outside the local metadata contract.

If external metadata support is introduced later, it must be implemented behind an explicit provider boundary.

The local repository profile must remain valid when no remote service is available.

---

# Availability Semantics

Metadata fields must distinguish between:

- Known.
- Unknown.
- Unavailable.
- Not applicable.

The implementation must not collapse these states into fabricated defaults.

For example:

    Default Branch : unavailable

is materially different from:

    Default Branch : main

when the repository does not provide sufficient evidence that `main` is the default branch.

Availability semantics must remain deterministic.

---

# Determinism

Repository Metadata must produce deterministic results.

For equivalent repository state and equivalent metadata configuration:

    Same Repository State
            |
            v
    Same Metadata Sources
            |
            v
    Same Repository Profile

Determinism applies to:

- Repository name.
- Owner representation.
- Branch information.
- Commit information.
- Commit statistics.
- Contributor ordering.
- Tag ordering.
- Release ordering.
- Repository age.
- Availability states.

Map iteration order, filesystem enumeration order, or incidental operating-system behavior must never determine the observable ordering of metadata.

---

# Ordering

Collections within the repository profile must have explicit deterministic ordering.

This applies to:

- Contributors.
- Tags.
- Releases.
- Commit statistics where represented as collections.

Ordering must be based on defined metadata fields rather than incidental implementation order.

The ordering contract must remain stable for equivalent repository state.

---

# Immutability

The repository profile is observational output.

Once produced, consumers must not be able to mutate the internal metadata state maintained by the capability.

Collections should be exposed through immutable or defensive representations according to the existing Limoxel engineering conventions.

Subsequent repository capabilities may derive information from the profile but must not mutate the original repository metadata.

---

# Error Handling

Repository Metadata shall integrate with the established Limoxel error architecture.

Metadata failures must distinguish between conditions that prevent the repository profile from being produced and conditions that only make individual metadata fields unavailable.

A failure to obtain optional metadata must not automatically invalidate otherwise usable repository information.

Examples of non-fatal conditions include:

- Missing optional Git metadata.
- Unavailable contributor history.
- Missing local release information.
- Incomplete repository history.
- Unsupported metadata source.

Fatal conditions may include:

- Invalid repository context.
- Unrecoverable repository metadata access failure.
- Corrupted repository state that prevents required metadata from being read safely.

Errors must retain useful diagnostic context without exposing unnecessary sensitive information.

---

# Security

Repository Metadata is a read-only capability.

Metadata collection must never:

- Modify repository files.
- Modify Git history.
- Create commits.
- Modify branches.
- Modify tags.
- Modify repository configuration.
- Execute repository hooks.
- Execute repository source code.
- Execute build commands.
- Execute package-manager lifecycle commands.

Repository-controlled metadata must be treated as untrusted data.

Metadata parsing must remain bounded and deterministic.

External metadata providers, if introduced later, must have explicit authentication and security boundaries.

---

# Performance

Repository Metadata must avoid unnecessary repeated repository traversal.

The implementation should reuse information already obtained by Repository Discovery.

The metadata capability must avoid:

- Repeated complete history scans when results can be reused.
- Repeated filesystem traversal.
- Duplicate metadata collection.
- Duplicate contributor calculation.
- Duplicate tag enumeration.

Performance optimizations must not compromise deterministic behavior.

Caching must not be introduced without a defined invalidation strategy.

---

# Large Repository Handling

Repository Metadata must remain suitable for repositories containing large commit histories and large contributor or tag sets.

The implementation must use bounded processing where required.

Large repository behavior must remain deterministic.

Where the available repository history is incomplete or intentionally bounded, the resulting metadata must represent the available evidence accurately.

The capability must not silently transform partial repository information into apparently complete statistics.

---

# API Boundary

Repository Metadata contracts remain internal to the repository capability layer unless explicitly promoted through the established public API architecture.

Internal metadata readers and source-control-specific mechanisms should remain private.

Consumers should interact with the repository profile rather than the implementation details used to collect individual fields.

The repository profile should prefer stable domain values and immutable representations.

---

# Extensibility

Repository Metadata must support future metadata sources without changing the repository profile unnecessarily.

Potential future metadata providers may include:

- Local Git metadata.
- Remote hosting metadata.
- Enterprise repository metadata.
- Repository service metadata.

Additional providers must not change the semantics of existing local metadata.

Provider-specific behavior must remain isolated from the repository profile contract.

A provider must never overwrite established local truth with less authoritative information.

---

# Compatibility

Repository Metadata must preserve the existing Limoxel foundation.

The capability must consume existing:

- Workspace contracts.
- Project contracts.
- Repository contracts.
- Filesystem contracts.
- Error contracts.
- Platform services.

No existing core contract may be modified merely to simplify metadata collection.

If an existing contract is insufficient, an additive adapter or capability-layer abstraction must be considered first.

Any exceptional core modification requires explicit architectural review and proof that an extension-layer solution is insufficient.

---

# Repository Profile Integrity

The repository profile must represent the state observed during metadata collection.

It must not:

- Fabricate missing fields.
- Merge unrelated identities.
- Treat partial history as complete history.
- Treat current branch as default branch without evidence.
- Treat local directory timestamps as repository age.
- Treat remote information as local truth.
- Infer ownership probabilistically.

Every populated field must have a deterministic source.

---

# Acceptance Criteria

Repository Metadata is considered complete when it can deterministically provide, where available:

- Repository name.
- Repository owner or namespace.
- Default branch.
- Latest commit.
- Commit statistics.
- Contributors.
- Tags.
- Releases available through supported local metadata.
- Repository age.

The implementation must additionally:

- Represent unavailable information explicitly.
- Preserve local repository truth.
- Avoid fabricated metadata.
- Preserve deterministic ordering.
- Preserve immutable profile semantics.
- Remain read-only.
- Reuse existing Limoxel repository infrastructure.
- Avoid unnecessary filesystem traversal.
- Avoid unnecessary history traversal.
- Avoid unnecessary dependencies.
- Remain independent of external network services.
- Preserve the established core architecture.

---

# Architectural Guardrails

Implementation must stop and return to architectural review if any proposed implementation requires:

- Modification of an existing core package merely for convenience.
- A replacement Repository domain model.
- A replacement filesystem abstraction.
- A replacement error architecture.
- A dependency from the core into `internal/capabilities`.
- Remote network access as a mandatory requirement for local repository metadata.
- Probabilistic contributor or owner identification.
- Fabrication of unavailable repository metadata.
- Modification of Git repository state.
- Execution of repository-controlled commands.
- Unbounded historical processing.
- Unjustified third-party dependencies.

These conditions represent architectural violations rather than implementation details.

---

# Architectural Stability

Repository Metadata is an additive repository capability.

Its implementation must remain replaceable behind stable internal contracts without requiring structural changes to the existing foundation.

Future repository capabilities should consume the repository profile rather than independently repeating metadata collection.

Repository Metadata should therefore remain focused on one responsibility:

> **Provide a deterministic, read-only, and evidence-based profile of repository metadata available from authoritative sources.**

---

# Authority

This document defines the repository metadata capability and its engineering boundaries.

The existing Limoxel Core Foundation documentation remains authoritative for Workspace, Project, Repository, Filesystem, Language, Parser, Extension, Runtime, Configuration, platform services, error handling, dependency direction, and established architectural principles.

Where this document conflicts with an established foundation contract, the existing foundation contract takes precedence and this document must be revised before implementation proceeds.

---

# Applicability

The principles and contracts defined in this document apply to repository metadata collection and all consumers of the resulting repository profile.

They govern:

- Repository identity metadata.
- Branch metadata.
- Commit metadata.
- Commit statistics.
- Contributor information.
- Tag information.
- Release information.
- Repository age.
- Metadata availability.
- Metadata providers.
- Repository profile construction.

All implementations must remain consistent with Limoxel's principles of deterministic behavior, read-only repository observation, explicit evidence, minimal coupling, and extension without unnecessary modification of established foundations.

---

# Change Policy

Repository Metadata should evolve through additive capability-layer changes whenever possible.

Changes must preserve:

- Existing core contracts.
- Existing repository domain semantics.
- Deterministic metadata behavior.
- Explicit availability semantics.
- Read-only repository behavior.
- Local repository truth.
- Compatibility with repository discovery consumers.

Changes that require modification of the existing core architecture require explicit architectural review before implementation.

New external metadata sources must be introduced through explicit provider boundaries and must not weaken the local repository metadata contract.