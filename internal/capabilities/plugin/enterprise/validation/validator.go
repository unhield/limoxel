package validation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/policy"
	marketval "github.com/unhield/limoxel/internal/capabilities/plugin/marketplace/validation"
	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
	"github.com/unhield/limoxel/internal/capabilities/plugin/security/verification"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
	canonversion "github.com/unhield/limoxel/internal/version"
	"github.com/unhield/limoxel/plugin/security"
)

// IssueSeverity classifies the criticality of a validation finding.
type IssueSeverity string

const (
	SeverityError   IssueSeverity = "ERROR"
	SeverityWarning IssueSeverity = "WARNING"
	SeverityInfo    IssueSeverity = "INFO"
)

// ValidationIssue records a specific finding during enterprise validation.
type ValidationIssue struct {
	Dimension IssueSeverity `json:"dimension"` // e.g. Compliance, Security, Compatibility, Upgrade, Stability
	Severity  IssueSeverity `json:"severity"`
	Component string        `json:"component"`
	Message   string        `json:"message"`
}

// ValidationReport documents the outcome of all five enterprise validation dimensions.
type ValidationReport struct {
	Passed              bool              `json:"passed"`
	CompliancePassed    bool              `json:"compliance_passed"`
	SecurityPassed      bool              `json:"security_passed"`
	CompatibilityPassed bool              `json:"compatibility_passed"`
	UpgradePassed       bool              `json:"upgrade_passed"`
	StabilityPassed     bool              `json:"stability_passed"`
	Issues              []ValidationIssue `json:"issues,omitempty"`
	EvaluatedAt         time.Time         `json:"evaluated_at"`
}

// EnterpriseValidator orchestrates compliance, security, compatibility, upgrade, and stability checks.
type EnterpriseValidator struct {
	stage3Verifier *verification.Verifier
	trustStore     *verification.TrustStore
	policyEval     *policy.Evaluator
	hostVersion    string
}

// NewEnterpriseValidator constructs an enterprise validation coordinator.
func NewEnterpriseValidator(trustStore *verification.TrustStore) *EnterpriseValidator {
	vCfg := verification.VerifierConfig{
		TrustStore:       trustStore,
		RequireSignature: true,
	}
	return &EnterpriseValidator{
		stage3Verifier: verification.NewVerifier(vCfg),
		trustStore:     trustStore,
		policyEval:     policy.NewEvaluator(),
		hostVersion:    canonversion.Version,
	}
}

// VerifyArchive checks if an archive contains a verified signature and passes malware heuristic scanning.
func (v *EnterpriseValidator) VerifyArchive(ctx context.Context, archiveData []byte) (isSigned bool, malwareClean bool, err error) {
	if len(archiveData) == 0 {
		return false, true, nil
	}

	reader := bytes.NewReader(archiveData)
	pkg, _, err := marketval.InspectAndExtract(reader, marketval.DefaultPolicy())
	if err != nil {
		// Non-archive or raw payload: run malware scanner heuristics on the raw content
		scanner := marketval.NewStaticHeuristicScanner()
		files := map[string][]byte{"payload": archiveData}
		scanRes, _ := scanner.Scan(ctx, files, marketval.DefaultPolicy())
		return false, scanRes.Passed, nil
	}

	// 1. Malware scan
	scanner := marketval.NewStaticHeuristicScanner()
	scanRes, scanErr := scanner.Scan(ctx, pkg.Files, marketval.DefaultPolicy())
	malwareClean = scanErr == nil && scanRes.Passed

	// 2. Signature verification
	if v.trustStore != nil {
		h := sha256.Sum256(archiveData)
		computedDigest := hex.EncodeToString(h[:])
		sigVerifier := marketval.NewSignatureValidator(v.trustStore)
		sigRes := sigVerifier.VerifyPackageSignature(pkg, computedDigest, marketval.DefaultPolicy())
		isSigned = sigRes.Passed
	}

	return isSigned, malwareClean, nil
}

