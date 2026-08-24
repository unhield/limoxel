package indexing

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
	"github.com/unhield/limoxel/internal/repository"
)

// Indexer coordinates deterministic repository source code indexing.
type Indexer struct {
	discoverer *discovery.Discoverer
}

// New constructs and validates an immutable Indexer instance.
func New(discoverer *discovery.Discoverer) (*Indexer, error) {
	if discoverer == nil {
		return nil, ErrNilDiscoverer
	}
	return &Indexer{
		discoverer: discoverer,
	}, nil
}

// Discoverer returns the underlying repository Discoverer.
func (idx *Indexer) Discoverer() *discovery.Discoverer {
	if idx == nil {
		return nil
	}
	return idx.discoverer
}

// Index compiles a complete IndexModel from an existing discovery result.
func (idx *Indexer) Index(discResult *discovery.Result) (*IndexModel, error) {
	return idx.IndexIncremental(discResult, nil)
}

// IndexIncremental performs incremental indexing reusing unchanged file records from previous IndexModel.
func (idx *Indexer) IndexIncremental(discResult *discovery.Result, previous *IndexModel) (*IndexModel, error) {
	if idx == nil {
		return nil, ErrNilIndexer
	}
	if discResult == nil {
		return nil, ErrNilDiscoveryResult
	}

	root := discResult.Root()
	discFiles := discResult.Files()
	diagnostics := discResult.Diagnostics()

	// Check if previous cache is valid for incremental reuse
	canReuseCache := previous != nil &&
		previous.SchemaVersion() == CurrentSchemaVersion &&
		previous.RepositoryRoot() == root

	var (
		indexedFiles      []*IndexedFile
		pkgFileMap        = make(map[string][]*IndexedFile)
		pkgNameMap        = make(map[string]string)
		pkgDocMap         = make(map[string]string)
		pkgImportMap      = make(map[string]map[string]struct{})
		pkgExportMap      = make(map[string]map[string]struct{})
		relationships     []*FileRelationship
		seenRelationships = make(map[string]struct{})
		langDist          = make(map[string]int)
		fileTypeDist      = make(map[FileType]int)
		totalLines        int
		codeLines         int
		commentLines      int
		blankLines        int
		configFiles       []string
		configTypesMap    = make(map[string]struct{})
		packagesWithDocs  int
		packagesWithTests int
	)

	// 1. Process or reuse indexed file records
	for _, df := range discFiles {
		relPath := df.RelPath()
		absPath := df.AbsPath()
		langID := ""
		if df.Language() != nil {
			langID = df.Language().ID()
		}

		var fileRec *IndexedFile
		hash, size, hashErr := computeContentHash(absPath)
		if hashErr != nil {
			diagnostics = append(diagnostics, discovery.NewDiagnostic(
				discovery.SeverityWarning,
				"FILE_HASH_ERROR",
				hashErr.Error(),
				relPath,
				false,
			))
			continue
		}

		if canReuseCache {
			if prevF := previous.FileByPath(relPath); prevF != nil && prevF.ContentHash() == hash && prevF.SizeBytes() == size {
				fileRec = prevF
			}
		}

		if fileRec == nil {
			encoding, lineEnding, totalL, blankL, commentL, inspErr := inspectFileContent(absPath)
			if inspErr != nil {
				diagnostics = append(diagnostics, discovery.NewDiagnostic(
					discovery.SeverityWarning,
					"FILE_INSPECT_ERROR",
					inspErr.Error(),
					relPath,
					false,
				))
				continue
			}

			isTest := strings.HasSuffix(relPath, "_test.go") ||
				strings.HasSuffix(relPath, ".test.js") ||
				strings.HasSuffix(relPath, ".test.ts") ||
				strings.HasSuffix(relPath, ".spec.ts") ||
				strings.HasPrefix(filepath.Base(relPath), "test_")

			fType := classifyFileType(relPath, langID, isTest, encoding)
			genStatus := detectGenerationStatus(absPath)

			fileRec = NewIndexedFile(
				relPath,
				relPath,
				fType,
				langID,
				isTest,
				genStatus,
				size,
				hash,
				encoding,
				lineEnding,
				totalL,
				blankL,
				commentL,
			)
		}

		indexedFiles = append(indexedFiles, fileRec)

		// Aggregate repository stats
		if fileRec.LanguageID() != "" {
			langDist[fileRec.LanguageID()]++
		}
		fileTypeDist[fileRec.FileType()]++
		totalLines += fileRec.LineCount()
		blankLines += fileRec.BlankLineCount()
		commentLines += fileRec.CommentLineCount()
		codeLines += fileRec.CodeLineCount()

		if fileRec.FileType() == FileTypeConfig {
			configFiles = append(configFiles, relPath)
			ext := strings.TrimPrefix(filepath.Ext(relPath), ".")
			if ext != "" {
				configTypesMap[ext] = struct{}{}
			}
		}

		// Group by package directory
		pkgDir := filepath.ToSlash(filepath.Dir(relPath))
		if pkgDir == "" {
			pkgDir = "."
		}
		pkgFileMap[pkgDir] = append(pkgFileMap[pkgDir], fileRec)

		// Extract Go package metadata if applicable
		if langID == "go" {
			pName, doc, imps, exps, _ := inspectGoSourceFile(absPath)
			if pName != "" {
				pkgNameMap[pkgDir] = pName
			}
			if doc != "" && pkgDocMap[pkgDir] == "" {
				pkgDocMap[pkgDir] = doc
			}
			if len(imps) > 0 {
				if pkgImportMap[pkgDir] == nil {
					pkgImportMap[pkgDir] = make(map[string]struct{})
				}
				for _, imp := range imps {
					pkgImportMap[pkgDir][imp] = struct{}{}

					// Record import relationship
					addRel := func(src, tgt string, rType RelationshipType, ev string) {
						k := src + "->" + tgt + ":" + string(rType)
						if _, seen := seenRelationships[k]; !seen && src != tgt {
							seenRelationships[k] = struct{}{}
							relationships = append(relationships, NewFileRelationship(src, tgt, rType, ev))
						}
					}
					addRel(relPath, imp, RelImport, "go_import_declaration")
				}
			}
			if len(exps) > 0 {
				if pkgExportMap[pkgDir] == nil {
					pkgExportMap[pkgDir] = make(map[string]struct{})
				}
				for _, exp := range exps {
					pkgExportMap[pkgDir][exp] = struct{}{}
				}
			}
		}
	}

	// 2. Build Indexed Packages
	var indexedPackages []*IndexedPackage
	for pkgDir, pFiles := range pkgFileMap {
		var (
			fPaths    []string
			srcCount  int
			testCount int
			genCount  int
			linesTot  int
			sizeTot   int64
		)

		for _, f := range pFiles {
			fPaths = append(fPaths, f.RelPath())
			linesTot += f.LineCount()
			sizeTot += f.SizeBytes()

			if f.IsTest() {
				testCount++
			} else {
				srcCount++
			}
			if f.GenerationStatus() == GenerationStatusGenerated {
				genCount++
			}

			// Add package membership relationship
			k := f.RelPath() + "->" + pkgDir + ":" + string(RelPackageMembership)
			if _, seen := seenRelationships[k]; !seen {
				seenRelationships[k] = struct{}{}
				relationships = append(relationships, NewFileRelationship(f.RelPath(), pkgDir, RelPackageMembership, "file_in_package_directory"))
			}

			// Add parent-child relationship
			k2 := pkgDir + "->" + f.RelPath() + ":" + string(RelParentChild)
			if _, seen := seenRelationships[k2]; !seen {
				seenRelationships[k2] = struct{}{}
				relationships = append(relationships, NewFileRelationship(pkgDir, f.RelPath(), RelParentChild, "directory_containment"))
			}

			// Check test-to-source mapping
			if f.IsTest() {
				base := strings.TrimSuffix(filepath.Base(f.RelPath()), "_test.go")
				srcCandidate := filepath.ToSlash(filepath.Join(pkgDir, base+".go"))
				for _, sf := range pFiles {
					if sf.RelPath() == srcCandidate && !sf.IsTest() {
						k3 := f.RelPath() + "->" + sf.RelPath() + ":" + string(RelTestToSource)
						if _, seen := seenRelationships[k3]; !seen {
							seenRelationships[k3] = struct{}{}
							relationships = append(relationships, NewFileRelationship(f.RelPath(), sf.RelPath(), RelTestToSource, "go_test_source_pair"))
						}
						break
					}
				}
			}

			// Check doc-to-module mapping
			if f.FileType() == FileTypeDoc {
				k4 := f.RelPath() + "->" + pkgDir + ":" + string(RelDocToModule)
				if _, seen := seenRelationships[k4]; !seen {
					seenRelationships[k4] = struct{}{}
					relationships = append(relationships, NewFileRelationship(f.RelPath(), pkgDir, RelDocToModule, "module_documentation"))
				}
			}

			// Check config-to-source mapping
			if f.FileType() == FileTypeConfig {
				k5 := f.RelPath() + "->" + pkgDir + ":" + string(RelConfigToSource)
				if _, seen := seenRelationships[k5]; !seen {
					seenRelationships[k5] = struct{}{}
					relationships = append(relationships, NewFileRelationship(f.RelPath(), pkgDir, RelConfigToSource, "module_configuration"))
				}
			}
		}

		pName := pkgNameMap[pkgDir]
		if pName == "" {
			pName = filepath.Base(pkgDir)
		}
		pDoc := pkgDocMap[pkgDir]
		if pDoc != "" {
			packagesWithDocs++
		}
		if testCount > 0 {
			packagesWithTests++
		}

		var pImports []string
		for imp := range pkgImportMap[pkgDir] {
			pImports = append(pImports, imp)
		}

		var pExports []string
		for exp := range pkgExportMap[pkgDir] {
			pExports = append(pExports, exp)
		}

		pStats := NewPackageStats(srcCount, testCount, genCount, linesTot, sizeTot)

		indexedPackages = append(indexedPackages, NewIndexedPackage(
			pName,
			pkgDir,
			pkgDir,
			fPaths,
			pImports,
			pExports,
			pDoc,
			"",
			pStats,
		))
	}

	// 3. Compute Repository Statistics
	var configTypes []string
	for ct := range configTypesMap {
		configTypes = append(configTypes, ct)
	}
	cfgStats := NewConfigStats(len(configFiles), configTypes)

	docCoverage := 0.0
	testCoverage := 0.0
	if len(indexedPackages) > 0 {
		docCoverage = float64(packagesWithDocs) / float64(len(indexedPackages))
		testCoverage = float64(packagesWithTests) / float64(len(indexedPackages))
	}

	repStats := NewRepositoryStats(
		len(indexedFiles),
		len(indexedPackages),
		1, // default module count if 1 workspace root
		totalLines,
		codeLines,
		commentLines,
		blankLines,
		langDist,
		fileTypeDist,
		docCoverage,
		testCoverage,
		cfgStats,
	)

	return NewIndexModel(
		CurrentSchemaVersion,
		root,
		indexedFiles,
		indexedPackages,
		relationships,
		repStats,
		diagnostics,
	), nil
}

