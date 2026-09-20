package sandbox

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
)

var (
	// ErrNetworkDenied indicates that network access is prohibited by policy.
	ErrNetworkDenied = errors.New("network guard: network access is disabled")
	// ErrDestinationBlocked indicates the target host or port is not on the approved allowlist.
	ErrDestinationBlocked = errors.New("network guard: destination not in approved allowlist")
	// ErrLoopbackBlocked indicates loopback connection is prohibited.
	ErrLoopbackBlocked = errors.New("network guard: localhost/loopback access is prohibited")
)

// NetworkGuard regulates network access, socket mediation, and destination allowlists.
type NetworkGuard struct {
	mu             sync.RWMutex
	networkAllowed bool
	allowLocalhost bool
	allowedHosts   []string
}

// NewNetworkGuard constructs an initialized network guard.
func NewNetworkGuard(networkAllowed, allowLocalhost bool, allowedHosts []string) *NetworkGuard {
	return &NetworkGuard{
		networkAllowed: networkAllowed,
		allowLocalhost: allowLocalhost,
		allowedHosts:   allowedHosts,
	}
}

// IsNetworkAllowed returns true if outbound network access is enabled for this sandbox.
func (g *NetworkGuard) IsNetworkAllowed() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.networkAllowed
}

// ValidateConnection checks whether a target network address (host:port) is authorized.
func (g *NetworkGuard) ValidateConnection(network, address string) error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if !g.networkAllowed {
		return fmt.Errorf("%w: cannot connect to %s over %s", ErrNetworkDenied, address, network)
	}

	host, _, err := net.SplitHostPort(address)
	if err != nil {
		// Might just be host without port
		host = address
	}
	host = strings.ToLower(strings.TrimSpace(host))

	// Check loopback and unspecified addresses
	isLoopback := host == "localhost" || host == "localhost.localdomain" || host == "ip6-localhost" || host == "ip6-loopback" ||
		host == "127.0.0.1" || host == "::1" || host == "0.0.0.0" || host == "::" || host == "[::1]" || strings.HasPrefix(host, "127.")
	if !isLoopback {
		cleanHost := strings.Trim(host, "[]")
		if ip := net.ParseIP(cleanHost); ip != nil {
			if ip.IsLoopback() || ip.IsUnspecified() {
				isLoopback = true
			}
		}
	}
	if isLoopback && !g.allowLocalhost {
		return fmt.Errorf("%w: attempt to connect to loopback or unspecified local address '%s'", ErrLoopbackBlocked, address)
	}

	// If no allowlist is configured, any destination is permitted once network is allowed
	if len(g.allowedHosts) == 0 {
		return nil
	}

	for _, pattern := range g.allowedHosts {
		normPattern := strings.ToLower(strings.TrimSpace(pattern))
		if normPattern == "*" {
			return nil
		}
		if normPattern == host || normPattern == address {
			return nil
		}
		// Wildcard domain, e.g. "*.example.com"
		if strings.HasPrefix(normPattern, "*.") {
			suffix := strings.TrimPrefix(normPattern, "*.")
			if strings.HasSuffix(host, "."+suffix) || host == suffix {
				return nil
			}
		}
	}

	return fmt.Errorf("%w: destination '%s' does not match any allowed host pattern", ErrDestinationBlocked, address)
}

// SanitizeEnvironment strips sensitive proxy and network environment variables.
func (g *NetworkGuard) SanitizeEnvironment(env []string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var sanitized []string
	sensitivePrefixes := []string{
		"HTTP_PROXY=", "http_proxy=",
		"HTTPS_PROXY=", "https_proxy=",
		"ALL_PROXY=", "all_proxy=",
		"NO_PROXY=", "no_proxy=",
		"SSH_AUTH_SOCK=", "AWS_ACCESS_KEY_ID=", "AWS_SECRET_ACCESS_KEY=",
		"GITHUB_TOKEN=", "GITLAB_TOKEN=",
	}

	for _, e := range env {
		blocked := false
		for _, prefix := range sensitivePrefixes {
			if strings.HasPrefix(e, prefix) {
				blocked = true
				break
			}
		}
		if !blocked {
			sanitized = append(sanitized, e)
		}
	}

	return sanitized
}
