package sdk_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/unhield/limoxel/sdk"
)

func createPerfSampleRepo(t testing.TB) string {
	dir := t.TempDir()
	repoPath := filepath.Join(dir, "perf_repo")
	if err := os.MkdirAll(filepath.Join(repoPath, "pkg", "service"), 0755); err != nil {
		t.Fatalf("failed to create sample repo dirs: %v", err)
	}

	mainFile := `package main

import "perf_repo/pkg/service"

func main() {
	service.Execute()
}
`
	serviceFile := `package service

// Execute runs the service handler.
func Execute() string {
	return "service-executed"
}
`
	if err := os.WriteFile(filepath.Join(repoPath, "main.go"), []byte(mainFile), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, "pkg", "service", "service.go"), []byte(serviceFile), 0644); err != nil {
		t.Fatalf("failed to write service.go: %v", err)
	}
	return repoPath
}

func BenchmarkOpenWorkspace(b *testing.B) {
	ctx := context.Background()
	repoPath := createPerfSampleRepo(b)

	b.ReportAllocs()

	for b.Loop() {
		client, err := sdk.OpenWorkspace(ctx, repoPath)
		if err != nil {
			b.Fatalf("OpenWorkspace failed: %v", err)
		}
		_ = client.Close()
	}
}

func BenchmarkSearchSymbols(b *testing.B) {
	ctx := context.Background()
	repoPath := createPerfSampleRepo(b)
	client, err := sdk.OpenWorkspace(ctx, repoPath)
	if err != nil {
		b.Fatalf("OpenWorkspace failed: %v", err)
	}
	defer client.Close()

	opts := sdk.PaginationOptions{Limit: 10}
	b.ReportAllocs()

	for b.Loop() {
		res, err := client.Search().SearchSymbols(ctx, "Execute", opts)
		if err != nil || res == nil {
			b.Fatalf("SearchSymbols failed: %v", err)
		}
	}
}

func BenchmarkRepositoryHealth(b *testing.B) {
	ctx := context.Background()
	repoPath := createPerfSampleRepo(b)
	client, err := sdk.OpenWorkspace(ctx, repoPath)
	if err != nil {
		b.Fatalf("OpenWorkspace failed: %v", err)
	}
	defer client.Close()

	b.ReportAllocs()

	for b.Loop() {
		report, err := client.Analysis().RepositoryHealth(ctx)
		if err != nil || report == nil {
			b.Fatalf("RepositoryHealth failed: %v", err)
		}
	}
}

func BenchmarkGraphExportMermaid(b *testing.B) {
	ctx := context.Background()
	repoPath := createPerfSampleRepo(b)
	client, err := sdk.OpenWorkspace(ctx, repoPath)
	if err != nil {
		b.Fatalf("OpenWorkspace failed: %v", err)
	}
	defer client.Close()

	filter := sdk.GraphFilter{MaxDepth: 2}
	b.ReportAllocs()

	for b.Loop() {
		res, err := client.Graph().ExportGraph(ctx, filter, sdk.ExportFormatMermaid)
		if err != nil || res == nil {
			b.Fatalf("ExportGraph failed: %v", err)
		}
	}
}

func TestPerformance_ConcurrentConsumers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	repoPath := createPerfSampleRepo(t)
	client, err := sdk.OpenWorkspace(ctx, repoPath)
	if err != nil {
		t.Fatalf("OpenWorkspace failed: %v", err)
	}
	defer client.Close()

	concurrency := 16
	iterations := 20
	var wg sync.WaitGroup

	for worker := 0; worker < concurrency; worker++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for iter := 0; iter < iterations; iter++ {
				// 1. Query Statistics
				stats, err := client.Repository().Statistics(ctx)
				if err != nil || stats == nil {
					t.Errorf("worker %d failed stats query: %v", id, err)
				}

				// 2. Search Symbols
				searchRes, err := client.Search().SearchSymbols(ctx, "Execute", sdk.PaginationOptions{Limit: 5})
				if err != nil || searchRes == nil {
					t.Errorf("worker %d failed search query: %v", id, err)
				}

				// 3. Health Analysis
				health, err := client.Analysis().RepositoryHealth(ctx)
				if err != nil || health == nil {
					t.Errorf("worker %d failed health query: %v", id, err)
				}

				// 4. Graph Export
				graphRes, err := client.Graph().ExportGraph(ctx, sdk.GraphFilter{MaxDepth: 1}, sdk.ExportFormatMermaid)
				if err != nil || graphRes == nil {
					t.Errorf("worker %d failed graph export: %v", id, err)
				}
			}
		}(worker)
	}

	wg.Wait()
}

func TestPerformance_MemoryAllocationBounds(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	repoPath := createPerfSampleRepo(t)
	client, err := sdk.OpenWorkspace(ctx, repoPath)
	if err != nil {
		t.Fatalf("OpenWorkspace failed: %v", err)
	}
	defer client.Close()

	// Measure baseline heap allocation
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	// Execute 200 repeated query cycles
	for i := 0; i < 200; i++ {
		_, _ = client.Repository().Statistics(ctx)
		_, _ = client.Search().SearchSymbols(ctx, "Execute", sdk.PaginationOptions{Limit: 5})
		_, _ = client.Analysis().RepositoryHealth(ctx)
	}

	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	// Heap growth should be bounded
	heapDelta := int64(m2.HeapAlloc) - int64(m1.HeapAlloc)
	t.Logf("Heap memory delta after 200 query cycles: %d bytes (%.2f KB)", heapDelta, float64(heapDelta)/1024.0)

	// Growth should not exceed 50 MB for sample workload
	if heapDelta > 50*1024*1024 {
		t.Errorf("excessive memory growth detected: %d bytes", heapDelta)
	}
}

func TestPerformance_ScalabilityUnderRepeatedQueries(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	repoPath := createPerfSampleRepo(t)
	client, err := sdk.OpenWorkspace(ctx, repoPath)
	if err != nil {
		t.Fatalf("OpenWorkspace failed: %v", err)
	}
	defer client.Close()

	start := time.Now()
	const queryCount = 500
	for i := 0; i < queryCount; i++ {
		res, err := client.Search().SearchSymbols(ctx, "Execute", sdk.PaginationOptions{Limit: 1})
		if err != nil || res == nil {
			t.Fatalf("query %d failed: %v", i, err)
		}
	}
	elapsed := time.Since(start)
	avgPerQuery := elapsed / queryCount

	t.Logf("Executed %d queries in %v (average: %v / query, throughput: %.1f qps)",
		queryCount, elapsed, avgPerQuery, float64(queryCount)/elapsed.Seconds())

	if avgPerQuery > 20*time.Millisecond {
		t.Errorf("average query latency exceeded budget: %v", avgPerQuery)
	}
}
