package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
)

// Filesystem defines the platform interface contract for operating system filesystem interactions.
type Filesystem interface {
	Exists(path string) bool
	Stat(path string) (*Entry, error)
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm os.FileMode) error
	ReadDir(path string) ([]*Entry, error)
	MkdirAll(path string, perm os.FileMode) error
	Remove(path string) error
	RemoveAll(path string) error
	Rename(oldPath, newPath string) error
}

// OSFilesystem is an immutable, thread-safe implementation of Filesystem interacting with the OS.
type OSFilesystem struct{}

// NewOSFilesystem constructs a new OSFilesystem instance.
func NewOSFilesystem() *OSFilesystem {
	return &OSFilesystem{}
}

// Exists reports whether the file or directory at path exists on the filesystem.
func (fs *OSFilesystem) Exists(path string) bool {
	if fs == nil || path == "" {
		return false
	}
	clean := filepath.Clean(path)
	_, err := os.Stat(clean)
	return err == nil || !os.IsNotExist(err)
}

// Stat retrieves filesystem Entry metadata for the specified path.
func (fs *OSFilesystem) Stat(path string) (*Entry, error) {
	if fs == nil {
		return nil, ErrNilFilesystem
	}
	clean := filepath.Clean(path)
	if clean == "" {
		return nil, ErrPathEmpty
	}

	info, err := os.Stat(clean)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrFileNotFound, clean)
		}
		return nil, err
	}

	return NewEntry(clean, info.IsDir(), info.Size(), info.ModTime()), nil
}

// ReadFile reads and returns the complete binary content of the file at path.
func (fs *OSFilesystem) ReadFile(path string) ([]byte, error) {
	if fs == nil {
		return nil, ErrNilFilesystem
	}
	clean := filepath.Clean(path)
	if clean == "" {
		return nil, ErrPathEmpty
	}

	data, err := os.ReadFile(clean)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrFileNotFound, clean)
		}
		return nil, err
	}

	return data, nil
}

// WriteFile writes data to a file at path with the given permission mode.
func (fs *OSFilesystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	if fs == nil {
		return ErrNilFilesystem
	}
	clean := filepath.Clean(path)
	if clean == "" {
		return ErrPathEmpty
	}
	return os.WriteFile(clean, data, perm)
}

// ReadDir reads the directory at path and returns a slice of Entry metadata for its children.
func (fs *OSFilesystem) ReadDir(path string) ([]*Entry, error) {
	if fs == nil {
		return nil, ErrNilFilesystem
	}
	clean := filepath.Clean(path)
	if clean == "" {
		return nil, ErrPathEmpty
	}

	entries, err := os.ReadDir(clean)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrFileNotFound, clean)
		}
		return nil, err
	}

	result := make([]*Entry, 0, len(entries))
	for _, dirEntry := range entries {
		info, infoErr := dirEntry.Info()
		if infoErr != nil {
			continue
		}
		childPath := filepath.Join(clean, dirEntry.Name())
		result = append(result, NewEntry(childPath, dirEntry.IsDir(), info.Size(), info.ModTime()))
	}

	return result, nil
}

// MkdirAll creates a directory at path along with any necessary parent directories.
func (fs *OSFilesystem) MkdirAll(path string, perm os.FileMode) error {
	if fs == nil {
		return ErrNilFilesystem
	}
	clean := filepath.Clean(path)
	if clean == "" {
		return ErrPathEmpty
	}
	return os.MkdirAll(clean, perm)
}

// Remove removes the file or empty directory at path.
func (fs *OSFilesystem) Remove(path string) error {
	if fs == nil {
		return ErrNilFilesystem
	}
	clean := filepath.Clean(path)
	if clean == "" {
		return ErrPathEmpty
	}
	err := os.Remove(clean)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrFileNotFound, clean)
		}
		return err
	}
	return nil
}

// RemoveAll removes path and any children it contains.
func (fs *OSFilesystem) RemoveAll(path string) error {
	if fs == nil {
		return ErrNilFilesystem
	}
	clean := filepath.Clean(path)
	if clean == "" {
		return ErrPathEmpty
	}
	return os.RemoveAll(clean)
}

// Rename renames (moves) oldPath to newPath.
func (fs *OSFilesystem) Rename(oldPath, newPath string) error {
	if fs == nil {
		return ErrNilFilesystem
	}
	cleanOld := filepath.Clean(oldPath)
	cleanNew := filepath.Clean(newPath)
	if cleanOld == "" || cleanNew == "" {
		return ErrPathEmpty
	}
	err := os.Rename(cleanOld, cleanNew)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrFileNotFound, cleanOld)
		}
		return err
	}
	return nil
}
