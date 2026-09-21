package sandbox

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

var (
	// ErrPathTraversal indicates an attempt to escape root via parent traversal.
	ErrPathTraversal = errors.New("filesystem guard: path traversal attempt detected")
	// ErrPathOutsideRoot indicates the target path is outside all authorized directories.
	ErrPathOutsideRoot = errors.New("filesystem guard: path outside authorized roots")
	// ErrSymlinkEscape indicates a symbolic link points outside permitted directories.
	ErrSymlinkEscape = errors.New("filesystem guard: symlink escape detected")
	// ErrIllegalPathFormat indicates an invalid, UNC, or reserved stream path format.
	ErrIllegalPathFormat = errors.New("filesystem guard: illegal path format")
	// ErrWriteDenied indicates writing to a read-only path was attempted.
	ErrWriteDenied = errors.New("filesystem guard: write permission denied for target path")
)

// FilesystemGuard enforces path containment, traversal protection, and read/write boundary checks.
type FilesystemGuard struct {
	mu             sync.RWMutex
	workspaceRoot  string
	pluginDataDir  string
	pluginTempDir  string
	allowedReads   []string
	allowRepoWrite bool
}

// NewFilesystemGuard constructs an initialized filesystem guard.
func NewFilesystemGuard(workspaceRoot, pluginDataDir, pluginTempDir string, allowRepoWrite bool, extraReads ...string) (*FilesystemGuard, error) {
	wsClean := filepath.Clean(workspaceRoot)
	if wsClean == "" || wsClean == "." {
		wsClean = "."
	}

	dataClean := filepath.Clean(pluginDataDir)
	tempClean := filepath.Clean(pluginTempDir)

	reads := []string{wsClean, dataClean, tempClean}
	for _, r := range extraReads {
		if c := filepath.Clean(r); c != "" {
			reads = append(reads, c)
		}
	}

	return &FilesystemGuard{
		workspaceRoot:  wsClean,
		pluginDataDir:  dataClean,
		pluginTempDir:  tempClean,
		allowedReads:   reads,
		allowRepoWrite: allowRepoWrite,
	}, nil
}

// WorkspaceRoot returns the canonical workspace path.
func (g *FilesystemGuard) WorkspaceRoot() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.workspaceRoot
}

// PluginDataDir returns the private storage directory allocated for the plugin.
func (g *FilesystemGuard) PluginDataDir() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.pluginDataDir
}

// PluginTempDir returns the ephemeral temp directory allocated for the plugin.
func (g *FilesystemGuard) PluginTempDir() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.pluginTempDir
}

// ValidateRead verifies that the requested path is permissible for reading.
func (g *FilesystemGuard) ValidateRead(targetPath string) (string, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	canon, err := g.canonicalizeAndCheckFormat(targetPath)
	if err != nil {
		return "", err
	}

	// Verify target is within at least one allowed read root
	for _, root := range g.allowedReads {
		if g.isSubpath(root, canon) {
			return canon, nil
		}
	}

	return "", fmt.Errorf("%w: path '%s' is not within permitted read roots", ErrPathOutsideRoot, targetPath)
}

// ValidateWrite verifies that the requested path is permissible for writing or creating.
func (g *FilesystemGuard) ValidateWrite(targetPath string) (string, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	canon, err := g.canonicalizeAndCheckFormat(targetPath)
	if err != nil {
		return "", err
	}

	// Writing is ALWAYS allowed inside private data and temp directories
	if g.isSubpath(g.pluginDataDir, canon) || g.isSubpath(g.pluginTempDir, canon) {
		return canon, nil
	}

	// Writing inside the repository workspace requires explicit authorization
	if g.isSubpath(g.workspaceRoot, canon) {
		if !g.allowRepoWrite {
			return "", fmt.Errorf("%w: repository workspace write access is disabled", ErrWriteDenied)
		}
		// Deny writes into sensitive workspace directories (.git, .limoxel) anywhere in hierarchy
		rel, _ := filepath.Rel(g.workspaceRoot, canon)
		relSlash := filepath.ToSlash(rel)
		components := strings.Split(relSlash, "/")
		for _, comp := range components {
			compLower := strings.ToLower(comp)
			if compLower == ".git" || compLower == ".limoxel" {
				return "", fmt.Errorf("%w: modifying host system or metadata directory '%s' is forbidden", ErrWriteDenied, relSlash)
			}
		}
		return canon, nil
	}

	return "", fmt.Errorf("%w: writing to '%s' is outside permitted storage roots", ErrPathOutsideRoot, targetPath)
}

