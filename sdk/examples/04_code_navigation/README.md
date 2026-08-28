# Code Navigation Example

This example demonstrates how to use the Limoxel Navigation SDK to resolve symbol definitions, locate call sites and references across the codebase, and compute incoming and outgoing call hierarchies.

## What It Demonstrates

1. Resolving declaration sites with `client.Navigation().GoToDefinition(ctx, symbolID)`
2. Finding usage references with `client.Navigation().FindReferences(ctx, symbolID, includeDeclarations)`
3. Querying caller and callee networks with `client.Navigation().CallHierarchy(ctx, symbolID)`
4. Contextual enrichment with `client.Navigation().EnrichContext(ctx, symbolID)`

## Running the Example

```bash
# Run against current workspace
go run ./sdk/examples/04_code_navigation

# Run against custom target codebase
go run ./sdk/examples/04_code_navigation /path/to/project
```

## Expected Output

Outputs target definition file coordinates, reference occurrences, and incoming/outgoing call hierarchy trees.
