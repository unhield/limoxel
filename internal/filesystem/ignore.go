package filesystem

import (
	"path/filepath"
	"sort"
	"strings"
)

// DefaultIgnoreRules returns the canonical built-in list of ignored system and metadata names.
func DefaultIgnoreRules() []string {
	return []string{
		".git",
		".svn",
		".hg",
		".idea",
		".vscode",
		"node_modules",
		"vendor",
		".cache",
		".DS_Store",
		"Thumbs.db",
	}
}

// Ignorer evaluates whether a given filesystem path should be ignored.
type Ignorer struct {
	rules map[string]struct{}
	list  []string
}

// NewIgnorer constructs a new immutable Ignorer with DefaultIgnoreRules and optional additional rules.
func NewIgnorer(additionalRules ...string) *Ignorer {
	ruleSet := make(map[string]struct{})

	for _, rule := range DefaultIgnoreRules() {
		clean := strings.TrimSpace(rule)
		if clean != "" {
			ruleSet[clean] = struct{}{}
		}
	}

	for _, rule := range additionalRules {
		clean := strings.TrimSpace(rule)
		if clean != "" {
			ruleSet[clean] = struct{}{}
		}
	}

	ruleList := make([]string, 0, len(ruleSet))
	for r := range ruleSet {
		ruleList = append(ruleList, r)
	}
	sort.Strings(ruleList)

	return &Ignorer{
		rules: ruleSet,
		list:  ruleList,
	}
}

// ShouldIgnore reports whether path matches any configured ignore rule segment.
func (ig *Ignorer) ShouldIgnore(path string) bool {
	if ig == nil || path == "" {
		return false
	}

	cleanPath := filepath.Clean(path)
	segments := strings.Split(cleanPath, string(filepath.Separator))

	for _, seg := range segments {
		segClean := strings.TrimSpace(seg)
		if segClean == "" {
			continue
		}
		if _, ignored := ig.rules[segClean]; ignored {
			return true
		}
	}

	base := filepath.Base(cleanPath)
	if _, ignored := ig.rules[base]; ignored {
		return true
	}

	return false
}

// Rules returns a defensive copy of all active ignore rules in sorted order.
func (ig *Ignorer) Rules() []string {
	if ig == nil || len(ig.list) == 0 {
		return nil
	}
	cloned := make([]string, len(ig.list))
	copy(cloned, ig.list)
	return cloned
}
