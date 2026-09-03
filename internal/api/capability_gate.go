package api

// capability_gate.go refuses a request the platform cannot serve, with a
// structured 501 that names the capability and says why (#750).
//
// The 501 pattern already existed in two places — VLAN creation and DHCP lease
// renewal — each with its own hand-written check and its own message key. This
// makes the check one thing, driven by internal/capabilities, so a handler
// gated here and the matrix in HARDWARE.md cannot disagree about whether a
// platform supports something.
//
// Deliberately a different refusal from the two it sits beside:
//
//	403  role      — you may not
//	402  tier      — your licence does not include it
//	501  platform  — this operating system cannot, and no amount of
//	                 privilege or money changes that here
//
// Three remedies. Returning the wrong one sends an operator somewhere useless.

import (
	"net/http"

	"github.com/MustardSeedNetworks/seed/internal/capabilities"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// requirePlatform reports whether the capability works here, and writes a 501
// naming it when it does not.
//
// Returns true when the caller should continue. The note carried in the
// response is the one internal/capabilities holds, so the API, the UI banner
// and the generated document all say the same words.
func (s *Server) requirePlatform(
	w http.ResponseWriter, r *http.Request, capability capabilities.Capability,
) bool {
	if capabilities.Supported(capability) {
		return true
	}

	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	var detail string
	for _, entry := range capabilities.Report() {
		if entry.Capability == capability {
			detail = entry.Note

			break
		}
	}

	sendErrorResponseWithDetails(
		w,
		logger,
		http.StatusNotImplemented,
		ErrCodeNotImplemented,
		localizer.T("errors.platform.capabilityUnsupported"),
		detail,
	)

	return false
}
