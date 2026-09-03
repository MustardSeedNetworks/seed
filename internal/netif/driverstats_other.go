//go:build !linux

package netif

// DriverStats is unavailable off Linux.
//
// Returning an error rather than an empty map on purpose: an empty map reads as
// "this NIC reports no errors", which is the most reassuring possible wrong
// answer. The caller turns this into a 501 that names the platform (#750).
func DriverStats(_ string) (map[string]uint64, error) {
	return nil, ErrDriverStatsUnsupported
}
