package analysis

import (
	"fmt"
	"sort"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/intelligence/crossrepo"
	"github.com/unhield/limoxel/internal/capabilities/repository/dependency"
	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
	"github.com/unhield/limoxel/internal/capabilities/repository/xref"
)

// DependencyAnalyzer executes Task 2: Dependency Analysis.
type DependencyAnalyzer struct {
	depModel       *dependency.DependencyModel
	crossRepoModel *crossrepo.CrossRepoModel
	symbolDB       *symbol.SymbolDatabase
	xrefModel      *xref.XRefModel
}

// NewDependencyAnalyzer constructs a DependencyAnalyzer.
func NewDependencyAnalyzer(
	depModel *dependency.DependencyModel,
	crossModel *crossrepo.CrossRepoModel,
	symDB *symbol.SymbolDatabase,
	xrefModel *xref.XRefModel,
) *DependencyAnalyzer {
	return &DependencyAnalyzer{
		depModel:       depModel,
		crossRepoModel: crossModel,
		symbolDB:       symDB,
		xrefModel:      xrefModel,
	}
}

// Analyze executes all dependency rules and returns an AnalyzerResult.
func (a *DependencyAnalyzer) Analyze() *AnalyzerResult {
	ruleResults := make(map[RuleID]*AnalysisRuleResult)

	ruleResults[RuleCircularDependencies] = a.analyzeCircularDependencies()
	ruleResults[RuleLayerViolations] = a.analyzeLayerViolations()
	ruleResults[RuleInvalidImports] = a.analyzeInvalidImports()
	ruleResults[RuleTightCoupling] = a.analyzeTightCoupling()
	ruleResults[RuleOrphanPackages] = a.analyzeOrphanPackages()

	return NewAnalyzerResult("dependency", ruleResults)
}

// CanonicalizeCycle returns a canonical string representation of a cycle.
func CanonicalizeCycle(cycle []string) string {
	if len(cycle) == 0 {
		return ""
	}
	// Find minimum element index
	minIdx := 0
	for i := 1; i < len(cycle); i++ {
		if cycle[i] < cycle[minIdx] {
			minIdx = i
		}
	}
	// Rotate cycle to start with minimum element
	rotated := make([]string, len(cycle))
	for i := 0; i < len(cycle); i++ {
		rotated[i] = cycle[(minIdx+i)%len(cycle)]
	}
	return strings.Join(rotated, " -> ")
}

// 9.1 Circular Dependencies
func (a *DependencyAnalyzer) analyzeCircularDependencies() *AnalysisRuleResult {
	var findings []*Finding
	seenCycles := make(map[string]bool)

	if a.depModel != nil {
		for _, cycle := range a.depModel.Cycles() {
			if len(cycle) < 2 {
				continue
			}
			canonical := CanonicalizeCycle(cycle)
			if seenCycles[canonical] {
				continue
			}
			seenCycles[canonical] = true

			finding := NewFinding(
				"dependency",
				RuleCircularDependencies,
				CategoryDependency,
				SeverityCritical,
				ConfidenceDefinite,
				fmt.Sprintf("Circular dependency: %s", canonical),
				fmt.Sprintf("Dependency cycle detected across %d packages: %s", len(cycle), canonical),
				"",
				"",
				cycle[0],
				"",
				"",
				nil,
				fmt.Sprintf("dependency graph cycle: %s", canonical),
				cycle,
				"Refactor cycle by introducing an interface boundary or extracting common dependencies.",
				"dependency_model",
			)
			findings = append(findings, finding)
		}
	}

	status := StatusNoFindings
	if len(findings) > 0 {
		status = StatusFindingsPresent
	}
	return NewAnalysisRuleResult(RuleCircularDependencies, status, findings, fmt.Sprintf("circular dependencies analysis evaluated %d findings", len(findings)))
}

