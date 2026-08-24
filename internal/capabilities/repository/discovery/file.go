package discovery

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/unhield/limoxel/internal/language"
)

// FileEntry represents an immutable discovered repository file or directory entry.
type FileEntry struct {
	relPath   string
	absPath   string
	isDir     bool
	size      int64
	modTime   time.Time
	extension string
	language  *language.Language
	isHidden  bool
	isSymlink bool
	isIgnored bool
}

// NewFileEntry creates and validates a new immutable FileEntry.
func NewFileEntry(
	relPath string,
	absPath string,
	isDir bool,
	size int64,
	modTime time.Time,
	extension string,
	lang *language.Language,
	isHidden bool,
	isSymlink bool,
	isIgnored bool,
) *FileEntry {
	// Normalize relative path with forward slashes for portable repository identity
	unified := strings.ReplaceAll(relPath, "\\", "/")
	cleanRel := path.Clean(unified)
	cleanRel = strings.TrimPrefix(cleanRel, "./")
	cleanRel = strings.TrimPrefix(cleanRel, "/")
	cleanAbs := filepath.Clean(absPath)
	cleanExt := strings.ToLower(strings.TrimSpace(extension))

	return &FileEntry{
		relPath:   cleanRel,
		absPath:   cleanAbs,
		isDir:     isDir,
		size:      size,
		modTime:   modTime,
		extension: cleanExt,
		language:  lang,
		isHidden:  isHidden,
		isSymlink: isSymlink,
		isIgnored: isIgnored,
	}
}

// RelPath returns the canonical repository-relative path (using forward slashes).
func (f *FileEntry) RelPath() string {
	if f == nil {
		return ""
	}
	return f.relPath
}

// AbsPath returns the normalized absolute filesystem path.
func (f *FileEntry) AbsPath() string {
	if f == nil {
		return ""
	}
	return f.absPath
}

// IsDir reports whether the entry is a directory.
func (f *FileEntry) IsDir() bool {
	if f == nil {
		return false
	}
	return f.isDir
}

// Size returns the file size in bytes.
func (f *FileEntry) Size() int64 {
	if f == nil {
		return 0
	}
	return f.size
}

// ModTime returns the modification timestamp of the entry.
func (f *FileEntry) ModTime() time.Time {
	if f == nil {
		return time.Time{}
	}
	return f.modTime
}

// Extension returns the lower-case file extension (including leading dot, e.g. ".go").
func (f *FileEntry) Extension() string {
	if f == nil {
		return ""
	}
	return f.extension
}

// Language returns the detected registered Language descriptor, or nil if unknown.
func (f *FileEntry) Language() *language.Language {
	if f == nil {
		return nil
	}
	return f.language
}

// LanguageID returns the registered language ID, or "unknown" if not classified.
func (f *FileEntry) LanguageID() string {
	if f == nil || f.language == nil {
		return "unknown"
	}
	return f.language.ID()
}

// LanguageName returns the registered language name, or "Unknown" if not classified.
func (f *FileEntry) LanguageName() string {
	if f == nil || f.language == nil {
		return "Unknown"
	}
	return f.language.Name()
}

// IsHidden reports whether the entry is a hidden file or dotfile.
func (f *FileEntry) IsHidden() bool {
	if f == nil {
		return false
	}
	return f.isHidden
}

// IsSymlink reports whether the entry is a symbolic link.
func (f *FileEntry) IsSymlink() bool {
	if f == nil {
		return false
	}
	return f.isSymlink
}

// IsIgnored reports whether the entry was matched by an active ignore rule.
func (f *FileEntry) IsIgnored() bool {
	if f == nil {
		return false
	}
	return f.isIgnored
}

// String returns a human-readable representation of the FileEntry.
func (f *FileEntry) String() string {
	if f == nil {
		return ""
	}
	lang := "unknown"
	if f.language != nil {
		lang = f.language.ID()
	}
	return fmt.Sprintf("FileEntry<%s>[size=%d, lang=%s]", f.relPath, f.size, lang)
}
