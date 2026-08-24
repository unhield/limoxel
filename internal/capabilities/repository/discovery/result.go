package discovery

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/unhield/limoxel/internal/repository"
)

// Result represents an immutable, deterministically sorted repository discovery result.
type Result struct {
	repo        *repository.Repository
	root        string
	files       []*FileEntry
	fileMap     map[string]*FileEntry
	languages   []*LanguageDistribution
	langMap     map[string]*LanguageDistribution
	metadata    *Metadata
	diagnostics []*Diagnostic
	nestedRepos []string
}

// NewResult constructs an immutable Result record with defensively copied, sorted collections.
func NewResult(
	repo *repository.Repository,
	root string,
	files []*FileEntry,
	languages []*LanguageDistribution,
	meta *Metadata,
	diagnostics []*Diagnostic,
	nestedRepos []string,
) *Result {
	// Defensive copy & deterministic sort for files by RelPath
	fileList := make([]*FileEntry, len(files))
	copy(fileList, files)
	sort.Slice(fileList, func(i, j int) bool {
		return fileList[i].relPath < fileList[j].relPath
	})

	fileMap := make(map[string]*FileEntry, len(fileList))
	for _, f := range fileList {
		fileMap[f.relPath] = f
	}

	// Defensive copy & deterministic sort for languages (by count desc, bytes desc, id asc)
	langList := make([]*LanguageDistribution, len(languages))
	copy(langList, languages)
	sort.Slice(langList, func(i, j int) bool {
		if langList[i].fileCount != langList[j].fileCount {
			return langList[i].fileCount > langList[j].fileCount
		}
		if langList[i].totalBytes != langList[j].totalBytes {
			return langList[i].totalBytes > langList[j].totalBytes
		}
		return langList[i].id < langList[j].id
	})

	langMap := make(map[string]*LanguageDistribution, len(langList))
	for _, l := range langList {
		langMap[l.id] = l
	}

	// Defensive copy for diagnostics
	diagList := make([]*Diagnostic, len(diagnostics))
	copy(diagList, diagnostics)

	// Defensive copy & sort for nested repositories
	repos := make([]string, len(nestedRepos))
	copy(repos, nestedRepos)
	sort.Strings(repos)

	return &Result{
		repo:        repo,
		root:        filepath.Clean(root),
		files:       fileList,
		fileMap:     fileMap,
		languages:   langList,
		langMap:     langMap,
		metadata:    meta,
		diagnostics: diagList,
		nestedRepos: repos,
	}
}

// Repository returns the canonical Repository domain model instance.
func (r *Result) Repository() *repository.Repository {
	if r == nil {
		return nil
	}
	return r.repo
}

// Root returns the cleaned absolute path to the repository root directory.
func (r *Result) Root() string {
	if r == nil {
		return ""
	}
	return r.root
}

// Files returns a defensive copy of all discovered FileEntry items in deterministic sorted order.
func (r *Result) Files() []*FileEntry {
	if r == nil || len(r.files) == 0 {
		return nil
	}
	cloned := make([]*FileEntry, len(r.files))
	copy(cloned, r.files)
	return cloned
}

// FileCount returns the total number of regular files discovered.
func (r *Result) FileCount() int {
	if r == nil {
		return 0
	}
	return len(r.files)
}

// File returns the FileEntry matching the specified repository-relative path, if present.
func (r *Result) File(relPath string) (*FileEntry, bool) {
	if r == nil || r.fileMap == nil {
		return nil, false
	}
	clean := filepath.ToSlash(filepath.Clean(relPath))
	clean = strings.TrimPrefix(clean, "./")
	entry, exists := r.fileMap[clean]
	return entry, exists
}

// Languages returns a defensive copy of aggregated LanguageDistribution records in deterministic sorted order.
func (r *Result) Languages() []*LanguageDistribution {
	if r == nil || len(r.languages) == 0 {
		return nil
	}
	cloned := make([]*LanguageDistribution, len(r.languages))
	copy(cloned, r.languages)
	return cloned
}

// Language returns the LanguageDistribution matching the specified language ID, if present.
func (r *Result) Language(id string) (*LanguageDistribution, bool) {
	if r == nil || r.langMap == nil {
		return nil, false
	}
	l, exists := r.langMap[strings.ToLower(strings.TrimSpace(id))]
	return l, exists
}

// Metadata returns the repository metadata summary.
func (r *Result) Metadata() *Metadata {
	if r == nil {
		return nil
	}
	return r.metadata
}

// Diagnostics returns a defensive copy of all discovery diagnostics in order of generation.
func (r *Result) Diagnostics() []*Diagnostic {
	if r == nil || len(r.diagnostics) == 0 {
		return nil
	}
	cloned := make([]*Diagnostic, len(r.diagnostics))
	copy(cloned, r.diagnostics)
	return cloned
}

// NestedRepositories returns a defensive copy of relative paths to discovered nested repositories.
func (r *Result) NestedRepositories() []string {
	if r == nil || len(r.nestedRepos) == 0 {
		return nil
	}
	cloned := make([]string, len(r.nestedRepos))
	copy(cloned, r.nestedRepos)
	return cloned
}

// String returns a human-readable representation of the Result.
func (r *Result) String() string {
	if r == nil {
		return ""
	}
	return fmt.Sprintf("Result<%s>[files=%d, languages=%d]", r.root, len(r.files), len(r.languages))
}
