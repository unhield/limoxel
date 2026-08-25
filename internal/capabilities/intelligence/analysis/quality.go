package analysis

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/intelligence/crossrepo"
	"github.com/unhield/limoxel/internal/capabilities/intelligence/semantic"
	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
	"github.com/unhield/limoxel/internal/capabilities/repository/xref"
)

// CodeQualityAnalyzer executes Task 1: Code Quality Analysis.
type CodeQualityAnalyzer struct {
	symbolDB       *symbol.SymbolDatabase
	xrefModel      *xref.XRefModel
	semanticModel  *semantic.SemanticModel
	crossRepoModel *crossrepo.CrossRepoModel
	discResult     *discovery.Result
}

// NewCodeQualityAnalyzer constructs a CodeQualityAnalyzer.
func NewCodeQualityAnalyzer(
	symDB *symbol.SymbolDatabase,
	xrefModel *xref.XRefModel,
	semModel *semantic.SemanticModel,
	crossModel *crossrepo.CrossRepoModel,
	discResult *discovery.Result,
) *CodeQualityAnalyzer {
	return &CodeQualityAnalyzer{
		symbolDB:       symDB,
		xrefModel:      xrefModel,
		semanticModel:  semModel,
		crossRepoModel: crossModel,
		discResult:     discResult,
	}
}

// Analyze executes all code quality rules and returns an AnalyzerResult.
func (a *CodeQualityAnalyzer) Analyze() *AnalyzerResult {
	ruleResults := make(map[RuleID]*AnalysisRuleResult)

	ruleResults[RuleDeadCode] = a.analyzeDeadCode()
	ruleResults[RuleUnusedImports] = a.analyzeUnusedImports()
	ruleResults[RuleUnusedExports] = a.analyzeUnusedExports()
	ruleResults[RuleDuplicateLogic] = a.analyzeDuplicateLogic()
	ruleResults[RuleLargeFiles] = a.analyzeLargeFiles()
	ruleResults[RuleLargeFunctions] = a.analyzeLargeFunctions()

	return NewAnalyzerResult("code_quality", ruleResults)
}

// 8.1 Dead Code
func (a *CodeQualityAnalyzer) analyzeDeadCode() *AnalysisRuleResult {
	if a.symbolDB == nil {
		return NewAnalysisRuleResult(RuleDeadCode, StatusInsufficientEvidence, nil, "symbol database unavailable")
	}

	var findings []*Finding

	// Build set of referenced targets
	referencedTargets := make(map[string]bool)
	if a.xrefModel != nil && a.xrefModel.References() != nil {
		for _, ref := range a.xrefModel.References().AllReferences() {
			if ref != nil {
				referencedTargets[ref.TargetSymbolID()] = true
			}
		}
	}
	if a.xrefModel != nil && a.xrefModel.CallGraph() != nil {
		for _, edge := range a.xrefModel.CallGraph().AllEdges() {
			if edge != nil {
				referencedTargets[edge.CalleeID()] = true
			}
		}
	}

	for _, sym := range a.symbolDB.AllSymbols() {
		if sym == nil {
			continue
		}

		// Filter out entry points, tests, exported public APIs, interface contracts
		if sym.Name() == "main" || sym.Name() == "init" {
			continue
		}
		if strings.HasSuffix(sym.FilePath(), "_test.go") {
			continue
		}
		if sym.IsExported() && !strings.Contains(sym.PackagePath(), "internal") {
			continue // Public API in external package
		}
		if sym.Kind() == symbol.SymbolKindInterface {
			continue // Interface contract
		}

		// Check if symbol is referenced
		if !referencedTargets[sym.ID()] && !referencedTargets[sym.Name()] {
			// Private or internal entity with zero references
			finding := NewFinding(
				"code_quality",
				RuleDeadCode,
				CategoryQuality,
				SeverityMedium,
				ConfidenceDefinite,
				fmt.Sprintf("Dead code: unreferenced symbol %s", sym.Name()),
				fmt.Sprintf("Private or internal symbol %s in package %s has no inbound references or call invocations in the repository.", sym.Name(), sym.PackagePath()),
				"",
				"",
				sym.PackagePath(),
				sym.FilePath(),
				sym.ID(),
				sym.Position(),
				fmt.Sprintf("zero inbound references in xref/callgraph for symbol ID: %s", sym.ID()),
				nil,
				"Remove unused symbol or verify if it is intended to be called via reflection/framework dispatch.",
				"symbol_db+xref_model",
			)
			findings = append(findings, finding)
		}
	}

	status := StatusNoFindings
	if len(findings) > 0 {
		status = StatusFindingsPresent
	}
	return NewAnalysisRuleResult(RuleDeadCode, status, findings, fmt.Sprintf("dead code analysis evaluated %d findings", len(findings)))
}

