package discovery

// profiler_resolve.go contains the DNS / NetBIOS / mDNS name-resolution path
// for DeviceProfiler, plus the accessor for resolved names and helpers for
// interface selection.

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/krisarmstrong/seed/internal/logging"
)

// resolveDeviceNames performs DNS, NetBIOS, and mDNS name resolution for a device.
func (p *DeviceProfiler) resolveDeviceNames(ctx context.Context, ip string) {
	if !p.config.EnableNameResolution {
		return
	}

	names := &ResolvedNames{}
	resolved := false

	// DNS PTR lookup
	if p.config.ResolveDNS {
		if hostname := p.resolveDNS(ctx, ip); hostname != "" {
			names.Hostname = hostname
			resolved = true
		}
	}

	// NetBIOS name query
	if p.config.ResolveNetBIOS && p.netbiosResolver != nil {
		if nbName := p.resolveNetBIOS(ctx, ip); nbName != "" {
			names.NetBIOSName = nbName
			resolved = true
		}
	}

	// mDNS name query
	if p.config.ResolveMDNS && p.mdnsResolver != nil {
		if mdnsName := p.resolveMDNS(ctx, ip); mdnsName != "" {
			names.MDNSName = mdnsName
			resolved = true
		}
	}

	// Only store if we resolved at least one name
	if resolved {
		p.mu.Lock()
		p.resolvedNames[ip] = names
		p.mu.Unlock()

		logging.GetLogger().DebugContext(ctx, "Name resolution completed",
			"ip", ip,
			"hostname", names.Hostname,
			"netbios", names.NetBIOSName,
			"mdns", names.MDNSName)
	}
}

// resolveDNS performs reverse DNS lookup for an IP.
func (p *DeviceProfiler) resolveDNS(ctx context.Context, ip string) string {
	timeout := p.config.NameResolutionTimeout
	if timeout == 0 {
		timeout = profilerNameResolveTimeMs * time.Millisecond
	}

	lookupCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	names, err := net.DefaultResolver.LookupAddr(lookupCtx, ip)
	if err != nil {
		logging.GetLogger().DebugContext(ctx, "DNS lookup failed", "ip", ip, "error", err)
		return ""
	}

	if len(names) > 0 {
		hostname := strings.TrimSuffix(names[0], ".")
		logging.GetLogger().DebugContext(ctx, "DNS resolved", "ip", ip, "hostname", hostname)
		return hostname
	}
	return ""
}

// resolveNetBIOS performs NetBIOS name query for an IP.
func (p *DeviceProfiler) resolveNetBIOS(ctx context.Context, ip string) string {
	if p.netbiosResolver == nil {
		return ""
	}

	name, err := p.netbiosResolver.ResolveIP(ctx, ip)
	if err != nil {
		logging.GetLogger().DebugContext(ctx, "NetBIOS lookup failed", "ip", ip, "error", err)
		return ""
	}
	if name != "" {
		logging.GetLogger().DebugContext(ctx, "NetBIOS resolved", "ip", ip, "name", name)
	}
	return name
}

// resolveMDNS performs mDNS name query for an IP.
func (p *DeviceProfiler) resolveMDNS(ctx context.Context, ip string) string {
	if p.mdnsResolver == nil {
		return ""
	}

	name, err := p.mdnsResolver.ResolveIP(ctx, ip)
	if err != nil {
		logging.GetLogger().DebugContext(ctx, "mDNS lookup failed", "ip", ip, "error", err)
		return ""
	}
	if name != "" {
		logging.GetLogger().DebugContext(ctx, "mDNS resolved", "ip", ip, "name", name)
	}
	return name
}

// GetResolvedNames returns the resolved names for an IP address.
func (p *DeviceProfiler) GetResolvedNames(ip string) *ResolvedNames {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.resolvedNames[ip]
}

// SetInterface sets the interface name for mDNS resolver.
func (p *DeviceProfiler) SetInterface(interfaceName string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.interfaceName = interfaceName
	if p.mdnsResolver != nil {
		p.mdnsResolver = NewMDNSResolver(interfaceName)
	}
}

// ClearResolvedNames removes all stored resolved names.
func (p *DeviceProfiler) ClearResolvedNames() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.resolvedNames = make(map[string]*ResolvedNames)
}
