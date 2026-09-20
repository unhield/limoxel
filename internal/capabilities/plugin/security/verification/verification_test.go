package verification

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
	"github.com/unhield/limoxel/plugin/security"
)

func TestIntegrityVerification_BytesAndFiles(t *testing.T) {
	data := []byte("hello limoxel security")
	digest := ComputeBytesDigest(data)
	if digest == "" {
		t.Fatal("expected non-empty digest")
	}

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	fileDigest, err := ComputeFileDigest(filePath)
	if err != nil {
		t.Fatalf("failed to compute file digest: %v", err)
	}

	if digest != fileDigest {
		t.Fatalf("expected matching digests, got %s vs %s", digest, fileDigest)
	}

	if err := VerifyFileDigest(filePath, digest); err != nil {
		t.Fatalf("expected verify file digest to pass: %v", err)
	}

	// Tampered digest check
	tamperedDigest := "0000000000000000000000000000000000000000000000000000000000000000"
	if err := VerifyFileDigest(filePath, tamperedDigest); err == nil {
		t.Fatal("expected tampered digest to fail verification")
	}
}

func TestIntegrityVerification_PackageChecksums(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := filepath.Join(tmpDir, "a.txt")
	f2 := filepath.Join(tmpDir, "b.txt")
	if err := os.WriteFile(f1, []byte("file a content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f2, []byte("file b content"), 0644); err != nil {
		t.Fatal(err)
	}

	h1 := ComputeBytesDigest([]byte("file a content"))
	h2 := ComputeBytesDigest([]byte("file b content"))

	checksumText := h1 + "  a.txt\n" + h2 + "  b.txt\n"
	manifest := ParseChecksums(checksumText)

	if len(manifest) != 2 {
		t.Fatalf("expected 2 checksum entries, got %d", len(manifest))
	}

	if err := VerifyPackageIntegrity(tmpDir, manifest); err != nil {
		t.Fatalf("expected package integrity to pass: %v", err)
	}

	// Tamper file a
	if err := os.WriteFile(f1, []byte("file a tampered"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := VerifyPackageIntegrity(tmpDir, manifest); err == nil {
		t.Fatal("expected package integrity to fail on tampered file")
	}
}

func TestSignatureVerification_ValidAndTampered(t *testing.T) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate ed25519 key: %v", err)
	}

	payload := BuildSignPayload("manifesthash123", "artifacthash456", "test-publisher", "1.0.0")
	sig, err := SignPayload(privKey, payload)
	if err != nil {
		t.Fatalf("failed to sign payload: %v", err)
	}

	// Valid verification
	if err := VerifySignature(pubKey, payload, sig); err != nil {
		t.Fatalf("expected signature verification to pass: %v", err)
	}

	// Tampered payload verification
	tamperedPayload := BuildSignPayload("manifesthash999", "artifacthash456", "test-publisher", "1.0.0")
	if err := VerifySignature(pubKey, tamperedPayload, sig); err == nil {
		t.Fatal("expected verification to fail for tampered payload")
	}

	// Wrong public key
	wrongPubKey, _, _ := ed25519.GenerateKey(rand.Reader)
	if err := VerifySignature(wrongPubKey, payload, sig); err == nil {
		t.Fatal("expected verification to fail with wrong public key")
	}
}

func TestTrustStore_KeyLifecycleAndTrustEvaluation(t *testing.T) {
	store := NewTrustStore()
	pubKey, _, _ := ed25519.GenerateKey(rand.Reader)

	keyID := "key-alice-1"
	if err := store.RegisterKey("alice", keyID, pubKey, "Alice testing key"); err != nil {
		t.Fatalf("failed to register key: %v", err)
	}

	// Trusted state
	state := store.EvaluateTrust("alice", keyID, true)
	if state != security.TrustTrusted {
		t.Fatalf("expected TrustTrusted, got %s", state)
	}

	// Invalid signature
	state = store.EvaluateTrust("alice", keyID, false)
	if state != security.TrustRejected {
		t.Fatalf("expected TrustRejected for invalid sig, got %s", state)
	}

	// Publisher mismatch
	state = store.EvaluateTrust("bob", keyID, true)
	if state != security.TrustRejected {
		t.Fatalf("expected TrustRejected for publisher mismatch, got %s", state)
	}

	// Unknown key ID
	state = store.EvaluateTrust("alice", "unknown-key", true)
	if state != security.TrustVerified {
		t.Fatalf("expected TrustVerified for unknown key with valid sig, got %s", state)
	}

	// Key revocation
	if err := store.RevokeKey(keyID); err != nil {
		t.Fatalf("failed to revoke key: %v", err)
	}

	state = store.EvaluateTrust("alice", keyID, true)
	if state != security.TrustRejected {
		t.Fatalf("expected TrustRejected for revoked key, got %s", state)
	}
}

func TestVerifier_PluginDirectoryVerification(t *testing.T) {
	pubKey, privKey, _ := ed25519.GenerateKey(rand.Reader)
	store := NewTrustStore()
	keyID := "trusted-pub-1"
	_ = store.RegisterKey("acme-corp", keyID, pubKey, "Acme official key")

	verifier := NewVerifier(VerifierConfig{TrustStore: store})

	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "plugin.exe")
	if err := os.WriteFile(binPath, []byte("binary bytecode executable"), 0755); err != nil {
		t.Fatal(err)
	}

	manifest := &model.Manifest{
		SchemaVersion: "1.0.0",
		ID:            model.Identity("acme-plugin"),
		Name:          "Acme Plugin",
		Version:       version.SemVer{Major: 1, Minor: 0, Patch: 0},
		Publisher:     "acme-corp",
		Entrypoint:    "plugin.exe",
	}

	// Compute digests
	manifestBytes, _ := json.Marshal(manifest)
	manifestDigest := ComputeBytesDigest(manifestBytes)
	artifactDigest, _ := ComputeFileDigest(binPath)

	sigMeta, err := CreateSignedMetadata(privKey, keyID, "acme-corp", manifestDigest, artifactDigest, "1.0.0")
	if err != nil {
		t.Fatalf("failed to create signed metadata: %v", err)
	}

	// Write plugin.sig
	sigBytes, _ := json.Marshal(sigMeta)
	if err := os.WriteFile(filepath.Join(tmpDir, "plugin.sig"), sigBytes, 0644); err != nil {
		t.Fatal(err)
	}

	// Test successful verification
	res, err := verifier.VerifyPluginDirectory(context.Background(), manifest, tmpDir)
	if err != nil {
		t.Fatalf("verify directory returned unexpected error: %v", err)
	}
	if !res.Valid {
		t.Fatalf("expected valid result, got errors: %v", res.Errors)
	}
	if res.State != security.TrustTrusted {
		t.Fatalf("expected TrustTrusted, got %s", res.State)
	}

	// Test Tampered Binary
	if err := os.WriteFile(binPath, []byte("altered binary bytecode"), 0755); err != nil {
		t.Fatal(err)
	}
	resTampered, _ := verifier.VerifyPluginDirectory(context.Background(), manifest, tmpDir)
	if resTampered.Valid {
		t.Fatal("expected verification to fail after binary was tampered")
	}
	if resTampered.State != security.TrustRejected {
		t.Fatalf("expected TrustRejected, got %s", resTampered.State)
	}
}

