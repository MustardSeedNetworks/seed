package api

// handlers_neighbours.go serves this device's own neighbour cache (#328).
//
// Deliberately not /topology/arp. That route serves SNMP-harvested ARP bindings
// from remote nodes, persisted and filtered by client and node — it answers
// "what does that switch think its ARP table is". This answers "what does this
// box see on the wire in front of it", which is what an operator needs when an
// IP is not resolving to a MAC on the segment they are plugged into. The two
// are easy to confuse, so the paths do not look alike.

import (
	"net"
	"net/http"

	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// NeighbourEntry is one entry of the local cache as the API reports it.
type NeighbourEntry struct {
	IP        string `json:"ip"`
	MAC       string `json:"mac"`
	Vendor    string `json:"vendor,omitempty"`
	Interface string `json:"interface,omitempty"`
	State     string `json:"state,omitempty"`
	// Family is "ipv4" or "ipv6", so the UI can tell an ARP entry from an NDP
	// one without re-parsing the address.
	Family string `json:"family"`
}

// NeighbourCacheResponse is the local neighbour cache.
type NeighbourCacheResponse struct {
	Entries []NeighbourEntry `json:"entries"`
	Total   int              `json:"total"`
}

// handleNeighbourCache serves GET /api/v1/network/neighbours.
//
// Read-only and ungated beyond authentication: it reports what this host can
// already see on its own link, which every authenticated role is entitled to
// read. Nothing here mutates state or reaches another device.
func (s *Server) handleNeighbourCache(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if s.discoveryService() == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable,
			ErrCodeInternal, localizer.T("errors.discovery.managerUnavailable"), "")

		return
	}

	entries, err := s.discoveryService().ReadNeighbourCache()
	if err != nil {
		logger.WarnContext(r.Context(), "Failed to read the neighbour cache", "error", err)
		sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
			ErrCodeInternal, localizer.T("errors.network.neighbourCacheFailed"), "")

		return
	}

	out := make([]NeighbourEntry, 0, len(entries))
	for _, entry := range entries {
		out = append(out, NeighbourEntry{
			IP:        entry.IP,
			MAC:       entry.MAC,
			Vendor:    entry.Vendor,
			Interface: entry.Interface,
			State:     entry.State,
			Family:    addressFamily(entry.IP),
		})
	}

	sendJSONResponse(w, logger, http.StatusOK, NeighbourCacheResponse{
		Entries: out,
		Total:   len(out),
	})
}

// addressFamily names the family of an address so the client does not have to
// parse it again to group the table.
func addressFamily(addr string) string {
	ip := net.ParseIP(addr)
	switch {
	case ip == nil:
		return "unknown"
	case ip.To4() != nil:
		return "ipv4"
	default:
		return "ipv6"
	}
}
