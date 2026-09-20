package permissions

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/unhield/limoxel/plugin/security"
)

// Decision represents the outcome of evaluating a permission request.
type Decision string

const (
	// DecisionGranted indicates the requested action is authorized.
	DecisionGranted Decision = "GRANTED"
	// DecisionDenied indicates the requested action is blocked.
	DecisionDenied Decision = "DENIED"
)

// PermissionRequest defines a requested security-sensitive action against a target resource.
type PermissionRequest struct {
	PluginID string            `json:"plugin_id"`
	Domain   security.Domain   `json:"domain"`
	Action   string            `json:"action"`   // e.g., "repository.read", "file.write"
	Resource string            `json:"resource"` // e.g., "src/main.go", "api.example.com:443"
	Context  map[string]string `json:"context,omitempty"`
}

// String provides a human-readable summary of the permission request.
func (r PermissionRequest) String() string {
	if r.Resource != "" {
		return fmt.Sprintf("%s on %s (plugin: %s)", r.Action, r.Resource, r.PluginID)
	}
	return fmt.Sprintf("%s (plugin: %s)", r.Action, r.PluginID)
}

// PermissionGrant represents an explicitly approved permission and its allowed resource scopes.
type PermissionGrant struct {
	Action string   `json:"action"`           // e.g. "file.read"
	Scopes []string `json:"scopes,omitempty"` // glob patterns or path prefixes, e.g. ["src/**"]
}

// Matches checks whether the granted action and scopes satisfy the request.
func (g PermissionGrant) Matches(action, resource string) bool {
	if g.Action != "*" && g.Action != action {
		// Support wildcard prefix matching, e.g. "repository.*" matching "repository.read"
		if strings.HasSuffix(g.Action, ".*") {
			prefix := strings.TrimSuffix(g.Action, ".*")
			if !strings.HasPrefix(action, prefix+".") {
				return false
			}
		} else {
			return false
		}
	}

	// If no scopes are specified, the grant applies unconditionally to all resources within the action
	if len(g.Scopes) == 0 {
		return true
	}

	// If a resource is requested, it must match at least one granted scope
	if resource == "" {
		return true
	}

	normResource := strings.ToLower(filepath.ToSlash(strings.TrimSpace(resource)))
	for _, scope := range g.Scopes {
		normScope := strings.ToLower(filepath.ToSlash(strings.TrimSpace(scope)))
		if normScope == "*" || normScope == "**" {
			return true
		}
		// Exact match
		if normScope == normResource {
			return true
		}
		// Standard glob match (e.g. "config/*.json")
		if matched, err := filepath.Match(normScope, normResource); err == nil && matched {
			return true
		}
		// Recursive glob match (e.g. "src/**")
		if strings.HasSuffix(normScope, "/**") {
			prefix := strings.TrimSuffix(normScope, "/**")
			if strings.HasPrefix(normResource, prefix+"/") || normResource == prefix {
				return true
			}
		} else if strings.HasSuffix(normScope, "/*") {
			prefix := strings.TrimSuffix(normScope, "/*")
			if strings.HasPrefix(normResource, prefix+"/") {
				return true
			}
		}
	}

	return false
}

// EvaluationResult encapsulates the evaluation decision and audit reasoning.
type EvaluationResult struct {
	Request   PermissionRequest `json:"request"`
	Decision  Decision          `json:"decision"`
	Reason    string            `json:"reason"`
	MatchedOn string            `json:"matched_on,omitempty"`
}

// Allowed returns true if the decision was granted.
func (e EvaluationResult) Allowed() bool {
	return e.Decision == DecisionGranted
}
