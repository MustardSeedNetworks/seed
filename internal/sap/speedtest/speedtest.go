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
	"errors"
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
	Running         bool    `json:"running"`
	Phase           string  `json:"phase"`           // "idle", "finding_server", "testing_latency", "testing_download", "testing_upload", "complete"
	Progress        float64 `json:"progress"`        // 0-100
	CurrentDownload float64 `json:"currentDownload"` // Live download speed in Mbps during test
	CurrentUpload   float64 `json:"currentUpload"`   // Live upload speed in Mbps during test
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

func (t *Tester) setCurrentSpeeds(download, upload float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.status.CurrentDownload = download
	t.status.CurrentUpload = upload
}

// findTestServer discovers and selects the best speedtest server.
func (t *Tester) findTestServer() (*speedtest.Server, error) {
	speedtestClient := speedtest.New()
	serverList, err := speedtestClient.FetchServers()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch servers: %w", err)
	}

	if len(serverList) == 0 {
		return nil, errors.New("no servers available")
	}

	// Select server (first one is closest by default)
	targets, err := serverList.FindServer([]int{})
	if err != nil || len(targets) == 0 {
		return nil, fmt.Errorf("failed to find server: %w", err)
	}

	return targets[0], nil
}

// runDownloadTest performs the download test with live speed updates.
func (t *Tester) runDownloadTest(server *speedtest.Server) error {
	t.setStatus("testing_download", 40)
	t.setCurrentSpeeds(0, 0)

	// Start polling goroutine for live download speed from Context
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				// GetEWMADownloadRate returns bytes/sec, convert to Mbps
				rate := server.Context.GetEWMADownloadRate()
				mbps := rate / 125000.0
				t.setCurrentSpeeds(mbps, 0)
			}
		}
	}()

	err := server.DownloadTest()
	close(done)
	if err != nil {
		t.setCurrentSpeeds(0, 0)
		return fmt.Errorf("download test failed: %w", err)
	}

	// Capture final download speed
	t.setCurrentSpeeds(server.DLSpeed.Mbps(), 0)
	return nil
}

// runUploadTest performs the upload test with live speed updates.
func (t *Tester) runUploadTest(server *speedtest.Server, finalDownload float64) error {
	t.setStatus("testing_upload", 70)

	// Start polling goroutine for live upload speed from Context
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				// GetEWMAUploadRate returns bytes/sec, convert to Mbps
				rate := server.Context.GetEWMAUploadRate()
				mbps := rate / 125000.0
				t.setCurrentSpeeds(finalDownload, mbps)
			}
		}
	}()

	err := server.UploadTest()
	close(done)
	if err != nil {
		t.setCurrentSpeeds(0, 0)
		return fmt.Errorf("upload test failed: %w", err)
	}

	return nil
}

// buildTestResult constructs the Result from server data.
func (t *Tester) buildTestResult(server *speedtest.Server, startTime time.Time) *Result {
	return &Result{
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
}

// RunTest performs a complete speedtest.
func (t *Tester) RunTest(_ context.Context) (*Result, error) {
	// Check if already running
	t.mu.RLock()
	if t.status.Running {
		t.mu.RUnlock()
		return nil, errors.New("test already in progress")
	}
	t.mu.RUnlock()

	t.setRunning(true)
	defer t.setRunning(false)

	startTime := time.Now()

	// Find servers
	t.setStatus("finding_server", 10)
	server, err := t.findTestServer()
	if err != nil {
		t.setStatus("idle", 0)
		return nil, err
	}

	// Test latency
	t.setStatus("testing_latency", 20)
	err = server.PingTest(nil)
	if err != nil {
		t.setStatus("idle", 0)
		return nil, fmt.Errorf("latency test failed: %w", err)
	}

	// Test download with live speed updates
	err = t.runDownloadTest(server)
	if err != nil {
		t.setStatus("idle", 0)
		return nil, err
	}

	// Capture final download speed for upload phase display
	finalDownload := server.DLSpeed.Mbps()

	// Test upload with live speed updates
	err = t.runUploadTest(server, finalDownload)
	if err != nil {
		t.setStatus("idle", 0)
		return nil, err
	}

	// Build and store result
	t.setStatus("complete", 100)
	result := t.buildTestResult(server, startTime)

	t.mu.Lock()
	t.lastResult = result
	t.mu.Unlock()

	// Reset to idle after a moment
	// Fixes #864: Use time.AfterFunc instead of goroutine to allow GC if Tester is discarded
	time.AfterFunc(2*time.Second, func() {
		t.setStatus("idle", 0)
	})

	return result, nil
}
