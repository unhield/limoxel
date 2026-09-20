package plugin

import (
	"context"
	"path/filepath"
	"sync"

	"github.com/unhield/limoxel/internal/capabilities/plugin/adapter"
	"github.com/unhield/limoxel/internal/capabilities/plugin/dependency"
	"github.com/unhield/limoxel/internal/capabilities/plugin/discovery"
	"github.com/unhield/limoxel/internal/capabilities/plugin/lifecycle"
	"github.com/unhield/limoxel/internal/capabilities/plugin/loader"
	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
	"github.com/unhield/limoxel/internal/capabilities/plugin/registry"
	"github.com/unhield/limoxel/internal/capabilities/plugin/validation"
	"github.com/unhield/limoxel/plugin"
)

// ServiceOptions specifies configuration parameters for initializing the plugin Service.
type ServiceOptions struct {
	WorkspaceRoot      string
	SearchDirectories  []string
	CustomCapabilities []string
	Loader             loader.Loader
	Host               plugin.Host
}

// Service provides a unified facade orchestrating discovery, validation, dependencies, registry, lifecycle, and runtime loading.
type Service struct {
	mu         sync.Mutex
	workspace  string
	searchDirs []string
	discoverer *discovery.Discoverer
	validator  *validation.Validator
	resolver   *dependency.Resolver
	registry   *registry.Registry
	manager    *lifecycle.Manager
	loader     loader.Loader
	host       plugin.Host
}

// NewService constructs an initialized Service instance.
func NewService(opts ServiceOptions) (*Service, error) {
	wsRoot := filepath.Clean(opts.WorkspaceRoot)
	if wsRoot == "" || wsRoot == "." {
		wsRoot = "."
	}

	host := opts.Host
	if host == nil {
		host = adapter.NewHostAdapter(wsRoot, opts.CustomCapabilities...)
	}

	ldr := opts.Loader
	if ldr == nil {
		ldr = loader.NewInProcessLoader()
	}

	val := validation.NewValidator(host.HostVersion())
	res := dependency.NewResolver()
	disc := discovery.NewDiscoverer()
	reg := registry.NewRegistry()
	mgr := lifecycle.NewManager(reg, ldr, val, res, host)

	return &Service{
		workspace:  wsRoot,
		searchDirs: opts.SearchDirectories,
		discoverer: disc,
		validator:  val,
		resolver:   res,
		registry:   reg,
		manager:    mgr,
		loader:     ldr,
		host:       host,
	}, nil
}

// Host returns the host environment facade.
func (s *Service) Host() plugin.Host {
	return s.host
}

// Discover probes configured or provided search directories for candidate plugins.
func (s *Service) Discover(ctx context.Context, dirs ...string) ([]*discovery.DiscoveredCandidate, error) {
	targetDirs := s.searchDirs
	if len(dirs) > 0 {
		targetDirs = dirs
	}
	return s.discoverer.Discover(ctx, targetDirs...)
}

// Install validates a candidate manifest and registers it into the managed registry.
func (s *Service) Install(ctx context.Context, candidate *discovery.DiscoveredCandidate) (*registry.PluginRecord, error) {
	return s.manager.Install(ctx, candidate.Manifest, candidate.Directory)
}

// InstallManifest validates an explicit manifest and registers it.
func (s *Service) InstallManifest(ctx context.Context, manifest *model.Manifest, directory string) (*registry.PluginRecord, error) {
	return s.manager.Install(ctx, manifest, directory)
}

// Activate resolves dependencies and starts a registered plugin instance.
func (s *Service) Activate(ctx context.Context, id model.Identity) error {
	return s.manager.Activate(ctx, id)
}

// Suspend pauses runtime execution of an active plugin.
func (s *Service) Suspend(ctx context.Context, id model.Identity) error {
	return s.manager.Suspend(ctx, id)
}

// Restart re-initializes and restarts an existing plugin instance.
func (s *Service) Restart(ctx context.Context, id model.Identity) error {
	return s.manager.Restart(ctx, id)
}

// Update updates an installed plugin with a new manifest and reloads it if active.
func (s *Service) Update(ctx context.Context, id model.Identity, newManifest *model.Manifest) error {
	return s.manager.Update(ctx, id, newManifest)
}

// Unload releases runtime resources associated with a plugin, keeping its registration.
func (s *Service) Unload(ctx context.Context, id model.Identity) error {
	return s.manager.Unload(ctx, id)
}

// Remove unloads and removes a plugin permanently from the registry.
func (s *Service) Remove(ctx context.Context, id model.Identity) error {
	return s.manager.Remove(ctx, id)
}

// Get retrieves a defensive copy of a plugin record by identity.
func (s *Service) Get(id model.Identity) (*registry.PluginRecord, bool) {
	return s.registry.Get(id)
}

// List returns all registered plugins sorted deterministically by identity.
func (s *Service) List() []*registry.PluginRecord {
	return s.registry.List()
}

// ListByState returns plugins matching the specified lifecycle state.
func (s *Service) ListByState(state plugin.State) []*registry.PluginRecord {
	return s.registry.ListByState(state)
}

// RegisterInProcessFactory registers an in-process plugin constructor if using InProcessLoader.
func (s *Service) RegisterInProcessFactory(id model.Identity, factory loader.PluginFactory) {
	if ipl, ok := s.loader.(*loader.InProcessLoader); ok {
		ipl.RegisterFactory(id, factory)
	}
}

// Close gracefully stops and unloads all active plugins in reverse topological order.
func (s *Service) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	active := s.registry.ListByState(plugin.StateActive)
	resolvables := make([]dependency.Resolvable, len(active))
	for i, a := range active {
		resolvables[i] = a
	}

	plan, err := s.resolver.Resolve(resolvables)
	if err != nil {
		// Fallback to list order if resolution fails during shutdown
		for _, a := range active {
			_ = s.manager.Unload(ctx, a.Manifest.ID)
		}
		return nil
	}

	for _, id := range plan.DeactivationOrder {
		_ = s.manager.Unload(ctx, id)
	}

	return nil
}
