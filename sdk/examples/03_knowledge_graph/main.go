package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/unhield/limoxel/sdk"
)

func main() {
	targetWorkspace := "."
	if len(os.Args) > 1 {
		targetWorkspace = os.Args[1]
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	fmt.Printf("=== Limoxel Knowledge Graph Explorer ===\n")
	fmt.Printf("Workspace: %s\n\n", targetWorkspace)

	client, err := sdk.OpenWorkspace(ctx, targetWorkspace)
	if err != nil {
		log.Fatalf("Failed to open workspace: %v", err)
	}
	defer client.Close()

	// 1. Export Graph to Mermaid Format
	fmt.Printf("--- 1. Mermaid Graph Export ---\n")
	mermaidResult, err := client.Graph().ExportGraph(ctx, sdk.GraphFilter{MaxDepth: 2}, sdk.ExportFormatMermaid)
	if err != nil {
		log.Fatalf("Failed to export Mermaid graph: %v", err)
	}
	fmt.Printf("Exported %d nodes and %d edges.\n", mermaidResult.NodeCount, mermaidResult.EdgeCount)
	fmt.Printf("Mermaid Preview (first 10 lines):\n")
	lines := splitLines(mermaidResult.Content, 10)
	for _, l := range lines {
		fmt.Println(l)
	}

	// 2. Export Graph to Graphviz DOT Format
	fmt.Printf("\n--- 2. Graphviz DOT Export ---\n")
	dotResult, err := client.Graph().ExportGraph(ctx, sdk.GraphFilter{MaxDepth: 2}, sdk.ExportFormatGraphviz)
	if err != nil {
		log.Fatalf("Failed to export DOT graph: %v", err)
	}
	fmt.Printf("Exported DOT graph (%d nodes, %d edges).\n", dotResult.NodeCount, dotResult.EdgeCount)

	// 3. Traverse Knowledge Graph from Root/Package
	fmt.Printf("\n--- 3. Graph Traversal ---\n")
	nodes, err := client.Graph().TraverseNodes(ctx, "root", sdk.GraphFilter{MaxDepth: 2})
	if err != nil {
		log.Printf("Traversal info: %v (workspace root node used)", err)
	} else {
		fmt.Printf("Discovered %d reachable nodes.\n", len(nodes))
		for i, n := range nodes {
			if i >= 5 {
				break
			}
			fmt.Printf("  [%d] Node ID: %s | Kind: %s | Name: %s\n", i+1, n.ID, n.Kind, n.Name)
		}
	}
}

func splitLines(content string, max int) []string {
	var out []string
	curr := ""
	for i := 0; i < len(content); i++ {
		if content[i] == '\n' {
			out = append(out, curr)
			curr = ""
			if len(out) >= max {
				return out
			}
		} else if content[i] != '\r' {
			curr += string(content[i])
		}
	}
	if curr != "" && len(out) < max {
		out = append(out, curr)
	}
	return out
}
