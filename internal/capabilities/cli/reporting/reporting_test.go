package reporting

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func sampleReportMetadata() ReportMetadata {
	return ReportMetadata{
		Title:         "Test Suite Report",
		ReportType:    "repository",
		Repository:    "limoxel",
		RootPath:      "/workspace/limoxel",
		GeneratedAt:   time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC),
		Generator:     "Limoxel Test Suite",
		Version:       "1.0.0",
		Authoritative: true,
	}
}

func sampleRepositoryData() *RepositoryReportData {
	return &RepositoryReportData{
		Metadata:     sampleReportMetadata(),
		FileCount:    315,
		DirCount:     45,
		PackageCount: 45,
		SymbolCount:  4150,
		DepCount:     300,
		RelCount:     11000,
		Languages:    []string{"Go", "Markdown", "YAML"},
		Packages:     []string{"cmd/limoxel", "internal/cli", "internal/repository"},
		TopSymbols: []SymbolSummary{
			{ID: "sym:1", Name: "NewApp", Kind: "function", PackagePath: "cmd/limoxel", FilePath: "cmd/limoxel/main.go", Exported: true},
			{ID: "sym:2", Name: "Run", Kind: "method", PackagePath: "internal/cli", FilePath: "internal/cli/app.go", Exported: true},
		},
		Dependencies: []DependencyEntry{
			{Name: "github.com/unhield/limoxel", Version: "v1.0.0", Type: "module", Direct: true, Scope: "production"},
		},
	}
}

func sampleArchitectureData() *ArchitectureReportData {
	return &ArchitectureReportData{
		Metadata:         sampleReportMetadata(),
		ArchitectureType: "Modular Architecture",
		ModuleCount:      1,
		PackageCount:     3,
		Components: []ComponentSummary{
			{ID: "c1", Name: "CLI", Layer: "Interface", PackageCount: 1, Packages: []string{"cmd/limoxel"}},
			{ID: "c2", Name: "Platform", Layer: "Core", PackageCount: 2, Packages: []string{"internal/cli", "internal/repository"}},
		},
		Boundaries: []BoundarySummary{
			{FromComponent: "CLI", ToComponent: "Platform", RelationCount: 5, Valid: true},
		},
		LayerOrder: []string{"Interface", "Core"},
	}
}

func sampleDependencyData() *DependencyReportData {
	return &DependencyReportData{
		Metadata:        sampleReportMetadata(),
		TotalCount:      2,
		DirectCount:     1,
		TransitiveCount: 1,
		CircularRisk:    false,
		DirectList: []DependencyEntry{
			{Name: "golang.org/x/tools", Version: "v0.1.0", Type: "module", Direct: true, Scope: "production"},
		},
		TransitiveList: []DependencyEntry{
			{Name: "golang.org/x/mod", Version: "v0.5.0", Type: "module", Direct: false, Scope: "transitive"},
		},
	}
}

func sampleHealthData() *HealthReportData {
	return &HealthReportData{
		Metadata:      sampleReportMetadata(),
		OverallScore:  92.5,
		Grade:         "A",
		TotalFindings: 1,
		SeverityCount: []CountEntry{{Name: "medium", Count: 1}},
		CategoryCount: []CountEntry{{Name: "quality", Count: 1}},
		Findings: []FindingSummary{
			{ID: "f1", RuleID: "QUAL-001", Severity: "medium", Category: "quality", Title: "Unreferenced variable", Location: "main.go:10", Remediation: "Remove unused variable"},
		},
	}
}

func sampleExecutiveSummaryData() *ExecutiveSummaryData {
	return &ExecutiveSummaryData{
		Metadata:      sampleReportMetadata(),
		HealthScore:   92.5,
		HealthGrade:   "A",
		RiskLevel:     "Low",
		SummaryStatus: "Repository is in a healthy state.",
		KeyMetrics: []MetricEntry{
			{Key: "Files", Value: "315"},
			{Key: "Packages", Value: "45"},
		},
		Recommendations: []RecommendationSummary{
			{ID: "r1", Priority: "high", Category: "quality", Title: "Update dependencies", Target: "go.mod", Action: "Run go get -u"},
		},
		TopFindings: []FindingSummary{
			{ID: "f1", RuleID: "QUAL-001", Severity: "medium", Category: "quality", Title: "Unreferenced variable", Location: "main.go:10", Remediation: "Remove unused variable"},
		},
	}
}