// 8.2 Unused Imports
func (a *CodeQualityAnalyzer) analyzeUnusedImports() *AnalysisRuleResult {
	if a.xrefModel == nil || a.xrefModel.References() == nil {
		return NewAnalysisRuleResult(RuleUnusedImports, StatusInsufficientEvidence, nil, "cross-reference model unavailable")
	}

	var findings []*Finding

	// Identify package-level import references
	fileImports := make(map[string][]string)
	referencedImports := make(map[string]map[string]bool)

	for _, ref := range a.xrefModel.References().AllReferences() {
		if ref == nil {
			continue
		}
		if strings.Contains(ref.TargetSymbolID(), "/") && !strings.Contains(ref.TargetSymbolID(), ":") {
			file := ref.FilePath()
			pkgTarget := ref.TargetSymbolID()
			if fileImports[file] == nil {
				fileImports[file] = []string{}
			}
			fileImports[file] = append(fileImports[file], pkgTarget)
		} else {
			file := ref.FilePath()
			if referencedImports[file] == nil {
				referencedImports[file] = make(map[string]bool)
			}
			referencedImports[file][ref.TargetSymbolID()] = true
		}
	}

	var sortedFiles []string
	for file := range fileImports {
		sortedFiles = append(sortedFiles, file)
	}
	sort.Strings(sortedFiles)

	for _, file := range sortedFiles {
		imports := fileImports[file]
		sort.Strings(imports)
		for _, imp := range imports {
			// Blank imports (indicated by "_" or explicit side-effect markers) are valid
			if imp == "_" || strings.HasPrefix(imp, "_") {
				continue
			}
			hasUsage := false
			for usedSymbol := range referencedImports[file] {
				if strings.Contains(usedSymbol, imp) {
					hasUsage = true
					break
				}
			}
			if !hasUsage {
				finding := NewFinding(
					"code_quality",
					RuleUnusedImports,
					CategoryQuality,
					SeverityLow,
					ConfidenceLikely,
					fmt.Sprintf("Unused import %s in %s", imp, file),
					fmt.Sprintf("Package import %s in file %s is declared but no symbols from this package are referenced.", imp, file),
					"",
					"",
					"",
					file,
					"",
					nil,
					fmt.Sprintf("import %s declared with 0 symbol references in file %s", imp, file),
					nil,
					"Remove unused import or use blank import `_` if intended for side-effects.",
					"xref_model",
				)
				findings = append(findings, finding)
			}
		}
	}

	status := StatusNoFindings
	if len(findings) > 0 {
		status = StatusFindingsPresent
	}
	return NewAnalysisRuleResult(RuleUnusedImports, status, findings, fmt.Sprintf("unused imports analysis evaluated %d findings", len(findings)))
}

