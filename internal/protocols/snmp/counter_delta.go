package snmp

import "math"

// Counter32Delta returns how far a Counter32 advanced between two readings,
// each taken alongside the agent's sysUpTime. It reports false when no delta
// is meaningful: sysUpTime went backwards, so the agent re-initialized between
// the readings and its counters restarted (an RFC 2863 counter
// discontinuity), or a reading does not fit in 32 bits. A smaller current
// value while sysUpTime advanced is one wrap through 2^32.
func Counter32Delta(prev, cur uint64, prevUpTime, curUpTime uint32) (uint64, bool) {
	if curUpTime < prevUpTime || prev > math.MaxUint32 || cur > math.MaxUint32 {
		return 0, false
	}
	if cur >= prev {
		return cur - prev, true
	}
	return math.MaxUint32 - prev + cur + 1, true
}
