package xref

import (
	"fmt"
	"path/filepath"

	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
)

// XRefModel represents the consolidated, immutable result of repository cross-reference and call graph analysis.
type XRefModel struct {
	repositoryRoot string
	references     *ReferenceDatabase
	callGraph      *CallGraph
	navigation     *NavigationEngine
	impact         *ChangeImpactAnalyzer
	validation     *ValidationReport
	diagnostics    []*discovery.Diagnostic
	fileHashes     map[string]string
}

// NewXRefModel constructs an immutable XRefModel.
func NewXRefModel(
	repositoryRoot string,
	references *ReferenceDatabase,
	callGraph *CallGraph,
	navigation *NavigationEngine,
	impact *ChangeImpactAnalyzer,
	validation *ValidationReport,
	diagnostics []*discovery.Diagnostic,
) *XRefModel {
	return NewXRefModelWithHashes(repositoryRoot, references, callGraph, navigation, impact, validation, diagnostics, nil)
}

// NewXRefModelWithHashes constructs an immutable XRefModel with file content hashes.
func NewXRefModelWithHashes(
	repositoryRoot string,
	references *ReferenceDatabase,
	callGraph *CallGraph,
	navigation *NavigationEngine,
	impact *ChangeImpactAnalyzer,
	validation *ValidationReport,
	diagnostics []*discovery.Diagnostic,
	fileHashes map[string]string,
) *XRefModel {
	diagList := make([]*discovery.Diagnostic, len(diagnostics))
	copy(diagList, diagnostics)

	hashes := make(map[string]string, len(fileHashes))
	for k, v := range fileHashes {
		hashes[filepath.ToSlash(filepath.Clean(k))] = v
	}

	return &XRefModel{
		repositoryRoot: filepath.ToSlash(filepath.Clean(repositoryRoot)),
		references:     references,
		callGraph:      callGraph,
		navigation:     navigation,
		impact:         impact,
		validation:     validation,
		diagnostics:    diagList,
		fileHashes:     hashes,
	}
}

// RepositoryRoot returns the absolute repository root path.
func (xm *XRefModel) RepositoryRoot() string {
	if xm == nil {
		return ""
	}
	return xm.repositoryRoot
}

// References returns the queryable reference database.
func (xm *XRefModel) References() *ReferenceDatabase {
	if xm == nil {
		return nil
	}
	return xm.references
}

// CallGraph returns the repository call graph.
func (xm *XRefModel) CallGraph() *CallGraph {
	if xm == nil {
		return nil
	}
	return xm.callGraph
}

// Navigation returns the navigation engine.
func (xm *XRefModel) Navigation() *NavigationEngine {
	if xm == nil {
		return nil
	}
	return xm.navigation
}

// Impact returns the change impact analyzer.
func (xm *XRefModel) Impact() *ChangeImpactAnalyzer {
	if xm == nil {
		return nil
	}
	return xm.impact
}

// Validation returns the relationship validation report.
func (xm *XRefModel) Validation() *ValidationReport {
	if xm == nil {
		return nil
	}
	return xm.validation
}

// Diagnostics returns a defensive copy of diagnostics recorded during analysis.
func (xm *XRefModel) Diagnostics() []*discovery.Diagnostic {
	if xm == nil || len(xm.diagnostics) == 0 {
		return nil
	}
	cloned := make([]*discovery.Diagnostic, len(xm.diagnostics))
	copy(cloned, xm.diagnostics)
	return cloned
}

// FileHash returns the recorded content hash for a given repository-relative file path.
func (xm *XRefModel) FileHash(relPath string) string {
	if xm == nil || len(xm.fileHashes) == 0 {
		return ""
	}
	clean := filepath.ToSlash(filepath.Clean(relPath))
	return xm.fileHashes[clean]
}

// FileHashes returns a defensive copy of all file content hashes.
func (xm *XRefModel) FileHashes() map[string]string {
	if xm == nil || len(xm.fileHashes) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(xm.fileHashes))
	for k, v := range xm.fileHashes {
		cloned[k] = v
	}
	return cloned
}

// String returns a human-readable summary.
func (xm *XRefModel) String() string {
	if xm == nil {
		return ""
	}
	refCount := 0
	if xm.references != nil {
		refCount = xm.references.TotalCount()
	}
	edgeCount := 0
	if xm.callGraph != nil {
		edgeCount = xm.callGraph.TotalEdges()
	}
	issueCount := 0
	if xm.validation != nil {
		issueCount = xm.validation.TotalIssues()
	}

	return fmt.Sprintf("XRefModel<root=%s, references=%d, call_edges=%d, validation_issues=%d>",
		xm.repositoryRoot, refCount, edgeCount, issueCount)
}

// Summary returns a multi-line formatted summary of the model.
func (xm *XRefModel) Summary() string {
	if xm == nil {
		return ""
	}
	return fmt.Sprintf("Cross-Reference Model for %s\n  Total References: %d\n  Call Edges: %d\n  Validation Issues: %d",
		xm.repositoryRoot,
		xm.References().TotalCount(),
		xm.CallGraph().TotalEdges(),
		xm.Validation().TotalIssues(),
	)
}
