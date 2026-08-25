package reasoning

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/unhield/limoxel/internal/capabilities/intelligence/knowledgegraph"
)

// BreakingChangeAnalyzer provides deterministic detection of API, package, symbol, interface, and versioning breaks.
type BreakingChangeAnalyzer struct{}

// NewBreakingChangeAnalyzer constructs an initialized BreakingChangeAnalyzer.
func NewBreakingChangeAnalyzer() *BreakingChangeAnalyzer {
	return &BreakingChangeAnalyzer{}
}

// AnalyzeBreakingChanges compares a baseline model with a target model to identify compatibility shifts and breaking changes.
func (b *BreakingChangeAnalyzer) AnalyzeBreakingChanges(baseline, target *knowledgegraph.KnowledgeGraphModel) (*BreakingChangeReport, error) {
	if target == nil {
		return nil, ErrNilGraphModel
	}

	var findings []*BreakingChangeFinding
	baselineRoot := ""
	if baseline != nil {
		baselineRoot = baseline.RootPath()
	}

	if baseline != nil {
		// 1. Symbol & API Change Detection
		baselineSymbols := make(map[string]*knowledgegraph.GraphEntity)
		for _, s := range baseline.EntitiesByType(knowledgegraph.EntitySymbol) {
			baselineSymbols[s.ID()] = s
		}

		targetSymbols := make(map[string]*knowledgegraph.GraphEntity)
		for _, s := range target.EntitiesByType(knowledgegraph.EntitySymbol) {
			targetSymbols[s.ID()] = s
		}

		// Detect removed symbols
		for id, baseSym := range baselineSymbols {
			if _, exists := targetSymbols[id]; !exists {
				isExported := len(baseSym.Name()) > 0 && unicode.IsUpper([]rune(baseSym.Name())[0])
				cat := BreakSymbolRemoval
				class := CompatBreaking
				if !isExported {
					class = CompatPotentiallyBreaking
				}
				change := fmt.Sprintf("symbol %s (%s) was removed", baseSym.Name(), baseSym.ID())
				findings = append(findings, &BreakingChangeFinding{
					ID:             CanonicalBreakingFindingID(cat, id, change),
					Category:       cat,
					Classification: class,
					AffectedEntity: id,
					ChangeSummary:  change,
					Reason:         "removed symbol breaks existing callers and consumers",
					Evidence:       fmt.Sprintf("present in baseline at %s:%s, missing in target", baseSym.PackagePath(), baseSym.FilePath()),
					Provenance:     "baseline_comparison",
				})
			}
		}

		// Detect added symbols and signature/type changes
		for id, tgtSym := range targetSymbols {
			baseSym, exists := baselineSymbols[id]
			if !exists {
				// Added symbol
				isExported := len(tgtSym.Name()) > 0 && unicode.IsUpper([]rune(tgtSym.Name())[0])
				if isExported {
					cat := BreakAPIChange
					change := fmt.Sprintf("exported symbol %s added to API", tgtSym.Name())
					findings = append(findings, &BreakingChangeFinding{
						ID:             CanonicalBreakingFindingID(cat, id, change),
						Category:       cat,
						Classification: CompatAdditive,
						AffectedEntity: id,
						ChangeSummary:  change,
						Reason:         "new exported API symbol expands public surface",
						Evidence:       fmt.Sprintf("added in target at %s:%s", tgtSym.PackagePath(), tgtSym.FilePath()),
						Provenance:     "target_comparison",
					})
				}
			} else {
				// Existing symbol: check signature or attributes
				baseSig := baseSym.Attribute("signature")
				tgtSig := tgtSym.Attribute("signature")
				if baseSig != "" && tgtSig != "" && baseSig != tgtSig {
					cat := BreakAPIChange
					change := fmt.Sprintf("signature changed from %q to %q", baseSig, tgtSig)
					findings = append(findings, &BreakingChangeFinding{
						ID:             CanonicalBreakingFindingID(cat, id, change),
						Category:       cat,
						Classification: CompatBreaking,
						AffectedEntity: id,
						ChangeSummary:  change,
						Reason:         "modified function or method signature breaks call sites",
						Evidence:       fmt.Sprintf("baseline: %s | target: %s", baseSig, tgtSig),
						Provenance:     "signature_comparison",
					})
				}
			}
		}

		// 2. Package Change Detection
		baselinePkgs := make(map[string]*knowledgegraph.GraphEntity)
		for _, p := range baseline.EntitiesByType(knowledgegraph.EntityPackage) {
			baselinePkgs[p.ID()] = p
		}

		targetPkgs := make(map[string]*knowledgegraph.GraphEntity)
		for _, p := range target.EntitiesByType(knowledgegraph.EntityPackage) {
			targetPkgs[p.ID()] = p
		}

		for id, basePkg := range baselinePkgs {
			if _, exists := targetPkgs[id]; !exists {
				cat := BreakPackageChange
				change := fmt.Sprintf("package %s was removed", basePkg.PackagePath())
				findings = append(findings, &BreakingChangeFinding{
					ID:             CanonicalBreakingFindingID(cat, id, change),
					Category:       cat,
					Classification: CompatBreaking,
					AffectedEntity: id,
					ChangeSummary:  change,
					Reason:         "deleted package breaks downstream imports and dependent modules",
					Evidence:       fmt.Sprintf("present in baseline at %s, missing in target", basePkg.PackagePath()),
					Provenance:     "package_comparison",
				})
			}
		}

		// 3. Interface Change Detection
		for _, tgtSym := range target.EntitiesByType(knowledgegraph.EntitySymbol) {
			if tgtSym.Attribute("kind") == "interface" {
				baseSym := baselineSymbols[tgtSym.ID()]
				if baseSym != nil {
					baseMethods := baseSym.Attribute("methods")
					tgtMethods := tgtSym.Attribute("methods")
					if baseMethods != "" && tgtMethods != "" && baseMethods != tgtMethods {
						cat := BreakInterfaceChange
						change := fmt.Sprintf("interface %s method contract modified", tgtSym.Name())
						findings = append(findings, &BreakingChangeFinding{
							ID:             CanonicalBreakingFindingID(cat, tgtSym.ID(), change),
							Category:       cat,
							Classification: CompatBreaking,
							AffectedEntity: tgtSym.ID(),
							ChangeSummary:  change,
							Reason:         "modifying interface method contract breaks implementing types",
							Evidence:       fmt.Sprintf("baseline methods: %s | target methods: %s", baseMethods, tgtMethods),
							Provenance:     "interface_comparison",
						})
					}
				}
			}
		}
	} else {
		// Single-model analysis: detect potentially unexported breaking API signatures
		for _, s := range target.EntitiesByType(knowledgegraph.EntitySymbol) {
			isExported := len(s.Name()) > 0 && unicode.IsUpper([]rune(s.Name())[0])
			if isExported {
				inbound := target.InboundRelationships(s.ID())
				for _, r := range inbound {
					src := target.EntityByID(r.SourceID())
					if src != nil && src.PackagePath() != s.PackagePath() && r.Kind() == knowledgegraph.RelCalls {
						// Cross-package caller
					}
				}
			}
		}
	}

	dedupFindings := DeduplicateAndSortBreakingFindings(findings)

	summaryByCat := make(map[string]int)
	summaryBySev := make(map[string]int)
	hasBreaking := false

	for _, f := range dedupFindings {
		summaryByCat[string(f.Category)]++
		summaryBySev[string(f.Classification)]++
		if f.Classification == CompatBreaking || f.Classification == CompatPotentiallyBreaking {
			hasBreaking = true
		}
	}

	return &BreakingChangeReport{
		BaselineRoot:       baselineRoot,
		TargetRoot:         target.RootPath(),
		HasBreakingChanges: hasBreaking,
		Findings:           dedupFindings,
		SummaryByCategory:  summaryByCat,
		SummaryBySeverity:  summaryBySev,
	}, nil
}

