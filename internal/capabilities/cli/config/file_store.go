package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/cli/reporting"
)

// FileStore coordinates reading and atomic writing of configuration files across YAML, JSON, and TOML formats.
type FileStore struct {
	structured *reporting.StructuredExporter
}

// NewFileStore constructs an initialized FileStore.
func NewFileStore() *FileStore {
	return &FileStore{
		structured: reporting.NewStructuredExporter(),
	}
}

// LoadFile reads and parses a configuration file from disk into a ConfigFileModel.
func (s *FileStore) LoadFile(path string) (*ConfigFileModel, error) {
	cleanPath := filepath.Clean(strings.TrimSpace(path))
	if cleanPath == "" {
		return nil, &ConfigError{
			Code:    ErrCodeFileNotFound,
			Message: "configuration file path is empty",
		}
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &ConfigError{
				Code:    ErrCodeFileNotFound,
				Message: fmt.Sprintf("configuration file %q does not exist", cleanPath),
				Cause:   err,
			}
		}
		return nil, &ConfigError{
			Code:    ErrCodePermissionDenied,
			Message: fmt.Sprintf("failed to read configuration file %q", cleanPath),
			Cause:   err,
		}
	}

	ext := strings.ToLower(filepath.Ext(cleanPath))
	var model ConfigFileModel

	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &model); err != nil {
			return nil, &ConfigError{
				Code:    ErrCodeFileMalformed,
				Message: fmt.Sprintf("invalid JSON in configuration file %q", cleanPath),
				Cause:   err,
			}
		}
	case ".yaml", ".yml":
		if err := parseYAMLConfig(data, &model); err != nil {
			return nil, &ConfigError{
				Code:    ErrCodeFileMalformed,
				Message: fmt.Sprintf("invalid YAML in configuration file %q", cleanPath),
				Cause:   err,
			}
		}
	case ".toml":
		if err := parseTOMLConfig(data, &model); err != nil {
			return nil, &ConfigError{
				Code:    ErrCodeFileMalformed,
				Message: fmt.Sprintf("invalid TOML in configuration file %q", cleanPath),
				Cause:   err,
			}
		}
	default:
		// Attempt JSON first, then YAML
		if errJSON := json.Unmarshal(data, &model); errJSON == nil {
			return &model, nil
		}
		if errYAML := parseYAMLConfig(data, &model); errYAML == nil {
			return &model, nil
		}
		return nil, &ConfigError{
			Code:    ErrCodeFileMalformed,
			Message: fmt.Sprintf("unrecognized configuration format for file %q", cleanPath),
		}
	}

	return &model, nil
}

// SaveFile atomically serializes and writes a ConfigFileModel to destinationPath.
func (s *FileStore) SaveFile(path string, model *ConfigFileModel, format string) error {
	cleanPath := filepath.Clean(strings.TrimSpace(path))
	if cleanPath == "" {
		return &ConfigError{
			Code:    ErrCodeInvalidKey,
			Message: "destination configuration path is empty",
		}
	}

	dir := filepath.Dir(cleanPath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return &ConfigError{
				Code:    ErrCodePermissionDenied,
				Message: fmt.Sprintf("failed to create directory %q", dir),
				Cause:   err,
			}
		}
	}

	var data []byte
	var err error

	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		data, err = json.MarshalIndent(model, "", "  ")
	case "toml":
		var buf bytes.Buffer
		err = s.structured.Export(reporting.FormatTOML, model, &buf)
		data = buf.Bytes()
	case "yaml", "yml", "":
		data, err = serializeYAML(model)
	default:
		return &ConfigError{
			Code:    ErrCodeInvalidValue,
			Message: fmt.Sprintf("unsupported configuration format %q for saving", format),
		}
	}

	if err != nil {
		return &ConfigError{
			Code:    ErrCodeFileMalformed,
			Message: "failed to serialize configuration model",
			Cause:   err,
		}
	}

	// Atomic write via temp file
	tmpFile := cleanPath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return &ConfigError{
			Code:    ErrCodePermissionDenied,
			Message: fmt.Sprintf("failed to write temporary config file %q", tmpFile),
			Cause:   err,
		}
	}

	if err := os.Rename(tmpFile, cleanPath); err != nil {
		_ = os.Remove(tmpFile)
		if errDirect := os.WriteFile(cleanPath, data, 0644); errDirect != nil {
			return &ConfigError{
				Code:    ErrCodePermissionDenied,
				Message: fmt.Sprintf("failed to atomically write config file %q", cleanPath),
				Cause:   errDirect,
			}
		}
	}

	return nil
}

