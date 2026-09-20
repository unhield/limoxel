package plugin

import (
	"context"

	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

// State represents the operational lifecycle state of a managed plugin.
type State string

const (
	// StateDiscovered indicates the plugin has been discovered on the filesystem.
	StateDiscovered State = "DISCOVERED"

	// StateValidated indicates the plugin manifest and constraints have passed validation.
	StateValidated State = "VALIDATED"

	// StateInstalled indicates the plugin is installed and ready for registration.
	StateInstalled State = "INSTALLED"

	// StateRegistered indicates the plugin is registered with the host registry.
	StateRegistered State = "REGISTERED"

	// StateActive indicates the plugin is active and operational.
	StateActive State = "ACTIVE"

	// StateSuspended indicates the plugin execution is temporarily paused.
	StateSuspended State = "SUSPENDED"

	// StateFailed indicates the plugin encountered an unrecoverable runtime error.
	StateFailed State = "FAILED"

	// StateUpdating indicates the plugin is undergoing a version or configuration update.
	StateUpdating State = "UPDATING"

	// StateRemoving indicates the plugin is being decommissioned.
	StateRemoving State = "REMOVING"

	// StateRemoved indicates the plugin has been removed from the registry.
	StateRemoved State = "REMOVED"

	// StateUnloaded indicates the plugin runtime instance has been unloaded from memory.
	StateUnloaded State = "UNLOADED"

	// StateResolved indicates the plugin dependencies and requirements are satisfied.
	StateResolved State = "RESOLVED"

	// StateLoading indicates the plugin is in the process of runtime initialization.
	StateLoading State = "LOADING"

	// StateUnloading indicates the plugin is in the process of deactivation and resource disposal.
	StateUnloading State = "UNLOADING"

	// StateInvalid indicates the plugin manifest or constraints failed validation.
	StateInvalid State = "INVALID"
)

// String returns the string representation of the lifecycle state.
func (s State) String() string {
	return string(s)
}

// Capability represents a specific functional capability offered by a plugin.
type Capability struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}

// RequiredCapability specifies a capability that must be provided by the host or another plugin.
type RequiredCapability struct {
	Name       string `json:"name"`
	MinVersion string `json:"min_version,omitempty"`
	MaxVersion string `json:"max_version,omitempty"`
	Optional   bool   `json:"optional,omitempty"`
}

// Dependency specifies an inter-plugin dependency requirement.
type Dependency struct {
	ID         string `json:"id"`
	Constraint string `json:"constraint,omitempty"`
	Optional   bool   `json:"optional,omitempty"`
}

// CompatibilityRequirement describes host and environment compatibility constraints.
type CompatibilityRequirement struct {
	MinHostVersion     string   `json:"min_host_version,omitempty"`
	MaxHostVersion     string   `json:"max_host_version,omitempty"`
	SupportedPlatforms []string `json:"supported_platforms,omitempty"`
}

// Manifest provides the declarative metadata and requirements of a plugin.
type Manifest struct {
	SchemaVersion        string                   `json:"schema_version"`
	ID                   string                   `json:"id"`
	Name                 string                   `json:"name"`
	Version              string                   `json:"version"`
	Publisher            string                   `json:"publisher,omitempty"`
	Description          string                   `json:"description,omitempty"`
	Entrypoint           string                   `json:"entrypoint,omitempty"`
	Capabilities         []Capability             `json:"capabilities,omitempty"`
	RequiredCapabilities []RequiredCapability     `json:"required_capabilities,omitempty"`
	Dependencies         []Dependency             `json:"dependencies,omitempty"`
	Compatibility        CompatibilityRequirement `json:"compatibility,omitempty"`
	Metadata             map[string]string        `json:"metadata,omitempty"`
}

// Plugin is the foundational interface that all Limoxel plugins must implement.
type Plugin interface {
	// ID returns the unique canonical identifier of the plugin.
	ID() string

	// Manifest returns the declarative specification of the plugin.
	Manifest() Manifest

	// Init initializes the plugin with the host environment before activation.
	Init(ctx context.Context, host Host) error

	// Start activates the plugin runtime execution.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the plugin and releases runtime resources.
	Stop(ctx context.Context) error
}

// Host provides plugins with controlled access to host services and platform capabilities.
type Host interface {
	// HostVersion returns the canonical semantic version of the Limoxel host.
	HostVersion() version.SemVer

	// Capabilities returns the list of all available capability identifiers.
	Capabilities() []string

	// HasCapability checks whether a specific capability is available on the host.
	HasCapability(name string) bool

	// Repository returns the repository service facade.
	Repository() RepositoryFacade

	// Events returns the event publishing and subscription facade.
	Events() EventFacade

	// Logger returns a contextual structured logger for the plugin.
	Logger() Logger
}
