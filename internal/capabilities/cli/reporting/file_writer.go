package reporting

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SafeFileWriter coordinates safe, validated writing of generated reports and diagrams to disk.
type SafeFileWriter struct{}

// NewSafeFileWriter constructs a new SafeFileWriter.
func NewSafeFileWriter() *SafeFileWriter {
	return &SafeFileWriter{}
}

// WriteFile atomically writes data to destinationPath, creating parent directories as needed.
func (w *SafeFileWriter) WriteFile(destinationPath string, data []byte, overwrite bool) error {
	cleanPath := filepath.Clean(strings.TrimSpace(destinationPath))
	if cleanPath == "" || cleanPath == "." || cleanPath == "/" || cleanPath == "\\" {
		return fmt.Errorf("reporting: invalid destination file path %q", destinationPath)
	}

	// Check existing file
	if !overwrite {
		if _, err := os.Stat(cleanPath); err == nil {
			return fmt.Errorf("reporting: destination file %q already exists (use overwrite if intended)", cleanPath)
		}
	}

	// Ensure parent directory exists
	dir := filepath.Dir(cleanPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("reporting: failed to create destination directory %q: %w", dir, err)
		}
	}

	// Write to temporary file in same directory first to ensure atomic write
	tmpFile := cleanPath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("reporting: failed to write temporary file %q: %w", tmpFile, err)
	}

	// Rename temporary file to destination
	if err := os.Rename(tmpFile, cleanPath); err != nil {
		_ = os.Remove(tmpFile)
		// Fallback direct write if rename fails (e.g. cross-volume on Windows)
		if errDirect := os.WriteFile(cleanPath, data, 0644); errDirect != nil {
			return fmt.Errorf("reporting: failed to write destination file %q: %w", cleanPath, errDirect)
		}
	}

	return nil
}

// WriteStream executes writerFunc writing to a buffer, then writes the buffer safely to destinationPath.
func (w *SafeFileWriter) WriteStream(destinationPath string, writerFunc func(buf *bytes.Buffer) error, overwrite bool) error {
	var buf bytes.Buffer
	if err := writerFunc(&buf); err != nil {
		return err
	}
	return w.WriteFile(destinationPath, buf.Bytes(), overwrite)
}
