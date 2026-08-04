package configuration

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// ValueType represents the data type classification of a configuration value.
type ValueType uint8

const (
	// ValueTypeNil indicates an empty or uninitialized value.
	ValueTypeNil ValueType = iota

	// ValueTypeString indicates a string configuration value.
	ValueTypeString

	// ValueTypeInt indicates an integer configuration value.
	ValueTypeInt

	// ValueTypeFloat indicates a floating point configuration value.
	ValueTypeFloat

	// ValueTypeBool indicates a boolean configuration value.
	ValueTypeBool

	// ValueTypeDuration indicates a time.Duration configuration value.
	ValueTypeDuration

	// ValueTypeStringSlice indicates a slice of strings configuration value.
	ValueTypeStringSlice
)

// String returns the string representation of ValueType.
func (vt ValueType) String() string {
	switch vt {
	case ValueTypeNil:
		return "NIL"
	case ValueTypeString:
		return "STRING"
	case ValueTypeInt:
		return "INT"
	case ValueTypeFloat:
		return "FLOAT"
	case ValueTypeBool:
		return "BOOL"
	case ValueTypeDuration:
		return "DURATION"
	case ValueTypeStringSlice:
		return "STRING_SLICE"
	default:
		return "UNKNOWN"
	}
}

// Value represents a typed, immutable configuration value container.
type Value struct {
	raw   any
	vType ValueType
}

// NewValue constructs a Value wrapping the provided raw data.
func NewValue(val any) Value {
	if val == nil {
		return Value{raw: nil, vType: ValueTypeNil}
	}

	switch v := val.(type) {
	case string:
		return Value{raw: v, vType: ValueTypeString}
	case int:
		return Value{raw: int64(v), vType: ValueTypeInt}
	case int64:
		return Value{raw: v, vType: ValueTypeInt}
	case float64:
		return Value{raw: v, vType: ValueTypeFloat}
	case bool:
		return Value{raw: v, vType: ValueTypeBool}
	case time.Duration:
		return Value{raw: v, vType: ValueTypeDuration}
	case []string:
		sliceCopy := make([]string, len(v))
		copy(sliceCopy, v)
		return Value{raw: sliceCopy, vType: ValueTypeStringSlice}
	default:
		return Value{raw: fmt.Sprintf("%v", val), vType: ValueTypeString}
	}
}

// Type returns the ValueType of the value.
func (v Value) Type() ValueType {
	return v.vType
}

// IsEmpty reports whether the Value is uninitialized or nil.
func (v Value) IsEmpty() bool {
	return v.vType == ValueTypeNil || v.raw == nil
}

// Raw returns the underlying raw value.
func (v Value) Raw() any {
	return v.raw
}

// String returns the string representation of the value or converts compatible types.
func (v Value) String() (string, error) {
	if v.IsEmpty() {
		return "", ErrKeyNotFound
	}
	switch val := v.raw.(type) {
	case string:
		return val, nil
	case int64:
		return strconv.FormatInt(val, 10), nil
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(val), nil
	case time.Duration:
		return val.String(), nil
	case []string:
		return strings.Join(val, ","), nil
	default:
		return fmt.Sprintf("%v", val), nil
	}
}

// Int returns the value as an integer, converting if possible.
func (v Value) Int() (int, error) {
	if v.IsEmpty() {
		return 0, ErrKeyNotFound
	}

	switch val := v.raw.(type) {
	case int64:
		if val < int64(math.MinInt) || val > int64(math.MaxInt) {
			return 0, fmt.Errorf("%w: integer overflow converting int64 to int", ErrTypeMismatch)
		}
		return int(val), nil
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(val))
		if err != nil {
			return 0, fmt.Errorf("%w: cannot convert string %q to int: %v", ErrTypeMismatch, val, err)
		}
		return i, nil
	case float64:
		if val < float64(math.MinInt) || val > float64(math.MaxInt) {
			return 0, fmt.Errorf("%w: float64 value %v overflows int range", ErrTypeMismatch, val)
		}
		return int(val), nil
	case bool:
		if val {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("%w: cannot convert type %s to int", ErrTypeMismatch, v.vType)
	}
}

