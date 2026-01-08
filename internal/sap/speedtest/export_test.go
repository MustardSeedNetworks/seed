package speedtest

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
