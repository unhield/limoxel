package logging

import (
	"context"
	"sync"
)

// Handler defines the contract for processing platform log entries.
type Handler interface {
	// Name returns the unique identifier for the handler.
	Name() string

	// Handle processes a single log Entry.
	Handle(ctx context.Context, entry Entry) error

	// Enabled reports whether the handler processes logs at the target Level.
	Enabled(level Level) bool
}

// Logger defines the platform logging contract interface.
type Logger interface {
	// With returns a child logger enriched with the given key-value field.
	With(key string, value any) Logger

	// WithFields returns a child logger enriched with the given structured Fields.
	WithFields(fields Fields) Logger

	// WithContext returns a child logger associated with a specific component/subsystem context name.
	WithContext(name string) Logger

	// Log emits a log entry at the specified Level.
	Log(ctx context.Context, level Level, msg string)

	// Trace emits a log entry at LevelTrace.
	Trace(ctx context.Context, msg string)

	// Debug emits a log entry at LevelDebug.
	Debug(ctx context.Context, msg string)

	// Info emits a log entry at LevelInfo.
	Info(ctx context.Context, msg string)

	// Warn emits a log entry at LevelWarn.
	Warn(ctx context.Context, msg string)

	// Error emits a log entry at LevelError.
	Error(ctx context.Context, msg string)

	// Fatal emits a log entry at LevelFatal.
	Fatal(ctx context.Context, msg string)

	// Level returns the minimum enabled Level for this logger.
	Level() Level

	// Enabled reports whether the logger emits logs at target Level.
	Enabled(level Level) bool
}

// CoreLogger is a thread-safe, contract-driven implementation of Logger.
type CoreLogger struct {
	mu          sync.RWMutex
	level       Level
	contextName string
	fields      Fields
	handlers    []Handler
}

// New constructs a new CoreLogger with the minimum Level threshold and handlers.
func New(minLevel Level, handlers ...Handler) *CoreLogger {
	hList := make([]Handler, 0, len(handlers))
	for _, h := range handlers {
		if h != nil {
			hList = append(hList, h)
		}
	}

	return &CoreLogger{
		level:    minLevel,
		fields:   make(Fields),
		handlers: hList,
	}
}

// Level returns the configured minimum Level threshold.
func (l *CoreLogger) Level() Level {
	if l == nil {
		return LevelInfo
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}

// Enabled reports whether level meets or exceeds the logger's threshold.
func (l *CoreLogger) Enabled(level Level) bool {
	if l == nil {
		return false
	}
	return level >= l.Level()
}

// With returns a child CoreLogger enriched with an additional key-value field.
func (l *CoreLogger) With(key string, value any) Logger {
	return l.WithFields(Fields{key: value})
}

// WithFields returns a child CoreLogger enriched with additional structured Fields.
func (l *CoreLogger) WithFields(fields Fields) Logger {
	if l == nil {
		return nil
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	newFields := l.fields.Clone()
	if newFields == nil {
		newFields = make(Fields)
	}

	for k, v := range fields {
		if k != "" {
			newFields[k] = v
		}
	}

	hCopy := make([]Handler, len(l.handlers))
	copy(hCopy, l.handlers)

	return &CoreLogger{
		level:       l.level,
		contextName: l.contextName,
		fields:      newFields,
		handlers:    hCopy,
	}
}

// WithContext returns a child CoreLogger associated with a component context name.
func (l *CoreLogger) WithContext(name string) Logger {
	if l == nil {
		return nil
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	hCopy := make([]Handler, len(l.handlers))
	copy(hCopy, l.handlers)

	return &CoreLogger{
		level:       l.level,
		contextName: name,
		fields:      l.fields.Clone(),
		handlers:    hCopy,
	}
}

// Log emits a log entry at the specified Level if enabled.
func (l *CoreLogger) Log(ctx context.Context, level Level, msg string) {
	if l == nil || !l.Enabled(level) {
		return
	}

	l.mu.RLock()
	entry := NewEntry(level, msg, l.fields, l.contextName)
	handlers := make([]Handler, len(l.handlers))
	copy(handlers, l.handlers)
	l.mu.RUnlock()

	for _, h := range handlers {
		if h != nil && h.Enabled(level) {
			_ = h.Handle(ctx, entry.Clone())
		}
	}
}

// Trace emits a log entry at LevelTrace.
func (l *CoreLogger) Trace(ctx context.Context, msg string) {
	l.Log(ctx, LevelTrace, msg)
}

// Debug emits a log entry at LevelDebug.
func (l *CoreLogger) Debug(ctx context.Context, msg string) {
	l.Log(ctx, LevelDebug, msg)
}

// Info emits a log entry at LevelInfo.
func (l *CoreLogger) Info(ctx context.Context, msg string) {
	l.Log(ctx, LevelInfo, msg)
}

// Warn emits a log entry at LevelWarn.
func (l *CoreLogger) Warn(ctx context.Context, msg string) {
	l.Log(ctx, LevelWarn, msg)
}

// Error emits a log entry at LevelError.
func (l *CoreLogger) Error(ctx context.Context, msg string) {
	l.Log(ctx, LevelError, msg)
}

// Fatal emits a log entry at LevelFatal.
func (l *CoreLogger) Fatal(ctx context.Context, msg string) {
	l.Log(ctx, LevelFatal, msg)
}
