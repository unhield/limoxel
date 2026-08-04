package event

import "sort"

// Metadata stores arbitrary key-value attributes associated with an Event.
type Metadata map[string]string

// Clone creates a deep copy of Metadata to preserve immutability.
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

// Get returns the metadata value for key.
func (m Metadata) Get(key string) string {
	if m == nil {
		return ""
	}
	return m[key]
}

// Has reports whether key exists in the Metadata.
func (m Metadata) Has(key string) bool {
	if m == nil {
		return false
	}
	_, exists := m[key]
	return exists
}

// Keys returns a sorted, deterministic slice of metadata keys.
func (m Metadata) Keys() []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
