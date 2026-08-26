package reporting

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"html"
	"io"
	"strings"
	"time"
)

// DocumentExporter coordinates generation of Markdown, HTML, and PDF reports.
type DocumentExporter struct{}

// NewDocumentExporter creates a new DocumentExporter.
func NewDocumentExporter() *DocumentExporter {
	return &DocumentExporter{}
}

// ExportMarkdown writes a GitHub-Flavored Markdown report to w.
func (e *DocumentExporter) ExportMarkdown(title string, meta ReportMetadata, sections []DocSection, w io.Writer) error {
	var sb strings.Builder

	fmt.Fprintf(&sb, "# %s\n\n", title)
	fmt.Fprintf(&sb, "> **Repository:** `%s` | **Date:** %s | **Generator:** Limoxel CLI v%s\n\n",
		meta.Repository, meta.GeneratedAt.Format("2006-01-02 15:04:05 UTC"), meta.Version)
	sb.WriteString("---\n\n")

	for _, sec := range sections {
		fmt.Fprintf(&sb, "## %s\n\n", sec.Title)
		if sec.Description != "" {
			fmt.Fprintf(&sb, "%s\n\n", sec.Description)
		}

		if len(sec.KeyValues) > 0 {
			sb.WriteString("| Property | Value |\n|---|---|\n")
			for _, kv := range sec.KeyValues {
				fmt.Fprintf(&sb, "| **%s** | %s |\n", kv[0], kv[1])
			}
			sb.WriteString("\n")
		}

		if len(sec.Headers) > 0 && len(sec.Rows) > 0 {
			sb.WriteString("| ")
			sb.WriteString(strings.Join(sec.Headers, " | "))
			sb.WriteString(" |\n")
			var dividers []string
			for range sec.Headers {
				dividers = append(dividers, "---")
			}
			sb.WriteString("| ")
			sb.WriteString(strings.Join(dividers, " | "))
			sb.WriteString(" |\n")

			for _, row := range sec.Rows {
				sb.WriteString("| ")
				sb.WriteString(strings.Join(row, " | "))
				sb.WriteString(" |\n")
			}
			sb.WriteString("\n")
		}

		if sec.CodeBlock != "" {
			fmt.Fprintf(&sb, "```%s\n%s\n```\n\n", sec.CodeLang, sec.CodeBlock)
		}
	}

	_, err := io.WriteString(w, sb.String())
	return err
}

