package config

import (
	"os"
	"path/filepath"
	"strings"
)

// SourceLoader provides abstraction for loading configuration key-value entries from various origins.
type SourceLoader struct {
	fileStore *FileStore
}

// NewSourceLoader creates a new SourceLoader.
func NewSourceLoader() *SourceLoader {
	return &SourceLoader{
		fileStore: NewFileStore(),
	}
}

// LoadDefaults returns the standard baseline configuration entries.
func (l *SourceLoader) LoadDefaults() map[string]ConfigEntry {
	return GetDefaultEntries()
}

// LoadConfigFile loads configuration from a specific file path and converts it into flat key ConfigEntries.
func (l *SourceLoader) LoadConfigFile(path string) (map[string]ConfigEntry, *ConfigFileModel, error) {
	model, err := l.fileStore.LoadFile(path)
	if err != nil {
		return nil, nil, err
	}

	entries := make(map[string]ConfigEntry)
	flattenSection("general", model.General, path, entries)
	flattenSection("repository", model.Repository, path, entries)
	flattenSection("analysis", model.Analysis, path, entries)
	flattenSection("output", model.Output, path, entries)
	flattenSection("logging", model.Logging, path, entries)
	flattenSection("performance", model.Performance, path, entries)
	flattenSection("custom", model.Custom, path, entries)

	if model.Version != "" {
		entries["general.version"] = ConfigEntry{
			Key:        "general.version",
			Value:      model.Version,
			Type:       TypeString,
			Source:     SourceFile,
			SourcePath: path,
			Precedence: PrecedenceFile,
		}
	}
	if model.ActiveProfile != "" {
		entries["general.active_profile"] = ConfigEntry{
			Key:        "general.active_profile",
			Value:      model.ActiveProfile,
			Type:       TypeString,
			Source:     SourceFile,
			SourcePath: path,
			Precedence: PrecedenceFile,
		}
	}

	return entries, model, nil
}

// DiscoverConfigFiles searches standard locations for Limoxel configuration files.
func (l *SourceLoader) DiscoverConfigFiles(workspaceDir, userHomeDir string) []string {
	var discovered []string

	// 1. User home configuration
	if userHomeDir != "" {
		userCandidates := []string{
			filepath.Join(userHomeDir, ".limoxel", "config.yaml"),
			filepath.Join(userHomeDir, ".limoxel", "config.json"),
			filepath.Join(userHomeDir, ".limoxel", "config.toml"),
			filepath.Join(userHomeDir, ".limoxelrc"),
		}
		for _, cand := range userCandidates {
			if fileExists(cand) {
				discovered = append(discovered, cand)
				break
			}
		}
	}

	// 2. Workspace configuration
	if workspaceDir != "" {
		wsCandidates := []string{
			filepath.Join(workspaceDir, ".limoxel.yaml"),
			filepath.Join(workspaceDir, ".limoxel.yml"),
			filepath.Join(workspaceDir, ".limoxel.json"),
			filepath.Join(workspaceDir, ".limoxel.toml"),
			filepath.Join(workspaceDir, ".limoxel", "config.yaml"),
			filepath.Join(workspaceDir, ".limoxel", "config.json"),
			filepath.Join(workspaceDir, ".limoxel", "config.toml"),
			filepath.Join(workspaceDir, "limoxel.config.yaml"),
		}
		for _, cand := range wsCandidates {
			if fileExists(cand) {
				discovered = append(discovered, cand)
				break
			}
		}
	}

	return discovered
}

// LoadProfileEntries extracts configuration entries for the requested profile name from model.
func (l *SourceLoader) LoadProfileEntries(profileName string, model *ConfigFileModel, sourcePath string) (map[string]ConfigEntry, error) {
	if model == nil || len(model.Profiles) == 0 {
		return nil, nil
	}

	targetProfile, exists := model.Profiles[profileName]
	if !exists {
		return nil, &ConfigError{
			Code:    ErrCodeProfileNotFound,
			Message: "profile not found",
			Source:  profileName,
		}
	}

	entries := make(map[string]ConfigEntry)

	// If inherits from another profile, load parent first
	if targetProfile.Inherits != "" && targetProfile.Inherits != profileName {
		parentEntries, err := l.LoadProfileEntries(targetProfile.Inherits, model, sourcePath)
		if err != nil {
			return nil, err
		}
		for k, v := range parentEntries {
			entries[k] = v
		}
	}

	// Apply current profile values
	for k, v := range targetProfile.Values {
		cleanKey := strings.ToLower(strings.TrimSpace(k))
		entries[cleanKey] = ConfigEntry{
			Key:        cleanKey,
			Value:      v,
			Type:       inferValueType(v),
			Source:     SourceProfile,
			SourcePath: sourcePath,
			Profile:    profileName,
			Precedence: PrecedenceProfile,
			IsSecret:   IsSecretKey(cleanKey),
		}
	}

	return entries, nil
}

