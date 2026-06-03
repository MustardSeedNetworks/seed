package discovery

// profiler_scan.go contains the active probing performed against a single
// device: TCP port scan (with banner grab), HTTP/HTTPS probe, SNMP probe,
// full-MIB SNMP collection, and the profileDevice orchestrator.

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/protocols/snmp"
)

// profileDevice performs the actual profiling.
func (p *DeviceProfiler) profileDevice(ip string) {
	defer func() {
		p.mu.Lock()
		delete(p.profiling, ip)
		p.mu.Unlock()
	}()

	stopCh := p.checkShutdown()
	if stopCh == nil {
		return
	}

	ctx, _, cleanup := p.createProfilingContext(stopCh)
	defer cleanup()

	profile := &DeviceProfile{
		ProfiledAt:  time.Now(),
		OpenPorts:   p.scanPorts(ctx, ip),
		DeviceIcons: []string{},
	}

	profile.HTTPInfo = p.probeHTTPFromOpenPorts(ctx, ip, profile.OpenPorts)
	profile.SNMPInfo = p.probeSNMPAndLog(ctx, ip)

	// Collect full SNMP MIB data if SNMP collector is available
	p.collectFullSNMPData(ctx, ip)

	// Perform name resolution (DNS, NetBIOS, mDNS)
	p.resolveDeviceNames(ctx, ip)

	p.inferDeviceType(profile)
	p.storeProfileAndLog(ctx, ip, profile)
}

// scanPorts scans the configured ports and returns open ports found.
func (p *DeviceProfiler) scanPorts(ctx context.Context, ip string) []OpenPort {
	portsToScan := p.config.GetPortsForIntensity()
	if len(portsToScan) == 0 {
		portsToScan = p.config.QuickPorts
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	openPorts := []OpenPort{}

	sem := make(chan struct{}, p.config.MaxConcurrent)

	for _, port := range portsToScan {
		wg.Add(1)
		go func(port int) {
			defer wg.Done()
			if result := p.scanSinglePort(ctx, ip, port, sem); result != nil {
				mu.Lock()
				openPorts = append(openPorts, *result)
				mu.Unlock()
			}
		}(port)
	}

	wg.Wait()
	return openPorts
}

// scanSinglePort scans a single port with rate limiting and returns the result if open.
func (p *DeviceProfiler) scanSinglePort(
	ctx context.Context,
	ip string,
	port int,
	sem chan struct{},
) *OpenPort {
	// Check for context cancellation before acquiring semaphore (fixes #834)
	select {
	case <-ctx.Done():
		return nil
	case sem <- struct{}{}:
	}
	defer func() { <-sem }()

	// Apply probe delay for IDS-friendly scanning with context check (fixes #834)
	if p.config.ProbeDelay > 0 {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(p.config.ProbeDelay):
		}
	}

	result := p.checkPortWithConfig(ctx, ip, port)
	if result.IsOpen {
		return &result
	}
	return nil
}

// probeHTTPFromOpenPorts probes HTTP/HTTPS on open ports and returns the first successful result.
func (p *DeviceProfiler) probeHTTPFromOpenPorts(
	ctx context.Context,
	ip string,
	openPorts []OpenPort,
) *HTTPInfo {
	for _, op := range openPorts {
		if op.Port == 80 || op.Port == 8080 {
			if info := p.probeHTTP(ctx, ip, op.Port, false); info != nil {
				return info
			}
		}
		if op.Port == 443 || op.Port == 8443 {
			if info := p.probeHTTP(ctx, ip, op.Port, true); info != nil {
				return info
			}
		}
	}
	return nil
}

// probeSNMPAndLog probes SNMP and logs the result if successful.
func (p *DeviceProfiler) probeSNMPAndLog(ctx context.Context, ip string) *SNMPInfo {
	info := p.probeSNMP(ctx, ip)
	if info != nil {
		logging.GetLogger().
			DebugContext(ctx, "Got SNMP info from device", "ip", ip, "sysName", info.SysName)
	}
	return info
}

// storeProfileAndLog stores the profile and logs the result.
func (p *DeviceProfiler) storeProfileAndLog(
	ctx context.Context,
	ip string,
	profile *DeviceProfile,
) {
	p.mu.Lock()
	p.profiles[ip] = profile
	p.mu.Unlock()

	logging.GetLogger().InfoContext(ctx,
		"Profiled device",
		"ip", ip,
		"open_ports", len(profile.OpenPorts),
		"type", profile.DeviceType,
		"icons", profile.DeviceIcons,
	)
}

