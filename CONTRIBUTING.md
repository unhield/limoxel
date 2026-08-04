# Contributing to Limoxel

Thank you for your interest in contributing to Limoxel.

Limoxel is an **Engineering Knowledge Infrastructure (EKI)** built with a strong emphasis on engineering quality, architectural consistency, documentation-first development, and long-term maintainability.

Every contribution should strengthen the platform while preserving the engineering foundation upon which it is built.

---

# Engineering Philosophy

Limoxel values engineering quality over contribution volume.

The objective is not to merge changes quickly, but to ensure that every accepted contribution improves the platform without compromising its architecture, engineering principles, repository organization, or long-term vision.

Engineering consistency always takes precedence over implementation convenience.

---

# Before You Contribute

Before beginning any contribution, please:

- Read the project documentation.
- Understand the engineering principles and architectural design.
- Review the repository organization.
- Familiarize yourself with existing engineering conventions.
- Verify that your proposed work aligns with the long-term direction of Limoxel.

---

# Contribution Approval

Limoxel follows a **maintainer approval** model.

While contributions are welcome, submission of a contribution does **not** guarantee that it will become part of the project.

Every contribution is reviewed for alignment with:

- Engineering Principles
- Architecture
- Repository Organization
- Engineering Contracts
- Documentation Standards
- Long-Term Project Vision
- Production Quality

A contribution may be:

- Accepted
- Accepted with requested revisions
- Deferred for a future milestone
- Declined if it conflicts with the engineering direction of the project

This review process exists to preserve the long-term quality, consistency, and maintainability of Limoxel.

---

# Discuss Before Large Contributions

If you plan to contribute a significant feature, architectural modification, repository-wide change, or any enhancement that may affect the engineering direction of the project, please discuss the proposal before beginning implementation.

Early discussion helps ensure that your effort aligns with the long-term roadmap and avoids unnecessary work.

If you are unsure whether your contribution is suitable, you are welcome to contact:

**hello.limoxel@gmail.com**

before starting development.

---

# Types of Contributions

Contributions are welcome in areas such as:

## Bug Fixes

Corrections that improve correctness, stability, reliability, or performance.

---

## Documentation

Improvements to engineering documentation, architectural specifications, implementation guidance, examples, and technical explanations.

---

## Platform Improvements

Enhancements that extend existing platform capabilities while preserving established engineering boundaries.

---

## Performance

Optimizations that improve efficiency without reducing maintainability, readability, or architectural clarity.

---

## Testing

Additional unit tests, integration tests, validation improvements, and engineering verification.

---

# Development Workflow

1. Fork the repository.
2. Create a dedicated branch.
3. Implement your changes.
4. Update documentation when applicable.
5. Execute the complete validation suite.
6. Commit using clear and meaningful commit messages.
7. Submit a Pull Request for review.

---

# Engineering Standards

Every contribution should adhere to the following principles.

- Maintain clear separation of responsibilities.
- Preserve modular package organization.
- Follow established naming conventions.
- Avoid unnecessary complexity.
- Prefer readability over cleverness.
- Maintain deterministic behavior.
- Keep documentation synchronized with implementation.
- Preserve backward compatibility whenever practical.

---

# Code Style

Limoxel follows the standard Go formatting conventions.

Before submitting changes, ensure that the repository passes the standard validation process.

```bash
go fmt ./...

go vet ./...

go test ./...

go mod tidy

go mod verify
```

---

# Documentation Requirements

Engineering documentation is considered a first-class component of the project.

Any contribution affecting:

- architecture,
- engineering contracts,
- repository organization,
- platform behavior,
- public interfaces,
- or developer workflows,

should include corresponding documentation updates within the same contribution.

Documentation and implementation should evolve together.

---

# Pull Request Guidelines

A high-quality Pull Request should:

- address a single logical objective;
- provide a clear description of the proposed change;
- include documentation updates where applicable;
- preserve repository organization;
- pass all validation checks; and
- remain consistent with the engineering standards of Limoxel.

Large, unrelated changes should be separated into multiple Pull Requests.

---

# Reporting Issues

When reporting an issue, please include:

- Operating system
- Go version
- Steps to reproduce
- Expected behavior
- Actual behavior
- Relevant logs or error messages

Clear and reproducible reports significantly improve issue resolution.

---

# Communication

For questions regarding:

- architecture,
- engineering decisions,
- repository organization,
- contribution suitability,
- or long-term project direction,

please open a GitHub Discussion or Issue.

If you are uncertain whether your proposed contribution aligns with the project, you may also contact:

**hello.limoxel@gmail.com**

before investing time in implementation.

---

# License

By submitting a contribution to Limoxel, you agree that your contribution will be licensed under the same MIT License that governs the project.

---

# Final Note

Limoxel is intended to remain a stable, production-grade engineering platform.

Every accepted contribution becomes part of that long-term foundation.

The review process exists not as a barrier to contribution, but as a commitment to preserving engineering quality, architectural consistency, and the long-term vision of the project.