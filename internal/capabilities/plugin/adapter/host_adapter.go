package adapter

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
	"github.com/unhield/limoxel/plugin"
)

// DefaultCapabilities enumerates the canonical Limoxel platform capabilities available to plugins.
var DefaultCapabilities = []string{
	"repository.metadata",
	"repository.files",
	"events.pubsub",
	"logging.structured",
}

// HostAdapter implements plugin.Host, isolating internal Limoxel state behind safe facades.
type HostAdapter struct {
	workspaceRoot string
	capabilities  map[string]struct{}
	eventBroker   *eventBroker
	loggerOut     io.Writer
}

// NewHostAdapter constructs an initialized HostAdapter for the given workspace.
func NewHostAdapter(workspaceRoot string, customCaps ...string) *HostAdapter {
	caps := make(map[string]struct{}, len(DefaultCapabilities)+len(customCaps))
	for _, c := range DefaultCapabilities {
		caps[c] = struct{}{}
	}
	for _, c := range customCaps {
		caps[strings.TrimSpace(c)] = struct{}{}
	}

	return &HostAdapter{
		workspaceRoot: filepath.Clean(workspaceRoot),
		capabilities:  caps,
		eventBroker:   newEventBroker(),
		loggerOut:     os.Stderr,
	}
}

// HostVersion returns the current canonical Limoxel application version.
func (h *HostAdapter) HostVersion() version.SemVer {
	return version.Current()
}

// Capabilities returns a sorted list of all available host capabilities.
func (h *HostAdapter) Capabilities() []string {
	out := make([]string, 0, len(h.capabilities))
	for c := range h.capabilities {
		out = append(out, c)
	}
	return out
}

// HasCapability checks whether a specific capability is provided by the host.
func (h *HostAdapter) HasCapability(name string) bool {
	_, ok := h.capabilities[strings.TrimSpace(name)]
	return ok
}

// Repository returns the safe repository workspace facade.
func (h *HostAdapter) Repository() plugin.RepositoryFacade {
	return &repositoryFacade{workspaceRoot: h.workspaceRoot}
}

// Events returns the safe event publishing and subscription facade.
func (h *HostAdapter) Events() plugin.EventFacade {
	return h.eventBroker
}

// Logger returns a contextual structured logger for a plugin.
func (h *HostAdapter) Logger() plugin.Logger {
	return &pluginLogger{out: h.loggerOut}
}

type repositoryFacade struct {
	workspaceRoot string
}

func (r *repositoryFacade) WorkspacePath() string {
	return r.workspaceRoot
}

func (r *repositoryFacade) Metadata(ctx context.Context) (*plugin.RepositoryMetadata, error) {
	info, err := os.Stat(r.workspaceRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect workspace %s: %w", r.workspaceRoot, err)
	}

	gitPath := filepath.Join(r.workspaceRoot, ".git")
	_, gitErr := os.Stat(gitPath)
	isGit := gitErr == nil

	return &plugin.RepositoryMetadata{
		Name:          filepath.Base(r.workspaceRoot),
		RootPath:      r.workspaceRoot,
		IsGit:         isGit,
		DefaultBranch: "main",
		CurrentBranch: "main",
		Properties: map[string]string{
			"is_dir": fmt.Sprintf("%t", info.IsDir()),
		},
	}, nil
}

func (r *repositoryFacade) ListFiles(ctx context.Context, prefix string) ([]string, error) {
	var files []string
	cleanPrefix := filepath.Clean(prefix)
	if cleanPrefix == "." {
		cleanPrefix = ""
	}

	err := filepath.Walk(r.workspaceRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip unreadable paths safely
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if info.IsDir() {
			if info.Name() == ".git" || info.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(r.workspaceRoot, path)
		if err != nil {
			return nil
		}

		rel = filepath.ToSlash(rel)
		if cleanPrefix == "" || strings.HasPrefix(rel, cleanPrefix) {
			files = append(files, rel)
		}
		return nil
	})

	return files, err
}

type eventBroker struct {
	mu          sync.RWMutex
	subscribers map[string]map[int]plugin.EventHandler
	nextID      int
}

func newEventBroker() *eventBroker {
	return &eventBroker{
		subscribers: make(map[string]map[int]plugin.EventHandler),
	}
}

func (b *eventBroker) Subscribe(topic string, handler plugin.EventHandler) (plugin.Subscription, error) {
	if handler == nil {
		return nil, fmt.Errorf("event handler cannot be nil")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.nextID++
	id := b.nextID

	if _, ok := b.subscribers[topic]; !ok {
		b.subscribers[topic] = make(map[int]plugin.EventHandler)
	}
	b.subscribers[topic][id] = handler

	return &eventSubscription{
		broker: b,
		topic:  topic,
		id:     id,
	}, nil
}

func (b *eventBroker) Publish(ctx context.Context, topic string, payload []byte) error {
	b.mu.RLock()
	var handlers []plugin.EventHandler
	for pattern, subs := range b.subscribers {
		if pattern == "*" || pattern == topic || (strings.HasSuffix(pattern, ".*") && strings.HasPrefix(topic, strings.TrimSuffix(pattern, ".*"))) {
			for _, h := range subs {
				handlers = append(handlers, h)
			}
		}
	}
	b.mu.RUnlock()

	for _, h := range handlers {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Execute handler with safe panic containment
			func() {
				defer func() {
					_ = recover() // Contain handler panics
				}()
				_ = h(ctx, topic, payload)
			}()
		}
	}

	return nil
}

func (b *eventBroker) unsubscribe(topic string, id int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if subs, ok := b.subscribers[topic]; ok {
		delete(subs, id)
		if len(subs) == 0 {
			delete(b.subscribers, topic)
		}
	}
}

type eventSubscription struct {
	broker *eventBroker
	topic  string
	id     int
	once   sync.Once
}

func (s *eventSubscription) Unsubscribe() error {
	s.once.Do(func() {
		s.broker.unsubscribe(s.topic, s.id)
	})
	return nil
}

type pluginLogger struct {
	out io.Writer
}

func (l *pluginLogger) log(level, msg string, keyvals ...any) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[%s] [%s] %s", time.Now().UTC().Format(time.RFC3339), level, msg))
	for i := 0; i < len(keyvals); i += 2 {
		if i+1 < len(keyvals) {
			sb.WriteString(fmt.Sprintf(" %v=%v", keyvals[i], keyvals[i+1]))
		} else {
			sb.WriteString(fmt.Sprintf(" %v=(missing)", keyvals[i]))
		}
	}
	sb.WriteString("\n")
	_, _ = io.WriteString(l.out, sb.String())
}

func (l *pluginLogger) Debug(msg string, keyvals ...any) { l.log("DEBUG", msg, keyvals...) }
func (l *pluginLogger) Info(msg string, keyvals ...any)  { l.log("INFO", msg, keyvals...) }
func (l *pluginLogger) Warn(msg string, keyvals ...any)  { l.log("WARN", msg, keyvals...) }
func (l *pluginLogger) Error(msg string, keyvals ...any) { l.log("ERROR", msg, keyvals...) }
