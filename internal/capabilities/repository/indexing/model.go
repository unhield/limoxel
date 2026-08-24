package indexing

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
)

const (
	// CurrentSchemaVersion defines the authoritative schema version of the Source Code Index.
	CurrentSchemaVersion = "1.0.0"
)

// IndexModel represents the consolidated, immutable result of repository source code indexing.
type IndexModel struct {
	schemaVersion  string
	repositoryRoot string
	files          []*IndexedFile
	fileMap        map[string]*IndexedFile
	packages       []*IndexedPackage
	packageMap     map[string]*IndexedPackage
	relationships  []*FileRelationship
	stats          *RepositoryStats
	diagnostics    []*discovery.Diagnostic
}

// NewIndexModel constructs an immutable IndexModel with indexed lookup tables.
func NewIndexModel(
	schemaVersion string,
	repositoryRoot string,
	files []*IndexedFile,
	packages []*IndexedPackage,
	relationships []*FileRelationship,
	stats *RepositoryStats,
	diagnostics []*discovery.Diagnostic,
) *IndexModel {
	v := strings.TrimSpace(schemaVersion)
	if v == "" {
		v = CurrentSchemaVersion
	}

	// 1. Sort files deterministically by RelPath
	fileList := make([]*IndexedFile, len(files))
	copy(fileList, files)
	sort.Slice(fileList, func(i, j int) bool {
		return fileList[i].relPath < fileList[j].relPath
	})
	fMap := make(map[string]*IndexedFile, len(fileList))
	for _, f := range fileList {
		fMap[f.relPath] = f
	}

	// 2. Sort packages deterministically by Path
	pkgList := make([]*IndexedPackage, len(packages))
	copy(pkgList, packages)
	sort.Slice(pkgList, func(i, j int) bool {
		return pkgList[i].path < pkgList[j].path
	})
	pMap := make(map[string]*IndexedPackage, len(pkgList))
	for _, p := range pkgList {
		pMap[p.path] = p
	}

	// 3. Sort relationships deterministically by SourceID -> TargetID -> RelType
	relList := make([]*FileRelationship, len(relationships))
	copy(relList, relationships)
	sort.Slice(relList, func(i, j int) bool {
		if relList[i].sourceID != relList[j].sourceID {
			return relList[i].sourceID < relList[j].sourceID
		}
		if relList[i].targetID != relList[j].targetID {
			return relList[i].targetID < relList[j].targetID
		}
		return relList[i].relType < relList[j].relType
	})

	// 4. Defensive copy for diagnostics
	diagList := make([]*discovery.Diagnostic, len(diagnostics))
	copy(diagList, diagnostics)

	return &IndexModel{
		schemaVersion:  v,
		repositoryRoot: filepath.ToSlash(filepath.Clean(repositoryRoot)),
		files:          fileList,
		fileMap:        fMap,
		packages:       pkgList,
		packageMap:     pMap,
		relationships:  relList,
		stats:          stats,
		diagnostics:    diagList,
	}
}

// SchemaVersion returns the schema version of the index model.
func (im *IndexModel) SchemaVersion() string {
	if im == nil {
		return ""
	}
	return im.schemaVersion
}

// RepositoryRoot returns the canonical repository root path.
func (im *IndexModel) RepositoryRoot() string {
	if im == nil {
		return ""
	}
	return im.repositoryRoot
}

// Files returns a defensive copy of all indexed files in deterministic order.
func (im *IndexModel) Files() []*IndexedFile {
	if im == nil || len(im.files) == 0 {
		return nil
	}
	cloned := make([]*IndexedFile, len(im.files))
	copy(cloned, im.files)
	return cloned
}

// Packages returns a defensive copy of all indexed packages in deterministic order.
func (im *IndexModel) Packages() []*IndexedPackage {
	if im == nil || len(im.packages) == 0 {
		return nil
	}
	cloned := make([]*IndexedPackage, len(im.packages))
	copy(cloned, im.packages)
	return cloned
}

// Relationships returns a defensive copy of all relationships in deterministic order.
func (im *IndexModel) Relationships() []*FileRelationship {
	if im == nil || len(im.relationships) == 0 {
		return nil
	}
	cloned := make([]*FileRelationship, len(im.relationships))
	copy(cloned, im.relationships)
	return cloned
}

// Stats returns repository aggregate statistics.
func (im *IndexModel) Stats() *RepositoryStats {
	if im == nil {
		return nil
	}
	return im.stats
}

// Diagnostics returns a defensive copy of diagnostics recorded during indexing.
func (im *IndexModel) Diagnostics() []*discovery.Diagnostic {
	if im == nil || len(im.diagnostics) == 0 {
		return nil
	}
	cloned := make([]*discovery.Diagnostic, len(im.diagnostics))
	copy(cloned, im.diagnostics)
	return cloned
}

// FileByPath provides $O(1)$ lookup for an indexed file by repository-relative path.
func (im *IndexModel) FileByPath(relPath string) *IndexedFile {
	if im == nil || len(im.fileMap) == 0 {
		return nil
	}
	clean := filepath.ToSlash(filepath.Clean(relPath))
	return im.fileMap[clean]
}

// PackageByPath provides $O(1)$ lookup for an indexed package by repository-relative path.
func (im *IndexModel) PackageByPath(pkgPath string) *IndexedPackage {
	if im == nil || len(im.packageMap) == 0 {
		return nil
	}
	clean := filepath.ToSlash(filepath.Clean(pkgPath))
	return im.packageMap[clean]
}

// FilesForPackage returns all indexed files belonging to the given package path.
func (im *IndexModel) FilesForPackage(pkgPath string) []*IndexedFile {
	if im == nil {
		return nil
	}
	pkg := im.PackageByPath(pkgPath)
	if pkg == nil {
		return nil
	}
	var res []*IndexedFile
	for _, fPath := range pkg.Files() {
		if f := im.FileByPath(fPath); f != nil {
			res = append(res, f)
		}
	}
	return res
}

// RelationshipsForSource returns all relationships originating from the specified source ID.
func (im *IndexModel) RelationshipsForSource(sourceID string) []*FileRelationship {
	if im == nil || len(im.relationships) == 0 {
		return nil
	}
	cleanSource := strings.TrimSpace(sourceID)
	var res []*FileRelationship
	for _, rel := range im.relationships {
		if rel.sourceID == cleanSource {
			res = append(res, rel)
		}
	}
	return res
}

// RelationshipsForTarget returns all relationships pointing to the specified target ID.
func (im *IndexModel) RelationshipsForTarget(targetID string) []*FileRelationship {
	if im == nil || len(im.relationships) == 0 {
		return nil
	}
	cleanTarget := strings.TrimSpace(targetID)
	var res []*FileRelationship
	for _, rel := range im.relationships {
		if rel.targetID == cleanTarget {
			res = append(res, rel)
		}
	}
	return res
}

// String returns a human-readable summary of the IndexModel.
func (im *IndexModel) String() string {
	if im == nil {
		return ""
	}
	return fmt.Sprintf("IndexModel<root=%s, schema=%s, files=%d, packages=%d, relationships=%d>",
		im.repositoryRoot, im.schemaVersion, len(im.files), len(im.packages), len(im.relationships))
}
