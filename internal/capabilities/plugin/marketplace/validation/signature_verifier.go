package validation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/plugin/security/verification"
	"github.com/unhield/limoxel/plugin/security"
)

// SignatureValidator verifies digital signatures and trust states of plugin packages.
type SignatureValidator struct {
	trustStore *verification.TrustStore
}

// NewSignatureValidator creates an initialized SignatureValidator.
func NewSignatureValidator(trustStore *verification.TrustStore) *SignatureValidator {
	return &SignatureValidator{
		trustStore: trustStore,
	}
}

// VerifyPackageSignature checks the signature against package manifest, artifact digest, and trust store.
func (v *SignatureValidator) VerifyPackageSignature(
	pkg *ExtractedPackage,
	artifactDigest string,
	policy ValidationPolicy,
) CheckResult {
	if v.trustStore == nil {
		if policy.RequireSignature {
			return CheckResult{
				Name:     "SignatureVerification",
				Passed:   false,
				Details:  "No trust store configured, but signatures are required by policy",
				Critical: true,
			}
		}
		return CheckResult{
			Name:     "SignatureVerification",
			Passed:   true,
			Details:  "Package signature check bypassed: no trust store configured",
			Critical: false,
		}
	}

	// Look for signature.json
	var sigData []byte
	for path, content := range pkg.Files {
		if filepath.Base(path) == "signature.json" {
			sigData = content
			break
		}
	}

	if sigData == nil {
		if policy.RequireSignature {
			return CheckResult{
				Name:     "SignatureVerification",
				Passed:   false,
				Details:  "Package missing required signature.json",
				Critical: true,
			}
		}
		return CheckResult{
			Name:     "SignatureVerification",
			Passed:   true,
			Details:  "Package is unsigned (allowed by current policy)",
			Critical: false,
		}
	}

	var sigMeta security.SignatureMetadata
	if err := json.Unmarshal(sigData, &sigMeta); err != nil {
		return CheckResult{
			Name:     "SignatureVerification",
			Passed:   false,
			Details:  fmt.Sprintf("Corrupt signature.json: %v", err),
			Critical: true,
		}
	}

	// 1. Check publisher alignment
	if sigMeta.Publisher != pkg.Manifest.Publisher {
		return CheckResult{
			Name:     "SignatureVerification",
			Passed:   false,
			Details:  fmt.Sprintf("Publisher mismatch: signature claims '%s', manifest has '%s'", sigMeta.Publisher, pkg.Manifest.Publisher),
			Critical: true,
		}
	}

	// 2. Lookup key in trust store
	key, err := v.trustStore.GetKey(sigMeta.KeyID)
	if err != nil {
		return CheckResult{
			Name:     "SignatureVerification",
			Passed:   false,
			Details:  fmt.Sprintf("Unknown signing key '%s' in trust store: %v", sigMeta.KeyID, err),
			Critical: true,
		}
	}

	if key.Revoked {
		return CheckResult{
			Name:     "SignatureVerification",
			Passed:   false,
			Details:  fmt.Sprintf("Signing key '%s' has been revoked", sigMeta.KeyID),
			Critical: true,
		}
	}

	// 3. Build sign payload and verify cryptographic signature against candidate digests:
	// Candidate 1: The archive digest itself (package-level detached signature)
	// Candidate 2: The declared entrypoint executable digest (embedded artifact signature)
	candidates := []string{artifactDigest}
	if ep := strings.TrimSpace(pkg.Manifest.Entrypoint); ep != "" {
		cleanEP := filepath.ToSlash(filepath.Clean(ep))
		if data, ok := pkg.Files[cleanEP]; ok {
			h := sha256.Sum256(data)
			candidates = append(candidates, hex.EncodeToString(h[:]))
		}
	}

	sigVerified := false
	var lastErr error

	for _, candDigest := range candidates {
		payload := verification.BuildSignPayload(
			pkg.ManifestDigest,
			candDigest,
			pkg.Manifest.Publisher,
			pkg.ParsedVersion.String(),
		)

		if err := verification.VerifySignature(key.PublicKey, payload, sigMeta.Signature); err == nil {
			sigVerified = true
			break
		} else {
			lastErr = err
		}
	}

	if !sigVerified {
		return CheckResult{
			Name:     "SignatureVerification",
			Passed:   false,
			Details:  fmt.Sprintf("Signature verification failed: %v", lastErr),
			Critical: true,
		}
	}

	// 4. Evaluate trust state
	trustState := v.trustStore.EvaluateTrust(pkg.Manifest.Publisher, sigMeta.KeyID, true)
	if !trustState.IsAcceptable() {
		return CheckResult{
			Name:     "SignatureVerification",
			Passed:   false,
			Details:  fmt.Sprintf("Publisher/key combination is not acceptable: state %s", trustState),
			Critical: true,
		}
	}

	return CheckResult{
		Name:     "SignatureVerification",
		Passed:   true,
		Details:  fmt.Sprintf("Cryptographically verified signature from trusted publisher '%s' (key %s)", pkg.Manifest.Publisher, sigMeta.KeyID),
		Critical: true,
	}
}
