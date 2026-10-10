package api

import (
	"net/http"
	"strconv"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"

	"github.com/MustardSeedNetworks/seed/internal/flows"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// handlers_flows.go serves top talkers and top conversations over the flows
// the collector (internal/listener/flow) stored. The window is resolved
// against the licence tier's horizons exactly as the history series are, and
// reported back the same way, so a client can say what it is showing.

// Top-N list sizes. The default is a dashboard card; the cap keeps one
// request bounded on an appliance.
const (
	defaultFlowTopLimit = 10
	maxFlowTopLimit     = 100
)

// FlowTalkersResponse is the top hosts by traffic sent and received.
type FlowTalkersResponse struct {
	Window  HistoryWindowResponse `json:"window"`
	By      flows.Rank            `json:"by"`
	Talkers []flows.Talker        `json:"talkers"`
}

// FlowConversationsResponse is the top host pairs, per protocol, by traffic
// in both directions.
type FlowConversationsResponse struct {
	Window        HistoryWindowResponse `json:"window"`
	By            flows.Rank            `json:"by"`
	Conversations []flows.Conversation  `json:"conversations"`
}

// flowRoutes registers the top-N reads, the application signature table and
// the threat indicator list. The reads have no role gate and no feature gate: the collector that fills
// flow_records is Pro, so below Pro they answer with empty lists, over the
// window the tier retains. Editing the signature table or the indicator
// list is a persistent write, so it is operator-gated.
func (s *Server) flowRoutes() []route.Route {
	get := []string{http.MethodGet}
	return []route.Route{
		{
			Path:    APIVersionPrefix + "/flows/top-talkers",
			Handler: s.handleFlowTopTalkers,
			Methods: get,
			Auth:    true,
		},
		{
			Path:    APIVersionPrefix + "/flows/top-conversations",
			Handler: s.handleFlowTopConversations,
			Methods: get,
			Auth:    true,
		},
		{
			Path:    APIVersionPrefix + "/flows/top-applications",
			Handler: s.handleFlowTopApplications,
			Methods: get,
			Auth:    true,
		},
		{
			Path:         APIVersionPrefix + "/flows/application-signatures",
			Handler:      s.handleAppSignatures,
			Methods:      []string{http.MethodGet, http.MethodPut, http.MethodDelete},
			Scope:        roles.Operator,
			MaxBodyBytes: MaxBodySizeConfig,
			Auth:         true,
			CSRF:         true,
		},
		{
			Path:         APIVersionPrefix + "/flows/threat-indicators",
			Handler:      s.handleFlowIndicators,
			Methods:      []string{http.MethodGet, http.MethodPut},
			Scope:        roles.Operator,
			MaxBodyBytes: MaxBodySizeJSON,
			Auth:         true,
			CSRF:         true,
		},
	}
}

// flowTopQuery is a parsed top-N request: the read, and the window it was
// resolved from, which the response reports.
type flowTopQuery struct {
	read   flows.Query
	window historyWindow
}

// handleFlowTopTalkers serves GET /api/v1/flows/top-talkers.
func (s *Server) handleFlowTopTalkers(w http.ResponseWriter, r *http.Request) {
	q, ok := s.flowTopQuery(w, r)
	if !ok {
		return
	}
	talkers, err := s.flows.TopTalkers(r.Context(), q.read)
	if err != nil {
		flowReadFailed(w, r, err)
		return
	}
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK, FlowTalkersResponse{
		Window:  windowResponse(q.window),
		By:      q.read.By,
		Talkers: emptyIfNil(talkers),
	})
}

// handleFlowTopConversations serves GET /api/v1/flows/top-conversations.
func (s *Server) handleFlowTopConversations(w http.ResponseWriter, r *http.Request) {
	q, ok := s.flowTopQuery(w, r)
	if !ok {
		return
	}
	conversations, err := s.flows.TopConversations(r.Context(), q.read)
	if err != nil {
		flowReadFailed(w, r, err)
		return
	}
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK, FlowConversationsResponse{
		Window:        windowResponse(q.window),
		By:            q.read.By,
		Conversations: emptyIfNil(conversations),
	})
}

// flowTopQuery parses ?range=, ?by= and ?limit=, or writes the rejection and
// reports false.
func (s *Server) flowTopQuery(w http.ResponseWriter, r *http.Request) (flowTopQuery, bool) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)
	params := r.URL.Query()

	by := flows.RankBytes
	switch raw := params.Get("by"); raw {
	case "", string(flows.RankBytes):
	case string(flows.RankPackets):
		by = flows.RankPackets
	default:
		sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
			ErrCodeValidation, localizer.T("errors.flows.invalidRank"), "")
		return flowTopQuery{}, false
	}

	limit := defaultFlowTopLimit
	if raw := params.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > maxFlowTopLimit {
			sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
				ErrCodeValidation, localizer.T("errors.flows.invalidLimit"), "")
			return flowTopQuery{}, false
		}
		limit = n
	}

	clientID, ok := s.callerClient(w, r)
	if !ok {
		return flowTopQuery{}, false
	}
	window, ok := s.historyWindow(w, r, resolveHistoryWindow)
	if !ok {
		return flowTopQuery{}, false
	}
	return flowTopQuery{
		read: flows.Query{
			ClientID: clientID,
			Tier:     flowTierFor(window),
			From:     window.From,
			To:       window.To,
			By:       by,
			Limit:    limit,
		},
		window: window,
	}, true
}

// flowTierFor maps a resolved window onto the flow table that answers it.
func flowTierFor(w historyWindow) flows.Tier {
	switch {
	case w.Source == historySourceRaw:
		return flows.TierRaw
	case w.Resolution == historyResolutionDaily:
		return flows.TierDaily
	default:
		return flows.TierHourly
	}
}

func flowReadFailed(w http.ResponseWriter, r *http.Request, err error) {
	logger := logging.FromContext(r.Context())
	logger.ErrorContext(r.Context(), "Failed to read flows", "error", err)
	sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
		ErrCodeInternal, i18n.FromRequest(r).T("errors.flows.readFailed"), "")
}
