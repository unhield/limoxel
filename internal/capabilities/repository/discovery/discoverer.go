package discovery

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/unhield/limoxel/internal/filesystem"
	"github.com/unhield/limoxel/internal/language"
	"github.com/unhield/limoxel/internal/project"
	"github.com/unhield/limoxel/internal/repository"
	"github.com/unhield/limoxel/internal/workspace"
)

// Discoverer coordinates deterministic, secure repository discovery.
type Discoverer struct {
	langReg *language.Registry
	options Options
}

// New constructs and validates a new immutable Discoverer instance with the given Language Registry and options.
func New(langReg *language.Registry, opts ...Options) (*Discoverer, error) {
	if langReg == nil {
		return nil, ErrNilRegistry
	}

	activeOpts := DefaultOptions()
	if len(opts) > 0 {
		activeOpts = opts[0]
	}

	return &Discoverer{
		langReg: langReg,
		options: activeOpts,
	}, nil
}

// LanguageRegistry returns the Language Registry configured for the Discoverer.
func (d *Discoverer) LanguageRegistry() *language.Registry {
	if d == nil {
		return nil
	}
	return d.langReg
}

// Options returns the Options configured for the Discoverer.
func (d *Discoverer) Options() Options {
	if d == nil {
		return DefaultOptions()
	}
	return d.options
}

// Discover executes discovery for an existing validated Repository domain instance.
func (d *Discoverer) Discover(repo *repository.Repository) (*Result, error) {
	if d == nil {
		return nil, ErrNilDiscoverer
	}
	if repo == nil {
		return nil, ErrNilRepository
	}

	return d.executeDiscovery(repo.Root(), repo.Name(), repo)
}

// DiscoverPath loads, validates, resolves root, and executes discovery for a filesystem path.
func (d *Discoverer) DiscoverPath(path string) (*Result, error) {
	if d == nil {
		return nil, ErrNilDiscoverer
	}

	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return nil, ErrPathEmpty
	}

	absPath, err := filepath.Abs(filepath.Clean(cleanPath))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDiscoveryFailed, err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrPathNotFound, absPath)
		}
		return nil, fmt.Errorf("%w: %v", ErrDiscoveryFailed, err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("%w: %s", ErrNotDirectory, absPath)
	}

	// Resolve effective repository root
	effectiveRoot := d.resolveEffectiveRoot(absPath)
	repoName := filepath.Base(effectiveRoot)

	// Construct canonical domain hierarchy
	ws, err := workspace.New(repoName+"-workspace", effectiveRoot)
	if err != nil {
		return nil, fmt.Errorf("%w: failed creating workspace: %v", ErrDiscoveryFailed, err)
	}

	proj, err := project.New(repoName+"-project", ws, effectiveRoot)
	if err != nil {
		return nil, fmt.Errorf("%w: failed creating project: %v", ErrDiscoveryFailed, err)
	}

	repo, err := repository.New(repoName, proj, effectiveRoot)
	if err != nil {
		return nil, fmt.Errorf("%w: failed creating repository: %v", ErrDiscoveryFailed, err)
	}

	return d.executeDiscovery(effectiveRoot, repoName, repo)
}

func (d *Discoverer) resolveEffectiveRoot(startPath string) string {
	current := filepath.Clean(startPath)

	for {
		gitDir := filepath.Join(current, ".git")
		if info, err := os.Stat(gitDir); err == nil && (info.IsDir() || !info.IsDir()) {
			return current
		}

		parent := filepath.Dir(current)
		if parent == current || parent == "" {
			break
		}
		current = parent
	}

	return filepath.Clean(startPath)
}

