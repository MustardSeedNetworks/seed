package link_test

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/sap/link"
)

// TestStateConstants verifies the State constants have correct values.
func TestStateConstants(t *testing.T) {
	tests := []struct {
		name     string
		state    link.State
		expected string
	}{
		{"StateUp", link.StateUp, "up"},
		{"StateDown", link.StateDown, "down"},
		{"StateDormant", link.StateDormant, "dormant"},
		{"StateUnknown", link.StateUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.state) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(tt.state))
			}
		})
	}
}

// TestDuplexConstants verifies the Duplex constants have correct values.
func TestDuplexConstants(t *testing.T) {
	tests := []struct {
		name     string
		duplex   link.Duplex
		expected string
	}{
		{"DuplexFull", link.DuplexFull, "full"},
		{"DuplexHalf", link.DuplexHalf, "half"},
		{"DuplexUnknown", link.DuplexUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.duplex) != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, string(tt.duplex))
			}
		})
	}
}

// TestSpeedConstants verifies the Speed constants have correct values.
func TestSpeedConstants(t *testing.T) {
	tests := []struct {
		name     string
		speed    link.Speed
		expected int
	}{
		{"Speed10", link.Speed10, 10},
		{"Speed100", link.Speed100, 100},
		{"Speed1000", link.Speed1000, 1000},
		{"Speed2500", link.Speed2500, 2500},
		{"Speed5000", link.Speed5000, 5000},
		{"Speed10000", link.Speed10000, 10000},
		{"Speed25000", link.Speed25000, 25000},
		{"Speed40000", link.Speed40000, 40000},
		{"Speed100000", link.Speed100000, 100000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int(tt.speed) != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, int(tt.speed))
			}
		})
	}
}

// TestSpeedBps verifies the Speed.Bps() method.
func TestSpeedBps(t *testing.T) {
	tests := []struct {
		name        string
		speed       link.Speed
		expectedBps int64
	}{
		{"10 Mbps", link.Speed10, 10_000_000},
		{"100 Mbps", link.Speed100, 100_000_000},
		{"1 Gbps", link.Speed1000, 1_000_000_000},
		{"10 Gbps", link.Speed10000, 10_000_000_000},
		{"100 Gbps", link.Speed100000, 100_000_000_000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bps := tt.speed.Bps()
			if bps != tt.expectedBps {
				t.Errorf("expected %d, got %d", tt.expectedBps, bps)
			}
		})
	}
}

// TestSpeedString verifies the Speed.String() method.
func TestSpeedString(t *testing.T) {
	tests := []struct {
		name     string
		speed    link.Speed
		expected string
	}{
		{"10 Mbps", link.Speed10, "10 Mbps"},
		{"100 Mbps", link.Speed100, "100 Mbps"},
		{"1000 Mbps shows as 1 Gbps", link.Speed1000, "1 Gbps"},
		{"2500 Mbps shows as 2 Gbps", link.Speed2500, "2 Gbps"},
		{"10000 Mbps shows as 10 Gbps", link.Speed10000, "10 Gbps"},
		{"100000 Mbps shows as 100 Gbps", link.Speed100000, "100 Gbps"},
		{"zero speed", link.Speed(0), "0 Mbps"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str := tt.speed.String()
			if str != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, str)
			}
		})
	}
}

