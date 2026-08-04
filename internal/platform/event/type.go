package event

import (
	"fmt"
	"strings"
)

// Type represents a canonical event classification string.
type Type string

const (
	// TypePlatformCreated designates platform creation events.
	TypePlatformCreated Type = "platform.created"

	// TypePlatformInitialized designates platform initialization events.
	TypePlatformInitialized Type = "platform.initialized"

	// TypePlatformPrepared designates platform preparation events.
	TypePlatformPrepared Type = "platform.prepared"

	// TypePlatformStarted designates platform startup events.
	TypePlatformStarted Type = "platform.started"

	// TypePlatformStopped designates platform shutdown events.
	TypePlatformStopped Type = "platform.stopped"

	// TypePlatformTerminated designates platform termination events.
	TypePlatformTerminated Type = "platform.terminated"

	// TypeComponentRegistered designates component registration events.
	TypeComponentRegistered Type = "component.registered"

	// TypeComponentUnregistered designates component unregistration events.
	TypeComponentUnregistered Type = "component.unregistered"
)

// String returns the string representation of Type.
func (t Type) String() string {
	return string(t)
}

// ValidateType verifies that a Type is non-empty and well-formed.
func ValidateType(t Type) error {
	if t == "" {
		return ErrTypeEmpty
	}
	if strings.Contains(string(t), " ") {
		return fmt.Errorf("%w: type cannot contain spaces", ErrTypeEmpty)
	}
	return nil
}
