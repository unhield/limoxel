package community

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

var (
	// ErrInvalidMetricTarget indicates missing or invalid plugin or version.
	ErrInvalidMetricTarget = errors.New("community: invalid metric target plugin or version")
)

// CatalogDownloadNotifier allows notifying the catalog index when a verified download completes.
type CatalogDownloadNotifier interface {
	IncrementDownloads(pluginID string, ver version.SemVer) error
}

// DownloadTracker tracks real completed downloads per plugin and version.
type DownloadTracker struct {
	mu        sync.RWMutex
	notifier  CatalogDownloadNotifier
	downloads map[string]int64       // pluginID -> total count
	events    map[string][]time.Time // pluginID -> list of download timestamps
}

// NewDownloadTracker creates an initialized DownloadTracker.
func NewDownloadTracker(notifier CatalogDownloadNotifier) *DownloadTracker {
	return &DownloadTracker{
		notifier:  notifier,
		downloads: make(map[string]int64),
		events:    make(map[string][]time.Time),
	}
}

// RecordDownload increments the download count for a plugin release upon confirmed completion.
func (t *DownloadTracker) RecordDownload(pluginID string, ver version.SemVer) error {
	if pluginID == "" {
		return ErrInvalidMetricTarget
	}

	now := time.Now().UTC()

	t.mu.Lock()
	t.downloads[pluginID]++
	t.events[pluginID] = append(t.events[pluginID], now)
	t.mu.Unlock()

	if t.notifier != nil {
		if err := t.notifier.IncrementDownloads(pluginID, ver); err != nil {
			return fmt.Errorf("failed to notify catalog of download: %w", err)
		}
	}

	return nil
}

// GetCount returns the recorded download count for a plugin in this tracker.
func (t *DownloadTracker) GetCount(pluginID string) int64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.downloads[pluginID]
}

// GetVelocity returns the count of verified completed downloads within the specified duration window.
func (t *DownloadTracker) GetVelocity(pluginID string, window time.Duration) int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	timestamps, ok := t.events[pluginID]
	if !ok || len(timestamps) == 0 {
		return 0
	}

	if window <= 0 {
		return len(timestamps)
	}

	cutoff := time.Now().UTC().Add(-window)
	count := 0
	for i := len(timestamps) - 1; i >= 0; i-- {
		if timestamps[i].After(cutoff) {
			count++
		} else {
			break
		}
	}
	return count
}
