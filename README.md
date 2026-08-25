<div align="center">

# Limoxel

### Engineering Knowledge Infrastructure

**A production-grade platform for repository analysis, language processing, extensibility, and engineering intelligence.**

<br>

![Release](https://img.shields.io/github/v/release/unhield/limoxel?style=for-the-badge)
![Go Version](https://img.shields.io/github/go-mod/go-version/unhield/limoxel?style=for-the-badge)
![License](https://img.shields.io/github/license/unhield/limoxel.svg?style=for-the-badge)
![CI](https://img.shields.io/github/actions/workflow/status/unhield/limoxel/ci.yml?branch=main&style=for-the-badge&label=CI)
![Documentation](https://img.shields.io/badge/Documentation-Complete-success?style=for-the-badge)

<br>

**Engineering First. Intelligence Through Foundation.**

</div>

---

> [!IMPORTANT]
>
> **Limoxel is an Engineering Knowledge Infrastructure (EKI).**
>
> It provides a stable engineering foundation for understanding software repositories through modular platform components, deterministic architecture, structured repository knowledge, and engineering intelligence.

---

<details>
<summary><strong>Table of Contents</strong></summary>

- [Overview](#overview)
- [Engineering Philosophy](#engineering-philosophy)
- [Architecture at a Glance](#architecture-at-a-glance)
- [Engineering Foundation](#engineering-foundation)
- [Core Platform Components](#core-platform-components)
- [Repository Capabilities](#repository-capabilities)
- [Engineering Intelligence](#engineering-intelligence)
- [Engineering Characteristics](#engineering-characteristics)
- [Validation](#validation)
- [Repository Organization](#repository-organization)
- [Getting Started](#getting-started)
- [Project Documentation](#project-documentation)
- [Current Release](#current-release)
- [Contributing](#contributing)
- [License](#license)
- [Acknowledgements](#acknowledgements)

</details>

---

# Overview

Software repositories are engineering systems composed of source code, project structures, documentation, programming languages, dependencies, and architectural relationships.

Limoxel establishes a unified engineering platform capable of understanding these systems through a modular, production-oriented architecture.

Rather than treating repository analysis as a collection of isolated tools, Limoxel provides a cohesive engineering foundation together with deterministic repository capabilities for discovering, indexing, analyzing, modeling, navigating, and querying software repositories.

On top of this foundation, Limoxel provides engineering intelligence for interpreting repository relationships, analyzing architecture, reasoning over structured repository knowledge, identifying engineering risks, evaluating change impact, and generating actionable engineering insights.

---

# Engineering Philosophy

The engineering philosophy of Limoxel is founded on a single principle:

> **A stable engineering foundation enables meaningful intelligence.**

The platform separates foundational engineering infrastructure from higher-level repository capabilities and intelligence.

This approach preserves architectural consistency, minimizes unnecessary coupling, and allows engineering analysis to operate on structured repository knowledge rather than bypassing established platform boundaries.

Limoxel is designed around:

- Deterministic behavior
- Explicit engineering contracts
- Structured repository knowledge
- Bounded capabilities
- Evidence-based analysis
- Stable interfaces
- Clear separation of responsibilities
- Long-term maintainability

---

# Architecture at a Glance

```text
                           User
                             │
                             ▼
                    Command-Line Interface
                             │
                             ▼
                     Engine Coordination
                             │
                             ▼
                  Platform Infrastructure
                             │
          ┌──────────────────┼──────────────────┐
          ▼                  ▼                  ▼
      Workspace /       Repository /       Language /
        Project          Filesystem          Parser /
                                           Extension
          │                  │                  │
          └──────────────────┼──────────────────┘
                             ▼
                  Repository Capabilities
                             │
        ┌────────────┬───────┼───────┬────────────┐
        ▼            ▼       ▼       ▼            ▼
    Discovery    Structure  Indexing Analysis   Search
        │            │       │       │            │
        └────────────┴───────┼───────┴────────────┘
                             ▼
                  Repository Knowledge
                             │
                             ▼
                  Engineering Intelligence
                             │
       ┌────────────┬────────┼────────┬────────────┐
       ▼            ▼        ▼        ▼            ▼
    Semantic     Navigation Analysis  Reasoning  Insights
    Analysis                & Health             & Impact
```

Repository capabilities and engineering intelligence build on the established Limoxel foundation rather than replacing its domain models or core infrastructure.

---

# Engineering Foundation

The engineering foundation of Limoxel establishes the permanent core of the platform.

It provides the production infrastructure required for repository analysis through modular engineering systems, well-defined package responsibilities, deterministic execution, and comprehensive validation.

The foundation provides the platform services required by repository capabilities and higher-level engineering intelligence.

Every production component contributes a distinct engineering responsibility and collaborates through clearly defined architectural boundaries.

---

# Core Platform Components

The engineering foundation is organized into specialized production components.

| Component | Responsibility |
|-----------|----------------|
| **CLI** | Application entry point, command processing, and execution control |
| **Engine** | Execution orchestration, workflow coordination, and processing pipeline management |
| **Workspace** | Workspace initialization and execution context management |
| **Repository** | Repository representation, repository-level coordination, and repository context |
| **Project** | Project abstraction and project lifecycle management |
| **Filesystem** | File discovery, ignore processing, filesystem abstraction, and repository traversal |
| **Language** | Programming language registration, discovery, metadata, and lifecycle management |
| **Parser** | Parser registration, parsing pipeline, execution, and parser lifecycle management |
| **Extension** | Extension registration, discovery, validation, isolation, and lifecycle management |
| **Platform** | Bootstrap, runtime, configuration, lifecycle, logging, registry, events, and shared infrastructure |

---

# Repository Capabilities

Limoxel provides bounded repository-analysis capabilities that build on the engineering foundation.

| Capability | Responsibility |
|------------|----------------|
| **Repository Discovery** | Establish repository boundaries, discover files, apply exclusion rules, identify languages, and collect deterministic metadata |
| **Repository Structure** | Represent directories, packages, modules, configuration, documentation, and repository organization |
| **Dependency Analysis** | Model internal and external dependency relationships and dependency graphs |
| **Source Indexing** | Index source files, packages, file relationships, metadata, and repository statistics |
| **AST & Symbol Analysis** | Parse supported source code and expose structured symbols and symbol relationships |
| **Cross-Reference Analysis** | Resolve repository relationships such as references, callers, callees, implementations, and dependencies |
| **Repository Knowledge Graph** | Represent repository entities and relationships as a unified graph with traversal and query support |
| **Repository Query APIs** | Provide stable internal services for repository, symbol, graph, and search operations |
| **Repository Search** | Search repository symbols, files, packages, documentation, configuration, and supported fuzzy matches |

These capabilities provide the structured repository knowledge consumed by higher-level engineering intelligence.

---

# Engineering Intelligence

Limoxel extends repository analysis with deterministic engineering intelligence built on structured repository knowledge.

The intelligence layer interprets repository entities and relationships to provide deeper engineering understanding without requiring probabilistic AI systems.

## Semantic Intelligence

Semantic intelligence provides structured understanding of repository entities beyond their raw syntax.

It includes:

- Repository semantic models
- Package semantic models
- Symbol semantic models
- Type resolution
- Interface resolution
- Function and variable semantics
- Generic type handling
- Scope resolution
- Symbol ownership
- Symbol visibility
- Semantic validation

---

## Cross-Repository Intelligence

Limoxel can analyze relationships across repository boundaries and workspace structures.

It includes:

- Cross-file analysis
- Cross-package analysis
- Cross-module analysis
- Workspace relationships
- Shared dependencies
- Shared configuration
- Package contracts
- Internal and public API relationships
- Repository evolution analysis

---

## Engineering Navigation

Engineering navigation provides deterministic traversal across repository assets and relationships.

It includes:

- Definition navigation
- Declaration navigation
- Implementation navigation
- Reference lookup
- Usage lookup
- Reverse dependency lookup
- Symbol hierarchy
- Type hierarchy
- Interface hierarchy
- Package hierarchy
- Call hierarchy
- Dependency-chain traversal

---

## Engineering Analysis

Limoxel analyzes repository structure and engineering quality through structured repository relationships.

Analysis capabilities include:

- Code quality analysis
- Dead-code detection
- Unused import and export analysis
- Duplicate logic detection
- Large-file and large-function analysis
- Dependency analysis
- Circular dependency detection
- Layer violation detection
- Coupling analysis
- Architecture analysis
- Module boundary analysis
- Layer consistency analysis
- Package cohesion analysis
- Configuration analysis
- Repository health analysis

Repository health can incorporate engineering, architecture, documentation, testing, and maintainability characteristics.

---

## Knowledge Graph Intelligence

The repository knowledge graph provides a structured model for engineering reasoning.

The intelligence layer enriches the graph with:

- Semantic relationships
- Ownership relationships
- Dependency relationships
- Documentation relationships
- Configuration relationships

Graph reasoning supports:

- Relationship inference
- Dependency inference
- Ownership inference
- Architecture inference
- Context generation
- Knowledge consistency validation
- Relationship validation
- Graph completeness validation

Repository context can be generated at multiple levels, including:

- Repository
- Package
- Symbol
- Module
- Architecture

Engineering insights can cover:

- Complexity
- Dependencies
- Architecture
- Repository growth
- Engineering risk

---

## Deterministic Reasoning

Limoxel provides deterministic engineering reasoning over structured repository knowledge.

Reasoning capabilities include:

- Change-impact analysis
- Symbol impact analysis
- Package impact analysis
- Module impact analysis
- Repository impact analysis
- Dependency impact analysis
- Refactoring analysis
- Safe rename analysis
- Safe move analysis
- Safe extraction analysis
- Safe deletion analysis
- Refactoring risk assessment
- Breaking-change detection
- API change analysis
- Package change analysis
- Symbol removal analysis
- Interface change analysis
- Version compatibility analysis
- Engineering recommendations

Recommendations can address:

- Dependencies
- Architecture
- Performance
- Repository organization
- Engineering quality

The reasoning system is designed to produce deterministic results from available repository evidence.

---

# Engineering Characteristics

<table>
<tr>
<td width="50%" valign="top">

### Architecture

- Modular engineering design
- Deterministic package boundaries
- Clear separation of concerns
- Layered platform architecture
- Extensible engineering model
- Stable engineering contracts
- Bounded repository capabilities
- Structured intelligence layer

</td>

<td width="50%" valign="top">

### Engineering Quality

- Documentation-first development
- Enterprise repository organization
- Comprehensive validation strategy
- Runtime verification
- Architecture verification
- Deterministic repository analysis
- Structured engineering reasoning
- Stable API contracts

</td>
</tr>
</table>

---

# Validation

Limoxel is validated through automated testing, static analysis, build verification, repository-level checks, and capability-specific validation.

Validation covers:

- Unit testing
- Integration testing
- Build verification
- Static analysis
- Dependency verification
- Runtime behavior
- Deterministic execution
- Repository integration
- Cross-package integration
- Cross-module integration
- Workspace integration
- API integration
- Performance characteristics
- Graph traversal
- Search behavior
- Repository isolation
- Input validation
- Graph integrity
- Resource-exhaustion handling
- Error handling
- Documentation integrity

Engineering intelligence is additionally validated across:

- Semantic analysis
- Navigation
- Engineering analysis
- Knowledge graph operations
- Deterministic reasoning
- Change-impact analysis
- Recommendation behavior
- Regression scenarios
- False-positive scenarios
- False-negative scenarios
- Performance benchmarks

---

> [!NOTE]
>
> Limoxel's engineering foundation is designed to remain stable while repository capabilities and intelligence operate through clearly defined boundaries.

---

# Repository Organization

The Limoxel repository follows a responsibility-oriented organizational model.

Production implementation, engineering documentation, executable entry points, repository-level testing, and repository metadata remain physically separated to preserve clarity, maintainability, and long-term scalability.

```text
limoxel/
│
├── .github/                 # Repository automation & workflows
├── cmd/                     # Executable entry points
├── docs/                    # Engineering documentation
├── internal/                # Internal implementation
│   └── capabilities/        # Repository and intelligence capabilities
├── tests/                   # Testing infrastructure
│
├── CHANGELOG.md             # Release history
├── CODEOWNERS               # Repository ownership
├── CODE_OF_CONDUCT.md       # Community guidelines
├── CONTRIBUTING.md          # Contribution guide
├── LICENSE                  # MIT License
├── README.md                # Project overview
├── SECURITY.md              # Security policy
├── go.mod                   # Go module definition
└── go.sum                   # Dependency checksums
```

Engineering documentation is maintained under:

```text
docs/
├── 01_foundation/
├── 02_architecture/
├── 03_engineering/
├── 04_repository/
└── 05_intelligence/
```

The complete repository organization, directory responsibilities, package responsibilities, engineering conventions, and repository governance are documented in:

`docs/01_foundation/12_Repository_Structure.md`

---

# Getting Started

Limoxel is built using the Go programming language and follows the standard Go module workflow.

The repository is organized to provide a deterministic development experience with minimal setup requirements.

## Prerequisites

| Requirement | Version |
|-------------|---------|
| Go | 1.26.5 or later |
| Git | Latest stable release |
| Operating System | Linux, macOS, or Windows |

---

## Clone the Repository

Clone the repository and enter the project directory.

```bash
git clone https://github.com/unhield/limoxel.git

cd limoxel
```

---

## Install Dependencies

```bash
go mod tidy
go mod verify
```

---

## Build

Compile the production executable:

```bash
go build ./...
```

Or build the executable directly:

```bash
go build -o limoxel ./cmd/limoxel
```

---

## Run

Execute Limoxel using the production entry point:

```bash
go run ./cmd/limoxel
```

Or execute the compiled binary:

```bash
./limoxel
```

---

# Validation

Run the standard repository validation commands:

```bash
go test ./...
go vet ./...
go build ./...
go mod tidy
go mod verify
```

For capability-specific validation, use the procedures documented within the relevant engineering documentation.

---

# Project Documentation

Engineering documentation is maintained as a first-class artifact of the repository.

Documentation is organized into dedicated engineering categories.

| Documentation | Purpose |
|---------------|---------|
| **Foundation** | Engineering principles, repository organization, platform philosophy, and permanent engineering contracts |
| **Architecture** | Component boundaries, dependency rules, package structure, infrastructure, runtime, communication, and system design |
| **Engineering** | Implementation specifications, engineering contracts, validation, development standards, and operational guidance |
| **Repository** | Repository discovery, structure, dependency, indexing, AST, symbols, cross-reference, graph, search, API, and repository-analysis specifications |
| **Intelligence** | Semantic intelligence, cross-repository analysis, engineering navigation, engineering analysis, knowledge graph intelligence, deterministic reasoning, impact analysis, and recommendations |

Complete documentation is available under:

```text
docs/
├── 01_foundation/
├── 02_architecture/
├── 03_engineering/
├── 04_repository/
└── 05_intelligence/
```

---

# Current Release

Limoxel provides a production engineering platform combining repository analysis with deterministic engineering intelligence.

| Capability | Status |
|------------|:------:|
| Engineering Foundation | ✅ |
| Platform Infrastructure | ✅ |
| Repository Capabilities | ✅ |
| Semantic Intelligence | ✅ |
| Cross-Repository Intelligence | ✅ |
| Engineering Navigation | ✅ |
| Engineering Analysis | ✅ |
| Knowledge Graph Intelligence | ✅ |
| Deterministic Reasoning | ✅ |
| Change-Impact Analysis | ✅ |
| Refactoring Analysis | ✅ |
| Breaking-Change Detection | ✅ |
| Engineering Recommendations | ✅ |
| Engineering Documentation | ✅ |
| Production Validation | ✅ |

Limoxel provides structured repository understanding across discovery, indexing, dependency analysis, AST and symbol analysis, cross-reference analysis, knowledge graph construction, semantic analysis, navigation, engineering analysis, deterministic reasoning, change-impact analysis, and engineering recommendations.

---

# Contributing

Contributions are welcome and appreciated.

Limoxel follows a **maintainer approval** model to preserve engineering quality, architectural consistency, repository organization, and long-term maintainability.

Every contribution is reviewed according to the project's engineering principles, engineering standards, documentation quality, and long-term direction.

Submission of an Issue, Pull Request, or proposed enhancement does **not** guarantee acceptance.

Before contributing, please read:

- `CONTRIBUTING.md`
- `CODE_OF_CONDUCT.md`

If you plan to contribute a significant feature, architectural change, repository-wide improvement, or are unsure whether your proposal aligns with the engineering direction of Limoxel, please contact:

**hello.limoxel@gmail.com**

before investing significant implementation effort.

---

## ❤️ Support Limoxel

Limoxel is developed and maintained as an independent open-source project.

If Limoxel saves you time, supports your research, or becomes part of your engineering workflow, consider supporting its continued development.

[![Sponsor](https://img.shields.io/badge/Sponsor-Buy%20Me%20A%20Coffee-FFDD00?logo=buymeacoffee&logoColor=000000)](https://buymeacoffee.com/limoxel)

---

# License

Limoxel is released under the **MIT License**.

See the `LICENSE` file for the complete license text.

---

# Acknowledgements

Limoxel represents a long-term engineering initiative focused on building a stable and extensible foundation for repository understanding.

The project emphasizes engineering quality, architectural discipline, documentation-first development, deterministic analysis, structured repository knowledge, and maintainable engineering intelligence.

<div align="center">

---

### Engineering First. Intelligence Through Foundation.

*Building structured engineering intelligence on a permanent engineering foundation.*

</div>
