package permissions

import (
	"encoding/json"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
	"github.com/unhield/limoxel/plugin/security"
)

// PluginPolicy encapsulates the configured permission grants and execution restrictions for a plugin.
type PluginPolicy struct {
	PluginID string                     `json:"plugin_id"`
	Profile  security.PermissionProfile `json:"profile"`
	Grants   []PermissionGrant          `json:"grants"`
}

// NewRestrictedPolicy creates a least-privilege default policy for an untrusted or unverified plugin.
func NewRestrictedPolicy(pluginID string) *PluginPolicy {
	return &PluginPolicy{
		PluginID: pluginID,
		Profile:  security.ProfileRestricted,
		Grants: []PermissionGrant{
			{Action: security.PermRepoMetadata},
			{Action: security.PermConfigRead},
		},
	}
}

// NewStandardPolicy creates a standard read-only workspace policy.
func NewStandardPolicy(pluginID string) *PluginPolicy {
	return &PluginPolicy{
		PluginID: pluginID,
		Profile:  security.ProfileStandard,
		Grants: []PermissionGrant{
			{Action: security.PermRepoMetadata},
			{Action: security.PermRepoRead},
			{Action: security.PermFileRead},
			{Action: security.PermConfigRead},
		},
	}
}

// ParseManifestPolicy extracts permissions and scopes from a plugin manifest and metadata.
func ParseManifestPolicy(manifest *model.Manifest) *PluginPolicy {
	if manifest == nil {
		return NewRestrictedPolicy("unknown")
	}

	pluginID := string(manifest.ID)
	policy := &PluginPolicy{
		PluginID: pluginID,
		Profile:  security.ProfileStandard,
		Grants:   make([]PermissionGrant, 0),
	}

	// Read custom attributes for permissions and scopes
	attrs := manifest.Metadata.CustomAttributes()

	var permList []string
	if permsRaw, ok := attrs["permissions"]; ok && permsRaw != "" {
		if strings.HasPrefix(permsRaw, "[") {
			_ = json.Unmarshal([]byte(permsRaw), &permList)
		} else {
			for _, p := range strings.Split(permsRaw, ",") {
				if trimmed := strings.TrimSpace(p); trimmed != "" {
					permList = append(permList, trimmed)
				}
			}
		}
	}

	scopesMap := make(map[string][]string)
	if scopesRaw, ok := attrs["permission_scopes"]; ok && scopesRaw != "" {
		_ = json.Unmarshal([]byte(scopesRaw), &scopesMap)
	}

	// Always provide baseline metadata permission
	hasMeta := false
	for _, p := range permList {
		if p == security.PermRepoMetadata {
			hasMeta = true
			break
		}
	}
	if !hasMeta {
		permList = append([]string{security.PermRepoMetadata}, permList...)
	}

	for _, action := range permList {
		grant := PermissionGrant{
			Action: action,
			Scopes: scopesMap[action],
		}
		policy.Grants = append(policy.Grants, grant)
	}

	return policy
}
