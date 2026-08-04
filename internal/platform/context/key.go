package context

import (
	"fmt"
	"strings"
)

// Key represents a strongly typed, canonical context key.
type Key struct {
	category string
	name     string
}

// NewKey creates a new context Key with the given name.
func NewKey(name string) Key {
	return NewCategorizedKey("", name)
}

// NewCategorizedKey creates a new context Key with a category and name.
func NewCategorizedKey(category, name string) Key {
	return Key{
		category: strings.TrimSpace(category),
		name:     strings.TrimSpace(name),
	}
}

// Name returns the key's name.
func (k Key) Name() string {
	return k.name
}

// Category returns the key's category.
func (k Key) Category() string {
	return k.category
}

// String returns the string representation of the Key.
func (k Key) String() string {
	if k.category == "" {
		return k.name
	}
	return fmt.Sprintf("%s.%s", k.category, k.name)
}

// IsEmpty reports whether the Key is empty.
func (k Key) IsEmpty() bool {
	return k.name == ""
}

// ValidateKey checks that the Key is non-empty.
func ValidateKey(k Key) error {
	if k.IsEmpty() {
		return ErrKeyEmpty
	}
	return nil
}
