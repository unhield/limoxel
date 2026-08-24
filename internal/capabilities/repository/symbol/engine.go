package symbol

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/unhield/limoxel/internal/capabilities/repository/discovery"
	"github.com/unhield/limoxel/internal/repository"
)

// Engine orchestrates AST parsing, symbol extraction, documentation extraction, and relationship graph construction.
type Engine struct {
	discoverer *discovery.Discoverer
	workers    int
}

// New constructs an operational Engine with default worker concurrency.
func New(discoverer *discovery.Discoverer) (*Engine, error) {
	if discoverer == nil {
		return nil, ErrNilDiscoverer
	}
	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}
	return &Engine{
		discoverer: discoverer,
		workers:    workers,
	}, nil
}

// NewWithWorkers constructs an operational Engine with custom worker concurrency.
func NewWithWorkers(discoverer *discovery.Discoverer, workers int) (*Engine, error) {
	if discoverer == nil {
		return nil, ErrNilDiscoverer
	}
	if workers < 1 {
		workers = 1
	}
	return &Engine{
		discoverer: discoverer,
		workers:    workers,
	}, nil
}

// Discoverer returns the underlying repository discoverer.
func (e *Engine) Discoverer() *discovery.Discoverer {
	if e == nil {
		return nil
	}
	return e.discoverer
}

// Parse processes an established Discovery Result and builds the complete SymbolModel.
func (e *Engine) Parse(discResult *discovery.Result) (*SymbolModel, error) {
	return e.ParseIncremental(discResult, nil)
}

// ParseIncremental processes a Discovery Result reusing cached symbols from previous SymbolModel where files are unchanged.
func (e *Engine) ParseIncremental(discResult *discovery.Result, previous *SymbolModel) (*SymbolModel, error) {
	if e == nil {
		return nil, ErrNilEngine
	}
	if discResult == nil {
		return nil, ErrNilDiscoveryResult
	}

	repoRoot := discResult.Root()
	files := discResult.Files()

	// Map previous symbols by file path for incremental reuse
	prevSymbolsByFile := make(map[string][]*Symbol)
	prevDocsByFile := make(map[string][]*DocEntry)
	prevRelsByFile := make(map[string][]*SymbolRelationship)

	if previous != nil && previous.RepositoryRoot() == filepath.ToSlash(filepath.Clean(repoRoot)) {
		if previous.Symbols() != nil {
			for _, s := range previous.Symbols().AllSymbols() {
				prevSymbolsByFile[s.FilePath()] = append(prevSymbolsByFile[s.FilePath()], s)
			}
		}
		if previous.Docs() != nil {
			for _, d := range previous.Docs().AllDocs() {
				if d.Position() != nil {
					prevDocsByFile[d.Position().File()] = append(prevDocsByFile[d.Position().File()], d)
				}
			}
		}
		if previous.Relationships() != nil {
			for _, r := range previous.Relationships().AllRelationships() {
				if r.Position() != nil {
					prevRelsByFile[r.Position().File()] = append(prevRelsByFile[r.Position().File()], r)
				}
			}
		}
	}

	type fileJob struct {
		absPath string
		relPath string
	}

	type parseResult struct {
		symbols       []*Symbol
		docs          []*DocEntry
		relationships []*SymbolRelationship
		diagnostics   []*discovery.Diagnostic
	}

	var (
		jobs        []fileJob
		reusedSyms  []*Symbol
		reusedDocs  []*DocEntry
		reusedRels  []*SymbolRelationship
		diagnostics []*discovery.Diagnostic
	)

	fileHashes := make(map[string]string)

	// Filter Go source files
	for _, f := range files {
		cleanRel := filepath.ToSlash(filepath.Clean(f.RelPath()))
		if !strings.HasSuffix(cleanRel, ".go") {
			continue
		}

		absPath := filepath.Join(repoRoot, filepath.FromSlash(cleanRel))
		currentHash, hashErr := computeFileContentHash(absPath)
		if hashErr == nil {
			fileHashes[cleanRel] = currentHash
		}

		// Check if we can reuse cached symbols based on content hash
		if previous != nil && hashErr == nil && previous.FileHash(cleanRel) == currentHash {
			if prevSyms, ok := prevSymbolsByFile[cleanRel]; ok && len(prevSyms) > 0 {
				reusedSyms = append(reusedSyms, prevSyms...)
				if dList, ok := prevDocsByFile[cleanRel]; ok {
					reusedDocs = append(reusedDocs, dList...)
				}
				if rList, ok := prevRelsByFile[cleanRel]; ok {
					reusedRels = append(reusedRels, rList...)
				}
				continue
			}
		}

		jobs = append(jobs, fileJob{
			absPath: absPath,
			relPath: cleanRel,
		})
	}

	results := make([]parseResult, len(jobs))
	var wg sync.WaitGroup
	jobChan := make(chan int, len(jobs))

	workerCount := e.workers
	if workerCount > len(jobs) {
		workerCount = len(jobs)
	}
	if workerCount < 1 {
		workerCount = 1
	}

	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobChan {
				job := jobs[idx]
				syms, docs, rels, diags := parseGoFile(job.absPath, job.relPath)
				results[idx] = parseResult{
					symbols:       syms,
					docs:          docs,
					relationships: rels,
					diagnostics:   diags,
				}
			}
		}()
	}

	for i := range jobs {
		jobChan <- i
	}
	close(jobChan)
	wg.Wait()

	allSymbols := append([]*Symbol{}, reusedSyms...)
	allDocs := append([]*DocEntry{}, reusedDocs...)
	allRels := append([]*SymbolRelationship{}, reusedRels...)

	for _, res := range results {
		allSymbols = append(allSymbols, res.symbols...)
		allDocs = append(allDocs, res.docs...)
		allRels = append(allRels, res.relationships...)
		diagnostics = append(diagnostics, res.diagnostics...)
	}

	// 4. Compute Interface Implementation Relationships across extracted symbols
	interfaceRels := computeInterfaceImplementations(allSymbols)
	allRels = append(allRels, interfaceRels...)

	symDB := NewSymbolDatabase(allSymbols)
	docDB := NewDocumentationDatabase(allDocs)
	relGraph := NewSymbolRelationshipGraph(allRels)

	return NewSymbolModelWithHashes(
		repoRoot,
		symDB,
		docDB,
		relGraph,
		diagnostics,
		fileHashes,
	), nil
}

