package dhcp_test

import (
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/dhcp"
)

// TestMonitorRecordPhaseComprehensive tests RecordPhase with comprehensive scenarios.
func TestMonitorRecordPhaseComprehensive(t *testing.T) {
	tests := []struct {
		name   string
		phases []struct {
			phase  dhcp.Phase
			offset time.Duration
		}
		expectComplete bool
		expectTiming   bool
	}{
		{
			name: "full transaction DORA",
			phases: []struct {
				phase  dhcp.Phase
				offset time.Duration
			}{
				{dhcp.PhaseDiscover, 0},
				{dhcp.PhaseOffer, 25 * time.Millisecond},
				{dhcp.PhaseRequest, 30 * time.Millisecond},
				{dhcp.PhaseAck, 100 * time.Millisecond},
			},
			expectComplete: true,
			expectTiming:   true,
		},
		{
			name: "discover only",
			phases: []struct {
				phase  dhcp.Phase
				offset time.Duration
			}{
				{dhcp.PhaseDiscover, 0},
			},
			expectComplete: false,
			expectTiming:   false,
		},
		{
			name: "discover and offer only",
			phases: []struct {
				phase  dhcp.Phase
				offset time.Duration
			}{
				{dhcp.PhaseDiscover, 0},
				{dhcp.PhaseOffer, 25 * time.Millisecond},
			},
			expectComplete: false,
			expectTiming:   false,
		},
		{
			name: "discover, offer, request only",
			phases: []struct {
				phase  dhcp.Phase
				offset time.Duration
			}{
				{dhcp.PhaseDiscover, 0},
				{dhcp.PhaseOffer, 25 * time.Millisecond},
				{dhcp.PhaseRequest, 30 * time.Millisecond},
			},
			expectComplete: false,
			expectTiming:   false,
		},
		{
			name: "only ack (incomplete)",
			phases: []struct {
				phase  dhcp.Phase
				offset time.Duration
			}{
				{dhcp.PhaseAck, 100 * time.Millisecond},
			},
			expectComplete: true, // Ack marks complete regardless
			expectTiming:   true, // But timing will be calculated
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor := dhcp.NewMonitor("eth0")
			now := time.Now()
			xid := uint32(0x12345678)

			for _, p := range tt.phases {
				monitor.RecordPhase(xid, p.phase, now.Add(p.offset))
			}

			timing := monitor.GetLastTiming()
			if tt.expectTiming {
				if timing == nil {
					t.Error("expected timing to be set")
				} else if timing.Complete != tt.expectComplete {
					t.Errorf("timing.Complete = %v, want %v", timing.Complete, tt.expectComplete)
				}
			} else {
				if timing != nil {
					t.Error("expected timing to be nil")
				}
			}
		})
	}
}

// TestMonitorCalculateTimingComprehensive tests calculateTiming with various scenarios.
func TestMonitorCalculateTimingComprehensive(t *testing.T) {
	tests := []struct {
		name             string
		tx               *dhcp.Transaction
		expectedDiscover time.Duration
		expectedOffer    time.Duration
		expectedRequest  time.Duration
		expectedTotal    time.Duration
	}{
		{
			name: "full timing",
			tx: func() *dhcp.Transaction {
				now := time.Now()
				return &dhcp.Transaction{
					XID:          1,
					DiscoverTime: now,
					OfferTime:    now.Add(25 * time.Millisecond),
					RequestTime:  now.Add(30 * time.Millisecond),
					AckTime:      now.Add(100 * time.Millisecond),
					Complete:     true,
				}
			}(),
			expectedDiscover: 25 * time.Millisecond,
			expectedOffer:    5 * time.Millisecond,
			expectedRequest:  70 * time.Millisecond,
			expectedTotal:    100 * time.Millisecond,
		},
		{
			name: "missing offer time",
			tx: func() *dhcp.Transaction {
				now := time.Now()
				return &dhcp.Transaction{
					XID:          2,
					DiscoverTime: now,
					RequestTime:  now.Add(30 * time.Millisecond),
					AckTime:      now.Add(100 * time.Millisecond),
					Complete:     true,
				}
			}(),
			expectedDiscover: 0, // No offer time means discover duration is 0
			expectedOffer:    0, // No offer time
			expectedRequest:  70 * time.Millisecond,
			expectedTotal:    100 * time.Millisecond,
		},
		{
			name: "missing request time",
			tx: func() *dhcp.Transaction {
				now := time.Now()
				return &dhcp.Transaction{
					XID:          3,
					DiscoverTime: now,
					OfferTime:    now.Add(25 * time.Millisecond),
					AckTime:      now.Add(100 * time.Millisecond),
					Complete:     true,
				}
			}(),
			expectedDiscover: 25 * time.Millisecond,
			expectedOffer:    0, // No request time
			expectedRequest:  0, // No request time
			expectedTotal:    100 * time.Millisecond,
		},
		{
			name: "only discover and ack",
			tx: func() *dhcp.Transaction {
				now := time.Now()
				return &dhcp.Transaction{
					XID:          4,
					DiscoverTime: now,
					AckTime:      now.Add(100 * time.Millisecond),
					Complete:     true,
				}
			}(),
			expectedDiscover: 0,
			expectedOffer:    0,
			expectedRequest:  0,
			expectedTotal:    100 * time.Millisecond,
		},
		{
			name: "all zero times",
			tx: &dhcp.Transaction{
				XID:      5,
				Complete: true,
			},
			expectedDiscover: 0,
			expectedOffer:    0,
			expectedRequest:  0,
			expectedTotal:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor := dhcp.NewMonitor("eth0")
			monitor.AddTransaction(tt.tx)
			monitor.CalculateTiming(tt.tx)

			timing := monitor.GetLastTiming()
			if timing == nil {
				t.Fatal("expected non-nil timing")
			}

			if timing.Discover != tt.expectedDiscover {
				t.Errorf("Discover = %v, want %v", timing.Discover, tt.expectedDiscover)
			}
			if timing.Offer != tt.expectedOffer {
				t.Errorf("Offer = %v, want %v", timing.Offer, tt.expectedOffer)
			}
			if timing.Request != tt.expectedRequest {
				t.Errorf("Request = %v, want %v", timing.Request, tt.expectedRequest)
			}
			if timing.Total != tt.expectedTotal {
				t.Errorf("Total = %v, want %v", timing.Total, tt.expectedTotal)
			}
		})
	}
}

