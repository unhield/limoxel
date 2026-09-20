package validation

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	pkgerr "github.com/unhield/limoxel/internal/capabilities/plugin/errors"
	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

// Validator encapsulates the composable rules required to validate a plugin before registration and execution.
type Validator struct {
	hostVersion      version.SemVer
	currentPlatform  string
	allowedPlatforms map[string]struct{}
}

// Option configures validator behavior.
type Option func(*Validator)

// WithPlatform overrides the target execution platform (defaults to runtime.GOOS).
func WithPlatform(platform string) Option {
	return func(v *Validator) {
		v.currentPlatform = strings.ToLower(strings.TrimSpace(platform))
	}
}

// NewValidator constructs a Validator configured for the specified host version.
func NewValidator(hostVersion version.SemVer, opts ...Option) *Validator {
	v := &Validator{
		hostVersion:     hostVersion,
		currentPlatform: runtime.GOOS,
	}
	for _, opt := range opts {
		opt(v)
	}
	return v
}

// Validate executes all static verification stages against a plugin manifest.
func (v *Validator) Validate(m *model.Manifest) error {
	if m == nil {
		return pkgerr.New(pkgerr.CodeManifestInvalid, "", "manifest cannot be nil")
	}

	// 1. Validate Identity
	if err := v.validateIdentity(m); err != nil {
		return err
	}

	// 2. Validate SemVer
	if err := v.validateVersion(m); err != nil {
		return err
	}

	// 3. Validate Capabilities
	if err := v.validateCapabilities(m); err != nil {
		return err
	}

	// 4. Validate Required Capabilities
	if err := v.validateRequiredCapabilities(m); err != nil {
		return err
	}

	// 5. Validate Dependencies
	if err := v.validateDependencies(m); err != nil {
		return err
	}

	// 6. Validate Host Compatibility
	if err := v.validateHostCompatibility(m); err != nil {
		return err
	}

	// 7. Validate Path Safety
	if err := v.validatePathSafety(m); err != nil {
		return err
	}

	return nil
}

func (v *Validator) validateIdentity(m *model.Manifest) error {
	if string(m.ID) == "" {
		return pkgerr.New(pkgerr.CodeIdentityInvalid, m.ID, "plugin identifier is required")
	}
	if _, err := model.ParseIdentity(string(m.ID)); err != nil {
		return pkgerr.Wrap(pkgerr.CodeIdentityInvalid, m.ID, "invalid plugin identifier syntax", err)
	}
	return nil
}

func (v *Validator) validateVersion(m *model.Manifest) error {
	if m.Version.Major < 0 || m.Version.Minor < 0 || m.Version.Patch < 0 {
		return pkgerr.New(pkgerr.CodeIncompatibleVersion, m.ID, "version numbers must be non-negative")
	}
	return nil
}

func (v *Validator) validateCapabilities(m *model.Manifest) error {
	seen := make(map[string]struct{}, len(m.Capabilities))
	for _, cap := range m.Capabilities {
		if _, exists := seen[cap.Name]; exists {
			return pkgerr.New(pkgerr.CodeManifestInvalid, m.ID, fmt.Sprintf("duplicate capability declaration: %q", cap.Name))
		}
		seen[cap.Name] = struct{}{}
	}
	return nil
}

func (v *Validator) validateRequiredCapabilities(m *model.Manifest) error {
	seen := make(map[string]struct{}, len(m.RequiredCapabilities))
	for _, req := range m.RequiredCapabilities {
		if _, exists := seen[req.Name]; exists {
			return pkgerr.New(pkgerr.CodeManifestInvalid, m.ID, fmt.Sprintf("duplicate required capability: %q", req.Name))
		}
		seen[req.Name] = struct{}{}
	}
	return nil
}

func (v *Validator) validateDependencies(m *model.Manifest) error {
	seen := make(map[model.Identity]struct{}, len(m.Dependencies))
	for _, dep := range m.Dependencies {
		if dep.ID == m.ID {
			return pkgerr.New(pkgerr.CodeDependencyCycle, m.ID, "self-referential dependency is not permitted")
		}
		if _, exists := seen[dep.ID]; exists {
			return pkgerr.New(pkgerr.CodeDependencyConflict, m.ID, fmt.Sprintf("duplicate dependency declaration for %s", dep.ID))
		}
		seen[dep.ID] = struct{}{}
	}
	return nil
}

func (v *Validator) validateHostCompatibility(m *model.Manifest) error {
	compat := m.Compatibility

	// Check min host version
	if compat.MinHostVersion != nil {
		if v.hostVersion.Compare(*compat.MinHostVersion) < 0 {
			err := pkgerr.New(pkgerr.CodeIncompatibleHost, m.ID,
				fmt.Sprintf("plugin requires minimum host version %s, but running on %s", compat.MinHostVersion, v.hostVersion))
			err.WithDetail("required_min", compat.MinHostVersion.String())
			err.WithDetail("host_version", v.hostVersion.String())
			return err
		}
	}

	// Check max host version
	if compat.MaxHostVersion != nil {
		if v.hostVersion.Compare(*compat.MaxHostVersion) > 0 {
			err := pkgerr.New(pkgerr.CodeIncompatibleHost, m.ID,
				fmt.Sprintf("plugin requires maximum host version %s, but running on %s", compat.MaxHostVersion, v.hostVersion))
			err.WithDetail("required_max", compat.MaxHostVersion.String())
			err.WithDetail("host_version", v.hostVersion.String())
			return err
		}
	}

	// Check platform support
	if len(compat.SupportedPlatforms) > 0 {
		matched := false
		for _, p := range compat.SupportedPlatforms {
			if strings.EqualFold(strings.TrimSpace(p), v.currentPlatform) {
				matched = true
				break
			}
		}
		if !matched {
			err := pkgerr.New(pkgerr.CodeIncompatibleHost, m.ID,
				fmt.Sprintf("plugin does not support host platform %q (supported: %v)", v.currentPlatform, compat.SupportedPlatforms))
			err.WithDetail("host_platform", v.currentPlatform)
			return err
		}
	}

	return nil
}

func (v *Validator) validatePathSafety(m *model.Manifest) error {
	if m.Entrypoint != "" {
		trimmed := strings.TrimSpace(m.Entrypoint)
		if strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, "\\") || filepath.IsAbs(trimmed) || filepath.VolumeName(trimmed) != "" {
			return pkgerr.New(pkgerr.CodeSecurityPolicyViolation, m.ID, "plugin entrypoint cannot be an absolute or volume path")
		}
		ep := filepath.Clean(trimmed)
		if ep == ".." || strings.HasPrefix(ep, ".."+string(filepath.Separator)) || strings.HasPrefix(ep, "../") || strings.Contains(ep, string(filepath.Separator)+"..") || strings.Contains(ep, "/..") {
			return pkgerr.New(pkgerr.CodeSecurityPolicyViolation, m.ID, "plugin entrypoint cannot traverse out of plugin directory")
		}
	}
	return nil
}
