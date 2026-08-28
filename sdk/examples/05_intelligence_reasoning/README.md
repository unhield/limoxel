# Intelligence & Reasoning Example

This example demonstrates how to use the Limoxel Reasoning SDK to compute blast-radius change impact analysis, evaluate semantic breaking changes, obtain refactoring safety guidance, and discover repository-level engineering insights.

## What It Demonstrates

1. Querying systemic engineering insights with `client.Reasoning().RepositoryInsights(ctx)`
2. Computing blast-radius entity impacts with `client.Reasoning().AnalyzeImpact(ctx, targetEntityID)`
3. Assessing semantic breaking changes and migration advice with `client.Reasoning().AssessBreakingChanges(ctx, targetEntityID)`
4. Evaluating automated refactoring safety with `client.Reasoning().RefactoringAdvice(ctx, operation, targetEntityID)`

## Running the Example

```bash
# Run against current workspace
go run ./sdk/examples/05_intelligence_reasoning

# Run against custom target codebase
go run ./sdk/examples/05_intelligence_reasoning /path/to/project
```

## Expected Output

Outputs engineering insights, blast-radius risk scores, breaking change severity, and refactoring risk evaluations.
