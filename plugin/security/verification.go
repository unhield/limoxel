package security

import "time"

// TrustState represents the cryptographic verification and trust level of a plugin.
type TrustState string

const (
	// TrustUnknown indicates verification has not been performed.
	TrustUnknown TrustState = "UNKNOWN"
	// TrustUnverified indicates an unsigned plugin or missing cryptographic signature.
	TrustUnverified TrustState = "UNVERIFIED"
	// TrustVerified indicates valid cryptographic signature, but unlisted publisher.
	TrustVerified TrustState = "VERIFIED"
	// TrustTrusted indicates valid signature by an approved root or publisher.
	TrustTrusted TrustState = "TRUSTED"
	// TrustRejected indicates invalid signature, digest mismatch, or blacklisted key.
	TrustRejected TrustState = "REJECTED"
)

// String returns the string representation of the trust state.
func (s TrustState) String() string {
	return string(s)
}

// IsAcceptable returns true if the trust state is allowed to run under standard security policy.
func (s TrustState) IsAcceptable() bool {
	return s == TrustVerified || s == TrustTrusted
}

// SignatureMetadata encapsulates the signature header within a signed plugin distribution.
type SignatureMetadata struct {
	Algorithm string    `json:"algorithm"` // e.g., "ed25519"
	KeyID     string    `json:"key_id"`    // Publisher public key fingerprint or identifier
	Publisher string    `json:"publisher"` // Declared publisher identity
	SignedAt  time.Time `json:"signed_at"` // Timestamp when the signature was produced
	Signature string    `json:"signature"` // Base64-encoded raw digital signature
}

// VerificationResult contains the comprehensive outcome of verifying a plugin.
type VerificationResult struct {
	State       TrustState        `json:"state"`
	Valid       bool              `json:"valid"`
	Publisher   string            `json:"publisher,omitempty"`
	KeyID       string            `json:"key_id,omitempty"`
	Algorithm   string            `json:"algorithm,omitempty"`
	Errors      []string          `json:"errors,omitempty"`
	Warnings    []string          `json:"warnings,omitempty"`
	VerifiedAt  time.Time         `json:"verified_at"`
	DigestMatch bool              `json:"digest_match"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}
