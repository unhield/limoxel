package registry

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"testing"

	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
	pubmarket "github.com/unhield/limoxel/plugin/marketplace"
)

func TestOrganizationRegistry_PublishAndDownload(t *testing.T) {
	tempDir := t.TempDir()
	rm, err := NewRegistryManager(tempDir)
	if err != nil {
		t.Fatalf("NewRegistryManager failed: %v", err)
	}

	ctx := context.Background()
	reg, err := rm.GetRegistry("org-acme")
	if err != nil {
		t.Fatalf("GetRegistry failed: %v", err)
	}

	payload := []byte("private enterprise archive bytes 1.0.0")
	v1, _ := version.ParseSemVer("1.0.0")

	// 1. Publish Private Plugin
	entry, err := reg.PublishPrivatePackage(ctx, "acme.payroll", "Payroll Plugin", "Internal payroll engine", "org-acme", v1, payload)
	if err != nil {
		t.Fatalf("PublishPrivatePackage failed: %v", err)
	}
	if entry.PluginID != "acme.payroll" || entry.Version != "1.0.0" {
		t.Errorf("unexpected entry: %+v", entry)
	}

	// 2. Prevent duplicate version
	_, err = reg.PublishPrivatePackage(ctx, "acme.payroll", "Payroll Plugin", "Internal payroll engine", "org-acme", v1, payload)
	if !errors.Is(err, ErrVersionExists) {
		t.Errorf("expected ErrVersionExists, got %v", err)
	}

	// 3. Download Artifact
	reader, downEntry, err := reg.DownloadArtifact(ctx, "acme.payroll", v1)
	if err != nil {
		t.Fatalf("DownloadArtifact failed: %v", err)
	}
	defer reader.Close()

	readBytes, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to read artifact: %v", err)
	}
	if string(readBytes) != string(payload) {
		t.Errorf("expected artifact payload %s, got %s", string(payload), string(readBytes))
	}
	if downEntry.DownloadCount != 1 {
		t.Errorf("expected download count 1, got %d", downEntry.DownloadCount)
	}
}

func TestOrganizationRegistry_PromotePublicPlugin(t *testing.T) {
	tempDir := t.TempDir()
	rm, _ := NewRegistryManager(tempDir)
	ctx := context.Background()
	reg, _ := rm.GetRegistry("org-acme")

	archiveData := []byte("public marketplace archive bytes v2.1.0")
	hasher := sha256.New()
	hasher.Write(archiveData)
	expectedDigest := hex.EncodeToString(hasher.Sum(nil))

	v2, _ := version.ParseSemVer("2.1.0")
	publicPlugin := &pubmarket.PluginDetail{
		PluginSummary: pubmarket.PluginSummary{
			ID:          "public.linter",
			Name:        "Public Linter",
			Description: "Community code linter",
		},
		Publisher: pubmarket.PublisherProfile{
			ID: "pub-community-dev",
		},
	}
	publicVersion := &pubmarket.VersionInfo{
		Version:        v2,
		ArtifactDigest: expectedDigest,
	}

	// 1. Successful promotion
	entry, err := reg.PromotePublicPlugin(
		ctx,
		publicPlugin,
		publicVersion,
		bytes.NewReader(archiveData),
		"sec-admin-1",
		"Approved after security code audit",
		[]string{"linter", "verified"},
	)
	if err != nil {
		t.Fatalf("PromotePublicPlugin failed: %v", err)
	}
	if !entry.IsPromotedPublic || entry.Promotion == nil {
		t.Fatalf("expected promoted public metadata, got: %+v", entry)
	}
	if entry.Promotion.OriginalPublisher != "pub-community-dev" {
		t.Errorf("expected original publisher preserved, got %s", entry.Promotion.OriginalPublisher)
	}
	if entry.Promotion.ApprovedBy != "sec-admin-1" {
		t.Errorf("expected approver sec-admin-1, got %s", entry.Promotion.ApprovedBy)
	}

	// 2. Substitution attack attempt (tampered archive data with mismatched digest)
	tamperedData := []byte("tampered malicious archive")
	_, err = reg.PromotePublicPlugin(
		ctx,
		publicPlugin,
		publicVersion,
		bytes.NewReader(tamperedData),
		"sec-admin-1",
		"Should fail",
		nil,
	)
	if !errors.Is(err, ErrIntegrityMismatch) && !errors.Is(err, ErrVersionExists) {
		t.Errorf("expected ErrIntegrityMismatch on digest mismatch, got %v", err)
	}
}

func TestOrganizationRegistry_TenantIsolation(t *testing.T) {
	tempDir := t.TempDir()
	rm, _ := NewRegistryManager(tempDir)
	ctx := context.Background()

	regA, _ := rm.GetRegistry("org-a")
	regB, _ := rm.GetRegistry("org-b")

	payload := []byte("secret org a plugin")
	v1, _ := version.ParseSemVer("1.0.0")

	_, _ = regA.PublishPrivatePackage(ctx, "secret.tool", "Secret Tool", "", "org-a", v1, payload)

	// Org B attempts to find Org A's plugin
	_, err := regB.GetPlugin(ctx, "secret.tool")
	if !errors.Is(err, ErrPluginNotFound) {
		t.Errorf("Org B should not find Org A's private plugin, got %v", err)
	}

	// Org B attempts to download Org A's plugin
	_, _, err = regB.DownloadArtifact(ctx, "secret.tool", v1)
	if !errors.Is(err, ErrPluginNotFound) {
		t.Errorf("Org B download should fail with ErrPluginNotFound, got %v", err)
	}
}
