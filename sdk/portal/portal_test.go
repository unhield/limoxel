package portal_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeveloperPortal_StaticFilesExist(t *testing.T) {
	requiredFiles := []string{
		"index.html",
		"docs.html",
		"api-explorer.html",
		"examples.html",
		"changelog.html",
		"css/portal.css",
		"js/portal.js",
	}

	for _, req := range requiredFiles {
		fullPath := filepath.Join(".", req)
		info, err := os.Stat(fullPath)
		if err != nil {
			t.Errorf("missing portal file %q: %v", req, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("portal file %q is empty", req)
		}
	}
}

func TestDeveloperPortal_ContractsCoverageInExplorer(t *testing.T) {
	content, err := os.ReadFile("api-explorer.html")
	if err != nil {
		t.Fatalf("failed to read api-explorer.html: %v", err)
	}

	htmlStr := string(content)

	expectedContracts := []string{
		"RepositoryManagementContract",
		"FileContract",
		"PackageContract",
		"SymbolContract",
		"SearchContract",
		"GraphContract",
		"AnalysisContract",
		"NavigationContract",
		"ReasoningContract",
		"EventContract",
		"IntelligenceContract",
	}

	for _, contract := range expectedContracts {
		if !strings.Contains(htmlStr, contract) {
			t.Errorf("api-explorer.html is missing documented contract %q", contract)
		}
	}
}

func TestDeveloperPortal_NavLinksConsistency(t *testing.T) {
	htmlFiles := []string{
		"index.html",
		"docs.html",
		"api-explorer.html",
		"examples.html",
		"changelog.html",
	}

	for _, file := range htmlFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Errorf("failed to read %s: %v", file, err)
			continue
		}
		htmlStr := string(content)

		for _, target := range htmlFiles {
			if !strings.Contains(htmlStr, `href="`+target+`"`) {
				t.Errorf("%s missing navigation link to %s", file, target)
			}
		}
	}
}
