// Package api provides HTTP/WebSocket server functionality.
// This file implements trusted proxy support for proper X-Forwarded-For handling.
//
// SECURITY BACKGROUND:
// The X-Forwarded-For header can be spoofed by any client. When not behind a trusted
// proxy, we MUST use RemoteAddr (the actual TCP connection source) for security-critical
// operations like rate limiting.
//
// However, when running behind a trusted reverse proxy (nginx, HAProxy, etc.), the proxy
// sets X-Forwarded-For to the real client IP. In this case, we should trust the header
// IF AND ONLY IF the request comes from a configured trusted proxy.
//
// Usage:
//
//	seed serve --trusted-proxies 127.0.0.1,10.0.0.0/8
//
// When running behind nginx:
//
//	location / {
//	    proxy_pass http://localhost:8443;
//	    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
//	    proxy_set_header X-Real-IP $remote_addr;
//	}
package api

import (
	"log/slog"
	"net"
	"net/http"
	"strings"
)

// TrustedProxies manages a list of trusted proxy IP addresses and CIDR ranges.
// Requests from these IPs will have their X-Forwarded-For headers trusted.
type TrustedProxies struct {
	ips    map[string]bool
	cidrs  []*net.IPNet
	logger *slog.Logger
}

// NewTrustedProxies creates a new TrustedProxies from a comma-separated list of
// IP addresses and/or CIDR ranges. Invalid entries are logged and skipped.
//
// Examples:
//   - "127.0.0.1" - single IP
//   - "10.0.0.0/8" - CIDR range
//   - "127.0.0.1,192.168.1.0/24" - multiple entries
func NewTrustedProxies(proxies string) *TrustedProxies {
	tp := &TrustedProxies{
		ips:    make(map[string]bool),
		cidrs:  make([]*net.IPNet, 0),
		logger: slog.Default(),
	}

	if proxies == "" {
		return tp
	}

	for entry := range strings.SplitSeq(proxies, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		// Try to parse as CIDR
		if strings.Contains(entry, "/") {
			_, cidr, err := net.ParseCIDR(entry)
			if err != nil {
				tp.logger.Warn("Invalid CIDR in trusted proxies, skipping",
					"entry", entry,
					"error", err)
				continue
			}
			tp.cidrs = append(tp.cidrs, cidr)
			tp.logger.Info("Added trusted proxy CIDR", "cidr", entry)
		} else {
			// Parse as single IP
			ip := net.ParseIP(entry)
			if ip == nil {
				tp.logger.Warn("Invalid IP in trusted proxies, skipping", "entry", entry)
				continue
			}
			tp.ips[ip.String()] = true
			tp.logger.Info("Added trusted proxy IP", "ip", entry)
		}
	}

	return tp
}

// IsTrusted checks if the given IP address is in the trusted proxies list.
func (tp *TrustedProxies) IsTrusted(ipStr string) bool {
	if tp == nil || (len(tp.ips) == 0 && len(tp.cidrs) == 0) {
		return false
	}

	// Parse the IP address
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	// Check exact IP match
	if tp.ips[ip.String()] {
		return true
	}

	// Check CIDR ranges
	for _, cidr := range tp.cidrs {
		if cidr.Contains(ip) {
			return true
		}
	}

	return false
}

// IsEmpty returns true if no trusted proxies are configured.
func (tp *TrustedProxies) IsEmpty() bool {
	return tp == nil || (len(tp.ips) == 0 && len(tp.cidrs) == 0)
}

// Count returns the total number of trusted entries (IPs + CIDRs).
func (tp *TrustedProxies) Count() int {
	if tp == nil {
		return 0
	}
	return len(tp.ips) + len(tp.cidrs)
}

// GetClientIPWithProxy extracts the client IP from the request, considering trusted proxies.
// If the request comes from a trusted proxy, it will check X-Forwarded-For.
// Otherwise, it uses RemoteAddr (the only secure option).
//
// X-Forwarded-For format: client, proxy1, proxy2, ...
// We take the FIRST IP which is the original client (if from trusted proxy).
func (tp *TrustedProxies) GetClientIPWithProxy(r *http.Request) string {
	// Extract RemoteAddr IP
	remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteIP = r.RemoteAddr
	}

	// If no trusted proxies configured, always use RemoteAddr
	if tp.IsEmpty() {
		return remoteIP
	}

	// Check if request is from a trusted proxy
	if !tp.IsTrusted(remoteIP) {
		// Not from trusted proxy - use RemoteAddr only
		// Log at debug level to help troubleshoot misconfiguration
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			slog.Debug("Ignoring X-Forwarded-For from untrusted source",
				"xff", xff,
				"remote_addr", r.RemoteAddr)
		}
		return remoteIP
	}

	// Request is from trusted proxy - check X-Forwarded-For
	xff := r.Header.Get("X-Forwarded-For")
	if xff == "" {
		// No XFF header, use X-Real-IP as fallback
		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			ip := net.ParseIP(strings.TrimSpace(xri))
			if ip != nil {
				slog.Debug("Using X-Real-IP from trusted proxy",
					"client_ip", ip.String(),
					"proxy", remoteIP)
				return ip.String()
			}
		}
		return remoteIP
	}

	// Parse X-Forwarded-For - take the first (leftmost) IP which is the original client
	ips := strings.Split(xff, ",")
	if len(ips) > 0 {
		clientIP := strings.TrimSpace(ips[0])
		ip := net.ParseIP(clientIP)
		if ip != nil {
			slog.Debug("Using X-Forwarded-For from trusted proxy",
				"client_ip", ip.String(),
				"proxy", remoteIP,
				"xff_chain", xff)
			return ip.String()
		}
	}

	return remoteIP
}
