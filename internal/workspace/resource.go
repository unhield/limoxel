package workspace

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrInvalidResourceID indicates a resource identifier is empty or invalid.
	ErrInvalidResourceID = errors.New("workspace: invalid or empty resource ID")

	// ErrInvalidResourcePath indicates a resource logical path is empty or invalid.
	ErrInvalidResourcePath = errors.New("workspace: invalid or empty resource path")

	// ErrNilResource indicates a nil Resource instance was provided.
	ErrNilResource = errors.New("workspace: resource instance is nil")
)

// ResourceID represents an immutable, validated resource identifier value object.
type ResourceID struct {
	value string
}

// NewResourceID constructs and validates a new ResourceID.
func NewResourceID(value string) (ResourceID, error) {
	cleanVal := strings.TrimSpace(value)
	if cleanVal == "" {
		return ResourceID{}, ErrInvalidResourceID
	}
	if strings.Contains(cleanVal, " ") {
		return ResourceID{}, fmt.Errorf("%w: resource ID cannot contain spaces", ErrInvalidResourceID)
	}
	return ResourceID{value: cleanVal}, nil
}

// Value returns the raw string value of ResourceID.
func (id ResourceID) Value() string {
	return id.value
}

// IsEmpty reports whether ResourceID is empty.
func (id ResourceID) IsEmpty() bool {
	return id.value == ""
}

// Resource represents an immutable Workspace resource domain model.
type Resource struct {
	id   ResourceID
	path string
}

// NewResource constructs and validates a new immutable Resource.
func NewResource(idStr string, path string) (*Resource, error) {
	resID, err := NewResourceID(idStr)
	if err != nil {
		return nil, err
	}

	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return nil, ErrInvalidResourcePath
	}

	return &Resource{
		id:   resID,
		path: cleanPath,
	}, nil
}

// ID returns the immutable ResourceID.
func (r *Resource) ID() ResourceID {
	if r == nil {
		return ResourceID{}
	}
	return r.id
}

// Path returns the resource logical path string.
func (r *Resource) Path() string {
	if r == nil {
		return ""
	}
	return r.path
}
