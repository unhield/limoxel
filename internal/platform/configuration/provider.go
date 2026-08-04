package configuration

import "context"

// Provider defines the contract for loading configuration key-value mappings.
type Provider interface {
	// Name returns the provider's unique identifier.
	Name() string

	// Load loads and returns the configuration values provided by this source.
	Load(ctx context.Context) (map[string]Value, error)
}

// MemoryProvider is an in-memory configuration provider.
type MemoryProvider struct {
	name   string
	values map[string]Value
}

// NewMemoryProvider constructs a MemoryProvider with the specified name and initial key-value map.
func NewMemoryProvider(name string, values map[string]Value) *MemoryProvider {
	if name == "" {
		name = "memory"
	}
	vCopy := make(map[string]Value)
	for k, v := range values {
		if err := ValidateKey(k); err == nil {
			vCopy[k] = v
		}
	}
	return &MemoryProvider{
		name:   name,
		values: vCopy,
	}
}

// Name returns the name of the MemoryProvider.
func (mp *MemoryProvider) Name() string {
	return mp.name
}

// Load returns a copy of the in-memory configuration values.
func (mp *MemoryProvider) Load(ctx context.Context) (map[string]Value, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}

	result := make(map[string]Value, len(mp.values))
	for k, v := range mp.values {
		result[k] = v
	}
	return result, nil
}