// computeInterfaceImplementations establishes RelInterfaceImplementation relationships where structs implement interfaces.
func computeInterfaceImplementations(symbols []*Symbol) []*SymbolRelationship {
	var (
		interfaces []*Symbol
		structs    []*Symbol
		methods    []*Symbol
	)

	for _, s := range symbols {
		switch s.Kind() {
		case SymbolKindInterface:
			interfaces = append(interfaces, s)
		case SymbolKindStruct:
			structs = append(structs, s)
		case SymbolKindMethod:
			methods = append(methods, s)
		}
	}

	// Map methods by receiver type symbol ID
	structMethods := make(map[string]map[string]string)
	for _, m := range methods {
		baseRecv := strings.TrimPrefix(m.ReceiverType(), "*")
		if baseRecv == "" {
			continue
		}
		var recvSymID string
		if m.PackagePath() != "" && m.PackagePath() != "." {
			recvSymID = fmt.Sprintf("%s.%s", m.PackagePath(), baseRecv)
		} else {
			recvSymID = baseRecv
		}
		if structMethods[recvSymID] == nil {
			structMethods[recvSymID] = make(map[string]string)
		}
		structMethods[recvSymID][m.Name()] = m.Signature()
	}

	var rels []*SymbolRelationship

	for _, iface := range interfaces {
		if len(iface.Fields()) == 0 {
			continue
		}

		type reqMethod struct {
			name string
			sig  string
		}
		var reqMethods []reqMethod

		for _, f := range iface.Fields() {
			if strings.HasPrefix(f, "embedded: ") {
				continue
			}
			parts := strings.SplitN(f, " ", 2)
			mName := strings.TrimSpace(parts[0])
			if mName == "" {
				continue
			}
			mSig := ""
			if len(parts) > 1 {
				mSig = strings.TrimSpace(parts[1])
			}
			reqMethods = append(reqMethods, reqMethod{name: mName, sig: mSig})
		}

		if len(reqMethods) == 0 {
			continue
		}

		// Check structs in the same package
		for _, str := range structs {
			if str.PackagePath() != iface.PackagePath() {
				continue
			}

			implementedMethods := structMethods[str.ID()]
			if implementedMethods == nil {
				continue
			}

			allMatch := true
			for _, rm := range reqMethods {
				implSig, ok := implementedMethods[rm.name]
				if !ok {
					allMatch = false
					break
				}
				if rm.sig != "" && implSig != "" && implSig != rm.sig {
					allMatch = false
					break
				}
			}

			if allMatch {
				rels = append(rels, NewSymbolRelationship(
					str.ID(),
					iface.ID(),
					RelInterfaceImplementation,
					"implements_interface_methods",
					str.Position(),
				))
			}
		}
	}

	return rels
}

// computeFileContentHash returns the hex-encoded SHA-256 hash of a file's contents.
func computeFileContentHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// ParseRepository extracts symbols from a Domain Repository instance.
func (e *Engine) ParseRepository(repo *repository.Repository) (*SymbolModel, error) {
	if e == nil {
		return nil, ErrNilEngine
	}
	if repo == nil {
		return nil, ErrNilRepository
	}

	discResult, err := e.discoverer.Discover(repo)
	if err != nil {
		return nil, err
	}

	return e.Parse(discResult)
}

// ParsePath extracts symbols from a repository path on disk.
func (e *Engine) ParsePath(path string) (*SymbolModel, error) {
	if e == nil {
		return nil, ErrNilEngine
	}
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return nil, ErrPathEmpty
	}

	discResult, err := e.discoverer.DiscoverPath(cleanPath)
	if err != nil {
		return nil, err
	}

	return e.Parse(discResult)
}
