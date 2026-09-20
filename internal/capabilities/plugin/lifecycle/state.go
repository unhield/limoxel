package lifecycle

import (
	"fmt"

	pkgerr "github.com/unhield/limoxel/internal/capabilities/plugin/errors"
	"github.com/unhield/limoxel/plugin"
)

// allowedTransitions maps source states to permitted target states.
var allowedTransitions = map[plugin.State]map[plugin.State]bool{
	plugin.StateDiscovered: {
		plugin.StateValidated: true,
		plugin.StateInvalid:   true,
		plugin.StateFailed:    true,
	},
	plugin.StateValidated: {
		plugin.StateInstalled: true,
		plugin.StateResolved:  true,
		plugin.StateFailed:    true,
	},
	plugin.StateInstalled: {
		plugin.StateRegistered: true,
		plugin.StateResolved:   true,
		plugin.StateFailed:     true,
	},
	plugin.StateRegistered: {
		plugin.StateResolved: true,
		plugin.StateActive:   true,
		plugin.StateRemoving: true,
		plugin.StateFailed:   true,
	},
	plugin.StateResolved: {
		plugin.StateInstalled:  true,
		plugin.StateRegistered: true,
		plugin.StateLoading:    true,
		plugin.StateActive:     true,
		plugin.StateFailed:     true,
	},
	plugin.StateLoading: {
		plugin.StateActive: true,
		plugin.StateFailed: true,
	},
	plugin.StateActive: {
		plugin.StateSuspended: true,
		plugin.StateUpdating:  true,
		plugin.StateUnloading: true,
		plugin.StateUnloaded:  true,
		plugin.StateFailed:    true,
	},
	plugin.StateSuspended: {
		plugin.StateActive:    true,
		plugin.StateUnloading: true,
		plugin.StateUnloaded:  true,
		plugin.StateFailed:    true,
	},
	plugin.StateUpdating: {
		plugin.StateActive: true,
		plugin.StateFailed: true,
	},
	plugin.StateUnloading: {
		plugin.StateUnloaded:   true,
		plugin.StateRegistered: true,
		plugin.StateRemoving:   true,
		plugin.StateFailed:     true,
	},
	plugin.StateUnloaded: {
		plugin.StateRegistered: true,
		plugin.StateRemoving:   true,
		plugin.StateFailed:     true,
	},
	plugin.StateRemoving: {
		plugin.StateRemoved: true,
		plugin.StateFailed:  true,
	},
	plugin.StateFailed: {
		plugin.StateRegistered: true, // Allow retry/recovery from failed state
		plugin.StateRemoving:   true,
		plugin.StateRemoved:    true,
	},
	plugin.StateInvalid: {
		plugin.StateRemoving: true,
		plugin.StateRemoved:  true,
	},
	plugin.StateRemoved: {}, // Terminal state, no outbound transitions permitted
}

// CanTransition returns true if the transition from source to target is permitted.
func CanTransition(from, to plugin.State) bool {
	if from == to {
		return true // Idempotent same-state transitions permitted
	}
	targets, ok := allowedTransitions[from]
	if !ok {
		return false
	}
	return targets[to]
}

// ValidateTransition verifies whether a lifecycle transition is allowed, returning a structured error if rejected.
func ValidateTransition(from, to plugin.State) error {
	if CanTransition(from, to) {
		return nil
	}
	return pkgerr.New(pkgerr.CodeInvalidTransition, "",
		fmt.Sprintf("invalid lifecycle transition from %s to %s", from, to))
}
