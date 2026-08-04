package cli

import (
	"path/filepath"
	"strings"
)

// Config represents the immutable bootstrap configuration for the CLI application.
type Config struct {
	appName string
	version string
	rootDir string
}

// NewConfig constructs and validates a new immutable Config.
func NewConfig(appName string, version string, rootDir string) (*Config, error) {
	cleanName := strings.TrimSpace(appName)
	if cleanName == "" {
		cleanName = "limoxel"
	}

	cleanVer := strings.TrimSpace(version)
	if cleanVer == "" {
		cleanVer = "1.0.0"
	}

	cleanDir := filepath.Clean(strings.TrimSpace(rootDir))
	if cleanDir == "" || cleanDir == "." {
		cleanDir = "."
	}

	return &Config{
		appName: cleanName,
		version: cleanVer,
		rootDir: cleanDir,
	}, nil
}

// AppName returns the application name string.
func (c *Config) AppName() string {
	if c == nil {
		return ""
	}
	return c.appName
}

// Version returns the application version string.
func (c *Config) Version() string {
	if c == nil {
		return ""
	}
	return c.version
}

// RootDir returns the root workspace directory path.
func (c *Config) RootDir() string {
	if c == nil {
		return ""
	}
	return c.rootDir
}
