//go:build linux

package netif

import (
	"fmt"

	"github.com/safchain/ethtool"
)

// DriverStats returns the interface's driver counters, raw.
//
// ethtool.Stats is the -S output: every counter the driver chooses to expose,
// which varies by driver and is why Curate matches on several spellings.
func DriverStats(name string) (map[string]uint64, error) {
	handle, err := ethtool.NewEthtool()
	if err != nil {
		return nil, fmt.Errorf("open ethtool: %w", err)
	}
	defer handle.Close()

	stats, err := handle.Stats(name)
	if err != nil {
		return nil, fmt.Errorf("read driver statistics for %s: %w", name, err)
	}

	return stats, nil
}
