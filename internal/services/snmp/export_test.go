package snmp

import (
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
)

// CollectorConfig returns the config for testing.
func (c *Collector) CollectorConfig() *config.SNMPConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config
}

// CollectorThresholds returns the thresholds for testing.
func (c *Collector) CollectorThresholds() Thresholds {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.thresholds
}

// CollectorMu exposes the mutex for testing.
func (c *Collector) CollectorMu() *sync.RWMutex {
	return &c.mu
}

// DetermineStatus is exported for testing.
func (c *Collector) DetermineStatus(elapsed time.Duration, thresholds Thresholds) Status {
	return c.determineStatus(elapsed, thresholds)
}

// ExportIsValidIP is exported for testing.
func ExportIsValidIP(ip string) bool {
	return isValidIP(ip)
}
