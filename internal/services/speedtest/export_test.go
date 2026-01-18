package speedtest

import (
	"fmt"
	"time"

	"github.com/showwin/speedtest-go/speedtest"
)

// TesterServerID returns the server ID for testing.
func (t *Tester) TesterServerID() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.serverID
}

// SetStatus is exported for testing.
func (t *Tester) SetStatus(phase string, progress float64) {
	t.setStatus(phase, progress)
}

// SetRunning is exported for testing.
func (t *Tester) SetRunning(running bool) {
	t.setRunning(running)
}

// SetCurrentSpeeds is exported for testing.
func (t *Tester) SetCurrentSpeeds(download, upload float64) {
	t.setCurrentSpeeds(download, upload)
}

// SetLastResult is exported for testing.
func (t *Tester) SetLastResult(result *Result) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lastResult = result
}

// Exported constants for testing.
const (
	ProgressFindingServer   = progressFindingServer
	ProgressTestingLatency  = progressTestingLatency
	ProgressTestingDownload = progressTestingDownload
	ProgressTestingUpload   = progressTestingUpload
	ProgressComplete        = progressComplete
	BytesToMbps             = bytesToMbps
	SpeedPollInterval       = speedPollInterval
	IdleResetDelay          = idleResetDelay
)

// BuildTestResultFromParams builds a Result from explicit parameters for testing.
// This allows testing the result building logic without needing a real speedtest.Server.
func (t *Tester) BuildTestResultFromParams(
	dlSpeed, ulSpeed float64,
	latency time.Duration,
	name, sponsor, country, host string,
	distance float64,
	startTime time.Time,
) *Result {
	return &Result{
		Download:     dlSpeed,
		Upload:       ulSpeed,
		Latency:      float64(latency.Milliseconds()),
		Server:       name,
		Location:     fmt.Sprintf("%s, %s", sponsor, country),
		Host:         host,
		Distance:     distance,
		Timestamp:    time.Now(),
		TestDuration: time.Since(startTime).Seconds(),
	}
}

// BuildTestResult is exported for testing the buildTestResult method directly.
func (t *Tester) BuildTestResult(server *speedtest.Server, startTime time.Time) *Result {
	return t.buildTestResult(server, startTime)
}

// FindTestServer is exported for integration testing.
func (t *Tester) FindTestServer() (*speedtest.Server, error) {
	return t.findTestServer()
}

// RunDownloadTest is exported for testing with a real server.
func (t *Tester) RunDownloadTest(server *speedtest.Server) error {
	return t.runDownloadTest(server)
}

// RunUploadTest is exported for testing with a real server.
func (t *Tester) RunUploadTest(server *speedtest.Server, finalDownload float64) error {
	return t.runUploadTest(server, finalDownload)
}