// 8.3 Unused Exports
func (a *CodeQualityAnalyzer) analyzeUnusedExports() *AnalysisRuleResult {
	if a.symbolDB == nil {
		return NewAnalysisRuleResult(RuleUnusedExports, StatusInsufficientEvidence, nil, "symbol database unavailable")
	}

	var findings []*Finding

	referencedTargets := make(map[string]bool)
	if a.xrefModel != nil && a.xrefModel.References() != nil {
		for _, ref := range a.xrefModel.References().AllReferences() {
			if ref != nil {
				referencedTargets[ref.TargetSymbolID()] = true
			}
		}
	}

	for _, sym := range a.symbolDB.AllSymbols() {
		if sym == nil || !sym.IsExported() {
			continue
		}
		// Only check symbols in `internal/...` packages where external consumption is strictly forbidden
		if strings.Contains(sym.PackagePath(), "internal") && !strings.HasSuffix(sym.FilePath(), "_test.go") {
			if !referencedTargets[sym.ID()] && !referencedTargets[sym.Name()] {
				finding := NewFinding(
					"code_quality",
					RuleUnusedExports,
					CategoryQuality,
					SeverityLow,
					ConfidenceLikely,
					fmt.Sprintf("Unused internal export: %s", sym.Name()),
					fmt.Sprintf("Exported symbol %s in internal package %s has no consumers within the workspace.", sym.Name(), sym.PackagePath()),
					"",
					"",
					sym.PackagePath(),
					sym.FilePath(),
					sym.ID(),
					sym.Position(),
					fmt.Sprintf("internal exported symbol %s has 0 references in workspace", sym.ID()),
					nil,
					"Consider unexporting symbol to private visibility if only intended for internal package use.",
					"symbol_db+xref_model",
				)
				findings = append(findings, finding)
			}
		}
	}

	status := StatusNoFindings
	if len(findings) > 0 {
		status = StatusFindingsPresent
	}
	return NewAnalysisRuleResult(RuleUnusedExports, status, findings, fmt.Sprintf("unused exports analysis evaluated %d findings", len(findings)))
}

// 8.4 Duplicate Logic
func (a *CodeQualityAnalyzer) analyzeDuplicateLogic() *AnalysisRuleResult {
	if a.symbolDB == nil {
		return NewAnalysisRuleResult(RuleDuplicateLogic, StatusInsufficientEvidence, nil, "symbol database unavailable")
	}

	var findings []*Finding
	signatures := make(map[string][]*symbol.Symbol)

	for _, sym := range a.symbolDB.AllSymbols() {
		if sym == nil || (sym.Kind() != symbol.SymbolKindFunction && sym.Kind() != symbol.SymbolKindMethod) {
			continue
		}
		// Ignore trivial getters/setters/constructors without signature or trivial body
		sig := strings.TrimSpace(sym.Signature())
		if len(sig) < 25 {
			continue
		}

		// Normalize signature key
		hasher := sha256.New()
		hasher.Write([]byte(sig))
		key := hex.EncodeToString(hasher.Sum(nil))[:16]

		signatures[key] = append(signatures[key], sym)
	}

	var sigKeys []string
	for k := range signatures {
		sigKeys = append(sigKeys, k)
	}
	sort.Strings(sigKeys)

	for _, k := range sigKeys {
		list := signatures[k]
		if len(list) > 1 {
			// Duplicate function signature and structure across distinct functions
			var names []string
			for _, s := range list {
				names = append(names, s.Name()+" ("+s.FilePath()+")")
			}
			f := list[0]
			finding := NewFinding(
				"code_quality",
				RuleDuplicateLogic,
				CategoryQuality,
				SeverityMedium,
				ConfidenceLikely,
				fmt.Sprintf("Duplicate function logic: %s", f.Name()),
				fmt.Sprintf("Identical structural function signatures detected across %d functions: %s", len(list), strings.Join(names, ", ")),
				"",
				"",
				f.PackagePath(),
				f.FilePath(),
				f.ID(),
				f.Position(),
				fmt.Sprintf("identical signature hash shared between %s", strings.Join(names, " and ")),
				nil,
				"Consolidate duplicate logic into a shared helper function.",
				"symbol_db",
			)
			findings = append(findings, finding)
		}
	}

	status := StatusNoFindings
	if len(findings) > 0 {
		status = StatusFindingsPresent
	}
	return NewAnalysisRuleResult(RuleDuplicateLogic, status, findings, fmt.Sprintf("duplicate logic analysis evaluated %d findings", len(findings)))
}

