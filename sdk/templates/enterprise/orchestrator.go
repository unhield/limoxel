package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/unhield/limoxel/sdk"
)

// RepoAuditSummary encapsulates analysis findings for an audited workspace.
type RepoAuditSummary struct {
	Path        string  `json:"path"`
	Name        string  `json:"name"`
	HealthScore float64 `json:"health_score"`
	Grade       string  `json:"grade"`
	Files       int     `json:"files"`
	Symbols     int     `json:"symbols"`
	Error       string  `json:"error,omitempty"`
}

// EnterpriseOrchestrator manages concurrent analysis across multiple repository workspaces.
type EnterpriseOrchestrator struct {
	concurrency int
}

// NewEnterpriseOrchestrator constructs an orchestrator with a worker pool limit.
func NewEnterpriseOrchestrator(concurrency int) *EnterpriseOrchestrator {
	if concurrency <= 0 {
		concurrency = 4
	}
	return &EnterpriseOrchestrator{
		concurrency: concurrency,
	}
}

// AuditRepositories analyzes multiple repositories concurrently.
func (o *EnterpriseOrchestrator) AuditRepositories(ctx context.Context, repoPaths []string) []RepoAuditSummary {
	var (
		mu      sync.Mutex
		results []RepoAuditSummary
		sem     = make(chan struct{}, o.concurrency)
		wg      sync.WaitGroup
	)

	for _, path := range repoPaths {
		wg.Add(1)
		sem <- struct{}{}

		go func(p string) {
			defer wg.Done()
			defer func() { <-sem }()

			summary := o.auditSingle(ctx, p)

			mu.Lock()
			results = append(results, summary)
			mu.Unlock()
		}(path)
	}

	wg.Wait()
	return results
}

func (o *EnterpriseOrchestrator) auditSingle(ctx context.Context, path string) RepoAuditSummary {
	subCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	client, err := sdk.OpenWorkspace(subCtx, path)
	if err != nil {
		return RepoAuditSummary{Path: path, Error: err.Error()}
	}
	defer client.Close()

	info, err := client.Repository().Info(subCtx)
	name := path
	if err == nil && info.Name != "" {
		name = info.Name
	}

	stats, _ := client.Repository().Statistics(subCtx)
	health, _ := client.Analysis().RepositoryHealth(subCtx)

	score := 0.0
	grade := "N/A"
	if health != nil {
		score = health.OverallScore
		grade = health.Grade
	}

	totalFiles := 0
	totalSymbols := 0
	if stats != nil {
		totalFiles = stats.TotalFiles
		totalSymbols = stats.TotalSymbols
	}

	return RepoAuditSummary{
		Path:        path,
		Name:        name,
		HealthScore: score,
		Grade:       grade,
		Files:       totalFiles,
		Symbols:     totalSymbols,
	}
}

func main() {
	repositories := []string{"."}

	fmt.Printf("=== Enterprise Multi-Repository Orchestrator ===\n")
	fmt.Printf("Auditing %d repositories...\n\n", len(repositories))

	orch := NewEnterpriseOrchestrator(2)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	results := orch.AuditRepositories(ctx, repositories)

	fmt.Printf("%-20s | %-12s | %-8s | %-8s | %-8s\n", "Repository", "Health Score", "Grade", "Files", "Symbols")
	fmt.Println("----------------------------------------------------------------------")
	for _, r := range results {
		if r.Error != "" {
			fmt.Printf("%-20s | ERROR: %s\n", r.Path, r.Error)
		} else {
			fmt.Printf("%-20s | %11.1f  | %-8s | %-8d | %-8d\n", r.Name, r.HealthScore, r.Grade, r.Files, r.Symbols)
		}
	}
}
