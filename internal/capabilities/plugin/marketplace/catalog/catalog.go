package catalog

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/plugin/marketplace/storage"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
	pubmarket "github.com/unhield/limoxel/plugin/marketplace"
)

var (
	// ErrPluginNotFound indicates plugin does not exist in the catalog.
	ErrPluginNotFound = errors.New("catalog: plugin not found")
	// ErrVersionExists indicates the version has already been registered.
	ErrVersionExists = errors.New("catalog: version already exists and is immutable")
	// ErrVersionNotFound indicates version is not found for plugin.
	ErrVersionNotFound = errors.New("catalog: version not found")
)

// Catalog maintains the in-memory index of plugins, releases, and categories backed by Storage.
type Catalog struct {
	mu       sync.RWMutex
	store    storage.Storage
	plugins  map[string]*pubmarket.PluginDetail
	catIndex map[pubmarket.Category]map[string]struct{}
}

// NewCatalog constructs an initialized Catalog and populates it from storage.
func NewCatalog(store storage.Storage) (*Catalog, error) {
	c := &Catalog{
		store:    store,
		plugins:  make(map[string]*pubmarket.PluginDetail),
		catIndex: make(map[pubmarket.Category]map[string]struct{}),
	}

	if err := c.reloadFromStorage(); err != nil {
		return nil, fmt.Errorf("failed to load catalog from storage: %w", err)
	}

	return c, nil
}

func (c *Catalog) reloadFromStorage() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	items, err := c.store.ListPluginDetails()
	if err != nil {
		return err
	}

	for _, item := range items {
		copyItem := item
		c.plugins[copyItem.ID] = &copyItem
		c.indexCategories(&copyItem)
	}

	return nil
}

func (c *Catalog) indexCategories(p *pubmarket.PluginDetail) {
	for _, cat := range p.Categories {
		if c.catIndex[cat] == nil {
			c.catIndex[cat] = make(map[string]struct{})
		}
		c.catIndex[cat][p.ID] = struct{}{}
	}
}

// RegisterPlugin inserts a newly submitted or imported plugin into the catalog and storage.
func (c *Catalog) RegisterPlugin(detail pubmarket.PluginDetail) error {
	if detail.ID == "" {
		return errors.New("catalog: empty plugin ID")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	detail.UpdatedAt = time.Now().UTC()
	if err := c.store.SavePluginDetail(detail); err != nil {
		return fmt.Errorf("failed to persist plugin: %w", err)
	}

	c.plugins[detail.ID] = &detail
	c.indexCategories(&detail)
	return nil
}

// UpdatePlugin updates metadata for an existing plugin.
func (c *Catalog) UpdatePlugin(detail pubmarket.PluginDetail) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.plugins[detail.ID]; !ok {
		return ErrPluginNotFound
	}

	detail.UpdatedAt = time.Now().UTC()
	if err := c.store.SavePluginDetail(detail); err != nil {
		return err
	}

	c.plugins[detail.ID] = &detail
	c.indexCategories(&detail)
	return nil
}

// GetPlugin retrieves detail for a plugin by ID.
func (c *Catalog) GetPlugin(id string) (*pubmarket.PluginDetail, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	p, ok := c.plugins[id]
	if !ok {
		return nil, ErrPluginNotFound
	}

	copied := *p
	return &copied, nil
}

// GetVersions returns all version entries for a plugin, sorted descending by semver.
func (c *Catalog) GetVersions(id string) ([]pubmarket.VersionInfo, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	p, ok := c.plugins[id]
	if !ok {
		return nil, ErrPluginNotFound
	}

	results := make([]pubmarket.VersionInfo, len(p.Versions))
	copy(results, p.Versions)

	sort.Slice(results, func(i, j int) bool {
		// Descending: newer version first
		return results[i].Version.Compare(results[j].Version) > 0
	})

	return results, nil
}

