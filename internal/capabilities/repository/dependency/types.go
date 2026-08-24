package dependency

// Ecosystem represents the package manager or language ecosystem of a dependency.
type Ecosystem string

const (
	// EcosystemGo represents Go module dependencies.
	EcosystemGo Ecosystem = "go"

	// EcosystemNpm represents npm / Node.js dependencies.
	EcosystemNpm Ecosystem = "npm"

	// EcosystemCargo represents Rust / Cargo dependencies.
	EcosystemCargo Ecosystem = "cargo"

	// EcosystemMaven represents Java / Maven dependencies.
	EcosystemMaven Ecosystem = "maven"

	// EcosystemGradle represents Java / Gradle dependencies.
	EcosystemGradle Ecosystem = "gradle"

	// EcosystemPython represents Python dependencies.
	EcosystemPython Ecosystem = "python"

	// EcosystemComposer represents PHP / Composer dependencies.
	EcosystemComposer Ecosystem = "composer"

	// EcosystemUnknown represents an unclassified ecosystem.
	EcosystemUnknown Ecosystem = "unknown"
)

// DependencyType represents the classification or scope of a dependency relationship.
type DependencyType string

const (
	// DependencyDirect indicates a directly declared dependency in a manifest.
	DependencyDirect DependencyType = "direct"

	// DependencyIndirect indicates a transitive or indirect dependency.
	DependencyIndirect DependencyType = "indirect"

	// DependencyInternal indicates an in-repository package or module dependency.
	DependencyInternal DependencyType = "internal"

	// DependencyExternal indicates a third-party or external dependency.
	DependencyExternal DependencyType = "external"

	// DependencyDevelopment indicates a development / test-only dependency.
	DependencyDevelopment DependencyType = "dev"

	// DependencyBuild indicates a build-time dependency.
	DependencyBuild DependencyType = "build"

	// DependencyRuntime indicates a runtime dependency.
	DependencyRuntime DependencyType = "runtime"

	// DependencyUnknown indicates an unclassified dependency type.
	DependencyUnknown DependencyType = "unknown"
)

// LicenseType represents the standardized software license classification.
type LicenseType string

const (
	// LicenseMIT represents the MIT License.
	LicenseMIT LicenseType = "MIT"

	// LicenseApache2 represents the Apache License 2.0.
	LicenseApache2 LicenseType = "Apache-2.0"

	// LicenseBSD represents BSD-style licenses.
	LicenseBSD LicenseType = "BSD"

	// LicenseGPL represents GNU General Public Licenses (GPLv2, GPLv3).
	LicenseGPL LicenseType = "GPL"

	// LicenseLGPL represents GNU Lesser General Public Licenses.
	LicenseLGPL LicenseType = "LGPL"

	// LicenseMPL represents Mozilla Public Licenses.
	LicenseMPL LicenseType = "MPL"

	// LicenseUnknown represents an unrecognized or unparseable license.
	LicenseUnknown LicenseType = "Unknown"

	// LicenseUnavailable indicates license metadata was unavailable.
	LicenseUnavailable LicenseType = "Unavailable"
)

// HealthStatus represents the maintenance health state of a dependency.
type HealthStatus string

const (
	// HealthActive indicates active and ongoing maintenance.
	HealthActive HealthStatus = "active"

	// HealthDeprecated indicates explicitly deprecated by maintainers.
	HealthDeprecated HealthStatus = "deprecated"

	// HealthArchived indicates the dependency repository is archived.
	HealthArchived HealthStatus = "archived"

	// HealthAbandoned indicates prolonged inactivity and lack of maintenance.
	HealthAbandoned HealthStatus = "abandoned"

	// HealthUnknown indicates health status is unclassified or unavailable.
	HealthUnknown HealthStatus = "unknown"
)
