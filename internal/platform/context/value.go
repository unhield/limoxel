package context

import (
	"fmt"
	"strconv"
	"strings"
)

// Value represents an immutable typed value container stored within a Context.
type Value struct {
	raw any
}

// NewValue constructs a Value wrapping raw.
func NewValue(raw any) Value {
	return Value{raw: raw}
}

// Raw returns the underlying raw value.
func (v Value) Raw() any {
	return v.raw
}

// IsEmpty reports whether the Value is nil or uninitialized.
func (v Value) IsEmpty() bool {
	return v.raw == nil
}

// String returns the string representation of the Value.
func (v Value) String() (string, error) {
	if v.IsEmpty() {
		return "", ErrKeyNotFound
	}
	switch val := v.raw.(type) {
	case string:
		return val, nil
	case fmt.Stringer:
		return val.String(), nil
	default:
		return fmt.Sprintf("%v", val), nil
	}
}

// Int returns the value as an integer if convertible.
func (v Value) Int() (int, error) {
	if v.IsEmpty() {
		return 0, ErrKeyNotFound
	}
	switch val := v.raw.(type) {
	case int:
		return val, nil
	case int64:
		return int(val), nil
	case float64:
		return int(val), nil
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(val))
		if err != nil {
			return 0, fmt.Errorf("%w: cannot convert string %q to int: %v", ErrTypeMismatch, val, err)
		}
		return i, nil
	default:
		return 0, fmt.Errorf("%w: cannot convert type %T to int", ErrTypeMismatch, v.raw)
	}
}

// Bool returns the value as a boolean if convertible.
func (v Value) Bool() (bool, error) {
	if v.IsEmpty() {
		return false, ErrKeyNotFound
	}
	switch val := v.raw.(type) {
	case bool:
		return val, nil
	case string:
		b, err := strconv.ParseBool(strings.TrimSpace(val))
		if err != nil {
			return false, fmt.Errorf("%w: cannot convert string %q to bool: %v", ErrTypeMismatch, val, err)
		}
		return b, nil
	default:
		return false, fmt.Errorf("%w: cannot convert type %T to bool", ErrTypeMismatch, v.raw)
	}
}
