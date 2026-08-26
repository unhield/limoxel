package reporting

import (
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
)

// ANSI Color constants for rich terminal output
const (
	ColorReset   = "\033[0m"
	ColorBold    = "\033[1m"
	ColorDim     = "\033[2m"
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorBlue    = "\033[34m"
	ColorMagenta = "\033[35m"
	ColorCyan    = "\033[36m"
	ColorGray    = "\033[90m"
)

// ConsoleRenderer provides rich terminal formatting, tables, trees, and progress output.
type ConsoleRenderer struct {
	w            io.Writer
	colorEnabled bool
}

// NewConsoleRenderer creates a new ConsoleRenderer targeting w.
func NewConsoleRenderer(w io.Writer) *ConsoleRenderer {
	if w == nil {
		w = os.Stdout
	}
	// Check if color is explicitly disabled by NO_COLOR or non-terminal
	colorOn := true
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		colorOn = false
	}
	return &ConsoleRenderer{
		w:            w,
		colorEnabled: colorOn,
	}
}

// SetColorEnabled toggles ANSI color output.
func (r *ConsoleRenderer) SetColorEnabled(enabled bool) {
	if r != nil {
		r.colorEnabled = enabled
	}
}

// Color applies an ANSI color code if colors are enabled.
func (r *ConsoleRenderer) Color(code, text string) string {
	if r == nil || !r.colorEnabled {
		return text
	}
	return code + text + ColorReset
}

// StatusIndicator prints a standardized, colored status badge.
func (r *ConsoleRenderer) Status(status, msg string) error {
	var badge string
	cleanStatus := strings.ToUpper(strings.TrimSpace(status))
	switch cleanStatus {
	case "SUCCESS", "OK", "READY", "PASSED":
		badge = r.Color(ColorGreen+ColorBold, "["+cleanStatus+"]")
	case "INFO", "RUNNING", "SCANNING", "ANALYZING":
		badge = r.Color(ColorCyan+ColorBold, "["+cleanStatus+"]")
	case "WARN", "WARNING", "DEPRECATED":
		badge = r.Color(ColorYellow+ColorBold, "["+cleanStatus+"]")
	case "ERROR", "FAILED", "BLOCKED":
		badge = r.Color(ColorRed+ColorBold, "["+cleanStatus+"]")
	case "PENDING", "WAITING":
		badge = r.Color(ColorGray+ColorBold, "["+cleanStatus+"]")
	default:
		badge = fmt.Sprintf("[%s]", cleanStatus)
	}

	_, err := fmt.Fprintf(r.w, "%s %s\n", badge, msg)
	return err
}

// SectionHeader writes a visually distinct section divider.
func (r *ConsoleRenderer) SectionHeader(title string) {
	styled := r.Color(ColorBold+ColorCyan, "=== "+strings.ToUpper(title)+" ===")
	fmt.Fprintf(r.w, "\n%s\n", styled)
}

// SubSectionHeader writes a subsection title.
func (r *ConsoleRenderer) SubSectionHeader(title string) {
	styled := r.Color(ColorBold, "--- "+title+" ---")
	fmt.Fprintf(r.w, "\n%s\n", styled)
}

// KeyValues writes aligned key-value pairs.
func (r *ConsoleRenderer) KeyValues(pairs [][2]string) error {
	maxKeyLen := 0
	for _, p := range pairs {
		if len(p[0]) > maxKeyLen {
			maxKeyLen = len(p[0])
		}
	}

	for _, p := range pairs {
		paddedKey := p[0] + strings.Repeat(" ", maxKeyLen-len(p[0]))
		keyColored := r.Color(ColorGray, paddedKey)
		valColored := r.Color(ColorBold, p[1])
		if _, err := fmt.Fprintf(r.w, "  %s : %s\n", keyColored, valColored); err != nil {
			return err
		}
	}
	return nil
}

