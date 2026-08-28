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

	fmt.Printf("=== Limoxel Repository Analysis Engine ===\n")
	fmt.Printf("Analyzing workspace: %s\n\n", targetWorkspace)

	// Initialize SDK and open workspace
	client, err := sdk.OpenWorkspace(ctx, targetWorkspace)
	if err != nil {
		log.Fatalf("Failed to open workspace: %v", err)
	}
	defer client.Close()

	// 1. Evaluate Overall Repository Health
	health, err := client.Analysis().RepositoryHealth(ctx)
	if err != nil {
		log.Fatalf("Failed to evaluate repository health: %v", err)
	}
	fmt.Printf("--- Overall Repository Health ---\n")
	fmt.Printf("Health Score:  %.1f / 100.0\n", health.OverallScore)
	fmt.Printf("Health Grade:  %s\n", health.Grade)
	fmt.Printf("Status:        %s\n\n", health.Status)

	fmt.Printf("--- Health Dimensions ---\n")
	for _, dim := range health.Dimensions {
		fmt.Printf("• %-20s: Score %5.1f | Confidence %4.2f | Coverage %4.2f\n", dim.Name, dim.Score, dim.Confidence, dim.Coverage)
	}

	// 2. Analyze Architecture Boundaries & Layers
	arch, err := client.Analysis().AnalyzeArchitecture(ctx, "")
	if err != nil {
		log.Fatalf("Failed to analyze architecture: %v", err)
	}
	fmt.Printf("\n--- Architecture Analysis ---\n")
	fmt.Printf("Total Components:    %d\n", arch.TotalComponents)
	fmt.Printf("Layer Count:         %d\n", arch.LayerCount)
	fmt.Printf("Boundary Violations: %d\n", len(arch.Violations))
	for i, v := range arch.Violations {
		fmt.Printf("  [%d] %s: %s (Severity: %s)\n", i+1, v.RuleID, v.Message, v.Severity)
	}

	// 3. Analyze Dependencies & Circular Graphs
	deps, err := client.Analysis().AnalyzeDependencies(ctx, "")
	if err != nil {
		log.Fatalf("Failed to analyze dependencies: %v", err)
	}
	fmt.Printf("\n--- Dependency Analysis ---\n")
	fmt.Printf("Total Dependencies:      %d\n", deps.TotalDependencies)
	fmt.Printf("Direct Dependencies:     %d\n", deps.DirectDependencies)
	fmt.Printf("Transitive Dependencies: %d\n", deps.TransitiveDependencies)
	fmt.Printf("Circular Dependencies:   %d\n", deps.CircularDependencies)

	// 4. Evaluate Code Quality Metrics
	quality, err := client.Analysis().AnalyzeQuality(ctx, "")
	if err != nil {
		log.Fatalf("Failed to evaluate code quality: %v", err)
	}
	fmt.Printf("\n--- Code Quality Metrics ---\n")
	fmt.Printf("Maintainability Score: %.1f / 100.0\n", quality.MaintainabilityScore)
	fmt.Printf("Complexity Score:      %.1f / 100.0\n", quality.ComplexityScore)
	fmt.Printf("Testability Score:     %.1f / 100.0\n", quality.TestabilityScore)
	fmt.Printf("Total Quality Issues:  %d\n", quality.TotalIssues)
}
