package validation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/plugin/security/verification"
)

var (
	// ErrValidationFailed indicates one or more critical validation checks failed.
	ErrValidationFailed = errors.New("validation: package validation failed")
	// ErrDigestMismatch indicates artifact bytes did not match expected digest.
	ErrDigestMismatch = errors.New("validation: artifact digest mismatch")
)

// Pipeline coordinates complete automated package validation before publication.
type Pipeline struct {
	policy       ValidationPolicy
	scanner      MalwareScanner
	sigValidator *SignatureValidator
}

// NewPipeline creates an initialized validation pipeline.
func NewPipeline(
	policy ValidationPolicy,
	scanner MalwareScanner,
	trustStore *verification.TrustStore,
) *Pipeline {
	if scanner == nil {
		scanner = NewStaticHeuristicScanner()
	}
	return &Pipeline{
		policy:       policy,
		scanner:      scanner,
		sigValidator: NewSignatureValidator(trustStore),
	}
}

// ValidatePackage runs the full validation workflow on raw package archive data.
func (p *Pipeline) ValidatePackage(
	ctx context.Context,
	archiveData []byte,
	expectedDigest string,
) (*ValidationResult, *ExtractedPackage, error) {
	// 1. Calculate artifact SHA-256
	h := sha256.Sum256(archiveData)
	computedDigest := hex.EncodeToString(h[:])

	if expectedDigest != "" && computedDigest != expectedDigest {
		return nil, nil, fmt.Errorf("%w: expected %s, got %s", ErrDigestMismatch, expectedDigest, computedDigest)
	}

	result := &ValidationResult{
		ArtifactDigest: computedDigest,
		ValidatedAt:    time.Now().UTC(),
		Passed:         true,
	}

	// 2. Archive Safety & Manifest Schema Check
	reader := bytes.NewReader(archiveData)
	extracted, extractChecks, err := InspectAndExtract(reader, p.policy)
	result.Checks = append(result.Checks, extractChecks...)

	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, err.Error())
		return result, nil, ErrValidationFailed
	}

	result.PluginID = extracted.Manifest.ID
	result.Version = extracted.ParsedVersion
	result.ManifestDigest = extracted.ManifestDigest

	// 3. Malware Heuristic Scan
	scanCheck, err := p.scanner.Scan(ctx, extracted.Files, p.policy)
	result.Checks = append(result.Checks, scanCheck)
	if err != nil || !scanCheck.Passed {
		result.Passed = false
		result.Errors = append(result.Errors, fmt.Sprintf("Malware scan: %s", scanCheck.Details))
	}

	// 4. Digital Signature & Trust Verification
	sigCheck := p.sigValidator.VerifyPackageSignature(extracted, computedDigest, p.policy)
	result.Checks = append(result.Checks, sigCheck)
	if !sigCheck.Passed && sigCheck.Critical {
		result.Passed = false
		result.Errors = append(result.Errors, fmt.Sprintf("Signature verification: %s", sigCheck.Details))
	}

	// 5. Host Version Compatibility Check
	compatCheck := CheckCompatibility(extracted.Manifest.MinHostVersion, p.policy.CurrentHostVer)
	result.Checks = append(result.Checks, compatCheck)
	if !compatCheck.Passed && compatCheck.Critical {
		result.Passed = false
		result.Errors = append(result.Errors, fmt.Sprintf("Host compatibility: %s", compatCheck.Details))
	}

	// Final determination
	for _, chk := range result.Checks {
		if !chk.Passed && chk.Critical {
			result.Passed = false
		}
	}

	if !result.Passed {
		return result, extracted, ErrValidationFailed
	}

	return result, extracted, nil
}

// ValidateFromReader validates from an arbitrary stream by reading into a buffer with size guard.
func (p *Pipeline) ValidateFromReader(
	ctx context.Context,
	r io.Reader,
	expectedDigest string,
) (*ValidationResult, *ExtractedPackage, error) {
	// Guard maximum download buffer size (50MB)
	lr := io.LimitReader(r, p.policy.MaxUncompressed)
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed reading archive stream: %w", err)
	}

	return p.ValidatePackage(ctx, data, expectedDigest)
}
