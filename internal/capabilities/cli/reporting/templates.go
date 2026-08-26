package reporting

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ReportOrchestrator coordinates rendering report data into any target format.
type ReportOrchestrator struct {
	consoleDoc *ConsoleRenderer
	structured *StructuredExporter
	document   *DocumentExporter
	visual     *VisualizationExporter
}

// NewReportOrchestrator constructs an initialized ReportOrchestrator.
func NewReportOrchestrator() *ReportOrchestrator {
	return &ReportOrchestrator{
		consoleDoc: NewConsoleRenderer(nil),
		structured: NewStructuredExporter(),
		document:   NewDocumentExporter(),
		visual:     NewVisualizationExporter(),
	}
}

// RenderReport formats and outputs report data according to the requested Format.
func (o *ReportOrchestrator) RenderReport(repType ReportType, data any, format Format, w io.Writer) error {
	if w == nil {
		return fmt.Errorf("reporting: output writer is nil")
	}

	// 1. Structured formats
	switch format {
	case FormatJSON, FormatYAML, FormatTOML, FormatXML, FormatCSV:
		return o.structured.Export(format, data, w)
	}

	// 2. Document & Console formats (convert data into DocSections)
	title, meta, sections, err := o.buildSections(repType, data)
	if err != nil {
		return err
	}

	switch format {
	case FormatMarkdown:
		return o.document.ExportMarkdown(title, meta, sections, w)
	case FormatHTML:
		return o.document.ExportHTML(title, meta, sections, w)
	case FormatPDF:
		return o.document.ExportPDF(title, meta, sections, w)
	case FormatText, "":
		return o.renderConsoleText(title, meta, sections, w)
	default:
		return fmt.Errorf("reporting: unsupported format %q for report %q", format, repType)
	}
}

func (o *ReportOrchestrator) renderConsoleText(title string, meta ReportMetadata, sections []DocSection, w io.Writer) error {
	cr := NewConsoleRenderer(w)
	cr.SectionHeader(title)
	_ = cr.KeyValues([][2]string{
		{"Repository", meta.Repository},
		{"Root Path", meta.RootPath},
		{"Generated At", meta.GeneratedAt.Format("2006-01-02 15:04:05 UTC")},
		{"Limoxel Version", meta.Version},
	})

	for _, sec := range sections {
		cr.SubSectionHeader(sec.Title)
		if sec.Description != "" {
			fmt.Fprintf(w, "  %s\n\n", sec.Description)
		}
		if len(sec.KeyValues) > 0 {
			_ = cr.KeyValues(sec.KeyValues)
		}
		if len(sec.Headers) > 0 && len(sec.Rows) > 0 {
			_ = cr.Table(sec.Headers, sec.Rows)
		}
		if sec.CodeBlock != "" {
			fmt.Fprintf(w, "\n%s\n", sec.CodeBlock)
		}
	}
	return nil
}

func (o *ReportOrchestrator) buildSections(repType ReportType, data any) (string, ReportMetadata, []DocSection, error) {
	switch d := data.(type) {
	case *RepositoryReportData:
		return "Repository Engineering Report", d.Metadata, o.buildRepoSections(d), nil
	case *ArchitectureReportData:
		return "Architecture Analysis Report", d.Metadata, o.buildArchSections(d), nil
	case *DependencyReportData:
		return "Dependency Analysis Report", d.Metadata, o.buildDepSections(d), nil
	case *HealthReportData:
		return "Repository Health & Quality Report", d.Metadata, o.buildHealthSections(d), nil
	case *ExecutiveSummaryData:
		return "Executive Engineering Summary", d.Metadata, o.buildSummarySections(d), nil
	default:
		return "", ReportMetadata{}, nil, fmt.Errorf("reporting: unknown report data payload type %T", data)
	}
}

func (o *ReportOrchestrator) buildRepoSections(d *RepositoryReportData) []DocSection {
	var sections []DocSection

	// 1. Overview
	sections = append(sections, DocSection{
		Title: "Repository Overview",
		KeyValues: [][2]string{
			{"Total Files", strconv.Itoa(d.FileCount)},
			{"Total Directories", strconv.Itoa(d.DirCount)},
			{"Total Packages", strconv.Itoa(d.PackageCount)},
			{"Total Symbols", strconv.Itoa(d.SymbolCount)},
			{"Total Dependencies", strconv.Itoa(d.DepCount)},
			{"Total Relationships", strconv.Itoa(d.RelCount)},
			{"Languages Detected", strings.Join(d.Languages, ", ")},
		},
	})

	// 2. Package Breakdown
	if len(d.Packages) > 0 {
		var rows [][]string
		for i, p := range d.Packages {
			rows = append(rows, []string{strconv.Itoa(i + 1), p})
		}
		sections = append(sections, DocSection{
			Title:   "Package Inventory",
			Headers: []string{"#", "Package Path"},
			Rows:    rows,
		})
	}

	// 3. Top Symbols
	if len(d.TopSymbols) > 0 {
		var rows [][]string
		for _, s := range d.TopSymbols {
			rows = append(rows, []string{s.Name, s.Kind, s.PackagePath, strconv.FormatBool(s.Exported)})
		}
		sections = append(sections, DocSection{
			Title:   "Primary Symbols",
			Headers: []string{"Symbol Name", "Kind", "Package", "Exported"},
			Rows:    rows,
		})
	}

	return sections
}

