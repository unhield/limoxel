package context

import (
	"fmt"
	"sort"
	"sync"
)

// Context defines the platform execution context interface contract.
type Context interface {
	Get(key Key) (Value, bool)
	GetString(key Key) (string, error)
	GetInt(key Key) (int, error)
	GetBool(key Key) (bool, error)
	Has(key Key) bool
	Keys() []Key
	Parent() Context
	With(key Key, value any) Context
	WithParent(parent Context) Context
	Merge(other Context) Context
	Clone() Context
}

// PlatformContext is an immutable, thread-safe implementation of Context.
type PlatformContext struct {
	mu     sync.RWMutex
	parent Context
	values map[string]Value
	keys   map[string]Key
}

// New constructs a new empty PlatformContext.
func New() *PlatformContext {
	return &PlatformContext{
		values: make(map[string]Value),
		keys:   make(map[string]Key),
	}
}

// NewWithParent constructs a new PlatformContext linked to parent.
func NewWithParent(parent Context) *PlatformContext {
	c := New()
	c.parent = parent
	return c
}

// Parent returns the parent Context, or nil if none exists.
func (c *PlatformContext) Parent() Context {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.parent
}

// Get retrieves a Value by Key, searching local state first then traversing the parent chain.
func (c *PlatformContext) Get(key Key) (Value, bool) {
	if c == nil || key.IsEmpty() {
		return Value{}, false
	}

	c.mu.RLock()
	val, exists := c.values[key.String()]
	parent := c.parent
	c.mu.RUnlock()

	if exists {
		return val, true
	}

	if parent != nil {
		return parent.Get(key)
	}

	return Value{}, false
}

// GetString returns the string value for Key.
func (c *PlatformContext) GetString(key Key) (string, error) {
	val, exists := c.Get(key)
	if !exists {
		return "", fmt.Errorf("%w: %s", ErrKeyNotFound, key)
	}
	return val.String()
}

// GetInt returns the integer value for Key.
func (c *PlatformContext) GetInt(key Key) (int, error) {
	val, exists := c.Get(key)
	if !exists {
		return 0, fmt.Errorf("%w: %s", ErrKeyNotFound, key)
	}
	return val.Int()
}

// GetBool returns the boolean value for Key.
func (c *PlatformContext) GetBool(key Key) (bool, error) {
	val, exists := c.Get(key)
	if !exists {
		return false, fmt.Errorf("%w: %s", ErrKeyNotFound, key)
	}
	return val.Bool()
}

// Has reports whether key exists in this context or its parent chain.
func (c *PlatformContext) Has(key Key) bool {
	_, exists := c.Get(key)
	return exists
}

// Keys returns a sorted, deterministic list of all unique Keys in this context and its parent chain.
func (c *PlatformContext) Keys() []Key {
	if c == nil {
		return []Key{}
	}

	keySet := make(map[string]Key)

	c.mu.RLock()
	parent := c.parent
	for k, keyObj := range c.keys {
		keySet[k] = keyObj
	}
	c.mu.RUnlock()

	if parent != nil {
		for _, parentKey := range parent.Keys() {
			if _, exists := keySet[parentKey.String()]; !exists {
				keySet[parentKey.String()] = parentKey
			}
		}
	}

	strKeys := make([]string, 0, len(keySet))
	for k := range keySet {
		strKeys = append(strKeys, k)
	}
	sort.Strings(strKeys)

	result := make([]Key, len(strKeys))
	for i, k := range strKeys {
		result[i] = keySet[k]
	}
	return result
}

// With returns a new immutable PlatformContext containing the added/updated key-value pair.
func (c *PlatformContext) With(key Key, value any) Context {
	if key.IsEmpty() {
		return c
	}

	next := NewWithParent(c.Parent())

	c.mu.RLock()
	for k, v := range c.values {
		next.values[k] = v
		next.keys[k] = c.keys[k]
	}
	c.mu.RUnlock()

	strKey := key.String()
	next.values[strKey] = NewValue(value)
	next.keys[strKey] = key

	return next
}

// WithParent returns a new immutable PlatformContext with parent assigned.
func (c *PlatformContext) WithParent(parent Context) Context {
	next := NewWithParent(parent)

	if c != nil {
		c.mu.RLock()
		for k, v := range c.values {
			next.values[k] = v
			next.keys[k] = c.keys[k]
		}
		c.mu.RUnlock()
	}

	return next
}

// Merge creates a new PlatformContext combining c and other. Keys in other override keys in c.
func (c *PlatformContext) Merge(other Context) Context {
	next := New()

	if c != nil {
		for _, k := range c.Keys() {
			if v, ok := c.Get(k); ok {
				next.values[k.String()] = v
				next.keys[k.String()] = k
			}
		}
	}

	if other != nil {
		for _, k := range other.Keys() {
			if v, ok := other.Get(k); ok {
				next.values[k.String()] = v
				next.keys[k.String()] = k
			}
		}
	}

	return next
}

// Clone creates a deep defensive copy of the PlatformContext.
func (c *PlatformContext) Clone() Context {
	if c == nil {
		return New()
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	cloned := NewWithParent(c.parent)
	for k, v := range c.values {
		cloned.values[k] = v
		cloned.keys[k] = c.keys[k]
	}

	return cloned
}
