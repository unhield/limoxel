package verification

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/unhield/limoxel/plugin/security"
)

var (
	// ErrInvalidSignature indicates the cryptographic signature failed verification.
	ErrInvalidSignature = errors.New("digital signature verification failed: invalid signature")
	// ErrUnsupportedAlgorithm indicates the requested signature algorithm is not supported.
	ErrUnsupportedAlgorithm = errors.New("unsupported signature algorithm: only ed25519 is supported")
	// ErrMalformedSignature indicates the signature string was not valid base64 or had incorrect length.
	ErrMalformedSignature = errors.New("malformed digital signature")
)

// BuildSignPayload constructs the canonical byte slice that is signed and verified.
func BuildSignPayload(manifestDigest, artifactDigest, publisher, version string) []byte {
	// Canonical representation: fields separated by newline
	canonical := fmt.Sprintf("limoxel-plugin-signature:v1\nmanifest:%s\nartifact:%s\npublisher:%s\nversion:%s\n",
		strings.ToLower(strings.TrimSpace(manifestDigest)),
		strings.ToLower(strings.TrimSpace(artifactDigest)),
		strings.TrimSpace(publisher),
		strings.TrimSpace(version),
	)
	return []byte(canonical)
}

// SignPayload signs the canonical payload using an Ed25519 private key.
func SignPayload(privKey ed25519.PrivateKey, payload []byte) (string, error) {
	if len(privKey) != ed25519.PrivateKeySize {
		return "", errors.New("invalid Ed25519 private key size")
	}

	sig := ed25519.Sign(privKey, payload)
	return base64.StdEncoding.EncodeToString(sig), nil
}

// VerifySignature verifies the digital signature using an Ed25519 public key.
func VerifySignature(pubKey ed25519.PublicKey, payload []byte, base64Sig string) error {
	if len(pubKey) != ed25519.PublicKeySize {
		return fmt.Errorf("%w: public key must be %d bytes", ErrInvalidKeyFormat, ed25519.PublicKeySize)
	}

	sigBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(base64Sig))
	if err != nil {
		return fmt.Errorf("%w: failed to decode base64: %v", ErrMalformedSignature, err)
	}

	if len(sigBytes) != ed25519.SignatureSize {
		return fmt.Errorf("%w: expected %d bytes, got %d", ErrMalformedSignature, ed25519.SignatureSize, len(sigBytes))
	}

	if !ed25519.Verify(pubKey, payload, sigBytes) {
		return ErrInvalidSignature
	}

	return nil
}

// CreateSignedMetadata constructs a complete SignatureMetadata object given private key and payload details.
func CreateSignedMetadata(privKey ed25519.PrivateKey, keyID, publisher, manifestDigest, artifactDigest, version string) (*security.SignatureMetadata, error) {
	payload := BuildSignPayload(manifestDigest, artifactDigest, publisher, version)
	sigBase64, err := SignPayload(privKey, payload)
	if err != nil {
		return nil, err
	}

	return &security.SignatureMetadata{
		Algorithm: "ed25519",
		KeyID:     keyID,
		Publisher: publisher,
		SignedAt:  time.Now().UTC(),
		Signature: sigBase64,
	}, nil
}
