package api

import (
	"errors"
	"io"
	"net/http"
	"net/netip"
	"slices"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
	"github.com/MustardSeedNetworks/seed/internal/lifecycle"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// handlers_lifecycle.go serves end-of-life tracking for inventoried hardware
// (P-D9). The operator imports Cisco's EoX data as a file; Seed never fetches
// it, so an air-gapped install stays silent.

// LifecycleEoXResponse is the imported EoX table, in product ID order.
type LifecycleEoXResponse struct {
	Records []lifecycle.EoXRecord `json:"records"`
}

// LifecycleComponent is one inventoried part and where its product stands.
// Record is absent when the imported data has no record for it.
type LifecycleComponent struct {
	ProductID string               `json:"productId"`
	Name      string               `json:"name,omitempty"`
	Class     string               `json:"class,omitempty"`
	SerialNum string               `json:"serialNum,omitempty"`
	Status    lifecycle.Status     `json:"status"              jsonschema:"enum=not_listed,enum=announced,enum=end_of_sale,enum=end_of_sw_maintenance,enum=end_of_security_support,enum=end_of_support"`
	Record    *lifecycle.EoXRecord `json:"record,omitempty"`
}

// LifecycleDevice is one device's inventoried parts.
type LifecycleDevice struct {
	IP         string               `json:"ip"`
	Name       string               `json:"name,omitempty"`
	Components []LifecycleComponent `json:"components"`
}

// LifecycleDevicesResponse is GET /api/v1/lifecycle/devices.
type LifecycleDevicesResponse struct {
	// Records is how many products the imported EoX data covers; zero means
	// nothing has been imported and every part is not_listed.
	Records int               `json:"records"`
	Devices []LifecycleDevice `json:"devices"`
}

func (s *Server) lifecycleRoutes() []route.Route {
	return []route.Route{
		{
			Path:         APIVersionPrefix + "/lifecycle/eox",
			Handler:      s.handleLifecycleEoX,
			Methods:      []string{http.MethodGet, http.MethodPut},
			Scope:        roles.Operator,
			MaxBodyBytes: MaxBodySizeEoXImport,
			Auth:         true,
			CSRF:         true,
		},
		{
			Path:    APIVersionPrefix + "/lifecycle/devices",
			Handler: s.handleLifecycleDevices,
			Methods: []string{http.MethodGet},
			Auth:    true,
		},
	}
}

// handleLifecycleEoX serves /api/v1/lifecycle/eox: GET reads the imported
// table and PUT replaces it with a Cisco EoX API response as downloaded. A
// response with no records clears it.
func (s *Server) handleLifecycleEoX(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())

	if r.Method == http.MethodPut {
		data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxBodySizeEoXImport))
		if _, tooLarge := errors.AsType[*http.MaxBytesError](err); tooLarge {
			sendErrorResponseWithDetails(w, logger, http.StatusRequestEntityTooLarge,
				ErrCodeValidation, "Request body too large", "")
			return
		}
		if err != nil {
			sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
				ErrCodeValidation, "Invalid request body", "")
			return
		}
		table, err := lifecycle.ParseCiscoEoX(data)
		if err != nil {
			sendErrorResponseWithDetails(w, logger, http.StatusBadRequest, ErrCodeValidation,
				"Not a usable Cisco EoX response", err.Error())
			return
		}
		if setErr := s.lifecycleEoX.SetEoX(r.Context(), table); setErr != nil {
			lifecycleFailed(w, r, setErr)
			return
		}
		logger.InfoContext(r.Context(), "EoX data imported", "records", table.Len())
	}

	table, err := s.lifecycleEoX.EoX(r.Context())
	if err != nil {
		lifecycleFailed(w, r, err)
		return
	}
	sendJSONResponse(w, logger, http.StatusOK, LifecycleEoXResponse{Records: table.Records()})
}

// handleLifecycleDevices reports every part the last discovery inventoried
// over ENTITY-MIB that has a model name, and where its product stands today.
func (s *Server) handleLifecycleDevices(w http.ResponseWriter, r *http.Request) {
	table, err := s.lifecycleEoX.EoX(r.Context())
	if err != nil {
		lifecycleFailed(w, r, err)
		return
	}
	var devices []*discovery.DiscoveredDevice
	if s.deviceDiscovery() != nil {
		devices = s.deviceDiscovery().GetDevices()
	}
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK, LifecycleDevicesResponse{
		Records: table.Len(),
		Devices: lifecycleDevices(table, devices, time.Now()),
	})
}

// lifecycleDevices lists the devices with at least one modelled part, in
// address order, each part in entity index order.
func lifecycleDevices(
	table *lifecycle.Table, devices []*discovery.DiscoveredDevice, day time.Time,
) []LifecycleDevice {
	out := []LifecycleDevice{}
	for _, device := range devices {
		if device.SNMPData == nil {
			continue
		}
		inventory := slices.Clone(device.SNMPData.Inventory)
		slices.SortFunc(inventory, func(a, b discovery.SNMPEntity) int { return a.Index - b.Index })
		var components []LifecycleComponent
		for _, entity := range inventory {
			productID := strings.TrimSpace(entity.ModelName)
			if productID == "" {
				continue
			}
			component := LifecycleComponent{
				ProductID: productID, Name: entity.Name, Class: entity.Class, SerialNum: entity.SerialNum,
			}
			record, status := table.Assess(productID, day)
			component.Status = status
			if status != lifecycle.StatusNotListed {
				component.Record = &record
			}
			components = append(components, component)
		}
		if len(components) > 0 {
			out = append(out, LifecycleDevice{IP: device.IP, Name: device.DisplayName, Components: components})
		}
	}
	slices.SortFunc(out, func(a, b LifecycleDevice) int {
		addrA, _ := netip.ParseAddr(a.IP)
		addrB, _ := netip.ParseAddr(b.IP)
		return addrA.Compare(addrB)
	})
	return out
}

func lifecycleFailed(w http.ResponseWriter, r *http.Request, err error) {
	logger := logging.FromContext(r.Context())
	logger.ErrorContext(r.Context(), "Failed to access EoX data", "error", err)
	sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
		ErrCodeInternal, i18n.FromRequest(r).T("errors.settings.saveFailed"), "")
}