func serializeYAML(model *ConfigFileModel) ([]byte, error) {
	var buf bytes.Buffer
	if model.Version != "" {
		fmt.Fprintf(&buf, "version: %q\n", model.Version)
	}
	if model.ActiveProfile != "" {
		fmt.Fprintf(&buf, "active_profile: %q\n", model.ActiveProfile)
	}

	writeSection := func(name string, m map[string]any) {
		if len(m) == 0 {
			return
		}
		fmt.Fprintf(&buf, "%s:\n", name)
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := m[k]
			switch val := v.(type) {
			case []string:
				if len(val) == 0 {
					fmt.Fprintf(&buf, "  %s: []\n", k)
				} else {
					fmt.Fprintf(&buf, "  %s: [%s]\n", k, strings.Join(quoteSlice(val), ", "))
				}
			case []any:
				if len(val) == 0 {
					fmt.Fprintf(&buf, "  %s: []\n", k)
				} else {
					strVals := make([]string, len(val))
					for idx, it := range val {
						strVals[idx] = fmt.Sprintf("%v", it)
					}
					fmt.Fprintf(&buf, "  %s: [%s]\n", k, strings.Join(quoteSlice(strVals), ", "))
				}
			case string:
				fmt.Fprintf(&buf, "  %s: %q\n", k, val)
			case bool, int, int64, float64:
				fmt.Fprintf(&buf, "  %s: %v\n", k, val)
			default:
				fmt.Fprintf(&buf, "  %s: %v\n", k, val)
			}
		}
	}

	writeSection("general", model.General)
	writeSection("repository", model.Repository)
	writeSection("analysis", model.Analysis)
	writeSection("output", model.Output)
	writeSection("logging", model.Logging)
	writeSection("performance", model.Performance)

	if len(model.Profiles) > 0 {
		buf.WriteString("profiles:\n")
		pNames := make([]string, 0, len(model.Profiles))
		for p := range model.Profiles {
			pNames = append(pNames, p)
		}
		sort.Strings(pNames)
		for _, pName := range pNames {
			prof := model.Profiles[pName]
			fmt.Fprintf(&buf, "  %s:\n", pName)
			if prof.Description != "" {
				fmt.Fprintf(&buf, "    description: %q\n", prof.Description)
			}
			if prof.Inherits != "" {
				fmt.Fprintf(&buf, "    inherits: %q\n", prof.Inherits)
			}
			if len(prof.Values) > 0 {
				buf.WriteString("    values:\n")
				vKeys := make([]string, 0, len(prof.Values))
				for vk := range prof.Values {
					vKeys = append(vKeys, vk)
				}
				sort.Strings(vKeys)
				for _, vk := range vKeys {
					fmt.Fprintf(&buf, "      %s: %v\n", vk, prof.Values[vk])
				}
			}
		}
	}

	if len(model.Custom) > 0 {
		writeSection("custom", model.Custom)
	}

	return buf.Bytes(), nil
}

func quoteSlice(items []string) []string {
	res := make([]string, len(items))
	for i, it := range items {
		res[i] = fmt.Sprintf("%q", it)
	}
	return res
}

// Minimal YAML parser for standard Limoxel configuration structures
func parseYAMLConfig(data []byte, model *ConfigFileModel) error {
	// Limoxel config files use simple hierarchical mappings:
	// section:
	//   key: value
	var genericMap map[string]any
	if err := json.Unmarshal(data, &genericMap); err == nil {
		// If JSON-compatible YAML, unmarshal directly
		return mapToConfigModel(genericMap, model)
	}

	// Line-based parser for standard YAML key-value trees
	genericMap = parseSimpleYAML(string(data))
	return mapToConfigModel(genericMap, model)
}

