package security

// Domain represents a security permission domain.
type Domain string

const (
	// DomainRepository encompasses repository metadata, file contents, and analysis.
	DomainRepository Domain = "repository"
	// DomainFile encompasses direct filesystem operations on files.
	DomainFile Domain = "file"
	// DomainNetwork encompasses outbound and inbound network access.
	DomainNetwork Domain = "network"
	// DomainConfig encompasses plugin and host configuration access.
	DomainConfig Domain = "config"
	// DomainRuntime encompasses process and execution controls.
	DomainRuntime Domain = "runtime"
)

// Permission identifiers recognized by Limoxel.
const (
	// Repository permissions
	PermRepoMetadata = "repository.metadata"
	PermRepoRead     = "repository.read"
	PermRepoWrite    = "repository.write"
	PermRepoAnalysis = "repository.analysis"

	// File permissions
	PermFileRead    = "file.read"
	PermFileWrite   = "file.write"
	PermFileDelete  = "file.delete"
	PermFileExecute = "file.execute"

	// Network permissions
	PermNetworkOutbound  = "network.outbound"
	PermNetworkInbound   = "network.inbound"
	PermNetworkLocalhost = "network.localhost"

	// Configuration permissions
	PermConfigRead  = "config.read"
	PermConfigWrite = "config.write"

	// Runtime permissions
	PermRuntimeSpawn   = "runtime.process.spawn"
	PermRuntimeRestart = "runtime.plugin.restart"
)

// PermissionProfile represents a named preset of permissions.
type PermissionProfile string

const (
	// ProfileRestricted grants minimal read-only metadata access (default).
	ProfileRestricted PermissionProfile = "restricted"
	// ProfileStandard grants typical read access to the workspace repository.
	ProfileStandard PermissionProfile = "standard"
	// ProfileFull grants all permissions requested by the plugin.
	ProfileFull PermissionProfile = "full"
)
