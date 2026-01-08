// Package snmp provides SNMP data collection and device discovery for the sap module.
// It wraps the internal/snmp package with higher-level abstractions for telemetry collection.
package snmp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
	coresnmp "github.com/krisarmstrong/seed/internal/snmp"
)

// Status represents the status of an SNMP operation.
type Status string

// SNMP operation status constants.
const (
	StatusSuccess Status = "success"
	StatusWarning Status = "warning"
	StatusError   Status = "error"
	StatusUnknown Status = "unknown"
)

// Default timeout and threshold values.
const (
	DefaultTimeoutSec          = 5
	DefaultRetries             = 2
	DefaultWarningThresholdMs  = 500
	DefaultCriticalThresholdMs = 2000
	SNMPTimeticksPerSecond     = 100
	defaultSNMPPort            = 161
	hoursPerDay                = 24
	minutesPerHour             = 60
	secondsPerMinute           = 60
)

// ErrNilConfig is returned when SNMP config is nil.
var ErrNilConfig = errors.New("SNMP config is nil")

// ErrNoCredentials is returned when no SNMP credentials are configured.
var ErrNoCredentials = errors.New("no SNMP communities or v3 credentials configured")

// DeviceInfo contains SNMP-collected device information.
type DeviceInfo struct {
	IP           string          `json:"ip"`
	SysName      string          `json:"sysName,omitempty"`
	SysDescr     string          `json:"sysDescr,omitempty"`
	SysLocation  string          `json:"sysLocation,omitempty"`
	SysContact   string          `json:"sysContact,omitempty"`
	SysUpTime    time.Duration   `json:"sysUpTime,omitempty"`
	SysUpTimeSec int64           `json:"sysUpTimeSec,omitempty"`
	Status       Status          `json:"status"`
	ResponseMs   float64         `json:"responseMs"`
	CollectedAt  time.Time       `json:"collectedAt"`
	Error        string          `json:"error,omitempty"`
	Interfaces   []InterfaceInfo `json:"interfaces,omitempty"`
}

// InterfaceInfo contains SNMP interface data.
type InterfaceInfo struct {
	Index       int    `json:"index"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type"`
	Speed       uint64 `json:"speedBps"`
	AdminStatus string `json:"adminStatus"`
	OperStatus  string `json:"operStatus"`
	InOctets    uint64 `json:"inOctets"`
	OutOctets   uint64 `json:"outOctets"`
	InErrors    uint64 `json:"inErrors"`
	OutErrors   uint64 `json:"outErrors"`
}

// VLANInfo contains SNMP VLAN data.
type VLANInfo struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Ports  []int  `json:"ports,omitempty"`
}

// MACEntry represents a MAC address table entry.
type MACEntry struct {
	MACAddress string `json:"macAddress"`
	Port       int    `json:"port"`
	VLANID     int    `json:"vlanId,omitempty"`
	Type       string `json:"type"` // dynamic, static
}

// CollectResult contains the result of a device collection.
type CollectResult struct {
	Device  *DeviceInfo `json:"device"`
	Success bool        `json:"success"`
	Error   string      `json:"error,omitempty"`
}

// Thresholds defines timing thresholds for SNMP operations.
type Thresholds struct {
	Warning  time.Duration
	Critical time.Duration
}

// DefaultThresholds returns reasonable default thresholds for SNMP.
func DefaultThresholds() Thresholds {
	return Thresholds{
		Warning:  DefaultWarningThresholdMs * time.Millisecond,
		Critical: DefaultCriticalThresholdMs * time.Millisecond,
	}
}

// Collector performs SNMP data collection.
type Collector struct {
	config     *config.SNMPConfig
	thresholds Thresholds
	mu         sync.RWMutex
}

// NewCollector creates a new SNMP collector with the given config.
func NewCollector(cfg *config.SNMPConfig) *Collector {
	return &Collector{
		config:     cfg,
		thresholds: DefaultThresholds(),
	}
}

// SetThresholds updates the thresholds.
func (c *Collector) SetThresholds(t Thresholds) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.thresholds = t
}

// GetThresholds returns the current thresholds.
func (c *Collector) GetThresholds() Thresholds {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.thresholds
}

// CollectDevice gathers SNMP system information from a device.
func (c *Collector) CollectDevice(ctx context.Context, ip string) *CollectResult {
	c.mu.RLock()
	cfg := c.config
	thresholds := c.thresholds
	c.mu.RUnlock()

	result := &CollectResult{
		Device: &DeviceInfo{
			IP:          ip,
			Status:      StatusUnknown,
			CollectedAt: time.Now(),
		},
	}

	// Validate IP address
	if !isValidIP(ip) {
		result.Error = "invalid IP address"
		result.Device.Error = result.Error
		result.Device.Status = StatusError
		return result
	}

	if cfg == nil {
		result.Error = "SNMP config is nil"
		result.Device.Error = result.Error
		result.Device.Status = StatusError
		return result
	}

	start := time.Now()
	sysInfo, err := coresnmp.GetSystemInfo(ctx, ip, cfg)
	elapsed := time.Since(start)

	result.Device.ResponseMs = float64(elapsed.Milliseconds())

	if err != nil {
		result.Error = err.Error()
		result.Device.Error = err.Error()
		result.Device.Status = StatusError
		return result
	}

	// Populate device info
	result.Device.SysName = sysInfo.SysName
	result.Device.SysDescr = sysInfo.SysDescr
	result.Device.SysLocation = sysInfo.SysLocation
	result.Device.SysContact = sysInfo.SysContact
	result.Device.SysUpTime = time.Duration(sysInfo.SysUpTime) * time.Second / SNMPTimeticksPerSecond
	result.Device.SysUpTimeSec = int64(result.Device.SysUpTime.Seconds())
	result.Device.Status = c.determineStatus(elapsed, thresholds)
	result.Success = true

	return result
}

