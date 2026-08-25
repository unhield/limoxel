package analysis

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/intelligence/crossrepo"
	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
	"github.com/unhield/limoxel/internal/capabilities/repository/xref"
)

// ArchitectureAnalyzer executes Task 3: Architecture Analysis.
type ArchitectureAnalyzer struct {
	crossRepoModel *crossrepo.CrossRepoModel
	symbolDB       *symbol.SymbolDatabase
	xrefModel      *xref.XRefModel
}

// NewArchitectureAnalyzer constructs an ArchitectureAnalyzer.
func NewArchitectureAnalyzer(
	crossModel *crossrepo.CrossRepoModel,
	symDB *symbol.SymbolDatabase,
	xrefModel *xref.XRefModel,
) *ArchitectureAnalyzer {
	return &ArchitectureAnalyzer{
		crossRepoModel: crossModel,
		symbolDB:       symDB,
		xrefModel:      xrefModel,
	}
}

// Analyze executes all architecture rules and returns an AnalyzerResult.
func (a *ArchitectureAnalyzer) Analyze() *AnalyzerResult {
	ruleResults := make(map[RuleID]*AnalysisRuleResult)

	ruleResults[RuleArchitectureViolations] = a.analyzeArchitectureViolations()
	ruleResults[RuleModuleBoundaries] = a.analyzeModuleBoundaries()
	ruleResults[RuleLayerConsistency] = a.analyzeLayerConsistency()
	ruleResults[RuleRepositoryOrganization] = a.analyzeRepositoryOrganization()
	ruleResults[RulePackageCohesion] = a.analyzePackageCohesion()

	return NewAnalyzerResult("architecture", ruleResults)
}

// 10.1 Architecture Violations
func (a *ArchitectureAnalyzer) analyzeArchitectureViolations() *AnalysisRuleResult {
	var findings []*Finding

	if a.crossRepoModel != nil {
		for _, comm := range a.crossRepoModel.PackageCommunications() {
			if comm == nil {
				continue
			}
			src := comm.SourcePackage()
			tgt := comm.TargetPackage()

			// Specific rule: Core engine / platform cannot depend on intelligence capabilities
			if (strings.Contains(src, "platform") || strings.Contains(src, "engine")) && strings.Contains(tgt, "intelligence") {
				finding := NewFinding(
					"architecture",
					RuleArchitectureViolations,
					CategoryArchitecture,
					SeverityCritical,
					ConfidenceDefinite,
					fmt.Sprintf("Architecture violation: %s imports %s", src, tgt),
					fmt.Sprintf("Core engine/platform component %s illegally imports high-level intelligence capability %s.", src, tgt),
					"",
					"",
					src,
					"",
					"",
					nil,
					fmt.Sprintf("forbidden dependency %s -> %s", src, tgt),
					[]string{src, tgt},
					"Remove direct dependency on intelligence; use dependency injection or events.",
					"cross_repo_model",
				)
				findings = append(findings, finding)
			}
		}
	}

	status := StatusNoFindings
	if len(findings) > 0 {
		status = StatusFindingsPresent
	}
	return NewAnalysisRuleResult(RuleArchitectureViolations, status, findings, fmt.Sprintf("architecture violations analysis evaluated %d findings", len(findings)))
}

// 10.2 Module Boundaries
func (a *ArchitectureAnalyzer) analyzeModuleBoundaries() *AnalysisRuleResult {
	var findings []*Finding

	if a.crossRepoModel != nil && a.crossRepoModel.Workspace() != nil {
		ws := a.crossRepoModel.Workspace()
		for _, repo := range ws.Repositories() {
			if repo == nil {
				continue
			}
			// Check cross-repo / cross-module communications
			for _, comm := range a.crossRepoModel.PackageCommunications() {
				if comm == nil {
					continue
				}
				src := comm.SourcePackage()
				tgt := comm.TargetPackage()

				// If target package contains "/internal/" but source is outside that repository / module
				if strings.Contains(tgt, "/internal/") || strings.HasPrefix(tgt, "internal/") {
					// Check if source belongs to a different module
					srcMod := getModulePrefix(src)
					tgtMod := getModulePrefix(tgt)
					if srcMod != "" && tgtMod != "" && srcMod != tgtMod {
						finding := NewFinding(
							"architecture",
							RuleModuleBoundaries,
							CategoryArchitecture,
							SeverityHigh,
							ConfidenceDefinite,
							fmt.Sprintf("Module boundary violation: %s imports private package %s", src, tgt),
							fmt.Sprintf("Module %s accesses private internal package %s belonging to external module %s.", srcMod, tgt, tgtMod),
							repo.Name(),
							srcMod,
							src,
							"",
							"",
							nil,
							fmt.Sprintf("illegal cross-module internal access %s -> %s", src, tgt),
							[]string{src, tgt},
							"Expose required functionality via public package or move components to the same module.",
							"cross_repo_model",
						)
						findings = append(findings, finding)
					}
				}
			}
		}
	}

	status := StatusNoFindings
	if len(findings) > 0 {
		status = StatusFindingsPresent
	}
	return NewAnalysisRuleResult(RuleModuleBoundaries, status, findings, fmt.Sprintf("module boundaries analysis evaluated %d findings", len(findings)))
}

