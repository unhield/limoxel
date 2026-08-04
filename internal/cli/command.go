package cli

import (
	"fmt"
	"strings"
)

// CommandDescriptor represents immutable metadata describing a CLI command.
type CommandDescriptor struct {
	id          string
	name        string
	description string
	aliases     []string
	category    string
	hidden      bool
}

// NewCommandDescriptor constructs and validates a new immutable CommandDescriptor.
func NewCommandDescriptor(id string, name string, description string, aliases []string, category string, hidden bool) (*CommandDescriptor, error) {
	cleanID := strings.TrimSpace(id)
	if cleanID == "" {
		return nil, ErrInvalidID
	}
	if strings.Contains(cleanID, " ") {
		return nil, fmt.Errorf("%w: command ID cannot contain spaces", ErrInvalidID)
	}

	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		return nil, ErrInvalidName
	}

	cleanAliases := make([]string, 0, len(aliases))
	seenAlias := make(map[string]struct{})
	for _, alias := range aliases {
		a := strings.ToLower(strings.TrimSpace(alias))
		if a != "" && a != strings.ToLower(cleanID) {
			if _, exists := seenAlias[a]; !exists {
				seenAlias[a] = struct{}{}
				cleanAliases = append(cleanAliases, a)
			}
		}
	}

	return &CommandDescriptor{
		id:          strings.ToLower(cleanID),
		name:        cleanName,
		description: strings.TrimSpace(description),
		aliases:     cleanAliases,
		category:    strings.TrimSpace(category),
		hidden:      hidden,
	}, nil
}

// ID returns the canonical lower-case command identifier string.
func (c *CommandDescriptor) ID() string {
	if c == nil {
		return ""
	}
	return c.id
}

// Name returns the display name of the command.
func (c *CommandDescriptor) Name() string {
	if c == nil {
		return ""
	}
	return c.name
}

// Description returns the summary description of the command.
func (c *CommandDescriptor) Description() string {
	if c == nil {
		return ""
	}
	return c.description
}

// Aliases returns a defensive copy of command alias strings.
func (c *CommandDescriptor) Aliases() []string {
	if c == nil || len(c.aliases) == 0 {
		return nil
	}
	cloned := make([]string, len(c.aliases))
	copy(cloned, c.aliases)
	return cloned
}

// Category returns the organizational category string of the command.
func (c *CommandDescriptor) Category() string {
	if c == nil {
		return ""
	}
	return c.category
}

// Hidden reports whether the command is hidden from standard help displays.
func (c *CommandDescriptor) Hidden() bool {
	if c == nil {
		return false
	}
	return c.hidden
}