// CollectDeviceWithCommunity gathers SNMP data using a specific community string.
func (c *Collector) CollectDeviceWithCommunity(
	ctx context.Context,
	ip, community string,
) *CollectResult {
	c.mu.RLock()
	baseCfg := c.config
	thresholds := c.thresholds
	c.mu.RUnlock()

	result := &CollectResult{
		Device: &DeviceInfo{
			IP:          ip,
			Status:      StatusUnknown,
			CollectedAt: time.Now(),
		},
	}

	if !isValidIP(ip) {
		result.Error = "invalid IP address"
		result.Device.Error = result.Error
		result.Device.Status = StatusError
		return result
	}

	if baseCfg == nil {
		result.Error = "SNMP config is nil"
		result.Device.Error = result.Error
		result.Device.Status = StatusError
		return result
	}

	// Create a copy of config with the specified community prepended
	cfg := *baseCfg
	if community != "" {
		cfg.Communities = append([]string{community}, cfg.Communities...)
	}

	start := time.Now()
	sysInfo, err := coresnmp.GetSystemInfo(ctx, ip, &cfg)
	elapsed := time.Since(start)

	result.Device.ResponseMs = float64(elapsed.Milliseconds())

	if err != nil {
		result.Error = err.Error()
		result.Device.Error = err.Error()
		result.Device.Status = StatusError
		return result
	}

	result.Device.SysName = sysInfo.SysName
	result.Device.SysDescr = sysInfo.SysDescr
	result.Device.SysLocation = sysInfo.SysLocation
	result.Device.SysContact = sysInfo.SysContact
	result.Device.SysUpTime = time.Duration(sysInfo.SysUpTime) * time.Second / SNMPTimeticksPerSecond
	result.Device.SysUpTimeSec = int64(result.Device.SysUpTime.Seconds())
	result.Device.Status = c.determineStatus(elapsed, thresholds)
	result.Success = true

	return result
}

// Query performs a single SNMP GET query.
func (c *Collector) Query(ctx context.Context, ip, oid string) (string, error) {
	c.mu.RLock()
	cfg := c.config
	c.mu.RUnlock()

	if !isValidIP(ip) {
		return "", fmt.Errorf("invalid IP address: %s", ip)
	}

	if cfg == nil {
		return "", ErrNilConfig
	}

	return coresnmp.Query(ctx, ip, oid, cfg)
}

// QueryMultiple performs multiple SNMP GET queries.
func (c *Collector) QueryMultiple(
	ctx context.Context,
	ip string,
	oids []string,
) (map[string]string, error) {
	c.mu.RLock()
	cfg := c.config
	c.mu.RUnlock()

	if !isValidIP(ip) {
		return nil, fmt.Errorf("invalid IP address: %s", ip)
	}

	if cfg == nil {
		return nil, ErrNilConfig
	}

	return coresnmp.QueryMultiple(ctx, ip, oids, cfg)
}

// determineStatus calculates status based on response time and thresholds.
func (c *Collector) determineStatus(elapsed time.Duration, thresholds Thresholds) Status {
	if elapsed >= thresholds.Critical {
		return StatusError
	}
	if elapsed >= thresholds.Warning {
		return StatusWarning
	}
	return StatusSuccess
}

// isValidIP validates an IP address.
func isValidIP(ip string) bool {
	if ip == "" {
		return false
	}
	return net.ParseIP(ip) != nil
}

// ValidateConfig validates an SNMP configuration.
func ValidateConfig(cfg *config.SNMPConfig) error {
	if cfg == nil {
		return ErrNilConfig
	}

	const maxPort = 65535
	if cfg.Port <= 0 || cfg.Port > maxPort {
		return fmt.Errorf("invalid SNMP port: %d", cfg.Port)
	}

	if cfg.Timeout <= 0 {
		return fmt.Errorf("invalid SNMP timeout: %v", cfg.Timeout)
	}

	if len(cfg.Communities) == 0 && len(cfg.V3Credentials) == 0 {
		return ErrNoCredentials
	}

	return nil
}

// DefaultConfig returns a default SNMP configuration for testing.
func DefaultConfig() *config.SNMPConfig {
	return &config.SNMPConfig{
		Port:        defaultSNMPPort,
		Timeout:     time.Duration(DefaultTimeoutSec) * time.Second,
		Retries:     DefaultRetries,
		Communities: []string{"public"},
	}
}

// FormatUptime formats an uptime duration as a human-readable string.
func FormatUptime(d time.Duration) string {
	if d < 0 {
		return "unknown"
	}

	days := int(d.Hours()) / hoursPerDay
	hours := int(d.Hours()) % hoursPerDay
	minutes := int(d.Minutes()) % minutesPerHour
	seconds := int(d.Seconds()) % secondsPerMinute

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
