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

// Validation limits for network-related fields.
const (
	// MaxHostnameLength is the maximum length of a valid hostname per RFC 1035.
	MaxHostnameLength = 253

	// MaxInterfaceNameLength is the maximum length of a network interface name.
	// Linux uses 16 characters (IFNAMSIZ), macOS is similar.
	MaxInterfaceNameLength = 16

	// MaxFilenameLength is the maximum length of a filename on most filesystems.
	MaxFilenameLength = 255

	// MaxSurveyIDLength is the maximum length of a survey ID.
	MaxSurveyIDLength = 64

	// DataURLPartCount is the expected number of parts when splitting a data URL by comma.
	DataURLPartCount = 2

	// DialerTimeout is the timeout for establishing connections.
	DialerTimeout = 10 * time.Second

	// DialerKeepAlive is the keep-alive interval for connections.
	DialerKeepAlive = 30 * time.Second
)

// validHostnameRegex matches valid hostnames (letters, numbers, dots, hyphens).
var validHostnameRegex = regexp.MustCompile(
	`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`,
)

// validInterfaceRegex matches valid network interface names.
// Linux: eth0, enp0s3, wlan0, docker0, br-xxx, vethXXX, lo.
// macOS: en0, en1, lo0, bridge0, utun0.
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
	if s == "" || len(s) > MaxHostnameLength {
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
		return errors.New("server address is required")
	}

	if IsValidIP(server) {
		return nil
	}

	if len(server) > MaxHostnameLength {
		return errors.New("server hostname too long")
	}

	if !validHostnameRegex.MatchString(server) {
		return errors.New("invalid server address: must be a valid IP or hostname")
	}

	return nil
}

// IsValidInterface checks if the string is a valid network interface name.
func IsValidInterface(iface string) bool {
	if iface == "" || len(iface) > MaxInterfaceNameLength {
		return false
	}
	return validInterfaceRegex.MatchString(iface)
}

