package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

var (
	// ErrNilWriter indicates an operation was attempted with a nil io.Writer.
	ErrNilWriter = errors.New("cli: writer instance is nil")

	// ErrNilTerminalRenderer indicates an operation was attempted on a nil TerminalRenderer.
	ErrNilTerminalRenderer = errors.New("cli: terminal renderer instance is nil")
)

// MessageType represents the classification of a terminal output message.
type MessageType int

const (
	// MessageInfo represents an informational output message.
	MessageInfo MessageType = iota

	// MessageSuccess represents a successful operation output message.
	MessageSuccess

	// MessageWarning represents a warning output message.
	MessageWarning

	// MessageError represents an error output message.
	MessageError
)

// String returns the human-readable textual prefix representation of MessageType.
func (m MessageType) String() string {
	switch m {
	case MessageInfo:
		return "INFO"
	case MessageSuccess:
		return "SUCCESS"
	case MessageWarning:
		return "WARNING"
	case MessageError:
		return "ERROR"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", int(m))
	}
}

// TerminalRenderer provides deterministic, synchronous formatting and presentation to an io.Writer.
type TerminalRenderer struct {
	stdout io.Writer
	stderr io.Writer
}

// NewTerminalRenderer constructs a new TerminalRenderer targeting stdout and stderr writers.
func NewTerminalRenderer(stdout io.Writer, stderr io.Writer) (*TerminalRenderer, error) {
	if stdout == nil || stderr == nil {
		return nil, ErrNilWriter
	}

	return &TerminalRenderer{
		stdout: stdout,
		stderr: stderr,
	}, nil
}

// Stdout returns the stdout io.Writer associated with the TerminalRenderer.
func (r *TerminalRenderer) Stdout() io.Writer {
	if r == nil {
		return nil
	}
	return r.stdout
}

// Stderr returns the stderr io.Writer associated with the TerminalRenderer.
func (r *TerminalRenderer) Stderr() io.Writer {
	if r == nil {
		return nil
	}
	return r.stderr
}

// RenderMessage formats and writes a classified message string to stdout (or stderr for errors).
func (r *TerminalRenderer) RenderMessage(msgType MessageType, msg string) error {
	if r == nil {
		return ErrNilTerminalRenderer
	}

	clean := strings.TrimSpace(msg)
	out := r.stdout
	if msgType == MessageError {
		out = r.stderr
	}

	formatted := fmt.Sprintf("[%s] %s\n", msgType.String(), clean)
	_, err := out.Write([]byte(formatted))
	if err != nil {
		return fmt.Errorf("cli: failed to write message to terminal: %w", err)
	}

	return nil
}

// RenderCommand formats and writes a single CommandDescriptor metadata representation to stdout.
func (r *TerminalRenderer) RenderCommand(cmd *CommandDescriptor) error {
	if r == nil {
		return ErrNilTerminalRenderer
	}
	if cmd == nil {
		return ErrNilCommandDescriptor
	}

	aliases := strings.Join(cmd.Aliases(), ", ")
	if aliases == "" {
		aliases = "none"
	}

	formatted := fmt.Sprintf("Command:     %s (%s)\nDescription: %s\nCategory:    %s\nAliases:     %s\n",
		cmd.Name(), cmd.ID(), cmd.Description(), cmd.Category(), aliases)

	_, err := r.stdout.Write([]byte(formatted))
	if err != nil {
		return fmt.Errorf("cli: failed to render command: %w", err)
	}

	return nil
}

// RenderCommandList formats and writes a list of CommandDescriptor metadata items as a formatted table to stdout.
func (r *TerminalRenderer) RenderCommandList(commands []*CommandDescriptor) error {
	if r == nil {
		return ErrNilTerminalRenderer
	}
	if len(commands) == 0 {
		return nil
	}

	w := tabwriter.NewWriter(r.stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "COMMAND\tNAME\tCATEGORY\tDESCRIPTION")

	for _, cmd := range commands {
		if cmd == nil {
			continue
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", cmd.ID(), cmd.Name(), cmd.Category(), cmd.Description())
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("cli: failed to flush command table: %w", err)
	}

	return nil
}

// RenderTable formats and writes tabular header and rows to stdout using tabwriter.
func (r *TerminalRenderer) RenderTable(headers []string, rows [][]string) error {
	if r == nil {
		return ErrNilTerminalRenderer
	}
	if len(headers) == 0 && len(rows) == 0 {
		return nil
	}

	w := tabwriter.NewWriter(r.stdout, 0, 0, 3, ' ', 0)
	if len(headers) > 0 {
		fmt.Fprintln(w, strings.Join(headers, "\t"))
	}

	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("cli: failed to flush table: %w", err)
	}

	return nil
}

// RenderKeyValue formats and writes key-value pairs to stdout in a deterministic tabular layout.
func (r *TerminalRenderer) RenderKeyValue(pairs map[string]string) error {
	if r == nil {
		return ErrNilTerminalRenderer
	}
	if len(pairs) == 0 {
		return nil
	}

	w := tabwriter.NewWriter(r.stdout, 0, 0, 3, ' ', 0)
	for k, v := range pairs {
		fmt.Fprintf(w, "%s:\t%s\n", strings.TrimSpace(k), strings.TrimSpace(v))
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("cli: failed to flush key-value output: %w", err)
	}

	return nil
}
