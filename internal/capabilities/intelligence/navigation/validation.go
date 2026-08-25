package navigation

import (
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
	"github.com/unhield/limoxel/internal/capabilities/repository/xref"
)

// NavigationValidator performs deterministic validation of navigation targets, references, and paths.
type NavigationValidator struct{}

// NewNavigationValidator constructs a NavigationValidator.
func NewNavigationValidator() *NavigationValidator {
	return &NavigationValidator{}
}

// Validate executes all validation passes over the navigation model and supporting indexes.
func (v *NavigationValidator) Validate(model *NavigationModel, symDB *symbol.SymbolDatabase, xrefModel *xref.XRefModel) *NavigationValidationReport {
	var findings []*ValidationFinding

	if model == nil {
		findings = append(findings, NewValidationFinding(
			NavValMissingTarget,
			SeverityError,
			"engine",
			"model",
			"navigation model is nil",
			"validator",
		))
		return NewNavigationValidationReport(findings)
	}

	// 1. Validate Definitions
	if symDB != nil {
		for symID, def := range model.definitions {
			if def == nil || def.Target() == nil {
				if def != nil && def.State() == NavStateAmbiguous {
					findings = append(findings, NewValidationFinding(
						NavValAmbiguousTarget,
						SeverityWarning,
						symID,
						"",
						"multiple candidate definition targets exist without unique tiebreaker",
						"definition_validator",
					))
				} else {
					findings = append(findings, NewValidationFinding(
						NavValMissingTarget,
						SeverityError,
						symID,
						"",
						"definition target could not be resolved",
						"definition_validator",
					))
				}
			} else {
				// Verify target exists in SymbolDB
				if symDB.SymbolByID(def.Target().SymbolID()) == nil && !strings.HasPrefix(def.Target().ID(), "pkg:") && !strings.HasPrefix(def.Target().ID(), "mod:") {
					findings = append(findings, NewValidationFinding(
						NavValMissingTarget,
						SeverityWarning,
						symID,
						def.Target().SymbolID(),
						"definition target symbol is not found in active SymbolDatabase",
						"definition_validator",
					))
				}
			}
		}
	}

	// 2. Validate References
	if xrefModel != nil && xrefModel.References() != nil {
		for _, ref := range xrefModel.References().AllReferences() {
			if ref == nil {
				continue
			}
			if ref.TargetSymbolID() == "" || ref.State() == xref.StateBroken {
				findings = append(findings, NewValidationFinding(
					NavValBrokenReference,
					SeverityError,
					ref.SourceSymbolID(),
					ref.TargetSymbolID(),
					"cross-reference targets an unresolvable or broken symbol",
					"reference_validator",
				))
			}
		}
	}

	// 3. Validate Usages
	for targetID, usageList := range model.usages {
		seenUsageIDs := make(map[string]bool)
		for _, u := range usageList {
			if u == nil {
				continue
			}
			if seenUsageIDs[u.ID()] {
				findings = append(findings, NewValidationFinding(
					NavValDuplicatePath,
					SeverityWarning,
					u.SourceSymbolID(),
					targetID,
					"duplicate usage record detected",
					"usage_validator",
				))
			}
			seenUsageIDs[u.ID()] = true
		}
	}

	// 4. Validate Package Hierarchies
	for pkgPath, pkgNode := range model.packageHierarchy {
		if pkgNode == nil {
			findings = append(findings, NewValidationFinding(
				NavValMissingTarget,
				SeverityError,
				pkgPath,
				"",
				"package hierarchy node is nil",
				"hierarchy_validator",
			))
		}
	}

	return NewNavigationValidationReport(findings)
}