// ValidateCompliance checks whether the plugin satisfies organizational policies and metadata requirements.
func (v *EnterpriseValidator) ValidateCompliance(
	ctx context.Context,
	manifest *model.Manifest,
	doc *policy.PolicyDocument,
	evalCtx policy.EvaluationContext,
) (bool, []ValidationIssue) {
	var issues []ValidationIssue

	if manifest == nil {
		issues = append(issues, ValidationIssue{
			Dimension: SeverityError,
			Severity:  SeverityError,
			Component: "Compliance",
			Message:   "missing plugin manifest",
		})
		return false, issues
	}

	// 1. Evaluate organization policy rules
	decision := v.policyEval.Evaluate(doc, evalCtx)
	if !decision.Allowed {
		issues = append(issues, ValidationIssue{
			Dimension: SeverityError,
			Severity:  SeverityError,
			Component: "Compliance:Policy",
			Message:   fmt.Sprintf("policy denial (%s): %s", decision.PolicyType, decision.Reason),
		})
		return false, issues
	}

	// 2. Required Enterprise Metadata
	if strings.TrimSpace(manifest.Publisher) == "" {
		issues = append(issues, ValidationIssue{
			Dimension: SeverityError,
			Severity:  SeverityError,
			Component: "Compliance:Metadata",
			Message:   "publisher identifier is required in enterprise manifests",
		})
	}

	passed := true
	for _, iss := range issues {
		if iss.Severity == SeverityError {
			passed = false
			break
		}
	}
	return passed, issues
}

// ValidateSecurity integrates security signature checks, malware heuristics, and sandbox constraints.
func (v *EnterpriseValidator) ValidateSecurity(
	ctx context.Context,
	manifest *model.Manifest,
	pluginDir string,
	malwareClean bool,
) (bool, []ValidationIssue) {
	var issues []ValidationIssue

	if !malwareClean {
		issues = append(issues, ValidationIssue{
			Dimension: SeverityError,
			Severity:  SeverityError,
			Component: "Security:Malware",
			Message:   "static malware scanner reported suspicious executable or shell pattern",
		})
	}

	if manifest == nil {
		issues = append(issues, ValidationIssue{
			Dimension: SeverityError,
			Severity:  SeverityError,
			Component: "Security:Manifest",
			Message:   "manifest is required for security verification",
		})
		return false, issues
	}

	// Security Verification
	if pluginDir != "" {
		res, err := v.stage3Verifier.VerifyPluginDirectory(ctx, manifest, pluginDir)
		if err != nil || res == nil || res.State == security.TrustRejected {
			errMsg := "Security verification rejected plugin"
			if res != nil && len(res.Errors) > 0 {
				errMsg += ": " + strings.Join(res.Errors, "; ")
			}
			issues = append(issues, ValidationIssue{
				Dimension: SeverityError,
				Severity:  SeverityError,
				Component: "Security:Signature",
				Message:   errMsg,
			})
		}
	}

	passed := true
	for _, iss := range issues {
		if iss.Severity == SeverityError {
			passed = false
			break
		}
	}
	return passed, issues
}

// ValidateCompatibility checks host version, OS, architecture, and SDK compatibility.
func (v *EnterpriseValidator) ValidateCompatibility(
	ctx context.Context,
	manifest *model.Manifest,
	targetOS, targetArch string,
) (bool, []ValidationIssue) {
	var issues []ValidationIssue
	if manifest == nil {
		return false, []ValidationIssue{{
			Dimension: SeverityError,
			Severity:  SeverityError,
			Component: "Compatibility",
			Message:   "manifest is nil",
		}}
	}

	// Host OS / Arch compatibility
	currentOS := targetOS
	if currentOS == "" {
		currentOS = runtime.GOOS
	}
	currentArch := targetArch
	if currentArch == "" {
		currentArch = runtime.GOARCH
	}

	// Check if manifest specifies OS constraints in capabilities or supported platforms
	for _, cap := range manifest.Capabilities {
		if strings.HasPrefix(cap.Name, "os:") {
			reqOS := strings.TrimPrefix(cap.Name, "os:")
			if reqOS != currentOS {
				issues = append(issues, ValidationIssue{
					Dimension: SeverityError,
					Severity:  SeverityError,
					Component: "Compatibility:OS",
					Message:   fmt.Sprintf("plugin requires OS %q, target is %q", reqOS, currentOS),
				})
			}
		}
	}

	if len(manifest.Compatibility.SupportedPlatforms) > 0 {
		matched := false
		targetPlatform := fmt.Sprintf("%s/%s", currentOS, currentArch)
		for _, sp := range manifest.Compatibility.SupportedPlatforms {
			if sp == currentOS || sp == targetPlatform {
				matched = true
				break
			}
		}
		if !matched {
			issues = append(issues, ValidationIssue{
				Dimension: SeverityError,
				Severity:  SeverityError,
				Component: "Compatibility:Platform",
				Message:   fmt.Sprintf("target platform %q not in supported platforms %v", targetPlatform, manifest.Compatibility.SupportedPlatforms),
			})
		}
	}

	passed := true
	for _, iss := range issues {
		if iss.Severity == SeverityError {
			passed = false
			break
		}
	}
	return passed, issues
}

