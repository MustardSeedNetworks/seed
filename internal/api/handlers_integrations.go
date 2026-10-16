package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/engine"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/meraki"
	"github.com/MustardSeedNetworks/seed/internal/outbound"
	"github.com/MustardSeedNetworks/seed/internal/paloalto"
)

// handlers_integrations.go serves the opt-in vendor management API
// integrations (P-D12): Cisco Meraki's Dashboard API and the PAN-OS XML API.
// Each is off until an admin answers on, so an air-gapped install stays
// silent.

// IntegrationSettingsResponse is GET /api/v1/integrations/{vendor}. Choice
// "unanswered" means the operator has not been asked yet, and the integration
// is off until they answer on. The API key is never served.
type IntegrationSettingsResponse struct {
	Choice    outbound.Choice `json:"choice"    jsonschema:"enum=unanswered,enum=off,enum=on"`
	Endpoint  string          `json:"endpoint"`
	HasAPIKey bool            `json:"hasApiKey"`
	// LastAttempt, LastSuccess and LastError describe the last read since the
	// daemon started; absent when none has run.
	LastAttempt *time.Time `json:"lastAttempt,omitempty"`
	LastSuccess *time.Time `json:"lastSuccess,omitempty"`
	LastError   string     `json:"lastError,omitempty"`
}

// IntegrationSettingsUpdate is the PUT body: the whole integration setting.
// Choice answers the question; it cannot be set back to unanswered. An empty
// endpoint means the vendor's default, which Meraki has and PAN-OS does not.
// An empty apiKey keeps the stored one; clearApiKey removes it.
type IntegrationSettingsUpdate struct {
	Choice      outbound.Choice `json:"choice"                jsonschema:"enum=off,enum=on"`
	Endpoint    string          `json:"endpoint,omitempty"`
	APIKey      string          `json:"apiKey,omitempty"`
	ClearAPIKey bool            `json:"clearApiKey,omitempty"`
}

// MerakiDevicesResponse is GET /api/v1/integrations/meraki/devices: the
// devices the last successful read returned, empty while the integration is
// off or before its first read.
type MerakiDevicesResponse struct {
	ReadAt  *time.Time      `json:"readAt,omitempty"`
	Devices []meraki.Device `json:"devices"`
}

// PaloAltoDevicesResponse is GET /api/v1/integrations/paloalto/devices: the
// endpoint and, when it is Panorama, the firewalls it manages, as the last
// successful read returned them; empty while the integration is off or
// before its first read.
type PaloAltoDevicesResponse struct {
	ReadAt  *time.Time        `json:"readAt,omitempty"`
	Devices []paloalto.Device `json:"devices"`
}

// vendorAPISettings is one integration's settings use-case.
type vendorAPISettings interface {
	Settings() outbound.Settings
	Update(u outbound.Update) error
}

// vendorAPIReader is what the settings route needs of an integration's reader.
type vendorAPIReader interface {
	Status() outbound.Status
	Wake()
}

func (s *Server) integrationRoutes() []route.Route {
	return []route.Route{
		{
			Path:         APIVersionPrefix + "/integrations/meraki",
			Handler:      s.handleMerakiSettings,
			Methods:      []string{http.MethodGet, http.MethodPut},
			Scope:        roles.Admin,
			MaxBodyBytes: MaxBodySizeConfig,
			Auth:         true,
			CSRF:         true,
		},
		{
			Path:    APIVersionPrefix + "/integrations/meraki/devices",
			Handler: s.handleMerakiDevices,
			Methods: []string{http.MethodGet},
			Auth:    true,
		},
		{
			Path:         APIVersionPrefix + "/integrations/paloalto",
			Handler:      s.handlePaloAltoSettings,
			Methods:      []string{http.MethodGet, http.MethodPut},
			Scope:        roles.Admin,
			MaxBodyBytes: MaxBodySizeConfig,
			Auth:         true,
			CSRF:         true,
		},
		{
			Path:    APIVersionPrefix + "/integrations/paloalto/devices",
			Handler: s.handlePaloAltoDevices,
			Methods: []string{http.MethodGet},
			Auth:    true,
		},
	}
}