// TestMonitorMultipleTransactions tests handling multiple concurrent transactions.
func TestMonitorMultipleTransactions(t *testing.T) {
	monitor := dhcp.NewMonitor("eth0")
	now := time.Now()

	// Record phases for multiple transactions
	xids := []uint32{0x11111111, 0x22222222, 0x33333333}

	for i, xid := range xids {
		offset := time.Duration(i*10) * time.Millisecond
		monitor.RecordPhase(xid, dhcp.PhaseDiscover, now.Add(offset))
	}

	// Verify all transactions exist
	for _, xid := range xids {
		if !monitor.TransactionExists(xid) {
			t.Errorf("transaction %x should exist", xid)
		}
	}

	// Complete only the first transaction
	monitor.RecordPhase(xids[0], dhcp.PhaseOffer, now.Add(25*time.Millisecond))
	monitor.RecordPhase(xids[0], dhcp.PhaseRequest, now.Add(30*time.Millisecond))
	monitor.RecordPhase(xids[0], dhcp.PhaseAck, now.Add(100*time.Millisecond))

	// First transaction should be removed (completed)
	if monitor.TransactionExists(xids[0]) {
		t.Error("completed transaction should be removed")
	}

	// Other transactions should still exist
	for _, xid := range xids[1:] {
		if !monitor.TransactionExists(xid) {
			t.Errorf("transaction %x should still exist", xid)
		}
	}

	// Timing should be set
	timing := monitor.GetLastTiming()
	if timing == nil {
		t.Error("expected timing to be set")
	}
}

// TestMonitorTransactionManagement tests transaction management functions.
func TestMonitorTransactionManagement(t *testing.T) {
	monitor := dhcp.NewMonitor("eth0")

	t.Run("add and get transaction", func(t *testing.T) {
		now := time.Now()
		tx := &dhcp.Transaction{
			XID:          0xDEADBEEF,
			Started:      now,
			DiscoverTime: now,
			Complete:     false,
		}

		monitor.AddTransaction(tx)

		retrieved, exists := monitor.GetTransaction(tx.XID)
		if !exists {
			t.Fatal("transaction should exist")
		}
		if retrieved.XID != tx.XID {
			t.Errorf("XID = %x, want %x", retrieved.XID, tx.XID)
		}
	})

	t.Run("transaction exists", func(t *testing.T) {
		if !monitor.TransactionExists(0xDEADBEEF) {
			t.Error("transaction should exist")
		}
		if monitor.TransactionExists(0x12345678) {
			t.Error("non-existent transaction should not exist")
		}
	})

	t.Run("monitor transactions map", func(t *testing.T) {
		txs := monitor.MonitorTransactions()
		if txs == nil {
			t.Fatal("transactions map should not be nil")
		}
		if len(txs) == 0 {
			t.Error("transactions map should not be empty")
		}
	})
}

