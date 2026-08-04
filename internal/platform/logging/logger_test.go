package logging_test

import (
	"context"
	"sync"
	"testing"

	"github.com/unhield/limoxel/internal/platform/logging"
)

type mockHandler struct {
	mu      sync.Mutex
	name    string
	minLvl  logging.Level
	entries []logging.Entry
}

func newMockHandler(name string, minLvl logging.Level) *mockHandler {
	return &mockHandler{
		name:    name,
		minLvl:  minLvl,
		entries: make([]logging.Entry, 0),
	}
}

func (m *mockHandler) Name() string { return m.name }
func (m *mockHandler) Enabled(level logging.Level) bool {
	return level >= m.minLvl
}
func (m *mockHandler) Handle(ctx context.Context, entry logging.Entry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = append(m.entries, entry)
	return nil
}
func (m *mockHandler) Entries() []logging.Entry {
	m.mu.Lock()
	defer m.mu.Unlock()
	res := make([]logging.Entry, len(m.entries))
	copy(res, m.entries)
	return res
}

func TestLevelParsingAndString(t *testing.T) {
	levels := map[logging.Level]string{
		logging.LevelTrace: "TRACE",
		logging.LevelDebug: "DEBUG",
		logging.LevelInfo:  "INFO",
		logging.LevelWarn:  "WARN",
		logging.LevelError: "ERROR",
		logging.LevelFatal: "FATAL",
		logging.Level(99):  "UNKNOWN(99)",
	}

	for lvl, str := range levels {
		if lvl.String() != str {
			t.Errorf("got %q, want %q", lvl.String(), str)
		}
	}

	parseTests := []struct {
		input string
		want  logging.Level
		err   bool
	}{
		{"trace", logging.LevelTrace, false},
		{"DEBUG", logging.LevelDebug, false},
		{"info", logging.LevelInfo, false},
		{"WARN", logging.LevelWarn, false},
		{"WARNING", logging.LevelWarn, false},
		{"error", logging.LevelError, false},
		{"FATAL", logging.LevelFatal, false},
		{"invalid", logging.LevelInfo, true},
	}

	for _, tt := range parseTests {
		got, err := logging.ParseLevel(tt.input)
		if tt.err && err == nil {
			t.Errorf("expected error parsing %q", tt.input)
		}
		if !tt.err && (err != nil || got != tt.want) {
			t.Errorf("parse %q got %v (%v), want %v", tt.input, got, err, tt.want)
		}
	}
}

func TestFieldsAndSortedFields(t *testing.T) {
	fields := logging.Fields{
		"z": 100,
		"a": "first",
		"m": true,
	}

	cloned := fields.Clone()
	if len(cloned) != 3 {
		t.Fatalf("got len %d, want 3", len(cloned))
	}
	cloned["z"] = 999
	if fields["z"] == 999 {
		t.Error("mutation leaked to original fields")
	}

	sorted := fields.SortedFields()
	if len(sorted) != 3 {
		t.Fatalf("got %d sorted fields, want 3", len(sorted))
	}
	if sorted[0].Key != "a" || sorted[1].Key != "m" || sorted[2].Key != "z" {
		t.Errorf("fields not sorted correctly: %v", sorted)
	}
}

func TestCoreLoggerEmissions(t *testing.T) {
	handler := newMockHandler("test-handler", logging.LevelDebug)
	logger := logging.New(logging.LevelDebug, handler, nil)

	ctx := context.Background()

	logger.Trace(ctx, "trace msg - skipped")
	logger.Debug(ctx, "debug msg")
	logger.Info(ctx, "info msg")
	logger.Warn(ctx, "warn msg")
	logger.Error(ctx, "error msg")
	logger.Fatal(ctx, "fatal msg")

	entries := handler.Entries()
	if len(entries) != 5 {
		t.Fatalf("expected 5 entries, got %d", len(entries))
	}

	if entries[0].Message != "debug msg" || entries[0].Level != logging.LevelDebug {
		t.Errorf("unexpected entry 0: %v", entries[0])
	}
}

func TestCoreLoggerWithFieldsAndContext(t *testing.T) {
	handler := newMockHandler("handler", logging.LevelInfo)
	parent := logging.New(logging.LevelInfo, handler)

	childCtx := parent.WithContext("subsystem-x")
	childFields := childCtx.With("req_id", "123").WithFields(logging.Fields{"env": "prod"})

	ctx := context.Background()
	childFields.Info(ctx, "processed request")

	entries := handler.Entries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	e := entries[0]
	if e.Context != "subsystem-x" {
		t.Errorf("got context %q, want subsystem-x", e.Context)
	}
	if e.Fields["req_id"] != "123" || e.Fields["env"] != "prod" {
		t.Errorf("got fields %v, want req_id and env", e.Fields)
	}
}

func TestNilCoreLoggerSafety(t *testing.T) {
	var l *logging.CoreLogger

	if l.Level() != logging.LevelInfo {
		t.Errorf("got level %v, want LevelInfo", l.Level())
	}
	if l.Enabled(logging.LevelInfo) {
		t.Error("nil logger should not be enabled")
	}
	if l.With("k", "v") != nil {
		t.Error("nil logger With should return nil")
	}
	if l.WithFields(logging.Fields{"k": "v"}) != nil {
		t.Error("nil logger WithFields should return nil")
	}
	if l.WithContext("ctx") != nil {
		t.Error("nil logger WithContext should return nil")
	}

	ctx := context.Background()
	l.Log(ctx, logging.LevelInfo, "test")
	l.Trace(ctx, "test")
	l.Debug(ctx, "test")
	l.Info(ctx, "test")
	l.Warn(ctx, "test")
	l.Error(ctx, "test")
	l.Fatal(ctx, "test")
}
