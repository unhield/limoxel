package reasoning

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/unhield/limoxel/internal/capabilities/intelligence/knowledgegraph"
)

// RefactoringAdvisor provides deterministic structural safety and risk analysis for proposed refactorings.
// All operations are purely analytical and read-only.
type RefactoringAdvisor struct{}

// NewRefactoringAdvisor constructs an initialized RefactoringAdvisor.
func NewRefactoringAdvisor() *RefactoringAdvisor {
	return &RefactoringAdvisor{}
}

// AnalyzeRename evaluates whether renaming targetID to newName is structurally safe.
func (r *RefactoringAdvisor) AnalyzeRename(model *knowledgegraph.KnowledgeGraphModel, targetID, newName string) (*RefactoringSafetyResult, error) {
	if model == nil {
		return nil, ErrNilGraphModel
	}
	if strings.TrimSpace(targetID) == "" {
		return nil, NewReasoningError(ErrCatMissingTarget, "target ID cannot be empty", "", ErrMissingTarget)
	}
	cleanNewName := strings.TrimSpace(newName)
	if cleanNewName == "" {
		return nil, NewReasoningError(ErrCatInvalidInput, "new name cannot be empty", targetID, nil)
	}

	target := model.EntityByID(targetID)
	if target == nil {
		return nil, NewReasoningError(ErrCatMissingTarget, fmt.Sprintf("target entity %q not found", targetID), targetID, ErrMissingTarget)
	}

	var blockingReasons []string
	var unresolvedRefs []string
	var affectedContracts []string

	// 1. Check for name collisions in the same package/file scope
	targetPkg := target.PackagePath()
	for _, ent := range model.Entities() {
		if ent.ID() != target.ID() && ent.PackagePath() == targetPkg && ent.Name() == cleanNewName {
			blockingReasons = append(blockingReasons, fmt.Sprintf("name collision with existing entity %s in package %s", ent.ID(), targetPkg))
		}
	}

	// 2. Check inbound callers and references
	inbound := model.InboundRelationships(target.ID())
	for _, rel := range inbound {
		if rel.Kind() == knowledgegraph.RelCalls || rel.Kind() == knowledgegraph.RelDependsOn || rel.Kind() == knowledgegraph.RelImports {
			unresolvedRefs = append(unresolvedRefs, fmt.Sprintf("%s (%s)", rel.SourceID(), rel.Kind()))
		}
		if rel.Kind() == knowledgegraph.RelImplements {
			affectedContracts = append(affectedContracts, fmt.Sprintf("interface contract with %s", rel.SourceID()))
			blockingReasons = append(blockingReasons, fmt.Sprintf("cannot rename symbol implementing interface %s without breaking contract", rel.SourceID()))
		}
	}

	// 3. Exported status check
	isExportedOld := len(target.Name()) > 0 && unicode.IsUpper([]rune(target.Name())[0])
	isExportedNew := len(cleanNewName) > 0 && unicode.IsUpper([]rune(cleanNewName)[0])

	if isExportedOld && !isExportedNew && len(unresolvedRefs) > 0 {
		blockingReasons = append(blockingReasons, "changing exported symbol to unexported symbol breaks external package callers")
	}

	sort.Strings(blockingReasons)
	sort.Strings(unresolvedRefs)
	sort.Strings(affectedContracts)

	safe := len(blockingReasons) == 0
	classification := SafetySafe
	if len(blockingReasons) > 0 {
		classification = SafetyBlocked
	} else if len(unresolvedRefs) > 0 {
		classification = SafetyUnsafe
	}

	evidence := fmt.Sprintf("inspected %d inbound relationships, %d name collisions, %d interface contracts", len(inbound), len(blockingReasons), len(affectedContracts))

	return &RefactoringSafetyResult{
		TargetID:             target.ID(),
		Kind:                 RefactorRename,
		Classification:       classification,
		Safe:                 safe,
		BlockingReasons:      blockingReasons,
		UnresolvedReferences: unresolvedRefs,
		AffectedContracts:    affectedContracts,
		Evidence:             evidence,
		Provenance:           "knowledge_graph_model",
	}, nil
}