func sampleGraphVisualData() *GraphVisualData {
	return &GraphVisualData{
		Title:       "Test Knowledge Graph",
		DiagramType: DiagramArchitecture,
		Nodes: []GraphNode{
			{ID: "pkg:main", Label: "main", Kind: "package", Shape: "box", Color: "#58a6ff"},
			{ID: "pkg:cli", Label: "cli", Kind: "package", Shape: "box", Color: "#58a6ff"},
			{ID: "sym:Run", Label: "Run", Kind: "symbol", Shape: "ellipse", Color: "#3fb950"},
		},
		Edges: []GraphEdge{
			{Source: "pkg:main", Target: "pkg:cli", Label: "imports", Kind: "imports", Dotted: false},
			{Source: "pkg:cli", Target: "sym:Run", Label: "declares", Kind: "owns", Dotted: true},
		},
		Subgraphs: []Subgraph{
			{ID: "sub1", Title: "Command Subsystem", NodeIDs: []string{"pkg:main", "pkg:cli"}},
		},
	}
}

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input    string
		expected Format
		wantErr  bool
	}{
		{"text", FormatText, false},
		{"console", FormatText, false},
		{"json", FormatJSON, false},
		{"yaml", FormatYAML, false},
		{"yml", FormatYAML, false},
		{"toml", FormatTOML, false},
		{"xml", FormatXML, false},
		{"csv", FormatCSV, false},
		{"markdown", FormatMarkdown, false},
		{"md", FormatMarkdown, false},
		{"html", FormatHTML, false},
		{"pdf", FormatPDF, false},
		{"mermaid", FormatMermaid, false},
		{"graphviz", FormatGraphviz, false},
		{"dot", FormatGraphviz, false},
		{"svg", FormatSVG, false},
		{"png", FormatPNG, false},
		{"interactive", FormatInteractive, false},
		{"invalid_fmt", "", true},
	}

	for _, tt := range tests {
		got, err := ParseFormat(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseFormat(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
		}
		if got != tt.expected {
			t.Errorf("ParseFormat(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestConsoleRenderer(t *testing.T) {
	var buf bytes.Buffer
	cr := NewConsoleRenderer(&buf)
	cr.SetColorEnabled(false)

	_ = cr.Status("SUCCESS", "Operation completed successfully")
	_ = cr.Status("ERROR", "Something went wrong")
	cr.SectionHeader("Repository Overview")
	cr.SubSectionHeader("General Info")

	_ = cr.KeyValues([][2]string{
		{"Language", "Go"},
		{"Status", "Healthy"},
	})

	_ = cr.Table([]string{"Name", "Role"}, [][]string{
		{"Limoxel", "Core Engine"},
		{"CLI", "Developer Tool"},
	})

	root := &TreeNode{
		Label: "Root Package",
		Children: []*TreeNode{
			{Label: "Child 1"},
			{
				Label: "Child 2",
				Children: []*TreeNode{
					{Label: "Grandchild 1"},
				},
			},
		},
	}
	_ = cr.Tree(root)

	cr.ProgressBar(5, 10, "Processing")
	cr.ProgressBar(10, 10, "Completed")

	tracker := NewStepTracker(cr, 3)
	tracker.Step("Step 1")
	tracker.Step("Step 2")
	tracker.Done("All steps complete")

	out := buf.String()
	if !strings.Contains(out, "[SUCCESS] Operation completed successfully") {
		t.Errorf("ConsoleRenderer missing success status: %s", out)
	}
	if !strings.Contains(out, "=== REPOSITORY OVERVIEW ===") {
		t.Errorf("ConsoleRenderer missing section header: %s", out)
	}
	if !strings.Contains(out, "Language") || !strings.Contains(out, "Go") {
		t.Errorf("ConsoleRenderer missing key values: %s", out)
	}
	if !strings.Contains(out, "├── Child 1") || !strings.Contains(out, "└── Child 2") {
		t.Errorf("ConsoleRenderer missing tree view: %s", out)
	}
}

func TestStructuredExporter(t *testing.T) {
	exp := NewStructuredExporter()
	repoData := sampleRepositoryData()

	formats := []Format{FormatJSON, FormatYAML, FormatTOML, FormatXML, FormatCSV}
	for _, fmtType := range formats {
		t.Run(string(fmtType), func(t *testing.T) {
			var buf bytes.Buffer
			err := exp.Export(fmtType, repoData, &buf)
			if err != nil {
				t.Fatalf("Export(%s) failed: %v", fmtType, err)
			}
			if buf.Len() == 0 {
				t.Fatalf("Export(%s) produced empty output", fmtType)
			}

			out := buf.String()
			switch fmtType {
			case FormatJSON:
				if !strings.HasPrefix(strings.TrimSpace(out), "{") {
					t.Errorf("JSON output invalid start: %s", out)
				}
			case FormatYAML:
				if !strings.Contains(out, "file_count:") {
					t.Errorf("YAML output missing file_count: %s", out)
				}
			case FormatTOML:
				if !strings.Contains(out, "file_count = 315") {
					t.Errorf("TOML output missing file_count: %s", out)
				}
			case FormatXML:
				if !strings.HasPrefix(strings.TrimSpace(out), "<?xml") {
					t.Errorf("XML output missing header: %s", out)
				}
			case FormatCSV:
				if !strings.Contains(out, "file_count") {
					t.Errorf("CSV output missing headers: %s", out)
				}
			}
		})
	}
}

func TestDocumentExporter(t *testing.T) {
	exp := NewDocumentExporter()
	meta := sampleReportMetadata()
	sections := []DocSection{
		{
			Title:       "Overview",
			Description: "This is a test report overview.",
			KeyValues: [][2]string{
				{"Files", "315"},
				{"Packages", "45"},
			},
			Headers: []string{"Item", "Value"},
			Rows: [][]string{
				{"Language", "Go"},
			},
			CodeBlock: "func main() {}",
			CodeLang:  "go",
		},
	}

	// 1. Markdown
	var mdBuf bytes.Buffer
	if err := exp.ExportMarkdown("Test Markdown", meta, sections, &mdBuf); err != nil {
		t.Fatalf("ExportMarkdown failed: %v", err)
	}
	if !strings.Contains(mdBuf.String(), "# Test Markdown") || !strings.Contains(mdBuf.String(), "```go") {
		t.Errorf("Markdown output malformed: %s", mdBuf.String())
	}

	// 2. HTML
	var htmlBuf bytes.Buffer
	if err := exp.ExportHTML("Test HTML", meta, sections, &htmlBuf); err != nil {
		t.Fatalf("ExportHTML failed: %v", err)
	}
	if !strings.Contains(htmlBuf.String(), "<!DOCTYPE html>") || !strings.Contains(htmlBuf.String(), "<h1>Test HTML</h1>") {
		t.Errorf("HTML output malformed: %s", htmlBuf.String())
	}

	// 3. PDF
	var pdfBuf bytes.Buffer
	if err := exp.ExportPDF("Test PDF", meta, sections, &pdfBuf); err != nil {
		t.Fatalf("ExportPDF failed: %v", err)
	}
	pdfBytes := pdfBuf.Bytes()
	if !bytes.HasPrefix(pdfBytes, []byte("%PDF-1.4")) {
		t.Errorf("PDF output missing standard header: %s", string(pdfBytes[:20]))
	}
	if !bytes.Contains(pdfBytes, []byte("%%EOF")) {
		t.Errorf("PDF output missing standard trailer %%EOF")
	}
}

func TestVisualizationExporter(t *testing.T) {
	exp := NewVisualizationExporter()
	graphData := sampleGraphVisualData()

	// 1. Mermaid
	var mermaidBuf bytes.Buffer
	if err := exp.ExportMermaid(graphData, &mermaidBuf); err != nil {
		t.Fatalf("ExportMermaid failed: %v", err)
	}
	if !strings.HasPrefix(strings.TrimSpace(mermaidBuf.String()), "flowchart TD") {
		t.Errorf("Mermaid output invalid: %s", mermaidBuf.String())
	}

	// 2. Graphviz
	var dotBuf bytes.Buffer
	if err := exp.ExportGraphviz(graphData, &dotBuf); err != nil {
		t.Fatalf("ExportGraphviz failed: %v", err)
	}
	if !strings.Contains(dotBuf.String(), "digraph G {") {
		t.Errorf("Graphviz output invalid: %s", dotBuf.String())
	}

	// 3. SVG
	var svgBuf bytes.Buffer
	if err := exp.ExportSVG(graphData, &svgBuf); err != nil {
		t.Fatalf("ExportSVG failed: %v", err)
	}
	if !strings.HasPrefix(strings.TrimSpace(svgBuf.String()), "<svg") {
		t.Errorf("SVG output invalid: %s", svgBuf.String())
	}

	// 4. PNG
	var pngBuf bytes.Buffer
	if err := exp.ExportPNG(graphData, &pngBuf); err != nil {
		t.Fatalf("ExportPNG failed: %v", err)
	}
	img, err := png.Decode(&pngBuf)
	if err != nil {
		t.Fatalf("ExportPNG produced invalid PNG image: %v", err)
	}
	if img.Bounds().Dx() <= 0 || img.Bounds().Dy() <= 0 {
		t.Errorf("ExportPNG produced invalid bounds: %v", img.Bounds())
	}

	// 5. Interactive
	var interactiveBuf bytes.Buffer
	if err := exp.ExportInteractive(graphData, &interactiveBuf); err != nil {
		t.Fatalf("ExportInteractive failed: %v", err)
	}
	if !strings.Contains(interactiveBuf.String(), "<!DOCTYPE html>") || !strings.Contains(interactiveBuf.String(), "Interactive Graph") {
		t.Errorf("Interactive output invalid: %s", interactiveBuf.String())
	}
}

func TestReportTemplates(t *testing.T) {
	orchestrator := NewReportOrchestrator()

	tests := []struct {
		name    string
		repType ReportType
		data    any
	}{
		{"Repository", ReportRepository, sampleRepositoryData()},
		{"Architecture", ReportArchitecture, sampleArchitectureData()},
		{"Dependency", ReportDependency, sampleDependencyData()},
		{"Health", ReportHealth, sampleHealthData()},
		{"Summary", ReportExecutiveSummary, sampleExecutiveSummaryData()},
	}

	formats := []Format{FormatText, FormatMarkdown, FormatHTML, FormatPDF, FormatJSON, FormatYAML, FormatTOML, FormatXML, FormatCSV}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, fmtType := range formats {
				var buf bytes.Buffer
				err := orchestrator.RenderReport(tt.repType, tt.data, fmtType, &buf)
				if err != nil {
					t.Fatalf("RenderReport(%s, %s) failed: %v", tt.name, fmtType, err)
				}
				if buf.Len() == 0 {
					t.Fatalf("RenderReport(%s, %s) produced empty output", tt.name, fmtType)
				}
			}
		})
	}
}