// ExportHTML writes a standalone, styled HTML5 report document to w.
func (e *DocumentExporter) ExportHTML(title string, meta ReportMetadata, sections []DocSection, w io.Writer) error {
	var sb strings.Builder

	sb.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	sb.WriteString("<meta charset=\"UTF-8\">\n")
	sb.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	fmt.Fprintf(&sb, "<title>%s — Limoxel Report</title>\n", html.EscapeString(title))
	sb.WriteString("<style>\n")
	sb.WriteString(`
		:root {
			--bg: #0d1117;
			--card-bg: #161b22;
			--border: #30363d;
			--text: #c9d1d9;
			--heading: #f0f6fc;
			--accent: #58a6ff;
			--accent-bg: rgba(56, 139, 253, 0.15);
			--green: #3fb950;
			--yellow: #d29922;
			--red: #f85149;
		}
		body {
			font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
			background-color: var(--bg);
			color: var(--text);
			margin: 0;
			padding: 40px 20px;
			line-height: 1.6;
		}
		.container {
			max-width: 1000px;
			margin: 0 auto;
		}
		.header {
			border-bottom: 1px solid var(--border);
			padding-bottom: 20px;
			margin-bottom: 30px;
		}
		h1 { color: var(--heading); margin: 0 0 10px 0; font-size: 28px; }
		.meta { color: #8b949e; font-size: 14px; }
		.badge {
			display: inline-block;
			padding: 2px 8px;
			border-radius: 12px;
			font-size: 12px;
			font-weight: 600;
			background: var(--accent-bg);
			color: var(--accent);
			border: 1px solid rgba(56, 139, 253, 0.4);
		}
		.section {
			background: var(--card-bg);
			border: 1px solid var(--border);
			border-radius: 8px;
			padding: 24px;
			margin-bottom: 24px;
		}
		h2 { color: var(--heading); margin-top: 0; font-size: 20px; border-bottom: 1px solid var(--border); padding-bottom: 8px; }
		table { width: 100%; border-collapse: collapse; margin: 16px 0; font-size: 14px; }
		th, td { text-align: left; padding: 10px 14px; border-bottom: 1px solid var(--border); }
		th { color: var(--heading); background: rgba(255,255,255,0.02); }
		tr:hover { background: rgba(255,255,255,0.02); }
		pre {
			background: #090d13;
			border: 1px solid var(--border);
			border-radius: 6px;
			padding: 16px;
			overflow-x: auto;
			font-family: "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace;
			font-size: 13px;
		}
		.footer { text-align: center; margin-top: 40px; color: #8b949e; font-size: 13px; }
	`)
	sb.WriteString("</style>\n</head>\n<body>\n<div class=\"container\">\n")

	// Header
	sb.WriteString("<div class=\"header\">\n")
	fmt.Fprintf(&sb, "<h1>%s</h1>\n", html.EscapeString(title))
	fmt.Fprintf(&sb, "<div class=\"meta\">Repository: <strong>%s</strong> | Generated: %s | <span class=\"badge\">Limoxel v%s</span></div>\n",
		html.EscapeString(meta.Repository), meta.GeneratedAt.Format("2006-01-02 15:04:05 UTC"), html.EscapeString(meta.Version))
	sb.WriteString("</div>\n")

	// Sections
	for _, sec := range sections {
		sb.WriteString("<div class=\"section\">\n")
		fmt.Fprintf(&sb, "<h2>%s</h2>\n", html.EscapeString(sec.Title))
		if sec.Description != "" {
			fmt.Fprintf(&sb, "<p>%s</p>\n", html.EscapeString(sec.Description))
		}

		if len(sec.KeyValues) > 0 {
			sb.WriteString("<table><thead><tr><th>Property</th><th>Value</th></tr></thead><tbody>\n")
			for _, kv := range sec.KeyValues {
				fmt.Fprintf(&sb, "<tr><td><strong>%s</strong></td><td>%s</td></tr>\n",
					html.EscapeString(kv[0]), html.EscapeString(kv[1]))
			}
			sb.WriteString("</tbody></table>\n")
		}

		if len(sec.Headers) > 0 && len(sec.Rows) > 0 {
			sb.WriteString("<table><thead><tr>\n")
			for _, h := range sec.Headers {
				fmt.Fprintf(&sb, "<th>%s</th>\n", html.EscapeString(h))
			}
			sb.WriteString("</tr></thead><tbody>\n")
			for _, row := range sec.Rows {
				sb.WriteString("<tr>\n")
				for _, col := range row {
					fmt.Fprintf(&sb, "<td>%s</td>\n", html.EscapeString(col))
				}
				sb.WriteString("</tr>\n")
			}
			sb.WriteString("</tbody></table>\n")
		}

		if sec.CodeBlock != "" {
			fmt.Fprintf(&sb, "<pre><code class=\"language-%s\">%s</code></pre>\n",
				html.EscapeString(sec.CodeLang), html.EscapeString(sec.CodeBlock))
		}
		sb.WriteString("</div>\n")
	}

	// Footer
	sb.WriteString("<div class=\"footer\">\n")
	sb.WriteString("Generated by Limoxel Engineering Intelligence Platform — Authoritative Report\n")
	sb.WriteString("</div>\n</div>\n</body>\n</html>\n")

	_, err := io.WriteString(w, sb.String())
	return err
}

// ExportPDF generates a standard-compliant PDF 1.4 binary document to w.
func (e *DocumentExporter) ExportPDF(title string, meta ReportMetadata, sections []DocSection, w io.Writer) error {
	pdf := newPDFBuilder()

	// Title
	pdf.addHeader(title, meta)

	// Sections
	for _, sec := range sections {
		pdf.addSection(sec)
	}

	return pdf.writeTo(w)
}

// DocSection encapsulates a structured document section for Markdown, HTML, and PDF export.
type DocSection struct {
	Title       string
	Description string
	KeyValues   [][2]string
	Headers     []string
	Rows        [][]string
	CodeBlock   string
	CodeLang    string
}

// -----------------------------------------------------------------------------
// Pure-Go Compliant PDF Builder (PDF 1.4 / 1.7)
// -----------------------------------------------------------------------------

type pdfBuilder struct {
	objects []string
	lines   []string
}

func newPDFBuilder() *pdfBuilder {
	return &pdfBuilder{
		objects: make([]string, 0),
		lines:   make([]string, 0),
	}
}

