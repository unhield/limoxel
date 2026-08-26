package config

import (
	"fmt"
	"strings"
	"time"
)

// PrecedenceLevel defines the rank in the precedence hierarchy. Higher numerical value overrides lower.
type PrecedenceLevel int

const (
	PrecedenceDefault PrecedenceLevel = 10
	PrecedenceFile    PrecedenceLevel = 20
	PrecedenceProfile PrecedenceLevel = 30
	PrecedenceEnv     PrecedenceLevel = 40
	PrecedenceRuntime PrecedenceLevel = 50
)

func (p PrecedenceLevel) String() string {
	switch p {
	case PrecedenceDefault:
		return "default"
	case PrecedenceFile:
		return "file"
	case PrecedenceProfile:
		return "profile"
	case PrecedenceEnv:
		return "environment"
	case PrecedenceRuntime:
		return "runtime"
	default:
		return "unknown"
	}
}

// SourceType represents the origin type of a configuration value.
type SourceType string

const (
	SourceDefault SourceType = "default"
	SourceFile    SourceType = "file"
	SourceProfile SourceType = "profile"
	SourceEnv     SourceType = "env"
	SourceRuntime SourceType = "runtime"
)

// ValueType classifies supported primitive and complex configuration types.
type ValueType string

const (
	TypeString   ValueType = "string"
	TypeBool     ValueType = "bool"
	TypeInt      ValueType = "int"
	TypeFloat    ValueType = "float"
	TypeDuration ValueType = "duration"
	TypeSlice    ValueType = "slice"
	TypeMap      ValueType = "map"
)

// ConfigEntry represents an individual resolved configuration key-value pair with full provenance metadata.
type ConfigEntry struct {
	Key         string          `json:"key" yaml:"key" toml:"key"`
	Value       any             `json:"value" yaml:"value" toml:"value"`
	Type        ValueType       `json:"type" yaml:"type" toml:"type"`
	Source      SourceType      `json:"source" yaml:"source" toml:"source"`
	SourcePath  string          `json:"source_path,omitempty" yaml:"source_path,omitempty" toml:"source_path,omitempty"`
	Precedence  PrecedenceLevel `json:"precedence" yaml:"precedence" toml:"precedence"`
	Profile     string          `json:"profile,omitempty" yaml:"profile,omitempty" toml:"profile,omitempty"`
	IsSecret    bool            `json:"is_secret" yaml:"is_secret" toml:"is_secret"`
	IsDefault   bool            `json:"is_default" yaml:"is_default" toml:"is_default"`
	Description string          `json:"description,omitempty" yaml:"description,omitempty" toml:"description,omitempty"`
}

// Profile represents a named configuration overlay.
type Profile struct {
	Name        string         `json:"name" yaml:"name" toml:"name"`
	Description string         `json:"description,omitempty" yaml:"description,omitempty" toml:"description,omitempty"`
	Inherits    string         `json:"inherits,omitempty" yaml:"inherits,omitempty" toml:"inherits,omitempty"`
	Values      map[string]any `json:"values" yaml:"values" toml:"values"`
}

// ConfigFileModel represents the structured root of a Limoxel configuration file (.yaml, .json, .toml).
type ConfigFileModel struct {
	Version       string             `json:"version,omitempty" yaml:"version,omitempty" toml:"version,omitempty"`
	ActiveProfile string             `json:"active_profile,omitempty" yaml:"active_profile,omitempty" toml:"active_profile,omitempty"`
	General       map[string]any     `json:"general,omitempty" yaml:"general,omitempty" toml:"general,omitempty"`
	Repository    map[string]any     `json:"repository,omitempty" yaml:"repository,omitempty" toml:"repository,omitempty"`
	Analysis      map[string]any     `json:"analysis,omitempty" yaml:"analysis,omitempty" toml:"analysis,omitempty"`
	Output        map[string]any     `json:"output,omitempty" yaml:"output,omitempty" toml:"output,omitempty"`
	Logging       map[string]any     `json:"logging,omitempty" yaml:"logging,omitempty" toml:"logging,omitempty"`
	Performance   map[string]any     `json:"performance,omitempty" yaml:"performance,omitempty" toml:"performance,omitempty"`
	Profiles      map[string]Profile `json:"profiles,omitempty" yaml:"profiles,omitempty" toml:"profiles,omitempty"`
	Custom        map[string]any     `json:"custom,omitempty" yaml:"custom,omitempty" toml:"custom,omitempty"`
}

