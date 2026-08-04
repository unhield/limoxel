package filesystem

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Entry represents an immutable filesystem entry discovered during traversal.
type Entry struct {
	path    string
	isDir   bool
	size    int64
	modTime time.Time
}

// NewEntry constructs and validates a new immutable Entry.
func NewEntry(path string, isDir bool, size int64, modTime time.Time) *Entry {
	return &Entry{
		path:    filepath.Clean(path),
		isDir:   isDir,
		size:    size,
		modTime: modTime,
	}
}

// Path returns the cleaned absolute path of the entry.
func (e *Entry) Path() string {
	if e == nil {
		return ""
	}
	return e.path
}

// IsDir reports whether the entry is a directory.
func (e *Entry) IsDir() bool {
	if e == nil {
		return false
	}
	return e.isDir
}

// Size returns the file size in bytes.
func (e *Entry) Size() int64 {
	if e == nil {
		return 0
	}
	return e.size
}

// ModTime returns the entry modification timestamp.
func (e *Entry) ModTime() time.Time {
	if e == nil {
		return time.Time{}
	}
	return e.modTime
}

// Result represents an immutable, deterministically sorted collection of discovered Entry objects.
type Result struct {
	root    string
	entries []*Entry
}

// Root returns the target root path of the discovery operation.
func (r *Result) Root() string {
	if r == nil {
		return ""
	}
	return r.root
}

// Count returns the total number of discovered entries.
func (r *Result) Count() int {
	if r == nil {
		return 0
	}
	return len(r.entries)
}

// Entries returns a defensive copy of all discovered Entry objects.
func (r *Result) Entries() []*Entry {
	if r == nil || len(r.entries) == 0 {
		return nil
	}
	cloned := make([]*Entry, len(r.entries))
	copy(cloned, r.entries)
	return cloned
}

// Discoverer coordinates deterministic filesystem discovery from a root path.
type Discoverer struct {
	root    string
	ignorer *Ignorer
}

// NewDiscoverer constructs and validates a new immutable Discoverer for rootPath with an optional Ignorer.
func NewDiscoverer(rootPath string, ignorer ...*Ignorer) (*Discoverer, error) {
	if rootPath == "" {
		return nil, ErrPathEmpty
	}

	cleanRoot := filepath.Clean(rootPath)
	absRoot, err := filepath.Abs(cleanRoot)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDiscoveryFailed, err)
	}

	info, err := os.Stat(absRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrRootNotFound, absRoot)
		}
		return nil, err
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("%w: %s", ErrNotDirectory, absRoot)
	}

	var activeIgnorer *Ignorer
	if len(ignorer) > 0 {
		activeIgnorer = ignorer[0]
	}

	return &Discoverer{
		root:    absRoot,
		ignorer: activeIgnorer,
	}, nil
}

// Root returns the root path associated with the Discoverer.
func (d *Discoverer) Root() string {
	if d == nil {
		return ""
	}
	return d.root
}

// Ignorer returns the optional Ignorer instance associated with the Discoverer.
func (d *Discoverer) Ignorer() *Ignorer {
	if d == nil {
		return nil
	}
	return d.ignorer
}

// Discover performs deterministic filesystem traversal starting from root.
func (d *Discoverer) Discover() (*Result, error) {
	if d == nil || d.root == "" {
		return nil, ErrNilDiscoverer
	}

	var entries []*Entry

	err := filepath.WalkDir(d.root, func(path string, dEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.ignorer != nil && d.ignorer.ShouldIgnore(path) {
			if dEntry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		info, err := dEntry.Info()
		if err != nil {
			return err
		}

		entries = append(entries, NewEntry(path, dEntry.IsDir(), info.Size(), info.ModTime()))
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDiscoveryFailed, err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].path < entries[j].path
	})

	return &Result{
		root:    d.root,
		entries: entries,
	}, nil
}
