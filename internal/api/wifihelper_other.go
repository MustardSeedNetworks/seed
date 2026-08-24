//go:build !darwin

package api

// wifihelper_other.go: only macOS needs a helper agent to hold Location
// Services authorization on the daemon's behalf. Every other platform scans
// in-process.

func (s *Server) startWiFiHelper() {}

func (s *Server) stopWiFiHelper() error { return nil }
