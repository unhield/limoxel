package crossrepo

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/intelligence/semantic"
	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
	"github.com/unhield/limoxel/internal/capabilities/repository/xref"
)

// CrossPackageAnalyzer evaluates inter-package communications, contracts, ownership, and APIs.
type CrossPackageAnalyzer struct{}

// NewCrossPackageAnalyzer creates a new CrossPackageAnalyzer.
func NewCrossPackageAnalyzer() *CrossPackageAnalyzer {
	return &CrossPackageAnalyzer{}
}

// Analyze performs cross-package analysis using semantic and repository models.
func (a *CrossPackageAnalyzer) Analyze(
	symDB *symbol.SymbolDatabase,
	xrefModel *xref.XRefModel,
	semModel *semantic.SemanticModel,
) (
	[]*PackageCommunication,
	[]*PackageContract,
	[]*APIEndpoint,
) {
	var pkgComms []*PackageCommunication
	var pkgContracts []*PackageContract
	var apiEndpoints []*APIEndpoint

	commMap := make(map[string]*PackageCommunication)
	contractMap := make(map[string]*PackageContract)
	apiMap := make(map[string]*APIEndpoint)

	// Collect package symbols and types
	pkgSymbols := make(map[string][]string)
	pkgTypes := make(map[string][]string)
	pkgInterfaces := make(map[string][]string)
	pkgDocs := make(map[string]string)

	if symDB != nil {
		for _, sym := range symDB.AllSymbols() {
			if sym == nil {
				continue
			}
			pkg := filepath.ToSlash(filepath.Clean(sym.PackagePath()))
			if sym.IsExported() {
				pkgSymbols[pkg] = append(pkgSymbols[pkg], sym.Name())
				if sym.Kind() == symbol.SymbolKindType || sym.Kind() == symbol.SymbolKindStruct {
					pkgTypes[pkg] = append(pkgTypes[pkg], sym.Name())
				} else if sym.Kind() == symbol.SymbolKindInterface {
					pkgInterfaces[pkg] = append(pkgInterfaces[pkg], sym.Name())
				}

				// Candidate API endpoint
				vis := APIVisibilityPublic
				if strings.Contains(pkg, "/internal/") || strings.HasPrefix(pkg, "internal/") {
					vis = APIVisibilityInternal
				}

				docText := ""
				if sym.Doc() != nil {
					docText = sym.Doc().Content()
				}

				apiMap[sym.ID()] = NewAPIEndpoint(
					sym.ID(),
					sym.Name(),
					pkg,
					nil,
					vis,
					sym.Signature(),
					docText,
				)
			}
		}
	}

	// Process Cross-References (XRef) for package communication
	if xrefModel != nil && xrefModel.References() != nil && symDB != nil {
		for _, ref := range xrefModel.References().AllReferences() {
			if ref == nil {
				continue
			}
			targetSym := symDB.SymbolByID(ref.TargetSymbolID())
			if targetSym == nil {
				continue
			}

			srcFile := filepath.ToSlash(filepath.Clean(ref.FilePath()))
			srcPkg := filepath.ToSlash(filepath.Clean(filepath.Dir(srcFile)))
			tgtPkg := filepath.ToSlash(filepath.Clean(targetSym.PackagePath()))

			if srcPkg != "" && tgtPkg != "" && srcPkg != tgtPkg {
				commID := "pkgcomm:" + srcPkg + "->" + tgtPkg + ":reference"
				if existing, exists := commMap[commID]; exists {
					syms := append(existing.SymbolsUsed(), targetSym.Name())
					commMap[commID] = NewPackageCommunication(
						srcPkg,
						tgtPkg,
						PkgCommTypeUsage,
						syms,
						existing.Calls(),
						"outbound",
					)
				} else {
					commMap[commID] = NewPackageCommunication(
						srcPkg,
						tgtPkg,
						PkgCommTypeUsage,
						[]string{targetSym.Name()},
						nil,
						"outbound",
					)
				}

				// Update API consumer packages
				if api, exists := apiMap[targetSym.ID()]; exists {
					consumers := append(api.ConsumerPackages(), srcPkg)
					apiMap[targetSym.ID()] = NewAPIEndpoint(
						api.SymbolID(),
						api.SymbolName(),
						api.OwningPackage(),
						consumers,
						api.Visibility(),
						api.Signature(),
						api.Doc(),
					)
				}
			}
		}

		// Process Call Graph
		if xrefModel.CallGraph() != nil {
			for _, edge := range xrefModel.CallGraph().AllEdges() {
				if edge == nil {
					continue
				}
				callerSym := symDB.SymbolByID(edge.CallerID())
				calleeSym := symDB.SymbolByID(edge.CalleeID())
				if callerSym != nil && calleeSym != nil {
					cPkg := filepath.ToSlash(filepath.Clean(callerSym.PackagePath()))
					tPkg := filepath.ToSlash(filepath.Clean(calleeSym.PackagePath()))
					if cPkg != "" && tPkg != "" && cPkg != tPkg {
						commID := "pkgcomm:" + cPkg + "->" + tPkg + ":call"
						callDesc := callerSym.Name() + "->" + calleeSym.Name()
						if existing, exists := commMap[commID]; exists {
							calls := append(existing.Calls(), callDesc)
							commMap[commID] = NewPackageCommunication(
								cPkg,
								tPkg,
								PkgCommCall,
								existing.SymbolsUsed(),
								calls,
								"outbound",
							)
						} else {
							commMap[commID] = NewPackageCommunication(
								cPkg,
								tPkg,
								PkgCommCall,
								[]string{calleeSym.Name()},
								[]string{callDesc},
								"outbound",
							)
						}
					}
				}
			}
		}
	}

	// 3. Build Package Contracts
	for pkg, syms := range pkgSymbols {
		types := pkgTypes[pkg]
		ifaces := pkgInterfaces[pkg]
		doc := pkgDocs[pkg]

		contractMap[pkg] = NewPackageContract(
			pkg,
			syms,
			types,
			ifaces,
			doc,
		)
	}

	// Convert maps to slices with deterministic sorting
	for _, c := range commMap {
		pkgComms = append(pkgComms, c)
	}
	sort.Slice(pkgComms, func(i, j int) bool { return pkgComms[i].ID() < pkgComms[j].ID() })

	for _, c := range contractMap {
		pkgContracts = append(pkgContracts, c)
	}
	sort.Slice(pkgContracts, func(i, j int) bool { return pkgContracts[i].ID() < pkgContracts[j].ID() })

	for _, a := range apiMap {
		apiEndpoints = append(apiEndpoints, a)
	}
	sort.Slice(apiEndpoints, func(i, j int) bool { return apiEndpoints[i].ID() < apiEndpoints[j].ID() })

	return pkgComms, pkgContracts, apiEndpoints
}