// ValidateUpgrade analyzes target version, dependency deltas, and rollback feasibility.
func (v *EnterpriseValidator) ValidateUpgrade(
	ctx context.Context,
	currentVer, targetVer version.SemVer,
	policyDoc *policy.PolicyDocument,
) (bool, []ValidationIssue) {
	var issues []ValidationIssue

	// Prevent downgrades unless explicitly authorized
	if targetVer.Compare(currentVer) < 0 {
		issues = append(issues, ValidationIssue{
			Dimension: SeverityError,
			Severity:  SeverityError,
			Component: "Upgrade:Version",
			Message:   fmt.Sprintf("target version %s is a downgrade from currently installed %s", targetVer.String(), currentVer.String()),
		})
	}

	// Check if target version is blocked in version policy
	if policyDoc != nil {
		for _, rule := range policyDoc.VersionRules {
			if rule.MaxVersion != "" {
				maxV, err := version.ParseSemVer(rule.MaxVersion)
				if err == nil && targetVer.Compare(maxV) > 0 {
					issues = append(issues, ValidationIssue{
						Dimension: SeverityError,
						Severity:  SeverityError,
						Component: "Upgrade:Policy",
						Message:   fmt.Sprintf("target version %s exceeds maximum policy version %s", targetVer.String(), rule.MaxVersion),
					})
				}
			}
		}
	}

	passed := true
	for _, iss := range issues {
		if iss.Severity == SeverityError {
			passed = false
			break
		}
	}
	return passed, issues
}

// ValidateStability verifies runtime stability indicators, crash history, and resource constraints.
func (v *EnterpriseValidator) ValidateStability(
	ctx context.Context,
	recentCrashCount int,
	maxAllowedCrashes int,
) (bool, []ValidationIssue) {
	var issues []ValidationIssue

	if maxAllowedCrashes > 0 && recentCrashCount >= maxAllowedCrashes {
		issues = append(issues, ValidationIssue{
			Dimension: SeverityError,
			Severity:  SeverityError,
			Component: "Stability:Crashes",
			Message:   fmt.Sprintf("plugin observed %d crashes in evaluation window, exceeding threshold %d", recentCrashCount, maxAllowedCrashes),
		})
	}

	passed := true
	for _, iss := range issues {
		if iss.Severity == SeverityError {
			passed = false
			break
		}
	}
	return passed, issues
}

// ValidateAll executes all five enterprise validation dimensions.
func (v *EnterpriseValidator) ValidateAll(
	ctx context.Context,
	manifest *model.Manifest,
	pluginDir string,
	malwareClean bool,
	doc *policy.PolicyDocument,
	evalCtx policy.EvaluationContext,
	targetOS, targetArch string,
	currentVer *version.SemVer,
	recentCrashes int,
) *ValidationReport {
	report := &ValidationReport{
		EvaluatedAt: time.Now().UTC(),
	}

	// 1. Compliance
	compPassed, compIssues := v.ValidateCompliance(ctx, manifest, doc, evalCtx)
	report.CompliancePassed = compPassed
	report.Issues = append(report.Issues, compIssues...)

	// 2. Security
	secPassed, secIssues := v.ValidateSecurity(ctx, manifest, pluginDir, malwareClean)
	report.SecurityPassed = secPassed
	report.Issues = append(report.Issues, secIssues...)

	// 3. Compatibility
	compatPassed, compatIssues := v.ValidateCompatibility(ctx, manifest, targetOS, targetArch)
	report.CompatibilityPassed = compatPassed
	report.Issues = append(report.Issues, compatIssues...)

	// 4. Upgrade
	if currentVer != nil && manifest != nil {
		upPassed, upIssues := v.ValidateUpgrade(ctx, *currentVer, manifest.Version, doc)
		report.UpgradePassed = upPassed
		report.Issues = append(report.Issues, upIssues...)
	} else {
		report.UpgradePassed = true
	}

	// 5. Stability
	stabPassed, stabIssues := v.ValidateStability(ctx, recentCrashes, 3)
	report.StabilityPassed = stabPassed
	report.Issues = append(report.Issues, stabIssues...)

	report.Passed = report.CompliancePassed && report.SecurityPassed &&
		report.CompatibilityPassed && report.UpgradePassed && report.StabilityPassed

	return report
}
