// Package speedtest exports internal functions for testing.
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
