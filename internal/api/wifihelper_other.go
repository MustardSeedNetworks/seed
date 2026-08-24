//go:build !darwin

package api

// wifihelper_other.go: only macOS needs a helper agent to hold Location
// Services authorization on the daemon's behalf. Every other platform scans
// in-process, so no helper is ever started here.

import "fmt"

func (s *Server) startWiFiHelper() {}

// stopWiFiHelper closes a helper session if one was somehow set. No platform
// other than macOS starts one, so this is a no-op in practice — but writing it
// in terms of the field keeps the shared Server struct honest everywhere
// rather than leaving a member that only one platform acknowledges.
func (s *Server) stopWiFiHelper() error {
	if s.wifiHelper == nil {
		return nil
	}

	if err := s.wifiHelper.Close(); err != nil {
		return fmt.Errorf("close wifi helper: %w", err)
	}
	s.wifiHelper = nil
	return nil
}