// IndexRepository discovers and indexes a domain repository instance.
func (idx *Indexer) IndexRepository(repo *repository.Repository) (*IndexModel, error) {
	if idx == nil {
		return nil, ErrNilIndexer
	}
	if repo == nil {
		return nil, ErrNilRepository
	}

	discResult, err := idx.discoverer.Discover(repo)
	if err != nil {
		return nil, fmt.Errorf("%w: discovery failed: %v", ErrIndexingFailed, err)
	}

	return idx.Index(discResult)
}

// IndexPath discovers and indexes a repository path.
func (idx *Indexer) IndexPath(path string) (*IndexModel, error) {
	if idx == nil {
		return nil, ErrNilIndexer
	}

	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return nil, ErrPathEmpty
	}

	absPath, err := filepath.Abs(filepath.Clean(cleanPath))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrIndexingFailed, err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrPathNotFound, absPath)
		}
		return nil, fmt.Errorf("%w: %v", ErrIndexingFailed, err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("%w: %s", ErrNotDirectory, absPath)
	}

	discResult, err := idx.discoverer.DiscoverPath(absPath)
	if err != nil {
		return nil, fmt.Errorf("%w: discovery failed: %v", ErrIndexingFailed, err)
	}

	return idx.Index(discResult)
}

func classifyFileType(relPath, langID string, isTest bool, encoding EncodingType) FileType {
	if isTest {
		return FileTypeTest
	}
	if encoding == EncodingUnknown {
		return FileTypeBinary
	}

	ext := strings.ToLower(filepath.Ext(relPath))
	base := strings.ToLower(filepath.Base(relPath))

	if ext == ".md" || ext == ".txt" || ext == ".rst" || ext == ".adoc" || strings.HasPrefix(base, "readme") {
		return FileTypeDoc
	}

	if ext == ".json" || ext == ".yaml" || ext == ".yml" || ext == ".toml" || ext == ".xml" || ext == ".ini" || ext == ".conf" {
		return FileTypeConfig
	}

	if langID != "" {
		return FileTypeSource
	}

	return FileTypeUnknown
}
