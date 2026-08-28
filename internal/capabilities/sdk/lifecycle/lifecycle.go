package lifecycle

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	sdkerr "github.com/unhield/limoxel/internal/capabilities/sdk/errors"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

var (
	// ErrInvalidLifecycleState indicates an unsupported or malformed lifecycle state.
	ErrInvalidLifecycleState = errors.New("lifecycle: invalid lifecycle state")
)

// LifecycleState defines the operational phase of an SDK public API contract or capability.
type LifecycleState string

const (
	// StateIntroduced represents an experimental or newly introduced public API undergoing initial stabilization.
	StateIntroduced LifecycleState = "INTRODUCED"

	// StateSupported represents a stable, fully supported, production-ready public API with backward-compatibility guarantees.
	StateSupported LifecycleState = "SUPPORTED"

	// StateDeprecated represents an API that remains functional and supported but is scheduled for future retirement.
	StateDeprecated LifecycleState = "DEPRECATED"

	// StateRemoved represents a retired API that has been removed from active support.
	StateRemoved LifecycleState = "REMOVED"
)

// String returns the string representation of LifecycleState.
func (s LifecycleState) String() string {
	return string(s)
}

// IsActive returns true if the API is currently callable (Introduced, Supported, or Deprecated).
func (s LifecycleState) IsActive() bool {
	return s == StateIntroduced || s == StateSupported || s == StateDeprecated
}

// CapabilityKind identifies the high-level capability domain owning the API.
type CapabilityKind string

const (
	// CapabilityRepository represents repository, workspace, and metadata management.
	CapabilityRepository CapabilityKind = "repository"

	// CapabilitySymbol represents symbol indexing, lookup, and cross-referencing.
	CapabilitySymbol CapabilityKind = "symbol"

	// CapabilityGraph represents knowledge graph modeling, traversal, and export.
	CapabilityGraph CapabilityKind = "graph"

	// CapabilitySearch represents repository-wide multi-domain search.
	CapabilitySearch CapabilityKind = "search"

	// CapabilityIntelligence represents analysis, reasoning, navigation, and events.
	CapabilityIntelligence CapabilityKind = "intelligence"
)

// String returns the string representation of CapabilityKind.
func (c CapabilityKind) String() string {
	return string(c)
}

// DeprecationInfo holds structured metadata for deprecated public APIs to provide migration guidance to consumers.
type DeprecationInfo struct {
	Since             version.SemVer
	PlannedRemoval    version.SemVer
	Replacement       string
	MigrationGuidance string
	Reason            string
}

// String formats deprecation guidance into a readable notice.
func (d *DeprecationInfo) String() string {
	if d == nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Deprecated since v%s.", d.Since))
	if d.PlannedRemoval.Major > 0 || d.PlannedRemoval.Minor > 0 {
		sb.WriteString(fmt.Sprintf(" Planned removal in v%s.", d.PlannedRemoval))
	}
	if d.Replacement != "" {
		sb.WriteString(fmt.Sprintf(" Use %q instead.", d.Replacement))
	}
	if d.MigrationGuidance != "" {
		sb.WriteString(" ")
		sb.WriteString(d.MigrationGuidance)
	}
	return sb.String()
}

// APIDescriptor encapsulates identity, capability domain, lifecycle state, and documentation for a public API contract.
type APIDescriptor struct {
	Name          string
	Capability    CapabilityKind
	Since         version.SemVer
	Lifecycle     LifecycleState
	Deprecation   *DeprecationInfo
	Documentation string
}

// Validate verifies that the APIDescriptor adheres to SDK lifecycle rules.
func (d APIDescriptor) Validate() error {
	if strings.TrimSpace(d.Name) == "" {
		return sdkerr.NewInvalidInput("API descriptor name cannot be empty")
	}
	if strings.TrimSpace(string(d.Capability)) == "" {
		return sdkerr.NewInvalidInput(fmt.Sprintf("API %q capability cannot be empty", d.Name))
	}

	switch d.Lifecycle {
	case StateIntroduced, StateSupported:
		return nil
	case StateDeprecated:
		if d.Deprecation == nil {
			return sdkerr.NewInvalidState(string(d.Lifecycle), fmt.Sprintf("API %q is marked DEPRECATED but missing DeprecationInfo", d.Name))
		}
		if d.Deprecation.Replacement == "" && d.Deprecation.MigrationGuidance == "" {
			return sdkerr.NewInvalidState(string(d.Lifecycle), fmt.Sprintf("API %q deprecation must provide either Replacement or MigrationGuidance", d.Name))
		}
		return nil
	case StateRemoved:
		return nil
	default:
		return sdkerr.NewInvalidState(string(d.Lifecycle), fmt.Sprintf("API %q has unknown lifecycle state %q", d.Name, d.Lifecycle))
	}
}

// Registry maintains a thread-safe registry of declared public API descriptors and verifies lifecycle transitions.
type Registry struct {
	mu   sync.RWMutex
	apis map[string]APIDescriptor
}

// NewRegistry constructs an initialized Lifecycle Registry.
func NewRegistry() *Registry {
	return &Registry{
		apis: make(map[string]APIDescriptor),
	}
}

// Register registers an APIDescriptor after validating its lifecycle contract.
func (r *Registry) Register(desc APIDescriptor) error {
	if err := desc.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.apis[desc.Name] = desc
	return nil
}

// Lookup retrieves an APIDescriptor by name.
func (r *Registry) Lookup(name string) (APIDescriptor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	desc, ok := r.apis[name]
	return desc, ok
}

// All returns a slice of all registered APIDescriptors.
func (r *Registry) All() []APIDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	res := make([]APIDescriptor, 0, len(r.apis))
	for _, d := range r.apis {
		res = append(res, d)
	}
	return res
}

// ValidateTransition verifies whether moving an API from oldState to newState complies with SDK lifecycle rules.
func ValidateTransition(oldState, newState LifecycleState) error {
	switch oldState {
	case StateIntroduced:
		if newState == StateSupported || newState == StateDeprecated || newState == StateRemoved {
			return nil
		}
	case StateSupported:
		if newState == StateDeprecated || newState == StateRemoved {
			return nil
		}
	case StateDeprecated:
		if newState == StateRemoved || newState == StateSupported {
			return nil
		}
	case StateRemoved:
		// Removed APIs are terminal
		return sdkerr.NewLifecycleViolation("API", string(oldState), "cannot transition out of REMOVED state")
	}

	if oldState == newState {
		return nil
	}

	return sdkerr.NewLifecycleViolation("API", string(newState), fmt.Sprintf("illegal lifecycle transition from %s to %s", oldState, newState))
}
