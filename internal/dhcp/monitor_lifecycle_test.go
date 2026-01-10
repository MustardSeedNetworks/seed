package dhcp_test

import (
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/dhcp"
)

// TestMonitorSetInterfaceComprehensive tests SetInterface with various scenarios.
func TestMonitorSetInterfaceComprehensive(t *testing.T) {
	tests := []struct {
		name         string
		initialIface string
		targetIface  string
		expectError  bool
	}{
		{
			name:         "change from eth0 to wlan0",
			initialIface: "eth0",
			targetIface:  "wlan0",
			expectError:  false,
		},
		{
			name:         "change from wlan0 to eth0",
			initialIface: "wlan0",
			targetIface:  "eth0",
			expectError:  false,
		},
		{
			name:         "change to same interface",
			initialIface: "eth0",
			targetIface:  "eth0",
			expectError:  false,
		},
		{
			name:         "change to loopback",
			initialIface: "eth0",
			targetIface:  "lo",
			expectError:  false,
		},
		{
			name:         "change from empty to eth0",
			initialIface: "",
			targetIface:  "eth0",
			expectError:  false,
		},
		{
			name:         "change to empty string",
			initialIface: "eth0",
			targetIface:  "",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor := dhcp.NewMonitor(tt.initialIface)

			err := monitor.SetInterface(tt.targetIface)
			if (err != nil) != tt.expectError {
				t.Errorf("SetInterface() error = %v, expectError = %v", err, tt.expectError)
			}

			if !tt.expectError && monitor.MonitorInterfaceName() != tt.targetIface {
				t.Errorf("interface = %q, want %q", monitor.MonitorInterfaceName(), tt.targetIface)
			}
		})
	}
}

// TestMonitorIsRunningStates tests IsRunning in various states.
func TestMonitorIsRunningStates(t *testing.T) {
	monitor := dhcp.NewMonitor("eth0")

	// Initially not running
	if monitor.IsRunning() {
		t.Error("should not be running initially")
	}

	// Still not running after setting interface
	if err := monitor.SetInterface("wlan0"); err != nil {
		t.Errorf("SetInterface error: %v", err)
	}
	if monitor.IsRunning() {
		t.Error("should not be running after SetInterface")
	}

	// Still not running after recording phases
	monitor.RecordPhase(12345, dhcp.PhaseDiscover, time.Now())
	if monitor.IsRunning() {
		t.Error("should not be running after RecordPhase")
	}

	// Still not running after Stop when already stopped
	monitor.Stop()
	if monitor.IsRunning() {
		t.Error("should not be running after Stop")
	}
}

// TestMonitorStopMultipleTimes tests calling Stop multiple times.
func TestMonitorStopMultipleTimes(t *testing.T) {
	monitor := dhcp.NewMonitor("eth0")

	// Call Stop multiple times when not running
	for i := range 10 {
		monitor.Stop()
		if monitor.IsRunning() {
			t.Errorf("iteration %d: should not be running", i)
		}
	}
}

// TestMonitorGetLastTimingNilInitially tests GetLastTiming returns nil initially.
func TestMonitorGetLastTimingNilInitially(t *testing.T) {
	monitor := dhcp.NewMonitor("eth0")

	timing := monitor.GetLastTiming()
	if timing != nil {
		t.Error("expected nil timing initially")
	}
}

// TestMonitorGetLastTimingAfterPartialTransaction tests timing after incomplete transaction.
func TestMonitorGetLastTimingAfterPartialTransaction(t *testing.T) {
	monitor := dhcp.NewMonitor("eth0")
	now := time.Now()

	// Record only discover phase
	monitor.RecordPhase(12345, dhcp.PhaseDiscover, now)

	timing := monitor.GetLastTiming()
	if timing != nil {
		t.Error("expected nil timing after only discover phase")
	}

	// Record offer
	monitor.RecordPhase(12345, dhcp.PhaseOffer, now.Add(10*time.Millisecond))

	timing = monitor.GetLastTiming()
	if timing != nil {
		t.Error("expected nil timing after discover and offer only")
	}
}

