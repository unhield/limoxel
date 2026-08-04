package extension

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

var (
	// ErrNilDiscoverer indicates an operation was attempted on a nil Discoverer instance.
	ErrNilDiscoverer = errors.New("extension: discoverer instance is nil")

	// ErrNoRootsProvided indicates no root paths were provided for discovery.
	ErrNoRootsProvided = errors.New("extension: no root paths provided for discovery")
)

// DiscoveryResult represents an immutable collection of discovered extension descriptors.
type DiscoveryResult struct {
	roots       []string
	descriptors []*Descriptor
}

// NewDiscoveryResult constructs a new immutable DiscoveryResult.
func NewDiscoveryResult(roots []string, descriptors []*Descriptor) *DiscoveryResult {
	clonedRoots := make([]string, len(roots))
	copy(clonedRoots, roots)

	clonedDescs := make([]*Descriptor, len(descriptors))
	copy(clonedDescs, descriptors)

	return &DiscoveryResult{
		roots:       clonedRoots,
		descriptors: clonedDescs,
	}
}

// Roots returns a defensive copy of the root paths used for discovery.
func (r *DiscoveryResult) Roots() []string {
	if r == nil || len(r.roots) == 0 {
		return nil
	}
	cloned := make([]string, len(r.roots))
	copy(cloned, r.roots)
	return cloned
}

// Descriptors returns a defensive copy of discovered Descriptor objects in deterministic order.
func (r *DiscoveryResult) Descriptors() []*Descriptor {
	if r == nil || len(r.descriptors) == 0 {
		return nil
	}
	cloned := make([]*Descriptor, len(r.descriptors))
	copy(cloned, r.descriptors)
	return cloned
}

// Count returns the total number of discovered extension descriptors.
func (r *DiscoveryResult) Count() int {
	if r == nil {
		return 0
	}
	return len(r.descriptors)
}

// Discoverer coordinates deterministic location and collection of extension descriptors.
type Discoverer struct {
	roots []string
}

// NewDiscoverer constructs and validates a new immutable Discoverer with root paths.
func NewDiscoverer(roots ...string) (*Discoverer, error) {
	if len(roots) == 0 {
		return nil, ErrNoRootsProvided
	}

	cleanedRoots := make([]string, 0, len(roots))
	seen := make(map[string]struct{})

	for _, root := range roots {
		clean := filepath.Clean(strings.TrimSpace(root))
		if clean == "" {
			continue
		}
		if _, exists := seen[clean]; !exists {
			seen[clean] = struct{}{}
			cleanedRoots = append(cleanedRoots, clean)
		}
	}

	if len(cleanedRoots) == 0 {
		return nil, ErrNoRootsProvided
	}

	return &Discoverer{roots: cleanedRoots}, nil
}

// Roots returns a defensive copy of configured discovery root paths.
func (d *Discoverer) Roots() []string {
	if d == nil || len(d.roots) == 0 {
		return nil
	}
	cloned := make([]string, len(d.roots))
	copy(cloned, d.roots)
	return cloned
}

// Discover collects extension descriptors from configured roots in deterministic order.
// Duplicate discoveries (by descriptor ID) are ignored.
func (d *Discoverer) Discover() (*DiscoveryResult, error) {
	if d == nil || len(d.roots) == 0 {
		return nil, ErrNilDiscoverer
	}

	descMap := make(map[string]*Descriptor)
	var discovered []*Descriptor

	for _, root := range d.roots {
		baseName := filepath.Base(root)
		cleanID := strings.ToLower(strings.TrimSpace(baseName))
		if cleanID == "" || cleanID == "." || cleanID == ".." {
			continue
		}

		if _, exists := descMap[cleanID]; exists {
			continue
		}

		desc, err := NewDescriptor(cleanID, baseName, "1.0.0", "Limoxel Discovery", fmt.Sprintf("Extension discovered at %s", root), nil)
		if err != nil {
			continue
		}

		descMap[cleanID] = desc
		discovered = append(discovered, desc)
	}

	sort.Slice(discovered, func(i, j int) bool {
		return discovered[i].ID() < discovered[j].ID()
	})

	return NewDiscoveryResult(d.roots, discovered), nil
}