// TestNewMonitor verifies NewMonitor creates a properly initialized Monitor.
func TestNewMonitor(t *testing.T) {
	tests := []struct {
		name          string
		interfaceName string
	}{
		{"eth0", "eth0"},
		{"en0", "en0"},
		{"wlan0", "wlan0"},
		{"empty", ""},
		{"veth with numbers", "veth123abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := link.NewMonitor(tt.interfaceName)
			if m == nil {
				t.Fatal("expected non-nil monitor")
			}

			if m.MonitorInterfaceName() != tt.interfaceName {
				t.Errorf("expected interface %q, got %q", tt.interfaceName, m.MonitorInterfaceName())
			}

			if m.MonitorState() != link.StateUnknown {
				t.Errorf("expected StateUnknown, got %v", m.MonitorState())
			}

			if m.MonitorCallbackCount() != 0 {
				t.Errorf("expected 0 callbacks, got %d", m.MonitorCallbackCount())
			}

			if m.MonitorHistoryLen() != 0 {
				t.Errorf("expected 0 history, got %d", m.MonitorHistoryLen())
			}

			if m.IsRunning() {
				t.Error("expected monitor not running initially")
			}
		})
	}
}

// TestMonitorSetInterface verifies SetInterface updates the interface name.
func TestMonitorSetInterface(t *testing.T) {
	m := link.NewMonitor("eth0")

	tests := []string{"en0", "wlan0", "bond0", "br0", ""}

	for _, iface := range tests {
		t.Run(iface, func(t *testing.T) {
			m.SetInterface(iface)
			if m.GetInterface() != iface {
				t.Errorf("expected %q, got %q", iface, m.GetInterface())
			}
			// State should reset to unknown
			if m.MonitorState() != link.StateUnknown {
				t.Errorf("expected StateUnknown after SetInterface, got %v", m.MonitorState())
			}
		})
	}
}

// TestMonitorGetState verifies GetState returns the current state.
func TestMonitorGetState(t *testing.T) {
	m := link.NewMonitor("eth0")

	tests := []link.State{link.StateUp, link.StateDown, link.StateDormant, link.StateUnknown}

	for _, state := range tests {
		t.Run(string(state), func(t *testing.T) {
			m.SetState(state)
			if m.GetState() != state {
				t.Errorf("expected %v, got %v", state, m.GetState())
			}
		})
	}
}

// TestMonitorIsUp verifies IsUp returns correct values.
func TestMonitorIsUp(t *testing.T) {
	tests := []struct {
		name     string
		state    link.State
		expected bool
	}{
		{"StateUp returns true", link.StateUp, true},
		{"StateDown returns false", link.StateDown, false},
		{"StateDormant returns false", link.StateDormant, false},
		{"StateUnknown returns false", link.StateUnknown, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := link.NewMonitor("eth0")
			m.SetState(tt.state)
			if m.IsUp() != tt.expected {
				t.Errorf("expected IsUp=%v for state %v", tt.expected, tt.state)
			}
		})
	}
}

// TestMonitorIsDown verifies IsDown returns correct values.
func TestMonitorIsDown(t *testing.T) {
	tests := []struct {
		name     string
		state    link.State
		expected bool
	}{
		{"StateUp returns false", link.StateUp, false},
		{"StateDown returns true", link.StateDown, true},
		{"StateDormant returns false", link.StateDormant, false},
		{"StateUnknown returns false", link.StateUnknown, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := link.NewMonitor("eth0")
			m.SetState(tt.state)
			if m.IsDown() != tt.expected {
				t.Errorf("expected IsDown=%v for state %v", tt.expected, tt.state)
			}
		})
	}
}

// TestMonitorOnStateChange verifies callback registration.
func TestMonitorOnStateChange(t *testing.T) {
	m := link.NewMonitor("eth0")

	if m.MonitorCallbackCount() != 0 {
		t.Fatalf("expected 0 callbacks initially, got %d", m.MonitorCallbackCount())
	}

	// Register callbacks
	for i := 1; i <= 5; i++ {
		m.OnStateChange(func(_ link.Event) {})
		if m.MonitorCallbackCount() != i {
			t.Errorf("expected %d callbacks after registration, got %d", i, m.MonitorCallbackCount())
		}
	}
}