// ValidateDelete verifies that the requested path is permissible for deletion.
func (g *FilesystemGuard) ValidateDelete(targetPath string) (string, error) {
	return g.ValidateWrite(targetPath)
}

func (g *FilesystemGuard) canonicalizeAndCheckFormat(p string) (string, error) {
	trimmed := strings.TrimSpace(p)
	if trimmed == "" {
		return "", fmt.Errorf("%w: path cannot be empty", ErrIllegalPathFormat)
	}

	// 1. Check for UNC network paths or Windows device paths (e.g. \\server\share, //server/share, \\.\, \\?\, \??\)
	if strings.HasPrefix(trimmed, `\\`) || strings.HasPrefix(trimmed, `//`) || strings.HasPrefix(trimmed, `\??\`) {
		return "", fmt.Errorf("%w: UNC network or device paths are prohibited: %s", ErrIllegalPathFormat, trimmed)
	}

	// 2. Check for NTFS alternate data streams on Windows (e.g. file.txt:secret)
	if runtime.GOOS == "windows" {
		// Drive letter prefix is allowed (e.g. C:\), but subsequent colons indicate ADS
		pathWithoutDrive := trimmed
		if len(pathWithoutDrive) >= 2 && pathWithoutDrive[1] == ':' {
			pathWithoutDrive = pathWithoutDrive[2:]
		}
		if strings.Contains(pathWithoutDrive, ":") {
			return "", fmt.Errorf("%w: alternate data stream access is prohibited: %s", ErrIllegalPathFormat, trimmed)
		}
	}

	// 3. Resolve relative to workspace if not absolute
	var fullPath string
	if filepath.IsAbs(trimmed) {
		fullPath = filepath.Clean(trimmed)
	} else {
		// Check for parent traversal attempts in relative path
		parts := strings.Split(filepath.ToSlash(trimmed), "/")
		for _, part := range parts {
			if part == ".." {
				return "", fmt.Errorf("%w: illegal traversal token '..' in relative path: %s", ErrPathTraversal, trimmed)
			}
		}
		fullPath = filepath.Clean(filepath.Join(g.workspaceRoot, trimmed))
	}

	// 4. Resolve symlinks if file/dir exists to prevent symlink jail escape
	if evalPath, err := filepath.EvalSymlinks(fullPath); err == nil {
		fullPath = filepath.Clean(evalPath)
	} else {
		// If the file does not exist yet (e.g. for write), evaluate symlinks on the deepest existing parent
		parent := filepath.Dir(fullPath)
		for parent != filepath.Dir(parent) {
			if evalParent, err := filepath.EvalSymlinks(parent); err == nil {
				rel, _ := filepath.Rel(parent, fullPath)
				fullPath = filepath.Clean(filepath.Join(evalParent, rel))
				break
			}
			parent = filepath.Dir(parent)
		}
	}

	return fullPath, nil
}

func (g *FilesystemGuard) isSubpath(root, target string) bool {
	cleanRoot := filepath.Clean(root)
	cleanTarget := filepath.Clean(target)

	// Case-insensitive comparison on Windows
	if runtime.GOOS == "windows" {
		cleanRoot = strings.ToLower(cleanRoot)
		cleanTarget = strings.ToLower(cleanTarget)
	}

	if cleanRoot == cleanTarget {
		return true
	}

	rel, err := filepath.Rel(cleanRoot, cleanTarget)
	if err != nil {
		return false
	}

	// If rel begins with "..", target is outside root
	if strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return false
	}

	return true
}

// EnsureStorageDirectories creates the plugin's private data and temp directories on disk.
func (g *FilesystemGuard) EnsureStorageDirectories() error {
	if err := os.MkdirAll(g.pluginDataDir, 0750); err != nil {
		return fmt.Errorf("failed to create plugin data directory: %w", err)
	}
	if err := os.MkdirAll(g.pluginTempDir, 0700); err != nil {
		return fmt.Errorf("failed to create plugin temp directory: %w", err)
	}
	return nil
}

// CleanupTempDirectory removes the ephemeral temporary directory.
func (g *FilesystemGuard) CleanupTempDirectory() error {
	if g.pluginTempDir != "" && g.pluginTempDir != "." {
		return os.RemoveAll(g.pluginTempDir)
	}
	return nil
}
