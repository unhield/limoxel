package logging

import (
	"fmt"
	"strings"
)

// Level defines the severity ranking for platform log messages.
type Level int8

const (
	// LevelTrace designates fine-grained informational events for detailed debugging.
	LevelTrace Level = iota - 2

	// LevelDebug designates fine-grained events useful to debug an application.
	LevelDebug

	// LevelInfo designates informational messages highlighting operational progress.
	LevelInfo

	// LevelWarn designates potentially harmful situations or warning conditions.
	LevelWarn

	// LevelError designates error events that might still allow execution to continue.
	LevelError

	// LevelFatal designates severe error events that undermine operational capability.
	LevelFatal
)

// String returns the canonical uppercase string representation of Level.
func (l Level) String() string {
	switch l {
	case LevelTrace:
		return "TRACE"
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", int8(l))
	}
}

// Enabled reports whether severity l meets or exceeds target level threshold.
func (l Level) Enabled(target Level) bool {
	return l >= target
}

// ParseLevel parses a string representation into a Level.
func ParseLevel(s string) (Level, error) {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "TRACE":
		return LevelTrace, nil
	case "DEBUG":
		return LevelDebug, nil
	case "INFO":
		return LevelInfo, nil
	case "WARN", "WARNING":
		return LevelWarn, nil
	case "ERROR":
		return LevelError, nil
	case "FATAL":
		return LevelFatal, nil
	default:
		return LevelInfo, fmt.Errorf("%w: %q", ErrInvalidLevel, s)
	}
}
