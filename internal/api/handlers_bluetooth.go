package api

// handlers_bluetooth.go contains Bluetooth discovery and scanning handlers. The
// handlers are pure transport (ADR-0020): they map the domain result onto the
// flat wire DTOs and map a bluetooth use-case sentinel to its HTTP status; the
// scan orchestration lives in internal/discovery/bluetooth, with the adapter in
// internal/app.

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/discovery/bluetooth"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// ============================================================================
// Bluetooth Discovery API Handlers
// ============================================================================

// BluetoothScanResponse contains Bluetooth scan results.
type BluetoothScanResponse struct {
	Devices      []BluetoothDevice        `json:"devices"`
	AdapterName  string                   `json:"adapterName"`
	ScanType     string                   `json:"scanType"`
	ScanTime     string                   `json:"scanTime"`
	ScanDuration int64                    `json:"scanDurationMs"`
	Stats        *BluetoothDiscoveryStats `json:"stats,omitempty"`
}

// BluetoothDevicesResponse contains Bluetooth devices.
type BluetoothDevicesResponse struct {
	Devices []BluetoothDevice `json:"devices"`
	Total   int               `json:"total"`
}

// BluetoothStatsResponse contains Bluetooth statistics.
type BluetoothStatsResponse struct {
	Stats *BluetoothDiscoveryStats `json:"stats"`
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

// writeBluetoothUnavailable writes the pre-strangle 503 for an absent scanner.
func writeBluetoothUnavailable(w http.ResponseWriter, logger *slog.Logger) {
	sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable,
		ErrCodeServiceUnavail, "Bluetooth scanner not available", "")
}

// handleBluetoothScan triggers a Bluetooth scan and returns results.
//
// POST /api/v1/security/bluetooth/scan
//
// Triggers an active Bluetooth scan on the configured adapter.
// Returns discovered devices including classic and BLE.
//
// Response: 200 OK with BluetoothScanResponse.
func (s *Server) handleBluetoothScan(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	scan, err := s.bluetoothScans.Scan(r.Context())
	if err != nil {
		if errors.Is(err, bluetooth.ErrUnavailable) {
			writeBluetoothUnavailable(w, logger)
			return
		}
		logger.ErrorContext(r.Context(), "Bluetooth scan failed", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, "Bluetooth scan failed: "+err.Error(), "")
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, toBluetoothScanResponse(scan.Result, scan.Stats))
}

// toBluetoothScanResponse maps a scan result + stats to the API wire shape.
// Shared by the scan handler and the bluetooth-scan job kind so both paths
// produce an identical response.
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

// handleBluetoothDevices returns discovered Bluetooth devices.
//
// GET /api/v1/security/bluetooth/devices
//
// Returns the list of Bluetooth devices from the most recent scan.
//
// Response: 200 OK with BluetoothDevicesResponse.
func (s *Server) handleBluetoothDevices(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	lastScan, err := s.bluetoothScans.Devices()
	if err != nil {
		writeBluetoothUnavailable(w, logger)
		return
	}

	if lastScan == nil {
		sendJSONResponse(w, logger, http.StatusOK, BluetoothDevicesResponse{
			Devices: []BluetoothDevice{},
			Total:   0,
		})
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, BluetoothDevicesResponse{
		Devices: toBluetoothDevices(lastScan.Devices),
		Total:   len(lastScan.Devices),
	})
}

// handleBluetoothStats returns Bluetooth discovery statistics.
//
// GET /api/v1/security/bluetooth/stats
//
// Returns aggregated statistics from Bluetooth discovery.
//
// Response: 200 OK with BluetoothStatsResponse.
func (s *Server) handleBluetoothStats(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	stats, err := s.bluetoothScans.Stats()
	if err != nil {
		writeBluetoothUnavailable(w, logger)
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, BluetoothStatsResponse{
		Stats: toBluetoothStats(stats),
	})
}

// handleBluetoothStatus returns the Bluetooth adapter status.
//
// GET /api/v1/security/bluetooth/status
//
// Returns the current Bluetooth adapter status and availability.
//
// Response: 200 OK with status information.
func (s *Server) handleBluetoothStatus(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	status := s.bluetoothScans.Status()

	var lastScanTime string
	var deviceCount int
	if status.LastScan != nil {
		lastScanTime = status.LastScan.ScanTime.Format("2006-01-02T15:04:05Z07:00")
		deviceCount = len(status.LastScan.Devices)
	}

	sendJSONResponse(w, logger, http.StatusOK, map[string]any{
		"available":    status.Available,
		"lastScanTime": lastScanTime,
		"deviceCount":  deviceCount,
	})
}