// Int64 returns the value as an int64, converting if possible.
func (v Value) Int64() (int64, error) {
	if v.IsEmpty() {
		return 0, ErrKeyNotFound
	}
	switch val := v.raw.(type) {
	case int64:
		return val, nil
	case float64:
		return int64(val), nil
	case string:
		i, err := strconv.ParseInt(strings.TrimSpace(val), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%w: cannot convert string %q to int64: %v", ErrTypeMismatch, val, err)
		}
		return i, nil
	case bool:
		if val {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("%w: cannot convert type %s to int64", ErrTypeMismatch, v.vType)
	}
}

// Float64 returns the value as a float64, converting if possible.
func (v Value) Float64() (float64, error) {
	if v.IsEmpty() {
		return 0, ErrKeyNotFound
	}
	switch val := v.raw.(type) {
	case float64:
		return val, nil
	case int64:
		return float64(val), nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
		if err != nil {
			return 0, fmt.Errorf("%w: cannot convert string %q to float64: %v", ErrTypeMismatch, val, err)
		}
		return f, nil
	default:
		return 0, fmt.Errorf("%w: cannot convert type %s to float64", ErrTypeMismatch, v.vType)
	}
}

// Bool returns the value as a boolean, converting if possible.
func (v Value) Bool() (bool, error) {
	if v.IsEmpty() {
		return false, ErrKeyNotFound
	}
	switch val := v.raw.(type) {
	case bool:
		return val, nil
	case int64:
		return val != 0, nil
	case string:
		b, err := strconv.ParseBool(strings.TrimSpace(val))
		if err != nil {
			return false, fmt.Errorf("%w: cannot convert string %q to bool: %v", ErrTypeMismatch, val, err)
		}
		return b, nil
	default:
		return false, fmt.Errorf("%w: cannot convert type %s to bool", ErrTypeMismatch, v.vType)
	}
}

// Duration returns the value as a time.Duration, converting if possible.
func (v Value) Duration() (time.Duration, error) {
	if v.IsEmpty() {
		return 0, ErrKeyNotFound
	}
	switch val := v.raw.(type) {
	case time.Duration:
		return val, nil
	case int64:
		return time.Duration(val), nil
	case string:
		d, err := time.ParseDuration(strings.TrimSpace(val))
		if err != nil {
			return 0, fmt.Errorf("%w: cannot convert string %q to duration: %v", ErrTypeMismatch, val, err)
		}
		return d, nil
	default:
		return 0, fmt.Errorf("%w: cannot convert type %s to duration", ErrTypeMismatch, v.vType)
	}
}

// StringSlice returns the value as a slice of strings.
func (v Value) StringSlice() ([]string, error) {
	if v.IsEmpty() {
		return nil, ErrKeyNotFound
	}
	switch val := v.raw.(type) {
	case []string:
		res := make([]string, len(val))
		copy(res, val)
		return res, nil
	case string:
		parts := strings.Split(val, ",")
		res := make([]string, 0, len(parts))
		for _, p := range parts {
			res = append(res, strings.TrimSpace(p))
		}
		return res, nil
	default:
		return nil, fmt.Errorf("%w: cannot convert type %s to string slice", ErrTypeMismatch, v.vType)
	}
}

// ValidateKey verifies key syntax. Keys must be non-empty and dot-delimited alphanumerics.
func ValidateKey(key string) error {
	if key == "" {
		return ErrKeyEmpty
	}
	if strings.HasPrefix(key, ".") || strings.HasSuffix(key, ".") || strings.Contains(key, "..") {
		return fmt.Errorf("%w: %s", ErrKeyInvalid, key)
	}
	return nil
}
