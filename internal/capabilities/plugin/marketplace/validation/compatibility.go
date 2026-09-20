package validation

import (
	"fmt"
	"strings"

	"github.com/unhield/limoxel/internal/capabilities/sdk/version"
)

// CheckCompatibility verifies that the host environment satisfies plugin version requirements.
func CheckCompatibility(minHostVerStr string, currentHostVer version.SemVer) CheckResult {
	minHostVerStr = strings.TrimSpace(minHostVerStr)
	if minHostVerStr == "" {
		return CheckResult{
			Name:     "HostCompatibility",
			Passed:   true,
			Details:  "No minimum host version specified; compatible with current host",
			Critical: false,
		}
	}

	required, err := version.ParseSemVer(minHostVerStr)
	if err != nil {
		return CheckResult{
			Name:     "HostCompatibility",
			Passed:   false,
			Details:  fmt.Sprintf("Invalid min_host_version '%s': %v", minHostVerStr, err),
			Critical: true,
		}
	}

	// Host version must be >= required min_host_version
	if currentHostVer.Compare(required) < 0 {
		return CheckResult{
			Name:     "HostCompatibility",
			Passed:   false,
			Details:  fmt.Sprintf("Host version %s is below required min_host_version %s", currentHostVer.String(), required.String()),
			Critical: true,
		}
	}

	return CheckResult{
		Name:     "HostCompatibility",
		Passed:   true,
		Details:  fmt.Sprintf("Host version %s satisfies requirement %s", currentHostVer.String(), required.String()),
		Critical: true,
	}
}
