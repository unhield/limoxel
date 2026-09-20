package dependency

import (
	"errors"
	"fmt"
	"sort"
	"time"

	coredeg "github.com/unhield/limoxel/internal/capabilities/plugin/dependency"
	pkgerr "github.com/unhield/limoxel/internal/capabilities/plugin/errors"
	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
	pubmarket "github.com/unhield/limoxel/plugin/marketplace"
)

// CatalogReader abstracts catalog lookup required for dependency resolution.
type CatalogReader interface {
	GetPlugin(id string) (*pubmarket.PluginDetail, error)
}

// Resolver resolves inter-plugin dependencies for marketplace distribution.
type Resolver struct {
	catalog CatalogReader
}

// NewResolver constructs a marketplace dependency resolver.
func NewResolver(catalog CatalogReader) *Resolver {
	return &Resolver{
		catalog: catalog,
	}
}

// resolvableCandidate wraps a catalog plugin release to satisfy the core resolver interface.
type resolvableCandidate struct {
	id           model.Identity
	ver          version.SemVer
	dependencies []model.Dependency
	artifact     pubmarket.ResolvedDependency
}

func (r *resolvableCandidate) GetID() model.Identity {
	return r.id
}

func (r *resolvableCandidate) GetVersion() version.SemVer {
	return r.ver
}

func (r *resolvableCandidate) GetDependencies() []model.Dependency {
	return r.dependencies
}

// Resolve builds an installation plan resolving all required dependencies for a target plugin and version.
func (r *Resolver) Resolve(targetID string, targetVer version.SemVer) (*pubmarket.DependencyResolutionPlan, error) {
	if targetID == "" {
		return nil, fmt.Errorf("%w: target plugin ID cannot be empty", pubmarket.ErrInvalidInput)
	}

	targetDetail, err := r.catalog.GetPlugin(targetID)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", pubmarket.ErrNotFound, targetID)
	}

	// Verify target version exists
	var targetVInfo *pubmarket.VersionInfo
	for _, v := range targetDetail.Versions {
		if v.Version.Compare(targetVer) == 0 {
			targetVInfo = &v
			break
		}
	}
	if targetVInfo == nil {
		return nil, fmt.Errorf("%w: %s version %s", pubmarket.ErrVersionNotFound, targetID, targetVer.String())
	}

	// Traversal state
	candidates := make(map[string]*resolvableCandidate)
	queue := []string{targetID}
	assignedVersions := map[string]version.SemVer{
		targetID: targetVer,
	}

	for len(queue) > 0 {
		currID := queue[0]
		queue = queue[1:]

		currDetail, err := r.catalog.GetPlugin(currID)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to fetch detail for %s", pubmarket.ErrNotFound, currID)
		}

		selectedVer := assignedVersions[currID]
		var matchedVInfo *pubmarket.VersionInfo
		for _, v := range currDetail.Versions {
			if v.Version.Compare(selectedVer) == 0 {
				matchedVInfo = &v
				break
			}
		}
		if matchedVInfo == nil {
			return nil, fmt.Errorf("%w: %s version %s not found in catalog", pubmarket.ErrVersionNotFound, currID, selectedVer.String())
		}

		var deps []model.Dependency
		for _, depSummary := range currDetail.Dependencies {
			depID := depSummary.ID
			depConstraint := depSummary.Constraint
			if depConstraint == "" {
				depConstraint = "*"
			}

			// Check if already assigned a version in this plan
			if existingVer, ok := assignedVersions[depID]; ok {
				// Verify compatibility with current constraint
				if !model.EvaluateConstraint(depConstraint, existingVer) {
					return nil, fmt.Errorf("%w: plugin %s requires %s constraint %q, but %s was selected",
						pubmarket.ErrDependencyConflict, currID, depID, depConstraint, existingVer.String())
				}
			} else {
				// Lookup dependency in catalog
				depDetail, err := r.catalog.GetPlugin(depID)
				if err != nil {
					if depSummary.Optional {
						continue
					}
					return nil, fmt.Errorf("%w: %s required by %s", pubmarket.ErrDependencyMissing, depID, currID)
				}

				// Find highest compatible version
				var compatibleVersions []pubmarket.VersionInfo
				for _, v := range depDetail.Versions {
					if v.Status.IsDiscoverable() && model.EvaluateConstraint(depConstraint, v.Version) {
						compatibleVersions = append(compatibleVersions, v)
					}
				}

				if len(compatibleVersions) == 0 {
					if depSummary.Optional {
						continue
					}
					return nil, fmt.Errorf("%w: no compatible version of %s satisfies %q for %s",
						pubmarket.ErrDependencyConflict, depID, depConstraint, currID)
				}

				// Sort descending to choose latest compatible
				sort.Slice(compatibleVersions, func(i, j int) bool {
					return compatibleVersions[i].Version.Compare(compatibleVersions[j].Version) > 0
				})

				best := compatibleVersions[0]
				assignedVersions[depID] = best.Version
				queue = append(queue, depID)
			}

			coreDep, err := model.NewDependency(model.Identity(depID), depConstraint, depSummary.Optional)
			if err == nil {
				deps = append(deps, coreDep)
			}
		}

		candidates[currID] = &resolvableCandidate{
			id:           model.Identity(currID),
			ver:          selectedVer,
			dependencies: deps,
			artifact: pubmarket.ResolvedDependency{
				ID:             currID,
				Version:        selectedVer,
				ArtifactDigest: matchedVInfo.ArtifactDigest,
				DownloadURL:    matchedVInfo.DownloadURL,
				Optional:       false,
			},
		}
	}

	// Adapt to core resolver slice
	resolvables := make([]coredeg.Resolvable, 0, len(candidates))
	for _, c := range candidates {
		resolvables = append(resolvables, c)
	}

	// Use core resolver for topological sort and cycle detection
	coreResolver := coredeg.NewResolver()
	plan, err := coreResolver.Resolve(resolvables)
	if err != nil {
		var limErr *pkgerr.PluginError
		if errors.As(err, &limErr) {
			if limErr.Code() == pkgerr.CodeDependencyCycle {
				return nil, fmt.Errorf("%w: %s", pubmarket.ErrCircularDependency, limErr.Message())
			}
			if limErr.Code() == pkgerr.CodeIncompatibleVersion {
				return nil, fmt.Errorf("%w: %s", pubmarket.ErrDependencyConflict, limErr.Message())
			}
			if limErr.Code() == pkgerr.CodeDependencyMissing {
				return nil, fmt.Errorf("%w: %s", pubmarket.ErrDependencyMissing, limErr.Message())
			}
		}
		return nil, fmt.Errorf("dependency resolution failed: %w", err)
	}

	var installOrder []pubmarket.ResolvedDependency
	depMap := make(map[string]pubmarket.ResolvedDependency, len(candidates))

	for _, id := range plan.ActivationOrder {
		cand := candidates[string(id)]
		if cand != nil {
			installOrder = append(installOrder, cand.artifact)
			depMap[string(id)] = cand.artifact
		}
	}

	return &pubmarket.DependencyResolutionPlan{
		TargetPlugin:  targetID,
		TargetVersion: targetVer,
		InstallOrder:  installOrder,
		Dependencies:  depMap,
		ResolvedAt:    time.Now().UTC(),
	}, nil
}
