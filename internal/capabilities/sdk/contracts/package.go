package contracts

import (
	"context"

	"github.com/unhield/limoxel/internal/capabilities/sdk/lifecycle"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

// PackageInfo represents public information about a repository package.
type PackageInfo struct {
	Path        string `json:"path"`
	Name        string `json:"name"`
	Module      string `json:"module,omitempty"`
	FileCount   int    `json:"file_count"`
	SymbolCount int    `json:"symbol_count"`
	DocComment  string `json:"doc_comment,omitempty"`
}

// PackageStatistics represents quantitative measurements of a package.
type PackageStatistics struct {
	Path                string `json:"path"`
	FileCount           int    `json:"file_count"`
	SymbolCount         int    `json:"symbol_count"`
	ExportedSymbolCount int    `json:"exported_symbol_count"`
	InternalSymbolCount int    `json:"internal_symbol_count"`
	ImportCount         int    `json:"import_count"`
	DependentCount      int    `json:"dependent_count"`
	TestFileCount       int    `json:"test_file_count"`
}

// PackageHierarchyNode represents a node in the package namespace hierarchy tree.
type PackageHierarchyNode struct {
	Package  PackageInfo            `json:"package"`
	Children []PackageHierarchyNode `json:"children,omitempty"`
}

// PackageRelationship represents a dependency or call relationship between two packages.
type PackageRelationship struct {
	SourcePackage string `json:"source_package"`
	TargetPackage string `json:"target_package"`
	Kind          string `json:"kind"`
	Weight        int    `json:"weight,omitempty"`
}

// PackageFilter provides criteria for filtering package discovery results.
type PackageFilter struct {
	Module  string `json:"module,omitempty"`
	Pattern string `json:"pattern,omitempty"`
}

// PackageContract defines the public contract for package discovery, lookup, statistics, hierarchy, and relationships.
type PackageContract interface {
	Contract
	DiscoverPackages(ctx context.Context, filter PackageFilter, opts PaginationOptions) ([]PackageInfo, error)
	LookupPackage(ctx context.Context, pkgPathOrName string) (*PackageInfo, error)
	GetPackageStatistics(ctx context.Context, pkgPathOrName string) (*PackageStatistics, error)
	GetPackageHierarchy(ctx context.Context, pkgPathOrName string) (*PackageHierarchyNode, error)
	GetPackageRelationships(ctx context.Context, pkgPathOrName string) ([]PackageRelationship, error)
}

// DefaultPackageContractMetadata returns default contract descriptor for Package operations.
func DefaultPackageContractMetadata() BaseContract {
	return NewBaseContract(
		"PackageContract",
		lifecycle.CapabilityRepository,
		version.SemVer{Major: 1, Minor: 0, Patch: 0},
		lifecycle.StateSupported,
		"Provides public package discovery, lookup, statistics, hierarchy navigation, and relationship queries.",
	)
}
