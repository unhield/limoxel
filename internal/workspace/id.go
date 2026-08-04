package workspace

import (
	"fmt"
	"strings"
)

// ID represents an immutable, strongly typed, validated workspace identifier value object.
type ID struct {
	value string
}

// NewID constructs and validates a new immutable ID.
func NewID(value string) (ID, error) {
	cleanVal := strings.TrimSpace(value)
	if cleanVal == "" {
		return ID{}, ErrInvalidID
	}
	if strings.Contains(cleanVal, " ") {
		return ID{}, fmt.Errorf("%w: identifier cannot contain spaces", ErrInvalidID)
	}
	return ID{value: cleanVal}, nil
}

// Value returns the raw string value of the ID.
func (id ID) Value() string {
	return id.value
}

// IsEmpty reports whether the ID is uninitialized or empty.
func (id ID) IsEmpty() bool {
	return id.value == ""
}
