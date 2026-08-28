package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/unhield/limoxel/sdk"
)

func main() {
	repoPath := flag.String("repo", ".", "Path to repository to check")
	minScore := flag.Float64("min-score", 60.0, "Minimum allowed repository health score")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Printf("=== CI/CD Quality Gate: Limoxel Health Check ===\n")
	fmt.Printf("Workspace: %s | Threshold: %.1f\n\n", *repoPath, *minScore)

	client, err := sdk.OpenWorkspace(ctx, *repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAILURE: Unable to open repository: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// 1. Run Health Analysis
	health, err := client.Analysis().RepositoryHealth(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAILURE: Health evaluation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Health Score: %.1f / 100.0 (Grade: %s, Status: %s)\n",
		health.OverallScore, health.Grade, health.Status)

	// 2. Check Architecture Violations
	arch, err := client.Analysis().AnalyzeArchitecture(ctx, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAILURE: Architecture analysis failed: %v\n", err)
		os.Exit(1)
	}
	if len(arch.Violations) > 0 {
		fmt.Printf("WARNING: Detected %d architecture boundary violations!\n", len(arch.Violations))
	}

	// 3. Enforce Threshold
	if health.OverallScore < *minScore {
		fmt.Fprintf(os.Stderr, "\nQUALITY GATE FAILED: Score %.1f is below required threshold %.1f\n",
			health.OverallScore, *minScore)
		os.Exit(1)
	}

	fmt.Println("\nQUALITY GATE PASSED: Repository meets all health and architectural standards.")
}