// AnalyzeMove evaluates whether moving targetID to destinationPkg is structurally safe.
func (r *RefactoringAdvisor) AnalyzeMove(model *knowledgegraph.KnowledgeGraphModel, targetID, destinationPkg string) (*RefactoringSafetyResult, error) {
	if model == nil {
		return nil, ErrNilGraphModel
	}
	if strings.TrimSpace(targetID) == "" {
		return nil, NewReasoningError(ErrCatMissingTarget, "target ID cannot be empty", "", ErrMissingTarget)
	}
	cleanDest := strings.TrimSpace(destinationPkg)
	if cleanDest == "" {
		return nil, NewReasoningError(ErrCatInvalidInput, "destination package cannot be empty", targetID, nil)
	}

	target := model.EntityByID(targetID)
	if target == nil {
		return nil, NewReasoningError(ErrCatMissingTarget, fmt.Sprintf("target entity %q not found", targetID), targetID, ErrMissingTarget)
	}

	var blockingReasons []string
	var unresolvedRefs []string
	var affectedContracts []string

	destPkgID := knowledgegraph.CanonicalEntityID(knowledgegraph.EntityPackage, cleanDest)
	destPkgEnt := model.EntityByID(destPkgID)
	if destPkgEnt == nil {
		blockingReasons = append(blockingReasons, fmt.Sprintf("destination package %s does not exist in repository", cleanDest))
	}

	// 1. Check unexported visibility
	isExported := len(target.Name()) > 0 && unicode.IsUpper([]rune(target.Name())[0])
	if !isExported && target.PackagePath() != cleanDest {
		inbound := model.InboundRelationships(target.ID())
		for _, rel := range inbound {
			src := model.EntityByID(rel.SourceID())
			if src != nil && src.PackagePath() == target.PackagePath() {
				unresolvedRefs = append(unresolvedRefs, fmt.Sprintf("unexported symbol will become inaccessible to caller %s in old package %s", src.ID(), target.PackagePath()))
			}
		}
	}

	// 2. Check for circular dependency creation
	// If destination package already depends on target's source package and target depends on destPkg
	if destPkgEnt != nil && target.PackagePath() != "" {
		for _, rel := range model.OutboundRelationships(destPkgID) {
			if rel.TargetID() == knowledgegraph.CanonicalEntityID(knowledgegraph.EntityPackage, target.PackagePath()) {
				blockingReasons = append(blockingReasons, fmt.Sprintf("move creates circular dependency between %s and %s", cleanDest, target.PackagePath()))
			}
		}
	}

	sort.Strings(blockingReasons)
	sort.Strings(unresolvedRefs)
	sort.Strings(affectedContracts)

	safe := len(blockingReasons) == 0 && len(unresolvedRefs) == 0
	classification := SafetySafe
	if len(blockingReasons) > 0 {
		classification = SafetyBlocked
	} else if len(unresolvedRefs) > 0 {
		classification = SafetyUnsafe
	}

	evidence := fmt.Sprintf("evaluated move of %s to %s against visibility, package boundaries, and circular dependency risks", target.ID(), cleanDest)

	return &RefactoringSafetyResult{
		TargetID:             target.ID(),
		Kind:                 RefactorMove,
		Classification:       classification,
		Safe:                 safe,
		BlockingReasons:      blockingReasons,
		UnresolvedReferences: unresolvedRefs,
		AffectedContracts:    affectedContracts,
		Evidence:             evidence,
		Provenance:           "knowledge_graph_model",
	}, nil
}

// AnalyzeExtraction evaluates whether extracting functionality from targetID into extractionScope is structurally coherent.
func (r *RefactoringAdvisor) AnalyzeExtraction(model *knowledgegraph.KnowledgeGraphModel, targetID, extractionScope string) (*RefactoringSafetyResult, error) {
	if model == nil {
		return nil, ErrNilGraphModel
	}
	if strings.TrimSpace(targetID) == "" {
		return nil, NewReasoningError(ErrCatMissingTarget, "target ID cannot be empty", "", ErrMissingTarget)
	}

	target := model.EntityByID(targetID)
	if target == nil {
		return nil, NewReasoningError(ErrCatMissingTarget, fmt.Sprintf("target entity %q not found", targetID), targetID, ErrMissingTarget)
	}

	var blockingReasons []string
	var unresolvedRefs []string
	var affectedContracts []string

	// Count outbound calls to check coupling
	outboundCalls := 0
	for _, rel := range model.OutboundRelationships(target.ID()) {
		if rel.Kind() == knowledgegraph.RelCalls {
			outboundCalls++
		}
		if rel.Kind() == knowledgegraph.RelImplements {
			affectedContracts = append(affectedContracts, fmt.Sprintf("extracting interface implementation %s", rel.TargetID()))
		}
	}

	if outboundCalls > 15 {
		blockingReasons = append(blockingReasons, fmt.Sprintf("high coupling: target has %d outbound calls, extraction may require complex parameter passing", outboundCalls))
	}

	sort.Strings(blockingReasons)
	sort.Strings(unresolvedRefs)
	sort.Strings(affectedContracts)

	safe := len(blockingReasons) == 0
	classification := SafetySafe
	if len(blockingReasons) > 0 {
		classification = SafetyUnsafe
	}

	evidence := fmt.Sprintf("evaluated extraction cohesion for %s with %d outbound calls and %d interface contracts", target.ID(), outboundCalls, len(affectedContracts))

	return &RefactoringSafetyResult{
		TargetID:             target.ID(),
		Kind:                 RefactorExtraction,
		Classification:       classification,
		Safe:                 safe,
		BlockingReasons:      blockingReasons,
		UnresolvedReferences: unresolvedRefs,
		AffectedContracts:    affectedContracts,
		Evidence:             evidence,
		Provenance:           "knowledge_graph_model",
	}, nil
}

