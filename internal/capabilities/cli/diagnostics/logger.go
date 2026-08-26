package diagnostics

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/cli/config"
)

const defaultRingBufferSize = 500

// Logger provides thread-safe, structured operational logging.
type Logger struct {
	mu           sync.RWMutex
	level        LogLevel
	format       LogFormat
	component    string
	fields       map[string]any
	out          io.Writer
	file         *os.File
	filePath     string
	ringBuffer   []LogEvent
	ringHead     int
	ringCapacity int
	ringCount    int
	redact       bool
}

// LoggerOptions configures a Logger instance.
type LoggerOptions struct {
	Level        LogLevel
	Format       LogFormat
	Component    string
	Output       io.Writer
	FilePath     string
	RingCapacity int
	Redact       bool
}

// NewLogger constructs a configured Logger instance.
func NewLogger(opts LoggerOptions) (*Logger, error) {
	cap := opts.RingCapacity
	if cap <= 0 {
		cap = defaultRingBufferSize
	}

	out := opts.Output
	if out == nil {
		out = os.Stderr
	}

	l := &Logger{
		level:        opts.Level,
		format:       opts.Format,
		component:    opts.Component,
		fields:       make(map[string]any),
		out:          out,
		ringBuffer:   make([]LogEvent, cap),
		ringCapacity: cap,
		redact:       true,
	}

	if opts.FilePath != "" {
		cleanPath := filepath.Clean(strings.TrimSpace(opts.FilePath))
		dir := filepath.Dir(cleanPath)
		if dir != "" && dir != "." {
			_ = os.MkdirAll(dir, 0755)
		}
		f, err := os.OpenFile(cleanPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file %q: %w", cleanPath, err)
		}
		l.file = f
		l.filePath = cleanPath
	}

	return l, nil
}

