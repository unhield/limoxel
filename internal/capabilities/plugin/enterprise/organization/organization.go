package organization

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/plugin/enterprise/identity"
)

var (
	ErrOrgExists             = errors.New("enterprise org: organization already exists")
	ErrOrgNotFound           = errors.New("enterprise org: organization not found")
	ErrOrgSuspended          = errors.New("enterprise org: organization is suspended")
	ErrMemberNotFound        = errors.New("enterprise org: member not found")
	ErrPluginOwnershipExists = errors.New("enterprise org: plugin ownership already registered")
	ErrOwnershipNotFound     = errors.New("enterprise org: plugin ownership record not found")
	ErrUnauthorizedTransfer  = errors.New("enterprise org: principal not authorized to transfer ownership")
	ErrInvalidOrgID          = errors.New("enterprise org: invalid organization id format")

	orgIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9\-_]{2,62}[a-zA-Z0-9]$`)
)

// ValidateOrganizationID verifies that an organization ID is alphanumeric, safe against path traversal, and within length bounds.
func ValidateOrganizationID(id string) error {
	trimmed := strings.TrimSpace(id)
	if !orgIDPattern.MatchString(trimmed) {
		return fmt.Errorf("%w: '%s' must be 4-64 alphanumeric characters, dashes, or underscores", ErrInvalidOrgID, id)
	}
	return nil
}

// OrgStatus represents the lifecycle state of an enterprise organization.
type OrgStatus string

const (
	OrgStatusActive    OrgStatus = "active"
	OrgStatusSuspended OrgStatus = "suspended"
	OrgStatusArchived  OrgStatus = "archived"
)

// Organization represents a tenant organization.
type Organization struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Status      OrgStatus         `json:"status"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Settings    map[string]string `json:"settings,omitempty"`
}

// Member represents an individual's membership and roles in an organization.
type Member struct {
	PrincipalID    string          `json:"principal_id"`
	OrganizationID string          `json:"organization_id"`
	Roles          []identity.Role `json:"roles"`
	JoinedAt       time.Time       `json:"joined_at"`
}

// OwnershipTransferRecord documents an auditable plugin ownership transition.
type OwnershipTransferRecord struct {
	PluginID      string    `json:"plugin_id"`
	FromOwnerID   string    `json:"from_owner_id"`
	ToOwnerID     string    `json:"to_owner_id"`
	TransferredBy string    `json:"transferred_by"`
	Timestamp     time.Time `json:"timestamp"`
	Reason        string    `json:"reason"`
}

// PluginOwnership defines the organization and principal responsible for a plugin.
type PluginOwnership struct {
	PluginID        string                    `json:"plugin_id"`
	OrganizationID  string                    `json:"organization_id"`
	PrimaryOwnerID  string                    `json:"primary_owner_id"`
	Maintainers     []string                  `json:"maintainers"`
	CreatedAt       time.Time                 `json:"created_at"`
	UpdatedAt       time.Time                 `json:"updated_at"`
	TransferHistory []OwnershipTransferRecord `json:"transfer_history,omitempty"`
}

// Manager manages organizations, memberships, and plugin ownership.
type Manager struct {
	mu         sync.RWMutex
	baseDir    string
	orgs       map[string]*Organization
	members    map[string]map[string]*Member // orgID -> principalID -> Member
	ownerships map[string]*PluginOwnership   // pluginID -> Ownership
}

// NewManager constructs an organization manager backed by persistent storage.
func NewManager(baseDir string) (*Manager, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create organization storage dir: %w", err)
	}

	mgr := &Manager{
		baseDir:    baseDir,
		orgs:       make(map[string]*Organization),
		members:    make(map[string]map[string]*Member),
		ownerships: make(map[string]*PluginOwnership),
	}

	if err := mgr.loadState(); err != nil {
		return nil, err
	}

	return mgr, nil
}

func (m *Manager) stateFile() string {
	return filepath.Join(m.baseDir, "organizations.json")
}

type persistedState struct {
	Orgs       map[string]*Organization      `json:"orgs"`
	Members    map[string]map[string]*Member `json:"members"`
	Ownerships map[string]*PluginOwnership   `json:"ownerships"`
}

func (m *Manager) loadState() error {
	filePath := m.stateFile()
	data, err := os.ReadFile(filePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read organization state: %w", err)
	}

	var state persistedState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to parse organization state: %w", err)
	}

	if state.Orgs != nil {
		m.orgs = state.Orgs
	}
	if state.Members != nil {
		m.members = state.Members
	}
	if state.Ownerships != nil {
		for k, v := range state.Ownerships {
			if v != nil {
				if strings.Contains(k, "::") {
					m.ownerships[k] = v
				} else if v.OrganizationID != "" && v.PluginID != "" {
					m.ownerships[ownershipKey(v.OrganizationID, v.PluginID)] = v
				} else {
					m.ownerships[k] = v
				}
			}
		}
	}
	return nil
}

func (m *Manager) saveStateLocked() error {
	state := persistedState{
		Orgs:       m.orgs,
		Members:    m.members,
		Ownerships: m.ownerships,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize organization state: %w", err)
	}

	tempPath := filepath.Join(m.baseDir, fmt.Sprintf(".state-%d.tmp", time.Now().UnixNano()))
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary state: %w", err)
	}

	if err := os.Rename(tempPath, m.stateFile()); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to commit organization state: %w", err)
	}

	return nil
}

