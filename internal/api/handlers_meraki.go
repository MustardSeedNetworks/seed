package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/meraki"
	"github.com/MustardSeedNetworks/seed/internal/outbound"
)

// handlers_meraki.go serves the opt-in Cisco Meraki Dashboard API integration
// (P-D12). It is off until an admin answers on, so an air-gapped install
// stays silent.

// MerakiSettingsResponse is GET /api/v1/integrations/meraki. Choice
// "unanswered" means the operator has not been asked yet, and the integration
// is off until they answer on. The API key is never served.
type MerakiSettingsResponse struct {
	Choice    outbound.Choice `json:"choice"    jsonschema:"enum=unanswered,enum=off,enum=on"`
	Endpoint  string          `json:"endpoint"`
	HasAPIKey bool            `json:"hasApiKey"`
	// LastAttempt, LastSuccess and LastError describe the last read since the
	// daemon started; absent when none has run.
	LastAttempt *time.Time `json:"lastAttempt,omitempty"`
	LastSuccess *time.Time `json:"lastSuccess,omitempty"`
	LastError   string     `json:"lastError,omitempty"`
}

// MerakiSettingsUpdate is the PUT body: the whole integration setting. Choice
// answers the question; it cannot be set back to unanswered. An empty
// endpoint means Meraki's. An empty apiKey keeps the stored one; clearApiKey
// removes it.
type MerakiSettingsUpdate struct {
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

// merakiSettings is the integration's settings use-case.
type merakiSettings interface {
	Settings() meraki.Settings
	Update(u meraki.Update) error
}

func (s *Server) merakiRoutes() []route.Route {
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
	}
}

// initMeraki builds the integration settings and registers the refresher,
// which reads only while the integration is on.
func (s *Server) initMeraki() {
	settings := app.NewMeraki(s.config, s.configPath)
	s.meraki = settings
	s.merakiRefresher = meraki.NewRefresher(settings, meraki.NewClient(nil),
		meraki.RefreshInterval, logging.GetLogger())
	if err := s.registerEngineIfLicensed(s.merakiRefresher); err != nil {
		logging.GetLogger().Warn("Meraki engine registration failed", "error", err)
	}
}

// handleMerakiSettings serves /api/v1/integrations/meraki: GET reads the
// settings and the last read, PUT records the admin's answer and settings and
// asks for a read now, which also drops held device state when the answer is
// off. Admin only: turning it on makes Seed call out, with a key it holds.
func (s *Server) handleMerakiSettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	if r.Method == http.MethodPut {
		var in MerakiSettingsUpdate
		if !decodeJSONStrict(w, r, &in, MaxBodySizeConfig) {
			return
		}
		err := s.meraki.Update(meraki.Update{
			Choice:      in.Choice,
			Endpoint:    strings.TrimSpace(in.Endpoint),
			APIKey:      strings.TrimSpace(in.APIKey),
			ClearAPIKey: in.ClearAPIKey,
		})
		if errors.Is(err, meraki.ErrInvalidSettings) {
			writeError(w, r, http.StatusBadRequest, ErrCodeValidation, err.Error())
			return
		}
		if err != nil {
			merakiFailed(w, r, err)
			return
		}
		// Update accepted only "on" or "off", so a bool records the whole answer
		// without echoing request text into the log.
		logger.InfoContext(r.Context(), "Meraki settings changed",
			"event", "integrations.meraki.updated", "enabled", in.Choice == outbound.ChoiceOn,
			"user", usernameFromContext(r))
		if s.merakiRefresher != nil {
			s.merakiRefresher.Wake()
		}
	}

	settings := s.meraki.Settings()
	resp := MerakiSettingsResponse{
		Choice:    settings.Choice,
		Endpoint:  settings.Endpoint,
		HasAPIKey: settings.HasAPIKey,
	}
	if s.merakiRefresher != nil {
		snapshot := s.merakiRefresher.Snapshot()
		if !snapshot.LastAttempt.IsZero() {
			resp.LastAttempt = &snapshot.LastAttempt
			resp.LastError = snapshot.LastError
		}
		if !snapshot.LastSuccess.IsZero() {
			resp.LastSuccess = &snapshot.LastSuccess
		}
	}
	sendJSONResponse(w, logger, http.StatusOK, resp)
}

// handleMerakiDevices serves the last read's devices. Nothing is served while
// the integration is off, even before the refresher has dropped what it held.
func (s *Server) handleMerakiDevices(w http.ResponseWriter, r *http.Request) {
	resp := MerakiDevicesResponse{Devices: []meraki.Device{}}
	if s.merakiRefresher != nil && s.meraki.Settings().Choice == outbound.ChoiceOn {
		snapshot := s.merakiRefresher.Snapshot()
		if !snapshot.LastSuccess.IsZero() {
			resp.ReadAt = &snapshot.LastSuccess
			resp.Devices = snapshot.Devices
		}
	}
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK, resp)
}

func merakiFailed(w http.ResponseWriter, r *http.Request, err error) {
	logger := logging.FromContext(r.Context())
	logger.ErrorContext(r.Context(), "Failed to save Meraki settings", "error", err)
	sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
		ErrCodeInternal, i18n.FromRequest(r).T("errors.settings.saveFailed"), "")
}
