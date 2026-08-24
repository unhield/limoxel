package language

// ModuleType represents the category of a detected project module.
type ModuleType string

const (
	// ModuleGo represents a Go module (go.mod).
	ModuleGo ModuleType = "go"

	// ModuleNpm represents a Node.js / JavaScript / TypeScript module (package.json).
	ModuleNpm ModuleType = "npm"

	// ModuleCargo represents a Rust module (Cargo.toml).
	ModuleCargo ModuleType = "cargo"

	// ModuleMaven represents a Java Maven module (pom.xml).
	ModuleMaven ModuleType = "maven"

	// ModuleGradle represents a Java / Kotlin Gradle module (build.gradle / build.gradle.kts).
	ModuleGradle ModuleType = "gradle"

	// ModulePython represents a Python module (pyproject.toml / requirements.txt / setup.py).
	ModulePython ModuleType = "python"

	// ModuleComposer represents a PHP Composer module (composer.json).
	ModuleComposer ModuleType = "composer"

	// ModuleUnknown represents an unclassified module type.
	ModuleUnknown ModuleType = "unknown"
)

// BuildSystemType represents a detected build automation system.
type BuildSystemType string

const (
	// BuildMake represents GNU Make / Makefile.
	BuildMake BuildSystemType = "make"

	// BuildTaskfile represents Task / Taskfile.yml.
	BuildTaskfile BuildSystemType = "taskfile"

	// BuildCMake represents CMake / CMakeLists.txt.
	BuildCMake BuildSystemType = "cmake"

	// BuildMaven represents Apache Maven.
	BuildMaven BuildSystemType = "maven"

	// BuildGradle represents Gradle.
	BuildGradle BuildSystemType = "gradle"

	// BuildNpm represents npm.
	BuildNpm BuildSystemType = "npm"

	// BuildPnpm represents pnpm.
	BuildPnpm BuildSystemType = "pnpm"

	// BuildYarn represents Yarn.
	BuildYarn BuildSystemType = "yarn"

	// BuildCargo represents Cargo.
	BuildCargo BuildSystemType = "cargo"

	// BuildUnknown represents an unclassified build system.
	BuildUnknown BuildSystemType = "unknown"
)

// ConfigType represents a detected configuration file format or category.
type ConfigType string

const (
	// ConfigYAML represents YAML configuration files (.yaml, .yml).
	ConfigYAML ConfigType = "yaml"

	// ConfigJSON represents JSON configuration files (.json).
	ConfigJSON ConfigType = "json"

	// ConfigTOML represents TOML configuration files (.toml).
	ConfigTOML ConfigType = "toml"

	// ConfigENV represents environment configuration files (.env).
	ConfigENV ConfigType = "env"

	// ConfigINI represents INI configuration files (.ini).
	ConfigINI ConfigType = "ini"

	// ConfigProperties represents Java/general properties files (.properties).
	ConfigProperties ConfigType = "properties"

	// ConfigXML represents XML configuration files (.xml).
	ConfigXML ConfigType = "xml"

	// ConfigUnknown represents an unclassified configuration format.
	ConfigUnknown ConfigType = "unknown"
)

// DocType represents the category of an engineering documentation asset.
type DocType string

const (
	// DocReadme represents project README documentation.
	DocReadme DocType = "readme"

	// DocContributing represents contribution guidelines.
	DocContributing DocType = "contributing"

	// DocSecurity represents security policy documentation.
	DocSecurity DocType = "security"

	// DocLicense represents software license documentation.
	DocLicense DocType = "license"

	// DocChangelog represents project change log documentation.
	DocChangelog DocType = "changelog"

	// DocRoadmap represents project roadmap documentation.
	DocRoadmap DocType = "roadmap"

	// DocADR represents Architecture Decision Records.
	DocADR DocType = "adr"

	// DocGeneral represents general engineering documentation files.
	DocGeneral DocType = "general"

	// DocUnknown represents unclassified documentation.
	DocUnknown DocType = "unknown"
)
