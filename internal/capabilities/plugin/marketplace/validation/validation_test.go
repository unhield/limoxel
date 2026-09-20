package validation

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/plugin/security/verification"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
	"github.com/unhield/limoxel/plugin/security"
)

// helper to create an in-memory tar.gz archive
func createArchive(files map[string][]byte) []byte {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for name, data := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(data)),
		}
		_ = tw.WriteHeader(hdr)
		_, _ = tw.Write(data)
	}

	_ = tw.Close()
	_ = gw.Close()
	return buf.Bytes()
}

func TestValidation_ValidPackage(t *testing.T) {
	manifest := ManifestSchema{
		ID:             "my-linter",
		Name:           "My Linter",
		Version:        "1.0.0",
		Publisher:      "trusted-pub",
		MinHostVersion: "1.0.0",
	}
	manifestBytes, _ := json.Marshal(manifest)

	archiveData := createArchive(map[string][]byte{
		"plugin.json": manifestBytes,
		"rules.json":  []byte(`{"rules": ["no-unused"]}`),
	})

	pipeline := NewPipeline(DefaultPolicy(), nil, nil)
	result, pkg, err := pipeline.ValidatePackage(context.Background(), archiveData, "")
	if err != nil {
		t.Fatalf("expected valid package to pass, got err: %v, errors: %+v", err, result.Errors)
	}

	if !result.Passed {
		t.Fatalf("expected result.Passed to be true, checks: %+v", result.Checks)
	}
	if pkg.Manifest.ID != "my-linter" {
		t.Fatalf("expected plugin ID my-linter, got %s", pkg.Manifest.ID)
	}
}

func TestValidation_PathTraversalDefense(t *testing.T) {
	manifest := ManifestSchema{
		ID:        "traversal-plugin",
		Name:      "Traversal Plugin",
		Version:   "1.0.0",
		Publisher: "attacker",
	}
	manifestBytes, _ := json.Marshal(manifest)

	// Inject illegal directory traversal path
	archiveData := createArchive(map[string][]byte{
		"plugin.json":       manifestBytes,
		"../escape_dir.txt": []byte("malicious content"),
	})

	pipeline := NewPipeline(DefaultPolicy(), nil, nil)
	result, _, err := pipeline.ValidatePackage(context.Background(), archiveData, "")
	if err == nil {
		t.Fatal("expected traversal error, got nil")
	}

	if result.Passed {
		t.Fatal("expected validation to fail for directory traversal")
	}
}

func TestValidation_MalwareDetection(t *testing.T) {
	manifest := ManifestSchema{
		ID:        "malware-plugin",
		Name:      "Malware Plugin",
		Version:   "1.0.0",
		Publisher: "attacker",
	}
	manifestBytes, _ := json.Marshal(manifest)

	// Inject a fake Windows PE binary header
	peFake := []byte{0x4D, 0x5A, 0x90, 0x00, 0x03, 0x00, 0x00, 0x00}

	archiveData := createArchive(map[string][]byte{
		"plugin.json": manifestBytes,
		"hidden.bin":  peFake,
	})

	pipeline := NewPipeline(DefaultPolicy(), nil, nil)
	result, _, err := pipeline.ValidatePackage(context.Background(), archiveData, "")
	if err == nil {
		t.Fatal("expected malware heuristic error, got nil")
	}

	if result.Passed {
		t.Fatal("expected validation to fail for embedded binary")
	}
}

func TestValidation_MissingOrInvalidManifest(t *testing.T) {
	pipeline := NewPipeline(DefaultPolicy(), nil, nil)

	// 1. Missing manifest
	noManifest := createArchive(map[string][]byte{
		"readme.txt": []byte("hello"),
	})
	res, _, err := pipeline.ValidatePackage(context.Background(), noManifest, "")
	if err == nil || res.Passed {
		t.Fatal("expected failure for missing manifest")
	}

	// 2. Invalid plugin ID format (contains uppercase or illegal symbols)
	badManifest := ManifestSchema{
		ID:        "Bad_ID!",
		Name:      "Bad ID",
		Version:   "1.0.0",
		Publisher: "test",
	}
	badBytes, _ := json.Marshal(badManifest)
	badArchive := createArchive(map[string][]byte{
		"plugin.json": badBytes,
	})
	res, _, err = pipeline.ValidatePackage(context.Background(), badArchive, "")
	if err == nil || res.Passed {
		t.Fatal("expected failure for bad plugin ID format")
	}
}