func TestVerifier_RejectsPublisherMismatch(t *testing.T) {
	pubKey, privKey, _ := ed25519.GenerateKey(rand.Reader)
	store := NewTrustStore()
	keyID := "pub-key-1"
	_ = store.RegisterKey("real-publisher", keyID, pubKey, "Real publisher key")

	verifier := NewVerifier(VerifierConfig{TrustStore: store})
	tmpDir := t.TempDir()

	manifest := &model.Manifest{
		SchemaVersion: "1.0.0",
		ID:            model.Identity("mismatch-plugin"),
		Name:          "Mismatch Plugin",
		Version:       version.SemVer{Major: 1, Minor: 0, Patch: 0},
		Publisher:     "evil-imposter", // Manifest claims to be evil-imposter
	}

	manifestBytes, _ := json.Marshal(manifest)
	manifestDigest := ComputeBytesDigest(manifestBytes)

	// Signature metadata was created for "real-publisher"
	sigMeta, err := CreateSignedMetadata(privKey, keyID, "real-publisher", manifestDigest, "", "1.0.0")
	if err != nil {
		t.Fatal(err)
	}

	sigBytes, _ := json.Marshal(sigMeta)
	if err := os.WriteFile(filepath.Join(tmpDir, "plugin.sig"), sigBytes, 0644); err != nil {
		t.Fatal(err)
	}

	res, err := verifier.VerifyPluginDirectory(context.Background(), manifest, tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if res.Valid {
		t.Fatal("expected verification to reject publisher mismatch")
	}
	if res.State != security.TrustRejected {
		t.Fatalf("expected TrustRejected, got %s", res.State)
	}
}
