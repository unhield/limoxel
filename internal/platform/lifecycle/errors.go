package lifecycle

import "errors"

var (
	// ErrManagerNil indicates an operation was invoked on a nil Manager instance.
	ErrManagerNil = errors.New("lifecycle: manager instance is nil")

	// ErrParticipantNil indicates an attempt to register a nil Participant.
	ErrParticipantNil = errors.New("lifecycle: participant is nil")

	// ErrParticipantDuplicate indicates an attempt to register a participant with a duplicate name.
	ErrParticipantDuplicate = errors.New("lifecycle: participant name already registered")

	// ErrInvalidTransition indicates an illegal lifecycle stage transition attempt.
	ErrInvalidTransition = errors.New("lifecycle: invalid stage transition")

	// ErrNilContext indicates a nil context was passed to a lifecycle transition.
	ErrNilContext = errors.New("lifecycle: context is nil")

	// ErrExecutionFailed indicates a lifecycle stage execution failed on a participant.
	ErrExecutionFailed = errors.New("lifecycle: participant execution failed")

	// ErrAlreadyTerminated indicates an operation was attempted on a terminated lifecycle manager.
	ErrAlreadyTerminated = errors.New("lifecycle: manager is already terminated")
)
