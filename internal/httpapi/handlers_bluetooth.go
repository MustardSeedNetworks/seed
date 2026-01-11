package httpapi

// handlers_bluetooth.go contains Bluetooth discovery and scanning handlers.

import (
	"net/http"

	"github.com/krisarmstrong/seed/internal/discovery"
	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
)

// ============================================================================
// Bluetooth Discovery API Handlers
// ============================================================================

// BluetoothScanResponse contains Bluetooth scan results.
type BluetoothScanResponse struct {
	Devices      []discovery.BluetoothDevice `json:"devices"`
	AdapterName  string                      `json:"adapterName"`
	ScanType     string                      `json:"scanType"`
	ScanTime     string                      `json:"scanTime"`
	ScanDuration int64                       `json:"scanDurationMs"`
	Stats        *discovery.BluetoothDiscoveryStats `json:"stats,omitempty"`
}

// BluetoothDevicesResponse contains Bluetooth devices.
type BluetoothDevicesResponse struct {
	Devices []discovery.BluetoothDevice `json:"devices"`
	Total   int                         `json:"total"`
}

// BluetoothStatsResponse contains Bluetooth statistics.
type BluetoothStatsResponse struct {
	Stats *discovery.BluetoothDiscoveryStats `json:"stats"`
}

// handleBluetoothScan triggers a Bluetooth scan and returns results.
//
// POST /api/v1/shell/bluetooth/scan
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
		logger.Error("Bluetooth scan failed", "error", err)
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

	resp := BluetoothScanResponse{
		Devices:      result.Devices,
		AdapterName:  result.AdapterName,
		ScanType:     result.ScanType,
		ScanTime:     result.ScanTime.Format("2006-01-02T15:04:05Z07:00"),
		ScanDuration: result.ScanDuration.Milliseconds(),
		Stats:        btScanner.GetStats(),
	}

	sendJSONResponse(w, logger, http.StatusOK, resp)
}

// handleBluetoothDevices returns discovered Bluetooth devices.
//
// GET /api/v1/shell/bluetooth/devices
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
			Devices: []discovery.BluetoothDevice{},
			Total:   0,
		})
		return
	}

	sendJSONResponse(w, logger, http.StatusOK, BluetoothDevicesResponse{
		Devices: lastScan.Devices,
		Total:   len(lastScan.Devices),
	})
}

// handleBluetoothStats returns Bluetooth discovery statistics.
//
// GET /api/v1/shell/bluetooth/stats
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
		Stats: stats,
	})
}

// handleBluetoothStatus returns the Bluetooth adapter status.
//
// GET /api/v1/shell/bluetooth/status
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