func (p *pdfBuilder) addHeader(title string, meta ReportMetadata) {
	p.lines = append(p.lines, fmt.Sprintf("=== %s ===", strings.ToUpper(title)))
	p.lines = append(p.lines, fmt.Sprintf("Repository : %s", meta.Repository))
	p.lines = append(p.lines, fmt.Sprintf("Root Path  : %s", meta.RootPath))
	p.lines = append(p.lines, fmt.Sprintf("Generated  : %s", meta.GeneratedAt.Format(time.RFC3339)))
	p.lines = append(p.lines, fmt.Sprintf("Generator  : %s (v%s)", meta.Generator, meta.Version))
	p.lines = append(p.lines, "--------------------------------------------------------------------------------")
	p.lines = append(p.lines, "")
}

func (p *pdfBuilder) addSection(sec DocSection) {
	p.lines = append(p.lines, fmt.Sprintf("## %s", strings.ToUpper(sec.Title)))
	if sec.Description != "" {
		p.lines = append(p.lines, sec.Description)
	}

	for _, kv := range sec.KeyValues {
		p.lines = append(p.lines, fmt.Sprintf("  * %-25s : %s", kv[0], kv[1]))
	}

	if len(sec.Headers) > 0 && len(sec.Rows) > 0 {
		p.lines = append(p.lines, "")
		p.lines = append(p.lines, strings.Join(sec.Headers, "  |  "))
		p.lines = append(p.lines, strings.Repeat("-", 70))
		for _, row := range sec.Rows {
			p.lines = append(p.lines, strings.Join(row, "  |  "))
		}
	}

	if sec.CodeBlock != "" {
		p.lines = append(p.lines, "")
		p.lines = append(p.lines, "--- Code Block ---")
		codeLines := strings.Split(sec.CodeBlock, "\n")
		for _, cl := range codeLines {
			p.lines = append(p.lines, "  "+cl)
		}
	}

	p.lines = append(p.lines, "")
}

func (p *pdfBuilder) writeTo(w io.Writer) error {
	var pageContent bytes.Buffer
	pageContent.WriteString("BT\n/F1 10 Tf\n12 TL\n50 750 Td\n")

	linesPerPage := 55
	currentPageLines := 0

	for _, line := range p.lines {
		if currentPageLines >= linesPerPage {
			// Soft page delimiter in text stream
			pageContent.WriteString("ET\n")
			pageContent.WriteString("BT\n/F1 10 Tf\n12 TL\n50 750 Td\n")
			currentPageLines = 0
		}
		escaped := escapePDFText(line)
		fmt.Fprintf(&pageContent, "(%s) '\n", escaped)
		currentPageLines++
	}
	pageContent.WriteString("ET\n")

	// Compress content stream with zlib for efficient PDF sizing
	var compressedStream bytes.Buffer
	zw := zlib.NewWriter(&compressedStream)
	_, _ = zw.Write(pageContent.Bytes())
	_ = zw.Close()

	// Build standard PDF 1.4 Object Graph
	// 1: Catalog
	// 2: Pages
	// 3: Page
	// 4: Font F1
	// 5: Contents stream

	obj1 := "<< /Type /Catalog /Pages 2 0 R >>"
	obj2 := "<< /Type /Pages /Kids [3 0 R] /Count 1 >>"
	obj3 := "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>"
	obj4 := "<< /Type /Font /Subtype /Type1 /BaseFont /Courier >>"
	obj5 := fmt.Sprintf("<< /Length %d /Filter /FlateDecode >>\nstream\n%s\nendstream",
		compressedStream.Len(), compressedStream.String())

	objects := []string{obj1, obj2, obj3, obj4, obj5}

	var pdf bytes.Buffer
	pdf.WriteString("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")

	offsets := make([]int, len(objects)+1)
	offsets[0] = 0

	for i, obj := range objects {
		offsets[i+1] = pdf.Len()
		fmt.Fprintf(&pdf, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}

	xrefOffset := pdf.Len()
	fmt.Fprintf(&pdf, "xref\n0 %d\n", len(objects)+1)
	pdf.WriteString("0000000000 65535 f \n")
	for i := 1; i <= len(objects); i++ {
		fmt.Fprintf(&pdf, "%010d 00000 n \n", offsets[i])
	}

	fmt.Fprintf(&pdf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n",
		len(objects)+1, xrefOffset)

	_, err := w.Write(pdf.Bytes())
	return err
}

func escapePDFText(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	return s
}
