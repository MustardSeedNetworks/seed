package api

// handlers_bluetooth.go contains Bluetooth discovery and scanning handlers.

import (
	"net/http"
	"time"

	"github.com/krisarmstrong/seed/internal/discovery"
	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
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
	DeviceID         string         `json:"device_id,omitempty"`
	Address          string         `json:"address"`
	Name             string         `json:"name"`
	Alias            string         `json:"alias"`
	Vendor           string         `json:"vendor"`
	IsConnected      bool           `json:"is_connected"`
	Type             string         `json:"type"`
	DeviceClass      string         `json:"device_class"`
	Appearance       uint16         `json:"appearance"`
	ClassOfDev       uint32         `json:"class_of_device,omitempty"`
	RSSI             int            `json:"rssi"`
	TxPower          int            `json:"tx_power"`
	EstDistanceM     float64        `json:"est_distance_m"`
	IsConnectable    bool           `json:"is_connectable"`
	ServiceUUIDs     []string       `json:"service_uuids,omitempty"`
	ManufacturerID   uint16         `json:"manufacturer_id,omitempty"`
	ManufacturerData []byte         `json:"manufacturer_data,omitempty"`
	IsAuthorized     bool           `json:"is_authorized"`
	IsTrusted        bool           `json:"is_trusted"`
	IsPaired         bool           `json:"is_paired"`
	IsBlocked        bool           `json:"is_blocked"`
	FirstSeen        time.Time      `json:"first_seen"`
	LastSeen         time.Time      `json:"last_seen"`
	Metadata         map[string]any `json:"metadata,omitempty"`
}

// BluetoothDiscoveryStats is the flat transport view of Bluetooth scan
// statistics, mirroring discovery.BluetoothDiscoveryStats's wire shape.
type BluetoothDiscoveryStats struct {
	TotalDevices      int            `json:"total_devices"`
	ClassicDevices    int            `json:"classic_devices"`
	BLEDevices        int            `json:"ble_devices"`
	DualDevices       int            `json:"dual_devices"`
	ConnectedDevices  int            `json:"connected_devices"`
	AuthorizedCount   int            `json:"authorized_count"`
	UnauthorizedCount int            `json:"unauthorized_count"`
	DevicesByClass    map[string]int `json:"devices_by_class"`
	VendorBreakdown   map[string]int `json:"vendor_breakdown"`
	LastScanTime      time.Time      `json:"last_scan_time"`
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
			ManufacturerID:   d.ManufacturerID,
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
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodPost {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		)
		return
	}

	btScanner := s.bluetoothScanner()
	if btScanner == nil {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			"Bluetooth scanner not available",
			"",
		)
		return
	}

	result, err := btScanner.Scan(r.Context())
	if err != nil {
		logger.ErrorContext(r.Context(), "Bluetooth scan failed", "error", err)
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusInternalServerError,
			ErrCodeInternal,
			"Bluetooth scan failed: "+err.Error(),
			"",
		)
		return
	}

	resp := toBluetoothScanResponse(result, btScanner.GetStats())

	sendJSONResponse(w, logger, http.StatusOK, resp)
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
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		)
		return
	}

	btScanner := s.bluetoothScanner()
	if btScanner == nil {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			"Bluetooth scanner not available",
			"",
		)
		return
	}

	lastScan := btScanner.GetLastScan()
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
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		)
		return
	}

	btScanner := s.bluetoothScanner()
	if btScanner == nil {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusServiceUnavailable,
			ErrCodeServiceUnavail,
			"Bluetooth scanner not available",
			"",
		)
		return
	}

	stats := btScanner.GetStats()
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
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		)
		return
	}

	btScanner := s.bluetoothScanner()
	available := btScanner != nil

	var lastScanTime string
	var deviceCount int
	if available {
		if lastScan := btScanner.GetLastScan(); lastScan != nil {
			lastScanTime = lastScan.ScanTime.Format("2006-01-02T15:04:05Z07:00")
			deviceCount = len(lastScan.Devices)
		}
	}

	sendJSONResponse(w, logger, http.StatusOK, map[string]any{
		"available":    available,
		"lastScanTime": lastScanTime,
		"deviceCount":  deviceCount,
	})
}

// bluetoothScanner returns the Bluetooth scanner from the service container.
func (s *Server) bluetoothScanner() *discovery.BluetoothScanner {
	if s.services == nil || s.services.Discovery == nil {
		return nil
	}
	return s.services.Discovery.BluetoothScanner
}
