//go:build darwin

//nolint:testpackage // export_test files need to be in the same package to access unexported functions
package vlan

// ExportGetVlanInfo exposes getVlanInfo for testing.
func ExportGetVlanInfo(ifname string) (string, int) {
	return getVlanInfo(ifname)
}
