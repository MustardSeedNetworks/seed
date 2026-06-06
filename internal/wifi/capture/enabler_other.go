//go:build !linux

package wificapture

// DefaultEnabler returns a no-op enabler on non-Linux platforms, where automatic
// monitor-mode switching is not implemented. Capture there is bring-your-own
// monitor: point SEED_WIFI_MONITOR_IFACE at an already-monitor interface.
func DefaultEnabler() Enabler { return nopEnabler{} }
