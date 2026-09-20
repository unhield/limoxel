package policy

import (
	"fmt"
	"strings"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

// Evaluator executes deterministic evaluation of enterprise policies.
type Evaluator struct{}

// NewEvaluator constructs an enterprise policy evaluation engine.
func NewEvaluator() *Evaluator {
	return &Evaluator{}
}

// Evaluate applies the organization policy document against the evaluation context.
func (e *Evaluator) Evaluate(doc *PolicyDocument, ctx EvaluationContext) PolicyDecision {
	now := time.Now().UTC()

	if doc == nil {
		return PolicyDecision{
			Allowed:   true,
			Reason:    "no organizational policy configured; defaulting to allowed",
			Timestamp: now,
		}
	}

	// 1. Evaluate Denylist Rules (Absolute priority over allow)
	for _, rule := range doc.DenylistRules {
		// Plugin ID
		for _, blockedID := range rule.PluginIDs {
			if matchIdentifier(blockedID, ctx.PluginID) {
				return PolicyDecision{
					Allowed:    false,
					PolicyType: PolicyTypeDenylist,
					RuleID:     rule.RuleID,
					Reason:     fmt.Sprintf("plugin %q blocked by denylist rule: %s", ctx.PluginID, rule.Reason),
					Timestamp:  now,
				}
			}
		}
		// Publisher
		for _, blockedPub := range rule.Publishers {
			if blockedPub == ctx.PublisherID {
				return PolicyDecision{
					Allowed:    false,
					PolicyType: PolicyTypeDenylist,
					RuleID:     rule.RuleID,
					Reason:     fmt.Sprintf("publisher %q blocked by denylist rule: %s", ctx.PublisherID, rule.Reason),
					Timestamp:  now,
				}
			}
		}
		// Blocked Versions
		for _, blockedVer := range rule.BlockedVersions {
			if ctx.Version.String() == blockedVer {
				return PolicyDecision{
					Allowed:    false,
					PolicyType: PolicyTypeDenylist,
					RuleID:     rule.RuleID,
					Reason:     fmt.Sprintf("version %s blocked by denylist rule: %s", ctx.Version.String(), rule.Reason),
					Timestamp:  now,
				}
			}
		}
		// Blocked Capabilities
		for _, cap := range ctx.Capabilities {
			for _, blockedCap := range rule.BlockedCapabilities {
				if cap == blockedCap {
					return PolicyDecision{
						Allowed:    false,
						PolicyType: PolicyTypeDenylist,
						RuleID:     rule.RuleID,
						Reason:     fmt.Sprintf("capability %q blocked by denylist rule: %s", cap, rule.Reason),
						Timestamp:  now,
					}
				}
			}
		}
		// Blocked Permissions
		for _, perm := range ctx.Permissions {
			for _, blockedPerm := range rule.BlockedPermissions {
				if perm == blockedPerm {
					return PolicyDecision{
						Allowed:    false,
						PolicyType: PolicyTypeDenylist,
						RuleID:     rule.RuleID,
						Reason:     fmt.Sprintf("permission %q blocked by denylist rule: %s", perm, rule.Reason),
						Timestamp:  now,
					}
				}
			}
		}
	}

	// 2. Evaluate Security Rules
	for _, rule := range doc.SecurityRules {
		if rule.RequireSigned && !ctx.IsSigned {
			return PolicyDecision{
				Allowed:    false,
				PolicyType: PolicyTypeSecurity,
				RuleID:     rule.RuleID,
				Reason:     "unsigned plugins prohibited by enterprise security policy",
				Timestamp:  now,
			}
		}
		if rule.RequireMalwareClean && !ctx.MalwareScanPassed {
			return PolicyDecision{
				Allowed:    false,
				PolicyType: PolicyTypeSecurity,
				RuleID:     rule.RuleID,
				Reason:     "failed malware heuristic scan; rejected by security policy",
				Timestamp:  now,
			}
		}
		if len(rule.TrustedRoots) > 0 {
			trusted := false
			for _, root := range rule.TrustedRoots {
				if root == ctx.TrustRootID {
					trusted = true
					break
				}
			}
			if !trusted {
				return PolicyDecision{
					Allowed:    false,
					PolicyType: PolicyTypeSecurity,
					RuleID:     rule.RuleID,
					Reason:     fmt.Sprintf("signature trust root %q not in enterprise trusted roots", ctx.TrustRootID),
					Timestamp:  now,
				}
			}
		}
		if rule.MaxAllowedPermissions > 0 && len(ctx.Permissions) > rule.MaxAllowedPermissions {
			return PolicyDecision{
				Allowed:    false,
				PolicyType: PolicyTypeSecurity,
				RuleID:     rule.RuleID,
				Reason:     fmt.Sprintf("plugin declared %d permissions; exceeds maximum allowed %d", len(ctx.Permissions), rule.MaxAllowedPermissions),
				Timestamp:  now,
			}
		}
	}

	// 3. Evaluate Allowlist Rules (If configured, plugin must match at least one rule)
	if len(doc.AllowlistRules) > 0 {
		matched := false
		for _, rule := range doc.AllowlistRules {
			// Check plugin ID
			idMatch := len(rule.PluginIDs) == 0
			for _, allowedID := range rule.PluginIDs {
				if matchIdentifier(allowedID, ctx.PluginID) {
					idMatch = true
					break
				}
			}
			// Check publisher
			pubMatch := len(rule.Publishers) == 0
			for _, allowedPub := range rule.Publishers {
				if allowedPub == ctx.PublisherID {
					pubMatch = true
					break
				}
			}
			// Check source
			srcMatch := len(rule.AllowedSources) == 0
			for _, allowedSrc := range rule.AllowedSources {
				if allowedSrc == ctx.DistributionSource {
					srcMatch = true
					break
				}
			}

			if idMatch && pubMatch && srcMatch {
				matched = true
				break
			}
		}
		if !matched {
			return PolicyDecision{
				Allowed:    false,
				PolicyType: PolicyTypeAllowlist,
				Reason:     fmt.Sprintf("plugin %q (publisher: %q, source: %q) not present in organization allowlist", ctx.PluginID, ctx.PublisherID, ctx.DistributionSource),
				Timestamp:  now,
			}
		}
	}

	// 4. Evaluate Version Rules
	for _, rule := range doc.VersionRules {
		if rule.PluginID == "" || matchIdentifier(rule.PluginID, ctx.PluginID) {
			if rule.PinnedVersion != "" && ctx.Version.String() != rule.PinnedVersion {
				return PolicyDecision{
					Allowed:    false,
					PolicyType: PolicyTypeVersion,
					RuleID:     rule.RuleID,
					Reason:     fmt.Sprintf("version %s does not match required pinned version %s", ctx.Version.String(), rule.PinnedVersion),
					Timestamp:  now,
				}
			}
			if rule.MinVersion != "" {
				minV, err := version.ParseSemVer(rule.MinVersion)
				if err == nil && ctx.Version.Compare(minV) < 0 {
					return PolicyDecision{
						Allowed:    false,
						PolicyType: PolicyTypeVersion,
						RuleID:     rule.RuleID,
						Reason:     fmt.Sprintf("version %s is below enterprise minimum version %s", ctx.Version.String(), rule.MinVersion),
						Timestamp:  now,
					}
				}
			}
			if rule.MaxVersion != "" {
				maxV, err := version.ParseSemVer(rule.MaxVersion)
				if err == nil && ctx.Version.Compare(maxV) > 0 {
					return PolicyDecision{
						Allowed:    false,
						PolicyType: PolicyTypeVersion,
						RuleID:     rule.RuleID,
						Reason:     fmt.Sprintf("version %s exceeds enterprise maximum version %s", ctx.Version.String(), rule.MaxVersion),
						Timestamp:  now,
					}
				}
			}
		}
	}

	// 5. Evaluate Deployment Rules
	for _, rule := range doc.DeploymentRules {
		if rule.PluginID == "" || matchIdentifier(rule.PluginID, ctx.PluginID) {
			if len(rule.AllowedEnvironments) > 0 && ctx.Environment != "" {
				envMatch := false
				for _, env := range rule.AllowedEnvironments {
					if env == ctx.Environment {
						envMatch = true
						break
					}
				}
				if !envMatch {
					return PolicyDecision{
						Allowed:    false,
						PolicyType: PolicyTypeDeployment,
						RuleID:     rule.RuleID,
						Reason:     fmt.Sprintf("environment %q not allowed for deployment of plugin %q", ctx.Environment, ctx.PluginID),
						Timestamp:  now,
					}
				}
			}
			if len(rule.AllowedPlatforms) > 0 && ctx.Platform != "" {
				platMatch := false
				for _, plat := range rule.AllowedPlatforms {
					if plat == ctx.Platform {
						platMatch = true
						break
					}
				}
				if !platMatch {
					return PolicyDecision{
						Allowed:    false,
						PolicyType: PolicyTypeDeployment,
						RuleID:     rule.RuleID,
						Reason:     fmt.Sprintf("platform %q not allowed for deployment of plugin %q", ctx.Platform, ctx.PluginID),
						Timestamp:  now,
					}
				}
			}
		}
	}

	return PolicyDecision{
		Allowed:   true,
		Reason:    "all enterprise policies successfully satisfied",
		Timestamp: now,
	}
}

func matchIdentifier(pattern, id string) bool {
	if pattern == "*" || pattern == id {
		return true
	}
	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, ".*")
		return strings.HasPrefix(id, prefix+".") || id == prefix
	}
	return false
}