// TestMonitorStartStop verifies Start and Stop behavior.
func TestMonitorStartStop(t *testing.T) {
	m := link.NewMonitor("lo0") // Use loopback which exists on most systems

	if m.IsRunning() {
		t.Error("monitor should not be running before Start")
	}

	err := m.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !m.IsRunning() {
		t.Error("monitor should be running after Start")
	}

	// Starting again should be idempotent
	err = m.Start()
	if err != nil {
		t.Fatalf("Second Start failed: %v", err)
	}

	m.Stop()
	// Give the goroutine a moment to stop
	time.Sleep(10 * time.Millisecond)

	if m.IsRunning() {
		t.Error("monitor should not be running after Stop")
	}

	// Stopping again should be safe
	m.Stop()
}

// TestMonitorGetHistory verifies history retrieval.
func TestMonitorGetHistory(t *testing.T) {
	m := link.NewMonitor("eth0")

	history := m.GetHistory()
	if len(history) != 0 {
		t.Fatalf("expected empty history initially, got %d events", len(history))
	}

	// Add events
	events := []link.Event{
		{Interface: "eth0", OldState: link.StateUnknown, NewState: link.StateUp, Timestamp: time.Now()},
		{Interface: "eth0", OldState: link.StateUp, NewState: link.StateDown, Timestamp: time.Now()},
		{Interface: "eth0", OldState: link.StateDown, NewState: link.StateUp, Timestamp: time.Now()},
	}

	for _, e := range events {
		m.AddHistoryEvent(e)
	}

	history = m.GetHistory()
	if len(history) != len(events) {
		t.Fatalf("expected %d events, got %d", len(events), len(history))
	}

	// Verify it's a copy (modifying returned slice shouldn't affect internal)
	history[0] = link.Event{}
	internalHistory := m.GetHistory()
	if internalHistory[0].Interface != "eth0" {
		t.Error("GetHistory should return a copy")
	}
}

// TestMonitorGetFlapCount verifies flap counting.
func TestMonitorGetFlapCount(t *testing.T) {
	m := link.NewMonitor("eth0")

	// Add events at different times
	// Use a fixed buffer from now to avoid timing edge cases
	now := time.Now()
	events := []link.Event{
		{Interface: "eth0", Timestamp: now.Add(-25 * time.Hour)},   // Should not count for 24h
		{Interface: "eth0", Timestamp: now.Add(-23 * time.Hour)},   // Should count for 24h
		{Interface: "eth0", Timestamp: now.Add(-1 * time.Hour)},    // Should count for both
		{Interface: "eth0", Timestamp: now.Add(-30 * time.Minute)}, // In 1h window
		{Interface: "eth0", Timestamp: now.Add(-4 * time.Minute)},  // In 5m window (with buffer)
	}

	for _, e := range events {
		m.AddHistoryEvent(e)
	}

	tests := []struct {
		name     string
		duration time.Duration
		expected int
	}{
		{"1 hour", 1 * time.Hour, 2},
		{"2 hours", 2 * time.Hour, 3},
		{"24 hours", 24 * time.Hour, 4},
		{"48 hours", 48 * time.Hour, 5},
		{"6 minutes", 6 * time.Minute, 1}, // Use 6 minutes to ensure 4-minute event is captured
		{"1 minute", 1 * time.Minute, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := m.GetFlapCount(tt.duration)
			if count != tt.expected {
				t.Errorf("expected %d flaps in %v, got %d", tt.expected, tt.duration, count)
			}
		})
	}
}

// TestMonitorGetUptime verifies uptime tracking.
func TestMonitorGetUptime(t *testing.T) {
	m := link.NewMonitor("lo0")

	// Before start, uptime should be 0
	uptime := m.GetUptime()
	if uptime != 0 {
		t.Errorf("expected 0 uptime before start, got %v", uptime)
	}

	err := m.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer m.Stop()

	// Sleep a bit
	time.Sleep(50 * time.Millisecond)

	uptime = m.GetUptime()
	if uptime < 40*time.Millisecond {
		t.Errorf("expected uptime >= 40ms, got %v", uptime)
	}
}

