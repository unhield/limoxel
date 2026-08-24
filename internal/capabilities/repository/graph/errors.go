package graph

import "errors"

var (
	// ErrNilEngine is returned when an operation is performed on a nil Engine.
	ErrNilEngine = errors.New("graph: engine is nil")

	// ErrNilDiscoverer is returned when constructing an engine with a nil discoverer.
	ErrNilDiscoverer = errors.New("graph: discoverer is nil")

	// ErrNilDiscoveryResult is returned when constructing a graph with a nil discovery result.
	ErrNilDiscoveryResult = errors.New("graph: discovery result is nil")

	// ErrNilMetadataProfile is returned when constructing a graph with a nil metadata profile.
	ErrNilMetadataProfile = errors.New("graph: metadata profile is nil")

	// ErrNilStructureModel is returned when constructing a graph with a nil structure model.
	ErrNilStructureModel = errors.New("graph: structure model is nil")

	// ErrNilDependencyModel is returned when constructing a graph with a nil dependency model.
	ErrNilDependencyModel = errors.New("graph: dependency model is nil")

	// ErrNilIndexModel is returned when constructing a graph with a nil index model.
	ErrNilIndexModel = errors.New("graph: index model is nil")

	// ErrNilSymbolModel is returned when constructing a graph with a nil symbol model.
	ErrNilSymbolModel = errors.New("graph: symbol model is nil")

	// ErrNilXRefModel is returned when constructing a graph with a nil xref model.
	ErrNilXRefModel = errors.New("graph: xref model is nil")

	// ErrNilRepository is returned when analyzing a nil domain repository.
	ErrNilRepository = errors.New("graph: repository is nil")

	// ErrPathEmpty is returned when an analyzed path is empty or whitespace.
	ErrPathEmpty = errors.New("graph: repository path is empty")

	// ErrNodeNotFound is returned when a requested node ID does not exist in the graph.
	ErrNodeNotFound = errors.New("graph: node not found")

	// ErrRelationshipNotFound is returned when a requested relationship ID does not exist.
	ErrRelationshipNotFound = errors.New("graph: relationship not found")

	// ErrInvalidNode is returned when a node violates identity or metadata constraints.
	ErrInvalidNode = errors.New("graph: invalid node")

	// ErrInvalidRelationship is returned when a relationship violates source/target constraints.
	ErrInvalidRelationship = errors.New("graph: invalid relationship")

	// ErrExportFailed is returned when graph export fails.
	ErrExportFailed = errors.New("graph: export failed")

	// ErrValidationFailed is returned when graph validation detects integrity violations.
	ErrValidationFailed = errors.New("graph: validation failed")

	// ErrMaxDepthExceeded is returned when traversal exceeds allowed depth limit.
	ErrMaxDepthExceeded = errors.New("graph: maximum traversal depth exceeded")
)
