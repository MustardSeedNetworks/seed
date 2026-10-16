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

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
	"github.com/MustardSeedNetworks/seed/internal/lifecycle"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/outbound"
)

// handlers_lifecycle.go serves end-of-life tracking for inventoried hardware
// (P-D9). The operator imports Cisco's EoX data as a file, or turns on the
// feed that fetches it. The feed is off until an admin answers on, so an
// air-gapped install stays silent.

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

// LifecycleFeedResponse is GET /api/v1/lifecycle/feed. Choice "unanswered"
// means the operator has not been asked yet, and the feed is off until they
// answer on. The client secret is never served.
type LifecycleFeedResponse struct {
	Choice          outbound.Choice `json:"choice"          jsonschema:"enum=unanswered,enum=off,enum=on"`
	Endpoint        string          `json:"endpoint"`
	TokenURL        string          `json:"tokenUrl"`
	ClientID        string          `json:"clientId"`
	HasClientSecret bool            `json:"hasClientSecret"`
	// LastAttempt, LastSuccess and LastError describe the last refresh since
	// the daemon started; absent when none has run.
	LastAttempt *time.Time `json:"lastAttempt,omitempty"`
	LastSuccess *time.Time `json:"lastSuccess,omitempty"`
	LastError   string     `json:"lastError,omitempty"`
	// Fetched is how many products the last successful refresh returned.
	Fetched int `json:"fetched"`
}

// LifecycleFeedUpdate is the PUT body: the whole feed setting. Choice answers
// the question; it cannot be set back to unanswered. Empty endpoint and
// tokenUrl mean Cisco's. An empty clientSecret keeps the stored one;
// clearClientSecret removes it.
type LifecycleFeedUpdate struct {
	Choice            outbound.Choice `json:"choice"                      jsonschema:"enum=off,enum=on"`
	Endpoint          string          `json:"endpoint,omitempty"`
	TokenURL          string          `json:"tokenUrl,omitempty"`
	ClientID          string          `json:"clientId,omitempty"`
	ClientSecret      string          `json:"clientSecret,omitempty"`
	ClearClientSecret bool            `json:"clearClientSecret,omitempty"`
}

// lifecycleFeedSettings is the feed's settings use-case.
type lifecycleFeedSettings interface {
	Settings() lifecycle.FeedSettings
	Update(u lifecycle.FeedUpdate) error
}

func (s *Server) lifecycleRoutes() []route.Route {
	return []route.Route{
		{
			Path:         APIVersionPrefix + "/lifecycle/feed",
			Handler:      s.handleLifecycleFeed,
			Methods:      []string{http.MethodGet, http.MethodPut},
			Scope:        roles.Admin,
			MaxBodyBytes: MaxBodySizeConfig,
			Auth:         true,
			CSRF:         true,
		},
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

// handleLifecycleFeed serves /api/v1/lifecycle/feed: GET reads the feed
// settings and the last refresh, PUT records the admin's answer and settings
// and, when the answer is on, asks for a refresh now. Admin only: turning it
// on makes Seed call out, with credentials it holds.
func (s *Server) handleLifecycleFeed(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	if r.Method == http.MethodPut {
		var in LifecycleFeedUpdate
		if !decodeJSONStrict(w, r, &in, MaxBodySizeConfig) {
			return
		}
		err := s.lifecycleFeed.Update(lifecycle.FeedUpdate{
			Choice:            in.Choice,
			Endpoint:          strings.TrimSpace(in.Endpoint),
			TokenURL:          strings.TrimSpace(in.TokenURL),
			ClientID:          strings.TrimSpace(in.ClientID),
			ClientSecret:      strings.TrimSpace(in.ClientSecret),
			ClearClientSecret: in.ClearClientSecret,
		})
		if errors.Is(err, lifecycle.ErrInvalidFeed) {
			writeError(w, r, http.StatusBadRequest, ErrCodeValidation, err.Error())
			return
		}
		if err != nil {
			lifecycleFailed(w, r, err)
			return
		}
		// Update accepted only "on" or "off", so a bool records the whole answer
		// without echoing request text into the log.
		enabled := in.Choice == outbound.ChoiceOn
		logger.InfoContext(r.Context(), "EoX feed settings changed",
			"event", "lifecycle.feed.updated", "enabled", enabled, "user", usernameFromContext(r))
		if enabled && s.lifecycleRefresher != nil {
			s.lifecycleRefresher.Wake()
		}
	}

	settings := s.lifecycleFeed.Settings()
	resp := LifecycleFeedResponse{
		Choice:          settings.Choice,
		Endpoint:        settings.Endpoint,
		TokenURL:        settings.TokenURL,
		ClientID:        settings.ClientID,
		HasClientSecret: settings.HasClientSecret,
	}
	if s.lifecycleRefresher != nil {
		status := s.lifecycleRefresher.Status()
		if !status.LastAttempt.IsZero() {
			resp.LastAttempt = &status.LastAttempt
			resp.LastError = status.LastError
		}
		if !status.LastSuccess.IsZero() {
			resp.LastSuccess = &status.LastSuccess
		}
		resp.Fetched = status.Fetched
	}
	sendJSONResponse(w, logger, http.StatusOK, resp)
}

// initLifecycleFeed builds the feed settings and registers the refresher,
// which asks about the inventory only while the feed is on.
func (s *Server) initLifecycleFeed() {
	feed := app.NewLifecycleFeed(s.config, s.configPath)
	s.lifecycleFeed = feed
	s.lifecycleRefresher = lifecycle.NewRefresher(feed, s.inventoryProductIDs, s.lifecycleEoX,
		lifecycle.NewFetcher(nil), lifecycle.RefreshInterval, logging.GetLogger())
	if err := s.registerEngineIfLicensed(s.lifecycleRefresher); err != nil {
		logging.GetLogger().Warn("lifecycle feed engine registration failed", "error", err)
	}
}

// inventoryProductIDs lists every distinct model name the last discovery
// inventoried, in order.
func (s *Server) inventoryProductIDs() []string {
	if s.deviceDiscovery() == nil {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, device := range s.deviceDiscovery().GetDevices() {
		if device.SNMPData == nil {
			continue
		}
		for _, entity := range device.SNMPData.Inventory {
			id := strings.ToUpper(strings.TrimSpace(entity.ModelName))
			if id != "" && !seen[id] {
				seen[id] = true
				out = append(out, id)
			}
		}
	}
	slices.Sort(out)
	return out
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
