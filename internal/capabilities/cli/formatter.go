package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/cli/reporting"
	origcli "github.com/unhield/limoxel/internal/cli"
)

// Formatter manages deterministic presentation and export of CLI output across all Stage 2 formats.
type Formatter struct {
	renderer   *origcli.TerminalRenderer
	console    *reporting.ConsoleRenderer
	structured *reporting.StructuredExporter
	fileWriter *reporting.SafeFileWriter
	format     OutputFormat
	stdout     io.Writer
	stderr     io.Writer
}

// NewFormatter constructs an initialized Formatter.
func NewFormatter(stdout, stderr io.Writer, format OutputFormat) (*Formatter, error) {
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}

	renderer, err := origcli.NewTerminalRenderer(stdout, stderr)
	if err != nil {
		return nil, err
	}

	if format == "" {
		format = FormatText
	}

	return &Formatter{
		renderer:   renderer,
		console:    reporting.NewConsoleRenderer(stdout),
		structured: reporting.NewStructuredExporter(),
		fileWriter: reporting.NewSafeFileWriter(),
		format:     format,
		stdout:     stdout,
		stderr:     stderr,
	}, nil
}

// Format returns the active output format.
func (f *Formatter) Format() OutputFormat {
	if f == nil {
		return FormatText
	}
	return f.format
}

// SetFormat configures the active output format.
func (f *Formatter) SetFormat(fmt OutputFormat) {
	if f == nil {
		return
	}
	f.format = fmt
}

// Stdout returns the underlying stdout writer.
func (f *Formatter) Stdout() io.Writer {
	if f == nil {
		return os.Stdout
	}
	return f.stdout
}

// Stderr returns the underlying stderr writer.
func (f *Formatter) Stderr() io.Writer {
	if f == nil {
		return os.Stderr
	}
	return f.stderr
}

// Console returns the reporting.ConsoleRenderer instance.
func (f *Formatter) Console() *reporting.ConsoleRenderer {
	if f == nil {
		return nil
	}
	return f.console
}

// FileWriter returns the reporting.SafeFileWriter instance.
func (f *Formatter) FileWriter() *reporting.SafeFileWriter {
	if f == nil {
		return nil
	}
	return f.fileWriter
}

// RenderJSON serializes payload as deterministic indented JSON to stdout.
func (f *Formatter) RenderJSON(payload any) error {
	if f == nil {
		return origcli.ErrNilTerminalRenderer
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON output: %w", err)
	}
	_, err = fmt.Fprintln(f.stdout, string(data))
	return err
}

// RenderStructured exports data in the currently active structured format (JSON, YAML, TOML, XML, CSV).
func (f *Formatter) RenderStructured(data any) error {
	if f == nil {
		return origcli.ErrNilTerminalRenderer
	}
	repFmt, err := reporting.ParseFormat(string(f.format))
	if err != nil {
		return f.RenderJSON(data)
	}
	return f.structured.Export(repFmt, data, f.stdout)
}

// RenderMessage writes a classified message to stdout/stderr.
func (f *Formatter) RenderMessage(msgType origcli.MessageType, msg string) error {
	if f == nil {
		return origcli.ErrNilTerminalRenderer
	}
	if f.format == FormatJSON {
		return f.RenderJSON(map[string]string{
			"type":    msgType.String(),
			"message": msg,
		})
	}
	if f.format == FormatYAML || f.format == FormatTOML || f.format == FormatXML {
		return f.RenderStructured(map[string]string{
			"type":    msgType.String(),
			"message": msg,
		})
	}
	return f.renderer.RenderMessage(msgType, msg)
}

// RenderInfo prints an informational message.
func (f *Formatter) RenderInfo(msg string) error {
	return f.RenderMessage(origcli.MessageInfo, msg)
}

// RenderSuccess prints a success message.
func (f *Formatter) RenderSuccess(msg string) error {
	return f.RenderMessage(origcli.MessageSuccess, msg)
}

// RenderWarning prints a warning message.
func (f *Formatter) RenderWarning(msg string) error {
	return f.RenderMessage(origcli.MessageWarning, msg)
}

// RenderError prints an error message.
func (f *Formatter) RenderError(msg string) error {
	return f.RenderMessage(origcli.MessageError, msg)
}

// RenderStatus prints a colored status badge and message.
func (f *Formatter) RenderStatus(status, msg string) error {
	if f == nil {
		return nil
	}
	return f.console.Status(status, msg)
}

// RenderKeyValue writes aligned key-value pairs to stdout.
func (f *Formatter) RenderKeyValue(pairs [][2]string) error {
	if f == nil {
		return origcli.ErrNilTerminalRenderer
	}
	if f.format != FormatText {
		kvMap := make(map[string]string, len(pairs))
		for _, p := range pairs {
			kvMap[p[0]] = p[1]
		}
		return f.RenderStructured(kvMap)
	}
	return f.console.KeyValues(pairs)
}

// RenderTable writes aligned tabular data.
func (f *Formatter) RenderTable(headers []string, rows [][]string) error {
	if f == nil {
		return origcli.ErrNilTerminalRenderer
	}
	if f.format == FormatJSON || f.format == FormatYAML || f.format == FormatTOML || f.format == FormatXML || f.format == FormatCSV {
		items := make([]map[string]string, len(rows))
		for i, row := range rows {
			item := make(map[string]string)
			for j, h := range headers {
				if j < len(row) {
					item[h] = row[j]
				}
			}
			items[i] = item
		}
		return f.RenderStructured(items)
	}
	return f.renderer.RenderTable(headers, rows)
}

// RenderTree writes a formatted hierarchical tree view.
func (f *Formatter) RenderTree(root *reporting.TreeNode) error {
	if f == nil {
		return nil
	}
	if f.format != FormatText {
		return f.RenderStructured(root)
	}
	return f.console.Tree(root)
}

// RenderSection prints a visually distinct section header.
func (f *Formatter) RenderSection(title string) {
	if f == nil || f.format != FormatText {
		return
	}
	fmt.Fprintf(f.stdout, "\n--- %s ---\n", strings.ToUpper(title))
}

// WriteOrPrint exports content or writes it safely to outputFile if specified.
func (f *Formatter) WriteOrPrint(outputFile string, exportFunc func(w io.Writer) error) error {
	if strings.TrimSpace(outputFile) == "" {
		return exportFunc(f.stdout)
	}

	var buf bytes.Buffer
	if err := exportFunc(&buf); err != nil {
		return err
	}

	if err := f.fileWriter.WriteFile(outputFile, buf.Bytes(), true); err != nil {
		return err
	}

	_ = f.RenderSuccess(fmt.Sprintf("Export successfully written to %q (%d bytes)", outputFile, buf.Len()))
	return nil
}
