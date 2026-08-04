package language

import (
	"fmt"
	"path/filepath"
	"strings"
)

// DiscoverByFilename discovers a registered Language by matching exact filename or file extension in O(1) time.
func (r *Registry) DiscoverByFilename(filename string) (*Language, error) {
	if r == nil {
		return nil, ErrNilRegistry
	}

	clean := strings.TrimSpace(filename)
	if clean == "" {
		return nil, ErrInvalidFilename
	}

	baseName := strings.ToLower(filepath.Base(clean))

	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.state == StateTerminated {
		return nil, ErrRegistryTerminated
	}

	// 1. Try exact filename match first in O(1)
	if lang, exists := r.filenames[baseName]; exists {
		return lang, nil
	}

	// 2. Try file extension match in O(1)
	ext := filepath.Ext(baseName)
	if ext != "" {
		if lang, exists := r.extensions[ext]; exists {
			return lang, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrLanguageNotFound, clean)
}

// DiscoverByExtension discovers a registered Language by matching file extension in O(1) time.
func (r *Registry) DiscoverByExtension(ext string) (*Language, error) {
	return r.GetByExtension(ext)
}

// DiscoverByAlias discovers a registered Language by matching alias in O(1) time.
func (r *Registry) DiscoverByAlias(alias string) (*Language, error) {
	return r.GetByAlias(alias)
}