// TestMonitorGetLastTimingAfterCompleteTransaction tests timing after complete transaction.
func TestMonitorGetLastTimingAfterCompleteTransaction(t *testing.T) {
	monitor := dhcp.NewMonitor("eth0")
	now := time.Now()

	// Complete transaction
	monitor.RecordPhase(12345, dhcp.PhaseDiscover, now)
	monitor.RecordPhase(12345, dhcp.PhaseOffer, now.Add(10*time.Millisecond))
	monitor.RecordPhase(12345, dhcp.PhaseRequest, now.Add(15*time.Millisecond))
	monitor.RecordPhase(12345, dhcp.PhaseAck, now.Add(50*time.Millisecond))

	timing := monitor.GetLastTiming()
	if timing == nil {
		t.Fatal("expected non-nil timing after complete transaction")
	}
	if !timing.Complete {
		t.Error("expected Complete = true")
	}
	if timing.Total != 50*time.Millisecond {
		t.Errorf("Total = %v, want 50ms", timing.Total)
	}
}

// TestMonitorMultipleTransactionsLastTimingUpdated tests that last timing is updated correctly.
func TestMonitorMultipleTransactionsLastTimingUpdated(t *testing.T) {
	monitor := dhcp.NewMonitor("eth0")
	now := time.Now()

	// First transaction with 50ms total
	xid1 := uint32(11111)
	monitor.RecordPhase(xid1, dhcp.PhaseDiscover, now)
	monitor.RecordPhase(xid1, dhcp.PhaseOffer, now.Add(10*time.Millisecond))
	monitor.RecordPhase(xid1, dhcp.PhaseRequest, now.Add(15*time.Millisecond))
	monitor.RecordPhase(xid1, dhcp.PhaseAck, now.Add(50*time.Millisecond))

	timing1 := monitor.GetLastTiming()
	if timing1 == nil || timing1.Total != 50*time.Millisecond {
		t.Fatalf("first transaction timing incorrect: %+v", timing1)
	}

	// Second transaction with 100ms total
	xid2 := uint32(22222)
	base := now.Add(100 * time.Millisecond)
	monitor.RecordPhase(xid2, dhcp.PhaseDiscover, base)
	monitor.RecordPhase(xid2, dhcp.PhaseOffer, base.Add(20*time.Millisecond))
	monitor.RecordPhase(xid2, dhcp.PhaseRequest, base.Add(30*time.Millisecond))
	monitor.RecordPhase(xid2, dhcp.PhaseAck, base.Add(100*time.Millisecond))

	timing2 := monitor.GetLastTiming()
	if timing2 == nil || timing2.Total != 100*time.Millisecond {
		t.Fatalf("second transaction timing should update: %+v", timing2)
	}
}

// TestMonitorRecordPhaseOutOfOrder tests recording phases in unusual order.
func TestMonitorRecordPhaseOutOfOrder(t *testing.T) {
	monitor := dhcp.NewMonitor("eth0")
	now := time.Now()
	xid := uint32(33333)

	// Record in unusual order (shouldn't happen in real DHCP, but should handle gracefully)
	monitor.RecordPhase(xid, dhcp.PhaseOffer, now.Add(10*time.Millisecond))
	monitor.RecordPhase(xid, dhcp.PhaseDiscover, now)
	monitor.RecordPhase(xid, dhcp.PhaseAck, now.Add(50*time.Millisecond))
	monitor.RecordPhase(xid, dhcp.PhaseRequest, now.Add(20*time.Millisecond))

	// Transaction should be complete (Ack was recorded)
	timing := monitor.GetLastTiming()
	if timing == nil {
		t.Fatal("expected timing after Ack")
	}
	if !timing.Complete {
		t.Error("expected Complete = true")
	}
}

