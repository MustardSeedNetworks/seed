package api

import (
	"net/http"
	"strconv"

	"github.com/MustardSeedNetworks/seed/internal/database"
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
	By      database.FlowRank     `json:"by"`
	Talkers []database.FlowTalker `json:"talkers"`
}

// FlowConversationsResponse is the top host pairs, per protocol, by traffic
// in both directions.
type FlowConversationsResponse struct {
	Window        HistoryWindowResponse       `json:"window"`
	By            database.FlowRank           `json:"by"`
	Conversations []database.FlowConversation `json:"conversations"`
}

// flowRoutes registers the top-N reads, the application signature table and
// the threat indicator list. The reads have no role gate and no feature gate: the collector that fills
// flow_records is Pro, so below Pro they answer with empty lists, over the
// window the tier retains. Editing the signature table or the indicator
// list is a persistent write, so it is operator-gated.
func (s *Server) flowRoutes() []route {
	get := []string{http.MethodGet}
	return []route{
		{
			path:    APIVersionPrefix + "/flows/top-talkers",
			handler: s.handleFlowTopTalkers,
			methods: get,
		},
		{
			path:    APIVersionPrefix + "/flows/top-conversations",
			handler: s.handleFlowTopConversations,
			methods: get,
		},
		{
			path:    APIVersionPrefix + "/flows/top-applications",
			handler: s.handleFlowTopApplications,
			methods: get,
		},
		{
			path:         APIVersionPrefix + "/flows/application-signatures",
			handler:      s.handleAppSignatures,
			methods:      []string{http.MethodGet, http.MethodPut, http.MethodDelete},
			minRole:      roles.Operator,
			maxBodyBytes: MaxBodySizeConfig,
		},
		{
			path:         APIVersionPrefix + "/flows/threat-indicators",
			handler:      s.handleFlowIndicators,
			methods:      []string{http.MethodGet, http.MethodPut},
			minRole:      roles.Operator,
			maxBodyBytes: MaxBodySizeJSON,
		},
	}
}

// flowTopQuery is a parsed top-N request.
type flowTopQuery struct {
	clientID string
	window   historyWindow
	tier     database.FlowTier
	by       database.FlowRank
	limit    int
}

// handleFlowTopTalkers serves GET /api/v1/flows/top-talkers.
func (s *Server) handleFlowTopTalkers(w http.ResponseWriter, r *http.Request) {
	q, ok := s.flowTopQuery(w, r)
	if !ok {
		return
	}
	talkers, err := s.db().FlowRecords().TopTalkers(r.Context(), q.clientID, q.tier,
		q.window.From, q.window.To, q.by, q.limit)
	if err != nil {
		flowReadFailed(w, r, err)
		return
	}
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK, FlowTalkersResponse{
		Window:  windowResponse(q.window),
		By:      q.by,
		Talkers: emptyIfNil(talkers),
	})
}

// handleFlowTopConversations serves GET /api/v1/flows/top-conversations.
func (s *Server) handleFlowTopConversations(w http.ResponseWriter, r *http.Request) {
	q, ok := s.flowTopQuery(w, r)
	if !ok {
		return
	}
	conversations, err := s.db().FlowRecords().TopConversations(r.Context(), q.clientID, q.tier,
		q.window.From, q.window.To, q.by, q.limit)
	if err != nil {
		flowReadFailed(w, r, err)
		return
	}
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK, FlowConversationsResponse{
		Window:        windowResponse(q.window),
		By:            q.by,
		Conversations: emptyIfNil(conversations),
	})
}

// flowTopQuery parses ?range=, ?by= and ?limit=, or writes the rejection and
// reports false.
func (s *Server) flowTopQuery(w http.ResponseWriter, r *http.Request) (flowTopQuery, bool) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)
	params := r.URL.Query()

	by := database.FlowRankBytes
	switch raw := params.Get("by"); raw {
	case "", string(database.FlowRankBytes):
	case string(database.FlowRankPackets):
		by = database.FlowRankPackets
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
		clientID: clientID,
		window:   window,
		tier:     flowTierFor(window),
		by:       by,
		limit:    limit,
	}, true
}

// flowTierFor maps a resolved window onto the flow table that answers it.
func flowTierFor(w historyWindow) database.FlowTier {
	switch {
	case w.Source == historySourceRaw:
		return database.FlowTierRaw
	case w.Resolution == historyResolutionDaily:
		return database.FlowTierDaily
	default:
		return database.FlowTierHourly
	}
}

func flowReadFailed(w http.ResponseWriter, r *http.Request, err error) {
	logger := logging.FromContext(r.Context())
	logger.ErrorContext(r.Context(), "Failed to read flows", "error", err)
	sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
		ErrCodeInternal, i18n.FromRequest(r).T("errors.flows.readFailed"), "")
}
