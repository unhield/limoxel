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
//
// Security Notice: This method performs in-process destination policy evaluation before
// connections are established. It does NOT intercept raw sockets or kernel syscalls of
// child processes, and does NOT prevent DNS rebinding attacks (IP pinning must be implemented
// at the socket dialer level).
func (g *NetworkGuard) ValidateConnection(network, address string) error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if !g.networkAllowed {
		return fmt.Errorf("%w: cannot connect to %s over %s", ErrNetworkDenied, address, network)
	}

	host, _, err := net.SplitHostPort(address)
	if err != nil {
		// Might just be host without port, or unbracketed IPv6 with port (e.g. ::1:8080)
		if ip := net.ParseIP(address); ip != nil {
			host = address
		} else if lastColon := strings.LastIndex(address, ":"); lastColon != -1 {
			possibleIP := address[:lastColon]
			possiblePort := address[lastColon+1:]
			if net.ParseIP(possibleIP) != nil && possiblePort != "" {
				host = possibleIP
			} else {
				host = address
			}
		} else {
			host = address
		}
	}
	host = strings.ToLower(strings.TrimSpace(host))

	cleanHost := strings.Trim(host, "[]")
	isLoopback := host == "localhost" || host == "localhost.localdomain" || host == "ip6-localhost" || host == "ip6-loopback" ||
		cleanHost == "127.0.0.1" || cleanHost == "::1" || cleanHost == "0.0.0.0" || cleanHost == "::" ||
		strings.HasPrefix(cleanHost, "127.") || strings.HasPrefix(cleanHost, "::ffff:127.") ||
		strings.HasPrefix(cleanHost, "::1:")
	if !isLoopback {
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
		// Wildcard domain, e.g. "*.example.com" matches "sub.example.com", NOT "example.com"
		if strings.HasPrefix(normPattern, "*.") {
			suffix := strings.TrimPrefix(normPattern, "*.")
			// Require suffix to contain at least one dot to prevent unrestricted TLD wildcards (*.com)
			if strings.Contains(suffix, ".") && !strings.HasPrefix(suffix, ".") && !strings.HasSuffix(suffix, ".") {
				if strings.HasSuffix(host, "."+suffix) {
					return nil
				}
			}
		}
	}

	return fmt.Errorf("%w: destination '%s' does not match any allowed host pattern", ErrDestinationBlocked, address)
}

// SanitizeEnvironment strips sensitive proxy and credential environment variables.
func (g *NetworkGuard) SanitizeEnvironment(env []string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var sanitized []string
	sensitiveKeys := map[string]struct{}{
		"HTTP_PROXY":            {},
		"HTTPS_PROXY":           {},
		"ALL_PROXY":             {},
		"NO_PROXY":              {},
		"SSH_AUTH_SOCK":         {},
		"AWS_ACCESS_KEY_ID":     {},
		"AWS_SECRET_ACCESS_KEY": {},
		"AWS_SESSION_TOKEN":     {},
		"GITHUB_TOKEN":          {},
		"GITLAB_TOKEN":          {},
	}

	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		key := strings.ToUpper(strings.TrimSpace(parts[0]))
		if _, blocked := sensitiveKeys[key]; !blocked {
			sanitized = append(sanitized, e)
		}
	}

	return sanitized
}
