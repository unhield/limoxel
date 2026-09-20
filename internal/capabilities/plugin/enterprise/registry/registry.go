package registry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
	pubmarket "github.com/unhield/limoxel/plugin/marketplace"
)

var (
	ErrPluginNotFound    = errors.New("enterprise registry: private plugin not found")
	ErrVersionNotFound   = errors.New("enterprise registry: plugin version not found")
	ErrVersionExists     = errors.New("enterprise registry: plugin version already exists in organization registry")
	ErrAccessDenied      = errors.New("enterprise registry: access denied to private organization registry")
	ErrIntegrityMismatch = errors.New("enterprise registry: artifact SHA-256 digest mismatch")
)

// PromotionMetadata contains enterprise compliance and approval details attached to a public plugin.
type PromotionMetadata struct {
	OriginalPublisher string    `json:"original_publisher"`
	ApprovedBy        string    `json:"approved_by"`
	ApprovedAt        time.Time `json:"approved_at"`
	ComplianceNotes   string    `json:"compliance_notes,omitempty"`
	InternalTags      []string  `json:"internal_tags,omitempty"`
}

// PrivateVersionEntry describes a released version in an organization registry.
type PrivateVersionEntry struct {
	PluginID         string             `json:"plugin_id"`
	Version          string             `json:"version"`
	ArtifactDigest   string             `json:"artifact_digest"`
	ArtifactSize     int64              `json:"artifact_size"`
	CreatedAt        time.Time          `json:"created_at"`
	IsPromotedPublic bool               `json:"is_promoted_public"`
	Promotion        *PromotionMetadata `json:"promotion,omitempty"`
	DownloadCount    int64              `json:"download_count"`
}

// PrivatePluginDetail holds complete metadata for a private or approved plugin in an org.
type PrivatePluginDetail struct {
	OrganizationID string                         `json:"organization_id"`
	PluginID       string                         `json:"plugin_id"`
	Name           string                         `json:"name"`
	Description    string                         `json:"description"`
	PublisherID    string                         `json:"publisher_id"`
	IsPrivate      bool                           `json:"is_private"`
	Versions       map[string]PrivateVersionEntry `json:"versions"`
	CreatedAt      time.Time                      `json:"created_at"`
	UpdatedAt      time.Time                      `json:"updated_at"`
}

// OrganizationRegistry manages an organization's private and approved plugins.
type OrganizationRegistry struct {
	mu         sync.RWMutex
	baseDir    string
	orgID      string
	plugins    map[string]*PrivatePluginDetail
	storageDir string
}

// RegistryManager coordinates tenant-isolated organization registries.
type RegistryManager struct {
	mu         sync.RWMutex
	rootDir    string
	registries map[string]*OrganizationRegistry
}

// NewRegistryManager constructs a manager for tenant-isolated organization registries.
func NewRegistryManager(rootDir string) (*RegistryManager, error) {
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create enterprise registry root: %w", err)
	}

	return &RegistryManager{
		rootDir:    rootDir,
		registries: make(map[string]*OrganizationRegistry),
	}, nil
}

// GetRegistry returns or initializes an organization-specific registry.
func (rm *RegistryManager) GetRegistry(orgID string) (*OrganizationRegistry, error) {
	if orgID == "" {
		return nil, errors.New("enterprise registry: organization ID is required")
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()

	if reg, exists := rm.registries[orgID]; exists {
		return reg, nil
	}

	orgDir := filepath.Join(rm.rootDir, orgID)
	artifactsDir := filepath.Join(orgDir, "artifacts")
	if err := os.MkdirAll(artifactsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to initialize organization registry dir: %w", err)
	}

	reg := &OrganizationRegistry{
		baseDir:    orgDir,
		orgID:      orgID,
		plugins:    make(map[string]*PrivatePluginDetail),
		storageDir: artifactsDir,
	}

	if err := reg.load(); err != nil {
		return nil, err
	}

	rm.registries[orgID] = reg
	return reg, nil
}

func (r *OrganizationRegistry) catalogFile() string {
	return filepath.Join(r.baseDir, "catalog.json")
}

func (r *OrganizationRegistry) load() error {
	data, err := os.ReadFile(r.catalogFile())
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read org registry catalog: %w", err)
	}

	var items map[string]*PrivatePluginDetail
	if err := json.Unmarshal(data, &items); err != nil {
		return fmt.Errorf("failed to parse org registry catalog: %w", err)
	}

	r.plugins = items
	return nil
}