// TestMonitorRecordPhaseSamePhaseMultipleTimes tests recording same phase multiple times.
func TestMonitorRecordPhaseSamePhaseMultipleTimes(t *testing.T) {
	monitor := dhcp.NewMonitor("eth0")
	now := time.Now()
	xid := uint32(44444)

	// Record Discover multiple times (shouldn't happen but handle gracefully)
	monitor.RecordPhase(xid, dhcp.PhaseDiscover, now)
	monitor.RecordPhase(xid, dhcp.PhaseDiscover, now.Add(5*time.Millisecond))
	monitor.RecordPhase(xid, dhcp.PhaseDiscover, now.Add(10*time.Millisecond))

	// Verify transaction exists
	if !monitor.TransactionExists(xid) {
		t.Error("transaction should exist")
	}

	// Complete the transaction
	monitor.RecordPhase(xid, dhcp.PhaseOffer, now.Add(20*time.Millisecond))
	monitor.RecordPhase(xid, dhcp.PhaseRequest, now.Add(25*time.Millisecond))
	monitor.RecordPhase(xid, dhcp.PhaseAck, now.Add(50*time.Millisecond))

	timing := monitor.GetLastTiming()
	if timing == nil {
		t.Fatal("expected timing")
	}
}

// TestMonitorAddTransactionDirect tests AddTransaction helper.
func TestMonitorAddTransactionDirect(t *testing.T) {
	monitor := dhcp.NewMonitor("eth0")
	now := time.Now()

	tx := &dhcp.Transaction{
		XID:          55555,
		Started:      now,
		DiscoverTime: now,
		Complete:     false,
	}

	monitor.AddTransaction(tx)

	// Verify it exists
	retrieved, exists := monitor.GetTransaction(tx.XID)
	if !exists {
		t.Fatal("transaction should exist")
	}
	if retrieved.XID != tx.XID {
		t.Errorf("XID = %d, want %d", retrieved.XID, tx.XID)
	}
}

// TestMonitorTransactionExistsVariousCases tests TransactionExists.
func TestMonitorTransactionExistsVariousCases(t *testing.T) {
	monitor := dhcp.NewMonitor("eth0")

	// Non-existent transaction
	if monitor.TransactionExists(99999) {
		t.Error("non-existent transaction should not exist")
	}

	// Add a transaction
	monitor.RecordPhase(12345, dhcp.PhaseDiscover, time.Now())

	// Should exist now
	if !monitor.TransactionExists(12345) {
		t.Error("transaction should exist after recording")
	}

	// Different XID should not exist
	if monitor.TransactionExists(12346) {
		t.Error("different XID should not exist")
	}
}

// TestMonitorMonitorTransactionsMap tests MonitorTransactions helper.
func TestMonitorMonitorTransactionsMap(t *testing.T) {
	monitor := dhcp.NewMonitor("eth0")

	// Initially empty
	txs := monitor.MonitorTransactions()
	if len(txs) != 0 {
		t.Errorf("expected empty map, got %d entries", len(txs))
	}

	// Add transactions
	now := time.Now()
	monitor.RecordPhase(11111, dhcp.PhaseDiscover, now)
	monitor.RecordPhase(22222, dhcp.PhaseDiscover, now)
	monitor.RecordPhase(33333, dhcp.PhaseDiscover, now)

	txs = monitor.MonitorTransactions()
	if len(txs) != 3 {
		t.Errorf("expected 3 transactions, got %d", len(txs))
	}
}

// TestMonitorCalculateTimingIncompleteTransaction tests calculateTiming with incomplete.
func TestMonitorCalculateTimingIncompleteTransaction(t *testing.T) {
	monitor := dhcp.NewMonitor("eth0")

	tx := &dhcp.Transaction{
		XID:      66666,
		Complete: false,
	}

	monitor.AddTransaction(tx)
	monitor.CalculateTiming(tx)

	// Should not set lastTiming for incomplete transaction
	if monitor.MonitorLastTiming() != nil {
		t.Error("expected nil timing for incomplete transaction")
	}

	// Transaction should still exist (not removed)
	if !monitor.TransactionExists(tx.XID) {
		t.Error("incomplete transaction should not be removed")
	}
}

