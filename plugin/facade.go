package plugin

import "context"

// RepositoryMetadata encapsulates basic metadata about the host repository workspace.
type RepositoryMetadata struct {
	Name          string            `json:"name"`
	RootPath      string            `json:"root_path"`
	DefaultBranch string            `json:"default_branch,omitempty"`
	CurrentBranch string            `json:"current_branch,omitempty"`
	IsGit         bool              `json:"is_git"`
	Properties    map[string]string `json:"properties,omitempty"`
}

// RepositoryFacade provides controlled read-only access to repository workspace information.
type RepositoryFacade interface {
	// WorkspacePath returns the absolute filesystem path of the repository workspace root.
	WorkspacePath() string

	// Metadata retrieves structured repository workspace metadata.
	Metadata(ctx context.Context) (*RepositoryMetadata, error)

	// ListFiles enumerates relative file paths matching the given prefix pattern.
	ListFiles(ctx context.Context, prefix string) ([]string, error)
}

// EventHandler represents a callback invoked upon event receipt.
type EventHandler func(ctx context.Context, topic string, payload []byte) error

// Subscription represents an active event subscription that can be cancelled.
type Subscription interface {
	// Unsubscribe terminates the event subscription.
	Unsubscribe() error
}

// EventFacade provides pub/sub event routing for plugins.
type EventFacade interface {
	// Subscribe registers an event handler for the given topic pattern.
	Subscribe(topic string, handler EventHandler) (Subscription, error)

	// Publish broadcasts an event payload to subscribed listeners.
	Publish(ctx context.Context, topic string, payload []byte) error
}

// Logger provides structured logging for plugins.
type Logger interface {
	Debug(msg string, keyvals ...any)
	Info(msg string, keyvals ...any)
	Warn(msg string, keyvals ...any)
	Error(msg string, keyvals ...any)
}
