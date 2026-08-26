package config

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Manager is the authoritative, thread-safe configuration lifecycle orchestrator for Limoxel.
type Manager struct {
	mu          sync.RWMutex
	options     ManagerOptions
	loader      *SourceLoader
	merger      *Merger
	validator   *Validator
	fileStore   *FileStore
	effective   *EffectiveConfig
	rawEntries  map[string]ConfigEntry
	activeModel *ConfigFileModel
	activePath  string
}

// NewManager constructs an initialized Manager with the specified options.
func NewManager(opts ...OptionFunc) (*Manager, error) {
	options := ManagerOptions{
		EnvPrefix:    "LIMOXEL_",
		WorkspaceDir: ".",
	}
	if home, err := os.UserHomeDir(); err == nil {
		options.UserHomeDir = home
	}

	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}

	m := &Manager{
		options:   options,
		loader:    NewSourceLoader(),
		merger:    NewMerger(),
		validator: NewValidator(),
		fileStore: NewFileStore(),
	}

	if err := m.Reload(); err != nil {
		return nil, err
	}

	return m, nil
}

// Reload executes full discovery, precedence resolution, and validation.
func (m *Manager) Reload() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var layers []map[string]ConfigEntry

	// 1. Built-in defaults (Precedence: 10)
	if !m.options.DisableDefaults {
		layers = append(layers, m.loader.LoadDefaults())
	}

	// 2. Configuration Files (Precedence: 20)
	var activeModel *ConfigFileModel
	var activePath string

	if !m.options.DisableFiles {
		var filesToLoad []string
		if m.options.ConfigFile != "" {
			activePath = m.options.ConfigFile
			if fileExists(m.options.ConfigFile) {
				filesToLoad = []string{m.options.ConfigFile}
			}
		} else {
			filesToLoad = m.loader.DiscoverConfigFiles(m.options.WorkspaceDir, m.options.UserHomeDir)
		}

		for _, filePath := range filesToLoad {
			fileEntries, model, err := m.loader.LoadConfigFile(filePath)
			if err != nil {
				if m.options.ConfigFile != "" {
					return err
				}
				continue
			}
			layers = append(layers, fileEntries)
			activeModel = model
			activePath = filePath
		}
	}

	// 3. Named Profiles (Precedence: 30)
	activeProfile := m.options.ActiveProfile
	if activeProfile == "" && activeModel != nil {
		activeProfile = activeModel.ActiveProfile
	}
	if activeProfile == "" {
		activeProfile = "default"
	}

	if activeModel != nil && len(activeModel.Profiles) > 0 && activeProfile != "default" {
		profileEntries, err := m.loader.LoadProfileEntries(activeProfile, activeModel, activePath)
		if err != nil {
			return err
		}
		if len(profileEntries) > 0 {
			layers = append(layers, profileEntries)
		}
	}

	// 4. Environment Variables (Precedence: 40)
	if !m.options.DisableEnv {
		envEntries := m.loader.LoadEnvironment(m.options.EnvPrefix)
		if len(envEntries) > 0 {
			layers = append(layers, envEntries)
		}
	}

	// 5. Runtime Overrides (Precedence: 50)
	if len(m.options.RuntimeOverrides) > 0 {
		runtimeEntries := m.loader.LoadRuntimeOverrides(m.options.RuntimeOverrides)
		layers = append(layers, runtimeEntries)
	}

	// Merge all layers deterministically according to precedence
	merged := m.merger.Merge(layers...)

	// Validate merged configuration before making it effective
	valResult := m.validator.Validate(merged)
	if !valResult.Valid && len(valResult.Errors) > 0 {
		return &valResult.Errors[0]
	}

	m.rawEntries = merged
	m.effective = NewEffectiveConfig(merged, activeProfile)
	m.activeModel = activeModel
	m.activePath = activePath

	return nil
}

// Effective returns the thread-safe read-only snapshot of effective configuration.
func (m *Manager) Effective() *EffectiveConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.effective
}

// Get returns the effective value for key.
func (m *Manager) Get(key string) (any, bool) {
	return m.Effective().Get(key)
}

// Set updates a configuration key in the active configuration file or target file.
func (m *Manager) Set(key string, val any, targetPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cleanKey := strings.ToLower(strings.TrimSpace(key))
	parts := strings.SplitN(cleanKey, ".", 2)
	if len(parts) != 2 {
		return &ConfigError{
			Code:    ErrCodeInvalidKey,
			Key:     key,
			Message: "key must be in 'section.property' format (e.g., 'output.format')",
		}
	}

	section := parts[0]
	prop := parts[1]

	destPath := targetPath
	if destPath == "" {
		destPath = m.activePath
	}
	if destPath == "" {
		destPath = filepath.Join(m.options.WorkspaceDir, ".limoxel.yaml")
	}

	model := m.activeModel
	if model == nil {
		model = &ConfigFileModel{
			Version: "1.0.0",
		}
	}

	switch section {
	case "general":
		if model.General == nil {
			model.General = make(map[string]any)
		}
		model.General[prop] = val
	case "repository":
		if model.Repository == nil {
			model.Repository = make(map[string]any)
		}
		model.Repository[prop] = val
	case "analysis":
		if model.Analysis == nil {
			model.Analysis = make(map[string]any)
		}
		model.Analysis[prop] = val
	case "output":
		if model.Output == nil {
			model.Output = make(map[string]any)
		}
		model.Output[prop] = val
	case "logging":
		if model.Logging == nil {
			model.Logging = make(map[string]any)
		}
		model.Logging[prop] = val
	case "performance":
		if model.Performance == nil {
			model.Performance = make(map[string]any)
		}
		model.Performance[prop] = val
	default:
		if model.Custom == nil {
			model.Custom = make(map[string]any)
		}
		model.Custom[cleanKey] = val
	}

	ext := strings.TrimPrefix(filepath.Ext(destPath), ".")
	if ext == "" {
		ext = "yaml"
	}

	if err := m.fileStore.SaveFile(destPath, model, ext); err != nil {
		return err
	}

	m.activeModel = model
	m.activePath = destPath

	// Re-resolve
	m.mu.Unlock()
	err := m.Reload()
	m.mu.Lock()
	return err
}