// 9.2 Layer Violations
func (a *DependencyAnalyzer) analyzeLayerViolations() *AnalysisRuleResult {
	var findings []*Finding

	// Standard Limoxel architectural layer ranks (lower rank = base layer, higher rank = consumer layer)
	getLayerRank := func(pkgPath string) int {
		clean := strings.ToLower(pkgPath)
		switch {
		case strings.Contains(clean, "platform") || strings.Contains(clean, "filesystem") || strings.Contains(clean, "parser") || strings.Contains(clean, "language"):
			return 1 // Foundation layer
		case strings.Contains(clean, "capabilities/repository") || strings.Contains(clean, "engine") || strings.Contains(clean, "repository"):
			return 2 // Repository capability layer
		case strings.Contains(clean, "capabilities/intelligence"):
			return 3 // Intelligence layer
		case strings.Contains(clean, "cli") || strings.Contains(clean, "cmd"):
			return 4 // Presentation / CLI layer
		default:
			return 0 // Unknown layer
		}
	}

	if a.crossRepoModel != nil {
		for _, comm := range a.crossRepoModel.PackageCommunications() {
			if comm == nil {
				continue
			}
			srcRank := getLayerRank(comm.SourcePackage())
			tgtRank := getLayerRank(comm.TargetPackage())

			// Foundation layer (rank 1) must never import capability (rank 2), intelligence (rank 3), or CLI (rank 4)
			// Repository layer (rank 2) must never import intelligence (rank 3) or CLI (rank 4)
			if srcRank > 0 && tgtRank > 0 && srcRank < tgtRank {
				finding := NewFinding(
					"dependency",
					RuleLayerViolations,
					CategoryDependency,
					SeverityHigh,
					ConfidenceDefinite,
					fmt.Sprintf("Layer violation: %s imports %s", comm.SourcePackage(), comm.TargetPackage()),
					fmt.Sprintf("Lower architectural layer %s (layer %d) imports higher architectural layer %s (layer %d).", comm.SourcePackage(), srcRank, comm.TargetPackage(), tgtRank),
					"",
					"",
					comm.SourcePackage(),
					"",
					"",
					nil,
					fmt.Sprintf("layer rank %d -> layer rank %d inversion", srcRank, tgtRank),
					[]string{comm.SourcePackage(), comm.TargetPackage()},
					"Invert dependency using abstraction or move consumer logic to a higher layer.",
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
	return NewAnalysisRuleResult(RuleLayerViolations, status, findings, fmt.Sprintf("layer violations analysis evaluated %d findings", len(findings)))
}

// 9.3 Invalid Imports
func (a *DependencyAnalyzer) analyzeInvalidImports() *AnalysisRuleResult {
	if a.xrefModel == nil || a.xrefModel.References() == nil {
		return NewAnalysisRuleResult(RuleInvalidImports, StatusInsufficientEvidence, nil, "cross-reference model unavailable")
	}

	var findings []*Finding

	for _, ref := range a.xrefModel.References().AllReferences() {
		if ref == nil {
			continue
		}
		if ref.State() == xref.StateBroken || ref.TargetSymbolID() == "" {
			finding := NewFinding(
				"dependency",
				RuleInvalidImports,
				CategoryDependency,
				SeverityHigh,
				ConfidenceDefinite,
				fmt.Sprintf("Invalid or unresolvable import in %s", ref.FilePath()),
				fmt.Sprintf("Import reference to target '%s' in file %s could not be resolved in the workspace.", ref.TargetSymbolID(), ref.FilePath()),
				"",
				"",
				"",
				ref.FilePath(),
				ref.SourceSymbolID(),
				ref.Position(),
				fmt.Sprintf("broken reference state %s targeting %s", ref.State(), ref.TargetSymbolID()),
				nil,
				"Verify import path and ensure the target dependency package exists.",
				"xref_model",
			)
			findings = append(findings, finding)
		}
	}

	status := StatusNoFindings
	if len(findings) > 0 {
		status = StatusFindingsPresent
	}
	return NewAnalysisRuleResult(RuleInvalidImports, status, findings, fmt.Sprintf("invalid imports analysis evaluated %d findings", len(findings)))
}

// 9.4 Tight Coupling
func (a *DependencyAnalyzer) analyzeTightCoupling() *AnalysisRuleResult {
	var findings []*Finding

	// Evaluate bidirectional package relationships and excessive fan-in / fan-out
	if a.crossRepoModel != nil {
		outboundCounts := make(map[string]int)
		inboundCounts := make(map[string]int)
		pairTraffic := make(map[string]map[string]int)

		for _, comm := range a.crossRepoModel.PackageCommunications() {
			if comm == nil {
				continue
			}
			src := comm.SourcePackage()
			tgt := comm.TargetPackage()
			outboundCounts[src]++
			inboundCounts[tgt]++

			if pairTraffic[src] == nil {
				pairTraffic[src] = make(map[string]int)
			}
			pairTraffic[src][tgt] += len(comm.SymbolsUsed()) + len(comm.Calls())
		}

		// Detect bidirectional tight coupling pairs
		checkedPairs := make(map[string]bool)
		var sortedSources []string
		for src := range pairTraffic {
			sortedSources = append(sortedSources, src)
		}
		sort.Strings(sortedSources)

		for _, src := range sortedSources {
			targets := pairTraffic[src]
			var sortedTargets []string
			for tgt := range targets {
				sortedTargets = append(sortedTargets, tgt)
			}
			sort.Strings(sortedTargets)

			for _, tgt := range sortedTargets {
				count := targets[tgt]
				pairKey := src + "<->" + tgt
				if src > tgt {
					pairKey = tgt + "<->" + src
				}
				if checkedPairs[pairKey] {
					continue
				}
				checkedPairs[pairKey] = true

				reverseCount := 0
				if pairTraffic[tgt] != nil {
					reverseCount = pairTraffic[tgt][src]
				}

				if reverseCount > 0 && (count+reverseCount) > 15 {
					// Significant bidirectional coupling
					finding := NewFinding(
						"dependency",
						RuleTightCoupling,
						CategoryDependency,
						SeverityMedium,
						ConfidenceLikely,
						fmt.Sprintf("Tight bidirectional coupling between %s and %s", src, tgt),
						fmt.Sprintf("High-traffic bidirectional dependency detected between packages %s and %s (%d outbound, %d inbound references).", src, tgt, count, reverseCount),
						"",
						"",
						src,
						"",
						"",
						nil,
						fmt.Sprintf("bidirectional traffic %d <-> %d", count, reverseCount),
						[]string{src, tgt},
						"Decouple bidirectional dependency by introducing an intermediary interface package.",
						"cross_repo_model",
					)
					findings = append(findings, finding)
				}
			}
		}

		// Detect extreme fan-out packages (> 15 outbound package dependencies)
		var sortedOutboundPkgs []string
		for pkg := range outboundCounts {
			sortedOutboundPkgs = append(sortedOutboundPkgs, pkg)
		}
		sort.Strings(sortedOutboundPkgs)

		for _, pkg := range sortedOutboundPkgs {
			count := outboundCounts[pkg]
			if count > 15 && !strings.Contains(pkg, "cmd") && !strings.Contains(pkg, "cli") {
				finding := NewFinding(
					"dependency",
					RuleTightCoupling,
					CategoryDependency,
					SeverityMedium,
					ConfidenceLikely,
					fmt.Sprintf("High fan-out coupling in %s (%d dependencies)", pkg, count),
					fmt.Sprintf("Package %s has excessive fan-out (%d outbound dependencies), indicating broad responsibility.", pkg, count),
					"",
					"",
					pkg,
					"",
					"",
					nil,
					fmt.Sprintf("outbound dependency count %d > threshold 15", count),
					nil,
					"Split package responsibilities to reduce external dependency fan-out.",
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
	return NewAnalysisRuleResult(RuleTightCoupling, status, findings, fmt.Sprintf("tight coupling analysis evaluated %d findings", len(findings)))
}

// 9.5 Orphan Packages
func (a *DependencyAnalyzer) analyzeOrphanPackages() *AnalysisRuleResult {
	var findings []*Finding

	if a.depModel != nil && len(a.depModel.Orphans()) > 0 {
		for _, orphan := range a.depModel.Orphans() {
			// Account for cmd entry points, tests, docs, build tools
			if strings.Contains(orphan, "cmd") || strings.Contains(orphan, "tests") || strings.Contains(orphan, "tools") {
				continue
			}
			finding := NewFinding(
				"dependency",
				RuleOrphanPackages,
				CategoryDependency,
				SeverityLow,
				ConfidenceLikely,
				fmt.Sprintf("Orphan package: %s", orphan),
				fmt.Sprintf("Package %s has no inbound or outbound dependency connections in the repository dependency graph.", orphan),
				"",
				"",
				orphan,
				"",
				"",
				nil,
				fmt.Sprintf("orphan package in dependency model: %s", orphan),
				nil,
				"Verify if package is dead code or intended to be an isolated standalone utility.",
				"dependency_model",
			)
			findings = append(findings, finding)
		}
	} else if a.crossRepoModel != nil {
		allPkgs := make(map[string]bool)
		connectedPkgs := make(map[string]bool)

		for _, comm := range a.crossRepoModel.PackageCommunications() {
			if comm != nil {
				allPkgs[comm.SourcePackage()] = true
				allPkgs[comm.TargetPackage()] = true
				connectedPkgs[comm.SourcePackage()] = true
				connectedPkgs[comm.TargetPackage()] = true
			}
		}

		var orphanList []string
		for pkg := range allPkgs {
			if !connectedPkgs[pkg] && !strings.Contains(pkg, "cmd") && !strings.Contains(pkg, "test") {
				orphanList = append(orphanList, pkg)
			}
		}
		sort.Strings(orphanList)

		for _, o := range orphanList {
			finding := NewFinding(
				"dependency",
				RuleOrphanPackages,
				CategoryDependency,
				SeverityLow,
				ConfidenceLikely,
				fmt.Sprintf("Orphan package: %s", o),
				fmt.Sprintf("Package %s has no active inbound or outbound relationships.", o),
				"",
				"",
				o,
				"",
				"",
				nil,
				"zero active communications in cross-repo model",
				nil,
				"Verify if package is required or should be integrated.",
				"cross_repo_model",
			)
			findings = append(findings, finding)
		}
	}

	status := StatusNoFindings
	if len(findings) > 0 {
		status = StatusFindingsPresent
	}
	return NewAnalysisRuleResult(RuleOrphanPackages, status, findings, fmt.Sprintf("orphan packages analysis evaluated %d findings", len(findings)))
}
