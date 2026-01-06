//go:build !linux

package network

// ethtool stub implementation for non-Linux platforms provides no-op implementations
// as ethtool is Linux-specific. macOS and other platforms use alternative approaches.

// getEthtoolSettings is a stub for non-Linux platforms.
// Ethtool functionality is only available on Linux.
func getEthtoolSettings(_ string) (bool, []string) {
	return false, nil
}
