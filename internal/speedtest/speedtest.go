package speedtest

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/showwin/speedtest-go/speedtest"
)

// Result contains the speedtest results
type Result struct {
	Download    float64   `json:"download"`    // Mbps
	Upload      float64   `json:"upload"`      // Mbps
	Latency     float64   `json:"latency"`     // ms
	Server      string    `json:"server"`      // Server name
	Location    string    `json:"location"`    // Server location
	Host        string    `json:"host"`        // Server host
	Distance    float64   `json:"distance"`    // km
	Timestamp   time.Time `json:"timestamp"`
	TestDuration float64  `json:"testDuration"` // seconds
}

// Status represents the current test status
type Status struct {
	Running  bool    `json:"running"`
	Phase    string  `json:"phase"`    // "idle", "finding_server", "testing_latency", "testing_download", "testing_upload", "complete"
	Progress float64 `json:"progress"` // 0-100
}

// Tester handles speedtest operations
type Tester struct {
	mu          sync.RWMutex
	status      Status
	lastResult  *Result
	serverID    string // Optional: specific server ID to use
}

// NewTester creates a new speedtest tester
func NewTester() *Tester {
	return &Tester{
		status: Status{Phase: "idle"},
	}
}

// GetStatus returns the current test status
func (t *Tester) GetStatus() Status {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.status
}

// GetLastResult returns the last test result
func (t *Tester) GetLastResult() *Result {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.lastResult
}

// SetServerID sets a specific server ID to use for tests
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

// RunTest performs a complete speedtest
func (t *Tester) RunTest(ctx context.Context) (*Result, error) {
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