// CreateOrganization registers a new enterprise organization and initial owner.
func (m *Manager) CreateOrganization(ctx context.Context, id, name, desc string, initialOwnerID string) (*Organization, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := ValidateOrganizationID(id); err != nil {
		return nil, err
	}

	if _, exists := m.orgs[id]; exists {
		return nil, ErrOrgExists
	}

	now := time.Now().UTC()
	org := &Organization{
		ID:          id,
		Name:        name,
		Description: desc,
		Status:      OrgStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
		Settings:    make(map[string]string),
	}

	m.orgs[id] = org

	if m.members[id] == nil {
		m.members[id] = make(map[string]*Member)
	}

	if initialOwnerID != "" {
		m.members[id][initialOwnerID] = &Member{
			PrincipalID:    initialOwnerID,
			OrganizationID: id,
			Roles:          []identity.Role{identity.RoleOwner},
			JoinedAt:       now,
		}
	}

	if err := m.saveStateLocked(); err != nil {
		delete(m.orgs, id)
		delete(m.members, id)
		return nil, err
	}

	return org, nil
}

// GetOrganization retrieves an organization by ID.
func (m *Manager) GetOrganization(ctx context.Context, id string) (*Organization, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	org, exists := m.orgs[id]
	if !exists {
		return nil, ErrOrgNotFound
	}
	return org, nil
}

// AddMember adds or updates a member's roles within an organization.
func (m *Manager) AddMember(ctx context.Context, orgID, principalID string, roles []identity.Role) (*Member, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	org, exists := m.orgs[orgID]
	if !exists {
		return nil, ErrOrgNotFound
	}
	if org.Status != OrgStatusActive {
		return nil, ErrOrgSuspended
	}

	if m.members[orgID] == nil {
		m.members[orgID] = make(map[string]*Member)
	}

	mem := &Member{
		PrincipalID:    principalID,
		OrganizationID: orgID,
		Roles:          roles,
		JoinedAt:       time.Now().UTC(),
	}

	m.members[orgID][principalID] = mem

	if err := m.saveStateLocked(); err != nil {
		delete(m.members[orgID], principalID)
		return nil, err
	}

	return mem, nil
}

func ownershipKey(orgID, pluginID string) string {
	return orgID + "::" + pluginID
}

// RegisterPluginOwnership assigns initial ownership of a plugin to an organization and principal.
func (m *Manager) RegisterPluginOwnership(ctx context.Context, pluginID, orgID, ownerID string, maintainers []string) (*PluginOwnership, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	org, exists := m.orgs[orgID]
	if !exists {
		return nil, ErrOrgNotFound
	}
	if org.Status != OrgStatusActive {
		return nil, ErrOrgSuspended
	}

	orgMembers := m.members[orgID]
	if _, ok := orgMembers[ownerID]; !ok {
		return nil, ErrMemberNotFound
	}

	key := ownershipKey(orgID, pluginID)
	if _, exists := m.ownerships[key]; exists {
		return nil, ErrPluginOwnershipExists
	}

	now := time.Now().UTC()
	ownership := &PluginOwnership{
		PluginID:       pluginID,
		OrganizationID: orgID,
		PrimaryOwnerID: ownerID,
		Maintainers:    maintainers,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	m.ownerships[key] = ownership

	if err := m.saveStateLocked(); err != nil {
		delete(m.ownerships, key)
		return nil, err
	}

	return ownership, nil
}

// TransferPluginOwnership reassigns the primary ownership of a plugin to a new principal within an organization.
func (m *Manager) TransferPluginOwnership(ctx context.Context, orgID, pluginID, actorID, newOwnerID, reason string) (*PluginOwnership, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var ownership *PluginOwnership
	var exists bool
	if orgID != "" {
		ownership, exists = m.ownerships[ownershipKey(orgID, pluginID)]
	} else {
		for _, v := range m.ownerships {
			if v.PluginID == pluginID {
				ownership = v
				exists = true
				break
			}
		}
	}

	if !exists {
		return nil, ErrOwnershipNotFound
	}

	// Verify actor is current primary owner or organization owner/admin
	isOwner := ownership.PrimaryOwnerID == actorID
	orgMembers := m.members[ownership.OrganizationID]
	var isOrgAdmin bool
	if member, ok := orgMembers[actorID]; ok {
		for _, r := range member.Roles {
			if r == identity.RoleOwner || r == identity.RoleAdmin {
				isOrgAdmin = true
				break
			}
		}
	}

	if !isOwner && !isOrgAdmin {
		return nil, ErrUnauthorizedTransfer
	}

	if _, ok := orgMembers[newOwnerID]; !ok {
		return nil, ErrMemberNotFound
	}

	now := time.Now().UTC()
	record := OwnershipTransferRecord{
		PluginID:      pluginID,
		FromOwnerID:   ownership.PrimaryOwnerID,
		ToOwnerID:     newOwnerID,
		TransferredBy: actorID,
		Timestamp:     now,
		Reason:        reason,
	}

	ownership.PrimaryOwnerID = newOwnerID
	ownership.UpdatedAt = now
	ownership.TransferHistory = append(ownership.TransferHistory, record)

	if err := m.saveStateLocked(); err != nil {
		return nil, err
	}

	return ownership, nil
}

// GetPluginOwnership returns the ownership record of a plugin scoped to an organization.
func (m *Manager) GetPluginOwnership(ctx context.Context, orgID, pluginID string) (*PluginOwnership, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var ownership *PluginOwnership
	var exists bool
	if orgID != "" {
		ownership, exists = m.ownerships[ownershipKey(orgID, pluginID)]
	} else {
		for _, v := range m.ownerships {
			if v.PluginID == pluginID {
				ownership = v
				exists = true
				break
			}
		}
	}

	if !exists {
		return nil, ErrOwnershipNotFound
	}

	clone := *ownership
	clone.Maintainers = append([]string(nil), ownership.Maintainers...)
	clone.TransferHistory = append([]OwnershipTransferRecord(nil), ownership.TransferHistory...)
	return &clone, nil
}
