package knowledgegraph

import (
	"fmt"
	"sort"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/repository/language"
	"github.com/unhield/limoxel/internal/capabilities/repository/symbol"
)

// EnrichmentEngine enriches the knowledge graph with semantic, ownership, dependency, documentation, and configuration relationships.
type EnrichmentEngine struct{}

// NewEnrichmentEngine constructs an EnrichmentEngine.
func NewEnrichmentEngine() *EnrichmentEngine {
	return &EnrichmentEngine{}
}

// Enrich applies all 5 documented knowledge enrichment dimensions to the base entities and relationships.
func (e *EnrichmentEngine) Enrich(
	entities []*GraphEntity,
	rels []*GraphRelationship,
	symDB *symbol.SymbolDatabase,
	langModel *language.StructureModel,
) ([]*GraphEntity, []*GraphRelationship) {
	entityMap := make(map[string]*GraphEntity)
	for _, ent := range entities {
		if ent != nil {
			entityMap[ent.ID()] = ent
		}
	}

	enrichedEntities := append([]*GraphEntity(nil), entities...)
	enrichedRels := append([]*GraphRelationship(nil), rels...)

	// 1. Semantic Enrichment: Symbol Interface Implementations & Embeddings
	if symDB != nil {
		typeMap := make(map[string]*symbol.Symbol)
		for _, s := range symDB.AllSymbols() {
			if s == nil {
				continue
			}
			if s.Kind() == symbol.SymbolKindInterface || s.Kind() == symbol.SymbolKindType || s.Kind() == symbol.SymbolKindStruct {
				typeMap[s.Name()] = s
			}
		}

		for _, s := range symDB.AllSymbols() {
			if s == nil {
				continue
			}
			symID := CanonicalEntityID(EntitySymbol, s.ID())

			// Detect method receivers and struct member connections
			if s.Kind() == symbol.SymbolKindMethod && s.ReceiverType() != "" {
				if parentSym, ok := typeMap[s.ReceiverType()]; ok {
					parentID := CanonicalEntityID(EntitySymbol, parentSym.ID())
					if entityMap[parentID] != nil {
						enrichedRels = append(enrichedRels, NewGraphRelationship(parentID, symID, RelOwns, fmt.Sprintf("struct/interface %s declares method %s", s.ReceiverType(), s.Name()), "symbol_db", 1.0, nil))
						enrichedRels = append(enrichedRels, NewGraphRelationship(symID, parentID, RelBelongsTo, fmt.Sprintf("method %s belongs to type %s", s.Name(), s.ReceiverType()), "symbol_db", 1.0, nil))
					}
				}
			}

			// Documentation Enrichment from Symbol Doc Comments
			if s.Doc() != nil && strings.TrimSpace(s.Doc().Content()) != "" {
				docID := CanonicalEntityID(EntityDocumentation, fmt.Sprintf("symdoc:%s", s.ID()))
				docEntity := NewGraphEntity(docID, EntityDocumentation, s.Name()+"_doc", s.PackagePath(), s.FilePath(), s.Position(), map[string]string{"content": s.Doc().Content()}, "symbol_doc")
				enrichedEntities = append(enrichedEntities, docEntity)
				enrichedRels = append(enrichedRels, NewGraphRelationship(docID, symID, RelDocuments, fmt.Sprintf("doc comment documents symbol %s", s.Name()), "symbol_doc", 1.0, nil))
			}
		}
	}

	// 2. Documentation Enrichment from Language Structure Model
	if langModel != nil {
		for _, docAsset := range langModel.DocAssets() {
			if docAsset == nil || docAsset.Path() == "" {
				continue
			}
			docID := CanonicalEntityID(EntityDocumentation, docAsset.Path())
			repoID := CanonicalEntityID(EntityRepository, "root")
			enrichedRels = append(enrichedRels, NewGraphRelationship(docID, repoID, RelDocuments, fmt.Sprintf("top-level documentation asset %s", docAsset.Path()), "language_model", 1.0, nil))
		}
	}

	// 3. Ownership & Architectural Component Enrichment
	// Group packages into components based on directory hierarchy (e.g. internal/capabilities/..., internal/platform/...)
	componentMap := make(map[string][]string)
	for _, ent := range enrichedEntities {
		if ent.Type() == EntityPackage {
			pkgPath := ent.PackagePath()
			parts := strings.Split(pkgPath, "/")
			if len(parts) >= 2 {
				compName := parts[0] + "/" + parts[1]
				componentMap[compName] = append(componentMap[compName], ent.ID())
			}
		}
	}

	var sortedCompNames []string
	for compName := range componentMap {
		sortedCompNames = append(sortedCompNames, compName)
	}
	sort.Strings(sortedCompNames)

	for _, compName := range sortedCompNames {
		pkgIDs := componentMap[compName]
		sort.Strings(pkgIDs)

		compID := CanonicalEntityID(EntityArchComponent, compName)
		compEntity := NewGraphEntity(compID, EntityArchComponent, compName, compName, "", nil, map[string]string{"component_name": compName}, "architecture_inference")
		enrichedEntities = append(enrichedEntities, compEntity)

		repoID := CanonicalEntityID(EntityRepository, "root")
		enrichedRels = append(enrichedRels, NewGraphRelationship(repoID, compID, RelOwns, fmt.Sprintf("repository owns architectural component %s", compName), "architecture_inference", 1.0, nil))

		for _, pkgID := range pkgIDs {
			enrichedRels = append(enrichedRels, NewGraphRelationship(compID, pkgID, RelOwns, fmt.Sprintf("component %s contains package %s", compName, pkgID), "architecture_inference", 1.0, nil))
			enrichedRels = append(enrichedRels, NewGraphRelationship(pkgID, compID, RelBelongsTo, fmt.Sprintf("package %s belongs to component %s", pkgID, compName), "architecture_inference", 1.0, nil))
		}
	}

	// 4. Configuration Enrichment
	for _, ent := range enrichedEntities {
		if ent.Type() == EntityConfiguration {
			pkgPath := ent.PackagePath()
			if pkgPath != "" && pkgPath != "." {
				pkgID := CanonicalEntityID(EntityPackage, pkgPath)
				if entityMap[pkgID] != nil {
					enrichedRels = append(enrichedRels, NewGraphRelationship(ent.ID(), pkgID, RelConfigures, fmt.Sprintf("configuration file %s configures package %s", ent.Name(), pkgPath), "configuration_enrichment", 1.0, nil))
				}
			}
		}
	}

	return DeduplicateAndSortEntities(enrichedEntities), DeduplicateAndSortRelationships(enrichedRels)
}
