package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Storage persists and retrieves versioned organization policy documents.
type Storage struct {
	mu      sync.RWMutex
	baseDir string
	cache   map[string]*PolicyDocument // orgID -> document
}

// NewStorage constructs an organization policy storage repository.
func NewStorage(baseDir string) (*Storage, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create policy storage dir: %w", err)
	}

	s := &Storage{
		baseDir: baseDir,
		cache:   make(map[string]*PolicyDocument),
	}

	if err := s.loadAll(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Storage) policyPath(orgID string) string {
	return filepath.Join(s.baseDir, fmt.Sprintf("%s.json", orgID))
}

func (s *Storage) loadAll() error {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return fmt.Errorf("failed to read policy directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasPrefix(entry.Name(), ".") && filepath.Ext(entry.Name()) == ".json" {
			data, err := os.ReadFile(filepath.Join(s.baseDir, entry.Name()))
			if err != nil {
				return fmt.Errorf("failed to read policy file %s: %w", entry.Name(), err)
			}
			var doc PolicyDocument
			if err := json.Unmarshal(data, &doc); err != nil {
				return fmt.Errorf("failed to parse policy file %s: %w", entry.Name(), err)
			}
			if err := ValidatePolicyDocument(&doc); err != nil {
				return fmt.Errorf("invalid policy file %s: %w", entry.Name(), err)
			}
			s.cache[doc.OrganizationID] = &doc
		}
	}
	return nil
}

// GetPolicy retrieves the active policy document for an organization.
func (s *Storage) GetPolicy(ctx context.Context, orgID string) (*PolicyDocument, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	doc, exists := s.cache[orgID]
	if !exists {
		return nil, ErrPolicyNotFound
	}
	// Return clone
	clone := *doc
	return &clone, nil
}

// SavePolicy writes an updated policy revision to persistent storage with atomic replacement.
func (s *Storage) SavePolicy(ctx context.Context, doc *PolicyDocument) error {
	if err := ValidatePolicyDocument(doc); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existing := s.cache[doc.OrganizationID]
	if existing != nil {
		doc.Revision = existing.Revision + 1
	} else {
		doc.Revision = 1
	}
	doc.UpdatedAt = time.Now().UTC()

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal policy document: %w", err)
	}

	tempPath := filepath.Join(s.baseDir, fmt.Sprintf(".%s-%d.tmp", doc.OrganizationID, time.Now().UnixNano()))
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary policy file: %w", err)
	}

	destPath := s.policyPath(doc.OrganizationID)
	if err := os.Rename(tempPath, destPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to commit policy file: %w", err)
	}

	// Update cache
	s.cache[doc.OrganizationID] = doc
	return nil
}