// ValidateInterface validates a network interface name.
func ValidateInterface(iface string) error {
	if iface == "" {
		return errors.New("interface name is required")
	}
	if len(iface) > MaxInterfaceNameLength {
		return errors.New("interface name too long (max 16 characters)")
	}
	if !validInterfaceRegex.MatchString(iface) {
		return errors.New(
			"invalid interface name: must contain only alphanumeric characters, hyphens, and underscores",
		)
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
		return errors.New("URL is required")
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
		return errors.New("URL must have a valid host")
	}

	// Validate host is valid IP or hostname
	if !IsValidHostOrIP(host) {
		return fmt.Errorf("invalid host in URL: %s", host)
	}

	// Check for private/internal IP addresses (SSRF prevention)
	if ip := net.ParseIP(host); ip != nil {
		if IsPrivateIP(ip) {
			return errors.New("URLs targeting private/internal IP addresses are not allowed")
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
		return fmt.Errorf("port must be between 1 and 65535, got %d", port)
	}
	return nil
}

// ValidateVLANID checks if the VLAN ID is valid (fixes #522).
// Valid VLAN IDs are 1-4094 (4095 is reserved, 0 is invalid).
func ValidateVLANID(vlanID int) error {
	if vlanID < 1 || vlanID > 4094 {
		return fmt.Errorf("VLAN ID must be between 1 and 4094, got %d", vlanID)
	}
	return nil
}

// ValidatePositiveInt checks if the integer is non-negative (fixes #522).
func ValidatePositiveInt(val int, fieldName string) error {
	if val < 0 {
		return fmt.Errorf("%s must be non-negative, got %d", fieldName, val)
	}
	return nil
}

// ValidateMTU checks if the MTU value is valid (fixes #522).
// Standard Ethernet: 1500, Jumbo frames: up to 9000.
func ValidateMTU(mtu int) error {
	if mtu < 68 || mtu > 9000 {
		return fmt.Errorf("MTU must be between 68 and 9000, got %d", mtu)
	}
	return nil
}

// ValidateDNSAddress checks if the string is a valid DNS server address.
// Only allows valid IPv4/IPv6 addresses, not hostnames, to prevent injection.
func ValidateDNSAddress(dns string) error {
	if dns == "" {
		return errors.New("DNS server address is required")
	}

	ip := net.ParseIP(dns)
	if ip == nil {
		return fmt.Errorf("invalid DNS server: must be a valid IP address, got %q", dns)
	}

	return nil
}

// ValidateDNSServers validates a slice of DNS server addresses.
func ValidateDNSServers(servers []string) error {
	for i, dns := range servers {
		if err := ValidateDNSAddress(dns); err != nil {
			return fmt.Errorf("DNS server %d: %w", i+1, err)
		}
	}
	return nil
}

// ValidateNetmask checks if the string is a valid IPv4 netmask.
func ValidateNetmask(netmask string) error {
	ip := net.ParseIP(netmask)
	if ip == nil {
		return errors.New("invalid netmask format")
	}

	ip4 := ip.To4()
	if ip4 == nil {
		return errors.New("netmask must be IPv4")
	}

	// Check it's a valid netmask (contiguous 1s followed by 0s)
	mask := net.IPv4Mask(ip4[0], ip4[1], ip4[2], ip4[3])
	ones, bits := mask.Size()
	if bits == 0 || ones == 0 {
		return errors.New("invalid netmask: not a valid subnet mask")
	}

	return nil
}

// ValidateStringLength validates that a string is within the specified length bounds.
// Returns an error if the string is empty (when required) or exceeds maxLen (fixes #695).
func ValidateStringLength(s, fieldName string, minLen, maxLen int) error {
	if len(s) < minLen {
		if minLen == 1 {
			return fmt.Errorf("%s is required", fieldName)
		}
		return fmt.Errorf("%s must be at least %d characters, got %d", fieldName, minLen, len(s))
	}
	if len(s) > maxLen {
		return fmt.Errorf("%s too long (max %d characters), got %d", fieldName, maxLen, len(s))
	}
	return nil
}

// ValidateIntRange validates that an integer is within the specified range (fixes #695).
func ValidateIntRange(val int, fieldName string, minVal, maxVal int) error {
	if val < minVal || val > maxVal {
		return fmt.Errorf("%s must be between %d and %d, got %d", fieldName, minVal, maxVal, val)
	}
	return nil
}

// ValidateFloatRange validates that a float is within the specified range (fixes #695).
func ValidateFloatRange(val float64, fieldName string, minVal, maxVal float64) error {
	if val < minVal || val > maxVal {
		return fmt.Errorf("%s must be between %f and %f, got %f", fieldName, minVal, maxVal, val)
	}
	return nil
}

// ValidatePath validates a file path to prevent directory traversal attacks (fixes #695).
// It ensures the path doesn't contain dangerous sequences like ".." or absolute paths.
func ValidatePath(path, fieldName string) error {
	if path == "" {
		return fmt.Errorf("%s is required", fieldName)
	}

	// Block absolute paths (they should be resolved by the backend)
	if strings.HasPrefix(path, "/") || strings.HasPrefix(path, "\\") {
		return fmt.Errorf("%s must not be an absolute path", fieldName)
	}

	// Block parent directory references
	if strings.Contains(path, "..") {
		return fmt.Errorf("%s must not contain '..' sequences", fieldName)
	}

	// Block null bytes (poison null byte attack)
	if strings.ContainsRune(path, 0) {
		return fmt.Errorf("%s contains invalid characters", fieldName)
	}

	return nil
}

// ValidateFilename validates a filename to ensure it's safe (fixes #695).
// Rejects filenames with path separators, control characters, or dangerous patterns.
func ValidateFilename(filename, fieldName string) error {
	if filename == "" {
		return fmt.Errorf("%s is required", fieldName)
	}

	// Check length (filesystem limits, typically 255)
	if len(filename) > MaxFilenameLength {
		return fmt.Errorf("%s too long (max 255 characters)", fieldName)
	}

	// Block path separators
	if strings.ContainsAny(filename, "/\\") {
		return fmt.Errorf("%s must not contain path separators", fieldName)
	}

	// Block control characters and null bytes
	for _, r := range filename {
		if r < 32 || r == 127 {
			return fmt.Errorf("%s contains invalid characters", fieldName)
		}
	}

	// Block dangerous filenames
	if filename == "." || filename == ".." {
		return fmt.Errorf("%s is not a valid filename", fieldName)
	}

	return nil
}

// ValidateSurveyID validates a survey ID (fixes #695).
// Survey IDs must be alphanumeric with hyphens/underscores only.
func ValidateSurveyID(id string) error {
	if id == "" {
		return errors.New("survey ID is required")
	}

	if len(id) > MaxSurveyIDLength {
		return errors.New("survey ID too long (max 64 characters)")
	}

	// Check for valid characters
	for _, r := range id {
		isValid := (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '-' || r == '_'
		if !isValid {
			return errors.New(
				"survey ID contains invalid characters (use only letters, numbers, hyphens, underscores)",
			)
		}
	}

	return nil
}

// ValidateImageDataURL validates a data URL for image uploads (fixes #695).
// Ensures it's a valid data URL with an allowed image MIME type.
func ValidateImageDataURL(dataURL string, maxSizeBytes int) error {
	if dataURL == "" {
		return errors.New("image data is required")
	}

	// Check for data URL prefix
	if !strings.HasPrefix(dataURL, "data:") {
		return errors.New("invalid image data format (must be a data URL)")
	}

	// Extract MIME type
	parts := strings.SplitN(dataURL, ",", DataURLPartCount)
	if len(parts) != DataURLPartCount {
		return errors.New("invalid image data format")
	}

	// Validate MIME type
	mimeSection := parts[0]
	allowedTypes := []string{"image/png", "image/jpeg", "image/jpg", "image/gif", "image/webp"}
	validType := false
	for _, allowed := range allowedTypes {
		if strings.Contains(mimeSection, allowed) {
			validType = true
			break
		}
	}

	if !validType {
		return errors.New("unsupported image type (allowed: PNG, JPEG, GIF, WebP)")
	}

	// Check size (rough estimate: base64 is ~1.33x larger than binary)
	if len(dataURL) > maxSizeBytes {
		return fmt.Errorf("image data too large (max %d bytes)", maxSizeBytes)
	}

	return nil
}

// ErrPrivateIPBlocked is returned when a connection attempt to a private IP is blocked.
var ErrPrivateIPBlocked = errors.New("connection to private/internal IP address blocked")

// SafeTransport returns an [http.Transport] that blocks connections to private IP addresses.
// This prevents DNS rebinding attacks where a hostname resolves to a public IP during
// validation but a private IP at connection time.
func SafeTransport() *http.Transport {
	dialer := &net.Dialer{
		Timeout:   DialerTimeout,
		KeepAlive: DialerKeepAlive,
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
					return nil, fmt.Errorf(
						"%w: %s resolved to %s",
						ErrPrivateIPBlocked,
						host,
						ip.IP,
					)
				}
			}

			// Connect to the first resolved IP
			targetAddr := net.JoinHostPort(ips[0].IP.String(), port)
			return dialer.DialContext(ctx, network, targetAddr)
		},
	}
}

// SafeHTTPClient returns an [http.Client] using SafeTransport.
func SafeHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Transport: SafeTransport(),
		Timeout:   timeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			// Don't follow redirects automatically - let the caller handle them
			// This prevents redirect-based SSRF attacks
			return http.ErrUseLastResponse
		},
	}
}
