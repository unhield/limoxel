package knowledgegraph

import (
	"sort"
	"strings"
)

// RepositoryContext represents comprehensive graph-derived repository context.
type RepositoryContext struct {
	RootPath          string         `json:"root_path"`
	RepositoryEntity  *GraphEntity   `json:"repository_entity"`
	Packages          []*GraphEntity `json:"packages"`
	ArchComponents    []*GraphEntity `json:"arch_components"`
	Documentation     []*GraphEntity `json:"documentation"`
	Configuration     []*GraphEntity `json:"configuration"`
	TotalSymbols      int            `json:"total_symbols"`
	TotalFiles        int            `json:"total_files"`
	DependencySummary map[string]int `json:"dependency_summary"`
}

// PackageContext represents graph context for a specific package.
type PackageContext struct {
	PackageEntity     *GraphEntity   `json:"package_entity"`
	ContainedFiles    []*GraphEntity `json:"contained_files"`
	DeclaredSymbols   []*GraphEntity `json:"declared_symbols"`
	OutboundDeps      []*GraphEntity `json:"outbound_deps"`
	InboundDependents []*GraphEntity `json:"inbound_dependents"`
	ParentComponent   *GraphEntity   `json:"parent_component,omitempty"`
	AttachedDoc       []*GraphEntity `json:"attached_doc,omitempty"`
	AttachedConfig    []*GraphEntity `json:"attached_config,omitempty"`
}

// ModuleContext represents graph context for a specific module or subsystem.
type ModuleContext struct {
	ModuleName        string         `json:"module_name"`
	ContainedFiles    []*GraphEntity `json:"contained_files"`
	ContainedSymbols  []*GraphEntity `json:"contained_symbols"`
	OutboundDeps      []*GraphEntity `json:"outbound_deps"`
	InboundDependents []*GraphEntity `json:"inbound_dependents"`
}

// SymbolContext represents graph context for an individual symbol.
type SymbolContext struct {
	SymbolEntity  *GraphEntity   `json:"symbol_entity"`
	DeclaredFile  *GraphEntity   `json:"declared_file,omitempty"`
	ParentPackage *GraphEntity   `json:"parent_package,omitempty"`
	ParentType    *GraphEntity   `json:"parent_type,omitempty"`
	Callers       []*GraphEntity `json:"callers"`
	Callees       []*GraphEntity `json:"callees"`
	AttachedDoc   []*GraphEntity `json:"attached_doc,omitempty"`
}

// ArchitectureContext represents repository-wide architectural context.
type ArchitectureContext struct {
	Components         []*GraphEntity       `json:"components"`
	CrossCompEdges     []*GraphRelationship `json:"cross_component_edges"`
	LayerRelationships map[string][]string  `json:"layer_relationships"`
}

// ContextGenerator generates structured engineering context from the knowledge graph model.
type ContextGenerator struct{}

// NewContextGenerator constructs a ContextGenerator.
func NewContextGenerator() *ContextGenerator {
	return &ContextGenerator{}
}

// GenerateRepositoryContext constructs the repository-level context.
func (g *ContextGenerator) GenerateRepositoryContext(model *KnowledgeGraphModel) *RepositoryContext {
	if model == nil {
		return nil
	}

	repoEnt := model.EntityByID(CanonicalEntityID(EntityRepository, "root"))
	pkgs := model.EntitiesByType(EntityPackage)
	comps := model.EntitiesByType(EntityArchComponent)
	docs := model.EntitiesByType(EntityDocumentation)
	confs := model.EntitiesByType(EntityConfiguration)
	syms := model.EntitiesByType(EntitySymbol)
	files := model.EntitiesByType(EntityFile)

	depSummary := make(map[string]int)
	for _, rel := range model.Relationships() {
		if rel.Kind() == RelDependsOn || rel.Kind() == RelImports {
			depSummary[string(rel.Kind())]++
		}
	}

	return &RepositoryContext{
		RootPath:          model.RootPath(),
		RepositoryEntity:  repoEnt,
		Packages:          pkgs,
		ArchComponents:    comps,
		Documentation:     docs,
		Configuration:     confs,
		TotalSymbols:      len(syms),
		TotalFiles:        len(files),
		DependencySummary: depSummary,
	}
}

