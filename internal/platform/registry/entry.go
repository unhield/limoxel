package registry

import "time"

// ComponentType identifies the functional classification of a registered platform component.
type ComponentType string

// Metadata stores arbitrary key-value attributes associated with a registered component.
type Metadata map[string]string

// Clone creates a deep copy of the Metadata to preserve immutability.
func (m Metadata) Clone() Metadata {
	if m == nil {
		return nil
	}
	cloned := make(Metadata, len(m))
	for k, v := range m {
		cloned[k] = v
	}
	return cloned
}

// Entry represents an authoritative registration record for a platform component.
type Entry struct {
	Name         string        `json:"name"`
	Type         ComponentType `json:"type"`
	Instance     any           `json:"-"`
	Metadata     Metadata      `json:"metadata"`
	RegisteredAt time.Time     `json:"registered_at"`
}

// Clone creates a defensive copy of the Entry to prevent mutation of internal registry state.
func (e Entry) Clone() Entry {
	return Entry{
		Name:         e.Name,
		Type:         e.Type,
		Instance:     e.Instance,
		Metadata:     e.Metadata.Clone(),
		RegisteredAt: e.RegisteredAt,
	}
}
