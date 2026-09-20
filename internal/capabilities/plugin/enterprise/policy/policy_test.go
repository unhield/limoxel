package policy

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

func TestPolicyEvaluator_DenylistPrecedence(t *testing.T) {
	evaluator := NewEvaluator()
	v1, _ := version.ParseSemVer("1.0.0")

	doc := &PolicyDocument{
		OrganizationID: "org-acme",
		AllowlistRules: []AllowlistRule{
			{RuleID: "allow-all", PluginIDs: []string{"*"}},
		},
		DenylistRules: []DenylistRule{
			{RuleID: "deny-banned", PluginIDs: []string{"banned.plugin"}, Reason: "known security flaw"},
		},
	}

	// 1. Allowed plugin
	ctx1 := EvaluationContext{
		PluginID:           "safe.plugin",
		Version:            v1,
		DistributionSource: "marketplace",
		IsSigned:           true,
		MalwareScanPassed:  true,
	}
	dec1 := evaluator.Evaluate(doc, ctx1)
	if !dec1.Allowed {
		t.Errorf("expected safe.plugin to be allowed, got: %s", dec1.Reason)
	}

	// 2. Denylisted plugin overrides allowlist
	ctx2 := EvaluationContext{
		PluginID:           "banned.plugin",
		Version:            v1,
		DistributionSource: "marketplace",
		IsSigned:           true,
		MalwareScanPassed:  true,
	}
	dec2 := evaluator.Evaluate(doc, ctx2)
	if dec2.Allowed || dec2.PolicyType != PolicyTypeDenylist {
		t.Errorf("expected banned.plugin to be denied by denylist, got: %+v", dec2)
	}
}

func TestPolicyEvaluator_SecurityRules(t *testing.T) {
	evaluator := NewEvaluator()
	v1, _ := version.ParseSemVer("1.0.0")

	doc := &PolicyDocument{
		OrganizationID: "org-acme",
		SecurityRules: []SecurityRule{
			{
				RuleID:              "sec-strict",
				RequireSigned:       true,
				RequireMalwareClean: true,
				TrustedRoots:        []string{"root-acme-prod"},
			},
		},
	}

	// Unsigned plugin rejected
	ctx1 := EvaluationContext{
		PluginID:          "plugin-a",
		Version:           v1,
		IsSigned:          false,
		MalwareScanPassed: true,
		TrustRootID:       "root-acme-prod",
	}
	dec1 := evaluator.Evaluate(doc, ctx1)
	if dec1.Allowed || dec1.PolicyType != PolicyTypeSecurity {
		t.Errorf("expected unsigned plugin to be denied by security policy, got: %+v", dec1)
	}

	// Malware scan failure rejected
	ctx2 := EvaluationContext{
		PluginID:          "plugin-a",
		Version:           v1,
		IsSigned:          true,
		MalwareScanPassed: false,
		TrustRootID:       "root-acme-prod",
	}
	dec2 := evaluator.Evaluate(doc, ctx2)
	if dec2.Allowed || dec2.PolicyType != PolicyTypeSecurity {
		t.Errorf("expected malware failure to be denied, got: %+v", dec2)
	}

	// Wrong root rejected
	ctx3 := EvaluationContext{
		PluginID:          "plugin-a",
		Version:           v1,
		IsSigned:          true,
		MalwareScanPassed: true,
		TrustRootID:       "root-untrusted",
	}
	dec3 := evaluator.Evaluate(doc, ctx3)
	if dec3.Allowed || dec3.PolicyType != PolicyTypeSecurity {
		t.Errorf("expected untrusted root to be denied, got: %+v", dec3)
	}

	// Fully compliant accepted
	ctx4 := EvaluationContext{
		PluginID:          "plugin-a",
		Version:           v1,
		IsSigned:          true,
		MalwareScanPassed: true,
		TrustRootID:       "root-acme-prod",
	}
	dec4 := evaluator.Evaluate(doc, ctx4)
	if !dec4.Allowed {
		t.Errorf("expected compliant plugin to be allowed, got: %s", dec4.Reason)
	}
}

func TestPolicyEvaluator_VersionRules(t *testing.T) {
	evaluator := NewEvaluator()
	vOld, _ := version.ParseSemVer("0.9.0")
	vValid, _ := version.ParseSemVer("1.5.0")
	vPinned, _ := version.ParseSemVer("2.0.0")

	doc := &PolicyDocument{
		OrganizationID: "org-acme",
		VersionRules: []VersionRule{
			{
				RuleID:     "ver-min",
				PluginID:   "acme.auth",
				MinVersion: "1.0.0",
				MaxVersion: "2.0.0",
			},
			{
				RuleID:        "ver-pinned",
				PluginID:      "acme.legacy",
				PinnedVersion: "2.0.0",
			},
		},
	}

	// 1. Below min version
	ctx1 := EvaluationContext{PluginID: "acme.auth", Version: vOld}
	dec1 := evaluator.Evaluate(doc, ctx1)
	if dec1.Allowed || dec1.PolicyType != PolicyTypeVersion {
		t.Errorf("expected below min version to be denied, got: %+v", dec1)
	}

	// 2. Valid range
	ctx2 := EvaluationContext{PluginID: "acme.auth", Version: vValid}
	dec2 := evaluator.Evaluate(doc, ctx2)
	if !dec2.Allowed {
		t.Errorf("expected valid version to be allowed, got: %s", dec2.Reason)
	}

	// 3. Pinned version check
	ctx3 := EvaluationContext{PluginID: "acme.legacy", Version: vValid}
	dec3 := evaluator.Evaluate(doc, ctx3)
	if dec3.Allowed || dec3.PolicyType != PolicyTypeVersion {
		t.Errorf("expected unpinned version to be denied, got: %+v", dec3)
	}

	ctx4 := EvaluationContext{PluginID: "acme.legacy", Version: vPinned}
	dec4 := evaluator.Evaluate(doc, ctx4)
	if !dec4.Allowed {
		t.Errorf("expected pinned version to be allowed, got: %s", dec4.Reason)
	}
}

