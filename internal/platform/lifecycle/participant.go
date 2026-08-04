package lifecycle

import "context"

// Participant defines the contract required for a component to participate in the platform lifecycle.
type Participant interface {
	// Name returns the unique identifier for the lifecycle participant.
	Name() string

	// Initialize prepares component resources prior to platform execution.
	Initialize(ctx context.Context) error

	// Prepare validates readiness and wires component dependencies before startup.
	Prepare(ctx context.Context) error

	// Start initiates active operation of the component.
	Start(ctx context.Context) error

	// Stop gracefully terminates component operations and releases resources.
	Stop(ctx context.Context) error
}

// BaseParticipant provides a default no-op implementation of the Participant interface.
// Structs can embed BaseParticipant to implement only relevant lifecycle methods.
type BaseParticipant struct {
	ParticipantName string
}

// Name returns the participant name.
func (b *BaseParticipant) Name() string {
	return b.ParticipantName
}

// Initialize performs default no-op initialization.
func (b *BaseParticipant) Initialize(ctx context.Context) error {
	return nil
}

// Prepare performs default no-op preparation.
func (b *BaseParticipant) Prepare(ctx context.Context) error {
	return nil
}

// Start performs default no-op startup.
func (b *BaseParticipant) Start(ctx context.Context) error {
	return nil
}

// Stop performs default no-op shutdown.
func (b *BaseParticipant) Stop(ctx context.Context) error {
	return nil
}