// initIntegrations builds each integration's settings and registers its
// reader, which reads only while the integration is on.
func (s *Server) initIntegrations() {
	logger := logging.GetLogger()

	merakiSettings := app.NewMeraki(s.config, s.configPath)
	s.meraki = merakiSettings
	s.merakiReader = outbound.NewReader("meraki", merakiSettings, meraki.NewClient(nil).Devices,
		meraki.RefreshInterval, logger)

	paloAltoSettings := app.NewPaloAlto(s.config, s.configPath)
	s.paloAlto = paloAltoSettings
	s.paloAltoReader = outbound.NewReader("paloalto", paloAltoSettings, paloalto.NewClient(nil).Devices,
		paloalto.RefreshInterval, logger)

	for _, reader := range []engine.Engine{s.merakiReader, s.paloAltoReader} {
		if err := s.registerEngineIfLicensed(reader); err != nil {
			logger.Warn("Vendor API engine registration failed", "integration", reader.Name(), "error", err)
		}
	}
}

func (s *Server) handleMerakiSettings(w http.ResponseWriter, r *http.Request) {
	s.serveIntegrationSettings(w, r, "meraki", s.meraki, s.merakiReader)
}

func (s *Server) handlePaloAltoSettings(w http.ResponseWriter, r *http.Request) {
	s.serveIntegrationSettings(w, r, "paloalto", s.paloAlto, s.paloAltoReader)
}

// serveIntegrationSettings serves /api/v1/integrations/{vendor}: GET reads
// the settings and the last read, PUT records the admin's answer and
// settings and asks for a read now, which also drops held device state when
// the answer is off. Admin only: turning it on makes Seed call out, with a
// key it holds.
func (s *Server) serveIntegrationSettings(
	w http.ResponseWriter, r *http.Request, vendor string, settings vendorAPISettings, reader vendorAPIReader,
) {
	logger := logging.FromContext(r.Context())
	if r.Method == http.MethodPut {
		var in IntegrationSettingsUpdate
		if !decodeJSONStrict(w, r, &in, MaxBodySizeConfig) {
			return
		}
		err := settings.Update(outbound.Update{
			Choice:      in.Choice,
			Endpoint:    strings.TrimSpace(in.Endpoint),
			APIKey:      strings.TrimSpace(in.APIKey),
			ClearAPIKey: in.ClearAPIKey,
		})
		if errors.Is(err, outbound.ErrInvalidSettings) {
			writeError(w, r, http.StatusBadRequest, ErrCodeValidation, err.Error())
			return
		}
		if err != nil {
			logger.ErrorContext(r.Context(), "Failed to save vendor API settings", "integration", vendor, "error", err)
			sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
				ErrCodeInternal, i18n.FromRequest(r).T("errors.settings.saveFailed"), "")
			return
		}
		// Update accepted only "on" or "off", so a bool records the whole answer
		// without echoing request text into the log.
		logger.InfoContext(r.Context(), "Vendor API settings changed",
			"event", "integrations."+vendor+".updated", "enabled", in.Choice == outbound.ChoiceOn,
			"user", usernameFromContext(r))
		reader.Wake()
	}

	current := settings.Settings()
	resp := IntegrationSettingsResponse{
		Choice:    current.Choice,
		Endpoint:  current.Endpoint,
		HasAPIKey: current.HasAPIKey,
	}
	status := reader.Status()
	if !status.LastAttempt.IsZero() {
		resp.LastAttempt = &status.LastAttempt
		resp.LastError = status.LastError
	}
	if !status.LastSuccess.IsZero() {
		resp.LastSuccess = &status.LastSuccess
	}
	sendJSONResponse(w, logger, http.StatusOK, resp)
}

func (s *Server) handleMerakiDevices(w http.ResponseWriter, r *http.Request) {
	readAt, devices := integrationDevices(s.meraki, s.merakiReader)
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK,
		MerakiDevicesResponse{ReadAt: readAt, Devices: devices})
}

func (s *Server) handlePaloAltoDevices(w http.ResponseWriter, r *http.Request) {
	readAt, devices := integrationDevices(s.paloAlto, s.paloAltoReader)
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK,
		PaloAltoDevicesResponse{ReadAt: readAt, Devices: devices})
}

// integrationDevices returns the last read's devices and when it ran. Nothing
// is returned while the integration is off, even before the reader has
// dropped what it held.
func integrationDevices[T any](settings vendorAPISettings, reader *outbound.Reader[T]) (*time.Time, []T) {
	if settings.Settings().Choice != outbound.ChoiceOn {
		return nil, []T{}
	}
	status := reader.Status()
	items := reader.Items()
	if status.LastSuccess.IsZero() || items == nil {
		return nil, []T{}
	}
	return &status.LastSuccess, items
}