// 8.5 Large Files
func (a *CodeQualityAnalyzer) analyzeLargeFiles() *AnalysisRuleResult {
	var findings []*Finding

	const MaxLines = 800
	const MaxBytes = 40000

	if a.discResult != nil && len(a.discResult.Files()) > 0 {
		for _, f := range a.discResult.Files() {
			if f == nil || f.IsIgnored() || f.IsDir() {
				continue
			}
			if f.Size() > MaxBytes {
				finding := NewFinding(
					"code_quality",
					RuleLargeFiles,
					CategoryQuality,
					SeverityLow,
					ConfidenceDefinite,
					fmt.Sprintf("Large file: %s (%d bytes)", f.RelPath(), f.Size()),
					fmt.Sprintf("File %s exceeds maximum recommended size threshold (%d bytes > %d bytes threshold).", f.RelPath(), f.Size(), MaxBytes),
					"",
					"",
					"",
					f.RelPath(),
					"",
					nil,
					fmt.Sprintf("file size %d bytes > threshold %d", f.Size(), MaxBytes),
					nil,
					"Decompose file into cohesive smaller modules or files.",
					"discovery_result",
				)
				findings = append(findings, finding)
			}
		}
	} else if a.symbolDB != nil {
		// Fallback estimating file length via maximum symbol position
		fileMaxLines := make(map[string]int)
		for _, sym := range a.symbolDB.AllSymbols() {
			if sym != nil && sym.Position() != nil {
				if sym.Position().Line() > fileMaxLines[sym.FilePath()] {
					fileMaxLines[sym.FilePath()] = sym.Position().Line()
				}
			}
		}
		var sortedPaths []string
		for path := range fileMaxLines {
			sortedPaths = append(sortedPaths, path)
		}
		sort.Strings(sortedPaths)

		for _, path := range sortedPaths {
			maxLine := fileMaxLines[path]
			if maxLine > MaxLines {
				finding := NewFinding(
					"code_quality",
					RuleLargeFiles,
					CategoryQuality,
					SeverityLow,
					ConfidenceDefinite,
					fmt.Sprintf("Large file: %s (%d lines)", path, maxLine),
					fmt.Sprintf("File %s exceeds maximum recommended line threshold (%d lines > %d lines threshold).", path, maxLine, MaxLines),
					"",
					"",
					"",
					path,
					"",
					nil,
					fmt.Sprintf("file symbol line %d > threshold %d", maxLine, MaxLines),
					nil,
					"Decompose file into cohesive smaller modules or files.",
					"symbol_db",
				)
				findings = append(findings, finding)
			}
		}
	}

	status := StatusNoFindings
	if len(findings) > 0 {
		status = StatusFindingsPresent
	}
	return NewAnalysisRuleResult(RuleLargeFiles, status, findings, fmt.Sprintf("large files analysis evaluated %d findings", len(findings)))
}

// 8.6 Large Functions
func (a *CodeQualityAnalyzer) analyzeLargeFunctions() *AnalysisRuleResult {
	if a.symbolDB == nil {
		return NewAnalysisRuleResult(RuleLargeFunctions, StatusInsufficientEvidence, nil, "symbol database unavailable")
	}

	var findings []*Finding
	const MaxFunctionLength = 80 // Line threshold

	for _, sym := range a.symbolDB.AllSymbols() {
		if sym == nil || (sym.Kind() != symbol.SymbolKindFunction && sym.Kind() != symbol.SymbolKindMethod) {
			continue
		}
		if sym.Doc() != nil && strings.Contains(sym.Doc().Content(), "@large_ok") {
			continue
		}

		// Calculate approximate line length from signature lines / doc lines
		lineCount := strings.Count(sym.Signature(), "\n") + 1
		if lineCount > MaxFunctionLength {
			finding := NewFinding(
				"code_quality",
				RuleLargeFunctions,
				CategoryQuality,
				SeverityMedium,
				ConfidenceDefinite,
				fmt.Sprintf("Large function: %s (%d lines)", sym.Name(), lineCount),
				fmt.Sprintf("Function %s in %s exceeds recommended size threshold (%d lines > %d lines threshold).", sym.Name(), sym.FilePath(), lineCount, MaxFunctionLength),
				"",
				"",
				sym.PackagePath(),
				sym.FilePath(),
				sym.ID(),
				sym.Position(),
				fmt.Sprintf("function size %d lines > threshold %d", lineCount, MaxFunctionLength),
				nil,
				"Extract sub-functions or refactor helper procedures.",
				"symbol_db",
			)
			findings = append(findings, finding)
		}
	}

	status := StatusNoFindings
	if len(findings) > 0 {
		status = StatusFindingsPresent
	}
	return NewAnalysisRuleResult(RuleLargeFunctions, status, findings, fmt.Sprintf("large functions analysis evaluated %d findings", len(findings)))
}