func (d *Discoverer) executeDiscovery(root string, repoName string, repo *repository.Repository) (*Result, error) {
	cleanRoot := filepath.Clean(root)
	ignorer := filesystem.NewIgnorer(d.options.AdditionalIgnoreRules...)

	var (
		files        []*FileEntry
		diagnostics  []*Diagnostic
		nestedRepos  []string
		dirCount     int
		totalBytes   int64
		visitedDirs  = make(map[string]struct{})
		visitedPaths = make(map[string]struct{})
	)

	walkErr := filepath.WalkDir(cleanRoot, func(currentPath string, dEntry fs.DirEntry, err error) error {
		if err != nil {
			diagnostics = append(diagnostics, NewDiagnostic(
				SeverityWarning,
				"FILESYSTEM_ACCESS_ERROR",
				err.Error(),
				d.rel(cleanRoot, currentPath),
				false,
			))
			return nil
		}

		cleanCurrent := filepath.Clean(currentPath)

		// 1. Boundary enforcement
		rel, err := filepath.Rel(cleanRoot, cleanCurrent)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || strings.HasPrefix(rel, "../") || strings.HasPrefix(rel, "..\\") {
			return fmt.Errorf("%w: path %s escapes root %s", ErrBoundaryViolation, cleanCurrent, cleanRoot)
		}

		// Prevent duplicate traversal
		if _, seen := visitedPaths[cleanCurrent]; seen {
			if dEntry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		visitedPaths[cleanCurrent] = struct{}{}

		// 2. Ignore evaluation (skip root itself)
		if cleanCurrent != cleanRoot {
			if ignorer.ShouldIgnore(cleanCurrent) {
				if dEntry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// 3. Depth limits
		if cleanCurrent != cleanRoot && d.options.MaxDepth > 0 {
			depth := len(strings.Split(filepath.ToSlash(rel), "/"))
			if depth > d.options.MaxDepth {
				if dEntry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// 4. Nested repository detection (subdirectories with .git)
		if dEntry.IsDir() && cleanCurrent != cleanRoot {
			subGit := filepath.Join(cleanCurrent, ".git")
			if _, statErr := os.Stat(subGit); statErr == nil {
				nestedRepos = append(nestedRepos, filepath.ToSlash(rel))
			}
		}

		// 5. Handle directory
		if dEntry.IsDir() {
			dirCount++
			visitedDirs[cleanCurrent] = struct{}{}
			return nil
		}

		// 6. Handle Symlinks
		isSymlink := dEntry.Type()&fs.ModeSymlink != 0
		if isSymlink {
			if !d.options.FollowSymlinks {
				// Record symlink without following
				info, infoErr := dEntry.Info()
				var size int64
				var modTime = info.ModTime()
				if infoErr == nil {
					size = info.Size()
				}

				baseName := filepath.Base(cleanCurrent)
				isHidden := strings.HasPrefix(baseName, ".")
				if !d.options.IncludeHidden && isHidden {
					return nil
				}

				ext := filepath.Ext(baseName)
				lang, _ := d.langReg.DiscoverByFilename(baseName)

				files = append(files, NewFileEntry(
					rel,
					cleanCurrent,
					false,
					size,
					modTime,
					ext,
					lang,
					isHidden,
					true,
					false,
				))
				return nil
			}

			// Follow symlink safely
			target, evalErr := filepath.EvalSymlinks(cleanCurrent)
			if evalErr != nil {
				diagnostics = append(diagnostics, NewDiagnostic(
					SeverityWarning,
					"UNRESOLVED_SYMLINK",
					evalErr.Error(),
					filepath.ToSlash(rel),
					false,
				))
				return nil
			}

			targetClean := filepath.Clean(target)
			targetRel, relErr := filepath.Rel(cleanRoot, targetClean)
			if relErr != nil || targetRel == ".." || strings.HasPrefix(targetRel, ".."+string(filepath.Separator)) || strings.HasPrefix(targetRel, "../") || strings.HasPrefix(targetRel, "..\\") {
				diagnostics = append(diagnostics, NewDiagnostic(
					SeverityWarning,
					"SYMLINK_ESCAPES_BOUNDARY",
					fmt.Sprintf("target %s is outside repository root", targetClean),
					filepath.ToSlash(rel),
					false,
				))
				return nil
			}
		}

		// 7. Hidden file check
		baseName := filepath.Base(cleanCurrent)
		isHidden := strings.HasPrefix(baseName, ".")
		if !d.options.IncludeHidden && isHidden {
			return nil
		}

		// 8. Max files limit
		if d.options.MaxFiles > 0 && len(files) >= d.options.MaxFiles {
			diagnostics = append(diagnostics, NewDiagnostic(
				SeverityWarning,
				"TRAVERSAL_LIMIT_REACHED",
				fmt.Sprintf("maximum file limit (%d) reached", d.options.MaxFiles),
				filepath.ToSlash(rel),
				false,
			))
			return fs.SkipAll
		}

		// 9. Stat regular file
		info, err := dEntry.Info()
		if err != nil {
			diagnostics = append(diagnostics, NewDiagnostic(
				SeverityWarning,
				"FILE_STAT_ERROR",
				err.Error(),
				filepath.ToSlash(rel),
				false,
			))
			return nil
		}

		ext := filepath.Ext(baseName)
		lang, _ := d.langReg.DiscoverByFilename(baseName)

		size := info.Size()
		totalBytes += size

		files = append(files, NewFileEntry(
			rel,
			cleanCurrent,
			false,
			size,
			info.ModTime(),
			ext,
			lang,
			isHidden,
			isSymlink,
			false,
		))

		return nil
	})

	if walkErr != nil && !errors.Is(walkErr, fs.SkipAll) {
		return nil, fmt.Errorf("%w: %v", ErrDiscoveryFailed, walkErr)
	}

	// Calculate language distributions
	langDistributions := d.calculateLanguageDistributions(files)

	// Extract read-only local Git metadata
	meta := d.extractMetadata(cleanRoot, repoName, len(files), dirCount, totalBytes, nestedRepos, &diagnostics)

	return NewResult(
		repo,
		cleanRoot,
		files,
		langDistributions,
		meta,
		diagnostics,
		nestedRepos,
	), nil
}

func (d *Discoverer) rel(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

func (d *Discoverer) calculateLanguageDistributions(files []*FileEntry) []*LanguageDistribution {
	if len(files) == 0 {
		return nil
	}

	type langStat struct {
		name       string
		count      int
		bytes      int64
		extensions map[string]struct{}
	}

	stats := make(map[string]*langStat)

	for _, f := range files {
		langID := f.LanguageID()
		langName := f.LanguageName()

		st, exists := stats[langID]
		if !exists {
			st = &langStat{
				name:       langName,
				extensions: make(map[string]struct{}),
			}
			stats[langID] = st
		}

		st.count++
		st.bytes += f.Size()
		if f.Extension() != "" {
			st.extensions[f.Extension()] = struct{}{}
		}
	}

	totalFiles := len(files)
	result := make([]*LanguageDistribution, 0, len(stats))

	for id, st := range stats {
		var pct float64
		if totalFiles > 0 {
			pct = (float64(st.count) / float64(totalFiles)) * 100.0
		}

		extList := make([]string, 0, len(st.extensions))
		for ext := range st.extensions {
			extList = append(extList, ext)
		}
		sort.Strings(extList)

		result = append(result, NewLanguageDistribution(
			id,
			st.name,
			st.count,
			st.bytes,
			pct,
			extList,
		))
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].FileCount() != result[j].FileCount() {
			return result[i].FileCount() > result[j].FileCount()
		}
		if result[i].TotalBytes() != result[j].TotalBytes() {
			return result[i].TotalBytes() > result[j].TotalBytes()
		}
		return result[i].LanguageID() < result[j].LanguageID()
	})

	return result
}

func (d *Discoverer) extractMetadata(
	root string,
	repoName string,
	totalFiles int,
	totalDirs int,
	totalBytes int64,
	nestedRepos []string,
	diagnostics *[]*Diagnostic,
) *Metadata {
	gitDir := filepath.Join(root, ".git")
	info, err := os.Stat(gitDir)
	isGit := err == nil && info.IsDir()

	if !isGit {
		return NewMetadata(
			repoName,
			root,
			false,
			"",
			"",
			"",
			totalFiles,
			totalDirs,
			totalBytes,
			nestedRepos,
		)
	}

	var currentBranch, latestCommit, defaultBranch string

	// Read HEAD safely
	headBytes, err := os.ReadFile(filepath.Join(gitDir, "HEAD"))
	if err == nil {
		headContent := strings.TrimSpace(string(headBytes))
		if strings.HasPrefix(headContent, "ref: ") {
			refPath := strings.TrimPrefix(headContent, "ref: ")
			if strings.HasPrefix(refPath, "refs/heads/") {
				currentBranch = strings.TrimPrefix(refPath, "refs/heads/")
			} else {
				currentBranch = refPath
			}

			// Read commit from ref file
			refFile := filepath.Join(gitDir, filepath.FromSlash(refPath))
			if refBytes, refErr := os.ReadFile(refFile); refErr == nil {
				latestCommit = strings.TrimSpace(string(refBytes))
			} else {
				// Check packed-refs
				commit, packedErr := d.readPackedRef(gitDir, refPath)
				if packedErr != nil {
					if diagnostics != nil {
						*diagnostics = append(*diagnostics, NewDiagnostic(
							SeverityWarning,
							"GIT_PACKED_REFS_READ_ERROR",
							packedErr.Error(),
							".git/packed-refs",
							false,
						))
					}
				} else {
					latestCommit = commit
				}
			}
		} else if len(headContent) == 40 || len(headContent) == 64 {
			// Detached HEAD commit hash
			latestCommit = headContent
			currentBranch = "HEAD"
		}
	}

	// Try finding default branch from refs/remotes/origin/HEAD or common branches
	defaultBranch = d.detectDefaultBranch(gitDir)

	return NewMetadata(
		repoName,
		root,
		true,
		currentBranch,
		defaultBranch,
		latestCommit,
		totalFiles,
		totalDirs,
		totalBytes,
		nestedRepos,
	)
}

func (d *Discoverer) readPackedRef(gitDir, refPath string) (string, error) {
	packedFile := filepath.Join(gitDir, "packed-refs")
	f, err := os.Open(packedFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("open packed-refs: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "^") || line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[1] == refPath {
			return parts[0], nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("scan packed-refs: %w", err)
	}

	return "", nil
}

func (d *Discoverer) detectDefaultBranch(gitDir string) string {
	// 1. Check remotes/origin/HEAD
	originHead := filepath.Join(gitDir, "refs", "remotes", "origin", "HEAD")
	if bytes, err := os.ReadFile(originHead); err == nil {
		content := strings.TrimSpace(string(bytes))
		if strings.HasPrefix(content, "ref: refs/remotes/origin/") {
			return strings.TrimPrefix(content, "ref: refs/remotes/origin/")
		}
	}

	// 2. Check if main or master exist
	if _, err := os.Stat(filepath.Join(gitDir, "refs", "heads", "main")); err == nil {
		return "main"
	}
	if _, err := os.Stat(filepath.Join(gitDir, "refs", "heads", "master")); err == nil {
		return "master"
	}

	return ""
}
