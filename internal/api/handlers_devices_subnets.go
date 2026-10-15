package api

// handlers_devices_subnets.go holds the discovery subnet endpoints: the
// configured target networks, pending learned subnets and their decisions.

import (
	"errors"
	"net/http"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"

	"github.com/MustardSeedNetworks/seed/internal/config"
	discoverysettings "github.com/MustardSeedNetworks/seed/internal/discovery/settings"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// SubnetRequest represents a subnet configuration request.
type SubnetRequest struct {
	CIDR    string `json:"cidr"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

// SubnetResponse represents a subnet in API responses.
type SubnetResponse struct {
	CIDR    string `json:"cidr"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	// Learned is true for a network Seed derived from a router's tables
	// rather than one an operator entered (seed#2695).
	Learned bool `json:"learned"`
	// Decision is the operator's answer to a learned network, "added" or
	// "dismissed"; empty while it awaits one (seed#3108).
	Decision config.LearnedDecision `json:"decision,omitempty" jsonschema:"enum=added,enum=dismissed"`
}

// SubnetDecisionRequest answers the prompt for a learned network: "added"
// switches it on, "dismissed" keeps it off and stops it being offered again.
type SubnetDecisionRequest struct {
	CIDR     string                 `json:"cidr"`
	Decision config.LearnedDecision `json:"decision" jsonschema:"enum=added,enum=dismissed"`
}

// handleDevicesSubnets handles GET/POST/DELETE for target networks (fixes #702 - uses r.Context()).
func (s *Server) handleDevicesSubnets(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	switch r.Method {
	case http.MethodGet:
		s.getDevicesSubnets(w, r)
	case http.MethodPost:
		s.addDevicesSubnet(w, r)
	case http.MethodPut:
		s.updateDevicesSubnet(w, r)
	case http.MethodDelete:
		s.deleteDevicesSubnet(w, r)
	default:
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			"Method not allowed",
			"",
		) // fixes #694, #699
	}
}

// targetNetworkRoutes serves the discovery target networks and the decisions
// on learned ones. Every write is operator+ (writeGated).
func (s *Server) targetNetworkRoutes() []route.Route {
	return []route.Route{
		{
			Path:    APIVersionPrefix + "/security/devices/subnets",
			Handler: s.handleDevicesSubnets,
			Methods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
			Scope:   roles.Operator,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/security/devices/subnets/pending",
			Handler: s.handlePendingSubnets,
			Methods: []string{http.MethodGet, http.MethodPost},
			Scope:   roles.Operator,
			Auth:    true,
			CSRF:    true,
		},
	}
}

// getDevicesSubnets lists the configured target networks. Thin transport.
func (s *Server) getDevicesSubnets(w http.ResponseWriter, r *http.Request) {
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK,
		subnetResponses(s.discoverySettings.Subnets()))
}

func subnetResponses(cfgSubnets []config.SubnetConfig) []SubnetResponse {
	subnets := make([]SubnetResponse, 0, len(cfgSubnets))
	for _, subnet := range cfgSubnets {
		subnets = append(subnets, SubnetResponse{
			CIDR:     subnet.CIDR,
			Name:     subnet.Name,
			Enabled:  subnet.Enabled,
			Learned:  subnet.Learned,
			Decision: subnet.Decision,
		})
	}
	return subnets
}

// handlePendingSubnets lists the learned networks awaiting the operator's
// decision (GET) and records one (POST). Nothing learned is swept until the
// operator adds it (seed#3108).
func (s *Server) handlePendingSubnets(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK,
			subnetResponses(s.discoverySettings.Pending()))
		return
	}

	var req SubnetDecisionRequest
	if !decodeJSONStrict(w, r, &req, MaxBodySizeJSON) {
		return
	}
	err := s.discoverySettings.Decide(req.CIDR, req.Decision)
	s.writeSubnetResult(w, r, err, "Decision recorded")
}

func (s *Server) addDevicesSubnet(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var req SubnetRequest
	if !decodeJSONStrict(w, r, &req, MaxBodySizeJSON) {
		return
	}

	err := s.discoverySettings.AddSubnet(config.SubnetConfig{
		CIDR: req.CIDR, Name: req.Name, Enabled: req.Enabled,
	})
	s.writeSubnetResult(w, r, err, "Subnet added")
}

func (s *Server) updateDevicesSubnet(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeJSON)

	var req SubnetRequest
	if !decodeJSONStrict(w, r, &req, MaxBodySizeJSON) {
		return
	}

	err := s.discoverySettings.UpdateSubnet(config.SubnetConfig{
		CIDR: req.CIDR, Name: req.Name, Enabled: req.Enabled,
	})
	s.writeSubnetResult(w, r, err, "Subnet updated")
}

func (s *Server) deleteDevicesSubnet(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	cidr := r.URL.Query().Get("cidr")
	if cidr == "" {
		sendErrorResponseWithDetails(
			w, logger, http.StatusBadRequest, ErrCodeBadRequest, "CIDR parameter required", "",
		)
		return
	}
	s.writeSubnetResult(w, r, s.discoverySettings.DeleteSubnet(cidr), "Subnet deleted")
}

// writeSubnetResult maps a discovery-settings subnet error to an HTTP response,
// or writes the success message. Centralizes the validation/conflict/not-found/
// save error mapping shared by the subnet add/update/delete handlers.
func (s *Server) writeSubnetResult(
	w http.ResponseWriter, r *http.Request, err error, successMsg string,
) {
	logger := logging.FromContext(r.Context())
	switch {
	case errors.Is(err, discoverysettings.ErrInvalidCIDR):
		sendErrorResponseWithDetails(
			w, logger, http.StatusBadRequest, ErrCodeBadRequest, "Invalid CIDR format", "")
	case errors.Is(err, discoverysettings.ErrSubnetExists):
		sendErrorResponseWithDetails(
			w, logger, http.StatusConflict, ErrCodeConflict, "Subnet already exists", "")
	case errors.Is(err, discoverysettings.ErrInvalidDecision):
		sendErrorResponseWithDetails(
			w, logger, http.StatusBadRequest, ErrCodeBadRequest,
			"Decision must be added or dismissed", "")
	case errors.Is(err, discoverysettings.ErrNotLearned):
		sendErrorResponseWithDetails(
			w, logger, http.StatusConflict, ErrCodeConflict, "Subnet was not learned", "")
	case errors.Is(err, discoverysettings.ErrSubnetNotFound):
		sendErrorResponseWithDetails(
			w, logger, http.StatusNotFound, ErrCodeNotFound, "Subnet not found", "")
	case err != nil:
		logger.ErrorContext(r.Context(), "Failed to save subnet change", "error", err)
		sendErrorResponseWithDetails(
			w, logger, http.StatusInternalServerError, ErrCodeInternal,
			i18n.FromRequest(r).T("errors.settings.saveFailed"), "")
	default:
		sendJSONResponse(w, logger, http.StatusOK, StatusMessageResponse{
			Status: statusSuccess, Message: successMsg,
		})
	}
}
