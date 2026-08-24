package indexing

import (
	"encoding/json"
	"fmt"

	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
)

type serializedFile struct {
	ID               string           `json:"id"`
	RelPath          string           `json:"rel_path"`
	FileType         FileType         `json:"file_type"`
	LanguageID       string           `json:"language_id"`
	IsTest           bool             `json:"is_test"`
	GenerationStatus GenerationStatus `json:"generation_status"`
	SizeBytes        int64            `json:"size_bytes"`
	ContentHash      string           `json:"content_hash"`
	Encoding         EncodingType     `json:"encoding"`
	LineEnding       LineEndingType   `json:"line_ending"`
	LineCount        int              `json:"line_count"`
	BlankLineCount   int              `json:"blank_line_count"`
	CommentLineCount int              `json:"comment_line_count"`
}

type serializedPackage struct {
	Name       string        `json:"name"`
	Path       string        `json:"path"`
	ModulePath string        `json:"module_path"`
	Files      []string      `json:"files"`
	Imports    []string      `json:"imports"`
	Exports    []string      `json:"exports"`
	Doc        string        `json:"doc"`
	Ownership  string        `json:"ownership"`
	Stats      *PackageStats `json:"stats"`
}

type serializedRelationship struct {
	SourceID string           `json:"source_id"`
	TargetID string           `json:"target_id"`
	Type     RelationshipType `json:"type"`
	Evidence string           `json:"evidence"`
}

type serializedConfigStats struct {
	TotalFiles int      `json:"total_files"`
	Types      []string `json:"types"`
}

type serializedStats struct {
	TotalFiles             int                    `json:"total_files"`
	TotalPackages          int                    `json:"total_packages"`
	TotalModules           int                    `json:"total_modules"`
	TotalLines             int                    `json:"total_lines"`
	CodeLines              int                    `json:"code_lines"`
	CommentLines           int                    `json:"comment_lines"`
	BlankLines             int                    `json:"blank_lines"`
	LanguageDistribution   map[string]int         `json:"language_distribution"`
	FileTypeDistribution   map[FileType]int       `json:"file_type_distribution"`
	DocumentationCoverage  float64                `json:"documentation_coverage"`
	StructuralTestCoverage float64                `json:"structural_test_coverage"`
	ConfigStats            *serializedConfigStats `json:"config_stats"`
}

type serializedIndex struct {
	SchemaVersion  string                    `json:"schema_version"`
	RepositoryRoot string                    `json:"repository_root"`
	Files          []*serializedFile         `json:"files"`
	Packages       []*serializedPackage      `json:"packages"`
	Relationships  []*serializedRelationship `json:"relationships"`
	Stats          *serializedStats          `json:"stats"`
}