func (o *ReportOrchestrator) buildArchSections(d *ArchitectureReportData) []DocSection {
	var sections []DocSection

	sections = append(sections, DocSection{
		Title: "Architecture Characteristics",
		KeyValues: [][2]string{
			{"Architecture Type", d.ArchitectureType},
			{"Total Modules", strconv.Itoa(d.ModuleCount)},
			{"Total Packages", strconv.Itoa(d.PackageCount)},
			{"Architectural Layers", strings.Join(d.LayerOrder, " -> ")},
		},
	})

	if len(d.Components) > 0 {
		var rows [][]string
		for _, c := range d.Components {
			rows = append(rows, []string{c.Name, c.Layer, strconv.Itoa(c.PackageCount), strings.Join(c.Packages, ", ")})
		}
		sections = append(sections, DocSection{
			Title:   "Architectural Components",
			Headers: []string{"Component", "Layer", "Packages", "Contained Packages"},
			Rows:    rows,
		})
	}

	if len(d.Boundaries) > 0 {
		var rows [][]string
		for _, b := range d.Boundaries {
			validStr := "valid"
			if !b.Valid {
				validStr = "violation"
			}
			rows = append(rows, []string{b.FromComponent, b.ToComponent, strconv.Itoa(b.RelationCount), validStr})
		}
		sections = append(sections, DocSection{
			Title:   "Component Boundaries",
			Headers: []string{"From Component", "To Component", "Edges", "Status"},
			Rows:    rows,
		})
	}

	return sections
}

func (o *ReportOrchestrator) buildDepSections(d *DependencyReportData) []DocSection {
	var sections []DocSection

	sections = append(sections, DocSection{
		Title: "Dependency Inventory Summary",
		KeyValues: [][2]string{
			{"Total Dependencies", strconv.Itoa(d.TotalCount)},
			{"Direct Dependencies", strconv.Itoa(d.DirectCount)},
			{"Transitive Dependencies", strconv.Itoa(d.TransitiveCount)},
			{"Circular Dependency Risk", strconv.FormatBool(d.CircularRisk)},
		},
	})

	if len(d.DirectList) > 0 {
		var rows [][]string
		for _, dep := range d.DirectList {
			rows = append(rows, []string{dep.Name, dep.Version, dep.Type, dep.Scope})
		}
		sections = append(sections, DocSection{
			Title:   "Direct Dependencies",
			Headers: []string{"Dependency", "Version", "Type", "Scope"},
			Rows:    rows,
		})
	}

	return sections
}

func (o *ReportOrchestrator) buildHealthSections(d *HealthReportData) []DocSection {
	var sections []DocSection

	sections = append(sections, DocSection{
		Title: "Repository Health Overview",
		KeyValues: [][2]string{
			{"Overall Score", fmt.Sprintf("%.1f / 100", d.OverallScore)},
			{"Health Grade", d.Grade},
			{"Total Findings", strconv.Itoa(d.TotalFindings)},
		},
	})

	if len(d.Findings) > 0 {
		var rows [][]string
		for _, f := range d.Findings {
			rows = append(rows, []string{f.Severity, f.RuleID, f.Title, f.Location, f.Remediation})
		}
		sections = append(sections, DocSection{
			Title:   "Active Code Quality & Architecture Findings",
			Headers: []string{"Severity", "Rule ID", "Finding", "Location", "Remediation"},
			Rows:    rows,
		})
	}

	return sections
}

func (o *ReportOrchestrator) buildSummarySections(d *ExecutiveSummaryData) []DocSection {
	var sections []DocSection

	sections = append(sections, DocSection{
		Title: "Executive Scorecard",
		KeyValues: [][2]string{
			{"Health Score", fmt.Sprintf("%.1f / 100", d.HealthScore)},
			{"Health Grade", d.HealthGrade},
			{"Overall Risk Level", d.RiskLevel},
			{"Status", d.SummaryStatus},
		},
	})

	if len(d.Recommendations) > 0 {
		var rows [][]string
		for _, r := range d.Recommendations {
			rows = append(rows, []string{r.Priority, r.Category, r.Title, r.Target, r.Action})
		}
		sections = append(sections, DocSection{
			Title:   "Key Engineering Recommendations",
			Headers: []string{"Priority", "Category", "Issue", "Target", "Recommended Action"},
			Rows:    rows,
		})
	}

	return sections
}
