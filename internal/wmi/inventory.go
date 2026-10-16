package wmi

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

// The queries an inventory is read with (ADR-0034 §4). Installed software is
// never read from Win32_Product: enumerating it makes Windows Installer check,
// and possibly repair, every MSI package on the host.
const (
	queryOS = "SELECT Caption, Version, BuildNumber, OSArchitecture, InstallDate, LastBootUpTime" +
		" FROM Win32_OperatingSystem"
	querySystem     = "SELECT Manufacturer, Model, Domain, TotalPhysicalMemory FROM Win32_ComputerSystem"
	queryFirmware   = "SELECT SerialNumber, SMBIOSBIOSVersion FROM Win32_BIOS"
	queryProcessors = "SELECT Name, NumberOfCores, NumberOfLogicalProcessors FROM Win32_Processor"
	queryDisks      = "SELECT DeviceID, Size, FreeSpace FROM Win32_LogicalDisk WHERE DriveType = 3"
	queryAdapters   = "SELECT Description, MACAddress, IPAddress" +
		" FROM Win32_NetworkAdapterConfiguration WHERE IPEnabled = TRUE"
	querySoftware = "SELECT Name, Version, Vendor FROM Win32_InstalledWin32Program"
)

// Inventory is what a Windows host reports about itself: its operating
// system, hardware and installed software.
type Inventory struct {
	OS         OperatingSystem  `json:"os"`
	System     ComputerSystem   `json:"system"`
	Firmware   Firmware         `json:"firmware"`
	Processors []Processor      `json:"processors"`
	Disks      []Disk           `json:"disks"`
	Adapters   []NetworkAdapter `json:"adapters"`
	Software   []Program        `json:"software"`
}

// OperatingSystem is Win32_OperatingSystem.
type OperatingSystem struct {
	Caption        string    `json:"caption"`
	Version        string    `json:"version"`
	BuildNumber    string    `json:"buildNumber"`
	Architecture   string    `json:"architecture"`
	InstallDate    time.Time `json:"installDate,omitzero"`
	LastBootUpTime time.Time `json:"lastBootUpTime,omitzero"`
}

// ComputerSystem is Win32_ComputerSystem.
type ComputerSystem struct {
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
	Domain       string `json:"domain"`
	MemoryBytes  uint64 `json:"memoryBytes"`
}

// Firmware is Win32_BIOS.
type Firmware struct {
	SerialNumber string `json:"serialNumber"`
	Version      string `json:"version"`
}

// Processor is one Win32_Processor: a socket.
type Processor struct {
	Name         string `json:"name"`
	Cores        int    `json:"cores"`
	LogicalCores int    `json:"logicalCores"`
}

// Disk is one fixed Win32_LogicalDisk: a drive letter.
type Disk struct {
	DeviceID  string `json:"deviceId"`
	SizeBytes uint64 `json:"sizeBytes"`
	FreeBytes uint64 `json:"freeBytes"`
}

// NetworkAdapter is one Win32_NetworkAdapterConfiguration with IP enabled.
type NetworkAdapter struct {
	Description string   `json:"description"`
	MACAddress  string   `json:"macAddress"`
	IPAddresses []string `json:"ipAddresses"`
}

// Program is one Win32_InstalledWin32Program.
type Program struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Vendor  string `json:"vendor"`
}

// Inventory reads the host's inventory, one query per class.
func (c *Client) Inventory(ctx context.Context) (Inventory, error) {
	var os, system, firmware, processors, disks, adapters, software []Instance
	for _, q := range []struct {
		wql string
		dst *[]Instance
	}{
		{queryOS, &os},
		{querySystem, &system},
		{queryFirmware, &firmware},
		{queryProcessors, &processors},
		{queryDisks, &disks},
		{queryAdapters, &adapters},
		{querySoftware, &software},
	} {
		got, err := c.Query(ctx, q.wql)
		if err != nil {
			return Inventory{}, err
		}
		*q.dst = got
	}

	var (
		inv Inventory
		p   parser
	)
	if len(os) > 0 {
		inv.OS = OperatingSystem{
			Caption:        os[0].Value("Caption"),
			Version:        os[0].Value("Version"),
			BuildNumber:    os[0].Value("BuildNumber"),
			Architecture:   os[0].Value("OSArchitecture"),
			InstallDate:    p.time(os[0].Value("InstallDate")),
			LastBootUpTime: p.time(os[0].Value("LastBootUpTime")),
		}
	}
	if len(system) > 0 {
		inv.System = ComputerSystem{
			Manufacturer: system[0].Value("Manufacturer"),
			Model:        system[0].Value("Model"),
			Domain:       system[0].Value("Domain"),
			MemoryBytes:  p.uint(system[0].Value("TotalPhysicalMemory")),
		}
	}
	if len(firmware) > 0 {
		inv.Firmware = Firmware{
			SerialNumber: firmware[0].Value("SerialNumber"),
			Version:      firmware[0].Value("SMBIOSBIOSVersion"),
		}
	}
	for _, i := range processors {
		inv.Processors = append(inv.Processors, Processor{
			Name:         i.Value("Name"),
			Cores:        p.int(i.Value("NumberOfCores")),
			LogicalCores: p.int(i.Value("NumberOfLogicalProcessors")),
		})
	}
	for _, i := range disks {
		inv.Disks = append(inv.Disks, Disk{
			DeviceID:  i.Value("DeviceID"),
			SizeBytes: p.uint(i.Value("Size")),
			FreeBytes: p.uint(i.Value("FreeSpace")),
		})
	}
	for _, i := range adapters {
		inv.Adapters = append(inv.Adapters, NetworkAdapter{
			Description: i.Value("Description"),
			MACAddress:  i.Value("MACAddress"),
			IPAddresses: i["IPAddress"],
		})
	}
	for _, i := range software {
		inv.Software = append(inv.Software, Program{
			Name:    i.Value("Name"),
			Version: i.Value("Version"),
			Vendor:  i.Value("Vendor"),
		})
	}
	if p.err != nil {
		return Inventory{}, p.err
	}
	return inv, nil
}

// parser converts property text and keeps the first malformed value. A null
// property arrives as "" and converts to the zero value.
type parser struct {
	err error
}

func (p *parser) uint(s string) uint64 {
	if s == "" {
		return 0
	}
	n, err := strconv.ParseUint(s, 10, 64)
	p.fail(err, s)
	return n
}

func (p *parser) int(s string) int {
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	p.fail(err, s)
	return n
}

// time parses a cim:Datetime, which WS-Management sends as an xs:dateTime.
func (p *parser) time(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	p.fail(err, s)
	return t.UTC()
}

func (p *parser) fail(err error, s string) {
	if err != nil && p.err == nil {
		p.err = fmt.Errorf("%w: malformed value %q: %w", ErrProtocol, s, err)
	}
}
