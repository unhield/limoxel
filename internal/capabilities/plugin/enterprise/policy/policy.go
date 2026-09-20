package policy

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

var (
	ErrPolicyNotFound = errors.New("enterprise policy: policy document not found")
	ErrInvalidPolicy  = errors.New("enterprise policy: invalid policy specification")

	orgPolicyIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9\-_]{2,62}[a-zA-Z0-9]$`)
)

// PolicyType identifies the domain of an enterprise policy.
type PolicyType string

const (
	PolicyTypeAllowlist  PolicyType = "allowlist"
	PolicyTypeDenylist   PolicyType = "denylist"
	PolicyTypeVersion    PolicyType = "version"
	PolicyTypeSecurity   PolicyType = "security"
	PolicyTypeDeployment PolicyType = "deployment"
)

// AllowlistRule permits deployment based on matched attributes.
type AllowlistRule struct {
	RuleID         string   `json:"rule_id"`
	PluginIDs      []string `json:"plugin_ids,omitempty"`      // Exact or wildcard
	Publishers     []string `json:"publishers,omitempty"`      // Trusted publisher IDs
	AllowedSources []string `json:"allowed_sources,omitempty"` // "marketplace", "private_registry"
}

// DenylistRule unconditionally blocks deployment based on matching attributes.
type DenylistRule struct {
	RuleID              string   `json:"rule_id"`
	PluginIDs           []string `json:"plugin_ids,omitempty"`
	Publishers          []string `json:"publishers,omitempty"`
	BlockedVersions     []string `json:"blocked_versions,omitempty"` // Specific versions or ranges
	BlockedCapabilities []string `json:"blocked_capabilities,omitempty"`
	BlockedPermissions  []string `json:"blocked_permissions,omitempty"`
	Reason              string   `json:"reason"`
}

// VersionRule restricts or enforces specific version constraints.
type VersionRule struct {
	RuleID          string `json:"rule_id"`
	PluginID        string `json:"plugin_id"`
	MinVersion      string `json:"min_version,omitempty"`
	MaxVersion      string `json:"max_version,omitempty"`
	PinnedVersion   string `json:"pinned_version,omitempty"`
	RequiredUpgrade string `json:"required_upgrade,omitempty"`
}

// SecurityRule defines cryptographic and sandboxing requirements.
type SecurityRule struct {
	RuleID                string   `json:"rule_id"`
	RequireSigned         bool     `json:"require_signed"`
	TrustedRoots          []string `json:"trusted_roots,omitempty"`
	RequiredClass         string   `json:"required_class,omitempty"` // e.g. "verified", "enterprise"
	RequireSandbox        bool     `json:"require_sandbox"`
	RequireMalwareClean   bool     `json:"require_malware_clean"`
	MaxAllowedPermissions int      `json:"max_allowed_permissions,omitempty"`
}

// DeploymentRule restricts execution to approved environments and infrastructure.
type DeploymentRule struct {
	RuleID               string   `json:"rule_id"`
	PluginID             string   `json:"plugin_id,omitempty"`
	AllowedEnvironments  []string `json:"allowed_environments,omitempty"` // "production", "staging", "dev"
	AllowedPlatforms     []string `json:"allowed_platforms,omitempty"`    // "linux/amd64", "windows/amd64"
	AllowedMachineGroups []string `json:"allowed_machine_groups,omitempty"`
}

// PolicyDocument represents a versioned collection of rules for an organization.
type PolicyDocument struct {
	OrganizationID  string           `json:"organization_id"`
	Revision        int64            `json:"revision"`
	UpdatedAt       time.Time        `json:"updated_at"`
	AuthorID        string           `json:"author_id"`
	AllowlistRules  []AllowlistRule  `json:"allowlist_rules,omitempty"`
	DenylistRules   []DenylistRule   `json:"denylist_rules,omitempty"`
	VersionRules    []VersionRule    `json:"version_rules,omitempty"`
	SecurityRules   []SecurityRule   `json:"security_rules,omitempty"`
	DeploymentRules []DeploymentRule `json:"deployment_rules,omitempty"`
}

// EvaluationContext contains all runtime details inspected during policy evaluation.
type EvaluationContext struct {
	OrganizationID     string
	PluginID           string
	Version            version.SemVer
	PublisherID        string
	DistributionSource string // "marketplace", "private_registry"
	IsSigned           bool
	TrustRootID        string
	SecurityClass      string
	Permissions        []string
	Capabilities       []string
	Environment        string
	Platform           string
	MachineGroup       string
	MalwareScanPassed  bool
}

// PolicyDecision communicates the definitive allow/deny verdict and rationale.
type PolicyDecision struct {
	Allowed    bool       `json:"allowed"`
	PolicyType PolicyType `json:"policy_type"`
	RuleID     string     `json:"rule_id,omitempty"`
	Reason     string     `json:"reason"`
	Timestamp  time.Time  `json:"timestamp"`
}

// ValidatePolicyDocument verifies the semantic and structural integrity of an enterprise policy document.
func ValidatePolicyDocument(doc *PolicyDocument) error {
	if doc == nil {
		return fmt.Errorf("%w: policy document cannot be nil", ErrInvalidPolicy)
	}

	orgID := strings.TrimSpace(doc.OrganizationID)
	if orgID == "" || !orgPolicyIDPattern.MatchString(orgID) {
		return fmt.Errorf("%w: invalid or missing organization ID '%s'", ErrInvalidPolicy, doc.OrganizationID)
	}

	for i, r := range doc.AllowlistRules {
		if strings.TrimSpace(r.RuleID) == "" {
			return fmt.Errorf("%w: allowlist rule at index %d has empty rule_id", ErrInvalidPolicy, i)
		}
	}

	for i, r := range doc.DenylistRules {
		if strings.TrimSpace(r.RuleID) == "" {
			return fmt.Errorf("%w: denylist rule at index %d has empty rule_id", ErrInvalidPolicy, i)
		}
	}

	for i, r := range doc.VersionRules {
		if strings.TrimSpace(r.RuleID) == "" {
			return fmt.Errorf("%w: version rule at index %d has empty rule_id", ErrInvalidPolicy, i)
		}
		if strings.TrimSpace(r.PluginID) == "" {
			return fmt.Errorf("%w: version rule '%s' has empty plugin_id", ErrInvalidPolicy, r.RuleID)
		}
		var minVer, maxVer *version.SemVer
		if r.MinVersion != "" {
			v, err := version.ParseSemVer(r.MinVersion)
			if err != nil {
				return fmt.Errorf("%w: version rule '%s' invalid min_version '%s': %v", ErrInvalidPolicy, r.RuleID, r.MinVersion, err)
			}
			minVer = &v
		}
		if r.MaxVersion != "" {
			v, err := version.ParseSemVer(r.MaxVersion)
			if err != nil {
				return fmt.Errorf("%w: version rule '%s' invalid max_version '%s': %v", ErrInvalidPolicy, r.RuleID, r.MaxVersion, err)
			}
			maxVer = &v
		}
		if minVer != nil && maxVer != nil && minVer.Compare(*maxVer) > 0 {
			return fmt.Errorf("%w: version rule '%s' min_version '%s' is greater than max_version '%s'", ErrInvalidPolicy, r.RuleID, r.MinVersion, r.MaxVersion)
		}
		if r.PinnedVersion != "" {
			if _, err := version.ParseSemVer(r.PinnedVersion); err != nil {
				return fmt.Errorf("%w: version rule '%s' invalid pinned_version '%s': %v", ErrInvalidPolicy, r.RuleID, r.PinnedVersion, err)
			}
		}
		if r.RequiredUpgrade != "" {
			if _, err := version.ParseSemVer(r.RequiredUpgrade); err != nil {
				return fmt.Errorf("%w: version rule '%s' invalid required_upgrade '%s': %v", ErrInvalidPolicy, r.RuleID, r.RequiredUpgrade, err)
			}
		}
	}

	for i, r := range doc.SecurityRules {
		if strings.TrimSpace(r.RuleID) == "" {
			return fmt.Errorf("%w: security rule at index %d has empty rule_id", ErrInvalidPolicy, i)
		}
		if r.MaxAllowedPermissions < 0 {
			return fmt.Errorf("%w: security rule '%s' max_allowed_permissions cannot be negative", ErrInvalidPolicy, r.RuleID)
		}
	}

	for i, r := range doc.DeploymentRules {
		if strings.TrimSpace(r.RuleID) == "" {
			return fmt.Errorf("%w: deployment rule at index %d has empty rule_id", ErrInvalidPolicy, i)
		}
	}

	return nil
}
