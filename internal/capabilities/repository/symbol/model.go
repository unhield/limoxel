package symbol

import (
	"fmt"
	"path/filepath"

	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
)

// SymbolModel represents the consolidated, immutable result of repository AST parsing and symbol extraction.
type SymbolModel struct {
	repositoryRoot string
	symbols        *SymbolDatabase
	docs           *DocumentationDatabase
	relationships  *SymbolRelationshipGraph
	diagnostics    []*discovery.Diagnostic
	fileHashes     map[string]string
}

// NewSymbolModel constructs an immutable SymbolModel.
func NewSymbolModel(
	repositoryRoot string,
	symbols *SymbolDatabase,
	docs *DocumentationDatabase,
	relationships *SymbolRelationshipGraph,
	diagnostics []*discovery.Diagnostic,
) *SymbolModel {
	return NewSymbolModelWithHashes(repositoryRoot, symbols, docs, relationships, diagnostics, nil)
}

// NewSymbolModelWithHashes constructs an immutable SymbolModel with file hashes.
func NewSymbolModelWithHashes(
	repositoryRoot string,
	symbols *SymbolDatabase,
	docs *DocumentationDatabase,
	relationships *SymbolRelationshipGraph,
	diagnostics []*discovery.Diagnostic,
	fileHashes map[string]string,
) *SymbolModel {
	diagList := make([]*discovery.Diagnostic, len(diagnostics))
	copy(diagList, diagnostics)

	hashes := make(map[string]string, len(fileHashes))
	for k, v := range fileHashes {
		hashes[filepath.ToSlash(filepath.Clean(k))] = v
	}

	return &SymbolModel{
		repositoryRoot: filepath.ToSlash(filepath.Clean(repositoryRoot)),
		symbols:        symbols,
		docs:           docs,
		relationships:  relationships,
		diagnostics:    diagList,
		fileHashes:     hashes,
	}
}

// RepositoryRoot returns the canonical repository root path.
func (sm *SymbolModel) RepositoryRoot() string {
	if sm == nil {
		return ""
	}
	return sm.repositoryRoot
}

// Symbols returns the queryable symbol database.
func (sm *SymbolModel) Symbols() *SymbolDatabase {
	if sm == nil {
		return nil
	}
	return sm.symbols
}

// Docs returns the queryable documentation database.
func (sm *SymbolModel) Docs() *DocumentationDatabase {
	if sm == nil {
		return nil
	}
	return sm.docs
}

// Relationships returns the symbol relationship graph.
func (sm *SymbolModel) Relationships() *SymbolRelationshipGraph {
	if sm == nil {
		return nil
	}
	return sm.relationships
}

// Diagnostics returns a defensive copy of diagnostics recorded during parsing.
func (sm *SymbolModel) Diagnostics() []*discovery.Diagnostic {
	if sm == nil || len(sm.diagnostics) == 0 {
		return nil
	}
	cloned := make([]*discovery.Diagnostic, len(sm.diagnostics))
	copy(cloned, sm.diagnostics)
	return cloned
}

// FileHash returns the recorded content hash for a given repository-relative file path.
func (sm *SymbolModel) FileHash(relPath string) string {
	if sm == nil || len(sm.fileHashes) == 0 {
		return ""
	}
	clean := filepath.ToSlash(filepath.Clean(relPath))
	return sm.fileHashes[clean]
}

// FileHashes returns a defensive copy of all file content hashes.
func (sm *SymbolModel) FileHashes() map[string]string {
	if sm == nil || len(sm.fileHashes) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(sm.fileHashes))
	for k, v := range sm.fileHashes {
		cloned[k] = v
	}
	return cloned
}

// String returns a human-readable summary of the SymbolModel.
func (sm *SymbolModel) String() string {
	if sm == nil {
		return ""
	}
	symCount := 0
	if sm.symbols != nil {
		symCount = sm.symbols.TotalCount()
	}
	docCount := 0
	if sm.docs != nil {
		docCount = sm.docs.TotalCount()
	}
	relCount := 0
	if sm.relationships != nil {
		relCount = sm.relationships.TotalCount()
	}

	return fmt.Sprintf("SymbolModel<root=%s, symbols=%d, docs=%d, relationships=%d>",
		sm.repositoryRoot, symCount, docCount, relCount)
}
