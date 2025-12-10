// Package validation provides input validation utilities.
package validation

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// validHostnameRegex matches valid hostnames (letters, numbers, dots, hyphens)
var validHostnameRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)

// validInterfaceRegex matches valid network interface names
// Linux: eth0, enp0s3, wlan0, docker0, br-xxx, vethXXX, lo
// macOS: en0, en1, lo0, bridge0, utun0
var validInterfaceRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{0,14}\d?$`)

// IsValidIP checks if the string is a valid IPv4 or IPv6 address.
func IsValidIP(s string) bool {
	return net.ParseIP(s) != nil
}

// IsValidIPv4 checks if the string is a valid IPv4 address.
func IsValidIPv4(s string) bool {
	ip := net.ParseIP(s)
	return ip != nil && ip.To4() != nil
}

// IsValidHostname checks if the string is a valid hostname.
func IsValidHostname(s string) bool {
	if s == "" || len(s) > 253 {
		return false
	}
	return validHostnameRegex.MatchString(s)
}

// IsValidHostOrIP checks if the string is a valid hostname or IP address.
func IsValidHostOrIP(s string) bool {
	return IsValidIP(s) || IsValidHostname(s)
}

// ValidateServerAddress validates a server address (IP or hostname).
func ValidateServerAddress(server string) error {
	if server == "" {
		return fmt.Errorf("server address is required")
	}

	if IsValidIP(server) {
		return nil
	}

	if len(server) > 253 {
		return fmt.Errorf("server hostname too long")
	}

	if !validHostnameRegex.MatchString(server) {
		return fmt.Errorf("invalid server address: must be a valid IP or hostname")
	}

	return nil
}

// IsValidInterface checks if the string is a valid network interface name.
func IsValidInterface(iface string) bool {
	if iface == "" || len(iface) > 16 {
		return false
	}
	return validInterfaceRegex.MatchString(iface)
}

// ValidateInterface validates a network interface name.
func ValidateInterface(iface string) error {
	if iface == "" {
		return fmt.Errorf("interface name is required")
	}
	if len(iface) > 16 {
		return fmt.Errorf("interface name too long (max 16 characters)")
	}
	if !validInterfaceRegex.MatchString(iface) {
		return fmt.Errorf("invalid interface name: must contain only alphanumeric characters, hyphens, and underscores")
	}
	return nil
}

// IsValidURL checks if the string is a valid HTTP/HTTPS URL.
func IsValidURL(s string) bool {
	if s == "" {
		return false
	}

	u, err := url.Parse(s)
	if err != nil {
		return false
	}

	// Must have http or https scheme
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}

	// Must have a valid host
	if u.Host == "" {
		return false
	}

	// Extract host without port
	host := u.Hostname()
	return IsValidHostOrIP(host)
}

// ValidateURL validates a URL for HTTP testing.
// It prevents SSRF by blocking private/internal IPs.
func ValidateURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("URL is required")
	}

	// Add scheme if missing for parsing
	testURL := rawURL
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		testURL = "https://" + rawURL
	}

	u, err := url.Parse(testURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("URL must have a valid host")
	}

	// Validate host is valid IP or hostname
	if !IsValidHostOrIP(host) {
		return fmt.Errorf("invalid host in URL: %s", host)
	}

	// Check for private/internal IP addresses (SSRF prevention)
	if ip := net.ParseIP(host); ip != nil {
		if IsPrivateIP(ip) {
			return fmt.Errorf("URLs targeting private/internal IP addresses are not allowed")
		}
	}

	return nil
}

// IsPrivateIP checks if an IP address is private/internal.
// This is exported for use by SafeTransport.
func IsPrivateIP(ip net.IP) bool {
	// Localhost
	if ip.IsLoopback() {
		return true
	}

	// Link-local
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	// Private IPv4 ranges
	privateBlocks := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"169.254.0.0/16",
		"127.0.0.0/8",
	}

	for _, block := range privateBlocks {
		_, cidr, err := net.ParseCIDR(block)
		if err == nil && cidr.Contains(ip) {
			return true
		}
	}

	return false
}

// ValidatePort checks if the port number is valid.
func ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	return nil
}

// ValidateNetmask checks if the string is a valid IPv4 netmask.
func ValidateNetmask(netmask string) error {
	ip := net.ParseIP(netmask)
	if ip == nil {
		return fmt.Errorf("invalid netmask format")
	}

	ip4 := ip.To4()
	if ip4 == nil {
		return fmt.Errorf("netmask must be IPv4")
	}

	// Check it's a valid netmask (contiguous 1s followed by 0s)
	mask := net.IPv4Mask(ip4[0], ip4[1], ip4[2], ip4[3])
	ones, bits := mask.Size()
	if bits == 0 || ones == 0 {
		return fmt.Errorf("invalid netmask: not a valid subnet mask")
	}

	return nil
}

// ErrPrivateIPBlocked is returned when a connection attempt to a private IP is blocked.
var ErrPrivateIPBlocked = errors.New("connection to private/internal IP address blocked")

// SafeTransport returns an http.Transport that blocks connections to private IP addresses.
// This prevents DNS rebinding attacks where a hostname resolves to a public IP during
// validation but a private IP at connection time.
func SafeTransport() *http.Transport {
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	return &http.Transport{
		DisableKeepAlives: true,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// Extract host from address
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, fmt.Errorf("invalid address: %w", err)
			}

			// Resolve hostname to IP addresses
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, fmt.Errorf("DNS lookup failed: %w", err)
			}

			if len(ips) == 0 {
				return nil, errors.New("no IP addresses resolved")
			}

			// Check ALL resolved IPs for private addresses
			for _, ip := range ips {
				if IsPrivateIP(ip.IP) {
					return nil, fmt.Errorf("%w: %s resolved to %s", ErrPrivateIPBlocked, host, ip.IP)
				}
			}

			// Connect to the first resolved IP
			targetAddr := net.JoinHostPort(ips[0].IP.String(), port)
			return dialer.DialContext(ctx, network, targetAddr)
		},
	}
}

// SafeHTTPClient returns an http.Client using SafeTransport.
func SafeHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Transport: SafeTransport(),
		Timeout:   timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Don't follow redirects automatically - let the caller handle them
			// This prevents redirect-based SSRF attacks
			return http.ErrUseLastResponse
		},
	}
}
