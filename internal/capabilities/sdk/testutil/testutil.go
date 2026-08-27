package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/repository/dependency"
	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
	"github.com/unhield/limoxel/internal/capabilities/repository/graph"
	"github.com/unhield/limoxel/internal/capabilities/repository/indexing"
	"github.com/unhield/limoxel/internal/capabilities/repository/language"
	"github.com/unhield/limoxel/internal/capabilities/repository/metadata"
	"github.com/unhield/limoxel/internal/capabilities/repository/query"
	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
	"github.com/unhield/limoxel/internal/capabilities/repository/xref"
	langreg "github.com/unhield/limoxel/internal/language"
)

// SetupTestLanguageRegistry creates an initialized Language Registry with standard languages.
func SetupTestLanguageRegistry(t *testing.T) *langreg.Registry {
	t.Helper()
	reg := langreg.NewRegistry()

	goLang, _ := langreg.New("go", "Go", []string{".go"}, nil, []string{"golang"})
	mdLang, _ := langreg.New("markdown", "Markdown", []string{".md"}, nil, []string{"md"})
	jsonLang, _ := langreg.New("json", "JSON", []string{".json"}, nil, []string{"json"})
	yamlLang, _ := langreg.New("yaml", "YAML", []string{".yaml", ".yml"}, nil, []string{"yaml"})

	_ = reg.Register(goLang)
	_ = reg.Register(mdLang)
	_ = reg.Register(jsonLang)
	_ = reg.Register(yamlLang)

	return reg
}

// SetupTestRepository creates a temporary repository workspace and returns the initialized Query RepositoryService.
func SetupTestRepository(t *testing.T) (*query.RepositoryService, string) {
	t.Helper()
	tempDir := t.TempDir()
	repoRoot := filepath.Join(tempDir, "sample_repo")
	_ = os.MkdirAll(filepath.Join(repoRoot, "pkg", "math"), 0755)
	_ = os.MkdirAll(filepath.Join(repoRoot, "pkg", "auth"), 0755)
	_ = os.MkdirAll(filepath.Join(repoRoot, "docs"), 0755)

	mainSrc := `package main

import "sample_repo/pkg/math"

func main() {
	math.Add(1, 2)
}
`
	mathSrc := `package math

// Calculator computes math operations.
type Calculator interface {
	Compute(a, b int) int
}

// Add adds two numbers together.
func Add(a, b int) int {
	return a + b
}
`
	authSrc := `package auth

// User represents a user entity.
type User struct {
	ID string
	Name string
}

// Authenticate verifies user credentials.
func Authenticate(username, password string) bool {
	return username != ""
}
`
	docSrc := `# Sample Repository Documentation
This is a test documentation file for SDK testing.
`
	cfgSrc := `app_name = "sample_repo"
version = "1.0.0"
`
	_ = os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte(mainSrc), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "pkg", "math", "math.go"), []byte(mathSrc), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "pkg", "auth", "auth.go"), []byte(authSrc), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "docs", "README.md"), []byte(docSrc), 0644)
	_ = os.WriteFile(filepath.Join(repoRoot, "config.toml"), []byte(cfgSrc), 0644)

	reg := SetupTestLanguageRegistry(t)
	disc, err := discovery.New(reg)
	if err != nil {
		t.Fatalf("failed to create discoverer: %v", err)
	}

	metaCollector, _ := metadata.New(disc)
	langAnalyzer, _ := language.New(disc)
	depAnalyzer, _ := dependency.New(disc)
	indexer, _ := indexing.New(disc)
	symEngine, _ := symbol.New(disc)
	xrefEngine, _ := xref.New(disc, symEngine, depAnalyzer)
	graphEngine, _ := graph.New(disc, metaCollector, langAnalyzer, depAnalyzer, indexer, symEngine, xrefEngine)

	svc := query.NewRepositoryService(
		disc,
		metaCollector,
		langAnalyzer,
		depAnalyzer,
		indexer,
		symEngine,
		xrefEngine,
		graphEngine,
	)

	if err := svc.Load(repoRoot); err != nil {
		t.Fatalf("failed to load test repository: %v", err)
	}

	return svc, repoRoot
}