// AnalyzeSymbolRemoval evaluates the breaking consequence of removing a single symbol in the current graph model.
func (b *BreakingChangeAnalyzer) AnalyzeSymbolRemoval(model *knowledgegraph.KnowledgeGraphModel, symbolID string) (*BreakingChangeFinding, error) {
	if model == nil {
		return nil, ErrNilGraphModel
	}
	if strings.TrimSpace(symbolID) == "" {
		return nil, NewReasoningError(ErrCatMissingTarget, "symbol ID cannot be empty", "", ErrMissingTarget)
	}

	sym := model.EntityByID(symbolID)
	if sym == nil {
		sym = model.EntityByID(knowledgegraph.CanonicalEntityID(knowledgegraph.EntitySymbol, symbolID))
		if sym == nil {
			return nil, NewReasoningError(ErrCatMissingTarget, fmt.Sprintf("symbol %q not found", symbolID), symbolID, ErrMissingTarget)
		}
	}

	inbound := model.InboundRelationships(sym.ID())
	callers := 0
	crossPkgCallers := 0
	for _, rel := range inbound {
		if rel.Kind() == knowledgegraph.RelCalls || rel.Kind() == knowledgegraph.RelDependsOn {
			callers++
			src := model.EntityByID(rel.SourceID())
			if src != nil && src.PackagePath() != "" && sym.PackagePath() != "" && src.PackagePath() != sym.PackagePath() {
				crossPkgCallers++
			}
		}
	}

	isExported := len(sym.Name()) > 0 && unicode.IsUpper([]rune(sym.Name())[0])

	var class CompatibilityClassification
	if isExported && callers > 0 {
		class = CompatBreaking
	} else if callers > 0 {
		class = CompatPotentiallyBreaking
	} else {
		class = CompatCompatible
	}

	change := fmt.Sprintf("symbol %s removal evaluated", sym.Name())
	cat := BreakSymbolRemoval

	return &BreakingChangeFinding{
		ID:             CanonicalBreakingFindingID(cat, sym.ID(), change),
		Category:       cat,
		Classification: class,
		AffectedEntity: sym.ID(),
		ChangeSummary:  change,
		Reason:         fmt.Sprintf("symbol has %d inbound callers (%d cross-package)", callers, crossPkgCallers),
		Evidence:       fmt.Sprintf("isExported=%v, inboundCount=%d", isExported, len(inbound)),
		Provenance:     "symbol_inbound_analysis",
	}, nil
}
