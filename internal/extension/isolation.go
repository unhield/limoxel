package extension

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrNilIsolationValidator indicates an operation was attempted on a nil IsolationValidator.
	ErrNilIsolationValidator = errors.New("extension: isolation validator instance is nil")

	// ErrIsolationViolation indicates an architectural boundary violation between extension descriptors.
	ErrIsolationViolation = errors.New("extension: isolation boundary violation")
)

// Scope represents the architectural scope boundary of an extension.
type Scope struct {
	id        string
	namespace string
}

// NewScope constructs a new immutable Scope boundary for an extension ID and namespace.
func NewScope(id string, namespace string) (*Scope, error) {
	cleanID := strings.ToLower(strings.TrimSpace(id))
	if cleanID == "" {
		return nil, ErrInvalidID
	}

	cleanNS := strings.ToLower(strings.TrimSpace(namespace))
	if cleanNS == "" {
		cleanNS = cleanID
	}

	return &Scope{
		id:        cleanID,
		namespace: cleanNS,
	}, nil
}

// ID returns the extension ID associated with the Scope.
func (s *Scope) ID() string {
	if s == nil {
		return ""
	}
	return s.id
}

// Namespace returns the architectural namespace string of the Scope.
func (s *Scope) Namespace() string {
	if s == nil {
		return ""
	}
	return s.namespace
}

// IsolationValidator enforces deterministic architectural isolation boundaries across extension descriptors.
type IsolationValidator struct{}

// NewIsolationValidator constructs a new IsolationValidator.
func NewIsolationValidator() *IsolationValidator {
	return &IsolationValidator{}
}

// ValidateIsolation checks that target extension descriptor does not violate architectural isolation boundaries with existing descriptors.
func (v *IsolationValidator) ValidateIsolation(target *Descriptor, existing []*Descriptor) error {
	if v == nil {
		return ErrNilIsolationValidator
	}
	if target == nil {
		return ErrNilDescriptor
	}

	targetScope, err := NewScope(target.ID(), target.Metadata()["namespace"])
	if err != nil {
		return fmt.Errorf("%w: invalid target scope: %v", ErrIsolationViolation, err)
	}

	for _, ext := range existing {
		if ext == nil {
			continue
		}
		if ext.ID() == target.ID() {
			return fmt.Errorf("%w: target extension '%s' collides with existing extension ID", ErrIsolationViolation, target.ID())
		}

		extScope, err := NewScope(ext.ID(), ext.Metadata()["namespace"])
		if err != nil {
			continue
		}

		if extScope.Namespace() == targetScope.Namespace() && ext.ID() != target.ID() {
			return fmt.Errorf("%w: extension '%s' namespace '%s' collides with existing extension '%s'", ErrIsolationViolation, target.ID(), targetScope.Namespace(), ext.ID())
		}
	}

	return nil
}