// AnalyzeDeletion evaluates whether targetID can be safely deleted without leaving broken references.
func (r *RefactoringAdvisor) AnalyzeDeletion(model *knowledgegraph.KnowledgeGraphModel, targetID string) (*RefactoringSafetyResult, error) {
	if model == nil {
		return nil, ErrNilGraphModel
	}
	if strings.TrimSpace(targetID) == "" {
		return nil, NewReasoningError(ErrCatMissingTarget, "target ID cannot be empty", "", ErrMissingTarget)
	}

	target := model.EntityByID(targetID)
	if target == nil {
		return nil, NewReasoningError(ErrCatMissingTarget, fmt.Sprintf("target entity %q not found", targetID), targetID, ErrMissingTarget)
	}

	var blockingReasons []string
	var unresolvedRefs []string
	var affectedContracts []string

	inbound := model.InboundRelationships(target.ID())
	for _, rel := range inbound {
		if rel.Kind() == knowledgegraph.RelCalls || rel.Kind() == knowledgegraph.RelDependsOn || rel.Kind() == knowledgegraph.RelImports {
			unresolvedRefs = append(unresolvedRefs, fmt.Sprintf("%s (%s)", rel.SourceID(), rel.Kind()))
			blockingReasons = append(blockingReasons, fmt.Sprintf("deletion leaves unresolved reference from %s", rel.SourceID()))
		}
		if rel.Kind() == knowledgegraph.RelImplements {
			affectedContracts = append(affectedContracts, fmt.Sprintf("interface implementation of %s", rel.SourceID()))
			blockingReasons = append(blockingReasons, fmt.Sprintf("deletion breaks interface implementation for %s", rel.SourceID()))
		}
	}

	sort.Strings(blockingReasons)
	sort.Strings(unresolvedRefs)
	sort.Strings(affectedContracts)

	safe := len(blockingReasons) == 0
	classification := SafetySafe
	if len(blockingReasons) > 0 {
		classification = SafetyBlocked
	}

	evidence := fmt.Sprintf("inspected %d inbound references for deletion of %s", len(inbound), target.ID())

	return &RefactoringSafetyResult{
		TargetID:             target.ID(),
		Kind:                 RefactorDeletion,
		Classification:       classification,
		Safe:                 safe,
		BlockingReasons:      blockingReasons,
		UnresolvedReferences: unresolvedRefs,
		AffectedContracts:    affectedContracts,
		Evidence:             evidence,
		Provenance:           "knowledge_graph_model",
	}, nil
}

// AssessRisk calculates a deterministic refactoring risk assessment for targetID.
func (r *RefactoringAdvisor) AssessRisk(model *knowledgegraph.KnowledgeGraphModel, targetID string) (*RefactoringRiskAssessment, error) {
	if model == nil {
		return nil, ErrNilGraphModel
	}
	if strings.TrimSpace(targetID) == "" {
		return nil, NewReasoningError(ErrCatMissingTarget, "target ID cannot be empty", "", ErrMissingTarget)
	}

	target := model.EntityByID(targetID)
	if target == nil {
		return nil, NewReasoningError(ErrCatMissingTarget, fmt.Sprintf("target entity %q not found", targetID), targetID, ErrMissingTarget)
	}

	var factors []string
	directRefs := 0
	crossModuleRefs := 0

	inbound := model.InboundRelationships(target.ID())
	for _, rel := range inbound {
		if rel.Kind() == knowledgegraph.RelCalls || rel.Kind() == knowledgegraph.RelDependsOn || rel.Kind() == knowledgegraph.RelImports {
			directRefs++
			src := model.EntityByID(rel.SourceID())
			if src != nil && src.PackagePath() != "" && target.PackagePath() != "" && src.PackagePath() != target.PackagePath() {
				crossModuleRefs++
			}
		}
	}

	isExported := len(target.Name()) > 0 && unicode.IsUpper([]rune(target.Name())[0])

	score := 0
	if directRefs > 0 {
		score += directRefs * 2
		factors = append(factors, fmt.Sprintf("%d direct references", directRefs))
	}
	if crossModuleRefs > 0 {
		score += crossModuleRefs * 5
		factors = append(factors, fmt.Sprintf("%d cross-package/module references", crossModuleRefs))
	}
	if isExported {
		score += 10
		factors = append(factors, "symbol is exported in public API")
	}

	outbound := model.OutboundRelationships(target.ID())
	transitiveDeps := len(outbound)
	if transitiveDeps > 5 {
		score += transitiveDeps
		factors = append(factors, fmt.Sprintf("%d outbound dependencies", transitiveDeps))
	}

	sort.Strings(factors)

	var risk RiskLevel
	switch {
	case score >= 30:
		risk = RiskCritical
	case score >= 15:
		risk = RiskHigh
	case score >= 5:
		risk = RiskMedium
	default:
		risk = RiskLow
	}

	return &RefactoringRiskAssessment{
		TargetID:            target.ID(),
		Risk:                risk,
		Score:               score,
		ContributingFactors: factors,
		DirectReferences:    directRefs,
		TransitiveDeps:      transitiveDeps,
		CrossModuleRefs:     crossModuleRefs,
		ExportedAPI:         isExported,
	}, nil
}