// Table writes a clean tabular display with headers and aligned columns.
func (r *ConsoleRenderer) Table(headers []string, rows [][]string) error {
	if len(headers) == 0 {
		return nil
	}

	tw := tabwriter.NewWriter(r.w, 2, 4, 3, ' ', 0)

	// Format headers in Bold
	headerLine := make([]string, len(headers))
	dividerLine := make([]string, len(headers))
	for i, h := range headers {
		headerLine[i] = r.Color(ColorBold, strings.ToUpper(h))
		dividerLine[i] = strings.Repeat("-", len(h)+2)
	}
	fmt.Fprintln(tw, strings.Join(headerLine, "\t"))
	fmt.Fprintln(tw, strings.Join(dividerLine, "\t"))

	// Format rows
	for _, row := range rows {
		rowCols := make([]string, len(headers))
		for i := 0; i < len(headers); i++ {
			if i < len(row) {
				rowCols[i] = row[i]
			} else {
				rowCols[i] = ""
			}
		}
		fmt.Fprintln(tw, strings.Join(rowCols, "\t"))
	}

	return tw.Flush()
}

// TreeNode represents a node in a hierarchical tree view.
type TreeNode struct {
	Label    string
	Children []*TreeNode
}

// Tree writes a formatted hierarchical tree view.
func (r *ConsoleRenderer) Tree(root *TreeNode) error {
	if root == nil {
		return nil
	}
	if _, err := fmt.Fprintf(r.w, "%s\n", r.Color(ColorBold+ColorCyan, root.Label)); err != nil {
		return err
	}
	return r.renderTreeChildren(root.Children, "")
}

func (r *ConsoleRenderer) renderTreeChildren(children []*TreeNode, prefix string) error {
	count := len(children)
	for i, child := range children {
		isLast := i == count-1
		branch := "├── "
		nextPrefix := prefix + "│   "
		if isLast {
			branch = "└── "
			nextPrefix = prefix + "    "
		}

		line := fmt.Sprintf("%s%s%s\n", prefix, r.Color(ColorGray, branch), child.Label)
		if _, err := fmt.Fprintf(r.w, "%s", line); err != nil {
			return err
		}

		if len(child.Children) > 0 {
			if err := r.renderTreeChildren(child.Children, nextPrefix); err != nil {
				return err
			}
		}
	}
	return nil
}

// ProgressBar renders a deterministic text-based progress bar.
func (r *ConsoleRenderer) ProgressBar(current, total int, label string) {
	if total <= 0 {
		total = 1
	}
	if current > total {
		current = total
	}

	percentage := float64(current) / float64(total) * 100.0
	barWidth := 30
	completed := int((float64(current) / float64(total)) * float64(barWidth))
	if completed > barWidth {
		completed = barWidth
	}

	bar := strings.Repeat("█", completed) + strings.Repeat("░", barWidth-completed)
	coloredBar := r.Color(ColorCyan, bar)
	pctStr := r.Color(ColorBold, fmt.Sprintf("%5.1f%%", percentage))
	countStr := r.Color(ColorGray, fmt.Sprintf("(%d/%d)", current, total))

	fmt.Fprintf(r.w, "\r  %-20s [%s] %s %s", label, coloredBar, pctStr, countStr)
	if current == total {
		fmt.Fprintln(r.w)
	}
}

// StepTracker logs deterministic sequential progress steps.
type StepTracker struct {
	renderer *ConsoleRenderer
	total    int
	current  int
}

// NewStepTracker creates an initialized StepTracker.
func NewStepTracker(renderer *ConsoleRenderer, totalSteps int) *StepTracker {
	return &StepTracker{
		renderer: renderer,
		total:    totalSteps,
		current:  0,
	}
}

// Step advances to the next step and prints status.
func (s *StepTracker) Step(name string) {
	if s == nil || s.renderer == nil {
		return
	}
	s.current++
	stepBadge := s.renderer.Color(ColorBlue+ColorBold, fmt.Sprintf("[%d/%d]", s.current, s.total))
	fmt.Fprintf(s.renderer.w, "  %s %s...\n", stepBadge, name)
}

// Done completes the step tracker.
func (s *StepTracker) Done(msg string) {
	if s == nil || s.renderer == nil {
		return
	}
	_ = s.renderer.Status("SUCCESS", msg)
}