// TestMonitorCalculateTimingRemovesTransaction tests that complete transaction is removed.
func TestMonitorCalculateTimingRemovesTransaction(t *testing.T) {
	monitor := dhcp.NewMonitor("eth0")
	now := time.Now()

	tx := &dhcp.Transaction{
		XID:          77777,
		DiscoverTime: now,
		OfferTime:    now.Add(10 * time.Millisecond),
		RequestTime:  now.Add(15 * time.Millisecond),
		AckTime:      now.Add(50 * time.Millisecond),
		Complete:     true,
	}

	monitor.AddTransaction(tx)

	// Transaction exists before calculation
	if !monitor.TransactionExists(tx.XID) {
		t.Error("transaction should exist before calculation")
	}

	monitor.CalculateTiming(tx)

	// Transaction should be removed after calculation
	if monitor.TransactionExists(tx.XID) {
		t.Error("complete transaction should be removed after calculation")
	}

	// Timing should be set
	timing := monitor.GetLastTiming()
	if timing == nil {
		t.Fatal("expected timing to be set")
	}
	if timing.Total != 50*time.Millisecond {
		t.Errorf("Total = %v, want 50ms", timing.Total)
	}
}

// TestMonitorInterfaceNameAfterCreation tests interface name accessor.
func TestMonitorInterfaceNameAfterCreation(t *testing.T) {
	interfaces := []string{"eth0", "wlan0", "en0", "lo", "br0", "docker0"}

	for _, iface := range interfaces {
		monitor := dhcp.NewMonitor(iface)
		if monitor.MonitorInterfaceName() != iface {
			t.Errorf("MonitorInterfaceName() = %q, want %q", monitor.MonitorInterfaceName(), iface)
		}
	}
}

// TestMonitorMonitorRunningAfterCreation tests running state accessor.
func TestMonitorMonitorRunningAfterCreation(t *testing.T) {
	monitor := dhcp.NewMonitor("eth0")

	if monitor.MonitorRunning() {
		t.Error("should not be running after creation")
	}
}

// TestMonitorStopWhenNotRunningLifecycle tests Stop when not running.
func TestMonitorStopWhenNotRunningLifecycle(t *testing.T) {
	monitor := dhcp.NewMonitor("eth0")

	// Should not panic when stopping non-running monitor
	monitor.Stop()

	if monitor.IsRunning() {
		t.Error("should still not be running after Stop")
	}
}

// TestPhaseConstantsValues tests all Phase constant values.
func TestPhaseConstantsValues(t *testing.T) {
	tests := []struct {
		phase    dhcp.Phase
		expected string
	}{
		{dhcp.PhaseDiscover, "discover"},
		{dhcp.PhaseOffer, "offer"},
		{dhcp.PhaseRequest, "request"},
		{dhcp.PhaseAck, "ack"},
	}

	for _, tt := range tests {
		t.Run(string(tt.phase), func(t *testing.T) {
			if string(tt.phase) != tt.expected {
				t.Errorf("Phase = %q, want %q", tt.phase, tt.expected)
			}
		})
	}
}

// TestTimingZeroValues tests Timing with all zero values.
func TestTimingZeroValues(t *testing.T) {
	timing := dhcp.Timing{}

	if timing.Discover != 0 {
		t.Errorf("Discover = %v, want 0", timing.Discover)
	}
	if timing.Offer != 0 {
		t.Errorf("Offer = %v, want 0", timing.Offer)
	}
	if timing.Request != 0 {
		t.Errorf("Request = %v, want 0", timing.Request)
	}
	if timing.Total != 0 {
		t.Errorf("Total = %v, want 0", timing.Total)
	}
	if timing.Complete {
		t.Error("Complete should be false by default")
	}

	ms := timing.ToMs()
	if ms.Discover != 0 || ms.Offer != 0 || ms.Request != 0 || ms.Total != 0 {
		t.Error("ToMs should return all zeros for zero timing")
	}
}

// TestTimingMsZeroValues tests TimingMs with all zero values.
func TestTimingMsZeroValues(t *testing.T) {
	ms := dhcp.TimingMs{}

	if ms.Discover != 0 {
		t.Errorf("Discover = %d, want 0", ms.Discover)
	}
	if ms.Offer != 0 {
		t.Errorf("Offer = %d, want 0", ms.Offer)
	}
	if ms.Request != 0 {
		t.Errorf("Request = %d, want 0", ms.Request)
	}
	if ms.Ack != 0 {
		t.Errorf("Ack = %d, want 0", ms.Ack)
	}
	if ms.Total != 0 {
		t.Errorf("Total = %d, want 0", ms.Total)
	}
}

