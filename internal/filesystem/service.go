package filesystem

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// FileService provides high-level file operations orchestrated through a Filesystem abstraction.
type FileService struct {
	fs Filesystem
}

// NewFileService constructs a new FileService using the provided Filesystem abstraction.
func NewFileService(fs Filesystem) (*FileService, error) {
	if fs == nil {
		return nil, ErrNilFilesystem
	}
	return &FileService{fs: fs}, nil
}

// Exists reports whether the file or directory at path exists.
func (s *FileService) Exists(path string) bool {
	if s == nil || s.fs == nil || path == "" {
		return false
	}
	return s.fs.Exists(path)
}

// ReadFile reads and returns the contents of the file at path.
func (s *FileService) ReadFile(path string) ([]byte, error) {
	if s == nil || s.fs == nil {
		return nil, ErrNilService
	}
	clean := filepath.Clean(path)
	if clean == "" {
		return nil, ErrPathEmpty
	}
	return s.fs.ReadFile(clean)
}

// WriteFile writes data to the file at path with permission perm.
func (s *FileService) WriteFile(path string, data []byte, perm os.FileMode) error {
	if s == nil || s.fs == nil {
		return ErrNilService
	}
	clean := filepath.Clean(path)
	if clean == "" {
		return ErrPathEmpty
	}
	return s.fs.WriteFile(clean, data, perm)
}

// MkdirAll creates a directory named path along with any necessary parents.
func (s *FileService) MkdirAll(path string, perm os.FileMode) error {
	if s == nil || s.fs == nil {
		return ErrNilService
	}
	clean := filepath.Clean(path)
	if clean == "" {
		return ErrPathEmpty
	}
	return s.fs.MkdirAll(clean, perm)
}

// EnsureDirectory verifies that a directory exists at path, creating it with 0755 permissions if missing.
func (s *FileService) EnsureDirectory(path string) error {
	if s == nil || s.fs == nil {
		return ErrNilService
	}
	clean := filepath.Clean(path)
	if clean == "" {
		return ErrPathEmpty
	}

	stat, err := s.fs.Stat(clean)
	if err == nil {
		if !stat.IsDir() {
			return fmt.Errorf("%w: %s", ErrNotDirectory, clean)
		}
		return nil
	}

	if errors.Is(err, ErrFileNotFound) {
		return s.fs.MkdirAll(clean, 0755)
	}

	return err
}

// Remove removes the file or empty directory at path.
func (s *FileService) Remove(path string) error {
	if s == nil || s.fs == nil {
		return ErrNilService
	}
	clean := filepath.Clean(path)
	if clean == "" {
		return ErrPathEmpty
	}
	return s.fs.Remove(clean)
}

// RemoveAll removes path and any children it contains.
func (s *FileService) RemoveAll(path string) error {
	if s == nil || s.fs == nil {
		return ErrNilService
	}
	clean := filepath.Clean(path)
	if clean == "" {
		return ErrPathEmpty
	}
	return s.fs.RemoveAll(clean)
}

// Rename renames (moves) oldPath to newPath.
func (s *FileService) Rename(oldPath, newPath string) error {
	if s == nil || s.fs == nil {
		return ErrNilService
	}
	cleanOld := filepath.Clean(oldPath)
	cleanNew := filepath.Clean(newPath)
	if cleanOld == "" || cleanNew == "" {
		return ErrPathEmpty
	}
	return s.fs.Rename(cleanOld, cleanNew)
}

// CopyFile copies the content of src to dst, preserving 0644 permissions.
func (s *FileService) CopyFile(src, dst string) error {
	if s == nil || s.fs == nil {
		return ErrNilService
	}
	cleanSrc := filepath.Clean(src)
	cleanDst := filepath.Clean(dst)
	if cleanSrc == "" || cleanDst == "" {
		return ErrPathEmpty
	}

	data, err := s.fs.ReadFile(cleanSrc)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSourceNotFound, err)
	}

	parentDir := filepath.Dir(cleanDst)
	if parentDir != "" && parentDir != "." {
		if err := s.EnsureDirectory(parentDir); err != nil {
			return err
		}
	}

	return s.fs.WriteFile(cleanDst, data, 0644)
}