// SchemaProperty defines schema constraints for a configuration key.
type SchemaProperty struct {
	Key            string
	Type           ValueType
	Default        any
	Description    string
	Required       bool
	EnumValues     []string
	MinInt         *int
	MaxInt         *int
	IsSecret       bool
	Deprecated     bool
	DeprecationMsg string
}

// ConfigError represents a typed, structured configuration error.
type ConfigError struct {
	Code    string
	Key     string
	Message string
	Source  string
	Cause   error
}

func (e *ConfigError) Error() string {
	if e == nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("config: ")
	if e.Code != "" {
		fmt.Fprintf(&sb, "[%s] ", e.Code)
	}
	if e.Key != "" {
		fmt.Fprintf(&sb, "key %q: ", e.Key)
	}
	sb.WriteString(e.Message)
	if e.Source != "" {
		fmt.Fprintf(&sb, " (source: %s)", e.Source)
	}
	if e.Cause != nil {
		fmt.Fprintf(&sb, ": %v", e.Cause)
	}
	return sb.String()
}

func (e *ConfigError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// Standard error codes
const (
	ErrCodeInvalidKey       = "INVALID_KEY"
	ErrCodeInvalidType      = "INVALID_TYPE"
	ErrCodeInvalidValue     = "INVALID_VALUE"
	ErrCodeMissingRequired  = "MISSING_REQUIRED"
	ErrCodeFileNotFound     = "FILE_NOT_FOUND"
	ErrCodeFileMalformed    = "FILE_MALFORMED"
	ErrCodeProfileNotFound  = "PROFILE_NOT_FOUND"
	ErrCodeDeprecatedOption = "DEPRECATED_OPTION"
	ErrCodePermissionDenied = "PERMISSION_DENIED"
	ErrCodeValidationFailed = "VALIDATION_FAILED"
)

// OptionFunc allows fluent configuration of Manager / Loader.
type OptionFunc func(*ManagerOptions)

// ManagerOptions encapsulates options passed when initializing a configuration manager.
type ManagerOptions struct {
	ConfigFile       string
	ActiveProfile    string
	WorkspaceDir     string
	UserHomeDir      string
	EnvPrefix        string
	RuntimeOverrides map[string]any
	DisableEnv       bool
	DisableFiles     bool
	DisableDefaults  bool
}

// EffectiveConfig provides read-only, thread-safe, typed access to resolved configuration.
type EffectiveConfig struct {
	entries map[string]ConfigEntry
	profile string
}

// NewEffectiveConfig creates an EffectiveConfig wrapper from resolved entries.
func NewEffectiveConfig(entries map[string]ConfigEntry, profile string) *EffectiveConfig {
	cp := make(map[string]ConfigEntry, len(entries))
	for k, v := range entries {
		cp[k] = v
	}
	return &EffectiveConfig{
		entries: cp,
		profile: profile,
	}
}

// Profile returns the active profile name used to resolve this configuration.
func (c *EffectiveConfig) Profile() string {
	if c == nil {
		return ""
	}
	return c.profile
}

// Has checks if a key exists in effective configuration.
func (c *EffectiveConfig) Has(key string) bool {
	if c == nil {
		return false
	}
	_, ok := c.entries[strings.ToLower(strings.TrimSpace(key))]
	return ok
}

// Get returns the raw value and existence flag for key.
func (c *EffectiveConfig) Get(key string) (any, bool) {
	if c == nil {
		return nil, false
	}
	entry, ok := c.entries[strings.ToLower(strings.TrimSpace(key))]
	if !ok {
		return nil, false
	}
	return entry.Value, true
}

// GetEntry returns the full provenance ConfigEntry for key.
func (c *EffectiveConfig) GetEntry(key string) (ConfigEntry, bool) {
	if c == nil {
		return ConfigEntry{}, false
	}
	entry, ok := c.entries[strings.ToLower(strings.TrimSpace(key))]
	return entry, ok
}

// GetString returns the string value for key, or fallback if absent or wrong type.
func (c *EffectiveConfig) GetString(key string, fallback string) string {
	val, ok := c.Get(key)
	if !ok || val == nil {
		return fallback
	}
	if s, ok := val.(string); ok {
		return s
	}
	return fmt.Sprint(val)
}

// GetBool returns the boolean value for key, or fallback if absent.
func (c *EffectiveConfig) GetBool(key string, fallback bool) bool {
	val, ok := c.Get(key)
	if !ok || val == nil {
		return fallback
	}
	switch v := val.(type) {
	case bool:
		return v
	case string:
		vLower := strings.ToLower(strings.TrimSpace(v))
		return vLower == "true" || vLower == "1" || vLower == "yes" || vLower == "on"
	case int:
		return v != 0
	case int64:
		return v != 0
	default:
		return fallback
	}
}

// GetInt returns the integer value for key, or fallback if absent.
func (c *EffectiveConfig) GetInt(key string, fallback int) int {
	val, ok := c.Get(key)
	if !ok || val == nil {
		return fallback
	}
	switch v := val.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		var i int
		if _, err := fmt.Sscanf(v, "%d", &i); err == nil {
			return i
		}
		return fallback
	default:
		return fallback
	}
}