// TestIsDHCPPortComprehensive tests isDHCPPort with all relevant ports.
func TestIsDHCPPortComprehensive(t *testing.T) {
	tests := []struct {
		name     string
		port     uint16
		expected bool
	}{
		// DHCP ports
		{"DHCP server port 67", 67, true},
		{"DHCP client port 68", 68, true},

		// Adjacent ports (should be false)
		{"port 66", 66, false},
		{"port 69 (TFTP)", 69, false},

		// Common ports
		{"HTTP port 80", 80, false},
		{"HTTPS port 443", 443, false},
		{"DNS port 53", 53, false},
		{"SSH port 22", 22, false},
		{"FTP port 21", 21, false},
		{"SMTP port 25", 25, false},

		// Edge cases
		{"port 0", 0, false},
		{"port 1", 1, false},
		{"port 65535", 65535, false},

		// BOOTP ports (same as DHCP)
		{"BOOTP server", 67, true},
		{"BOOTP client", 68, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dhcp.IsDHCPPort(tt.port)
			if result != tt.expected {
				t.Errorf("IsDHCPPort(%d) = %v, want %v", tt.port, result, tt.expected)
			}
		})
	}
}

// TestMsgTypeToPhaseComprehensive tests msgTypeToPhase with all message types.
func TestMsgTypeToPhaseComprehensive(t *testing.T) {
	tests := []struct {
		name          string
		msgType       byte
		expectedPhase dhcp.Phase
		expectedOk    bool
	}{
		// Standard DORA message types
		{"DHCPDISCOVER (1)", 1, dhcp.PhaseDiscover, true},
		{"DHCPOFFER (2)", 2, dhcp.PhaseOffer, true},
		{"DHCPREQUEST (3)", 3, dhcp.PhaseRequest, true},
		{"DHCPDECLINE (4)", 4, "", false},
		{"DHCPACK (5)", 5, dhcp.PhaseAck, true},
		{"DHCPNAK (6)", 6, "", false},
		{"DHCPRELEASE (7)", 7, "", false},
		{"DHCPINFORM (8)", 8, "", false},

		// Invalid/undefined message types
		{"type 0 (invalid)", 0, "", false},
		{"type 9 (undefined)", 9, "", false},
		{"type 10 (undefined)", 10, "", false},
		{"type 255 (invalid)", 255, "", false},

		// DHCPv4 extension types (typically not used)
		{"DHCPFORCERENEW (9)", 9, "", false},
		{"DHCPLEASEQUERY (10)", 10, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			phase, ok := dhcp.MsgTypeToPhase(tt.msgType)
			if ok != tt.expectedOk {
				t.Errorf("MsgTypeToPhase(%d) ok = %v, want %v", tt.msgType, ok, tt.expectedOk)
			}
			if phase != tt.expectedPhase {
				t.Errorf("MsgTypeToPhase(%d) phase = %q, want %q", tt.msgType, phase, tt.expectedPhase)
			}
		})
	}
}

// TestFindDHCPMessageTypeComprehensive tests findDHCPMessageType with various option formats.
func TestFindDHCPMessageTypeComprehensive(t *testing.T) {
	tests := []struct {
		name     string
		options  []byte
		expected byte
	}{
		// Standard message types
		{
			name:     "DISCOVER message",
			options:  []byte{53, 1, 1, 255},
			expected: 1,
		},
		{
			name:     "OFFER message",
			options:  []byte{53, 1, 2, 255},
			expected: 2,
		},
		{
			name:     "REQUEST message",
			options:  []byte{53, 1, 3, 255},
			expected: 3,
		},
		{
			name:     "ACK message",
			options:  []byte{53, 1, 5, 255},
			expected: 5,
		},
		{
			name:     "NAK message",
			options:  []byte{53, 1, 6, 255},
			expected: 6,
		},

		// With preceding options
		{
			name: "message type after subnet mask",
			options: []byte{
				1, 4, 255, 255, 255, 0, // Subnet mask
				53, 1, 1, // DISCOVER
				255,
			},
			expected: 1,
		},
		{
			name: "message type after multiple options",
			options: []byte{
				1, 4, 255, 255, 255, 0, // Subnet mask
				3, 4, 192, 168, 1, 1, // Router
				6, 4, 8, 8, 8, 8, // DNS
				53, 1, 3, // REQUEST
				255,
			},
			expected: 3,
		},

		// With pad options
		{
			name: "with pad options before",
			options: []byte{
				0, 0, 0, // Pad options
				53, 1, 2, // OFFER
				255,
			},
			expected: 2,
		},
		{
			name: "with pad options interspersed",
			options: []byte{
				1, 4, 255, 255, 255, 0, // Subnet mask
				0, 0, // Pad
				53, 1, 5, // ACK
				0, // Pad
				255,
			},
			expected: 5,
		},

		// Edge cases
		{
			name:     "empty options",
			options:  []byte{},
			expected: 0,
		},
		{
			name:     "only end option",
			options:  []byte{255},
			expected: 0,
		},
		{
			name:     "only pad options then end",
			options:  []byte{0, 0, 0, 255},
			expected: 0,
		},
		{
			name:     "truncated at option type",
			options:  []byte{53},
			expected: 0,
		},
		{
			name:     "truncated at option length",
			options:  []byte{53, 1},
			expected: 0,
		},
		{
			name:     "length exceeds remaining",
			options:  []byte{53, 10, 1, 2},
			expected: 0,
		},
		{
			name:     "zero length message type",
			options:  []byte{53, 0, 255},
			expected: 0,
		},
		{
			name:     "no message type option present",
			options:  []byte{1, 4, 255, 255, 255, 0, 255},
			expected: 0,
		},

		// Message type with additional data (length > 1)
		{
			name:     "message type with longer data",
			options:  []byte{53, 3, 1, 0, 0, 255}, // Unusual but valid per spec
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dhcp.FindDHCPMessageType(tt.options)
			if result != tt.expected {
				t.Errorf("FindDHCPMessageType() = %d, want %d", result, tt.expected)
			}
		})
	}
}

