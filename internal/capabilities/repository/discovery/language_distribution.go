package discovery

import (
	"fmt"
	"sort"
)

// LanguageDistribution represents aggregated file statistics for a single detected language.
type LanguageDistribution struct {
	id         string
	name       string
	fileCount  int
	totalBytes int64
	percentage float64
	extensions []string
}

// NewLanguageDistribution creates an immutable LanguageDistribution record.
func NewLanguageDistribution(
	id string,
	name string,
	fileCount int,
	totalBytes int64,
	percentage float64,
	extensions []string,
) *LanguageDistribution {
	exts := make([]string, len(extensions))
	copy(exts, extensions)
	sort.Strings(exts)

	return &LanguageDistribution{
		id:         id,
		name:       name,
		fileCount:  fileCount,
		totalBytes: totalBytes,
		percentage: percentage,
		extensions: exts,
	}
}

// LanguageID returns the language identifier string.
func (l *LanguageDistribution) LanguageID() string {
	if l == nil {
		return ""
	}
	return l.id
}

// LanguageName returns the human-readable language name.
func (l *LanguageDistribution) LanguageName() string {
	if l == nil {
		return ""
	}
	return l.name
}

// FileCount returns the number of files detected for this language.
func (l *LanguageDistribution) FileCount() int {
	if l == nil {
		return 0
	}
	return l.fileCount
}

// TotalBytes returns the sum of file sizes in bytes for this language.
func (l *LanguageDistribution) TotalBytes() int64 {
	if l == nil {
		return 0
	}
	return l.totalBytes
}

// Percentage returns the percentage of total repository files belonging to this language.
func (l *LanguageDistribution) Percentage() float64 {
	if l == nil {
		return 0.0
	}
	return l.percentage
}

// Extensions returns a defensive copy of sorted file extensions associated with this distribution.
func (l *LanguageDistribution) Extensions() []string {
	if l == nil || len(l.extensions) == 0 {
		return nil
	}
	cloned := make([]string, len(l.extensions))
	copy(cloned, l.extensions)
	return cloned
}

// String returns a human-readable summary of the language distribution.
func (l *LanguageDistribution) String() string {
	if l == nil {
		return ""
	}
	return fmt.Sprintf("%s: %d files (%.1f%%, %d bytes)", l.name, l.fileCount, l.percentage, l.totalBytes)
}