func TestSafeFileWriter(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "limoxel_reporting_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	writer := NewSafeFileWriter()
	destFile := filepath.Join(tmpDir, "reports", "sub", "test.txt")

	// Write file creating parent directories
	data := []byte("Hello Limoxel Report")
	if err := writer.WriteFile(destFile, data, false); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	readBack, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("failed to read back written file: %v", err)
	}
	if string(readBack) != "Hello Limoxel Report" {
		t.Errorf("readBack content mismatch: got %q, want %q", string(readBack), "Hello Limoxel Report")
	}

	// Overwrite guard
	if err := writer.WriteFile(destFile, []byte("Overwrite"), false); err == nil {
		t.Errorf("expected error overwriting existing file with overwrite=false")
	}

	// Overwrite allowed
	if err := writer.WriteFile(destFile, []byte("Overwrite"), true); err != nil {
		t.Fatalf("failed to overwrite existing file with overwrite=true: %v", err)
	}

	// Invalid path error
	if err := writer.WriteFile("", []byte("fail"), true); err == nil {
		t.Errorf("expected error for empty destination path")
	}
}

func TestDeterminism(t *testing.T) {
	orchestrator := NewReportOrchestrator()
	data := sampleRepositoryData()

	var buf1 bytes.Buffer
	var buf2 bytes.Buffer

	_ = orchestrator.RenderReport(ReportRepository, data, FormatJSON, &buf1)
	_ = orchestrator.RenderReport(ReportRepository, data, FormatJSON, &buf2)

	if buf1.String() != buf2.String() {
		t.Errorf("JSON report non-deterministic:\nrun1:\n%s\nrun2:\n%s", buf1.String(), buf2.String())
	}

	var yml1 bytes.Buffer
	var yml2 bytes.Buffer
	_ = orchestrator.RenderReport(ReportRepository, data, FormatYAML, &yml1)
	_ = orchestrator.RenderReport(ReportRepository, data, FormatYAML, &yml2)

	if yml1.String() != yml2.String() {
		t.Errorf("YAML report non-deterministic:\nrun1:\n%s\nrun2:\n%s", yml1.String(), yml2.String())
	}
}
