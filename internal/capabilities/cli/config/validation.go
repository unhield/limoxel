package config

import (
	"fmt"
	"sort"
	"strings"
)

// ValidationWarning captures non-fatal configuration issues such as deprecated options.
type ValidationWarning struct {
	Key     string `json:"key" yaml:"key" toml:"key"`
	Message string `json:"message" yaml:"message" toml:"message"`
}

// ValidationResult summarizes the outcome of a configuration validation pass.
type ValidationResult struct {
	Valid    bool                `json:"valid" yaml:"valid" toml:"valid"`
	Errors   []ConfigError       `json:"errors,omitempty" yaml:"errors,omitempty" toml:"errors,omitempty"`
	Warnings []ValidationWarning `json:"warnings,omitempty" yaml:"warnings,omitempty" toml:"warnings,omitempty"`
}

// Validator validates resolved configuration entries against schema rules and constraints.
type Validator struct {
	schema map[string]SchemaProperty
}

// NewValidator constructs a Validator pre-populated with SchemaRegistry properties.
func NewValidator() *Validator {
	sMap := make(map[string]SchemaProperty, len(SchemaRegistry))
	for _, prop := range SchemaRegistry {
		sMap[prop.Key] = prop
	}
	return &Validator{
		schema: sMap,
	}
}

// Validate executes all schema checks, type bounds, and enum validations on entries.
func (v *Validator) Validate(entries map[string]ConfigEntry) ValidationResult {
	res := ValidationResult{
		Valid: true,
	}

	// Sort keys for deterministic error and warning reporting
	sortedKeys := make([]string, 0, len(entries))
	for k := range entries {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	// 1. Check existing entries against schema
	for _, k := range sortedKeys {
		entry := entries[k]
		prop, inSchema := v.schema[k]
		if !inSchema {
			// Custom keys are allowed under custom.*
			if !strings.HasPrefix(k, "custom.") {
				// Non-fatal warning or schema check
			}
			continue
		}

		// Deprecation check
		if prop.Deprecated {
			msg := prop.DeprecationMsg
			if msg == "" {
				msg = fmt.Sprintf("configuration key %q is deprecated", k)
			}
			res.Warnings = append(res.Warnings, ValidationWarning{
				Key:     k,
				Message: msg,
			})
		}

		// Type validation
		if err := v.validateType(entry, prop); err != nil {
			res.Valid = false
			res.Errors = append(res.Errors, *err)
			continue
		}

		// Enum validation
		if len(prop.EnumValues) > 0 {
			if err := v.validateEnum(entry, prop); err != nil {
				res.Valid = false
				res.Errors = append(res.Errors, *err)
				continue
			}
		}

		// Range validation
		if err := v.validateRange(entry, prop); err != nil {
			res.Valid = false
			res.Errors = append(res.Errors, *err)
			continue
		}
	}

	// 2. Check for missing required schema properties
	for _, prop := range SchemaRegistry {
		if prop.Required && !prop.Deprecated {
			entry, exists := entries[prop.Key]
			if !exists || entry.Value == nil || fmt.Sprint(entry.Value) == "" {
				res.Valid = false
				res.Errors = append(res.Errors, ConfigError{
					Code:    ErrCodeMissingRequired,
					Key:     prop.Key,
					Message: fmt.Sprintf("required configuration key %q is missing or empty", prop.Key),
				})
			}
		}
	}

	return res
}

func (v *Validator) validateType(entry ConfigEntry, prop SchemaProperty) *ConfigError {
	val := entry.Value
	if val == nil {
		return nil
	}

	switch prop.Type {
	case TypeBool:
		switch v := val.(type) {
		case bool:
			return nil
		case string:
			s := strings.ToLower(strings.TrimSpace(v))
			if s == "true" || s == "false" || s == "1" || s == "0" || s == "yes" || s == "no" {
				return nil
			}
		}
		return &ConfigError{
			Code:    ErrCodeInvalidType,
			Key:     entry.Key,
			Message: fmt.Sprintf("expected boolean value, got %T (%v)", MaskValue(entry.Key, val), MaskValue(entry.Key, val)),
			Source:  string(entry.Source),
		}

	case TypeInt:
		switch v := val.(type) {
		case int, int64, float64:
			return nil
		case string:
			var i int
			if _, err := fmt.Sscanf(v, "%d", &i); err == nil {
				return nil
			}
		}
		return &ConfigError{
			Code:    ErrCodeInvalidType,
			Key:     entry.Key,
			Message: fmt.Sprintf("expected integer value, got %T (%v)", MaskValue(entry.Key, val), MaskValue(entry.Key, val)),
			Source:  string(entry.Source),
		}

	case TypeFloat:
		switch v := val.(type) {
		case float64, int, int64:
			return nil
		case string:
			var f float64
			if _, err := fmt.Sscanf(v, "%f", &f); err == nil {
				return nil
			}
		}
		return &ConfigError{
			Code:    ErrCodeInvalidType,
			Key:     entry.Key,
			Message: fmt.Sprintf("expected floating-point value, got %T (%v)", MaskValue(entry.Key, val), MaskValue(entry.Key, val)),
			Source:  string(entry.Source),
		}

	case TypeSlice:
		switch val.(type) {
		case []string, []any, string:
			return nil
		}
		return &ConfigError{
			Code:    ErrCodeInvalidType,
			Key:     entry.Key,
			Message: fmt.Sprintf("expected list/slice value, got %T", MaskValue(entry.Key, val)),
			Source:  string(entry.Source),
		}

	case TypeString:
		return nil
	}

	return nil
}

func (v *Validator) validateEnum(entry ConfigEntry, prop SchemaProperty) *ConfigError {
	valStr := strings.ToLower(strings.TrimSpace(fmt.Sprint(entry.Value)))
	for _, allowed := range prop.EnumValues {
		if strings.ToLower(allowed) == valStr {
			return nil
		}
	}
	return &ConfigError{
		Code:    ErrCodeInvalidValue,
		Key:     entry.Key,
		Message: fmt.Sprintf("value %q is invalid; allowed values are [%s]", MaskValue(entry.Key, valStr), strings.Join(prop.EnumValues, ", ")),
		Source:  string(entry.Source),
	}
}

func (v *Validator) validateRange(entry ConfigEntry, prop SchemaProperty) *ConfigError {
	var intVal int
	switch val := entry.Value.(type) {
	case int:
		intVal = val
	case int64:
		intVal = int(val)
	case float64:
		intVal = int(val)
	default:
		return nil
	}

	if prop.MinInt != nil && intVal < *prop.MinInt {
		return &ConfigError{
			Code:    ErrCodeInvalidValue,
			Key:     entry.Key,
			Message: fmt.Sprintf("value %d is below minimum permitted value %d", intVal, *prop.MinInt),
			Source:  string(entry.Source),
		}
	}

	if prop.MaxInt != nil && intVal > *prop.MaxInt {
		return &ConfigError{
			Code:    ErrCodeInvalidValue,
			Key:     entry.Key,
			Message: fmt.Sprintf("value %d exceeds maximum permitted value %d", intVal, *prop.MaxInt),
			Source:  string(entry.Source),
		}
	}

	return nil
}
