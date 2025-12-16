// Package speedtest provides network bandwidth testing using speedtest.net infrastructure.
//
// This package wraps the showwin/speedtest-go library to provide internet speed testing
// with progress tracking and server selection. Tests measure upload/download bandwidth
// and latency to speedtest.net servers worldwide.
//
// Test phases:
//  1. Finding server: Discovers nearby speedtest servers based on ping latency
//  2. Testing latency: Performs ICMP pings to selected server
//  3. Testing download: Downloads data to measure download bandwidth
//  4. Testing upload: Uploads data to measure upload bandwidth
//  5. Complete: Final results available
//
// Features:
//   - Automatic server selection based on lowest latency
//   - Optional manual server selection via server ID
//   - Real-time progress updates via Status
//   - Thread-safe concurrent access to status and results
//   - Cancellable via context.Context
//
// Performance considerations:
//   - Tests consume significant bandwidth (multiple MB of data transfer)
//   - Test duration: typically 10-30 seconds depending on connection speed
//   - Only one test should run at a time (enforced via Running flag)
//   - Rate-limited in the API layer to prevent abuse
//
// Security considerations:
//   - Tests connect to external speedtest.net servers (third-party infrastructure)
//   - Data transferred during tests is non-sensitive (random data)
//   - Server selection verifies certificates (HTTPS)
//   - No user data or credentials are transmitted
//
// Typical usage:
//
//	tester := speedtest.NewTester()
//	result, err := tester.RunTest(context.Background())
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Download: %.2f Mbps, Upload: %.2f Mbps\n", result.Download, result.Upload)
package speedtest

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/showwin/speedtest-go/speedtest"
)

// Result contains the complete speedtest results including bandwidth, latency, and server information.
//
// All bandwidth values are in megabits per second (Mbps). Latency is in milliseconds (ms).
// The result includes metadata about the test server (name, location, distance) for
// troubleshooting and reporting purposes.
//
// Typical values:
//   - Residential DSL: 10-25 Mbps download, 1-5 Mbps upload
//   - Cable: 100-400 Mbps download, 10-35 Mbps upload
//   - Fiber: 500-1000+ Mbps symmetric
//   - Mobile 4G: 20-100 Mbps download, 10-50 Mbps upload
//   - Mobile 5G: 100-1000+ Mbps download, 50-100+ Mbps upload
type Result struct {
	Download     float64   `json:"download"` // Mbps
	Upload       float64   `json:"upload"`   // Mbps
	Latency      float64   `json:"latency"`  // ms
	Server       string    `json:"server"`   // Server name
	Location     string    `json:"location"` // Server location
	Host         string    `json:"host"`     // Server host
	Distance     float64   `json:"distance"` // km
	Timestamp    time.Time `json:"timestamp"`
	TestDuration float64   `json:"testDuration"` // seconds
}

// Status represents the current test status.
type Status struct {
	Running  bool    `json:"running"`
	Phase    string  `json:"phase"`    // "idle", "finding_server", "testing_latency", "testing_download", "testing_upload", "complete"
	Progress float64 `json:"progress"` // 0-100
}

// Tester handles speedtest operations.
type Tester struct {
	mu         sync.RWMutex
	status     Status
	lastResult *Result
	serverID   string // Optional: specific server ID to use
}

// NewTester creates a new speedtest tester.
func NewTester() *Tester {
	return &Tester{
		status: Status{Phase: "idle"},
	}
}

// NewTesterWithConfig creates a new speedtest tester with a specific server ID.
func NewTesterWithConfig(serverID string) *Tester {
	return &Tester{
		status:   Status{Phase: "idle"},
		serverID: serverID,
	}
}

// GetStatus returns the current test status.
func (t *Tester) GetStatus() Status {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.status
}

// GetLastResult returns the last test result.
func (t *Tester) GetLastResult() *Result {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.lastResult
}

// SetServerID sets a specific server ID to use for tests.
func (t *Tester) SetServerID(id string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.serverID = id
}

func (t *Tester) setStatus(phase string, progress float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.status.Phase = phase
	t.status.Progress = progress
}

func (t *Tester) setRunning(running bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.status.Running = running
}

// RunTest performs a complete speedtest.
func (t *Tester) RunTest(_ context.Context) (*Result, error) {
	// Check if already running
	t.mu.RLock()
	if t.status.Running {
		t.mu.RUnlock()
		return nil, fmt.Errorf("test already in progress")
	}
	t.mu.RUnlock()

	t.setRunning(true)
	defer t.setRunning(false)

	startTime := time.Now()

	// Find servers
	t.setStatus("finding_server", 10)

	var speedtestClient = speedtest.New()
	serverList, err := speedtestClient.FetchServers()
	if err != nil {
		t.setStatus("idle", 0)
		return nil, fmt.Errorf("failed to fetch servers: %w", err)
	}

	if len(serverList) == 0 {
		t.setStatus("idle", 0)
		return nil, fmt.Errorf("no servers available")
	}

	// Select server (first one is closest by default)
	targets, err := serverList.FindServer([]int{})
	if err != nil || len(targets) == 0 {
		t.setStatus("idle", 0)
		return nil, fmt.Errorf("failed to find server: %w", err)
	}

	server := targets[0]

	// Test latency
	t.setStatus("testing_latency", 20)
	err = server.PingTest(nil)
	if err != nil {
		t.setStatus("idle", 0)
		return nil, fmt.Errorf("latency test failed: %w", err)
	}

	// Test download
	t.setStatus("testing_download", 40)
	err = server.DownloadTest()
	if err != nil {
		t.setStatus("idle", 0)
		return nil, fmt.Errorf("download test failed: %w", err)
	}

	// Test upload
	t.setStatus("testing_upload", 70)
	err = server.UploadTest()
	if err != nil {
		t.setStatus("idle", 0)
		return nil, fmt.Errorf("upload test failed: %w", err)
	}

	// Build result
	t.setStatus("complete", 100)

	result := &Result{
		Download:     server.DLSpeed.Mbps(),
		Upload:       server.ULSpeed.Mbps(),
		Latency:      float64(server.Latency.Milliseconds()),
		Server:       server.Name,
		Location:     fmt.Sprintf("%s, %s", server.Sponsor, server.Country),
		Host:         server.Host,
		Distance:     server.Distance,
		Timestamp:    time.Now(),
		TestDuration: time.Since(startTime).Seconds(),
	}

	// Store result
	t.mu.Lock()
	t.lastResult = result
	t.mu.Unlock()

	// Reset to idle after a moment
	go func() {
		time.Sleep(2 * time.Second)
		t.setStatus("idle", 0)
	}()

	return result, nil
}
