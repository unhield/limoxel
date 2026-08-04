package parser

import "errors"

var (
	// ErrNilRegistry indicates an operation was attempted on a nil Registry instance.
	ErrNilRegistry = errors.New("parser: registry instance is nil")

	// ErrNilParser indicates an attempt to register a nil Descriptor instance.
	ErrNilParser = errors.New("parser: descriptor instance is nil")

	// ErrInvalidID indicates a parser ID string is empty or invalid.
	ErrInvalidID = errors.New("parser: parser ID is invalid or empty")

	// ErrInvalidName indicates a parser name string is empty or invalid.
	ErrInvalidName = errors.New("parser: parser name is invalid or empty")

	// ErrInvalidLanguageID indicates a target language ID string is empty or invalid.
	ErrInvalidLanguageID = errors.New("parser: language ID is invalid or empty")

	// ErrDuplicateParser indicates a parser with the same ID is already registered.
	ErrDuplicateParser = errors.New("parser: parser is already registered")

	// ErrParserNotFound indicates the requested parser was not found in the registry.
	ErrParserNotFound = errors.New("parser: parser not found")

	// ErrAlreadyActive indicates an attempt to activate a parser that is already active.
	ErrAlreadyActive = errors.New("parser: parser is already active")

	// ErrAlreadyInactive indicates an attempt to deactivate a parser that is already inactive.
	ErrAlreadyInactive = errors.New("parser: parser is already inactive")

	// ErrNilPipeline indicates an operation was attempted on a nil Pipeline instance.
	ErrNilPipeline = errors.New("parser: pipeline instance is nil")

	// ErrInvalidPipelineSequence indicates a pipeline stage sequence is empty or out of canonical order.
	ErrInvalidPipelineSequence = errors.New("parser: invalid pipeline stage sequence")

	// ErrNilResult indicates an operation was attempted on a nil Result instance.
	ErrNilResult = errors.New("parser: result instance is nil")

	// ErrInvalidStatus indicates a result status is invalid or unknown.
	ErrInvalidStatus = errors.New("parser: invalid result status")
)
