package compatibility

import (
	"fmt"
	"strings"
	"sync"

	"github.com/unhield/limoxel/internal/capabilities/sdk/compatibility"
	"github.com/unhield/limoxel/internal/capabilities/sdk/lifecycle"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
	canonversion "github.com/unhield/limoxel/internal/version"
)

// Re-exported types for public compatibility consumers.
type (
	ChangeKind            = compatibility.ChangeKind
	APIChange             = compatibility.APIChange
	CompatibilityDecision = compatibility.CompatibilityDecision
	MigrationGuide        = compatibility.MigrationGuide
	DeprecationInfo       = lifecycle.DeprecationInfo
	LifecycleState        = lifecycle.LifecycleState
	SemVer                = version.SemVer
	ReleaseKind           = version.ReleaseKind
)

// Re-exported constants.
const (
	ChangeBreaking      = compatibility.ChangeBreaking
	ChangeAddition      = compatibility.ChangeAddition
	ChangeFix           = compatibility.ChangeFix
	ChangeDocumentation = compatibility.ChangeDocumentation

	StateIntroduced = lifecycle.StateIntroduced
	StateSupported  = lifecycle.StateSupported
	StateDeprecated = lifecycle.StateDeprecated
	StateRemoved    = lifecycle.StateRemoved

	ReleaseMajor = version.ReleaseMajor
	ReleaseMinor = version.ReleaseMinor
	ReleasePatch = version.ReleasePatch
)

// DeprecationRecord defines a registered deprecation entry for public tracking.
type DeprecationRecord struct {
	APIName           string `json:"api_name"`
	Since             string `json:"since"`
	PlannedRemoval    string `json:"planned_removal,omitempty"`
	Replacement       string `json:"replacement,omitempty"`
	Reason            string `json:"reason"`
	MigrationGuidance string `json:"migration_guidance,omitempty"`
}

// DeprecationTracker manages public API deprecations across SDK version releases.
type DeprecationTracker struct {
	mu      sync.RWMutex
	records map[string]DeprecationRecord
}

// NewDeprecationTracker constructs a thread-safe DeprecationTracker.
func NewDeprecationTracker() *DeprecationTracker {
	return &DeprecationTracker{
		records: make(map[string]DeprecationRecord),
	}
}

// Register adds or updates a public API deprecation record.
func (t *DeprecationTracker) Register(rec DeprecationRecord) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.records[rec.APIName] = rec
}

// Lookup queries deprecation metadata for a given API identifier.
func (t *DeprecationTracker) Lookup(apiName string) (DeprecationRecord, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	rec, ok := t.records[apiName]
	return rec, ok
}

// All returns all registered deprecation records.
func (t *DeprecationTracker) All() []DeprecationRecord {
	t.mu.RLock()
	defer t.mu.RUnlock()
	list := make([]DeprecationRecord, 0, len(t.records))
	for _, r := range t.records {
		list = append(list, r)
	}
	return list
}

// UpgradeValidator tests and validates whether consumer applications can safely upgrade between SDK versions.
type UpgradeValidator struct {
	evaluator *compatibility.Evaluator
}

// NewUpgradeValidator constructs an UpgradeValidator.
func NewUpgradeValidator() *UpgradeValidator {
	return &UpgradeValidator{
		evaluator: compatibility.NewEvaluator(),
	}
}

// ValidateUpgrade evaluates a planned version upgrade against a set of API changes.
func (u *UpgradeValidator) ValidateUpgrade(currentVersion, targetVersion version.SemVer, changes []APIChange) CompatibilityDecision {
	plannedRelease := version.ClassifyRelease(currentVersion, targetVersion)
	return u.evaluator.Evaluate(changes, plannedRelease)
}

// GenerateMigrationGuide builds a markdown migration document for version transitions.
func (u *UpgradeValidator) GenerateMigrationGuide(from, to version.SemVer, changes []APIChange) *MigrationGuide {
	return compatibility.NewMigrationGuide(from, to, changes)
}

// CurrentVersion returns the authoritative SemVer of the SDK.
func CurrentVersion() version.SemVer {
	return version.Current()
}

// CurrentVersionString returns the canonical version string.
func CurrentVersionString() string {
	return canonversion.Version
}

// FormatDeprecationNotice creates a standardized user-facing deprecation warning message.
func FormatDeprecationNotice(rec DeprecationRecord) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[DEPRECATION] API %q is deprecated since v%s.", rec.APIName, rec.Since))
	if rec.PlannedRemoval != "" {
		sb.WriteString(fmt.Sprintf(" Scheduled for removal in v%s.", rec.PlannedRemoval))
	}
	if rec.Replacement != "" {
		sb.WriteString(fmt.Sprintf(" Replacement: %q.", rec.Replacement))
	}
	if rec.MigrationGuidance != "" {
		sb.WriteString(fmt.Sprintf(" Guidance: %s", rec.MigrationGuidance))
	}
	return sb.String()
}
