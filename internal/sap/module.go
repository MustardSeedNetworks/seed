// Package sap provides live network telemetry and testing services.
// Color: Cyan #0891b2
package sap

import (
	"context"
	"sync"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/database"
)

// Module is the main Sap module providing telemetry services.
type Module struct {
	mu          sync.RWMutex
	cfg         *config.Config
	db          *database.DB
	link        *LinkService
	cable       *CableService
	dhcp        *DHCPService
	dns         *DNSService
	gateway     *GatewayService
	snmp        *SNMPService
	performance *PerformanceService
	vlan        *VLANService
	telemetry   *TelemetryService
}

// New creates a new Sap module instance.
func New(cfg *config.Config, db *database.DB) *Module {
	m := &Module{
		cfg: cfg,
		db:  db,
	}

	m.link = NewLinkService(cfg)
	m.cable = NewCableService(cfg)
	m.dhcp = NewDHCPService(cfg)
	m.dns = NewDNSService(cfg)
	m.gateway = NewGatewayService(cfg)
	m.snmp = NewSNMPService(cfg)
	m.performance = NewPerformanceService(cfg)
	m.vlan = NewVLANService(cfg)
	m.telemetry = NewTelemetryService(cfg, db)

	return m
}

// Link returns the link monitoring service.
func (m *Module) Link() *LinkService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.link
}

// Cable returns the cable testing service.
func (m *Module) Cable() *CableService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cable
}

// DHCP returns the DHCP testing service.
func (m *Module) DHCP() *DHCPService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.dhcp
}

// DNS returns the DNS testing service.
func (m *Module) DNS() *DNSService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.dns
}

// Gateway returns the gateway health service.
func (m *Module) Gateway() *GatewayService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.gateway
}

// SNMP returns the SNMP collector service.
func (m *Module) SNMP() *SNMPService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.snmp
}

// Performance returns the performance testing service.
func (m *Module) Performance() *PerformanceService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.performance
}

// VLAN returns the VLAN management service.
func (m *Module) VLAN() *VLANService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.vlan
}

// Telemetry returns the telemetry aggregation service.
func (m *Module) Telemetry() *TelemetryService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.telemetry
}

// Start initializes and starts the Sap module services.
func (m *Module) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Start link monitoring
	if err := m.link.Start(ctx); err != nil {
		return err
	}

	// Start gateway monitoring
	if err := m.gateway.Start(ctx); err != nil {
		return err
	}

	// Start telemetry aggregation
	if err := m.telemetry.Start(ctx); err != nil {
		return err
	}

	return nil
}

// Stop gracefully shuts down all Sap module services.
func (m *Module) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.link != nil {
		m.link.Stop()
	}

	if m.gateway != nil {
		m.gateway.Stop()
	}

	if m.telemetry != nil {
		m.telemetry.Stop()
	}

	if m.performance != nil {
		m.performance.Stop()
	}

	return nil
}