// GeneratePackageContext constructs the package-level context.
func (g *ContextGenerator) GeneratePackageContext(model *KnowledgeGraphModel, packagePath string) *PackageContext {
	if model == nil || packagePath == "" {
		return nil
	}

	pkgID := CanonicalEntityID(EntityPackage, packagePath)
	pkgEnt := model.EntityByID(pkgID)
	if pkgEnt == nil {
		return nil
	}

	var containedFiles []*GraphEntity
	var declaredSymbols []*GraphEntity
	var outboundDeps []*GraphEntity
	var inboundDependents []*GraphEntity
	var attachedDoc []*GraphEntity
	var attachedConfig []*GraphEntity
	var parentComp *GraphEntity

	for _, rel := range model.OutboundRelationships(pkgID) {
		tgt := model.EntityByID(rel.TargetID())
		if tgt == nil {
			continue
		}
		switch rel.Kind() {
		case RelOwns:
			if tgt.Type() == EntityFile {
				containedFiles = append(containedFiles, tgt)
			}
		case RelDependsOn, RelImports:
			if tgt.Type() == EntityPackage {
				outboundDeps = append(outboundDeps, tgt)
			}
		case RelBelongsTo:
			if tgt.Type() == EntityArchComponent {
				parentComp = tgt
			}
		}
	}

	for _, rel := range model.InboundRelationships(pkgID) {
		src := model.EntityByID(rel.SourceID())
		if src == nil {
			continue
		}
		switch rel.Kind() {
		case RelDependsOn, RelImports:
			if src.Type() == EntityPackage {
				inboundDependents = append(inboundDependents, src)
			}
		case RelDocuments:
			attachedDoc = append(attachedDoc, src)
		case RelConfigures:
			attachedConfig = append(attachedConfig, src)
		}
	}

	// Gather symbols in package
	for _, sym := range model.EntitiesByType(EntitySymbol) {
		if sym.PackagePath() == packagePath {
			declaredSymbols = append(declaredSymbols, sym)
		}
	}

	return &PackageContext{
		PackageEntity:     pkgEnt,
		ContainedFiles:    DeduplicateAndSortEntities(containedFiles),
		DeclaredSymbols:   DeduplicateAndSortEntities(declaredSymbols),
		OutboundDeps:      DeduplicateAndSortEntities(outboundDeps),
		InboundDependents: DeduplicateAndSortEntities(inboundDependents),
		ParentComponent:   parentComp,
		AttachedDoc:       DeduplicateAndSortEntities(attachedDoc),
		AttachedConfig:    DeduplicateAndSortEntities(attachedConfig),
	}
}

// GenerateModuleContext constructs module-level context.
func (g *ContextGenerator) GenerateModuleContext(model *KnowledgeGraphModel, moduleName string) *ModuleContext {
	if model == nil || moduleName == "" {
		return nil
	}

	var containedFiles []*GraphEntity
	var containedSymbols []*GraphEntity
	var outboundDeps []*GraphEntity
	var inboundDependents []*GraphEntity

	for _, f := range model.EntitiesByType(EntityFile) {
		if strings.HasPrefix(f.PackagePath(), moduleName) || strings.HasPrefix(f.FilePath(), moduleName) {
			containedFiles = append(containedFiles, f)
		}
	}

	for _, s := range model.EntitiesByType(EntitySymbol) {
		if strings.HasPrefix(s.PackagePath(), moduleName) || strings.HasPrefix(s.FilePath(), moduleName) {
			containedSymbols = append(containedSymbols, s)
		}
	}

	for _, p := range model.EntitiesByType(EntityPackage) {
		if strings.HasPrefix(p.PackagePath(), moduleName) {
			for _, r := range model.OutboundRelationships(p.ID()) {
				if (r.Kind() == RelDependsOn || r.Kind() == RelImports) && !strings.HasPrefix(r.TargetID(), "package:"+moduleName) {
					tgt := model.EntityByID(r.TargetID())
					if tgt != nil {
						outboundDeps = append(outboundDeps, tgt)
					}
				}
			}
			for _, r := range model.InboundRelationships(p.ID()) {
				if (r.Kind() == RelDependsOn || r.Kind() == RelImports) && !strings.HasPrefix(r.SourceID(), "package:"+moduleName) {
					src := model.EntityByID(r.SourceID())
					if src != nil {
						inboundDependents = append(inboundDependents, src)
					}
				}
			}
		}
	}

	return &ModuleContext{
		ModuleName:        moduleName,
		ContainedFiles:    DeduplicateAndSortEntities(containedFiles),
		ContainedSymbols:  DeduplicateAndSortEntities(containedSymbols),
		OutboundDeps:      DeduplicateAndSortEntities(outboundDeps),
		InboundDependents: DeduplicateAndSortEntities(inboundDependents),
	}
}

