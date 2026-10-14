//go:build darwin

package dhcp

import "time"

// ExportStoreLease parses a dhcp_lease table into the result it reports, the
// lease's start and its expiry (darwin only). The bool is false when the
// table holds no lease.
func ExportStoreLease(table string) (*TestResult, time.Time, time.Time, bool) {
	lease := parseStoreLease(table)
	if lease == nil {
		return nil, time.Time{}, time.Time{}, false
	}
	result := &TestResult{}
	lease.fill(result)
	return result, lease.start, lease.expiry, true
}
