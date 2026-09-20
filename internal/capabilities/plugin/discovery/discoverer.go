package discovery

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	pkgerr "github.com/unhield/limoxel/internal/capabilities/plugin/errors"
	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
)

// ManifestFilenames lists the standard supported plugin manifest filenames.
var ManifestFilenames = []string{
	"plugin.json",
	"limoxel-plugin.json",
}

// DiscoveredCandidate represents a successfully discovered plugin ready for validation and installation.
type DiscoveredCandidate struct {
	Manifest     *model.Manifest
	Directory    string
	ManifestPath string
}

// DiagnosticRecord captures a non-fatal anomaly encountered during discovery.
type DiagnosticRecord struct {
	Path    string
	Message string
	Error   error
}

// Discoverer scans local directories for candidate plugin distributions.
type Discoverer struct {
	diagnostics []DiagnosticRecord
}

// NewDiscoverer constructs an initialized Discoverer.
func NewDiscoverer() *Discoverer {
	return &Discoverer{}
}

// Diagnostics returns any non-fatal diagnostic records encountered during the last discovery pass.
func (d *Discoverer) Diagnostics() []DiagnosticRecord {
	if d == nil || d.diagnostics == nil {
		return nil
	}
	cp := make([]DiagnosticRecord, len(d.diagnostics))
	copy(cp, d.diagnostics)
	return cp
}

// Discover scans the provided directories for plugin distributions.
func (d *Discoverer) Discover(ctx context.Context, searchDirs ...string) ([]*DiscoveredCandidate, error) {
	d.diagnostics = nil
	candidates := make(map[model.Identity]*DiscoveredCandidate)

	for _, rawDir := range searchDirs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		cleanDir := filepath.Clean(strings.TrimSpace(rawDir))
		if cleanDir == "" {
			continue
		}

		info, err := os.Stat(cleanDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue // Gracefully skip non-existent search directories
			}
			return nil, pkgerr.Wrap(pkgerr.CodeLoadingFailed, "", fmt.Sprintf("failed to inspect search directory %s", cleanDir), err)
		}

		if !info.IsDir() {
			continue
		}

		// First, check if cleanDir itself contains a manifest
		candidate, found, err := d.probeDirectory(cleanDir)
		if err != nil {
			d.diagnostics = append(d.diagnostics, DiagnosticRecord{
				Path:    cleanDir,
				Message: "failed to probe directory for manifest",
				Error:   err,
			})
		} else if found {
			if existing, collision := candidates[candidate.Manifest.ID]; collision {
				d.diagnostics = append(d.diagnostics, DiagnosticRecord{
					Path:    candidate.Directory,
					Message: fmt.Sprintf("duplicate identity %s discovered; keeping first at %s", candidate.Manifest.ID, existing.Directory),
					Error:   pkgerr.New(pkgerr.CodeDuplicateIdentity, candidate.Manifest.ID, "duplicate identity discovered"),
				})
			} else {
				candidates[candidate.Manifest.ID] = candidate
			}
		}

		// Second, inspect direct subdirectories
		entries, err := os.ReadDir(cleanDir)
		if err != nil {
			d.diagnostics = append(d.diagnostics, DiagnosticRecord{
				Path:    cleanDir,
				Message: "failed to read search directory entries",
				Error:   err,
			})
			continue
		}

		for _, entry := range entries {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}

			isDir := entry.IsDir()
			if !isDir && (entry.Type()&os.ModeSymlink != 0) {
				if fi, statErr := os.Stat(filepath.Join(cleanDir, entry.Name())); statErr == nil && fi.IsDir() {
					isDir = true
				}
			}
			if !isDir {
				continue
			}

			subDir := filepath.Join(cleanDir, entry.Name())
			cand, found, err := d.probeDirectory(subDir)
			if err != nil {
				d.diagnostics = append(d.diagnostics, DiagnosticRecord{
					Path:    subDir,
					Message: "failed to probe subdirectory for manifest",
					Error:   err,
				})
				continue
			}
			if !found {
				continue
			}

			if existing, collision := candidates[cand.Manifest.ID]; collision {
				d.diagnostics = append(d.diagnostics, DiagnosticRecord{
					Path:    cand.Directory,
					Message: fmt.Sprintf("duplicate identity %s discovered at %s; keeping first at %s", cand.Manifest.ID, cand.Directory, existing.Directory),
					Error:   pkgerr.New(pkgerr.CodeDuplicateIdentity, cand.Manifest.ID, "duplicate identity discovered"),
				})
				continue
			}

			candidates[cand.Manifest.ID] = cand
		}
	}

	// Produce deterministically sorted slice by plugin ID then Directory
	result := make([]*DiscoveredCandidate, 0, len(candidates))
	for _, c := range candidates {
		result = append(result, c)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Manifest.ID != result[j].Manifest.ID {
			return result[i].Manifest.ID.Less(result[j].Manifest.ID)
		}
		return result[i].Directory < result[j].Directory
	})

	return result, nil
}

func (d *Discoverer) probeDirectory(dir string) (*DiscoveredCandidate, bool, error) {
	for _, name := range ManifestFilenames {
		manifestPath := filepath.Join(dir, name)
		info, err := os.Stat(manifestPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, false, err
		}

		if info.IsDir() {
			continue
		}

		m, err := model.ParseManifestFile(manifestPath)
		if err != nil {
			return nil, false, err
		}

		return &DiscoveredCandidate{
			Manifest:     m,
			Directory:    dir,
			ManifestPath: manifestPath,
		}, true, nil
	}

	return nil, false, nil
}