// GenerateSymbolContext constructs symbol-level context.
func (g *ContextGenerator) GenerateSymbolContext(model *KnowledgeGraphModel, symbolID string) *SymbolContext {
	if model == nil || symbolID == "" {
		return nil
	}

	fullID := CanonicalEntityID(EntitySymbol, symbolID)
	symEnt := model.EntityByID(fullID)
	if symEnt == nil {
		symEnt = model.EntityByID(symbolID)
		if symEnt == nil {
			return nil
		}
		fullID = symEnt.ID()
	}

	var callers []*GraphEntity
	var callees []*GraphEntity
	var attachedDoc []*GraphEntity
	var declFile *GraphEntity
	var parentPkg *GraphEntity
	var parentType *GraphEntity

	for _, rel := range model.InboundRelationships(fullID) {
		src := model.EntityByID(rel.SourceID())
		if src == nil {
			continue
		}
		switch rel.Kind() {
		case RelCalls:
			callers = append(callers, src)
		case RelDocuments:
			attachedDoc = append(attachedDoc, src)
		case RelOwns:
			if src.Type() == EntityFile {
				declFile = src
			} else if src.Type() == EntitySymbol {
				parentType = src
			}
		}
	}

	for _, rel := range model.OutboundRelationships(fullID) {
		tgt := model.EntityByID(rel.TargetID())
		if tgt == nil {
			continue
		}
		switch rel.Kind() {
		case RelCalls:
			callees = append(callees, tgt)
		case RelBelongsTo:
			if tgt.Type() == EntityPackage {
				parentPkg = tgt
			}
		}
	}

	if parentPkg == nil && symEnt.PackagePath() != "" {
		parentPkg = model.EntityByID(CanonicalEntityID(EntityPackage, symEnt.PackagePath()))
	}
	if declFile == nil && symEnt.FilePath() != "" {
		declFile = model.EntityByID(CanonicalEntityID(EntityFile, symEnt.FilePath()))
	}

	return &SymbolContext{
		SymbolEntity:  symEnt,
		DeclaredFile:  declFile,
		ParentPackage: parentPkg,
		ParentType:    parentType,
		Callers:       DeduplicateAndSortEntities(callers),
		Callees:       DeduplicateAndSortEntities(callees),
		AttachedDoc:   DeduplicateAndSortEntities(attachedDoc),
	}
}

// GenerateArchitectureContext constructs the architecture-level context.
func (g *ContextGenerator) GenerateArchitectureContext(model *KnowledgeGraphModel) *ArchitectureContext {
	if model == nil {
		return nil
	}

	comps := model.EntitiesByType(EntityArchComponent)
	var crossEdges []*GraphRelationship
	layerRels := make(map[string][]string)

	for _, rel := range model.Relationships() {
		src := model.EntityByID(rel.SourceID())
		tgt := model.EntityByID(rel.TargetID())
		if src != nil && tgt != nil && src.Type() == EntityPackage && tgt.Type() == EntityPackage {
			// Find parent components
			srcParent := ""
			tgtParent := ""
			for _, r := range model.OutboundRelationships(src.ID()) {
				if r.Kind() == RelBelongsTo && model.EntityByID(r.TargetID()) != nil && model.EntityByID(r.TargetID()).Type() == EntityArchComponent {
					srcParent = r.TargetID()
					break
				}
			}
			for _, r := range model.OutboundRelationships(tgt.ID()) {
				if r.Kind() == RelBelongsTo && model.EntityByID(r.TargetID()) != nil && model.EntityByID(r.TargetID()).Type() == EntityArchComponent {
					tgtParent = r.TargetID()
					break
				}
			}

			if srcParent != "" && tgtParent != "" && srcParent != tgtParent {
				crossEdges = append(crossEdges, rel)
				layerRels[srcParent] = append(layerRels[srcParent], tgtParent)
			}
		}
	}

	// Sort layer map values
	for k := range layerRels {
		sort.Strings(layerRels[k])
	}

	return &ArchitectureContext{
		Components:         comps,
		CrossCompEdges:     DeduplicateAndSortRelationships(crossEdges),
		LayerRelationships: layerRels,
	}
}