// Serialize encodes an IndexModel into a deterministic JSON representation.
func Serialize(model *IndexModel) ([]byte, error) {
	if model == nil {
		return nil, ErrNilIndexModel
	}

	var sFiles []*serializedFile
	for _, f := range model.Files() {
		sFiles = append(sFiles, &serializedFile{
			ID:               f.ID(),
			RelPath:          f.RelPath(),
			FileType:         f.FileType(),
			LanguageID:       f.LanguageID(),
			IsTest:           f.IsTest(),
			GenerationStatus: f.GenerationStatus(),
			SizeBytes:        f.SizeBytes(),
			ContentHash:      f.ContentHash(),
			Encoding:         f.Encoding(),
			LineEnding:       f.LineEnding(),
			LineCount:        f.LineCount(),
			BlankLineCount:   f.BlankLineCount(),
			CommentLineCount: f.CommentLineCount(),
		})
	}

	var sPackages []*serializedPackage
	for _, p := range model.Packages() {
		sPackages = append(sPackages, &serializedPackage{
			Name:       p.Name(),
			Path:       p.Path(),
			ModulePath: p.ModulePath(),
			Files:      p.Files(),
			Imports:    p.Imports(),
			Exports:    p.Exports(),
			Doc:        p.Doc(),
			Ownership:  p.Ownership(),
			Stats:      p.Stats(),
		})
	}

	var sRelationships []*serializedRelationship
	for _, r := range model.Relationships() {
		sRelationships = append(sRelationships, &serializedRelationship{
			SourceID: r.SourceID(),
			TargetID: r.TargetID(),
			Type:     r.Type(),
			Evidence: r.Evidence(),
		})
	}

	var sStats *serializedStats
	if st := model.Stats(); st != nil {
		var sCfg *serializedConfigStats
		if cfg := st.ConfigStats(); cfg != nil {
			sCfg = &serializedConfigStats{
				TotalFiles: cfg.TotalFiles(),
				Types:      cfg.Types(),
			}
		}

		sStats = &serializedStats{
			TotalFiles:             st.TotalFiles(),
			TotalPackages:          st.TotalPackages(),
			TotalModules:           st.TotalModules(),
			TotalLines:             st.TotalLines(),
			CodeLines:              st.CodeLines(),
			CommentLines:           st.CommentLines(),
			BlankLines:             st.BlankLines(),
			LanguageDistribution:   st.LanguageDistribution(),
			FileTypeDistribution:   st.FileTypeDistribution(),
			DocumentationCoverage:  st.DocumentationCoverage(),
			StructuralTestCoverage: st.StructuralTestCoverage(),
			ConfigStats:            sCfg,
		}
	}

	rootPayload := &serializedIndex{
		SchemaVersion:  model.SchemaVersion(),
		RepositoryRoot: model.RepositoryRoot(),
		Files:          sFiles,
		Packages:       sPackages,
		Relationships:  sRelationships,
		Stats:          sStats,
	}

	return json.MarshalIndent(rootPayload, "", "  ")
}

// Deserialize decodes JSON data into an immutable IndexModel validating schema version compatibility.
func Deserialize(data []byte) (*IndexModel, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("%w: empty payload", ErrCorruptedIndex)
	}

	var raw serializedIndex
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCorruptedIndex, err)
	}

	if raw.SchemaVersion != CurrentSchemaVersion {
		return nil, fmt.Errorf("%w: expected %s, got %s", ErrIncompatibleSchema, CurrentSchemaVersion, raw.SchemaVersion)
	}

	var files []*IndexedFile
	for _, f := range raw.Files {
		files = append(files, NewIndexedFile(
			f.ID,
			f.RelPath,
			f.FileType,
			f.LanguageID,
			f.IsTest,
			f.GenerationStatus,
			f.SizeBytes,
			f.ContentHash,
			f.Encoding,
			f.LineEnding,
			f.LineCount,
			f.BlankLineCount,
			f.CommentLineCount,
		))
	}

	var packages []*IndexedPackage
	for _, p := range raw.Packages {
		packages = append(packages, NewIndexedPackage(
			p.Name,
			p.Path,
			p.ModulePath,
			p.Files,
			p.Imports,
			p.Exports,
			p.Doc,
			p.Ownership,
			p.Stats,
		))
	}

	var relationships []*FileRelationship
	for _, r := range raw.Relationships {
		relationships = append(relationships, NewFileRelationship(
			r.SourceID,
			r.TargetID,
			r.Type,
			r.Evidence,
		))
	}

	var stats *RepositoryStats
	if s := raw.Stats; s != nil {
		var cfgStats *ConfigStats
		if s.ConfigStats != nil {
			cfgStats = NewConfigStats(s.ConfigStats.TotalFiles, s.ConfigStats.Types)
		}

		stats = NewRepositoryStats(
			s.TotalFiles,
			s.TotalPackages,
			s.TotalModules,
			s.TotalLines,
			s.CodeLines,
			s.CommentLines,
			s.BlankLines,
			s.LanguageDistribution,
			s.FileTypeDistribution,
			s.DocumentationCoverage,
			s.StructuralTestCoverage,
			cfgStats,
		)
	}

	return NewIndexModel(
		raw.SchemaVersion,
		raw.RepositoryRoot,
		files,
		packages,
		relationships,
		stats,
		[]*discovery.Diagnostic{},
	), nil
}