// TestTransactionZeroValues tests Transaction with all zero values.
func TestTransactionZeroValues(t *testing.T) {
	tx := dhcp.Transaction{}

	if tx.XID != 0 {
		t.Errorf("XID = %d, want 0", tx.XID)
	}
	if !tx.Started.IsZero() {
		t.Error("Started should be zero time")
	}
	if !tx.DiscoverTime.IsZero() {
		t.Error("DiscoverTime should be zero time")
	}
	if !tx.OfferTime.IsZero() {
		t.Error("OfferTime should be zero time")
	}
	if !tx.RequestTime.IsZero() {
		t.Error("RequestTime should be zero time")
	}
	if !tx.AckTime.IsZero() {
		t.Error("AckTime should be zero time")
	}
	if tx.Complete {
		t.Error("Complete should be false by default")
	}
}

// TestLeaseInfoZeroValues tests LeaseInfo with zero values.
func TestLeaseInfoZeroValues(t *testing.T) {
	info := dhcp.LeaseInfo{}

	if info.DHCPServer != "" {
		t.Errorf("DHCPServer = %q, want empty", info.DHCPServer)
	}
	if info.Gateway != "" {
		t.Errorf("Gateway = %q, want empty", info.Gateway)
	}
	if info.LeaseTime != 0 {
		t.Errorf("LeaseTime = %d, want 0", info.LeaseTime)
	}
	if info.DNS != nil {
		t.Errorf("DNS = %v, want nil", info.DNS)
	}
}

// TestMonitorConcurrentOperations tests concurrent access to monitor.
func TestMonitorConcurrentOperations(t *testing.T) {
	t.Parallel()

	monitor := dhcp.NewMonitor("eth0")

	done := make(chan bool, 100)

	// Writers - record phases
	for i := range 10 {
		go func(id int) {
			for j := range 50 {
				xid := uint32(id*1000 + j)
				now := time.Now()
				monitor.RecordPhase(xid, dhcp.PhaseDiscover, now)
				monitor.RecordPhase(xid, dhcp.PhaseOffer, now.Add(10*time.Millisecond))
				monitor.RecordPhase(xid, dhcp.PhaseRequest, now.Add(15*time.Millisecond))
				monitor.RecordPhase(xid, dhcp.PhaseAck, now.Add(50*time.Millisecond))
			}
			done <- true
		}(i)
	}

	// Readers - read timing
	for i := range 10 {
		go func(_ int) {
			for range 50 {
				_ = monitor.GetLastTiming()
				_ = monitor.IsRunning()
				_ = monitor.MonitorInterfaceName()
			}
			done <- true
		}(i)
	}

	// Interface changers
	for i := range 5 {
		go func(id int) {
			for j := range 20 {
				iface := "eth" + string(rune('0'+id)) + string(rune('0'+j%10))
				_ = monitor.SetInterface(iface)
			}
			done <- true
		}(i)
	}

	// Wait for all
	for range 25 {
		<-done
	}

	// Verify no panics occurred
}

// TestSimulateTimingConsistency tests SimulateTiming returns consistent values.
func TestSimulateTimingConsistency(t *testing.T) {
	timing1 := dhcp.SimulateTiming()
	timing2 := dhcp.SimulateTiming()

	if timing1.Discover != timing2.Discover {
		t.Error("SimulateTiming should return consistent Discover")
	}
	if timing1.Offer != timing2.Offer {
		t.Error("SimulateTiming should return consistent Offer")
	}
	if timing1.Request != timing2.Request {
		t.Error("SimulateTiming should return consistent Request")
	}
	if timing1.Total != timing2.Total {
		t.Error("SimulateTiming should return consistent Total")
	}
	if timing1.Complete != timing2.Complete {
		t.Error("SimulateTiming should return consistent Complete")
	}
}
