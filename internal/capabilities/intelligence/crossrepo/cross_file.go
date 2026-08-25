package crossrepo

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/intelligence/semantic"
	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
	"github.com/unhield/limoxel/internal/capabilities/repository/xref"
)

// CrossFileAnalyzer performs cross-file relationship, symbol propagation, dependency, and configuration analysis.
type CrossFileAnalyzer struct{}

// NewCrossFileAnalyzer creates a new CrossFileAnalyzer.
func NewCrossFileAnalyzer() *CrossFileAnalyzer {
	return &CrossFileAnalyzer{}
}

// Analyze performs cross-file analysis using repository models.
func (a *CrossFileAnalyzer) Analyze(
	symDB *symbol.SymbolDatabase,
	xrefModel *xref.XRefModel,
	semModel *semantic.SemanticModel,
	knownConfigs []string,
) (
	[]*FileRelationship,
	[]*SymbolPropagation,
	[]*CrossFileDependency,
	[]*SharedConfig,
) {
	var fileRels []*FileRelationship
	var symbolProps []*SymbolPropagation
	var crossFileDeps []*CrossFileDependency
	var sharedConfigs []*SharedConfig

	fileRelMap := make(map[string]*FileRelationship)
	depMap := make(map[string]*CrossFileDependency)
	propMap := make(map[string]*SymbolPropagation)

	// 1. Process Symbols and Symbol Propagations
	if symDB != nil {
		for _, sym := range symDB.AllSymbols() {
			if sym == nil {
				continue
			}
			srcFile := filepath.ToSlash(filepath.Clean(sym.Position().File()))
			pkgPath := filepath.ToSlash(filepath.Clean(sym.PackagePath()))

			// File-to-package membership relationship
			relID := "filerel:" + srcFile + ":" + pkgPath + ":membership"
			if _, exists := fileRelMap[relID]; !exists {
				fileRelMap[relID] = NewFileRelationship(
					relID,
					FileRelPackageMembership,
					srcFile,
					pkgPath,
					"package "+pkgPath,
					"symbol_db",
				)
			}

			// Test source association
			if strings.HasSuffix(srcFile, "_test.go") {
				prodFile := strings.TrimSuffix(srcFile, "_test.go") + ".go"
				testRelID := "filerel:" + srcFile + "->" + prodFile + ":test_source"
				if _, exists := fileRelMap[testRelID]; !exists {
					fileRelMap[testRelID] = NewFileRelationship(
						testRelID,
						FileRelTestSource,
						srcFile,
						prodFile,
						"unit test file for production source",
						"naming_convention",
					)
				}
			}

			// Initialize propagation tracking for exported symbols or symbols with references
			symID := sym.ID()
			propMap[symID] = NewSymbolPropagation(
				symID,
				sym.Name(),
				srcFile,
				srcFile,
				pkgPath,
				nil,
				nil,
				[]string{srcFile},
			)
		}
	}

	// 2. Process Cross-References (XRef)
	if xrefModel != nil && xrefModel.References() != nil {
		for _, ref := range xrefModel.References().AllReferences() {
			if ref == nil {
				continue
			}
			targetSymID := ref.TargetSymbolID()
			refFile := filepath.ToSlash(filepath.Clean(ref.FilePath()))

			// Update SymbolPropagation
			if prop, exists := propMap[targetSymID]; exists {
				if refFile != prop.DeclaringFile() {
					newRefFiles := append(prop.ReferencingFiles(), refFile)
					newPath := append(prop.PropagationPath(), refFile)

					propMap[targetSymID] = NewSymbolPropagation(
						prop.SymbolID(),
						prop.SymbolName(),
						prop.DeclaringFile(),
						prop.DefiningFile(),
						prop.ExportingPackage(),
						newRefFiles,
						prop.ConsumingPackages(),
						newPath,
					)

					// Establish FileRelationship: FileRelReference
					fRelID := "filerel:" + refFile + "->" + prop.DeclaringFile() + ":reference"
					if _, relExists := fileRelMap[fRelID]; !relExists {
						fileRelMap[fRelID] = NewFileRelationship(
							fRelID,
							FileRelReference,
							refFile,
							prop.DeclaringFile(),
							"reference to symbol "+prop.SymbolName(),
							"xref_db",
						)
					}

					// Establish CrossFileDependency
					depID := "filedep:" + refFile + "->" + prop.DeclaringFile()
					if existingDep, depExists := depMap[depID]; depExists {
						syms := append(existingDep.SymbolMediated(), prop.SymbolName())
						depMap[depID] = NewCrossFileDependency(
							refFile,
							prop.DeclaringFile(),
							prop.ExportingPackage(),
							syms,
							true,
						)
					} else {
						depMap[depID] = NewCrossFileDependency(
							refFile,
							prop.DeclaringFile(),
							prop.ExportingPackage(),
							[]string{prop.SymbolName()},
							true,
						)
					}
				}
			}
		}
	}

	// 3. Process Semantic Model Call Graph / Relationships if present
	if semModel != nil {
		for _, sym := range semModel.AllSymbols() {
			if sym == nil {
				continue
			}
			for _, call := range sym.Calls() {
				targetSym := semModel.SymbolByID(call)
				if targetSym != nil && targetSym.FilePath() != "" && targetSym.FilePath() != sym.FilePath() {
					// Add cross-file call relationship
					callRelID := "filerel:" + sym.FilePath() + "->" + targetSym.FilePath() + ":call"
					if _, exists := fileRelMap[callRelID]; !exists {
						fileRelMap[callRelID] = NewFileRelationship(
							callRelID,
							FileRelDependency,
							sym.FilePath(),
							targetSym.FilePath(),
							"function call from "+sym.Name()+" to "+targetSym.Name(),
							"semantic_model",
						)
					}
				}
			}
		}
	}

	// 4. Process Shared Configurations
	var allSourceFiles []string
	var allPackages []string
	srcFileSet := make(map[string]bool)
	pkgSet := make(map[string]bool)

	for _, rel := range fileRelMap {
		if rel.SourceFile() != "" {
			srcFileSet[rel.SourceFile()] = true
		}
		if rel.Kind() == FileRelPackageMembership && rel.TargetFile() != "" {
			pkgSet[rel.TargetFile()] = true
		}
	}
	for sf := range srcFileSet {
		allSourceFiles = append(allSourceFiles, sf)
	}
	sort.Strings(allSourceFiles)
	for p := range pkgSet {
		allPackages = append(allPackages, p)
	}
	sort.Strings(allPackages)

	for _, cfgPath := range knownConfigs {
		cleanCfg := filepath.ToSlash(filepath.Clean(cfgPath))
		ext := strings.ToLower(filepath.Ext(cleanCfg))
		format := strings.TrimPrefix(ext, ".")
		if format == "" {
			format = "config"
		}

		cfgObj := NewSharedConfig(
			cleanCfg,
			format,
			allSourceFiles,
			allPackages,
			nil,
			nil,
		)
		sharedConfigs = append(sharedConfigs, cfgObj)

		// Create FileRelConfigUsage for affected files
		for _, sf := range allSourceFiles {
			cfgRelID := "filerel:" + sf + "->" + cleanCfg + ":config_usage"
			if _, exists := fileRelMap[cfgRelID]; !exists {
				fileRelMap[cfgRelID] = NewFileRelationship(
					cfgRelID,
					FileRelConfigUsage,
					sf,
					cleanCfg,
					"shared configuration file "+filepath.Base(cleanCfg),
					"config_discovery",
				)
			}
		}
	}

	// Convert maps to slices with deterministic sorting
	for _, r := range fileRelMap {
		fileRels = append(fileRels, r)
	}
	sort.Slice(fileRels, func(i, j int) bool { return fileRels[i].ID() < fileRels[j].ID() })

	for _, p := range propMap {
		symbolProps = append(symbolProps, p)
	}
	sort.Slice(symbolProps, func(i, j int) bool { return symbolProps[i].ID() < symbolProps[j].ID() })

	for _, d := range depMap {
		crossFileDeps = append(crossFileDeps, d)
	}
	sort.Slice(crossFileDeps, func(i, j int) bool { return crossFileDeps[i].ID() < crossFileDeps[j].ID() })

	sort.Slice(sharedConfigs, func(i, j int) bool { return sharedConfigs[i].ID() < sharedConfigs[j].ID() })

	return fileRels, symbolProps, crossFileDeps, sharedConfigs
}
