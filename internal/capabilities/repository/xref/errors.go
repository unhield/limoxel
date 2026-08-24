package xref

import "errors"

var (
	// ErrNilEngine is returned when an operation is invoked on a nil Engine receiver.
	ErrNilEngine = errors.New("xref: engine is nil")

	// ErrNilDiscoverer is returned when constructing an Engine with a nil discoverer.
	ErrNilDiscoverer = errors.New("xref: discoverer is nil")

	// ErrNilDiscoveryResult is returned when analyzing a nil discovery result.
	ErrNilDiscoveryResult = errors.New("xref: discovery result is nil")

	// ErrNilSymbolModel is returned when analyzing a nil symbol model.
	ErrNilSymbolModel = errors.New("xref: symbol model is nil")

	// ErrNilDependencyModel is returned when analyzing with a nil dependency model.
	ErrNilDependencyModel = errors.New("xref: dependency model is nil")

	// ErrNilDependencyResult is kept as an alias for backwards compatibility.
	ErrNilDependencyResult = ErrNilDependencyModel

	// ErrNilRepository is returned when analyzing a nil domain repository.
	ErrNilRepository = errors.New("xref: repository is nil")

	// ErrPathEmpty is returned when analyzing an empty filesystem path.
	ErrPathEmpty = errors.New("xref: repository path cannot be empty")

	// ErrPathNotFound is returned when an analyzed path does not exist on disk.
	ErrPathNotFound = errors.New("xref: repository path not found")

	// ErrNotDirectory is returned when an analyzed path is not a directory.
	ErrNotDirectory = errors.New("xref: repository path is not a directory")

	// ErrSymbolNotFound is returned when a query target symbol does not exist.
	ErrSymbolNotFound = errors.New("xref: symbol not found")

	// ErrAmbiguousDefinition is returned when go-to-definition encounters multiple unresolved candidates.
	ErrAmbiguousDefinition = errors.New("xref: multiple ambiguous definition candidates found")

	// ErrAnalysisFailed is returned when an analysis pipeline step encounters an unrecoverable failure.
	ErrAnalysisFailed = errors.New("xref: cross-reference analysis failed")
)
