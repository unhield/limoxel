package logging

import (
	"sort"
	"time"
)

// Field represents a key-value attribute associated with a log Entry.
type Field struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

// Fields represents a map of structured attributes.
type Fields map[string]any

// Clone creates a deep copy of Fields to ensure immutability.
func (f Fields) Clone() Fields {
	if f == nil {
		return nil
	}
	cloned := make(Fields, len(f))
	for k, v := range f {
		cloned[k] = v
	}
	return cloned
}

// SortedFields returns structured fields as a slice sorted by key.
func (f Fields) SortedFields() []Field {
	if len(f) == 0 {
		return nil
	}
	keys := make([]string, 0, len(f))
	for k := range f {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	result := make([]Field, len(keys))
	for i, k := range keys {
		result[i] = Field{Key: k, Value: f[k]}
	}
	return result
}

// Entry represents an immutable, structured platform log payload.
type Entry struct {
	Time    time.Time `json:"time"`
	Level   Level     `json:"level"`
	Message string    `json:"message"`
	Fields  Fields    `json:"fields"`
	Context string    `json:"context,omitempty"`
}

// NewEntry constructs a new immutable Entry.
func NewEntry(level Level, message string, fields Fields, contextName string) Entry {
	now := time.Now()
	var fCopy Fields
	if fields != nil {
		fCopy = fields.Clone()
	} else {
		fCopy = make(Fields)
	}

	return Entry{
		Time:    now,
		Level:   level,
		Message: message,
		Fields:  fCopy,
		Context: contextName,
	}
}

// Clone returns a defensive copy of the Entry.
func (e Entry) Clone() Entry {
	return Entry{
		Time:    e.Time,
		Level:   e.Level,
		Message: e.Message,
		Fields:  e.Fields.Clone(),
		Context: e.Context,
	}
}
