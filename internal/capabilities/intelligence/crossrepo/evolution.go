package crossrepo

import (
	"fmt"
	"sort"
	"time"
)

// EvolutionAnalyzer analyzes historical evolution, structural changes, and growth metrics.
type EvolutionAnalyzer struct{}

// NewEvolutionAnalyzer creates a new EvolutionAnalyzer.
func NewEvolutionAnalyzer() *EvolutionAnalyzer {
	return &EvolutionAnalyzer{}
}

// Analyze processes historical commits and current repository state to produce an EvolutionModel.
func (a *EvolutionAnalyzer) Analyze(
	repoID string,
	commits []*CommitEvent,
	currentFiles int,
	currentPackages int,
	currentModules int,
	currentSymbols int,
	currentDeps int,
) *EvolutionModel {
	var structural []*StructuralEvolution
	var arch []*ArchitectureEvolution
	var depEvolution []*DependencyEvolution
	var growth []*GrowthMetrics

	// Sort commits chronologically
	sortedCommits := make([]*CommitEvent, len(commits))
	copy(sortedCommits, commits)
	sort.Slice(sortedCommits, func(i, j int) bool {
		return sortedCommits[i].Timestamp().Before(sortedCommits[j].Timestamp())
	})

	runningFileCount := 0
	runningPkgCount := 1

	for idx, c := range sortedCommits {
		added := len(c.FilesAdded())
		removed := len(c.FilesDeleted())
		moved := 0 // Detected if file added matches file deleted in same commit

		runningFileCount += (added - removed)
		if runningFileCount < 0 {
			runningFileCount = 0
		}

		// Structural evolution point
		sID := fmt.Sprintf("struct_evo:%d", idx)
		structural = append(structural, NewStructuralEvolution(
			sID,
			c.Timestamp(),
			added,
			removed,
			moved,
			runningPkgCount,
			1,
		))

		// Architecture evolution point
		if added > 5 || removed > 5 {
			aID := fmt.Sprintf("arch_evo:%d", idx)
			arch = append(arch, NewArchitectureEvolution(
				aID,
				c.Timestamp(),
				[]string{"structural_expansion"},
				[]string{"internal_refactoring"},
				[]string{c.Message()},
			))
		}

		// Dependency evolution point
		dID := fmt.Sprintf("dep_evo:%d", idx)
		depEvolution = append(depEvolution, NewDependencyEvolution(
			dID,
			c.Timestamp(),
			nil,
			nil,
			nil,
		))
	}

	// Calculate overall growth metrics
	now := time.Now().UTC()
	var fileGrowthRate float64
	var pkgGrowthRate float64

	if len(sortedCommits) > 1 {
		fileGrowthRate = float64(currentFiles) / float64(len(sortedCommits))
		pkgGrowthRate = float64(currentPackages) / float64(len(sortedCommits))
	}

	growth = append(growth, NewGrowthMetrics(
		"growth:summary",
		now,
		currentFiles,
		currentPackages,
		currentModules,
		currentSymbols,
		currentDeps,
		fileGrowthRate,
		pkgGrowthRate,
	))

	return NewEvolutionModel(
		repoID,
		sortedCommits,
		structural,
		arch,
		depEvolution,
		growth,
	)
}
