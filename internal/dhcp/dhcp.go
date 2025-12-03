// Package dhcp provides DHCP transaction timing and monitoring.
package dhcp

import (
	"sync"
	"time"
)

// Phase represents a DHCP transaction phase.
type Phase string

const (
	PhaseDiscover Phase = "discover"
	PhaseOffer    Phase = "offer"
	PhaseRequest  Phase = "request"
	PhaseAck      Phase = "ack"
)

// Timing contains timing information for a complete DHCP transaction.
type Timing struct {
	Discover time.Duration `json:"discover"` // Time from Discover to Offer
	Offer    time.Duration `json:"offer"`    // Time from Offer to Request
	Request  time.Duration `json:"request"`  // Time from Request to Ack
	Total    time.Duration `json:"total"`    // Total transaction time
	Complete bool          `json:"complete"` // Whether all phases completed
}

// TimingMs contains timing in milliseconds for JSON serialization.
type TimingMs struct {
	Discover int64 `json:"discover"`
	Offer    int64 `json:"offer"`
	Request  int64 `json:"request"`
	Ack      int64 `json:"ack"`
	Total    int64 `json:"total"`
}

// ToMs converts Timing to milliseconds.
func (t *Timing) ToMs() TimingMs {
	return TimingMs{
		Discover: t.Discover.Milliseconds(),
		Offer:    t.Offer.Milliseconds(),
		Request:  t.Request.Milliseconds(),
		Total:    t.Total.Milliseconds(),
	}
}

// Transaction represents an in-progress DHCP transaction.
type Transaction struct {
	XID          uint32
	Started      time.Time
	DiscoverTime time.Time
	OfferTime    time.Time
	RequestTime  time.Time
	AckTime      time.Time
	Complete     bool
}

// Monitor watches for DHCP transactions and records timing.
type Monitor struct {
	mu           sync.RWMutex
	running      bool
	interfaceName string
	lastTiming   *Timing
	transactions map[uint32]*Transaction
}

// NewMonitor creates a new DHCP monitor.
func NewMonitor(interfaceName string) *Monitor {
	return &Monitor{
		interfaceName: interfaceName,
		transactions:  make(map[uint32]*Transaction),
	}
}

// Start begins monitoring for DHCP packets.
// Note: Requires root/CAP_NET_RAW for packet capture.
func (m *Monitor) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return nil
	}

	// TODO: Implement actual packet capture using gopacket/pcap
	// For now, we just mark as running but don't capture
	// This requires root access and the gopacket library
	m.running = true

	return nil
}

// Stop stops monitoring.
func (m *Monitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.running = false
}

// IsRunning returns whether the monitor is active.
func (m *Monitor) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// SetInterface changes the monitored interface.
func (m *Monitor) SetInterface(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.interfaceName = name
	return nil
}

// GetLastTiming returns the most recent complete DHCP transaction timing.
func (m *Monitor) GetLastTiming() *Timing {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastTiming
}

// RecordPhase records a DHCP phase timestamp (used by packet capture).
func (m *Monitor) RecordPhase(xid uint32, phase Phase, timestamp time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tx, exists := m.transactions[xid]
	if !exists {
		tx = &Transaction{
			XID:     xid,
			Started: timestamp,
		}
		m.transactions[xid] = tx
	}

	switch phase {
	case PhaseDiscover:
		tx.DiscoverTime = timestamp
	case PhaseOffer:
		tx.OfferTime = timestamp
	case PhaseRequest:
		tx.RequestTime = timestamp
	case PhaseAck:
		tx.AckTime = timestamp
		tx.Complete = true
		m.calculateTiming(tx)
	}
}

// calculateTiming computes the timing from a completed transaction.
func (m *Monitor) calculateTiming(tx *Transaction) {
	if !tx.Complete {
		return
	}

	timing := &Timing{
		Complete: true,
	}

	// Calculate phase durations
	if !tx.OfferTime.IsZero() && !tx.DiscoverTime.IsZero() {
		timing.Discover = tx.OfferTime.Sub(tx.DiscoverTime)
	}
	if !tx.RequestTime.IsZero() && !tx.OfferTime.IsZero() {
		timing.Offer = tx.RequestTime.Sub(tx.OfferTime)
	}
	if !tx.AckTime.IsZero() && !tx.RequestTime.IsZero() {
		timing.Request = tx.AckTime.Sub(tx.RequestTime)
	}

	// Total time
	if !tx.AckTime.IsZero() && !tx.DiscoverTime.IsZero() {
		timing.Total = tx.AckTime.Sub(tx.DiscoverTime)
	}

	m.lastTiming = timing

	// Cleanup old transaction
	delete(m.transactions, tx.XID)
}

// SimulateTiming creates simulated timing data for testing.
// This is useful when packet capture isn't available.
func SimulateTiming() *Timing {
	return &Timing{
		Discover: 50 * time.Millisecond,
		Offer:    10 * time.Millisecond,
		Request:  45 * time.Millisecond,
		Total:    105 * time.Millisecond,
		Complete: true,
	}
}
