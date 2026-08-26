package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"

	origcli "github.com/unhield/limoxel/internal/cli"
)

// HandlerFunc defines the execution logic of a CLI command or subcommand.
type HandlerFunc func(ctx *Context, flags *Flags) error

// OptionDoc defines a documented command-line option.
type OptionDoc struct {
	Name        string
	Short       string
	Description string
	DefaultVal  string
}

// Command represents an actionable, hierarchical CLI command or subcommand.
type Command struct {
	mu          sync.RWMutex
	id          string
	name        string
	description string
	usage       string
	category    CommandCategory
	aliases     []string
	options     []OptionDoc
	hidden      bool
	handler     HandlerFunc
	subcommands map[string]*Command
	subOrder    []string
	parent      *Command
}

// NewCommand creates a new Command instance.
func NewCommand(name, description, usage string, category CommandCategory, handler HandlerFunc) *Command {
	cleanName := strings.ToLower(strings.TrimSpace(name))
	return &Command{
		id:          cleanName,
		name:        cleanName,
		description: strings.TrimSpace(description),
		usage:       strings.TrimSpace(usage),
		category:    category,
		options:     make([]OptionDoc, 0),
		handler:     handler,
		subcommands: make(map[string]*Command),
		subOrder:    make([]string, 0),
	}
}

// ID returns canonical command identifier.
func (c *Command) ID() string {
	if c == nil {
		return ""
	}
	return c.id
}

// Name returns the command name.
func (c *Command) Name() string {
	if c == nil {
		return ""
	}
	return c.name
}

// Description returns the command description.
func (c *Command) Description() string {
	if c == nil {
		return ""
	}
	return c.description
}

// Usage returns the command usage syntax.
func (c *Command) Usage() string {
	if c == nil {
		return ""
	}
	return c.usage
}

// Category returns the organizational category.
func (c *Command) Category() CommandCategory {
	if c == nil {
		return CategoryGeneral
	}
	return c.category
}

// Aliases returns registered command aliases.
func (c *Command) Aliases() []string {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	res := make([]string, len(c.aliases))
	copy(res, c.aliases)
	return res
}

// AddAlias registers an alternative alias for this command.
func (c *Command) AddAlias(alias string) *Command {
	if c == nil {
		return nil
	}
	clean := strings.ToLower(strings.TrimSpace(alias))
	if clean != "" && clean != c.id {
		c.mu.Lock()
		c.aliases = append(c.aliases, clean)
		c.mu.Unlock()
	}
	return c
}

// AddOption adds a documented option flag to this command.
func (c *Command) AddOption(name, short, description, defaultVal string) *Command {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.options = append(c.options, OptionDoc{
		Name:        name,
		Short:       short,
		Description: description,
		DefaultVal:  defaultVal,
	})
	return c
}

// SetHidden marks the command as hidden from standard help listings.
func (c *Command) SetHidden(hidden bool) *Command {
	if c == nil {
		return nil
	}
	c.hidden = hidden
	return c
}

// IsHidden reports whether the command is hidden.
func (c *Command) IsHidden() bool {
	if c == nil {
		return false
	}
	return c.hidden
}

// AddSubcommand attaches a child subcommand to this command.
func (c *Command) AddSubcommand(sub *Command) *Command {
	if c == nil || sub == nil {
		return c
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	sub.parent = c
	id := sub.ID()
	c.subcommands[id] = sub
	for _, alias := range sub.Aliases() {
		c.subcommands[alias] = sub
	}
	c.subOrder = append(c.subOrder, id)
	return c
}

// GetSubcommand retrieves a registered child subcommand by name or alias.
func (c *Command) GetSubcommand(name string) *Command {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	clean := strings.ToLower(strings.TrimSpace(name))
	return c.subcommands[clean]
}

// Subcommands returns all direct child subcommands in registration order.
func (c *Command) Subcommands() []*Command {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()

	res := make([]*Command, 0, len(c.subOrder))
	seen := make(map[string]bool)
	for _, id := range c.subOrder {
		if sub, ok := c.subcommands[id]; ok && !seen[sub.id] {
			seen[sub.id] = true
			res = append(res, sub)
		}
	}
	return res
}

// FullCommandPath returns the full ancestor path (e.g. "limoxel repo info").
func (c *Command) FullCommandPath() string {
	if c == nil {
		return ""
	}
	if c.parent != nil {
		return c.parent.FullCommandPath() + " " + c.name
	}
	return c.name
}

// Execute executes this command or delegates to child subcommands based on positional arguments.
func (c *Command) Execute(ctx *Context, flags *Flags) error {
	if c == nil {
		return ErrNilCommand
	}

	// Check if first argument is a child subcommand
	if flags != nil && flags.NArg() > 0 {
		subName := flags.Arg(0)
		if sub := c.GetSubcommand(subName); sub != nil {
			// Shift positional arguments for subcommand
			subFlags := &Flags{
				values: flags.values,
				bools:  flags.bools,
				args:   flags.args[1:],
			}
			return sub.Execute(ctx, subFlags)
		}
	}

	// If help requested, render command help
	if flags != nil && flags.Bool("help") {
		c.RenderHelp(ctx.Formatter().Stdout())
		return nil
	}

	// If handler defined, execute it
	if c.handler != nil {
		return c.handler(ctx, flags)
	}

	// If no handler but has subcommands, display help
	if len(c.subcommands) > 0 {
		c.RenderHelp(ctx.Formatter().Stdout())
		return nil
	}

	return UsageError(c.FullCommandPath(), fmt.Sprintf("no execution handler for command %q", c.name))
}

// RenderHelp formats and writes comprehensive command help to w.
func (c *Command) RenderHelp(w io.Writer) {
	if c == nil || w == nil {
		return
	}

	fmt.Fprintf(w, "\nUsage: %s\n", c.Usage())
	if c.description != "" {
		fmt.Fprintf(w, "\nDescription:\n  %s\n", c.description)
	}

	if len(c.aliases) > 0 {
		fmt.Fprintf(w, "\nAliases: %s\n", strings.Join(c.aliases, ", "))
	}

	// Options
	if len(c.options) > 0 {
		fmt.Fprintf(w, "\nOptions:\n")
		for _, opt := range c.options {
			flagsStr := "--" + opt.Name
			if opt.Short != "" {
				flagsStr = fmt.Sprintf("-%s, %s", opt.Short, flagsStr)
			}
			defStr := ""
			if opt.DefaultVal != "" {
				defStr = fmt.Sprintf(" (default: %s)", opt.DefaultVal)
			}
			fmt.Fprintf(w, "  %-24s %s%s\n", flagsStr, opt.Description, defStr)
		}
	}

	// Subcommands
	subs := c.Subcommands()
	if len(subs) > 0 {
		fmt.Fprintf(w, "\nAvailable Subcommands:\n")
		sort.Slice(subs, func(i, j int) bool {
			return subs[i].name < subs[j].name
		})
		for _, sub := range subs {
			if !sub.hidden {
				fmt.Fprintf(w, "  %-16s %s\n", sub.name, sub.description)
			}
		}
		fmt.Fprintf(w, "\nUse \"%s <subcommand> --help\" for more information on a specific subcommand.\n", c.FullCommandPath())
	}
	fmt.Fprintln(w)
}

// ToDescriptor creates an immutable origcli.CommandDescriptor.
func (c *Command) ToDescriptor() (*origcli.CommandDescriptor, error) {
	if c == nil {
		return nil, ErrNilCommand
	}
	return origcli.NewCommandDescriptor(
		c.id,
		c.name,
		c.description,
		c.Aliases(),
		string(c.category),
		c.hidden,
	)
}
