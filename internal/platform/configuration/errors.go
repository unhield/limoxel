package configuration

import "errors"

var (
	// ErrConfigurationNil indicates an operation was attempted on a nil Configuration instance.
	ErrConfigurationNil = errors.New("configuration: instance is nil")

	// ErrBuilderNil indicates an operation was attempted on a nil Builder instance.
	ErrBuilderNil = errors.New("configuration: builder instance is nil")

	// ErrProviderNil indicates an attempt to register or use a nil Provider instance.
	ErrProviderNil = errors.New("configuration: provider instance is nil")

	// ErrKeyEmpty indicates a configuration key is empty.
	ErrKeyEmpty = errors.New("configuration: key cannot be empty")

	// ErrKeyInvalid indicates a configuration key format is invalid.
	ErrKeyInvalid = errors.New("configuration: invalid key format")

	// ErrKeyNotFound indicates the requested key was not found in configuration.
	ErrKeyNotFound = errors.New("configuration: key not found")

	// ErrTypeMismatch indicates a value conversion failed due to a type mismatch.
	ErrTypeMismatch = errors.New("configuration: value type mismatch")

	// ErrValidationFailed indicates configuration validation failed against specified rules.
	ErrValidationFailed = errors.New("configuration: validation failed")

	// ErrProviderFailed indicates a provider failed during configuration loading.
	ErrProviderFailed = errors.New("configuration: provider load failed")
)