func TestPolicyStorage_PersistenceAndRevision(t *testing.T) {
	tempDir := t.TempDir()
	store, err := NewStorage(tempDir)
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	ctx := context.Background()
	doc := &PolicyDocument{
		OrganizationID: "org-acme",
		AuthorID:       "admin-1",
		AllowlistRules: []AllowlistRule{
			{RuleID: "rule-1", PluginIDs: []string{"acme.*"}},
		},
	}

	// First save: Revision 1
	if err := store.SavePolicy(ctx, doc); err != nil {
		t.Fatalf("SavePolicy 1 failed: %v", err)
	}
	if doc.Revision != 1 {
		t.Errorf("expected revision 1, got %d", doc.Revision)
	}

	// Second save: Revision 2
	doc.AllowlistRules = append(doc.AllowlistRules, AllowlistRule{RuleID: "rule-2", PluginIDs: []string{"partner.*"}})
	if err := store.SavePolicy(ctx, doc); err != nil {
		t.Fatalf("SavePolicy 2 failed: %v", err)
	}
	if doc.Revision != 2 {
		t.Errorf("expected revision 2, got %d", doc.Revision)
	}

	// Reload from disk
	storeReloaded, err := NewStorage(tempDir)
	if err != nil {
		t.Fatalf("NewStorage reload failed: %v", err)
	}

	loaded, err := storeReloaded.GetPolicy(ctx, "org-acme")
	if err != nil {
		t.Fatalf("GetPolicy failed: %v", err)
	}
	if loaded.Revision != 2 || len(loaded.AllowlistRules) != 2 {
		t.Errorf("unexpected reloaded policy: %+v", loaded)
	}
}

func TestStorage_IgnoreSubdirectoriesAndNonJSON(t *testing.T) {
	tempDir := t.TempDir()
	ctx := context.Background()

	// Create a subdirectory inside baseDir
	subDir := filepath.Join(tempDir, "nested_folder")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("failed to create subDir: %v", err)
	}

	// Create a non-JSON file
	if err := os.WriteFile(filepath.Join(tempDir, "notes.txt"), []byte("some notes"), 0644); err != nil {
		t.Fatalf("failed to create txt file: %v", err)
	}

	store, err := NewStorage(tempDir)
	if err != nil {
		t.Fatalf("NewStorage with subDir should succeed: %v", err)
	}

	// Save policy
	doc := &PolicyDocument{
		OrganizationID: "org-subtest",
		AuthorID:       "admin-1",
	}
	if err := store.SavePolicy(ctx, doc); err != nil {
		t.Fatalf("SavePolicy failed: %v", err)
	}

	// Reload
	store2, err := NewStorage(tempDir)
	if err != nil {
		t.Fatalf("NewStorage reload failed: %v", err)
	}
	loaded, err := store2.GetPolicy(ctx, "org-subtest")
	if err != nil || loaded == nil {
		t.Fatalf("GetPolicy failed after reload: %v", err)
	}
}

func TestPolicyDocument_Validation(t *testing.T) {
	// 1. Nil doc
	if err := ValidatePolicyDocument(nil); !errors.Is(err, ErrInvalidPolicy) {
		t.Errorf("expected ErrInvalidPolicy for nil doc, got %v", err)
	}

	// 2. Missing org ID
	if err := ValidatePolicyDocument(&PolicyDocument{}); !errors.Is(err, ErrInvalidPolicy) {
		t.Errorf("expected ErrInvalidPolicy for empty org ID, got %v", err)
	}

	// 3. Invalid org ID format (traversal)
	if err := ValidatePolicyDocument(&PolicyDocument{OrganizationID: "../bad"}); !errors.Is(err, ErrInvalidPolicy) {
		t.Errorf("expected ErrInvalidPolicy for traversal org ID, got %v", err)
	}

	// 4. Invalid version rule (min > max)
	badVerDoc := &PolicyDocument{
		OrganizationID: "org-test",
		VersionRules: []VersionRule{
			{
				RuleID:     "rule-1",
				PluginID:   "my-plugin",
				MinVersion: "2.0.0",
				MaxVersion: "1.0.0",
			},
		},
	}
	if err := ValidatePolicyDocument(badVerDoc); !errors.Is(err, ErrInvalidPolicy) {
		t.Errorf("expected ErrInvalidPolicy for min > max, got %v", err)
	}

	// 5. Negative max allowed permissions
	badSecDoc := &PolicyDocument{
		OrganizationID: "org-test",
		SecurityRules: []SecurityRule{
			{
				RuleID:                "sec-1",
				MaxAllowedPermissions: -1,
			},
		},
	}
	if err := ValidatePolicyDocument(badSecDoc); !errors.Is(err, ErrInvalidPolicy) {
		t.Errorf("expected ErrInvalidPolicy for negative permissions, got %v", err)
	}
}

func TestStorage_CorruptPolicyFailsLoad(t *testing.T) {
	tempDir := t.TempDir()
	// Write corrupt JSON to policy storage directory
	corruptFile := filepath.Join(tempDir, "corrupt.json")
	if err := os.WriteFile(corruptFile, []byte("{corrupt json"), 0644); err != nil {
		t.Fatalf("failed to write corrupt file: %v", err)
	}

	_, err := NewStorage(tempDir)
	if err == nil {
		t.Fatal("expected NewStorage to fail on corrupt policy file")
	}
}
