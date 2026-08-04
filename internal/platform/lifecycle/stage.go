package lifecycle

import (
	"fmt"
)

// Stage represents a canonical stage in the Limoxel platform lifecycle.
type Stage uint8

const (
	// StageCreated indicates the lifecycle manager has been created.
	StageCreated Stage = iota

	// StageInitializing indicates participants are undergoing initialization.
	StageInitializing

	// StageInitialized indicates all participants have completed initialization.
	StageInitialized

	// StagePreparing indicates participants are preparing for execution readiness.
	StagePreparing

	// StagePrepared indicates all participants are prepared for startup.
	StagePrepared

	// StageStarting indicates participants are starting operational execution.
	StageStarting

	// StageRunning indicates all participants are running normally.
	StageRunning

	// StageStopping indicates participants are undergoing orderly shutdown.
	StageStopping

	// StageStopped indicates all participants have completed shutdown.
	StageStopped

	// StageTerminated indicates the lifecycle manager has terminated permanently.
	StageTerminated
)

// String returns the human-readable string representation of the Stage.
func (s Stage) String() string {
	switch s {
	case StageCreated:
		return "CREATED"
	case StageInitializing:
		return "INITIALIZING"
	case StageInitialized:
		return "INITIALIZED"
	case StagePreparing:
		return "PREPARING"
	case StagePrepared:
		return "PREPARED"
	case StageStarting:
		return "STARTING"
	case StageRunning:
		return "RUNNING"
	case StageStopping:
		return "STOPPING"
	case StageStopped:
		return "STOPPED"
	case StageTerminated:
		return "TERMINATED"
	default:
		return fmt.Sprintf("UNKNOWN_STAGE(%d)", uint8(s))
	}
}

// CanTransitionTo reports whether transitioning from current to target stage is legal.
func (s Stage) CanTransitionTo(target Stage) bool {
	if s == target {
		return false
	}
	if s == StageTerminated {
		return false
	}

	switch s {
	case StageCreated:
		return target == StageInitializing || target == StageStopping || target == StageTerminated
	case StageInitializing:
		return target == StageInitialized || target == StageStopping || target == StageTerminated
	case StageInitialized:
		return target == StagePreparing || target == StageStopping || target == StageTerminated
	case StagePreparing:
		return target == StagePrepared || target == StageStopping || target == StageTerminated
	case StagePrepared:
		return target == StageStarting || target == StageStopping || target == StageTerminated
	case StageStarting:
		return target == StageRunning || target == StageStopping || target == StageTerminated
	case StageRunning:
		return target == StageStopping
	case StageStopping:
		return target == StageStopped || target == StageTerminated
	case StageStopped:
		return target == StageTerminated
	default:
		return false
	}
}