// collectFullSNMPData performs full SNMP MIB collection for a device.
// This is called automatically when SNMP credentials are configured.
func (p *DeviceProfiler) collectFullSNMPData(ctx context.Context, ip string) {
	if p.snmpCollector == nil {
		return
	}

	logging.GetLogger().DebugContext(ctx, "Starting full SNMP MIB collection", "ip", ip)

	data, err := p.snmpCollector.Collect(ctx, ip)
	if err != nil {
		logging.GetLogger().DebugContext(ctx, "SNMP MIB collection failed", "ip", ip, "error", err)
		return
	}

	// Store the SNMP data
	p.mu.Lock()
	p.snmpData[ip] = data
	p.mu.Unlock()

	// Log summary of what was collected
	ifCount := len(data.Interfaces)
	macCount := len(data.MACTable)
	vlanCount := len(data.VLANs)
	lldpCount := len(data.LLDPNeighbors)
	errCount := len(data.Errors)

	if ifCount > 0 || macCount > 0 || vlanCount > 0 || lldpCount > 0 {
		logging.GetLogger().InfoContext(ctx, "SNMP MIB data collected",
			"ip", ip,
			"interfaces", ifCount,
			"macs", macCount,
			"vlans", vlanCount,
			"lldp", lldpCount,
			"errors", errCount)
	} else if errCount > 0 {
		logging.GetLogger().DebugContext(ctx, "SNMP MIB collection had errors",
			"ip", ip,
			"errors", data.Errors)
	}
}

// checkPortWithConfig checks if a TCP port is open using configurable settings.
// Uses ConnectTimeout from config and respects BannerGrab setting.
func (p *DeviceProfiler) checkPortWithConfig(ctx context.Context, ip string, port int) OpenPort {
	result := OpenPort{
		Port:     port,
		Protocol: "tcp",
		Service:  portToService(port),
		IsOpen:   false,
	}

	// Use configured connect timeout
	timeout := p.config.ConnectTimeout
	if timeout == 0 {
		timeout = p.config.Timeout
	}

	address := fmt.Sprintf("%s:%d", ip, port)
	d := net.Dialer{Timeout: timeout}

	conn, err := d.DialContext(ctx, "tcp", address)
	if err != nil {
		return result
	}
	defer func() { _ = conn.Close() }()

	result.IsOpen = true

	// Only grab banner if enabled in config
	if p.config.BannerGrab {
		// Fixes #902: Check context before attempting banner read to avoid blocking on cancelled context
		select {
		case <-ctx.Done():
			return result
		default:
		}
		// Try to grab banner for certain ports that typically send banners
		if port == portSSHProf || port == portFTP || port == portTelnet || port == portSMTP ||
			port == portPOP3 ||
			port == portIMAP ||
			port == portMySQL ||
			port == portPostgreSQL ||
			port == portRedis ||
			port == portMongoDB {
			_ = conn.SetReadDeadline(time.Now().Add(profilerBannerReadMs * time.Millisecond))
			banner := make([]byte, profilerBannerBufferSize)
			n, _ := conn.Read(banner)
			if n > 0 {
				result.Banner = strings.TrimSpace(string(banner[:n]))
			}
		}
	}

	return result
}

// probeHTTP probes an HTTP/HTTPS endpoint.
func (p *DeviceProfiler) probeHTTP(
	ctx context.Context,
	ip string,
	port int,
	isHTTPS bool,
) *HTTPInfo {
	scheme := "http"
	if isHTTPS {
		scheme = "https"
	}

	url := fmt.Sprintf("%s://%s:%d/", scheme, ip, port)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "The Seed/1.0")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil
	}
	defer func() { _ = resp.Body.Close() }()

	info := &HTTPInfo{
		Port:       port,
		StatusCode: resp.StatusCode,
		Server:     resp.Header.Get("Server"),
		IsHTTPS:    isHTTPS,
	}

	// Read limited body to extract title
	body, err := io.ReadAll(io.LimitReader(resp.Body, profilerHTTPBodyLimit))
	if err == nil {
		info.Title = extractHTMLTitle(string(body))
	}

	return info
}

// probeSNMP attempts to retrieve SNMP information from the device.
func (p *DeviceProfiler) probeSNMP(ctx context.Context, ip string) *SNMPInfo {
	if p.snmpConfig == nil {
		logging.GetLogger().DebugContext(ctx, "SNMP probe skipped - no SNMP config", "ip", ip)
		return nil
	}
	if len(p.snmpConfig.Communities) == 0 && len(p.snmpConfig.V3Credentials) == 0 {
		logging.GetLogger().DebugContext(
			ctx,
			"SNMP probe skipped - no communities or v3 credentials configured",
			"ip", ip,
		)
		return nil
	}

	logging.GetLogger().DebugContext(ctx,
		"Attempting SNMP probe",
		"ip",
		ip,
		"communities",
		len(p.snmpConfig.Communities),
		"v3creds",
		len(p.snmpConfig.V3Credentials),
	)

	// Query system information
	sysInfo, err := snmp.GetSystemInfo(ctx, ip, p.snmpConfig)
	if err != nil {
		logging.GetLogger().DebugContext(ctx, "SNMP probe failed", "ip", ip, "error", err)
		return nil
	}

	logging.GetLogger().InfoContext(ctx,
		"SNMP probe succeeded",
		"ip",
		ip,
		"sysName",
		sysInfo.SysName,
		"sysDescr",
		truncateString(sysInfo.SysDescr, profilerLogTruncateLen),
	)

	return &SNMPInfo{
		SysDescr:    sysInfo.SysDescr,
		SysName:     sysInfo.SysName,
		SysContact:  sysInfo.SysContact,
		SysLocation: sysInfo.SysLocation,
	}
}
