package knowledgegraph

import (
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/intelligence/crossrepo"
	"github.com/unhield/limoxel/internal/capabilities/intelligence/semantic"
	"github.com/unhield/limoxel/internal/capabilities/repository/dependency"
	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
	"github.com/unhield/limoxel/internal/capabilities/repository/language"
	"github.com/unhield/limoxel/internal/capabilities/repository/metadata"
	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
	"github.com/unhield/limoxel/internal/capabilities/repository/xref"
)

// GraphBuildParams bundles repository and intelligence models for knowledge graph construction.
type GraphBuildParams struct {
	RootPath        string
	DiscoveryResult *discovery.Result
	SymbolDB        *symbol.SymbolDatabase
	XRefModel       *xref.XRefModel
	DependencyModel *dependency.DependencyModel
	LanguageModel   *language.StructureModel
	MetadataProfile *metadata.Profile
	SemanticModel   *semantic.SemanticModel
	CrossRepoModel  *crossrepo.CrossRepoModel
}

// KnowledgeGraphBuilder converts repository capabilities into an initial typed knowledge graph.
type KnowledgeGraphBuilder struct{}

// NewKnowledgeGraphBuilder constructs a KnowledgeGraphBuilder.
func NewKnowledgeGraphBuilder() *KnowledgeGraphBuilder {
	return &KnowledgeGraphBuilder{}
}

