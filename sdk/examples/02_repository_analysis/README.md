# Repository Analysis Example

This example demonstrates how to use the Limoxel Analysis SDK to perform deep repository health assessments, analyze architecture layer boundaries, detect circular dependencies, and compute code quality metrics.

## What It Demonstrates

1. Evaluating multidimensional repository health with `client.Analysis().RepositoryHealth(ctx)`
2. Inspecting architectural layer boundaries and violation findings with `client.Analysis().AnalyzeArchitecture(ctx)`
3. Detecting direct, transitive, and circular dependencies with `client.Analysis().AnalyzeDependencies(ctx)`
4. Calculating maintainability, complexity, and testability scores with `client.Analysis().CodeQuality(ctx)`

## Running the Example

```bash
# Run against current workspace
go run ./sdk/examples/02_repository_analysis

# Run against custom target codebase
go run ./sdk/examples/02_repository_analysis /path/to/project
```

## Expected Output

The program displays an overall health scorecard, dimension breakdown, architecture boundary compliance, circular dependency checks, and quality metrics.
