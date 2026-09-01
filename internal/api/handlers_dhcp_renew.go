package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dhcp"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// RenewDHCPLeaseRequest asks for a forced DHCP renewal on one interface.
//
// The interface is required rather than defaulted. A renewal changes the
// host's own addressing, and inferring which interface to do that to is not a
// guess worth making on the operator's behalf.
type RenewDHCPLeaseRequest struct {
	Interface string `json:"interface" validate:"required"`
}

// handleRenewDHCPLease handles POST /api/v1/telemetry/dhcp/renew.
//
// Registered operator+ and behind CSRF: this restarts the DHCP client on a
// live interface, which can return a different address and, on a misconfigured
// network, none at all. That is the hazard seed#50 describes, so it is gated
// like the other persistent-write routes rather than registered plainly.
//
// Platforms that cannot do it are refused up front with 501 rather than
// attempted and failed, matching how the VLAN handler declines.
func (s *Server) handleRenewDHCPLease(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if !s.renewSupported() {
		sendErrorResponseWithDetails(w, logger, http.StatusNotImplemented,
			ErrCodeNotImplemented, localizer.T("errors.dhcp.renewNotSupported"), "")

		return
	}

	var req RenewDHCPLeaseRequest
	if !decodeJSONStrictLocalized(w, r, &req, MaxBodySizeJSON, logger, localizer) {
		return
	}
	if req.Interface == "" {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeValidation, localizer.T("errors.dhcp.interfaceRequired"), "")

		return
	}

	if err := s.renewLease(r.Context(), req.Interface); err != nil {
		// An unsupported platform can also surface here when the managing
		// client disappears between the check above and the attempt.
		status, code := http.StatusInternalServerError, ErrCodeInternal
		message := localizer.T("errors.dhcp.renewFailed")
		if errors.Is(err, dhcp.ErrRenewUnsupported) {
			status, code = http.StatusNotImplemented, ErrCodeNotImplemented
			message = localizer.T("errors.dhcp.renewNotSupported")
		}

		logger.ErrorContext(r.Context(), "DHCP lease renewal failed",
			"error", err, "interface", req.Interface)
		sendErrorResponseWithDetails(w, logger, status, code, message, "")

		return
	}

	logger.InfoContext(r.Context(), "DHCP lease renewed", "interface", req.Interface)
	sendJSONResponse(w, nil, http.StatusOK, map[string]any{
		"status":    statusSuccess,
		"interface": req.Interface,
	})
}

// renewSupported reports platform support, through the server's seam when a
// test has set one.
//
// The seam exists because the real function's counterpart, renewLease, would
// renew the DHCP lease of the machine running the tests -- not something a
// unit test may do to its host. Nil means production.
func (s *Server) renewSupported() bool {
	if s.renewSupportedFn != nil {
		return s.renewSupportedFn()
	}

	return dhcp.RenewSupported()
}

// renewLease forces the renewal, through the server's seam when a test has set
// one. Nil means production.
func (s *Server) renewLease(ctx context.Context, interfaceName string) error {
	if s.renewLeaseFn != nil {
		return s.renewLeaseFn(ctx, interfaceName)
	}

	return dhcp.RenewLease(ctx, interfaceName)
}