func TestValidation_DigitalSignature(t *testing.T) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	trustStore := verification.NewTrustStore()
	if err := trustStore.RegisterKey("verified-pub", "key-1", pubKey, "Verified Key"); err != nil {
		t.Fatal(err)
	}

	manifest := ManifestSchema{
		ID:         "signed-plugin",
		Name:       "Signed Plugin",
		Version:    "1.0.0",
		Publisher:  "verified-pub",
		Entrypoint: "lib.js",
	}
	manifestBytes, _ := json.Marshal(manifest)

	hManifest := sha256.Sum256(manifestBytes)
	manifestDigest := hex.EncodeToString(hManifest[:])

	libBytes := []byte("console.log('hi');")
	hArtifact := sha256.Sum256(libBytes)
	artifactDigest := hex.EncodeToString(hArtifact[:])

	// Sign payload using verification helper
	payload := verification.BuildSignPayload(manifestDigest, artifactDigest, "verified-pub", "1.0.0")
	sigStr, err := verification.SignPayload(privKey, payload)
	if err != nil {
		t.Fatal(err)
	}

	sigMeta := security.SignatureMetadata{
		Algorithm: "ed25519",
		KeyID:     "key-1",
		Publisher: "verified-pub",
		Signature: sigStr,
	}
	sigBytes, _ := json.Marshal(sigMeta)

	// Now build signed package containing signature.json
	signedArchive := createArchive(map[string][]byte{
		"plugin.json":    manifestBytes,
		"lib.js":         []byte("console.log('hi');"),
		"signature.json": sigBytes,
	})

	policy := DefaultPolicy()
	policy.RequireSignature = true
	pipeline := NewPipeline(policy, nil, trustStore)

	// Validate against the expected artifactDigest
	result, _, err := pipeline.ValidatePackage(context.Background(), signedArchive, "")
	if err != nil {
		t.Fatalf("expected signed package to pass, got err: %v, errors: %+v", err, result.Errors)
	}
	if !result.Passed {
		t.Fatalf("expected validation passed, checks: %+v", result.Checks)
	}

	// Tampering test: alter the publisher in signature
	tamperedSigMeta := sigMeta
	tamperedSigMeta.Publisher = "malicious-actor"
	tamperedSigBytes, _ := json.Marshal(tamperedSigMeta)

	tamperedArchive := createArchive(map[string][]byte{
		"plugin.json":    manifestBytes,
		"lib.js":         []byte("console.log('hi');"),
		"signature.json": tamperedSigBytes,
	})

	tamperedRes, _, tamperedErr := pipeline.ValidatePackage(context.Background(), tamperedArchive, "")
	if tamperedErr == nil || tamperedRes.Passed {
		t.Fatal("expected tampered signature to fail validation")
	}

	// Adversarial test: Sign arbitrary file (readme.txt) instead of declared entrypoint or archive
	readmeBytes := []byte("arbitrary readme content")
	hReadme := sha256.Sum256(readmeBytes)
	readmeDigest := hex.EncodeToString(hReadme[:])
	readmePayload := verification.BuildSignPayload(manifestDigest, readmeDigest, "verified-pub", "1.0.0")
	readmeSigStr, err := verification.SignPayload(privKey, readmePayload)
	if err != nil {
		t.Fatal(err)
	}
	readmeSigMeta := security.SignatureMetadata{
		Algorithm: "ed25519",
		KeyID:     "key-1",
		Publisher: "verified-pub",
		Signature: readmeSigStr,
	}
	readmeSigBytes, _ := json.Marshal(readmeSigMeta)
	readmeArchive := createArchive(map[string][]byte{
		"plugin.json":    manifestBytes,
		"lib.js":         []byte("console.log('hi');"),
		"readme.txt":     readmeBytes,
		"signature.json": readmeSigBytes,
	})
	readmeRes, _, readmeErr := pipeline.ValidatePackage(context.Background(), readmeArchive, "")
	if readmeErr == nil || readmeRes.Passed {
		t.Fatal("expected signature over non-entrypoint/non-archive file to be rejected")
	}
}

func TestValidation_HostCompatibility(t *testing.T) {
	currentHost := version.SemVer{Major: 1, Minor: 5, Patch: 0}

	// 1. Host satisfies min version 1.2.0
	res := CheckCompatibility("1.2.0", currentHost)
	if !res.Passed {
		t.Fatalf("expected compatible host, got: %s", res.Details)
	}

	// 2. Host fails min version 2.0.0
	res = CheckCompatibility("2.0.0", currentHost)
	if res.Passed {
		t.Fatalf("expected incompatible host for 2.0.0")
	}
}
