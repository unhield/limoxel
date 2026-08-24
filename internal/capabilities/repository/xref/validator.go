package xref

import (
	"fmt"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/repository/dependency"
	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
)

// ValidationEngine validates cross-reference integrity across symbols, references, and call graphs.
type ValidationEngine struct {
	symModel  *symbol.SymbolModel
	refDB     *ReferenceDatabase
	callGraph *CallGraph
	depModel  *dependency.DependencyModel
}

// NewValidationEngine constructs a ValidationEngine.
func NewValidationEngine(
	symModel *symbol.SymbolModel,
	refDB *ReferenceDatabase,
	callGraph *CallGraph,
	depModel *dependency.DependencyModel,
) *ValidationEngine {
	return &ValidationEngine{
		symModel:  symModel,
		refDB:     refDB,
		callGraph: callGraph,
		depModel:  depModel,
	}
}

// Validate executes relationship integrity checks and returns an immutable ValidationReport.
func (ve *ValidationEngine) Validate() *ValidationReport {
	var issues []*ValidationIssue

	if ve == nil || ve.symModel == nil || ve.refDB == nil {
		return NewValidationReport(nil)
	}

	symDB := ve.symModel.Symbols()

	// 1. Broken References & Missing Symbols Check
	for _, ref := range ve.refDB.AllReferences() {
		if ref.State() == StateBroken {
			issues = append(issues, NewValidationIssue(
				ValidationBrokenRef,
				fmt.Sprintf("broken reference from %s to missing internal symbol %s", ref.SourceSymbolID(), ref.TargetSymbolID()),
				ref.SourceSymbolID(),
				ref.TargetSymbolID(),
				ref.FilePath(),
				ref.Position(),
			))
			continue
		}

		if ref.State() == StateResolved {
			targetSym := symDB.SymbolByID(ref.TargetSymbolID())
			if targetSym == nil {
				issues = append(issues, NewValidationIssue(
					ValidationMissingSymbol,
					fmt.Sprintf("resolved reference target %s not found in symbol database", ref.TargetSymbolID()),
					ref.SourceSymbolID(),
					ref.TargetSymbolID(),
					ref.FilePath(),
					ref.Position(),
				))
			}
		}
	}

	// 2. Duplicate Symbols in same package check
	seenSymIDs := make(map[string]*symbol.Symbol)
	for _, sym := range symDB.AllSymbols() {
		if existing, exists := seenSymIDs[sym.ID()]; exists {
			// Check if same file/line (exact same instance) or true collision
			if existing.FilePath() != sym.FilePath() || (existing.Position() != nil && sym.Position() != nil && existing.Position().Line() != sym.Position().Line()) {
				issues = append(issues, NewValidationIssue(
					ValidationDuplicateSymbol,
					fmt.Sprintf("duplicate symbol declaration for identity %s in files %s and %s", sym.ID(), existing.FilePath(), sym.FilePath()),
					sym.ID(),
					sym.ID(),
					sym.FilePath(),
					sym.Position(),
				))
			}
		} else {
			seenSymIDs[sym.ID()] = sym
		}
	}

	// 3. Circular Recursion / Cycle Reporting
	if ve.callGraph != nil {
		for _, cycle := range ve.callGraph.RecursiveCycles() {
			if len(cycle) > 1 {
				cyclePath := strings.Join(cycle, " -> ")
				issues = append(issues, NewValidationIssue(
					ValidationCircularRef,
					fmt.Sprintf("recursive call cycle detected: %s", cyclePath),
					cycle[0],
					cycle[len(cycle)-1],
					"",
					nil,
				))
			}
		}
	}

	return NewValidationReport(issues)
}
