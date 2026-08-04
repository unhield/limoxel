package engine

import (
	"fmt"
	"strings"
)

// Config represents the immutable configuration for an Engine instance.
type Config struct {
	id   string
	name string
}

// NewConfig constructs and validates a new immutable Config instance.
func NewConfig(id string, name string) (*Config, error) {
	cleanID := strings.TrimSpace(id)
	if cleanID == "" {
		return nil, fmt.Errorf("%w: engine ID is empty", ErrInvalidConfig)
	}
	if strings.Contains(cleanID, " ") {
		return nil, fmt.Errorf("%w: engine ID cannot contain spaces", ErrInvalidConfig)
	}

	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		return nil, fmt.Errorf("%w: engine name is empty", ErrInvalidConfig)
	}

	return &Config{
		id:   strings.ToLower(cleanID),
		name: cleanName,
	}, nil
}

// ID returns the canonical lower-case engine identifier string.
func (c *Config) ID() string {
	if c == nil {
		return ""
	}
	return c.id
}

// Name returns the human-readable engine name.
func (c *Config) Name() string {
	if c == nil {
		return ""
	}
	return c.name
}