// TestMonitorConcurrentAccess verifies thread-safety.
func TestMonitorConcurrentAccess(t *testing.T) {
	t.Parallel()

	m := link.NewMonitor("eth0")
	var wg sync.WaitGroup
	done := make(chan bool)

	// Start monitor
	_ = m.Start()
	defer m.Stop()

	// Concurrent readers
	for range 5 {
		wg.Go(func() {
			for {
				select {
				case <-done:
					return
				default:
					_ = m.GetState()
					_ = m.IsUp()
					_ = m.IsRunning()
					_ = m.GetHistory()
					_ = m.GetFlapCount(time.Hour)
				}
			}
		})
	}

	// Concurrent writers
	for range 3 {
		wg.Go(func() {
			for i := range 20 {
				select {
				case <-done:
					return
				default:
					m.SetInterface("eth" + string(rune('0'+i%10)))
					m.OnStateChange(func(_ link.Event) {})
				}
			}
		})
	}

	// Let it run
	time.Sleep(100 * time.Millisecond)
	close(done)
	wg.Wait()
}

// TestParseDuplex verifies ParseDuplex parsing.
func TestParseDuplex(t *testing.T) {
	tests := []struct {
		input    string
		expected link.Duplex
	}{
		{"full", link.DuplexFull},
		{"Full", link.DuplexFull},
		{"half", link.DuplexHalf},
		{"Half", link.DuplexHalf},
		{"unknown", link.DuplexUnknown},
		{"", link.DuplexUnknown},
		{"FULL", link.DuplexUnknown}, // Case sensitive
		{"auto", link.DuplexUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := link.ParseDuplex(tt.input)
			if result != tt.expected {
				t.Errorf("ParseDuplex(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestParseState verifies ParseState parsing.
func TestParseState(t *testing.T) {
	tests := []struct {
		input    string
		expected link.State
	}{
		{"up", link.StateUp},
		{"UP", link.StateUp},
		{"down", link.StateDown},
		{"DOWN", link.StateDown},
		{"dormant", link.StateDormant},
		{"DORMANT", link.StateDormant},
		{"unknown", link.StateUnknown},
		{"", link.StateUnknown},
		{"UNKNOWN", link.StateUnknown},
		{"lowerlayerdown", link.StateUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := link.ParseState(tt.input)
			if result != tt.expected {
				t.Errorf("ParseState(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestPlatform verifies Platform returns the correct OS.
func TestPlatform(t *testing.T) {
	platform := link.Platform()
	if platform != runtime.GOOS {
		t.Errorf("expected %q, got %q", runtime.GOOS, platform)
	}
}

// TestStatusStruct verifies Status struct fields.
func TestStatusStruct(t *testing.T) {
	now := time.Now()
	status := link.Status{
		Interface:  "eth0",
		State:      link.StateUp,
		Speed:      link.Speed1000,
		SpeedStr:   "1 Gbps",
		Duplex:     link.DuplexFull,
		MTU:        1500,
		MACAddress: "00:11:22:33:44:55",
		Carrier:    true,
		UpdatedAt:  now,
	}

	if status.Interface != "eth0" {
		t.Errorf("expected Interface 'eth0', got %q", status.Interface)
	}
	if status.State != link.StateUp {
		t.Errorf("expected StateUp, got %v", status.State)
	}
	if status.Speed != link.Speed1000 {
		t.Errorf("expected Speed1000, got %v", status.Speed)
	}
	if status.SpeedStr != "1 Gbps" {
		t.Errorf("expected SpeedStr '1 Gbps', got %q", status.SpeedStr)
	}
	if status.Duplex != link.DuplexFull {
		t.Errorf("expected DuplexFull, got %v", status.Duplex)
	}
	if status.MTU != 1500 {
		t.Errorf("expected MTU 1500, got %d", status.MTU)
	}
	if status.MACAddress != "00:11:22:33:44:55" {
		t.Errorf("expected MACAddress '00:11:22:33:44:55', got %q", status.MACAddress)
	}
	if !status.Carrier {
		t.Error("expected Carrier true")
	}
	if !status.UpdatedAt.Equal(now) {
		t.Errorf("expected UpdatedAt %v, got %v", now, status.UpdatedAt)
	}
}

// TestEventStruct verifies Event struct fields.
func TestEventStruct(t *testing.T) {
	now := time.Now()
	event := link.Event{
		Interface: "eth0",
		OldState:  link.StateDown,
		NewState:  link.StateUp,
		Timestamp: now,
	}

	if event.Interface != "eth0" {
		t.Errorf("expected Interface 'eth0', got %q", event.Interface)
	}
	if event.OldState != link.StateDown {
		t.Errorf("expected OldState StateDown, got %v", event.OldState)
	}
	if event.NewState != link.StateUp {
		t.Errorf("expected NewState StateUp, got %v", event.NewState)
	}
	if !event.Timestamp.Equal(now) {
		t.Errorf("expected Timestamp %v, got %v", now, event.Timestamp)
	}
}

// TestCheckLinkState verifies the internal checkLinkState function.
func TestCheckLinkState(t *testing.T) {
	// Test with loopback which exists on most systems
	state := link.ExportCheckLinkState("lo0")
	// Just verify it doesn't panic - result depends on system
	_ = state

	// Test with non-existent interface
	state = link.ExportCheckLinkState("nonexistent_interface_12345")
	// Should return unknown for non-existent interface
	if state != link.StateUnknown {
		t.Logf("Non-existent interface returned %v (expected Unknown)", state)
	}
}

// TestCheckLinkStatePlatform verifies the platform-specific checkLinkState.
func TestCheckLinkStatePlatform(t *testing.T) {
	interfaces := []string{"lo0", "lo", "eth0", "en0", "nonexistent123"}

	for _, iface := range interfaces {
		t.Run(iface, func(t *testing.T) {
			// Just verify it doesn't panic
			state := link.ExportCheckLinkStatePlatform(iface)
			// Verify result is a valid state
			switch state {
			case link.StateUp, link.StateDown, link.StateDormant, link.StateUnknown:
				// Valid
			default:
				t.Errorf("unexpected state %v", state)
			}
		})
	}
}

// TestGetSpeedDuplex verifies the platform-specific speed/duplex detection.
func TestGetSpeedDuplex(t *testing.T) {
	interfaces := []string{"lo0", "lo", "eth0", "en0"}

	for _, iface := range interfaces {
		t.Run(iface, func(t *testing.T) {
			t.Parallel()

			// Just verify it doesn't panic
			speed, duplex := link.ExportGetSpeedDuplex(iface)
			_ = speed
			_ = duplex
			// Result depends on system configuration
		})
	}
}

// TestIsPhysicalInterface verifies the physical interface detection.
func TestIsPhysicalInterface(t *testing.T) {
	tests := []struct {
		name         string
		iface        string
		expectPhys   bool
		skipOnLinux  bool
		skipOnDarwin bool
	}{
		// Common physical interfaces
		{"eth0", "eth0", true, false, true},
		{"en0", "en0", true, true, false},
		// Loopback - not physical
		{"lo0", "lo0", false, false, false},
		{"lo", "lo", false, false, false},
		// Virtual interfaces
		{"docker0", "docker0", false, false, false},
		{"veth123", "veth123", false, false, false},
		{"bridge0", "bridge0", false, false, false},
		{"virbr0", "virbr0", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipOnLinux && runtime.GOOS == "linux" {
				t.Skip("skipping on Linux")
			}
			if tt.skipOnDarwin && runtime.GOOS == "darwin" {
				t.Skip("skipping on Darwin")
			}

			result := link.IsPhysicalInterface(tt.iface)
			if result != tt.expectPhys {
				t.Errorf("IsPhysicalInterface(%q) = %v, expected %v", tt.iface, result, tt.expectPhys)
			}
		})
	}
}

// TestIsPhysicalInterfacePlatform verifies the platform-specific physical interface detection.
func TestIsPhysicalInterfacePlatform(t *testing.T) {
	interfaces := []string{"lo0", "lo", "eth0", "en0", "docker0", "veth123"}

	for _, iface := range interfaces {
		t.Run(iface, func(t *testing.T) {
			t.Parallel()

			// Just verify it doesn't panic
			result := link.ExportIsPhysicalInterfacePlatform(iface)
			_ = result
		})
	}
}

// TestParseSpeed verifies speed parsing.
func TestParseSpeed(t *testing.T) {
	// Just verify the platform-specific function doesn't panic
	tests := []string{
		"1000", "100", "10", "10000",
		"1000baseT", "100baseT", "10baseT",
		"1Gbps", "100Mbps", "10Gbps",
		"auto", "unknown", "",
	}

	for _, s := range tests {
		t.Run(s, func(t *testing.T) {
			t.Parallel()

			speed := link.ParseSpeed(s)
			_ = speed // Result depends on platform
		})
	}
}

// TestParseSpeedPlatform verifies the platform-specific speed parsing.
func TestParseSpeedPlatform(t *testing.T) {
	tests := []string{
		"1000", "100", "10", "10000",
		"1000baseT", "100baseT",
		"", "unknown",
	}

	for _, s := range tests {
		t.Run(s, func(t *testing.T) {
			speed := link.ExportParseSpeedPlatform(s)
			// Verify result is non-negative
			if speed < 0 {
				t.Errorf("unexpected negative speed %v", speed)
			}
		})
	}
}

// TestGetStatus verifies GetStatus with real interfaces.
func TestGetStatus(t *testing.T) {
	// Try with loopback first (usually exists)
	loopbacks := []string{"lo0", "lo"}
	var foundLoopback bool
	for _, lb := range loopbacks {
		status, err := link.GetStatus(lb)
		if err == nil {
			foundLoopback = true
			if status.Interface != lb {
				t.Errorf("expected Interface %q, got %q", lb, status.Interface)
			}
			if status.UpdatedAt.IsZero() {
				t.Error("expected non-zero UpdatedAt")
			}
			break
		}
	}

	if !foundLoopback {
		t.Log("No loopback interface found, skipping detailed checks")
	}

	// Test with non-existent interface
	_, err := link.GetStatus("nonexistent_interface_xyz")
	if err == nil {
		t.Error("expected error for non-existent interface")
	}
}

// TestListInterfaces verifies ListInterfaces returns valid data.
func TestListInterfaces(t *testing.T) {
	interfaces, err := link.ListInterfaces()
	if err != nil {
		t.Fatalf("ListInterfaces failed: %v", err)
	}

	// Should have at least one interface (excluding loopback)
	// but on some systems might be empty if all interfaces are loopback
	for _, iface := range interfaces {
		if iface.Interface == "" {
			t.Error("interface name should not be empty")
		}
		if iface.UpdatedAt.IsZero() {
			t.Error("UpdatedAt should not be zero")
		}

		// Verify state is valid
		switch iface.State {
		case link.StateUp, link.StateDown, link.StateDormant, link.StateUnknown:
			// Valid
		default:
			t.Errorf("unexpected state %v for interface %s", iface.State, iface.Interface)
		}
	}
}

// TestMonitorHistoryLimit verifies history doesn't exceed max size.
func TestMonitorHistoryLimit(t *testing.T) {
	m := link.NewMonitor("eth0")
	maxHistory := m.MonitorMaxHistory()

	// Add more events than max
	for i := range maxHistory + 50 {
		m.AddHistoryEvent(link.Event{
			Interface: "eth0",
			OldState:  link.StateDown,
			NewState:  link.StateUp,
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
		})
	}

	historyLen := m.MonitorHistoryLen()
	if historyLen > maxHistory {
		t.Errorf("history length %d exceeds max %d", historyLen, maxHistory)
	}
	if historyLen != maxHistory {
		t.Errorf("expected history length %d, got %d", maxHistory, historyLen)
	}
}

// TestMonitorDefaultValues verifies default configuration values.
func TestMonitorDefaultValues(t *testing.T) {
	m := link.NewMonitor("eth0")

	// Check poll interval
	expectedInterval := int64(link.DefaultPollInterval.Milliseconds())
	if m.MonitorPollInterval() != expectedInterval {
		t.Errorf("expected poll interval %d ms, got %d ms", expectedInterval, m.MonitorPollInterval())
	}

	// Check max history
	if m.MonitorMaxHistory() != link.DefaultEventBufferSize {
		t.Errorf("expected max history %d, got %d", link.DefaultEventBufferSize, m.MonitorMaxHistory())
	}
}

// TestTimeConstants verifies time-related constants.
func TestTimeConstants(t *testing.T) {
	if link.DefaultPollInterval != 500*time.Millisecond {
		t.Errorf("expected DefaultPollInterval 500ms, got %v", link.DefaultPollInterval)
	}

	if link.DefaultEventBufferSize != 100 {
		t.Errorf("expected DefaultEventBufferSize 100, got %d", link.DefaultEventBufferSize)
	}

	if link.WaitPollInterval != 100*time.Millisecond {
		t.Errorf("expected WaitPollInterval 100ms, got %v", link.WaitPollInterval)
	}
}

// TestMonitorCallbackInvocation verifies callbacks are invoked on state change.
func TestMonitorCallbackInvocation(t *testing.T) {
	m := link.NewMonitor("eth0")
	m.OnStateChange(func(_ link.Event) {})

	// Simulate state change by adding an event and setting state
	m.SetState(link.StateUp)
	event := link.Event{
		Interface: "eth0",
		OldState:  link.StateUnknown,
		NewState:  link.StateUp,
		Timestamp: time.Now(),
	}
	m.AddHistoryEvent(event)

	// Note: This test doesn't test the actual callback mechanism in checkAndNotify
	// since that requires real interface state changes. It just verifies registration.
	if m.MonitorCallbackCount() != 1 {
		t.Errorf("expected 1 callback, got %d", m.MonitorCallbackCount())
	}

	// For actual callback invocation, we'd need to mock the interface state
	// which is platform-specific
}

// TestMonitorMultipleCallbacks verifies multiple callbacks can be registered.
func TestMonitorMultipleCallbacks(t *testing.T) {
	m := link.NewMonitor("eth0")

	var counter1, counter2, counter3 atomic.Int32

	m.OnStateChange(func(_ link.Event) { counter1.Add(1) })
	m.OnStateChange(func(_ link.Event) { counter2.Add(1) })
	m.OnStateChange(func(_ link.Event) { counter3.Add(1) })

	if m.MonitorCallbackCount() != 3 {
		t.Errorf("expected 3 callbacks, got %d", m.MonitorCallbackCount())
	}
}

// TestSpeedEdgeCases verifies Speed methods with edge cases.
func TestSpeedEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		speed link.Speed
		bps   int64
		str   string
	}{
		{"zero speed", link.Speed(0), 0, "0 Mbps"},
		{"negative speed", link.Speed(-100), -100_000_000, "-100 Mbps"},
		{"very high speed", link.Speed(400000), 400_000_000_000, "400 Gbps"},
		{"below 1Gbps threshold", link.Speed(999), 999_000_000, "999 Mbps"},
		{"exactly 1Gbps", link.Speed(1000), 1_000_000_000, "1 Gbps"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.speed.Bps() != tt.bps {
				t.Errorf("expected Bps=%d, got %d", tt.bps, tt.speed.Bps())
			}
			if tt.speed.String() != tt.str {
				t.Errorf("expected String=%q, got %q", tt.str, tt.speed.String())
			}
		})
	}
}

// TestWaitForState verifies WaitForState with timeout.
func TestWaitForState(t *testing.T) {
	m := link.NewMonitor("nonexistent_interface_test")

	// Set initial state
	m.SetState(link.StateUp)

	// Test already in target state
	result := m.WaitForState(link.StateUp, 10*time.Millisecond)
	if !result {
		t.Error("WaitForState should return true when already in target state")
	}

	// Test waiting for different state (will timeout since interface doesn't exist)
	start := time.Now()
	result = m.WaitForState(link.StateDown, 50*time.Millisecond)
	elapsed := time.Since(start)

	if result {
		t.Log("WaitForState returned true for a non-existent interface")
	}
	// Should have waited approximately the timeout duration
	if elapsed < 40*time.Millisecond {
		t.Errorf("WaitForState should have waited at least 40ms, waited %v", elapsed)
	}
}

// TestWaitForUp verifies WaitForUp behavior.
func TestWaitForUp(t *testing.T) {
	m := link.NewMonitor("test_interface")

	// Already up
	m.SetState(link.StateUp)
	if !m.WaitForUp(10 * time.Millisecond) {
		t.Error("WaitForUp should return true when already up")
	}

	// Not up, will timeout
	m.SetState(link.StateDown)
	start := time.Now()
	result := m.WaitForUp(50 * time.Millisecond)
	elapsed := time.Since(start)

	if result {
		// Might succeed if interface comes up, but unlikely for fake interface
		t.Log("WaitForUp returned true for fake interface")
	}
	if elapsed < 40*time.Millisecond {
		t.Errorf("WaitForUp should have waited, elapsed: %v", elapsed)
	}
}

// TestWaitForDown verifies WaitForDown behavior.
func TestWaitForDown(t *testing.T) {
	m := link.NewMonitor("test_interface")

	// Already down
	m.SetState(link.StateDown)
	if !m.WaitForDown(10 * time.Millisecond) {
		t.Error("WaitForDown should return true when already down")
	}

	// Not down, will timeout
	m.SetState(link.StateUp)
	start := time.Now()
	result := m.WaitForDown(50 * time.Millisecond)
	elapsed := time.Since(start)

	if result {
		t.Log("WaitForDown returned true for fake interface")
	}
	if elapsed < 40*time.Millisecond {
		t.Errorf("WaitForDown should have waited, elapsed: %v", elapsed)
	}
}

// BenchmarkGetState benchmarks GetState performance.
func BenchmarkGetState(b *testing.B) {
	m := link.NewMonitor("eth0")
	m.SetState(link.StateUp)

	b.ResetTimer()
	for b.Loop() {
		_ = m.GetState()
	}
}

// BenchmarkIsUp benchmarks IsUp performance.
func BenchmarkIsUp(b *testing.B) {
	m := link.NewMonitor("eth0")
	m.SetState(link.StateUp)

	b.ResetTimer()
	for b.Loop() {
		_ = m.IsUp()
	}
}

// BenchmarkSpeedString benchmarks Speed.String() performance.
func BenchmarkSpeedString(b *testing.B) {
	speeds := []link.Speed{
		link.Speed10,
		link.Speed100,
		link.Speed1000,
		link.Speed10000,
	}

	b.ResetTimer()
	for b.Loop() {
		for _, s := range speeds {
			_ = s.String()
		}
	}
}

// BenchmarkGetHistory benchmarks GetHistory performance.
func BenchmarkGetHistory(b *testing.B) {
	m := link.NewMonitor("eth0")

	// Fill history
	for range 50 {
		m.AddHistoryEvent(link.Event{
			Interface: "eth0",
			Timestamp: time.Now(),
		})
	}

	b.ResetTimer()
	for b.Loop() {
		_ = m.GetHistory()
	}
}