// Unset removes a configuration key from the active configuration file.
func (m *Manager) Unset(key string, targetPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cleanKey := strings.ToLower(strings.TrimSpace(key))
	parts := strings.SplitN(cleanKey, ".", 2)
	if len(parts) != 2 {
		return &ConfigError{
			Code:    ErrCodeInvalidKey,
			Key:     key,
			Message: "key must be in 'section.property' format",
		}
	}

	destPath := targetPath
	if destPath == "" {
		destPath = m.activePath
	}
	if destPath == "" {
		return &ConfigError{
			Code:    ErrCodeFileNotFound,
			Message: "no active configuration file to modify",
		}
	}

	if m.activeModel == nil {
		return nil
	}

	section := parts[0]
	prop := parts[1]

	switch section {
	case "general":
		delete(m.activeModel.General, prop)
	case "repository":
		delete(m.activeModel.Repository, prop)
	case "analysis":
		delete(m.activeModel.Analysis, prop)
	case "output":
		delete(m.activeModel.Output, prop)
	case "logging":
		delete(m.activeModel.Logging, prop)
	case "performance":
		delete(m.activeModel.Performance, prop)
	default:
		delete(m.activeModel.Custom, cleanKey)
	}

	ext := strings.TrimPrefix(filepath.Ext(destPath), ".")
	if ext == "" {
		ext = "yaml"
	}

	if err := m.fileStore.SaveFile(destPath, m.activeModel, ext); err != nil {
		return err
	}

	m.mu.Unlock()
	err := m.Reload()
	m.mu.Lock()
	return err
}

// Inspect returns a sorted list of all resolved entries with source, precedence, and redacted secret values.
func (m *Manager) Inspect(redactSecrets bool) []ConfigEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]string, 0, len(m.rawEntries))
	for k := range m.rawEntries {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	entries := make([]ConfigEntry, 0, len(keys))
	for _, k := range keys {
		entry := m.rawEntries[k]
		if redactSecrets && (entry.IsSecret || IsSecretKey(entry.Key)) {
			entry.Value = MaskedValueConstant
		}
		entries = append(entries, entry)
	}
	return entries
}

// CreateProfile adds or updates a named profile in the configuration file.
func (m *Manager) CreateProfile(profile Profile, targetPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cleanName := strings.ToLower(strings.TrimSpace(profile.Name))
	if cleanName == "" {
		return &ConfigError{
			Code:    ErrCodeInvalidKey,
			Message: "profile name cannot be empty",
		}
	}
	profile.Name = cleanName

	destPath := targetPath
	if destPath == "" {
		destPath = m.activePath
	}
	if destPath == "" {
		destPath = filepath.Join(m.options.WorkspaceDir, ".limoxel.yaml")
	}

	if m.activeModel == nil {
		m.activeModel = &ConfigFileModel{
			Version: "1.0.0",
		}
	}
	if m.activeModel.Profiles == nil {
		m.activeModel.Profiles = make(map[string]Profile)
	}

	m.activeModel.Profiles[cleanName] = profile

	ext := strings.TrimPrefix(filepath.Ext(destPath), ".")
	if ext == "" {
		ext = "yaml"
	}

	if err := m.fileStore.SaveFile(destPath, m.activeModel, ext); err != nil {
		return err
	}

	m.activePath = destPath
	m.mu.Unlock()
	err := m.Reload()
	m.mu.Lock()
	return err
}

// DeleteProfile removes a named profile from configuration.
func (m *Manager) DeleteProfile(profileName string, targetPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cleanName := strings.ToLower(strings.TrimSpace(profileName))
	if cleanName == "" || m.activeModel == nil || len(m.activeModel.Profiles) == 0 {
		return nil
	}

	delete(m.activeModel.Profiles, cleanName)

	destPath := targetPath
	if destPath == "" {
		destPath = m.activePath
	}
	if destPath == "" {
		destPath = filepath.Join(m.options.WorkspaceDir, ".limoxel.yaml")
	}

	ext := strings.TrimPrefix(filepath.Ext(destPath), ".")
	if ext == "" {
		ext = "yaml"
	}

	if err := m.fileStore.SaveFile(destPath, m.activeModel, ext); err != nil {
		return err
	}

	m.activePath = destPath
	m.mu.Unlock()
	err := m.Reload()
	m.mu.Lock()
	return err
}

// ListProfiles returns all profile names defined in active configuration.
func (m *Manager) ListProfiles() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := []string{"default"}
	if m.activeModel != nil && len(m.activeModel.Profiles) > 0 {
		for p := range m.activeModel.Profiles {
			names = append(names, p)
		}
	}
	sort.Strings(names)
	return names
}

// ValidateFile parses and validates an external configuration file without making it active.
func (m *Manager) ValidateFile(path string) ValidationResult {
	fileEntries, _, err := m.loader.LoadConfigFile(path)
	if err != nil {
		return ValidationResult{
			Valid: false,
			Errors: []ConfigError{
				{
					Code:    ErrCodeFileMalformed,
					Message: err.Error(),
					Source:  path,
				},
			},
		}
	}

	defaults := m.loader.LoadDefaults()
	merged := m.merger.Merge(defaults, fileEntries)
	return m.validator.Validate(merged)
}