// AddVersion adds a new published release to the plugin. Published versions are immutable.
func (c *Catalog) AddVersion(id string, ver pubmarket.VersionInfo) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	p, ok := c.plugins[id]
	if !ok {
		return ErrPluginNotFound
	}

	for _, existing := range p.Versions {
		if existing.Version.String() == ver.Version.String() {
			return ErrVersionExists
		}
	}

	if ver.PublishedAt.IsZero() {
		ver.PublishedAt = time.Now().UTC()
	}

	clone := *p
	clone.Versions = make([]pubmarket.VersionInfo, len(p.Versions), len(p.Versions)+1)
	copy(clone.Versions, p.Versions)
	clone.Versions = append(clone.Versions, ver)
	if clone.LatestVersion.Compare(ver.Version) < 0 {
		clone.LatestVersion = ver.Version
	}
	clone.UpdatedAt = time.Now().UTC()

	if err := c.store.SavePluginDetail(clone); err != nil {
		return err
	}

	c.plugins[id] = &clone
	c.indexCategories(&clone)
	return nil
}

// UpdateStatus changes the publication status of a plugin or version.
func (c *Catalog) UpdateStatus(id string, ver *version.SemVer, status pubmarket.PublicationStatus) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	p, ok := c.plugins[id]
	if !ok {
		return ErrPluginNotFound
	}

	clone := *p
	if ver != nil {
		clone.Versions = make([]pubmarket.VersionInfo, len(p.Versions))
		copy(clone.Versions, p.Versions)
		found := false
		for i := range clone.Versions {
			if clone.Versions[i].Version.String() == ver.String() {
				clone.Versions[i].Status = status
				found = true
				break
			}
		}
		if !found {
			return ErrVersionNotFound
		}
	} else {
		clone.Status = status
	}

	clone.UpdatedAt = time.Now().UTC()
	if err := c.store.SavePluginDetail(clone); err != nil {
		return err
	}

	c.plugins[id] = &clone
	c.indexCategories(&clone)
	return nil
}

// ListPlugins returns summaries for plugins, optionally filtered by discoverable status.
func (c *Catalog) ListPlugins(discoverableOnly bool) []pubmarket.PluginSummary {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var summaries []pubmarket.PluginSummary
	for _, p := range c.plugins {
		if discoverableOnly && !p.Status.IsDiscoverable() {
			continue
		}
		summaries = append(summaries, p.PluginSummary)
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].ID < summaries[j].ID
	})

	return summaries
}

// IncrementDownloads records a successful download for a plugin and specific version.
func (c *Catalog) IncrementDownloads(id string, ver version.SemVer) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	p, ok := c.plugins[id]
	if !ok {
		return ErrPluginNotFound
	}

	clone := *p
	clone.Versions = make([]pubmarket.VersionInfo, len(p.Versions))
	copy(clone.Versions, p.Versions)
	clone.TotalDownloads++
	for i := range clone.Versions {
		if clone.Versions[i].Version.String() == ver.String() {
			clone.Versions[i].Downloads++
			break
		}
	}

	if err := c.store.SavePluginDetail(clone); err != nil {
		return err
	}

	c.plugins[id] = &clone
	return nil
}

// UpdateRating updates aggregate rating metrics on the plugin.
func (c *Catalog) UpdateRating(id string, rating pubmarket.RatingSummary) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	p, ok := c.plugins[id]
	if !ok {
		return ErrPluginNotFound
	}

	clone := *p
	clone.Rating = rating
	clone.AverageRating = rating.Average
	clone.RatingCount = rating.Count

	if err := c.store.SavePluginDetail(clone); err != nil {
		return err
	}

	c.plugins[id] = &clone
	return nil
}

// GetCategories returns all available categories that contain at least one published plugin.
func (c *Catalog) GetCategories() []pubmarket.Category {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cats := pubmarket.AllCategories()
	return cats
}

// GetCategoryCount returns the number of plugins in a category.
func (c *Catalog) GetCategoryCount(cat pubmarket.Category) int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ids, ok := c.catIndex[cat]
	if !ok {
		return 0
	}
	return len(ids)
}