func parseSimpleYAML(content string) map[string]any {
	result := make(map[string]any)
	lines := strings.Split(content, "\n")

	type stackItem struct {
		indent int
		m      map[string]any
	}
	stack := []stackItem{{indent: -1, m: result}}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		indent := 0
		for _, r := range line {
			if r == ' ' {
				indent++
			} else if r == '\t' {
				indent += 2
			} else {
				break
			}
		}

		for len(stack) > 1 && stack[len(stack)-1].indent >= indent {
			stack = stack[:len(stack)-1]
		}

		parent := stack[len(stack)-1].m

		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		if val == "" {
			childMap := make(map[string]any)
			parent[key] = childMap
			stack = append(stack, stackItem{indent: indent, m: childMap})
		} else {
			parent[key] = parseScalar(val)
		}
	}

	return result
}

func parseTOMLConfig(data []byte, model *ConfigFileModel) error {
	genericMap := make(map[string]any)
	lines := strings.Split(string(data), "\n")
	var currentTable string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			currentTable = strings.TrimSpace(trimmed[1 : len(trimmed)-1])
			if _, exists := genericMap[currentTable]; !exists {
				genericMap[currentTable] = make(map[string]any)
			}
		} else {
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 {
				k := strings.TrimSpace(parts[0])
				v := strings.TrimSpace(parts[1])
				val := parseScalar(v)
				if currentTable != "" {
					if tbl, ok := genericMap[currentTable].(map[string]any); ok {
						tbl[k] = val
					}
				} else {
					genericMap[k] = val
				}
			}
		}
	}

	return mapToConfigModel(genericMap, model)
}

func mapToConfigModel(m map[string]any, model *ConfigFileModel) error {
	if v, ok := m["version"].(string); ok {
		model.Version = v
	}
	if p, ok := m["active_profile"].(string); ok {
		model.ActiveProfile = p
	}
	if g, ok := m["general"].(map[string]any); ok {
		model.General = g
	}
	if r, ok := m["repository"].(map[string]any); ok {
		model.Repository = r
	}
	if a, ok := m["analysis"].(map[string]any); ok {
		model.Analysis = a
	}
	if o, ok := m["output"].(map[string]any); ok {
		model.Output = o
	}
	if l, ok := m["logging"].(map[string]any); ok {
		model.Logging = l
	}
	if p, ok := m["performance"].(map[string]any); ok {
		model.Performance = p
	}
	if profs, ok := m["profiles"].(map[string]any); ok {
		model.Profiles = make(map[string]Profile)
		for pName, pData := range profs {
			if pMap, okMap := pData.(map[string]any); okMap {
				prof := Profile{
					Name:   pName,
					Values: make(map[string]any),
				}
				if desc, okDesc := pMap["description"].(string); okDesc {
					prof.Description = desc
				}
				if inh, okInh := pMap["inherits"].(string); okInh {
					prof.Inherits = inh
				}
				if vals, okVals := pMap["values"].(map[string]any); okVals {
					prof.Values = vals
				}
				model.Profiles[pName] = prof
			}
		}
	}
	if c, ok := m["custom"].(map[string]any); ok {
		model.Custom = c
	}
	return nil
}

func parseScalar(val string) any {
	clean := strings.Trim(strings.TrimSpace(val), "\"'`")
	if clean == "true" {
		return true
	}
	if clean == "false" {
		return false
	}
	var i int
	if _, err := fmt.Sscanf(clean, "%d", &i); err == nil && !strings.Contains(clean, ".") {
		return i
	}
	var f float64
	if _, err := fmt.Sscanf(clean, "%f", &f); err == nil && strings.Contains(clean, ".") {
		return f
	}
	if strings.HasPrefix(clean, "[") && strings.HasSuffix(clean, "]") {
		inner := strings.TrimSpace(clean[1 : len(clean)-1])
		if inner == "" {
			return []string{}
		}
		items := strings.Split(inner, ",")
		res := make([]string, len(items))
		for idx, it := range items {
			res[idx] = strings.Trim(strings.TrimSpace(it), "\"'`")
		}
		return res
	}
	return clean
}
