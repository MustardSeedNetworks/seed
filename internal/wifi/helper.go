package wifi

import "github.com/MustardSeedNetworks/seed/internal/wifi/wifihelper"

// Helper is a session with a Wi-Fi helper agent.
//
// On macOS the daemon runs as root outside any login session and cannot hold
// the Location Services grant that un-redacts SSID and BSSID, so when CoreWLAN
// refuses in-process the work is delegated to a helper running in the user's
// session. No other platform needs one, and passes a nil Helper.
type Helper interface {
	Scan() ([]wifihelper.Network, error)
	Current() (wifihelper.Network, error)
	Saved() ([]string, error)
}

// SetHelper registers the helper this scanner delegates to. Passing nil clears
// it. Platforms that scan in-process ignore it.
func (s *Scanner) SetHelper(h Helper) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.helper = h
}

func (s *Scanner) currentHelper() Helper {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.helper
}

// SetHelper registers the helper this manager delegates to. Passing nil clears
// it. Platforms that read Wi-Fi state in-process ignore it.
func (m *Manager) SetHelper(h Helper) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.helper = h
}

func (m *Manager) currentHelper() Helper {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.helper
}
