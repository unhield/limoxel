package language

import (
	"fmt"
	"strings"
)

// Language represents an immutable programming language descriptor with metadata.
type Language struct {
	id         string
	name       string
	extensions []string
	filenames  []string
	aliases    []string
}

// New constructs and validates a new immutable Language descriptor with metadata.
func New(id string, name string, extensions []string, filenames []string, aliases []string) (*Language, error) {
	cleanID := strings.TrimSpace(id)
	if cleanID == "" {
		return nil, ErrInvalidID
	}
	if strings.Contains(cleanID, " ") {
		return nil, fmt.Errorf("%w: ID cannot contain spaces", ErrInvalidID)
	}

	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		return nil, ErrInvalidName
	}

	extSet := make(map[string]struct{})
	extList := make([]string, 0, len(extensions))
	for _, ext := range extensions {
		cleaned := strings.ToLower(strings.TrimSpace(ext))
		if cleaned == "" {
			continue
		}
		if !strings.HasPrefix(cleaned, ".") {
			cleaned = "." + cleaned
		}
		if _, exists := extSet[cleaned]; !exists {
			extSet[cleaned] = struct{}{}
			extList = append(extList, cleaned)
		}
	}

	fnSet := make(map[string]struct{})
	fnList := make([]string, 0, len(filenames))
	for _, fn := range filenames {
		cleaned := strings.TrimSpace(fn)
		if cleaned == "" {
			continue
		}
		lowerFn := strings.ToLower(cleaned)
		if _, exists := fnSet[lowerFn]; !exists {
			fnSet[lowerFn] = struct{}{}
			fnList = append(fnList, cleaned)
		}
	}

	aliasSet := make(map[string]struct{})
	aliasList := make([]string, 0, len(aliases))
	for _, alias := range aliases {
		cleaned := strings.ToLower(strings.TrimSpace(alias))
		if cleaned == "" {
			continue
		}
		if _, exists := aliasSet[cleaned]; !exists {
			aliasSet[cleaned] = struct{}{}
			aliasList = append(aliasList, cleaned)
		}
	}

	return &Language{
		id:         strings.ToLower(cleanID),
		name:       cleanName,
		extensions: extList,
		filenames:  fnList,
		aliases:    aliasList,
	}, nil
}

// ID returns the canonical lower-case language identifier string.
func (l *Language) ID() string {
	if l == nil {
		return ""
	}
	return l.id
}

// Name returns the human-readable language name.
func (l *Language) Name() string {
	if l == nil {
		return ""
	}
	return l.name
}

// Extensions returns a defensive copy of the lower-case file extensions.
func (l *Language) Extensions() []string {
	if l == nil || len(l.extensions) == 0 {
		return nil
	}
	cloned := make([]string, len(l.extensions))
	copy(cloned, l.extensions)
	return cloned
}

// Filenames returns a defensive copy of exact filenames associated with the language.
func (l *Language) Filenames() []string {
	if l == nil || len(l.filenames) == 0 {
		return nil
	}
	cloned := make([]string, len(l.filenames))
	copy(cloned, l.filenames)
	return cloned
}

// Aliases returns a defensive copy of the lower-case language aliases.
func (l *Language) Aliases() []string {
	if l == nil || len(l.aliases) == 0 {
		return nil
	}
	cloned := make([]string, len(l.aliases))
	copy(cloned, l.aliases)
	return cloned
}

// String returns the formatted string representation of the Language.
func (l *Language) String() string {
	if l == nil {
		return ""
	}
	return fmt.Sprintf("Language<%s>(%s)", l.id, l.name)
}
