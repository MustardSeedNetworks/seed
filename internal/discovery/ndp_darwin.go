//go:build darwin
// +build darwin

// Package discovery implements multi-protocol network device discovery.
// NDP (Neighbor Discovery Protocol) support for macOS is a stub implementation
// as IPv6 neighbor discovery on macOS is complex and the primary production target is Linux.
package discovery

import (
	"fmt"
	"time"
)

// NDPScanner is a stub for macOS (production target is Linux).
type NDPScanner struct {
	interfaceName string
	neighbors     map[string]*NDPNeighbor
}

// NDPNeighbor represents an IPv6 neighbor.
type NDPNeighbor struct {
	IPv6     string
	MAC      string
	IsRouter bool
	State    string
	LastSeen time.Time
}

// NewNDPScanner creates a new IPv6 NDP scanner.
func NewNDPScanner(interfaceName string) *NDPScanner {
	return &NDPScanner{
		interfaceName: interfaceName,
		neighbors:     make(map[string]*NDPNeighbor),
	}
}

// Start is a stub on macOS.
func (ns *NDPScanner) Start() error {
	return fmt.Errorf("IPv6 NDP scanning not implemented on macOS (production target is Linux)")
}

// Stop is a stub on macOS.
func (ns *NDPScanner) Stop() error {
	return nil
}

// IsRunning returns false on macOS.
func (ns *NDPScanner) IsRunning() bool {
	return false
}

// GetNeighbors returns empty map on macOS.
func (ns *NDPScanner) GetNeighbors() map[string]*NDPNeighbor {
	return make(map[string]*NDPNeighbor)
}

// CleanupStale is a no-op on macOS.
func (ns *NDPScanner) CleanupStale(_ time.Duration) {
	// No-op
}
