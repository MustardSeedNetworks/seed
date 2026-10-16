package wmi_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/wmi"
)

func TestInventoryReadsEveryClass(t *testing.T) {
	h := startHost(t, false)
	c := wmi.NewClient(h.endpoint(h.fingerprint()))
	defer c.Close()

	inv, err := c.Inventory(t.Context())
	require.NoError(t, err)

	require.Equal(t, wmi.OperatingSystem{
		Caption:        "Microsoft Windows 11 Pro",
		Version:        "10.0.26100",
		BuildNumber:    "26100",
		Architecture:   "64-bit",
		InstallDate:    time.Date(2025, 3, 1, 18, 15, 30, 0, time.UTC),
		LastBootUpTime: time.Date(2026, 10, 14, 14, 2, 11, 0, time.UTC),
	}, inv.OS)
	require.Equal(t, wmi.ComputerSystem{
		Manufacturer: "QEMU", Model: "Standard PC (Q35 + ICH9, 2009)", Domain: "WORKGROUP", MemoryBytes: 8588746752,
	}, inv.System)
	require.Equal(t, wmi.Firmware{SerialNumber: "VM-0001-LAB", Version: "4.2025.05-1"}, inv.Firmware)
	require.Equal(t, []wmi.Processor{{Name: "AMD EPYC 7302P 16-Core Processor", Cores: 4, LogicalCores: 4}},
		inv.Processors)
	require.Equal(t, []wmi.Disk{{DeviceID: "C:", SizeBytes: 85348298752, FreeBytes: 41275392000}}, inv.Disks)
	require.Equal(t, []wmi.NetworkAdapter{{
		Description: "Intel(R) Ethernet Connection I219-LM",
		MACAddress:  "00:15:5D:1E:2A:3B",
		IPAddresses: []string{"10.44.30.70", "fe80::1c2b:3a4d:5e6f:7081"},
	}}, inv.Adapters)
	require.Len(t, inv.Software, 3, "the software enumeration is pulled to its end")
	require.Equal(t, wmi.Program{
		Name: "Microsoft OneDrive", Version: "26.178.0913.0007", Vendor: "Microsoft Corporation",
	}, inv.Software[1])

	for _, body := range h.bodies {
		require.NotContains(t, body, "Win32_Product",
			"enumerating Win32_Product makes Windows Installer check every package")
	}
}

func TestInventoryRefusesAMalformedNumber(t *testing.T) {
	h := startHost(t, false)
	h.malformedDisks = true
	c := wmi.NewClient(h.endpoint(h.fingerprint()))
	defer c.Close()

	_, err := c.Inventory(t.Context())
	require.ErrorIs(t, err, wmi.ErrProtocol)
	require.ErrorContains(t, err, `"lots"`)
}

func TestFingerprintMatchesPowerShell(t *testing.T) {
	// $cert.GetCertHashString('SHA256') is uppercase hex with no separators.
	got := wmi.Fingerprint([]byte("certificate"))
	require.Len(t, got, 64)
	require.Equal(t, strings.ToUpper(got), got)
}
