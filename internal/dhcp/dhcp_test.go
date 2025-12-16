// Package dhcp provides DHCP transaction timing and monitoring.
// Test suite validates DHCP transaction phases, timing measurements,
// and protocol state management.
package dhcp

import (
	"testing"
	"time"
)

func TestPhaseConstants(t *testing.T) {
	if PhaseDiscover != "discover" {
		t.Errorf("expected PhaseDiscover = 'discover', got %q", PhaseDiscover)
	}
	if PhaseOffer != "offer" {
		t.Errorf("expected PhaseOffer = 'offer', got %q", PhaseOffer)
	}
	if PhaseRequest != "request" {
		t.Errorf("expected PhaseRequest = 'request', got %q", PhaseRequest)
	}
	if PhaseAck != "ack" {
		t.Errorf("expected PhaseAck = 'ack', got %q", PhaseAck)
	}
}

func TestNewMonitor(t *testing.T) {
	monitor := NewMonitor("eth0")
	if monitor == nil {
		t.Fatal("expected non-nil monitor")
	}

	if monitor.interfaceName != "eth0" {
		t.Errorf("expected interfaceName 'eth0', got %q", monitor.interfaceName)
	}
	if monitor.running {
		t.Error("expected running to be false initially")
	}
	if monitor.transactions == nil {
		t.Error("expected non-nil transactions map")
	}
}

func TestMonitorStartStop(t *testing.T) {
	monitor := NewMonitor("eth0")

	// Start monitoring
	err := monitor.Start()
	if err != nil {
		t.Errorf("unexpected error starting monitor: %v", err)
	}
	if !monitor.IsRunning() {
		t.Error("expected IsRunning() to be true after Start()")
	}

	// Start again should be no-op
	err = monitor.Start()
	if err != nil {
		t.Errorf("unexpected error on second Start(): %v", err)
	}

	// Stop monitoring
	monitor.Stop()
	if monitor.IsRunning() {
		t.Error("expected IsRunning() to be false after Stop()")
	}
}

func TestMonitorSetInterface(t *testing.T) {
	monitor := NewMonitor("eth0")

	err := monitor.SetInterface("en0")
	if err != nil {
		t.Errorf("unexpected error setting interface: %v", err)
	}
	if monitor.interfaceName != "en0" {
		t.Errorf("expected interfaceName 'en0', got %q", monitor.interfaceName)
	}
}

func TestMonitorGetLastTiming(t *testing.T) {
	monitor := NewMonitor("eth0")

	// Initially nil
	timing := monitor.GetLastTiming()
	if timing != nil {
		t.Error("expected nil timing initially")
	}
}

func TestMonitorRecordPhase(t *testing.T) {
	monitor := NewMonitor("eth0")

	now := time.Now()
	xid := uint32(12345)

	// Record all phases
	monitor.RecordPhase(xid, PhaseDiscover, now)
	monitor.RecordPhase(xid, PhaseOffer, now.Add(50*time.Millisecond))
	monitor.RecordPhase(xid, PhaseRequest, now.Add(60*time.Millisecond))
	monitor.RecordPhase(xid, PhaseAck, now.Add(105*time.Millisecond))

	// Should have timing now
	timing := monitor.GetLastTiming()
	if timing == nil {
		t.Fatal("expected non-nil timing after complete transaction")
	}
	if !timing.Complete {
		t.Error("expected Complete to be true")
	}
	if timing.Total != 105*time.Millisecond {
		t.Errorf("expected Total 105ms, got %v", timing.Total)
	}
}

func TestTimingToMs(t *testing.T) {
	timing := &Timing{
		Discover: 50 * time.Millisecond,
		Offer:    10 * time.Millisecond,
		Request:  45 * time.Millisecond,
		Total:    105 * time.Millisecond,
		Complete: true,
	}

	ms := timing.ToMs()
	if ms.Discover != 50 {
		t.Errorf("expected Discover 50, got %d", ms.Discover)
	}
	if ms.Offer != 10 {
		t.Errorf("expected Offer 10, got %d", ms.Offer)
	}
	if ms.Request != 45 {
		t.Errorf("expected Request 45, got %d", ms.Request)
	}
	if ms.Total != 105 {
		t.Errorf("expected Total 105, got %d", ms.Total)
	}
}