// TestSimulateTimingValues tests the simulated timing values.
func TestSimulateTimingValues(t *testing.T) {
	timing := dhcp.SimulateTiming()

	if timing == nil {
		t.Fatal("expected non-nil timing")
	}
	if !timing.Complete {
		t.Error("simulated timing should be complete")
	}

	// Check expected values (from constants)
	if timing.Discover != 50*time.Millisecond {
		t.Errorf("Discover = %v, want 50ms", timing.Discover)
	}
	if timing.Offer != 10*time.Millisecond {
		t.Errorf("Offer = %v, want 10ms", timing.Offer)
	}
	if timing.Request != 45*time.Millisecond {
		t.Errorf("Request = %v, want 45ms", timing.Request)
	}
	if timing.Total != 105*time.Millisecond {
		t.Errorf("Total = %v, want 105ms", timing.Total)
	}
}

// TestTimingToMsComprehensive tests ToMs conversion with various timing values.
func TestTimingToMsComprehensive(t *testing.T) {
	tests := []struct {
		name     string
		timing   *dhcp.Timing
		expected dhcp.TimingMs
	}{
		{
			name: "standard timing",
			timing: &dhcp.Timing{
				Discover: 50 * time.Millisecond,
				Offer:    10 * time.Millisecond,
				Request:  45 * time.Millisecond,
				Total:    105 * time.Millisecond,
				Complete: true,
			},
			expected: dhcp.TimingMs{
				Discover: 50,
				Offer:    10,
				Request:  45,
				Total:    105,
			},
		},
		{
			name: "zero timing",
			timing: &dhcp.Timing{
				Complete: true,
			},
			expected: dhcp.TimingMs{
				Discover: 0,
				Offer:    0,
				Request:  0,
				Total:    0,
			},
		},
		{
			name: "microsecond precision loss",
			timing: &dhcp.Timing{
				Discover: 50500 * time.Microsecond,  // 50.5ms
				Total:    100500 * time.Microsecond, // 100.5ms
				Complete: true,
			},
			expected: dhcp.TimingMs{
				Discover: 50,  // Truncated to 50ms
				Total:    100, // Truncated to 100ms
			},
		},
		{
			name: "large values",
			timing: &dhcp.Timing{
				Discover: 1000 * time.Millisecond,
				Offer:    500 * time.Millisecond,
				Request:  2000 * time.Millisecond,
				Total:    3500 * time.Millisecond,
				Complete: true,
			},
			expected: dhcp.TimingMs{
				Discover: 1000,
				Offer:    500,
				Request:  2000,
				Total:    3500,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.timing.ToMs()

			if result.Discover != tt.expected.Discover {
				t.Errorf("Discover = %d, want %d", result.Discover, tt.expected.Discover)
			}
			if result.Offer != tt.expected.Offer {
				t.Errorf("Offer = %d, want %d", result.Offer, tt.expected.Offer)
			}
			if result.Request != tt.expected.Request {
				t.Errorf("Request = %d, want %d", result.Request, tt.expected.Request)
			}
			if result.Total != tt.expected.Total {
				t.Errorf("Total = %d, want %d", result.Total, tt.expected.Total)
			}
		})
	}
}