func getModulePrefix(pkgPath string) string {
	parts := strings.Split(filepath.ToSlash(pkgPath), "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// 10.3 Layer Consistency
func (a *ArchitectureAnalyzer) analyzeLayerConsistency() *AnalysisRuleResult {
	var findings []*Finding

	if a.crossRepoModel != nil {
		for _, comm := range a.crossRepoModel.PackageCommunications() {
			if comm == nil {
				continue
			}
			src := comm.SourcePackage()
			tgt := comm.TargetPackage()

			// Check CLI importing internal capabilities directly without engine coordination
			if strings.HasPrefix(src, "cmd/") && strings.Contains(tgt, "capabilities/intelligence/navigation") {
				// CLI should use engine rather than sub-packages directly
				finding := NewFinding(
					"architecture",
					RuleLayerConsistency,
					CategoryArchitecture,
					SeverityLow,
					ConfidenceLikely,
					fmt.Sprintf("Layer inconsistency: %s directly accesses internal subpackage %s", src, tgt),
					fmt.Sprintf("Top-level entry %s bypasses coordinator and directly references subpackage %s.", src, tgt),
					"",
					"",
					src,
					"",
					"",
					nil,
					fmt.Sprintf("direct subpackage dependency %s -> %s", src, tgt),
					[]string{src, tgt},
					"Route capability requests through the unified engine coordinator.",
					"cross_repo_model",
				)
				findings = append(findings, finding)
			}
		}
	}

	status := StatusNoFindings
	if len(findings) > 0 {
		status = StatusFindingsPresent
	}
	return NewAnalysisRuleResult(RuleLayerConsistency, status, findings, fmt.Sprintf("layer consistency analysis evaluated %d findings", len(findings)))
}

// 10.4 Repository Organization
func (a *ArchitectureAnalyzer) analyzeRepositoryOrganization() *AnalysisRuleResult {
	var findings []*Finding

	if a.symbolDB != nil {
		seenFiles := make(map[string]bool)
		for _, sym := range a.symbolDB.AllSymbols() {
			if sym == nil {
				continue
			}
			fPath := filepath.ToSlash(sym.FilePath())
			if seenFiles[fPath] {
				continue
			}
			seenFiles[fPath] = true

			// Check files placed outside canonical root folders (cmd, internal, pkg, docs, tests)
			if !strings.Contains(fPath, "/") {
				// Root file (allow main.go, go.mod, etc.)
				base := filepath.Base(fPath)
				if base != "main.go" && base != "root.go" && !strings.HasSuffix(base, ".go") {
					continue
				}
				if base != "main.go" {
					finding := NewFinding(
						"architecture",
						RuleRepositoryOrganization,
						CategoryArchitecture,
						SeverityInfo,
						ConfidenceLikely,
						fmt.Sprintf("Misplaced root source file: %s", fPath),
						fmt.Sprintf("Source file %s is placed directly in the repository root directory instead of cmd/ or internal/.", fPath),
						"",
						"",
						"",
						fPath,
						"",
						nil,
						fmt.Sprintf("root source location: %s", fPath),
						nil,
						"Move source files into cmd/ or internal/ packages.",
						"symbol_db",
					)
					findings = append(findings, finding)
				}
			}
		}
	}

	status := StatusNoFindings
	if len(findings) > 0 {
		status = StatusFindingsPresent
	}
	return NewAnalysisRuleResult(RuleRepositoryOrganization, status, findings, fmt.Sprintf("repository organization analysis evaluated %d findings", len(findings)))
}

// 10.5 Package Cohesion
func (a *ArchitectureAnalyzer) analyzePackageCohesion() *AnalysisRuleResult {
	var findings []*Finding

	if a.symbolDB != nil {
		pkgSymbols := make(map[string][]*symbol.Symbol)
		for _, sym := range a.symbolDB.AllSymbols() {
			if sym != nil && sym.PackagePath() != "" {
				pkgSymbols[sym.PackagePath()] = append(pkgSymbols[sym.PackagePath()], sym)
			}
		}

		var sortedPkgs []string
		for pkgPath := range pkgSymbols {
			sortedPkgs = append(sortedPkgs, pkgPath)
		}
		sort.Strings(sortedPkgs)

		for _, pkgPath := range sortedPkgs {
			symbols := pkgSymbols[pkgPath]
			// A package with more than 40 symbols across 10 distinct files is evaluated for fragmentation
			files := make(map[string]bool)
			for _, s := range symbols {
				files[s.FilePath()] = true
			}
			if len(symbols) > 50 && len(files) > 10 {
				finding := NewFinding(
					"architecture",
					RulePackageCohesion,
					CategoryArchitecture,
					SeverityLow,
					ConfidenceTentative,
					fmt.Sprintf("Low package cohesion in %s (%d symbols across %d files)", pkgPath, len(symbols), len(files)),
					fmt.Sprintf("Package %s contains %d symbols distributed over %d files, suggesting potential responsibility sprawl.", pkgPath, len(symbols), len(files)),
					"",
					"",
					pkgPath,
					"",
					"",
					nil,
					fmt.Sprintf("symbol count %d across %d files", len(symbols), len(files)),
					nil,
					"Consider partitioning package into cohesive sub-packages with distinct single responsibilities.",
					"symbol_db",
				)
				findings = append(findings, finding)
			}
		}
	}

	// Sort findings
	sort.Slice(findings, func(i, j int) bool {
		return findings[i].ID() < findings[j].ID()
	})

	status := StatusNoFindings
	if len(findings) > 0 {
		status = StatusFindingsPresent
	}
	return NewAnalysisRuleResult(RulePackageCohesion, status, findings, fmt.Sprintf("package cohesion analysis evaluated %d findings", len(findings)))
}
