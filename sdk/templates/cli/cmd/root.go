package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/unhield/limoxel/sdk"
)

// Execute runs the CLI root command with flag parsing.
func Execute() error {
	fs := flag.NewFlagSet("custom-tool", flag.ExitOnError)
	repoPath := fs.String("repo", ".", "Path to repository root")
	action := fs.String("action", "inspect", "Action to perform: inspect, health, graph")
	timeoutSec := fs.Int("timeout", 30, "Command timeout in seconds")

	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeoutSec)*time.Second)
	defer cancel()

	client, err := sdk.OpenWorkspace(ctx, *repoPath)
	if err != nil {
		return fmt.Errorf("failed to open workspace at %q: %w", *repoPath, err)
	}
	defer client.Close()

	switch *action {
	case "inspect":
		info, err := client.Repository().Info(ctx)
		if err != nil {
			return err
		}
		stats, err := client.Repository().Statistics(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("Repository: %s (Root: %s)\n", info.Name, info.RootPath)
		fmt.Printf("Files: %d | Packages: %d | Symbols: %d\n", stats.TotalFiles, stats.TotalPackages, stats.TotalSymbols)

	case "health":
		health, err := client.Analysis().RepositoryHealth(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("Health Score: %.1f / 100.0 (Grade: %s, Status: %s)\n", health.OverallScore, health.Grade, health.Status)

	case "graph":
		mermaid, err := client.Graph().ExportGraph(ctx, sdk.GraphFilter{MaxDepth: 1}, sdk.ExportFormatMermaid)
		if err != nil {
			return err
		}
		fmt.Printf("Exported Graph: %d nodes, %d edges\n", mermaid.NodeCount, mermaid.EdgeCount)

	default:
		return fmt.Errorf("unknown action %q; supported: inspect, health, graph", *action)
	}

	return nil
}
