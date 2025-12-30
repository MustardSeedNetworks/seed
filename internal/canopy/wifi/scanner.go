package wifi

import (
	"sort"
	"sync"
	"time"
)

// ScannedNetwork represents a WiFi network discovered during scanning.
type ScannedNetwork struct {
	SSID         string    `json:"ssid"`
	BSSID        string    `json:"bssid"`
	Signal       int       `json:"signal"` // dBm
	Channel      int       `json:"channel"`
	Frequency    int       `json:"frequency"` // MHz
	Security     string    `json:"security"`
	ChannelWidth int       `json:"channelWidth"` // 20, 40, 80, 160, 320 MHz
	NoiseFloor   int       `json:"noiseFloor"`   // dBm (typically -90 to -100)
	SNR          int       `json:"snr"`          // Signal-to-Noise Ratio (Signal - NoiseFloor)
	HTMode       string    `json:"htMode"`       // HT20, HT40, VHT80, HE160, EHT320, etc.
	IsDFS        bool      `json:"isDFS"`        // true if channel is DFS (Dynamic Frequency Selection)
	LastSeen     time.Time `json:"lastSeen"`
}

// Scanner scans for available WiFi networks.
type Scanner struct {
	interfaceName string
	mu            sync.RWMutex
	lastScan      time.Time
	networks      map[string]*ScannedNetwork // key is BSSID
}

// NewScanner creates a new WiFi scanner.
func NewScanner(interfaceName string) *Scanner {
	return &Scanner{
		interfaceName: interfaceName,
		networks:      make(map[string]*ScannedNetwork),
	}
}

// SetInterface updates the interface to scan.
func (s *Scanner) SetInterface(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.interfaceName = name
}

// Scan performs a WiFi network scan.
// Returns all networks discovered, sorted by signal strength (strongest first).
func (s *Scanner) Scan() ([]*ScannedNetwork, error) {
	s.mu.Lock()
	iface := s.interfaceName
	s.mu.Unlock()

	// Perform platform-specific scan
	networks, err := scanPlatform(iface)
	if err != nil {
		return nil, err
	}

	// Update cached networks
	s.mu.Lock()
	s.lastScan = time.Now()
	s.networks = make(map[string]*ScannedNetwork)
	for _, network := range networks {
		network.LastSeen = s.lastScan
		s.networks[network.BSSID] = network
	}
	s.mu.Unlock()

	// Sort by signal strength (strongest first)
	sort.Slice(networks, func(i, j int) bool {
		return networks[i].Signal > networks[j].Signal
	})

	return networks, nil
}

// GetCachedNetworks returns the networks from the last scan.
func (s *Scanner) GetCachedNetworks() []*ScannedNetwork {
	s.mu.RLock()
	defer s.mu.RUnlock()

	networks := make([]*ScannedNetwork, 0, len(s.networks))
	for _, network := range s.networks {
		networks = append(networks, network)
	}

	// Sort by signal strength
	sort.Slice(networks, func(i, j int) bool {
		return networks[i].Signal > networks[j].Signal
	})

	return networks
}

// GetLastScanTime returns the timestamp of the last scan.
func (s *Scanner) GetLastScanTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastScan
}
