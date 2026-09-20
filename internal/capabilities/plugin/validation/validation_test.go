package validation_test

import (
	"testing"

	pkgerr "github.com/unhield/limoxel/internal/capabilities/plugin/errors"
	"github.com/unhield/limoxel/internal/capabilities/plugin/model"
	"github.com/unhield/limoxel/internal/capabilities/plugin/validation"
	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

func sampleManifest() *model.Manifest {
	m, _ := model.ParseManifestJSON([]byte(`{
		"schema_version": "1.0.0",
		"id": "org.limoxel.valid",
		"name": "Valid Plugin",
		"version": "1.0.0",
		"entrypoint": "plugin.go",
		"capabilities": [
			{"name": "cap.one", "version": "1.0.0"}
		],
		"compatibility": {
			"min_host_version": "1.0.0",
			"max_host_version": "2.0.0",
			"supported_platforms": ["windows", "linux"]
		}
	}`))
	return m
}

func TestValidator_Valid(t *testing.T) {
	hostVer, _ := version.ParseSemVer("1.4.0")
	val := validation.NewValidator(hostVer, validation.WithPlatform("linux"))

	m := sampleManifest()
	if err := val.Validate(m); err != nil {
		t.Fatalf("expected validation to pass, got: %v", err)
	}
}

func TestValidator_HostVersionIncompatible(t *testing.T) {
	m := sampleManifest()

	// Host version too low
	lowHost, _ := version.ParseSemVer("0.9.0")
	valLow := validation.NewValidator(lowHost, validation.WithPlatform("linux"))
	err := valLow.Validate(m)
	if err == nil {
		t.Fatal("expected validation error for low host version, got nil")
	}
	pErr, ok := err.(*pkgerr.PluginError)
	if !ok || pErr.Code() != pkgerr.CodeIncompatibleHost {
		t.Errorf("expected CodeIncompatibleHost, got %v", err)
	}

	// Host version too high
	highHost, _ := version.ParseSemVer("3.0.0")
	valHigh := validation.NewValidator(highHost, validation.WithPlatform("linux"))
	err = valHigh.Validate(m)
	if err == nil {
		t.Fatal("expected validation error for high host version, got nil")
	}
	pErr, ok = err.(*pkgerr.PluginError)
	if !ok || pErr.Code() != pkgerr.CodeIncompatibleHost {
		t.Errorf("expected CodeIncompatibleHost, got %v", err)
	}
}

func TestValidator_PlatformIncompatible(t *testing.T) {
	hostVer, _ := version.ParseSemVer("1.4.0")
	val := validation.NewValidator(hostVer, validation.WithPlatform("freebsd"))

	m := sampleManifest() // supports windows, linux
	err := val.Validate(m)
	if err == nil {
		t.Fatal("expected validation error for unsupported platform, got nil")
	}
	pErr, ok := err.(*pkgerr.PluginError)
	if !ok || pErr.Code() != pkgerr.CodeIncompatibleHost {
		t.Errorf("expected CodeIncompatibleHost, got %v", err)
	}
}

func TestValidator_PathTraversal(t *testing.T) {
	hostVer, _ := version.ParseSemVer("1.4.0")
	val := validation.NewValidator(hostVer, validation.WithPlatform("linux"))

	m := sampleManifest()
	m.Entrypoint = "../../../etc/passwd"

	err := val.Validate(m)
	if err == nil {
		t.Fatal("expected security policy violation error for path traversal, got nil")
	}
	pErr, ok := err.(*pkgerr.PluginError)
	if !ok || pErr.Code() != pkgerr.CodeSecurityPolicyViolation {
		t.Errorf("expected CodeSecurityPolicyViolation, got %v", err)
	}
}

func TestValidator_PathSafety_VolumeAndUNC(t *testing.T) {
	hostVer, _ := version.ParseSemVer("1.4.0")
	val := validation.NewValidator(hostVer, validation.WithPlatform("windows"))

	m := sampleManifest()

	// Volume path
	m.Entrypoint = "C:escape.exe"
	if err := val.Validate(m); err == nil {
		t.Error("expected volume path C:escape.exe to be rejected")
	}

	// UNC path
	m.Entrypoint = `\\server\share\escape.exe`
	if err := val.Validate(m); err == nil {
		t.Error("expected UNC path to be rejected")
	}

	// Absolute Unix path
	m.Entrypoint = "/bin/sh"
	if err := val.Validate(m); err == nil {
		t.Error("expected absolute path /bin/sh to be rejected")
	}
}

func TestValidator_PathSafety_WindowsTargetComprehensive(t *testing.T) {
	hostVer, _ := version.ParseSemVer("1.4.0")
	val := validation.NewValidator(hostVer, validation.WithPlatform("windows"))

	rejectedPaths := []string{
		"C:escape.exe",
		`C:\escape.exe`,
		"C:/escape.exe",
		"d:sub/app.exe",
		`\\server\share\escape.exe`,
		"//server/share/escape.exe",
		"/absolute/path",
		`\absolute\path`,
		"..",
		"../escape.exe",
		`..\escape.exe`,
		"dir/../../escape.exe",
		`dir\..\..\escape.exe`,
		"plugin.exe:ads",
	}

	for _, p := range rejectedPaths {
		m := sampleManifest()
		m.Entrypoint = p
		if err := val.Validate(m); err == nil {
			t.Errorf("expected path %q to be rejected for Windows target", p)
		}
	}

	acceptedPaths := []string{
		"plugin.exe",
		"bin/plugin.exe",
		`bin\plugin.exe`,
		"nested/dir/plugin.exe",
	}

	for _, p := range acceptedPaths {
		m := sampleManifest()
		m.Entrypoint = p
		if err := val.Validate(m); err != nil {
			t.Errorf("expected path %q to be accepted for Windows target, got error: %v", p, err)
		}
	}
}

func TestValidator_PathSafety_UnixTargetComprehensive(t *testing.T) {
	hostVer, _ := version.ParseSemVer("1.4.0")
	val := validation.NewValidator(hostVer, validation.WithPlatform("linux"))

	rejectedPaths := []string{
		"/bin/sh",
		"/usr/local/bin/plugin",
		"..",
		"../escape",
		"dir/../../escape",
		"C:escape",
		`\\server\share\escape`,
		"//server/share/escape",
	}

	for _, p := range rejectedPaths {
		m := sampleManifest()
		m.Entrypoint = p
		if err := val.Validate(m); err == nil {
			t.Errorf("expected path %q to be rejected for Linux target", p)
		}
	}

	acceptedPaths := []string{
		"plugin",
		"bin/plugin",
		"nested/dir/plugin",
	}

	for _, p := range acceptedPaths {
		m := sampleManifest()
		m.Entrypoint = p
		if err := val.Validate(m); err != nil {
			t.Errorf("expected path %q to be accepted for Linux target, got error: %v", p, err)
		}
	}
}
