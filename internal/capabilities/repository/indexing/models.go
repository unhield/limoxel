package indexing

import (
	"path/filepath"
	"sort"
	"strings"
)

// IndexedFile represents the immutable, deterministic index record of an individual file.
type IndexedFile struct {
	id               string
	relPath          string
	fileType         FileType
	languageID       string
	isTest           bool
	generationStatus GenerationStatus
	sizeBytes        int64
	contentHash      string
	encoding         EncodingType
	lineEnding       LineEndingType
	lineCount        int
	blankLineCount   int
	commentLineCount int
}

// NewIndexedFile constructs an immutable IndexedFile record.
func NewIndexedFile(
	id string,
	relPath string,
	fileType FileType,
	languageID string,
	isTest bool,
	genStatus GenerationStatus,
	sizeBytes int64,
	contentHash string,
	encoding EncodingType,
	lineEnding LineEndingType,
	lineCount int,
	blankLineCount int,
	commentLineCount int,
) *IndexedFile {
	cleanRel := filepath.ToSlash(filepath.Clean(relPath))
	cleanID := strings.TrimSpace(id)
	if cleanID == "" {
		cleanID = cleanRel
	}

	return &IndexedFile{
		id:               cleanID,
		relPath:          cleanRel,
		fileType:         fileType,
		languageID:       strings.ToLower(strings.TrimSpace(languageID)),
		isTest:           isTest,
		generationStatus: genStatus,
		sizeBytes:        sizeBytes,
		contentHash:      strings.TrimSpace(contentHash),
		encoding:         encoding,
		lineEnding:       lineEnding,
		lineCount:        lineCount,
		blankLineCount:   blankLineCount,
		commentLineCount: commentLineCount,
	}
}

// ID returns the stable identifier of the indexed file.
func (f *IndexedFile) ID() string {
	if f == nil {
		return ""
	}
	return f.id
}

// RelPath returns the canonical repository-relative path.
func (f *IndexedFile) RelPath() string {
	if f == nil {
		return ""
	}
	return f.relPath
}

// FileType returns the structural classification of the file.
func (f *IndexedFile) FileType() FileType {
	if f == nil {
		return FileTypeUnknown
	}
	return f.fileType
}

// LanguageID returns the recognized language identifier or empty string.
func (f *IndexedFile) LanguageID() string {
	if f == nil {
		return ""
	}
	return f.languageID
}

// IsTest reports whether the file is classified as a test file.
func (f *IndexedFile) IsTest() bool {
	if f == nil {
		return false
	}
	return f.isTest
}

// GenerationStatus returns the machine-generation classification.
func (f *IndexedFile) GenerationStatus() GenerationStatus {
	if f == nil {
		return GenerationStatusUnknown
	}
	return f.generationStatus
}

// SizeBytes returns the file size in bytes.
func (f *IndexedFile) SizeBytes() int64 {
	if f == nil {
		return 0
	}
	return f.sizeBytes
}

// ContentHash returns the cryptographic SHA-256 content hash of the file.
func (f *IndexedFile) ContentHash() string {
	if f == nil {
		return ""
	}
	return f.contentHash
}

// Encoding returns the detected character encoding.
func (f *IndexedFile) Encoding() EncodingType {
	if f == nil {
		return EncodingUnknown
	}
	return f.encoding
}

// LineEnding returns the detected line-ending format.
func (f *IndexedFile) LineEnding() LineEndingType {
	if f == nil {
		return LineEndingUnknown
	}
	return f.lineEnding
}

// LineCount returns total number of lines in the file.
func (f *IndexedFile) LineCount() int {
	if f == nil {
		return 0
	}
	return f.lineCount
}

// BlankLineCount returns count of blank/whitespace-only lines.
func (f *IndexedFile) BlankLineCount() int {
	if f == nil {
		return 0
	}
	return f.blankLineCount
}

// CommentLineCount returns count of comment lines.
func (f *IndexedFile) CommentLineCount() int {
	if f == nil {
		return 0
	}
	return f.commentLineCount
}

// CodeLineCount returns count of structural code lines (total - blank - comment).
func (f *IndexedFile) CodeLineCount() int {
	if f == nil {
		return 0
	}
	c := f.lineCount - f.blankLineCount - f.commentLineCount
	if c < 0 {
		return 0
	}
	return c
}

// PackageStats holds deterministic structural measurements for a package.
type PackageStats struct {
	sourceFiles    int
	testFiles      int
	generatedFiles int
	totalLines     int
	sizeBytes      int64
}

// NewPackageStats constructs an immutable PackageStats record.
func NewPackageStats(sourceFiles, testFiles, generatedFiles, totalLines int, sizeBytes int64) *PackageStats {
	return &PackageStats{
		sourceFiles:    sourceFiles,
		testFiles:      testFiles,
		generatedFiles: generatedFiles,
		totalLines:     totalLines,
		sizeBytes:      sizeBytes,
	}
}

// SourceFiles returns the count of non-test source files in the package.
func (ps *PackageStats) SourceFiles() int {
	if ps == nil {
		return 0
	}
	return ps.sourceFiles
}

