package bootstrap

// Option defines a functional option for configuring a Bootstrapper instance.
type Option func(*Bootstrapper)

// PrerequisiteValidator defines a function signature for prerequisite validation checks.
type PrerequisiteValidator func() error

// WithPrerequisiteValidator adds a custom prerequisite validation check to the bootstrap sequence.
func WithPrerequisiteValidator(validator PrerequisiteValidator) Option {
	return func(b *Bootstrapper) {
		if validator != nil {
			b.validators = append(b.validators, validator)
		}
	}
}
