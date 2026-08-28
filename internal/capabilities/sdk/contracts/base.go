package contracts

import (
	"strings"

	sdkerr "github.com/unhield/limoxel/internal/capabilities/sdk/errors"
	"github.com/unhield/limoxel/internal/capabilities/sdk/lifecycle"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

// Contract is the base interface that every public SDK contract must satisfy.
type Contract interface {
	Name() string
	Capability() lifecycle.CapabilityKind
	Since() version.SemVer
	Lifecycle() lifecycle.LifecycleState
	Validate() error
}

// BaseContract provides default embedded metadata management for SDK contracts.
type BaseContract struct {
	Descriptor lifecycle.APIDescriptor
}

// NewBaseContract constructs an initialized BaseContract.
func NewBaseContract(name string, capKind lifecycle.CapabilityKind, since version.SemVer, state lifecycle.LifecycleState, doc string) BaseContract {
	return BaseContract{
		Descriptor: lifecycle.APIDescriptor{
			Name:          strings.TrimSpace(name),
			Capability:    capKind,
			Since:         since,
			Lifecycle:     state,
			Documentation: strings.TrimSpace(doc),
		},
	}
}

// Name returns the contract's public name.
func (b BaseContract) Name() string {
	return b.Descriptor.Name
}

// Capability returns the contract's capability domain.
func (b BaseContract) Capability() lifecycle.CapabilityKind {
	return b.Descriptor.Capability
}

// Since returns the version in which the contract was introduced.
func (b BaseContract) Since() version.SemVer {
	return b.Descriptor.Since
}

// Lifecycle returns the active lifecycle state.
func (b BaseContract) Lifecycle() lifecycle.LifecycleState {
	return b.Descriptor.Lifecycle
}

// Validate verifies that the contract metadata adheres to SDK rules.
func (b BaseContract) Validate() error {
	return b.Descriptor.Validate()
}

// PaginationOptions provides standard pagination controls for collection-returning SDK operations.
type PaginationOptions struct {
	Limit  int
	Offset int
}

// Normalize ensures PaginationOptions bounds are valid and sensible.
func (p PaginationOptions) Normalize(defaultLimit, maxLimit int) PaginationOptions {
	limit := p.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if maxLimit > 0 && limit > maxLimit {
		limit = maxLimit
	}
	offset := p.Offset
	if offset < 0 {
		offset = 0
	}
	return PaginationOptions{
		Limit:  limit,
		Offset: offset,
	}
}

// ContextOptions encapsulates standard execution flags for SDK operations.
type ContextOptions struct {
	TimeoutSeconds int
	Strict         bool
}

// ValidateContract validates any implementation of the Contract interface.
func ValidateContract(c Contract) error {
	if c == nil {
		return sdkerr.NewInvalidInput("contract instance cannot be nil")
	}
	return c.Validate()
}