// TestTransactionFields tests Transaction struct fields.
func TestTransactionFieldsComprehensive(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name string
		tx   dhcp.Transaction
	}{
		{
			name: "complete transaction",
			tx: dhcp.Transaction{
				XID:          0xDEADBEEF,
				Started:      now,
				DiscoverTime: now,
				OfferTime:    now.Add(25 * time.Millisecond),
				RequestTime:  now.Add(30 * time.Millisecond),
				AckTime:      now.Add(100 * time.Millisecond),
				Complete:     true,
			},
		},
		{
			name: "incomplete transaction",
			tx: dhcp.Transaction{
				XID:          0x12345678,
				Started:      now,
				DiscoverTime: now,
				Complete:     false,
			},
		},
		{
			name: "zero XID",
			tx: dhcp.Transaction{
				XID:      0,
				Started:  now,
				Complete: false,
			},
		},
		{
			name: "max XID",
			tx: dhcp.Transaction{
				XID:      0xFFFFFFFF,
				Started:  now,
				Complete: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the struct is properly initialized
			if tt.tx.XID == 0 && tt.name != "zero XID" {
				t.Error("XID should be set")
			}
		})
	}
}

// TestMonitorConcurrentRecordPhase tests concurrent access to RecordPhase.
func TestMonitorConcurrentRecordPhase(t *testing.T) {
	monitor := dhcp.NewMonitor("eth0")
	now := time.Now()

	done := make(chan bool, 100)

	// Spawn goroutines that record phases concurrently
	for i := range 10 {
		go func(id int) {
			for j := range 100 {
				xid := uint32(id*1000 + j)
				monitor.RecordPhase(xid, dhcp.PhaseDiscover, now)
				monitor.RecordPhase(xid, dhcp.PhaseOffer, now.Add(10*time.Millisecond))
				monitor.RecordPhase(xid, dhcp.PhaseRequest, now.Add(20*time.Millisecond))
				monitor.RecordPhase(xid, dhcp.PhaseAck, now.Add(50*time.Millisecond))
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for range 10 {
		<-done
	}

	// Verify no panics occurred and last timing is set
	timing := monitor.GetLastTiming()
	if timing == nil {
		t.Error("expected timing to be set after concurrent operations")
	}
}

// TestMonitorStateTransitions tests monitor state transitions.
func TestMonitorStateTransitions(t *testing.T) {
	monitor := dhcp.NewMonitor("lo0")

	// Initial state
	if monitor.IsRunning() {
		t.Error("monitor should not be running initially")
	}

	// Stop when not running should be safe
	monitor.Stop()
	if monitor.IsRunning() {
		t.Error("monitor should not be running after Stop")
	}

	// Get timing when no transactions recorded
	if monitor.GetLastTiming() != nil {
		t.Error("timing should be nil initially")
	}

	// Set interface when not running
	err := monitor.SetInterface("en0")
	if err != nil {
		t.Errorf("SetInterface should succeed when not running: %v", err)
	}
	if monitor.MonitorInterfaceName() != "en0" {
		t.Errorf("interface should be 'en0', got %q", monitor.MonitorInterfaceName())
	}
}

// TestLeaseInfoFields tests LeaseInfo struct fields.
func TestLeaseInfoFieldsComprehensive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		info dhcp.LeaseInfo
	}{
		{
			name: "full lease info",
			info: dhcp.LeaseInfo{
				DHCPServer: "192.168.1.1",
				Gateway:    "192.168.1.1",
				LeaseTime:  86400,
				DNS:        []string{"8.8.8.8", "8.8.4.4"},
			},
		},
		{
			name: "minimal lease info",
			info: dhcp.LeaseInfo{
				DHCPServer: "10.0.0.1",
			},
		},
		{
			name: "empty lease info",
			info: dhcp.LeaseInfo{},
		},
		{
			name: "lease info with single DNS",
			info: dhcp.LeaseInfo{
				DHCPServer: "192.168.1.1",
				Gateway:    "192.168.1.1",
				LeaseTime:  3600,
				DNS:        []string{"1.1.1.1"},
			},
		},
		{
			name: "lease info with many DNS servers",
			info: dhcp.LeaseInfo{
				DNS: []string{"8.8.8.8", "8.8.4.4", "1.1.1.1", "1.0.0.1", "9.9.9.9"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Verify fields are accessible
			_ = tt.info.DHCPServer
			_ = tt.info.Gateway
			_ = tt.info.LeaseTime
			_ = tt.info.DNS
		})
	}
}