// LoadEnvironment scans the process environment for variables with the specified prefix (default: "LIMOXEL_").
func (l *SourceLoader) LoadEnvironment(prefix string) map[string]ConfigEntry {
	if prefix == "" {
		prefix = "LIMOXEL_"
	}
	prefix = strings.ToUpper(prefix)

	entries := make(map[string]ConfigEntry)

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		envKey := parts[0]
		envVal := parts[1]

		if strings.HasPrefix(strings.ToUpper(envKey), prefix) {
			configKey := envKeyToConfigKey(envKey, prefix)
			if configKey == "" {
				continue
			}
			parsedVal := parseScalar(envVal)
			entries[configKey] = ConfigEntry{
				Key:        configKey,
				Value:      parsedVal,
				Type:       inferValueType(parsedVal),
				Source:     SourceEnv,
				SourcePath: envKey,
				Precedence: PrecedenceEnv,
				IsSecret:   IsSecretKey(configKey) || IsSecretKey(envKey),
			}
		}
	}

	return entries
}

// LoadRuntimeOverrides converts a raw key-value map into runtime ConfigEntries.
func (l *SourceLoader) LoadRuntimeOverrides(overrides map[string]any) map[string]ConfigEntry {
	entries := make(map[string]ConfigEntry, len(overrides))
	for k, v := range overrides {
		cleanKey := strings.ToLower(strings.TrimSpace(k))
		entries[cleanKey] = ConfigEntry{
			Key:        cleanKey,
			Value:      v,
			Type:       inferValueType(v),
			Source:     SourceRuntime,
			Precedence: PrecedenceRuntime,
			IsSecret:   IsSecretKey(cleanKey),
		}
	}
	return entries
}

func envKeyToConfigKey(envKey, prefix string) string {
	raw := strings.TrimPrefix(strings.ToUpper(envKey), prefix)
	raw = strings.ToLower(raw)

	// Common standard translations
	switch raw {
	case "output_format", "format":
		return "output.format"
	case "output_color", "color", "no_color":
		return "output.color"
	case "output_theme", "theme":
		return "output.theme"
	case "repository_root", "root", "repo":
		return "repository.root"
	case "logging_level", "log_level", "level":
		return "logging.level"
	case "logging_format", "log_format":
		return "logging.format"
	case "logging_file", "log_file":
		return "logging.file"
	case "performance_workers", "workers":
		return "performance.workers"
	case "performance_timeout", "timeout":
		return "performance.timeout_seconds"
	case "profile", "active_profile":
		return "general.active_profile"
	case "strict_mode", "analysis_strict":
		return "analysis.strict_mode"
	}

	// General dot substitution: SECTION_KEY -> section.key
	parts := strings.SplitN(raw, "_", 2)
	if len(parts) == 2 {
		return parts[0] + "." + parts[1]
	}
	return raw
}

func flattenSection(section string, data map[string]any, sourcePath string, entries map[string]ConfigEntry) {
	if data == nil {
		return
	}
	for k, v := range data {
		fullKey := section + "." + strings.ToLower(strings.TrimSpace(k))
		entries[fullKey] = ConfigEntry{
			Key:        fullKey,
			Value:      v,
			Type:       inferValueType(v),
			Source:     SourceFile,
			SourcePath: sourcePath,
			Precedence: PrecedenceFile,
			IsSecret:   IsSecretKey(fullKey),
		}
	}
}

func inferValueType(v any) ValueType {
	switch v.(type) {
	case bool:
		return TypeBool
	case int, int64:
		return TypeInt
	case float64:
		return TypeFloat
	case []string, []any:
		return TypeSlice
	case map[string]any:
		return TypeMap
	default:
		return TypeString
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
