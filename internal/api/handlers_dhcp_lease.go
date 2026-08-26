package api

import (
	"net/http"
	"slices"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dhcp"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// DHCPLeaseRequest asks for the lease held by one interface.
type DHCPLeaseRequest struct {
	// Interface to inspect. Empty means the currently selected interface.
	Interface string `json:"interface,omitempty"`
}

// handleDHCPLease reports the DHCP lease an interface currently holds.
//
// This inspects existing state rather than exercising the server: macOS reads
// the cached packet via `ipconfig getpacket` and Linux parses the lease file.
// Neither sends a DISCOVER, so the response deliberately makes no claim about
// how quickly a DHCP server answers — see TestResult.ResponseTime.
//
// What it does answer is the question a technician asks on a site visit: what
// did DHCP actually give this interface, and is any of it missing. A lease with
// an address but no gateway or no resolver is a common misconfiguration and is
// reported as a warning rather than a success.
func (s *Server) handleDHCPLease(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	var req DHCPLeaseRequest
	if r.ContentLength > 0 &&
		!decodeJSONStrictLocalized(w, r, &req, MaxBodySizeJSON, logger, localizer) {
		return
	}

	iface := req.Interface
	if iface == "" {
		iface = s.netManager().GetCurrentInterface()
	}
	if iface == "" {
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeValidation, localizer.T("errors.tools.invalidTarget"), "")
		return
	}
	// The name reaches a platform command as an argv element, so there is no
	// shell to inject into — but an unknown interface should be a 400 rather
	// than a 200 carrying a diagnostic failure, so it is checked against the
	// interfaces this host actually has.
	if !hostHasInterface(iface) {
		logger.WarnContext(r.Context(), "DHCP lease requested for an unknown interface",
			"event", "dhcp.lease.unknown_interface")
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeValidation, localizer.T("errors.tools.invalidTarget"), "")
		return
	}

	result := dhcp.NewTester(iface).Test(r.Context())
	sendJSONResponse(w, logger, http.StatusOK, result)
}

// hostHasInterface reports whether the named interface exists on this host.
func hostHasInterface(name string) bool {
	names, err := hostInterfaceNames()
	if err != nil {
		return false
	}
	return slices.Contains(names, name)
}

// hostInterfaceNames returns the interfaces this host has.
func hostInterfaceNames() ([]string, error) {
	return dhcp.GetSystemInterfaces()
}