func TestTimingMsFields(t *testing.T) {
	ms := TimingMs{
		Discover: 50,
		Offer:    20,
		Request:  15,
		Ack:      5,
		Total:    90,
	}

	if ms.Discover != 50 {
		t.Errorf("expected Discover 50, got %d", ms.Discover)
	}
	if ms.Offer != 20 {
		t.Errorf("expected Offer 20, got %d", ms.Offer)
	}
	if ms.Request != 15 {
		t.Errorf("expected Request 15, got %d", ms.Request)
	}
	if ms.Ack != 5 {
		t.Errorf("expected Ack 5, got %d", ms.Ack)
	}
	if ms.Total != 90 {
		t.Errorf("expected Total 90, got %d", ms.Total)
	}
}

func TestTransactionFields(t *testing.T) {
	now := time.Now()
	tx := Transaction{
		XID:          12345,
		Started:      now,
		DiscoverTime: now,
		OfferTime:    now.Add(50 * time.Millisecond),
		RequestTime:  now.Add(60 * time.Millisecond),
		AckTime:      now.Add(105 * time.Millisecond),
		Complete:     true,
	}

	if tx.XID != 12345 {
		t.Errorf("expected XID 12345, got %d", tx.XID)
	}
	if !tx.Complete {
		t.Error("expected Complete to be true")
	}
}

func TestSimulateTiming(t *testing.T) {
	timing := SimulateTiming()

	if timing == nil {
		t.Fatal("expected non-nil timing")
	}
	if !timing.Complete {
		t.Error("expected Complete to be true")
	}
	if timing.Discover != 50*time.Millisecond {
		t.Errorf("expected Discover 50ms, got %v", timing.Discover)
	}
	if timing.Offer != 10*time.Millisecond {
		t.Errorf("expected Offer 10ms, got %v", timing.Offer)
	}
	if timing.Request != 45*time.Millisecond {
		t.Errorf("expected Request 45ms, got %v", timing.Request)
	}
	if timing.Total != 105*time.Millisecond {
		t.Errorf("expected Total 105ms, got %v", timing.Total)
	}
}

func TestLeaseInfoFields(t *testing.T) {
	info := LeaseInfo{
		DHCPServer: "192.168.1.1",
		Gateway:    "192.168.1.1",
		LeaseTime:  86400,
		DNS:        []string{"8.8.8.8", "8.8.4.4"},
	}

	if info.DHCPServer != "192.168.1.1" {
		t.Errorf("expected DHCPServer '192.168.1.1', got %q", info.DHCPServer)
	}
	if info.Gateway != "192.168.1.1" {
		t.Errorf("expected Gateway '192.168.1.1', got %q", info.Gateway)
	}
	if info.LeaseTime != 86400 {
		t.Errorf("expected LeaseTime 86400, got %d", info.LeaseTime)
	}
	if len(info.DNS) != 2 {
		t.Errorf("expected 2 DNS servers, got %d", len(info.DNS))
	}
}

func TestExtractValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"option with semicolon", "option dhcp-server-identifier 192.168.1.1;", "192.168.1.1"},
		{"option without semicolon", "option routers 192.168.1.1", "192.168.1.1"},
		{"empty string", "", ""},
		{"single value", "value", "value"},
		{"multiple values", "option dns 8.8.8.8 8.8.4.4", "8.8.4.4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractValue(tt.input)
			if result != tt.expected {
				t.Errorf("extractValue(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetLeaseInfo(t *testing.T) {
	// On different platforms and network configurations, this may return nil or populated LeaseInfo.
	// The test only verifies that GetLeaseInfo does not panic or crash.
	info, err := GetLeaseInfo("eth0")
	_ = info
	_ = err
}

func TestCalculateTimingIncomplete(t *testing.T) {
	monitor := NewMonitor("eth0")

	tx := &Transaction{
		XID:      12345,
		Complete: false,
	}

	// Should not set lastTiming for incomplete transaction
	monitor.calculateTiming(tx)
	if monitor.lastTiming != nil {
		t.Error("expected nil lastTiming for incomplete transaction")
	}
}

func TestCalculateTimingComplete(t *testing.T) {
	monitor := NewMonitor("eth0")

	now := time.Now()
	tx := &Transaction{
		XID:          12345,
		Started:      now,
		DiscoverTime: now,
		OfferTime:    now.Add(50 * time.Millisecond),
		RequestTime:  now.Add(60 * time.Millisecond),
		AckTime:      now.Add(105 * time.Millisecond),
		Complete:     true,
	}

	monitor.transactions[tx.XID] = tx
	monitor.calculateTiming(tx)

	if monitor.lastTiming == nil {
		t.Fatal("expected non-nil lastTiming for complete transaction")
	}
	if !monitor.lastTiming.Complete {
		t.Error("expected Complete to be true")
	}
	// Transaction should be removed after calculation
	if _, exists := monitor.transactions[tx.XID]; exists {
		t.Error("expected transaction to be removed after calculation")
	}
}

func TestConcurrentMonitorAccess(t *testing.T) {
	monitor := NewMonitor("eth0")

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				_ = monitor.IsRunning()
				_ = monitor.GetLastTiming()
				monitor.RecordPhase(uint32(id*100+j), PhaseDiscover, time.Now())
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestGetLeaseInfoDarwin(t *testing.T) {
	// This test is only meaningful on macOS; skip or ignore on other platforms.
	info, err := getLeaseInfoDarwin("en0")
	// Just verify it doesn't panic
	_ = info
	_ = err
}

func TestGetLeaseInfoLinux(t *testing.T) {
	// This test is only meaningful on Linux; on other platforms it is effectively skipped.
	info, err := getLeaseInfoLinux("eth0")
	// Just verify it doesn't panic
	_ = info
	_ = err
}

func TestParseDHClientLeaseFile(t *testing.T) {
	// Test with a path that doesn't exist
	result := parseDHClientLeaseFile("/nonexistent/path", "eth0")
	if result != nil {
		t.Error("expected nil for non-existent file")
	}
}

func TestParseNMLeaseFile(t *testing.T) {
	// Test with a path that doesn't exist
	result := parseNMLeaseFile("/nonexistent/path")
	if result != nil {
		t.Error("expected nil for non-existent file")
	}
}

func TestParseNetworkdLeaseFile(t *testing.T) {
	// Test with a path that doesn't exist
	result := parseNetworkdLeaseFile("/nonexistent/path")
	if result != nil {
		t.Error("expected nil for non-existent file")
	}
}

func TestExtractValueMoreCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"with spaces", "   option value   ", "value"},
		{"option routers", "option routers 192.168.1.1;", "192.168.1.1"},
		{"option lease-time", "option dhcp-lease-time 86400;", "86400"},
		{"complex line", "option domain-name-servers 8.8.8.8, 8.8.4.4;", "8.8.4.4"},
		{"no spaces", "value", "value"},
		{"multiple spaces", "a   b   c   d", "d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractValue(tt.input)
			if result != tt.expected {
				t.Errorf("extractValue(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCalculateTimingWithZeroTimes(t *testing.T) {
	monitor := NewMonitor("eth0")

	// Test with some zero times
	tx := &Transaction{
		XID:      12345,
		Complete: true,
	}

	monitor.transactions[tx.XID] = tx
	monitor.calculateTiming(tx)

	timing := monitor.GetLastTiming()
	if timing == nil {
		t.Fatal("expected non-nil timing")
	}
	// With zero times, durations should be 0
	if timing.Total != 0 {
		t.Errorf("expected Total 0 for zero times, got %v", timing.Total)
	}
}

// TestCalculateTimingPartialPhases verifies that the timing calculation correctly handles
// transactions with only partial phase timestamps (specifically, only Discover and Ack times).
// It checks that the total time is computed as the difference between Discover and Ack,
// and that intermediate phase durations (such as Discover) are zero when their corresponding
// timestamps are missing.
func TestCalculateTimingPartialPhases(t *testing.T) {
	monitor := NewMonitor("eth0")

	now := time.Now()
	// Only discover and ack, missing offer and request times
	tx := &Transaction{
		XID:          12346,
		DiscoverTime: now,
		AckTime:      now.Add(100 * time.Millisecond),
		Complete:     true,
	}

	monitor.transactions[tx.XID] = tx
	monitor.calculateTiming(tx)

	timing := monitor.GetLastTiming()
	if timing == nil {
		t.Fatal("expected non-nil timing")
	}
	if !timing.Complete {
		t.Error("expected Complete to be true")
	}
	if timing.Total != 100*time.Millisecond {
		t.Errorf("expected Total 100ms, got %v", timing.Total)
	}
	// Discover, Offer, Request should be 0 since intermediate times are zero
	if timing.Discover != 0 {
		t.Errorf("expected Discover 0 (no OfferTime), got %v", timing.Discover)
	}
}

func TestTimingFields(t *testing.T) {
	timing := Timing{
		Discover: 25 * time.Millisecond,
		Offer:    15 * time.Millisecond,
		Request:  30 * time.Millisecond,
		Total:    70 * time.Millisecond,
		Complete: true,
	}

	if timing.Discover != 25*time.Millisecond {
		t.Errorf("expected Discover 25ms, got %v", timing.Discover)
	}
	if timing.Offer != 15*time.Millisecond {
		t.Errorf("expected Offer 15ms, got %v", timing.Offer)
	}
	if timing.Request != 30*time.Millisecond {
		t.Errorf("expected Request 30ms, got %v", timing.Request)
	}
	if timing.Total != 70*time.Millisecond {
		t.Errorf("expected Total 70ms, got %v", timing.Total)
	}
	if !timing.Complete {
		t.Error("expected Complete true")
	}
}

func TestRecordPhaseNewTransaction(t *testing.T) {
	monitor := NewMonitor("eth0")

	now := time.Now()
	xid := uint32(99999)

	// Record first phase (Discover) - should create new transaction
	monitor.RecordPhase(xid, PhaseDiscover, now)

	// Verify transaction was created
	monitor.mu.RLock()
	tx, exists := monitor.transactions[xid]
	monitor.mu.RUnlock()

	if !exists {
		t.Fatal("expected transaction to be created")
	}
	if tx.XID != xid {
		t.Errorf("expected XID %d, got %d", xid, tx.XID)
	}
	if tx.DiscoverTime != now {
		t.Error("expected DiscoverTime to be set")
	}
}

func TestRecordPhaseAllPhases(t *testing.T) {
	monitor := NewMonitor("eth0")

	now := time.Now()
	xid := uint32(88888)

	// Record all phases
	monitor.RecordPhase(xid, PhaseDiscover, now)
	monitor.RecordPhase(xid, PhaseOffer, now.Add(10*time.Millisecond))
	monitor.RecordPhase(xid, PhaseRequest, now.Add(20*time.Millisecond))
	monitor.RecordPhase(xid, PhaseAck, now.Add(50*time.Millisecond))

	// After Ack, timing should be calculated and transaction removed
	monitor.mu.RLock()
	_, exists := monitor.transactions[xid]
	monitor.mu.RUnlock()

	if exists {
		t.Error("expected transaction to be removed after Ack")
	}

	timing := monitor.GetLastTiming()
	if timing == nil {
		t.Fatal("expected non-nil timing")
	}
	if !timing.Complete {
		t.Error("expected Complete true")
	}
}
