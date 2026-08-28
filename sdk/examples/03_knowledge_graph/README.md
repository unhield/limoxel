# Knowledge Graph Example

This example demonstrates how to interact with the Limoxel Knowledge Graph SDK to traverse entity relationships, find paths between nodes, and export knowledge graphs into Mermaid and Graphviz DOT formats.

## What It Demonstrates

1. Exporting structured graph relationships into Mermaid syntax with `client.Graph().Export(ctx, filter, sdk.ExportFormatMermaid)`
2. Exporting digraphs into Graphviz DOT syntax with `client.Graph().Export(ctx, filter, sdk.ExportFormatGraphviz)`
3. Breadth-first graph traversal using `client.Graph().Traverse(ctx, startNodeID, filter)`
4. Inspecting graph nodes, properties, and relationship evidence

## Running the Example

```bash
# Run against current workspace
go run ./sdk/examples/03_knowledge_graph

# Run against custom target codebase
go run ./sdk/examples/03_knowledge_graph /path/to/project
```

## Expected Output

Outputs node and relationship metrics, preview of Mermaid flowchart syntax, and traversed entity summaries.