// TestFiles returns the count of test files in the package.
func (ps *PackageStats) TestFiles() int {
	if ps == nil {
		return 0
	}
	return ps.testFiles
}

// GeneratedFiles returns the count of generated files in the package.
func (ps *PackageStats) GeneratedFiles() int {
	if ps == nil {
		return 0
	}
	return ps.generatedFiles
}

// TotalLines returns the total line count across all package files.
func (ps *PackageStats) TotalLines() int {
	if ps == nil {
		return 0
	}
	return ps.totalLines
}

// SizeBytes returns the total size in bytes of package files.
func (ps *PackageStats) SizeBytes() int64 {
	if ps == nil {
		return 0
	}
	return ps.sizeBytes
}

// IndexedPackage represents the immutable index record for a discovered package.
type IndexedPackage struct {
	name       string
	path       string
	modulePath string
	files      []string
	imports    []string
	exports    []string
	doc        string
	ownership  string
	stats      *PackageStats
}

// NewIndexedPackage constructs an immutable IndexedPackage with deterministic sorting.
func NewIndexedPackage(
	name string,
	path string,
	modulePath string,
	files []string,
	imports []string,
	exports []string,
	doc string,
	ownership string,
	stats *PackageStats,
) *IndexedPackage {
	cleanFiles := make([]string, len(files))
	for i, f := range files {
		cleanFiles[i] = filepath.ToSlash(filepath.Clean(f))
	}
	sort.Strings(cleanFiles)

	cleanImports := make([]string, len(imports))
	copy(cleanImports, imports)
	sort.Strings(cleanImports)

	cleanExports := make([]string, len(exports))
	copy(cleanExports, exports)
	sort.Strings(cleanExports)

	return &IndexedPackage{
		name:       strings.TrimSpace(name),
		path:       filepath.ToSlash(filepath.Clean(path)),
		modulePath: filepath.ToSlash(filepath.Clean(modulePath)),
		files:      cleanFiles,
		imports:    cleanImports,
		exports:    cleanExports,
		doc:        strings.TrimSpace(doc),
		ownership:  strings.TrimSpace(ownership),
		stats:      stats,
	}
}

// Name returns the authoritative package name.
func (p *IndexedPackage) Name() string {
	if p == nil {
		return ""
	}
	return p.name
}

// Path returns the repository-relative directory path of the package.
func (p *IndexedPackage) Path() string {
	if p == nil {
		return ""
	}
	return p.path
}

// ModulePath returns the repository-relative path of the enclosing module.
func (p *IndexedPackage) ModulePath() string {
	if p == nil {
		return ""
	}
	return p.modulePath
}

// Files returns a defensive copy of relative file paths in this package.
func (p *IndexedPackage) Files() []string {
	if p == nil || len(p.files) == 0 {
		return nil
	}
	cloned := make([]string, len(p.files))
	copy(cloned, p.files)
	return cloned
}

// Imports returns a defensive copy of imports declared within the package.
func (p *IndexedPackage) Imports() []string {
	if p == nil || len(p.imports) == 0 {
		return nil
	}
	cloned := make([]string, len(p.imports))
	copy(cloned, p.imports)
	return cloned
}

// Exports returns a defensive copy of exported names where deterministically known.
func (p *IndexedPackage) Exports() []string {
	if p == nil || len(p.exports) == 0 {
		return nil
	}
	cloned := make([]string, len(p.exports))
	copy(cloned, p.exports)
	return cloned
}

// Doc returns the package-level documentation comment.
func (p *IndexedPackage) Doc() string {
	if p == nil {
		return ""
	}
	return p.doc
}

// Ownership returns the repository-defined package owner if known.
func (p *IndexedPackage) Ownership() string {
	if p == nil {
		return ""
	}
	return p.ownership
}

// Stats returns structural statistics for the package.
func (p *IndexedPackage) Stats() *PackageStats {
	if p == nil {
		return nil
	}
	return p.stats
}

// FileRelationship represents a directed relationship between repository artifacts.
type FileRelationship struct {
	sourceID string
	targetID string
	relType  RelationshipType
	evidence string
}

// NewFileRelationship constructs an immutable FileRelationship record.
func NewFileRelationship(sourceID, targetID string, relType RelationshipType, evidence string) *FileRelationship {
	return &FileRelationship{
		sourceID: strings.TrimSpace(sourceID),
		targetID: strings.TrimSpace(targetID),
		relType:  relType,
		evidence: strings.TrimSpace(evidence),
	}
}

// SourceID returns the identifier of the source entity.
func (fr *FileRelationship) SourceID() string {
	if fr == nil {
		return ""
	}
	return fr.sourceID
}

// TargetID returns the identifier of the target entity.
func (fr *FileRelationship) TargetID() string {
	if fr == nil {
		return ""
	}
	return fr.targetID
}

// Type returns the relationship type classification.
func (fr *FileRelationship) Type() RelationshipType {
	if fr == nil {
		return RelUnknown
	}
	return fr.relType
}

// Evidence returns the deterministic rationale or structural evidence for this relationship.
func (fr *FileRelationship) Evidence() string {
	if fr == nil {
		return ""
	}
	return fr.evidence
}

