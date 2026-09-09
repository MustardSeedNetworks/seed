package api

// bluetooth_wire.go holds the flat wire DTOs for a Bluetooth scan and the
// mapping from the discovery domain onto them. The only transport that carries
// them is the "bluetooth-scan" job kind (ADR-0005, jobs_bluetooth.go); the
// legacy /security/bluetooth/* REST routes were retired with the rest of the
// unconsumed route set.

import (
	"time"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
)

// BluetoothScanResponse contains Bluetooth scan results.
type BluetoothScanResponse struct {
	Devices      []BluetoothDevice        `json:"devices"`
	AdapterName  string                   `json:"adapterName"`
	ScanType     string                   `json:"scanType"`
	ScanTime     string                   `json:"scanTime"`
	ScanDuration int64                    `json:"scanDurationMs"`
	Stats        *BluetoothDiscoveryStats `json:"stats,omitempty"`
}

// BluetoothDevice is the flat transport view of a discovered Bluetooth device,
// mirroring discovery.BluetoothDevice's wire shape so the published schema does
// not depend on the discovery domain package. Type and DeviceClass are plain
// strings (the domain's BluetoothType/BluetoothDeviceClass are string enums).
type BluetoothDevice struct {
	ID               string         `json:"id"`
	DeviceID         string         `json:"deviceId,omitempty"`
	Address          string         `json:"address"`
	Name             string         `json:"name"`
	Alias            string         `json:"alias"`
	Vendor           string         `json:"vendor"`
	IsConnected      bool           `json:"isConnected"`
	Type             string         `json:"type"`
	DeviceClass      string         `json:"deviceClass"`
	Appearance       uint16         `json:"appearance"`
	ClassOfDev       uint32         `json:"classOfDevice,omitempty"`
	RSSI             int            `json:"rssi"`
	TxPower          int            `json:"txPower"`
	EstDistanceM     float64        `json:"estDistanceM"`
	IsConnectable    bool           `json:"isConnectable"`
	ServiceUUIDs     []string       `json:"serviceUuids,omitempty"`
	ServiceNames     []string       `json:"serviceNames,omitempty"` // decoded GATT service names
	ManufacturerID   uint16         `json:"manufacturerId,omitempty"`
	CompanyName      string         `json:"companyName,omitempty"`     // decoded manufacturer ID
	AppearanceLabel  string         `json:"appearanceLabel,omitempty"` // decoded BLE appearance
	ManufacturerData []byte         `json:"manufacturerData,omitempty"`
	IsAuthorized     bool           `json:"isAuthorized"`
	IsTrusted        bool           `json:"isTrusted"`
	IsPaired         bool           `json:"isPaired"`
	IsBlocked        bool           `json:"isBlocked"`
	FirstSeen        time.Time      `json:"firstSeen"`
	LastSeen         time.Time      `json:"lastSeen"`
	Metadata         map[string]any `json:"metadata,omitempty"`
}

// BluetoothDiscoveryStats is the flat transport view of Bluetooth scan
// statistics, mirroring discovery.BluetoothDiscoveryStats's wire shape.
type BluetoothDiscoveryStats struct {
	TotalDevices      int            `json:"totalDevices"`
	ClassicDevices    int            `json:"classicDevices"`
	BLEDevices        int            `json:"bleDevices"`
	DualDevices       int            `json:"dualDevices"`
	ConnectedDevices  int            `json:"connectedDevices"`
	AuthorizedCount   int            `json:"authorizedCount"`
	UnauthorizedCount int            `json:"unauthorizedCount"`
	DevicesByClass    map[string]int `json:"devicesByClass"`
	VendorBreakdown   map[string]int `json:"vendorBreakdown"`
	LastScanTime      time.Time      `json:"lastScanTime"`
}

// toBluetoothDevices maps discovered Bluetooth devices onto their flat
// transport view. It always returns a non-nil slice so an empty scan
// serializes as [] not null.
func toBluetoothDevices(devices []discovery.BluetoothDevice) []BluetoothDevice {
	out := make([]BluetoothDevice, 0, len(devices))
	for _, d := range devices {
		out = append(out, BluetoothDevice{
			ID:               d.ID,
			DeviceID:         d.DeviceID,
			Address:          d.Address,
			Name:             d.Name,
			Alias:            d.Alias,
			Vendor:           d.Vendor,
			IsConnected:      d.IsConnected,
			Type:             string(d.Type),
			DeviceClass:      string(d.DeviceClass),
			Appearance:       d.Appearance,
			ClassOfDev:       d.ClassOfDev,
			RSSI:             d.RSSI,
			TxPower:          d.TxPower,
			EstDistanceM:     d.EstDistanceM,
			IsConnectable:    d.IsConnectable,
			ServiceUUIDs:     d.ServiceUUIDs,
			ServiceNames:     decodeBTServices(d.ServiceUUIDs),
			ManufacturerID:   d.ManufacturerID,
			CompanyName:      decodeBTCompany(d.ManufacturerID),
			AppearanceLabel:  decodeBTAppearance(d.Appearance),
			ManufacturerData: d.ManufacturerData,
			IsAuthorized:     d.IsAuthorized,
			IsTrusted:        d.IsTrusted,
			IsPaired:         d.IsPaired,
			IsBlocked:        d.IsBlocked,
			FirstSeen:        d.FirstSeen,
			LastSeen:         d.LastSeen,
			Metadata:         d.Metadata,
		})
	}
	return out
}

// toBluetoothStats maps Bluetooth scan statistics onto their flat transport
// view, preserving nil so an absent stats block stays omitted.
func toBluetoothStats(stats *discovery.BluetoothDiscoveryStats) *BluetoothDiscoveryStats {
	if stats == nil {
		return nil
	}
	return &BluetoothDiscoveryStats{
		TotalDevices:      stats.TotalDevices,
		ClassicDevices:    stats.ClassicDevices,
		BLEDevices:        stats.BLEDevices,
		DualDevices:       stats.DualDevices,
		ConnectedDevices:  stats.ConnectedDevices,
		AuthorizedCount:   stats.AuthorizedCount,
		UnauthorizedCount: stats.UnauthorizedCount,
		DevicesByClass:    stats.DevicesByClass,
		VendorBreakdown:   stats.VendorBreakdown,
		LastScanTime:      stats.LastScanTime,
	}
}

// toBluetoothScanResponse maps a scan result + stats to the API wire shape the
// bluetooth-scan job returns as its result.
func toBluetoothScanResponse(
	result *discovery.BluetoothScanResult, stats *discovery.BluetoothDiscoveryStats,
) BluetoothScanResponse {
	return BluetoothScanResponse{
		Devices:      toBluetoothDevices(result.Devices),
		AdapterName:  result.AdapterName,
		ScanType:     result.ScanType,
		ScanTime:     result.ScanTime.Format("2006-01-02T15:04:05Z07:00"),
		ScanDuration: result.ScanDuration.Milliseconds(),
		Stats:        toBluetoothStats(stats),
	}
}