// BuildBaseGraph constructs the primary entity and relationship sets from repository models.
func (b *KnowledgeGraphBuilder) BuildBaseGraph(params GraphBuildParams) ([]*GraphEntity, []*GraphRelationship) {
	var entities []*GraphEntity
	var rels []*GraphRelationship

	rootID := CanonicalEntityID(EntityRepository, "root")
	repoAttrs := map[string]string{"path": params.RootPath}
	if params.MetadataProfile != nil {
		repoAttrs["name"] = params.MetadataProfile.Name()
		repoAttrs["root"] = params.MetadataProfile.Root()
	}
	repoEntity := NewGraphEntity(rootID, EntityRepository, filepath.Base(params.RootPath), "", "", nil, repoAttrs, "metadata_profile")
	entities = append(entities, repoEntity)

	// 1. Ingest Discovery Result -> Files & Packages
	packageSet := make(map[string]bool)
	if params.DiscoveryResult != nil {
		for _, f := range params.DiscoveryResult.Files() {
			if f == nil || f.IsIgnored() || f.IsDir() {
				continue
			}
			baseName := path.Base(f.RelPath())
			fileID := CanonicalEntityID(EntityFile, f.RelPath())
			fileAttrs := map[string]string{
				"rel_path":  f.RelPath(),
				"size":      fmt.Sprintf("%d", f.Size()),
				"extension": f.Extension(),
			}
			fileEntity := NewGraphEntity(fileID, EntityFile, baseName, filepath.ToSlash(filepath.Dir(f.RelPath())), f.RelPath(), nil, fileAttrs, "discovery_result")
			entities = append(entities, fileEntity)

			pkgPath := filepath.ToSlash(filepath.Dir(f.RelPath()))
			if pkgPath != "" && pkgPath != "." {
				packageSet[pkgPath] = true
				// Package owns File
				pkgID := CanonicalEntityID(EntityPackage, pkgPath)
				rels = append(rels, NewGraphRelationship(pkgID, fileID, RelOwns, fmt.Sprintf("file %s in directory %s", f.RelPath(), pkgPath), "discovery_result", 1.0, nil))
				rels = append(rels, NewGraphRelationship(fileID, pkgID, RelBelongsTo, fmt.Sprintf("file belongs to package %s", pkgPath), "discovery_result", 1.0, nil))
			} else {
				// Root repository owns file
				rels = append(rels, NewGraphRelationship(rootID, fileID, RelOwns, "top-level file in repository root", "discovery_result", 1.0, nil))
				rels = append(rels, NewGraphRelationship(fileID, rootID, RelBelongsTo, "file in root repository", "discovery_result", 1.0, nil))
			}

			// Configuration & Documentation Entity Detection
			lowerPath := strings.ToLower(f.RelPath())
			if strings.HasSuffix(lowerPath, ".md") || strings.Contains(lowerPath, "doc") {
				docID := CanonicalEntityID(EntityDocumentation, f.RelPath())
				docEntity := NewGraphEntity(docID, EntityDocumentation, baseName, pkgPath, f.RelPath(), nil, fileAttrs, "discovery_result")
				entities = append(entities, docEntity)
				rels = append(rels, NewGraphRelationship(docID, rootID, RelDocuments, fmt.Sprintf("documentation asset %s", f.RelPath()), "discovery_result", 1.0, nil))
			}
			if strings.HasSuffix(lowerPath, ".yaml") || strings.HasSuffix(lowerPath, ".json") || strings.HasSuffix(lowerPath, ".toml") || strings.Contains(lowerPath, "config") {
				confID := CanonicalEntityID(EntityConfiguration, f.RelPath())
				confEntity := NewGraphEntity(confID, EntityConfiguration, baseName, pkgPath, f.RelPath(), nil, fileAttrs, "discovery_result")
				entities = append(entities, confEntity)
				rels = append(rels, NewGraphRelationship(confID, rootID, RelConfigures, fmt.Sprintf("configuration asset %s", f.RelPath()), "discovery_result", 1.0, nil))
			}
		}
	}

	// 2. Ingest Symbols -> Symbol Entities & Declarations
	if params.SymbolDB != nil {
		for _, sym := range params.SymbolDB.AllSymbols() {
			if sym == nil {
				continue
			}
			symID := CanonicalEntityID(EntitySymbol, sym.ID())
			symAttrs := map[string]string{
				"kind":        string(sym.Kind()),
				"exported":    fmt.Sprintf("%t", sym.IsExported()),
				"signature":   sym.Signature(),
				"symbol_name": sym.Name(),
			}
			symEntity := NewGraphEntity(symID, EntitySymbol, sym.Name(), sym.PackagePath(), sym.FilePath(), sym.Position(), symAttrs, "symbol_db")
			entities = append(entities, symEntity)

			// File declares Symbol
			fileID := CanonicalEntityID(EntityFile, sym.FilePath())
			rels = append(rels, NewGraphRelationship(fileID, symID, RelOwns, fmt.Sprintf("file %s declares symbol %s", sym.FilePath(), sym.Name()), "symbol_db", 1.0, nil))
			rels = append(rels, NewGraphRelationship(symID, fileID, RelBelongsTo, fmt.Sprintf("symbol %s belongs to file %s", sym.Name(), sym.FilePath()), "symbol_db", 1.0, nil))

			if sym.PackagePath() != "" {
				packageSet[sym.PackagePath()] = true
			}
		}
	}

	// Register all Package Entities
	var sortedPkgs []string
	for p := range packageSet {
		sortedPkgs = append(sortedPkgs, p)
	}
	sort.Strings(sortedPkgs)

	for _, p := range sortedPkgs {
		pkgID := CanonicalEntityID(EntityPackage, p)
		pkgEntity := NewGraphEntity(pkgID, EntityPackage, filepath.Base(p), p, "", nil, map[string]string{"pkg_path": p}, "symbol_db+discovery")
		entities = append(entities, pkgEntity)

		// Repository owns Package
		rels = append(rels, NewGraphRelationship(rootID, pkgID, RelOwns, fmt.Sprintf("repository contains package %s", p), "structure", 1.0, nil))
		rels = append(rels, NewGraphRelationship(pkgID, rootID, RelBelongsTo, fmt.Sprintf("package %s belongs to repository", p), "structure", 1.0, nil))
	}

	// 3. Ingest XRef References -> Symbol Calls and Imports
	if params.XRefModel != nil && params.XRefModel.References() != nil {
		for _, ref := range params.XRefModel.References().AllReferences() {
			if ref == nil || ref.TargetSymbolID() == "" {
				continue
			}
			sourceSymID := CanonicalEntityID(EntitySymbol, ref.SourceSymbolID())
			targetSymID := CanonicalEntityID(EntitySymbol, ref.TargetSymbolID())

			switch ref.Kind() {
			case xref.RefFunction, xref.RefMethod:
				rels = append(rels, NewGraphRelationship(sourceSymID, targetSymID, RelCalls, fmt.Sprintf("reference call in %s", ref.FilePath()), "xref_model", 1.0, nil))
			case xref.RefInterface:
				rels = append(rels, NewGraphRelationship(sourceSymID, targetSymID, RelImplements, fmt.Sprintf("interface implementation in %s", ref.FilePath()), "xref_model", 1.0, nil))
			case xref.RefType:
				rels = append(rels, NewGraphRelationship(sourceSymID, targetSymID, RelSemantic, fmt.Sprintf("type reference in %s", ref.FilePath()), "xref_model", 1.0, nil))
			default:
				rels = append(rels, NewGraphRelationship(sourceSymID, targetSymID, RelDependsOn, fmt.Sprintf("reference of kind %s in %s", ref.Kind(), ref.FilePath()), "xref_model", 1.0, nil))
			}
		}
	}

	// 4. Ingest Dependency Graph -> Package-Level Dependencies
	if params.DependencyModel != nil && params.DependencyModel.Graph() != nil {
		for _, edge := range params.DependencyModel.Graph().Edges() {
			if edge == nil {
				continue
			}
			srcPkgID := CanonicalEntityID(EntityPackage, edge.SourceID())
			tgtPkgID := CanonicalEntityID(EntityPackage, edge.TargetID())
			rels = append(rels, NewGraphRelationship(srcPkgID, tgtPkgID, RelDependsOn, fmt.Sprintf("package dependency edge %s -> %s", edge.SourceID(), edge.TargetID()), "dependency_model", 1.0, nil))
			rels = append(rels, NewGraphRelationship(srcPkgID, tgtPkgID, RelImports, fmt.Sprintf("package import %s -> %s", edge.SourceID(), edge.TargetID()), "dependency_model", 1.0, nil))
		}
	}

	// 5. Ingest Cross-Repository Communications -> Cross-Module Boundaries
	if params.CrossRepoModel != nil {
		for _, comm := range params.CrossRepoModel.PackageCommunications() {
			if comm == nil {
				continue
			}
			srcPkgID := CanonicalEntityID(EntityPackage, comm.SourcePackage())
			tgtPkgID := CanonicalEntityID(EntityPackage, comm.TargetPackage())
			rels = append(rels, NewGraphRelationship(srcPkgID, tgtPkgID, RelSemantic, fmt.Sprintf("cross-package communication %s -> %s (%s)", comm.SourcePackage(), comm.TargetPackage(), comm.Kind().String()), "cross_repo_model", 1.0, nil))
		}
	}

	return entities, rels
}