// ConfigStats holds aggregate statistics about repository configuration files.
type ConfigStats struct {
	totalFiles int
	types      []string
}

// NewConfigStats creates an immutable ConfigStats record.
func NewConfigStats(totalFiles int, types []string) *ConfigStats {
	typeList := make([]string, len(types))
	copy(typeList, types)
	sort.Strings(typeList)

	return &ConfigStats{
		totalFiles: totalFiles,
		types:      typeList,
	}
}

// TotalFiles returns the count of configuration files.
func (cs *ConfigStats) TotalFiles() int {
	if cs == nil {
		return 0
	}
	return cs.totalFiles
}

// Types returns a defensive copy of detected configuration types.
func (cs *ConfigStats) Types() []string {
	if cs == nil || len(cs.types) == 0 {
		return nil
	}
	cloned := make([]string, len(cs.types))
	copy(cloned, cs.types)
	return cloned
}

// RepositoryStats represents deterministic aggregate structural measurements of the repository.
type RepositoryStats struct {
	totalFiles             int
	totalPackages          int
	totalModules           int
	totalLines             int
	codeLines              int
	commentLines           int
	blankLines             int
	languageDistribution   map[string]int
	fileTypeDistribution   map[FileType]int
	documentationCoverage  float64
	structuralTestCoverage float64
	configStats            *ConfigStats
}

// NewRepositoryStats constructs an immutable RepositoryStats record with defensive copies.
func NewRepositoryStats(
	totalFiles, totalPackages, totalModules, totalLines, codeLines, commentLines, blankLines int,
	langDist map[string]int,
	fileTypeDist map[FileType]int,
	docCov, testCov float64,
	cfgStats *ConfigStats,
) *RepositoryStats {
	langCopy := make(map[string]int, len(langDist))
	for k, v := range langDist {
		langCopy[k] = v
	}

	typeCopy := make(map[FileType]int, len(fileTypeDist))
	for k, v := range fileTypeDist {
		typeCopy[k] = v
	}

	return &RepositoryStats{
		totalFiles:             totalFiles,
		totalPackages:          totalPackages,
		totalModules:           totalModules,
		totalLines:             totalLines,
		codeLines:              codeLines,
		commentLines:           commentLines,
		blankLines:             blankLines,
		languageDistribution:   langCopy,
		fileTypeDistribution:   typeCopy,
		documentationCoverage:  docCov,
		structuralTestCoverage: testCov,
		configStats:            cfgStats,
	}
}

// TotalFiles returns total indexed file count.
func (rs *RepositoryStats) TotalFiles() int {
	if rs == nil {
		return 0
	}
	return rs.totalFiles
}

// TotalPackages returns total indexed package count.
func (rs *RepositoryStats) TotalPackages() int {
	if rs == nil {
		return 0
	}
	return rs.totalPackages
}

// TotalModules returns total module count.
func (rs *RepositoryStats) TotalModules() int {
	if rs == nil {
		return 0
	}
	return rs.totalModules
}

// TotalLines returns total line count across all indexed files.
func (rs *RepositoryStats) TotalLines() int {
	if rs == nil {
		return 0
	}
	return rs.totalLines
}

// CodeLines returns total code line count.
func (rs *RepositoryStats) CodeLines() int {
	if rs == nil {
		return 0
	}
	return rs.codeLines
}

// CommentLines returns total comment line count.
func (rs *RepositoryStats) CommentLines() int {
	if rs == nil {
		return 0
	}
	return rs.commentLines
}

// BlankLines returns total blank line count.
func (rs *RepositoryStats) BlankLines() int {
	if rs == nil {
		return 0
	}
	return rs.blankLines
}

// LanguageDistribution returns a defensive copy of language distribution map.
func (rs *RepositoryStats) LanguageDistribution() map[string]int {
	if rs == nil || len(rs.languageDistribution) == 0 {
		return nil
	}
	cloned := make(map[string]int, len(rs.languageDistribution))
	for k, v := range rs.languageDistribution {
		cloned[k] = v
	}
	return cloned
}

// FileTypeDistribution returns a defensive copy of file type distribution map.
func (rs *RepositoryStats) FileTypeDistribution() map[FileType]int {
	if rs == nil || len(rs.fileTypeDistribution) == 0 {
		return nil
	}
	cloned := make(map[FileType]int, len(rs.fileTypeDistribution))
	for k, v := range rs.fileTypeDistribution {
		cloned[k] = v
	}
	return cloned
}

// DocumentationCoverage returns the structural documentation coverage ratio (0.0 to 1.0).
func (rs *RepositoryStats) DocumentationCoverage() float64 {
	if rs == nil {
		return 0
	}
	return rs.documentationCoverage
}

// StructuralTestCoverage returns the structural test coverage ratio (0.0 to 1.0).
func (rs *RepositoryStats) StructuralTestCoverage() float64 {
	if rs == nil {
		return 0
	}
	return rs.structuralTestCoverage
}

// ConfigStats returns aggregate configuration statistics.
func (rs *RepositoryStats) ConfigStats() *ConfigStats {
	if rs == nil {
		return nil
	}
	return rs.configStats
}
