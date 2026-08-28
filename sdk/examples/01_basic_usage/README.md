# Basic Usage Example

This example demonstrates how to initialize the Limoxel SDK, open and index a repository workspace, inspect repository metadata and metrics, discover source files, and search for symbols.

## What It Demonstrates

1. Initializing the client with `sdk.New(sdk.WithWorkspace(path))`
2. Opening a workspace session with `client.Repository().Open(ctx, path)`
3. Querying repository quantitative statistics with `client.Repository().Statistics(ctx)`
4. Filtering and listing files using `client.Files().DiscoverFiles(ctx, filter, pagination)`
5. Performing symbol searches using `client.Search().SearchSymbols(ctx, query, pagination)`
6. Resource cleanup with `defer client.Close()`

## Running the Example

```bash
# Run against the current directory
go run ./sdk/examples/01_basic_usage

# Or run against a specific repository path
go run ./sdk/examples/01_basic_usage /path/to/repository
```

## Expected Output

The program outputs the repository metadata, total file/symbol counts, top 5 discovered Go files, and matching symbol search results.
