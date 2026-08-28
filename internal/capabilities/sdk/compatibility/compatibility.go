package compatibility

import (
	"fmt"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

// ChangeKind classifies the nature and backward-compatibility impact of a public API modification.
type ChangeKind string

const (
	// ChangeBreaking represents an incompatible modification (e.g. removal, signature change, tighter constraint).
	ChangeBreaking ChangeKind = "BREAKING"

	// ChangeAddition represents a backward-compatible feature or optional parameter addition.
	ChangeAddition ChangeKind = "ADDITION"

	// ChangeFix represents a backward-compatible bug or security correction.
	ChangeFix ChangeKind = "FIX"

	// ChangeDocumentation represents a non-functional doc comment or example update.
	ChangeDocumentation ChangeKind = "DOCUMENTATION"
)

// String returns the string representation of ChangeKind.
func (k ChangeKind) String() string {
	return string(k)
}

// APIChange encapsulates details of a public API change.
type APIChange struct {
	APIName           string     `json:"api_name"`
	Kind              ChangeKind `json:"kind"`
	Description       string     `json:"description"`
	OldSignature      string     `json:"old_signature,omitempty"`
	NewSignature      string     `json:"new_signature,omitempty"`
	MigrationGuidance string     `json:"migration_guidance,omitempty"`
}

// CompatibilityDecision encapsulates the outcome of a compatibility evaluation against Semantic Versioning policy.
type CompatibilityDecision struct {
	IsCompatible   bool                `json:"is_compatible"`
	TargetRelease  version.ReleaseKind `json:"target_release"`
	Violations     []string            `json:"violations,omitempty"`
	Summary        string              `json:"summary"`
	RequiredSemVer version.ReleaseKind `json:"required_release"`
}

// Evaluator validates API change lists against Semantic Versioning release policies.
type Evaluator struct{}

// NewEvaluator constructs an initialized compatibility Evaluator.
func NewEvaluator() *Evaluator {
	return &Evaluator{}
}

// Evaluate checks whether the set of API changes is legally permissible under plannedRelease.
func (e *Evaluator) Evaluate(changes []APIChange, plannedRelease version.ReleaseKind) CompatibilityDecision {
	hasBreaking := false
	hasAddition := false
	violations := make([]string, 0)

	for _, ch := range changes {
		switch ch.Kind {
		case ChangeBreaking:
			hasBreaking = true
			if plannedRelease != version.ReleaseMajor {
				violations = append(violations, fmt.Sprintf("breaking change in %q (%s) requires a MAJOR release, but planned release is %s", ch.APIName, ch.Description, plannedRelease))
			}
		case ChangeAddition:
			hasAddition = true
			if plannedRelease == version.ReleasePatch {
				violations = append(violations, fmt.Sprintf("feature addition in %q (%s) requires at least a MINOR release, but planned release is PATCH", ch.APIName, ch.Description))
			}
		}
	}

	requiredRelease := version.ReleasePatch
	if hasBreaking {
		requiredRelease = version.ReleaseMajor
	} else if hasAddition {
		requiredRelease = version.ReleaseMinor
	}

	isCompatible := len(violations) == 0
	var summary string
	if isCompatible {
		summary = fmt.Sprintf("All %d API changes are fully compliant with planned %s release.", len(changes), plannedRelease)
	} else {
		summary = fmt.Sprintf("Found %d compatibility policy violations for planned %s release (requires %s).", len(violations), plannedRelease, requiredRelease)
	}

	return CompatibilityDecision{
		IsCompatible:   isCompatible,
		TargetRelease:  plannedRelease,
		Violations:     violations,
		Summary:        summary,
		RequiredSemVer: requiredRelease,
	}
}

// MigrationGuide generates structured migration instructions for consumers transitioning across versions.
type MigrationGuide struct {
	FromVersion        version.SemVer
	ToVersion          version.SemVer
	BreakingChanges    []APIChange
	RecommendedActions []string
}

// NewMigrationGuide builds a MigrationGuide from evaluated changes.
func NewMigrationGuide(from, to version.SemVer, changes []APIChange) *MigrationGuide {
	breaking := make([]APIChange, 0)
	actions := make([]string, 0)

	for _, ch := range changes {
		if ch.Kind == ChangeBreaking {
			breaking = append(breaking, ch)
			if ch.MigrationGuidance != "" {
				actions = append(actions, fmt.Sprintf("[%s] %s", ch.APIName, ch.MigrationGuidance))
			} else {
				actions = append(actions, fmt.Sprintf("[%s] Review updated API contract: %s", ch.APIName, ch.Description))
			}
		}
	}

	return &MigrationGuide{
		FromVersion:        from,
		ToVersion:          to,
		BreakingChanges:    breaking,
		RecommendedActions: actions,
	}
}

// FormatMarkdown formats the migration guide into a clean Markdown document.
func (m *MigrationGuide) FormatMarkdown() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Migration Guide: v%s -> v%s\n\n", m.FromVersion, m.ToVersion))

	if len(m.BreakingChanges) == 0 {
		sb.WriteString("No breaking API changes detected. Upgrade is 100% backward-compatible.\n")
		return sb.String()
	}

	sb.WriteString("## Breaking Changes\n\n")
	for i, b := range m.BreakingChanges {
		sb.WriteString(fmt.Sprintf("### %d. %s\n", i+1, b.APIName))
		sb.WriteString(fmt.Sprintf("- **Description**: %s\n", b.Description))
		if b.OldSignature != "" {
			sb.WriteString(fmt.Sprintf("- **Previous Signature**: `%s`\n", b.OldSignature))
		}
		if b.NewSignature != "" {
			sb.WriteString(fmt.Sprintf("- **New Signature**: `%s`\n", b.NewSignature))
		}
		if b.MigrationGuidance != "" {
			sb.WriteString(fmt.Sprintf("- **Action**: %s\n", b.MigrationGuidance))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## Action Checklist\n\n")
	for _, act := range m.RecommendedActions {
		sb.WriteString(fmt.Sprintf("- [ ] %s\n", act))
	}

	return sb.String()
}
