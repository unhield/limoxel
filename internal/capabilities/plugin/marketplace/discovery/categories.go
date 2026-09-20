package discovery

import (
	pubmarket "github.com/unhield/limoxel/plugin/marketplace"
)

// CategoryInfo provides metadata and active plugin counts for a category.
type CategoryInfo struct {
	Category    pubmarket.Category `json:"category"`
	DisplayName string             `json:"display_name"`
	PluginCount int                `json:"plugin_count"`
}

// GetCategoryBreakdown returns all registered marketplace categories with current plugin counts.
func (e *Engine) GetCategoryBreakdown() ([]CategoryInfo, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	all := e.provider.ListPlugins(true)

	counts := make(map[pubmarket.Category]int)
	for _, p := range all {
		if !p.Status.IsDiscoverable() {
			continue
		}
		for _, cat := range p.Categories {
			counts[cat]++
		}
	}

	allCats := pubmarket.AllCategories()
	result := make([]CategoryInfo, 0, len(allCats))

	for _, c := range allCats {
		result = append(result, CategoryInfo{
			Category:    c,
			DisplayName: categoryDisplayName(c),
			PluginCount: counts[c],
		})
	}

	return result, nil
}

// categoryDisplayName maps a category key to a friendly human-readable label.
func categoryDisplayName(c pubmarket.Category) string {
	switch c {
	case pubmarket.CategoryAnalysis:
		return "Code Analysis & Linting"
	case pubmarket.CategorySecurity:
		return "Security & Vulnerability Scanners"
	case pubmarket.CategoryVisualization:
		return "Visualization & Diagnostics"
	case pubmarket.CategoryIntegration:
		return "Integration & Connectors"
	case pubmarket.CategoryLanguage:
		return "Language Support & Parsers"
	case pubmarket.CategoryTools:
		return "Developer Tooling & Automation"
	case pubmarket.CategoryUtility:
		return "General Utilities & Helpers"
	default:
		return string(c)
	}
}
