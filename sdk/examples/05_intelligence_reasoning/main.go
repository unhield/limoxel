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

	fmt.Printf("=== Limoxel Engineering Intelligence & Reasoning ===\n")
	fmt.Printf("Workspace: %s\n\n", targetWorkspace)

	client, err := sdk.OpenWorkspace(ctx, targetWorkspace)
	if err != nil {
		log.Fatalf("Failed to open workspace: %v", err)
	}
	defer client.Close()

	// 1. Discover Repository Insights
	insights, err := client.Reasoning().EngineeringInsights(ctx)
	if err != nil {
		log.Printf("EngineeringInsights note: %v", err)
	} else {
		fmt.Printf("--- 1. Repository Engineering Insights (%d) ---\n", len(insights))
		for i, ins := range insights {
			if i >= 5 {
				break
			}
			fmt.Printf("  [%d] %s: %s (Severity: %s)\n", i+1, ins.Title, ins.Description, ins.Severity)
		}
		fmt.Println()
	}

	// 2. Perform Blast-Radius Impact Analysis on a target package or symbol
	targetEntity := "main"
	fmt.Printf("--- 2. Blast-Radius Change Impact Analysis for %q ---\n", targetEntity)
	impact, err := client.Reasoning().AnalyzeImpact(ctx, targetEntity)
	if err != nil {
		log.Printf("AnalyzeImpact note: %v", err)
	} else if impact != nil {
		fmt.Printf("Target Entity:        %s\n", impact.TargetEntity)
		fmt.Printf("Risk Level:           %s\n", impact.RiskLevel)
		fmt.Printf("Impact Score:         %.2f\n", impact.ImpactScore)
		fmt.Printf("Directly Impacted:    %d entities\n", len(impact.DirectlyImpacted))
		for _, dep := range impact.DirectlyImpacted {
			fmt.Printf("  -> %s\n", dep)
		}
		fmt.Printf("Indirectly Impacted:  %d entities\n\n", len(impact.IndirectlyImpacted))
	}

	// 3. Assess Breaking Change Risks
	fmt.Printf("--- 3. Breaking Change Risk Assessment for %q ---\n", targetEntity)
	breaking, err := client.Reasoning().AnalyzeBreakingChanges(ctx, targetEntity)
	if err != nil {
		log.Printf("AnalyzeBreakingChanges note: %v", err)
	} else if breaking != nil {
		fmt.Printf("Has Breaking Changes: %t\n", breaking.HasBreakingChanges)
		fmt.Printf("Severity:             %s\n", breaking.Severity)
		if len(breaking.MigrationAdvice) > 0 {
			fmt.Printf("Migration Advice:\n")
			for _, adv := range breaking.MigrationAdvice {
				fmt.Printf("  • %s\n", adv)
			}
		}
		fmt.Println()
	}

	// 4. Refactoring Advice
	fmt.Printf("--- 4. Refactoring Safety Guidance for %q ---\n", targetEntity)
	advice, err := client.Reasoning().RefactoringAdvice(ctx, targetEntity, "NewName")
	if err != nil {
		log.Printf("RefactoringAdvice note: %v", err)
	} else if advice != nil {
		fmt.Printf("Operation:  %s\n", advice.Operation)
		fmt.Printf("Is Safe:    %t\n", advice.IsSafe)
		if len(advice.Risks) > 0 {
			fmt.Printf("Risks Identified:\n")
			for _, r := range advice.Risks {
				fmt.Printf("  ! %s\n", r)
			}
		}
	}
}
