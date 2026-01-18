//go:build darwin

package vlan

// ExportGetVlanInfo exposes getVlanInfo for testing.
func ExportGetVlanInfo(ifname string) (string, int) {
	return getVlanInfo(ifname)
}

// ExportDetectVlanSubinterfacesPlatform exposes detectVlanSubinterfacesPlatform for testing.
func ExportDetectVlanSubinterfacesPlatform(iface string) []int {
	return detectVlanSubinterfacesPlatform(iface)
}

// ExportCreateVlanInterfacePlatform exposes createVlanInterfacePlatform for testing.
func ExportCreateVlanInterfacePlatform(parentIface string, vlanID int) error {
	return createVlanInterfacePlatform(parentIface, vlanID)
}

// ExportDeleteVlanInterfacePlatform exposes deleteVlanInterfacePlatform for testing.
func ExportDeleteVlanInterfacePlatform(parentIface string, vlanID int) error {
	return deleteVlanInterfacePlatform(parentIface, vlanID)
}
