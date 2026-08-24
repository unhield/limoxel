package discovery

// Options configures the behavior and safety limits of repository discovery.
type Options struct {
	// MaxDepth specifies the maximum directory recursion depth (0 means unlimited).
	MaxDepth int

	// MaxFiles specifies the maximum number of discovered files (0 means unlimited).
	MaxFiles int

	// FollowSymlinks specifies whether symbolic links should be followed within repository boundaries.
	FollowSymlinks bool

	// IncludeHidden specifies whether hidden dotfiles (e.g. .gitignore) are included in inventory.
	IncludeHidden bool

	// AdditionalIgnoreRules specifies extra exclusion rules in addition to default rules.
	AdditionalIgnoreRules []string
}

// DefaultOptions returns the canonical safe default options for repository discovery.
func DefaultOptions() Options {
	return Options{
		MaxDepth:              0,
		MaxFiles:              0,
		FollowSymlinks:        false,
		IncludeHidden:         true,
		AdditionalIgnoreRules: nil,
	}
}
