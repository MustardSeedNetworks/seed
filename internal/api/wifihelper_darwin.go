//go:build darwin

package api

// wifihelper_darwin.go starts the local socket the macOS Wi-Fi helper agent
// connects to. macOS grants Location Services authorization — which is what
// un-redacts SSID and BSSID in CoreWLAN results — per user, to a signed
// application bundle, in a logged-in GUI session. Seed runs as root from a
// LaunchDaemon and can hold no such grant, so the helper does the scanning and
// the daemon consumes it over this socket.

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/wifi/wifihelper"
)

const (
	// wifiHelperSocketDir is created root-owned and not group- or
	// world-writable, so an unprivileged process cannot replace the socket
	// with one of its own and feed the daemon fabricated scan results.
	wifiHelperSocketDir  = "/var/run/seed"
	wifiHelperSocketName = "wifi-helper.sock"
	wifiHelperSocketPerm = 0o755

	// wifiHelperRequirement is the code-signing requirement a connecting helper
	// must satisfy. Verification uses the peer's audit token, so an
	// unprivileged process running as the same user cannot impersonate the
	// helper. A locally-built unsigned helper will not satisfy this and will be
	// refused, which is the intended failure direction.
	wifiHelperRequirement = `identifier "net.mustardseed.seed.wifihelper"` +
		` and anchor apple generic` +
		` and certificate leaf[subject.OU] = "X6JWYP43HG"`
)

// startWiFiHelper begins listening for the helper agent. A failure here is not
// fatal: seed still runs, and Wi-Fi calls report why they cannot see network
// names rather than reporting an empty airspace.
func (s *Server) startWiFiHelper() {
	if err := os.MkdirAll(wifiHelperSocketDir, wifiHelperSocketPerm); err != nil {
		logging.GetLogger().Warn("Wi-Fi helper socket directory unavailable",
			"event", "wifi.helper.unavailable", "error", err)
		return
	}

	path := filepath.Join(wifiHelperSocketDir, wifiHelperSocketName)
	srv, err := wifihelper.NewServer(path, wifiHelperRequirement, logging.GetLogger())
	if err != nil {
		logging.GetLogger().Warn("Wi-Fi helper socket unavailable",
			"event", "wifi.helper.unavailable", "error", err)
		return
	}

	s.wifiHelper = srv
	s.wifiScan.SetHelper(srv)
	s.wifiMgr.SetHelper(srv)

	logging.GetLogger().Info("Wi-Fi helper socket listening",
		"event", "wifi.helper.listening", "path", path)
}

// stopWiFiHelper releases the socket.
func (s *Server) stopWiFiHelper() error {
	if s.wifiHelper == nil {
		return nil
	}

	s.wifiScan.SetHelper(nil)
	s.wifiMgr.SetHelper(nil)
	if err := s.wifiHelper.Close(); err != nil {
		return fmt.Errorf("close wifi helper: %w", err)
	}
	s.wifiHelper = nil
	return nil
}