// Level returns current minimum enabled LogLevel.
func (l *Logger) Level() LogLevel {
	if l == nil {
		return LevelInfo
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

// SetLevel updates the minimum enabled LogLevel.
func (l *Logger) SetLevel(level LogLevel) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetFormat updates the log serialization format.
func (l *Logger) SetFormat(f LogFormat) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.format = f
}

// With returns a child logger enriched with a key-value attribute.
func (l *Logger) With(key string, value any) *Logger {
	if l == nil {
		return nil
	}
	l.mu.RLock()
	defer l.mu.RUnlock()

	child := &Logger{
		level:        l.level,
		format:       l.format,
		component:    l.component,
		fields:       make(map[string]any, len(l.fields)+1),
		out:          l.out,
		file:         l.file,
		filePath:     l.filePath,
		ringBuffer:   l.ringBuffer,
		ringCapacity: l.ringCapacity,
		redact:       l.redact,
	}
	for k, v := range l.fields {
		child.fields[k] = v
	}
	child.fields[key] = value
	return child
}

// WithFields returns a child logger enriched with multiple fields.
func (l *Logger) WithFields(fields map[string]any) *Logger {
	if l == nil {
		return nil
	}
	l.mu.RLock()
	defer l.mu.RUnlock()

	child := &Logger{
		level:        l.level,
		format:       l.format,
		component:    l.component,
		fields:       make(map[string]any, len(l.fields)+len(fields)),
		out:          l.out,
		file:         l.file,
		filePath:     l.filePath,
		ringBuffer:   l.ringBuffer,
		ringCapacity: l.ringCapacity,
		redact:       l.redact,
	}
	for k, v := range l.fields {
		child.fields[k] = v
	}
	for k, v := range fields {
		child.fields[k] = v
	}
	return child
}

// WithComponent returns a child logger tagged with a component name.
func (l *Logger) WithComponent(component string) *Logger {
	if l == nil {
		return nil
	}
	l.mu.RLock()
	defer l.mu.RUnlock()

	child := &Logger{
		level:        l.level,
		format:       l.format,
		component:    component,
		fields:       make(map[string]any, len(l.fields)),
		out:          l.out,
		file:         l.file,
		filePath:     l.filePath,
		ringBuffer:   l.ringBuffer,
		ringCapacity: l.ringCapacity,
		redact:       l.redact,
	}
	for k, v := range l.fields {
		child.fields[k] = v
	}
	return child
}

// Log emits a structured log entry at target LogLevel.
func (l *Logger) Log(level LogLevel, msg string, fields ...map[string]any) {
	if l == nil {
		return
	}
	if level < l.Level() {
		return
	}

	mergedContext := make(map[string]any)
	l.mu.RLock()
	for k, v := range l.fields {
		mergedContext[k] = v
	}
	for _, fMap := range fields {
		for k, v := range fMap {
			mergedContext[k] = v
		}
	}
	component := l.component
	redact := l.redact
	l.mu.RUnlock()

	if redact {
		mergedContext = config.RedactMap(mergedContext)
	}

	event := LogEvent{
		Timestamp: time.Now().UTC(),
		Level:     level,
		LevelName: level.String(),
		Component: component,
		Message:   msg,
		Context:   mergedContext,
	}

	// Extract standard contextual properties if present
	if op, ok := mergedContext["operation"].(string); ok {
		event.Operation = op
	}
	if ev, ok := mergedContext["event"].(string); ok {
		event.Event = ev
	}
	if errVal, ok := mergedContext["error"]; ok {
		event.Error = fmt.Sprint(errVal)
	}
	if reqID, ok := mergedContext["request_id"].(string); ok {
		event.RequestID = reqID
	}
	if dur, ok := mergedContext["duration_ms"].(float64); ok {
		event.DurationMs = dur
	}

	l.writeEvent(event)
}

// Debug logs at LevelDebug.
func (l *Logger) Debug(msg string, fields ...map[string]any) {
	l.Log(LevelDebug, msg, fields...)
}

// Info logs at LevelInfo.
func (l *Logger) Info(msg string, fields ...map[string]any) {
	l.Log(LevelInfo, msg, fields...)
}

// Warn logs at LevelWarn.
func (l *Logger) Warn(msg string, fields ...map[string]any) {
	l.Log(LevelWarn, msg, fields...)
}

// Error logs at LevelError.
func (l *Logger) Error(msg string, fields ...map[string]any) {
	l.Log(LevelError, msg, fields...)
}

// Critical logs at LevelCritical.
func (l *Logger) Critical(msg string, fields ...map[string]any) {
	l.Log(LevelCritical, msg, fields...)
}

func (l *Logger) writeEvent(event LogEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Append to ring buffer
	if l.ringCapacity > 0 {
		l.ringBuffer[l.ringHead] = event
		l.ringHead = (l.ringHead + 1) % l.ringCapacity
		if l.ringCount < l.ringCapacity {
			l.ringCount++
		}
	}

	var formatted []byte
	switch l.format {
	case LogFormatJSON:
		data, err := json.Marshal(event)
		if err == nil {
			formatted = append(data, '\n')
		}
	default:
		// Text format: [TIMESTAMP] [LEVEL] [COMPONENT] Message key=value
		var sb strings.Builder
		sb.WriteString(event.Timestamp.Format("2006-01-02 15:04:05.000"))
		sb.WriteString(" [")
		sb.WriteString(event.LevelName)
		sb.WriteString("]")
		if event.Component != "" {
			sb.WriteString(" [")
			sb.WriteString(event.Component)
			sb.WriteString("]")
		}
		sb.WriteString(" ")
		sb.WriteString(event.Message)

		if len(event.Context) > 0 {
			sb.WriteString(" {")
			var keys []string
			for k := range event.Context {
				if k == "operation" || k == "event" || k == "request_id" || k == "duration_ms" {
					continue
				}
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for i, k := range keys {
				if i > 0 {
					sb.WriteString(", ")
				}
				fmt.Fprintf(&sb, "%s: %v", k, event.Context[k])
			}
			sb.WriteString("}")
		}
		if event.Error != "" {
			sb.WriteString(" err=")
			sb.WriteString(event.Error)
		}
		sb.WriteString("\n")
		formatted = []byte(sb.String())
	}

	if l.out != nil && len(formatted) > 0 {
		_, _ = l.out.Write(formatted)
	}
	if l.file != nil && len(formatted) > 0 {
		_, _ = l.file.Write(formatted)
	}
}

// GetRecentLogs retrieves a chronological slice of the most recent log events.
func (l *Logger) GetRecentLogs(limit int) []LogEvent {
	if l == nil {
		return nil
	}
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.ringCount == 0 {
		return []LogEvent{}
	}

	count := l.ringCount
	if limit > 0 && limit < count {
		count = limit
	}

	res := make([]LogEvent, count)
	start := (l.ringHead - l.ringCount + l.ringCapacity) % l.ringCapacity
	if count < l.ringCount {
		start = (l.ringHead - count + l.ringCapacity) % l.ringCapacity
	}

	for i := 0; i < count; i++ {
		idx := (start + i) % l.ringCapacity
		res[i] = l.ringBuffer[idx]
	}

	return res
}

// Close flushes and closes active file logging descriptors.
func (l *Logger) Close() error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		err := l.file.Close()
		l.file = nil
		return err
	}
	return nil
}
