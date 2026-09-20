package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/identity"
)

var (
	ErrAuditTampered = errors.New("enterprise audit: audit log integrity verification failed; hash mismatch")
)

// AuditEvent represents an immutable, tamper-evident record of an enterprise operation.
type AuditEvent struct {
	Sequence         int64                  `json:"sequence"`
	Timestamp        time.Time              `json:"timestamp"`
	OrganizationID   string                 `json:"organization_id"`
	ActorID          string                 `json:"actor_id"`
	ActorType        identity.PrincipalType `json:"actor_type"`
	Action           string                 `json:"action"`
	Target           string                 `json:"target"`
	PluginID         string                 `json:"plugin_id,omitempty"`
	Version          string                 `json:"version,omitempty"`
	DeploymentTarget string                 `json:"deployment_target,omitempty"`
	PolicyResult     string                 `json:"policy_result,omitempty"`
	Outcome          string                 `json:"outcome"`
	Reason           string                 `json:"reason,omitempty"`
	CorrelationID    string                 `json:"correlation_id"`
	PrevHash         string                 `json:"prev_hash"`
	Hash             string                 `json:"hash"`
}

// Logger maintains an append-only, tamper-evident audit log with SHA-256 hash chaining.
type Logger struct {
	mu       sync.Mutex
	logPath  string
	lastHash string
	lastSeq  int64
}

// NewLogger constructs an initialized audit logger.
func NewLogger(logDir string) (*Logger, error) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create audit log directory: %w", err)
	}

	logPath := filepath.Join(logDir, "enterprise_audit.log")
	l := &Logger{
		logPath:  logPath,
		lastHash: "0000000000000000000000000000000000000000000000000000000000000000",
		lastSeq:  0,
	}

	// Read existing log to establish tail hash and sequence
	if err := l.initializeTail(); err != nil {
		return nil, err
	}

	return l, nil
}

func (l *Logger) initializeTail() error {
	events, err := l.ReadEvents()
	if err != nil {
		return err
	}
	if len(events) > 0 {
		tail := events[len(events)-1]
		l.lastSeq = tail.Sequence
		l.lastHash = tail.Hash
	}
	return nil
}

func computeEventHash(e *AuditEvent) string {
	hasher := sha256.New()
	raw := fmt.Sprintf("%d:%s:%s:%s:%s:%s:%s:%s:%s:%s:%s:%s:%s:%s:%s",
		e.Sequence,
		e.Timestamp.Format(time.RFC3339Nano),
		e.OrganizationID,
		e.ActorID,
		e.ActorType,
		e.Action,
		e.Target,
		e.PluginID,
		e.Version,
		e.DeploymentTarget,
		e.PolicyResult,
		e.Outcome,
		e.Reason,
		e.CorrelationID,
		e.PrevHash,
	)
	hasher.Write([]byte(raw))
	return hex.EncodeToString(hasher.Sum(nil))
}

// RecordEvent appends a new signed and chained audit record.
func (l *Logger) RecordEvent(event AuditEvent) (*AuditEvent, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.lastSeq++
	event.Sequence = l.lastSeq
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	event.PrevHash = l.lastHash
	event.Hash = computeEventHash(&event)

	data, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal audit event: %w", err)
	}

	f, err := os.OpenFile(l.logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log for appending: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return nil, fmt.Errorf("failed to append audit event: %w", err)
	}

	l.lastHash = event.Hash
	return &event, nil
}

// ReadEvents returns all recorded audit events in chronological order after verifying chain integrity.
func (l *Logger) ReadEvents() ([]AuditEvent, error) {
	events, err := l.readEventsRaw()
	if err != nil {
		return nil, err
	}
	if err := verifyEvents(events); err != nil {
		return nil, err
	}
	return events, nil
}

// VerifyIntegrity traverses the cryptographic chain from genesis to tail.
func (l *Logger) VerifyIntegrity() (bool, error) {
	events, err := l.readEventsRaw()
	if err != nil {
		return false, err
	}
	if err := verifyEvents(events); err != nil {
		return false, err
	}
	return true, nil
}

func (l *Logger) readEventsRaw() ([]AuditEvent, error) {
	data, err := os.ReadFile(l.logPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read audit log: %w", err)
	}

	var events []AuditEvent
	lines := splitLines(data)
	for idx, line := range lines {
		if len(line) == 0 {
			continue
		}
		var ev AuditEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			return nil, fmt.Errorf("corrupted audit event at line %d: %w", idx+1, err)
		}
		events = append(events, ev)
	}
	return events, nil
}

func verifyEvents(events []AuditEvent) error {
	expectedPrev := "0000000000000000000000000000000000000000000000000000000000000000"
	var expectedSeq int64 = 1

	for _, ev := range events {
		if ev.Sequence != expectedSeq {
			return fmt.Errorf("%w: sequence gap at seq %d, expected %d", ErrAuditTampered, ev.Sequence, expectedSeq)
		}
		if ev.PrevHash != expectedPrev {
			return fmt.Errorf("%w: previous hash mismatch at seq %d", ErrAuditTampered, ev.Sequence)
		}
		calcHash := computeEventHash(&ev)
		if ev.Hash != calcHash {
			return fmt.Errorf("%w: hash tampering detected at seq %d", ErrAuditTampered, ev.Sequence)
		}
		expectedPrev = ev.Hash
		expectedSeq++
	}

	return nil
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}
