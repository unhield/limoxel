package config

import (
	"fmt"
	"sort"
	"strings"
)

// Merger executes deterministic, precedence-based merging across configuration entry sets.
type Merger struct{}

// NewMerger creates a new Merger instance.
func NewMerger() *Merger {
	return &Merger{}
}

// Merge combines multiple layers of configuration entries.
// Higher precedence layers overwrite values from lower precedence layers.
// If two layers have equal precedence, later layers in layerOrder take precedence.
func (m *Merger) Merge(layers ...map[string]ConfigEntry) map[string]ConfigEntry {
	result := make(map[string]ConfigEntry)

	for _, layer := range layers {
		if layer == nil {
			continue
		}

		// Sort keys to ensure deterministic processing
		sortedKeys := make([]string, 0, len(layer))
		for k := range layer {
			sortedKeys = append(sortedKeys, k)
		}
		sort.Strings(sortedKeys)

		for _, k := range sortedKeys {
			incoming := layer[k]
			cleanKey := strings.ToLower(strings.TrimSpace(k))
			incoming.Key = cleanKey

			existing, exists := result[cleanKey]
			if !exists {
				result[cleanKey] = incoming
				continue
			}

			// Precedence check: incoming overrides existing if incoming.Precedence >= existing.Precedence
			if incoming.Precedence >= existing.Precedence {
				// Deep merge for maps if both are map type
				if existing.Type == TypeMap && incoming.Type == TypeMap {
					mergedMap := mergeMapValues(existing.Value, incoming.Value)
					incoming.Value = mergedMap
				}
				result[cleanKey] = incoming
			}
		}
	}

	return result
}

func mergeMapValues(base, overlay any) any {
	baseMap, baseOk := base.(map[string]any)
	overlayMap, overlayOk := overlay.(map[string]any)

	if !baseOk || !overlayOk {
		return overlay
	}

	res := make(map[string]any, len(baseMap)+len(overlayMap))
	for k, v := range baseMap {
		res[k] = v
	}
	for k, v := range overlayMap {
		if subBase, hasSub := res[k]; hasSub {
			if subBaseMap, subBaseOk := subBase.(map[string]any); subBaseOk {
				if subOverlayMap, subOverlayOk := v.(map[string]any); subOverlayOk {
					res[k] = mergeMapValues(subBaseMap, subOverlayMap)
					continue
				}
			}
		}
		res[k] = v
	}
	return res
}

// Diff compares two configuration maps and returns added, modified, and removed keys.
func (m *Merger) Diff(oldEntries, newEntries map[string]ConfigEntry) (added, modified, removed []string) {
	for k, newV := range newEntries {
		if oldV, exists := oldEntries[k]; exists {
			if fmt.Sprint(oldV.Value) != fmt.Sprint(newV.Value) {
				modified = append(modified, k)
			}
		} else {
			added = append(added, k)
		}
	}
	for k := range oldEntries {
		if _, exists := newEntries[k]; !exists {
			removed = append(removed, k)
		}
	}
	sort.Strings(added)
	sort.Strings(modified)
	sort.Strings(removed)
	return added, modified, removed
}
