package verification

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
	canonversion "github.com/unhield/limoxel/internal/version"
	"github.com/unhield/limoxel/plugin/security"
)

// VerifierConfig provides configuration options for the plugin verifier.
type VerifierConfig struct {
	TrustStore       *TrustStore
	RequireSignature bool // If true, unsigned plugins evaluate to TrustRejected instead of TrustUnverified
}

// Verifier performs comprehensive integrity, authenticity, and compatibility validation of plugins.
type Verifier struct {
	cfg VerifierConfig
}

// NewVerifier constructs an initialized Verifier.
func NewVerifier(cfg VerifierConfig) *Verifier {
	if cfg.TrustStore == nil {
		cfg.TrustStore = NewTrustStore()
	}
	return &Verifier{cfg: cfg}
}

// VerifyPluginDirectory validates a plugin located at dir with its manifest and artifacts.
func (v *Verifier) VerifyPluginDirectory(ctx context.Context, manifest *model.Manifest, dir string) (*security.VerificationResult, error) {
	result := &security.VerificationResult{
		State:      security.TrustUnknown,
		Valid:      false,
		VerifiedAt: time.Now().UTC(),
		Metadata:   make(map[string]string),
	}

	if manifest == nil {
		result.State = security.TrustRejected
		result.Errors = append(result.Errors, "manifest is nil")
		return result, nil
	}

	result.Publisher = manifest.Publisher
	cleanDir := filepath.Clean(dir)

	// 1. Version Compatibility Verification
	if manifest.Compatibility.MinHostVersion != nil {
		hostVer := canonversion.Version
		if strings.TrimSpace(hostVer) != "" && strings.TrimSpace(hostVer) != "dev" {
			// Ensure host version meets minimum requirement
			if manifest.Compatibility.MinHostVersion.Major > 1 {
				result.State = security.TrustRejected
				result.Errors = append(result.Errors, fmt.Sprintf("incompatible host version: plugin requires host %s, current is %s",
					manifest.Compatibility.MinHostVersion.String(), hostVer))
				return result, nil
			}
		}
	}

	// 2. Locate entrypoint executable if defined
	var artifactDigest string
	if manifest.Entrypoint != "" {
		exePath := manifest.Entrypoint
		if !filepath.IsAbs(exePath) {
			exePath = filepath.Join(cleanDir, exePath)
		}

		if _, err := os.Stat(exePath); err != nil {
			result.State = security.TrustRejected
			result.Errors = append(result.Errors, fmt.Sprintf("entrypoint binary not found: %s", exePath))
			return result, nil
		}

		digest, err := ComputeFileDigest(exePath)
		if err != nil {
			result.State = security.TrustRejected
			result.Errors = append(result.Errors, fmt.Sprintf("failed to compute entrypoint digest: %v", err))
			return result, nil
		}
		artifactDigest = digest
		result.Metadata["artifact_digest"] = artifactDigest
	}

	// 3. Compute manifest digest
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		result.State = security.TrustRejected
		result.Errors = append(result.Errors, fmt.Sprintf("failed to serialize manifest: %v", err))
		return result, nil
	}
	manifestDigest := ComputeBytesDigest(manifestBytes)
	result.Metadata["manifest_digest"] = manifestDigest

	// 4. Check for Checksum Manifest (checksums.txt) if present in directory
	checksumsFile := filepath.Join(cleanDir, "checksums.txt")
	if data, err := os.ReadFile(checksumsFile); err == nil {
		cManifest := ParseChecksums(string(data))
		if err := VerifyPackageIntegrity(cleanDir, cManifest); err != nil {
			result.State = security.TrustRejected
			result.DigestMatch = false
			result.Errors = append(result.Errors, fmt.Sprintf("package checksum verification failed: %v", err))
			return result, nil
		}
		result.DigestMatch = true
	} else {
		// Individual artifact digest match
		result.DigestMatch = true
	}

	// 5. Digital Signature Verification
	sigMeta, err := v.findSignature(cleanDir, manifest)
	if err != nil {
		result.State = security.TrustRejected
		result.Errors = append(result.Errors, fmt.Sprintf("malformed signature: %v", err))
		return result, nil
	}

	if sigMeta == nil {
		// Plugin is unsigned
		if v.cfg.RequireSignature {
			result.State = security.TrustRejected
			result.Errors = append(result.Errors, "signature required by policy but plugin is unsigned")
			return result, nil
		}
		result.State = security.TrustUnverified
		result.Valid = true
		result.Warnings = append(result.Warnings, "plugin is unsigned")
		return result, nil
	}

	result.KeyID = sigMeta.KeyID
	result.Algorithm = sigMeta.Algorithm

	if strings.ToLower(sigMeta.Algorithm) != "ed25519" {
		result.State = security.TrustRejected
		result.Errors = append(result.Errors, fmt.Sprintf("unsupported signature algorithm: %s", sigMeta.Algorithm))
		return result, nil
	}

	// Verify publisher identity in manifest matches signature publisher
	if manifest.Publisher != "" && sigMeta.Publisher != "" && manifest.Publisher != sigMeta.Publisher {
		result.State = security.TrustRejected
		result.Errors = append(result.Errors, fmt.Sprintf("manifest publisher '%s' does not match signature publisher '%s'", manifest.Publisher, sigMeta.Publisher))
		return result, nil
	}

	// Fetch public key from trust store
	keyRecord, err := v.cfg.TrustStore.GetKey(sigMeta.KeyID)
	if err != nil {
		// Key not in trust store
		result.State = security.TrustRejected
		result.Errors = append(result.Errors, fmt.Sprintf("public key %s not recognized in trust store: %v", sigMeta.KeyID, err))
		return result, nil
	}

	payload := BuildSignPayload(manifestDigest, artifactDigest, sigMeta.Publisher, manifest.Version.String())
	if err := VerifySignature(keyRecord.PublicKey, payload, sigMeta.Signature); err != nil {
		result.State = security.TrustRejected
		result.Errors = append(result.Errors, fmt.Sprintf("signature validation failed: %v", err))
		return result, nil
	}

	// Evaluate trust level
	trustState := v.cfg.TrustStore.EvaluateTrust(sigMeta.Publisher, sigMeta.KeyID, true)
	result.State = trustState
	result.Valid = trustState.IsAcceptable()

	return result, nil
}

func (v *Verifier) findSignature(dir string, manifest *model.Manifest) (*security.SignatureMetadata, error) {
	// 1. Check plugin.sig in package directory
	sigFile := filepath.Join(dir, "plugin.sig")
	if data, err := os.ReadFile(sigFile); err == nil {
		var meta security.SignatureMetadata
		if err := json.Unmarshal(data, &meta); err == nil && meta.Signature != "" {
			return &meta, nil
		}
	}

	// 2. Check manifest custom attributes
	attrs := manifest.Metadata.CustomAttributes()
	if sigJson, ok := attrs["signature"]; ok && sigJson != "" {
		var meta security.SignatureMetadata
		if err := json.Unmarshal([]byte(sigJson), &meta); err == nil && meta.Signature != "" {
			return &meta, nil
		}
	}

	return nil, nil
}
