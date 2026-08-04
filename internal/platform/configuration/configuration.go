package configuration

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Validator defines a function signature for validating a Configuration instance.
type Validator func(*Configuration) error

// Configuration represents an immutable, thread-safe platform configuration snapshot.
type Configuration struct {
	mu     sync.RWMutex
	values map[string]Value
}

// New constructs an empty, immutable Configuration instance.
func New() *Configuration {
	return &Configuration{
		values: make(map[string]Value),
	}
}

// NewFromMap constructs an immutable Configuration initialized with the provided key-value map.
func NewFromMap(vals map[string]Value) (*Configuration, error) {
	c := New()
	for k, v := range vals {
		if err := ValidateKey(k); err != nil {
			return nil, err
		}
		c.values[k] = v
	}
	return c, nil
}

// Get returns the Value for key and a boolean indicating whether the key exists.
func (c *Configuration) Get(key string) (Value, bool) {
	if c == nil {
		return Value{}, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()

	val, exists := c.values[key]
	return val, exists
}

// GetString returns the string value for key.
func (c *Configuration) GetString(key string) (string, error) {
	val, exists := c.Get(key)
	if !exists {
		return "", fmt.Errorf("%w: %s", ErrKeyNotFound, key)
	}
	return val.String()
}

// GetInt returns the integer value for key.
func (c *Configuration) GetInt(key string) (int, error) {
	val, exists := c.Get(key)
	if !exists {
		return 0, fmt.Errorf("%w: %s", ErrKeyNotFound, key)
	}
	return val.Int()
}

// GetInt64 returns the int64 value for key.
func (c *Configuration) GetInt64(key string) (int64, error) {
	val, exists := c.Get(key)
	if !exists {
		return 0, fmt.Errorf("%w: %s", ErrKeyNotFound, key)
	}
	return val.Int64()
}

// GetFloat64 returns the float64 value for key.
func (c *Configuration) GetFloat64(key string) (float64, error) {
	val, exists := c.Get(key)
	if !exists {
		return 0, fmt.Errorf("%w: %s", ErrKeyNotFound, key)
	}
	return val.Float64()
}

// GetBool returns the boolean value for key.
func (c *Configuration) GetBool(key string) (bool, error) {
	val, exists := c.Get(key)
	if !exists {
		return false, fmt.Errorf("%w: %s", ErrKeyNotFound, key)
	}
	return val.Bool()
}

// GetDuration returns the time.Duration value for key.
func (c *Configuration) GetDuration(key string) (time.Duration, error) {
	val, exists := c.Get(key)
	if !exists {
		return 0, fmt.Errorf("%w: %s", ErrKeyNotFound, key)
	}
	return val.Duration()
}

// GetStringSlice returns the []string slice value for key.
func (c *Configuration) GetStringSlice(key string) ([]string, error) {
	val, exists := c.Get(key)
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrKeyNotFound, key)
	}
	return val.StringSlice()
}

// Has reports whether the specified key exists in the Configuration.
func (c *Configuration) Has(key string) bool {
	_, exists := c.Get(key)
	return exists
}

// Keys returns a sorted, deterministic list of all configuration keys.
func (c *Configuration) Keys() []string {
	if c == nil {
		return []string{}
	}
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0, len(c.values))
	for k := range c.values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Merge creates a new Configuration combining c and other. Values in other override values in c.
func (c *Configuration) Merge(other *Configuration) (*Configuration, error) {
	merged := New()

	if c != nil {
		c.mu.RLock()
		for k, v := range c.values {
			merged.values[k] = v
		}
		c.mu.RUnlock()
	}

	if other != nil {
		other.mu.RLock()
		for k, v := range other.values {
			merged.values[k] = v
		}
		other.mu.RUnlock()
	}

	return merged, nil
}

// Validate evaluates custom validation rules against the Configuration.
func (c *Configuration) Validate(rules ...Validator) error {
	if c == nil {
		return ErrConfigurationNil
	}
	for _, rule := range rules {
		if rule != nil {
			if err := rule(c); err != nil {
				return fmt.Errorf("%w: %v", ErrValidationFailed, err)
			}
		}
	}
	return nil
}

// Builder constructs a Configuration instance from defaults, providers, and overrides.
type Builder struct {
	defaults  map[string]Value
	providers []Provider
	overrides map[string]Value
}

// NewBuilder creates a new Builder instance.
func NewBuilder() *Builder {
	return &Builder{
		defaults:  make(map[string]Value),
		providers: make([]Provider, 0),
		overrides: make(map[string]Value),
	}
}

// WithDefault adds a default key-value pair to the Builder.
func (b *Builder) WithDefault(key string, val any) *Builder {
	if b == nil {
		return b
	}
	if err := ValidateKey(key); err == nil {
		b.defaults[key] = NewValue(val)
	}
	return b
}

// WithOverride adds an override key-value pair to the Builder.
func (b *Builder) WithOverride(key string, val any) *Builder {
	if b == nil {
		return b
	}
	if err := ValidateKey(key); err == nil {
		b.overrides[key] = NewValue(val)
	}
	return b
}

// WithProvider registers a Provider with the Builder.
func (b *Builder) WithProvider(provider Provider) *Builder {
	if b == nil || provider == nil {
		return b
	}
	b.providers = append(b.providers, provider)
	return b
}

// Build loads all providers in order and constructs an immutable Configuration.
func (b *Builder) Build(ctx context.Context) (*Configuration, error) {
	if b == nil {
		return nil, ErrBuilderNil
	}

	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}

	accumulated := make(map[string]Value)

	// Apply defaults
	for k, v := range b.defaults {
		accumulated[k] = v
	}

	// Apply providers sequentially
	for _, p := range b.providers {
		if p == nil {
			continue
		}
		pVals, err := p.Load(ctx)
		if err != nil {
			return nil, fmt.Errorf("%w (%s): %v", ErrProviderFailed, p.Name(), err)
		}
		for k, v := range pVals {
			if err := ValidateKey(k); err == nil {
				accumulated[k] = v
			}
		}
	}

	// Apply overrides
	for k, v := range b.overrides {
		accumulated[k] = v
	}

	return NewFromMap(accumulated)
}