func (r *OrganizationRegistry) saveLocked() error {
	data, err := json.MarshalIndent(r.plugins, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal catalog: %w", err)
	}

	tempPath := filepath.Join(r.baseDir, fmt.Sprintf(".catalog-%d.tmp", time.Now().UnixNano()))
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp catalog: %w", err)
	}

	if err := os.Rename(tempPath, r.catalogFile()); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to commit catalog: %w", err)
	}

	return nil
}

// PublishPrivatePackage stores and registers a proprietary private plugin archive.
func (r *OrganizationRegistry) PublishPrivatePackage(
	ctx context.Context,
	pluginID, name, desc, publisherID string,
	ver version.SemVer,
	archiveData []byte,
) (*PrivateVersionEntry, error) {
	if len(archiveData) == 0 {
		return nil, errors.New("enterprise registry: archive data cannot be empty")
	}

	hasher := sha256.New()
	hasher.Write(archiveData)
	digest := hex.EncodeToString(hasher.Sum(nil))

	r.mu.Lock()
	defer r.mu.Unlock()

	detail := r.plugins[pluginID]
	now := time.Now().UTC()
	if detail == nil {
		detail = &PrivatePluginDetail{
			OrganizationID: r.orgID,
			PluginID:       pluginID,
			Name:           name,
			Description:    desc,
			PublisherID:    publisherID,
			IsPrivate:      true,
			Versions:       make(map[string]PrivateVersionEntry),
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		r.plugins[pluginID] = detail
	}

	verStr := ver.String()
	if _, exists := detail.Versions[verStr]; exists {
		return nil, ErrVersionExists
	}

	// Store artifact content-addressed
	artifactPath := filepath.Join(r.storageDir, fmt.Sprintf("%s.tar.gz", digest))
	if err := os.WriteFile(artifactPath, archiveData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write artifact: %w", err)
	}

	entry := PrivateVersionEntry{
		PluginID:         pluginID,
		Version:          verStr,
		ArtifactDigest:   digest,
		ArtifactSize:     int64(len(archiveData)),
		CreatedAt:        now,
		IsPromotedPublic: false,
		DownloadCount:    0,
	}

	detail.Versions[verStr] = entry
	detail.UpdatedAt = now

	if err := r.saveLocked(); err != nil {
		delete(detail.Versions, verStr)
		_ = os.Remove(artifactPath)
		return nil, err
	}

	return &entry, nil
}

// PromotePublicPlugin approves and mirrors a public marketplace plugin into the enterprise registry.
func (r *OrganizationRegistry) PromotePublicPlugin(
	ctx context.Context,
	publicPlugin *pubmarket.PluginDetail,
	publicVersion *pubmarket.VersionInfo,
	archiveReader io.Reader,
	approverID, notes string,
	internalTags []string,
) (*PrivateVersionEntry, error) {
	if publicPlugin == nil || publicVersion == nil {
		return nil, errors.New("enterprise registry: public plugin and version info required")
	}

	// Stream archive to temp and verify digest
	tempPath := filepath.Join(r.storageDir, fmt.Sprintf(".promoted-%d.tmp", time.Now().UnixNano()))
	f, err := os.Create(tempPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp promotion file: %w", err)
	}

	hasher := sha256.New()
	multiWriter := io.MultiWriter(f, hasher)
	size, err := io.Copy(multiWriter, archiveReader)
	closeErr := f.Close()
	if err != nil {
		_ = os.Remove(tempPath)
		return nil, fmt.Errorf("failed to stream promotion artifact: %w", err)
	}
	if closeErr != nil {
		_ = os.Remove(tempPath)
		return nil, fmt.Errorf("failed to close promotion temp file: %w", closeErr)
	}

	computedDigest := hex.EncodeToString(hasher.Sum(nil))
	if computedDigest != publicVersion.ArtifactDigest {
		_ = os.Remove(tempPath)
		return nil, fmt.Errorf("%w: expected %s, got %s", ErrIntegrityMismatch, publicVersion.ArtifactDigest, computedDigest)
	}

	finalArtifactPath := filepath.Join(r.storageDir, fmt.Sprintf("%s.tar.gz", computedDigest))
	if err := os.Rename(tempPath, finalArtifactPath); err != nil {
		_ = os.Remove(tempPath)
		return nil, fmt.Errorf("failed to commit promoted artifact: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	detail := r.plugins[publicPlugin.ID]
	now := time.Now().UTC()
	if detail == nil {
		detail = &PrivatePluginDetail{
			OrganizationID: r.orgID,
			PluginID:       publicPlugin.ID,
			Name:           publicPlugin.Name,
			Description:    publicPlugin.Description,
			PublisherID:    publicPlugin.Publisher.ID,
			IsPrivate:      false,
			Versions:       make(map[string]PrivateVersionEntry),
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		r.plugins[publicPlugin.ID] = detail
	}

	verStr := publicVersion.Version.String()
	if _, exists := detail.Versions[verStr]; exists {
		return nil, ErrVersionExists
	}

	entry := PrivateVersionEntry{
		PluginID:         publicPlugin.ID,
		Version:          verStr,
		ArtifactDigest:   computedDigest,
		ArtifactSize:     size,
		CreatedAt:        now,
		IsPromotedPublic: true,
		Promotion: &PromotionMetadata{
			OriginalPublisher: publicPlugin.Publisher.ID,
			ApprovedBy:        approverID,
			ApprovedAt:        now,
			ComplianceNotes:   notes,
			InternalTags:      internalTags,
		},
		DownloadCount: 0,
	}

	detail.Versions[verStr] = entry
	detail.UpdatedAt = now

	if err := r.saveLocked(); err != nil {
		delete(detail.Versions, verStr)
		return nil, err
	}

	return &entry, nil
}

// GetPlugin returns the private or promoted plugin details.
func (r *OrganizationRegistry) GetPlugin(ctx context.Context, pluginID string) (*PrivatePluginDetail, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	detail, exists := r.plugins[pluginID]
	if !exists {
		return nil, ErrPluginNotFound
	}
	clone := *detail
	return &clone, nil
}

// DownloadArtifact retrieves the artifact stream and verifies download integrity.
func (r *OrganizationRegistry) DownloadArtifact(ctx context.Context, pluginID string, ver version.SemVer) (io.ReadCloser, *PrivateVersionEntry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	detail, exists := r.plugins[pluginID]
	if !exists {
		return nil, nil, ErrPluginNotFound
	}

	entry, exists := detail.Versions[ver.String()]
	if !exists {
		return nil, nil, ErrVersionNotFound
	}

	artifactPath := filepath.Join(r.storageDir, fmt.Sprintf("%s.tar.gz", entry.ArtifactDigest))
	f, err := os.Open(artifactPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open artifact file: %w", err)
	}

	// Increment download count
	entry.DownloadCount++
	detail.Versions[ver.String()] = entry
	_ = r.saveLocked()

	return f, &entry, nil
}

// ListPlugins returns summaries of all plugins registered in this organization registry.
func (r *OrganizationRegistry) ListPlugins(ctx context.Context) ([]PrivatePluginDetail, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []PrivatePluginDetail
	for _, p := range r.plugins {
		list = append(list, *p)
	}
	return list, nil
}