// GetFloat64 returns the float64 value for key, or fallback if absent.
func (c *EffectiveConfig) GetFloat64(key string, fallback float64) float64 {
	val, ok := c.Get(key)
	if !ok || val == nil {
		return fallback
	}
	switch v := val.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		var f float64
		if _, err := fmt.Sscanf(v, "%f", &f); err == nil {
			return f
		}
		return fallback
	default:
		return fallback
	}
}

// GetDuration returns the time.Duration value for key, or fallback if absent.
func (c *EffectiveConfig) GetDuration(key string, fallback time.Duration) time.Duration {
	val, ok := c.Get(key)
	if !ok || val == nil {
		return fallback
	}
	switch v := val.(type) {
	case time.Duration:
		return v
	case int:
		return time.Duration(v) * time.Millisecond
	case int64:
		return time.Duration(v) * time.Millisecond
	case string:
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
		return fallback
	default:
		return fallback
	}
}

// GetStringSlice returns the slice of strings for key, or fallback if absent.
func (c *EffectiveConfig) GetStringSlice(key string, fallback []string) []string {
	val, ok := c.Get(key)
	if !ok || val == nil {
		return fallback
	}
	switch v := val.(type) {
	case []string:
		return v
	case []any:
		res := make([]string, len(v))
		for i, elem := range v {
			res[i] = fmt.Sprint(elem)
		}
		return res
	case string:
		if strings.Contains(v, ",") {
			parts := strings.Split(v, ",")
			res := make([]string, len(parts))
			for i, p := range parts {
				res[i] = strings.TrimSpace(p)
			}
			return res
		}
		return []string{v}
	default:
		return fallback
	}
}

// AllEntries returns a map of all resolved configuration entries.
func (c *EffectiveConfig) AllEntries() map[string]ConfigEntry {
	if c == nil {
		return nil
	}
	res := make(map[string]ConfigEntry, len(c.entries))
	for k, v := range c.entries {
		res[k] = v
	}
	return res
}
