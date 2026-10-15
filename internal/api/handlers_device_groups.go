package api

// Device group endpoints (P-D10):
//
//   GET    /api/v1/device-groups              list
//   POST   /api/v1/device-groups              create
//   GET    /api/v1/device-groups/{groupId}    fetch one
//   PUT    /api/v1/device-groups/{groupId}    full update, members included
//   DELETE /api/v1/device-groups/{groupId}    delete (409 while an alert rule names it)
//   GET    /api/v1/device-groups/{groupId}/summary  the group's dashboard read
//
// A group is a set of the caller's polling targets, scoped to the client on the
// session claim as /polling-targets is. Mutations are operator+, the same as
// the targets and the alert rules a group scopes.

import (
	"errors"
	"net/http"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"

	"github.com/MustardSeedNetworks/seed/internal/devicegroups"
	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

const deviceGroupsPath = APIVersionPrefix + "/device-groups"

// DeviceGroupRequest is the POST and PUT body. Members are polling target ids;
// PUT replaces the whole list.
type DeviceGroupRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Members     []string `json:"members"`
}

// DeviceGroupResponse is one device group on the wire. Members are polling
// target ids, sorted. Timestamps are RFC 3339 UTC.
type DeviceGroupResponse struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Members     []string `json:"members"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
}

// DeviceGroupSummaryResponse is the group dashboard read: each member's last
// poll and the members' unresolved alerts. An alert is a member's when its
// source is the member's polling target id or names its address ("host:port"
// for syslog and traps).
type DeviceGroupSummaryResponse struct {
	ID         string                      `json:"id"`
	Name       string                      `json:"name"`
	Members    []DeviceGroupMemberResponse `json:"members"`
	OpenAlerts DeviceGroupAlertCounts      `json:"openAlerts"`
}

// DeviceGroupMemberResponse is one member and how many unresolved alerts are
// its own. Members are ordered by name.
type DeviceGroupMemberResponse struct {
	Target     PollingTargetResponse `json:"target"`
	OpenAlerts int                   `json:"openAlerts"`
}

// DeviceGroupAlertCounts is the members' unresolved alerts by severity. Total
// counts each alert once, including one of a severity not named here.
type DeviceGroupAlertCounts struct {
	Total    int `json:"total"`
	Critical int `json:"critical"`
	Error    int `json:"error"`
	Warning  int `json:"warning"`
	Info     int `json:"info"`
}

// DeviceGroupListResponse is the GET /device-groups envelope.
type DeviceGroupListResponse struct {
	Count  int                   `json:"count"`
	Groups []DeviceGroupResponse `json:"groups"`
}

func (s *Server) deviceGroupRoutes() []route.Route {
	return []route.Route{
		{
			Path:    APIVersionPrefix + "/device-groups",
			Handler: s.handleDeviceGroups,
			Methods: []string{http.MethodGet, http.MethodPost},
			Scope:   roles.Operator,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/device-groups/{groupId}",
			Handler: s.handleDeviceGroupByID,
			Methods: []string{http.MethodGet, http.MethodPut, http.MethodDelete},
			Scope:   roles.Operator,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/device-groups/{groupId}/summary",
			Handler: s.handleDeviceGroupSummary,
			Methods: []string{http.MethodGet},
			Auth:    true,
		},
	}
}

func (s *Server) handleDeviceGroups(w http.ResponseWriter, r *http.Request) {
	clientID, ok := s.callerClient(w, r)
	if !ok {
		return
	}
	if r.Method == http.MethodPost {
		s.saveDeviceGroup(w, r, clientID, "")
		return
	}
	groups, err := s.deviceGroups.List(r.Context(), clientID)
	if err != nil {
		writeDeviceGroupError(w, r, err)
		return
	}
	out := make([]DeviceGroupResponse, 0, len(groups))
	for _, g := range groups {
		out = append(out, encodeDeviceGroup(g))
	}
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK,
		DeviceGroupListResponse{Count: len(out), Groups: out})
}

func (s *Server) handleDeviceGroupByID(w http.ResponseWriter, r *http.Request) {
	clientID, ok := s.callerClient(w, r)
	if !ok {
		return
	}
	id := r.PathValue("groupId")
	switch r.Method {
	case http.MethodPut:
		s.saveDeviceGroup(w, r, clientID, id)
	case http.MethodDelete:
		if err := s.deviceGroups.Delete(r.Context(), clientID, id); err != nil {
			writeDeviceGroupError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		g, err := s.deviceGroups.Get(r.Context(), clientID, id)
		if err != nil {
			writeDeviceGroupError(w, r, err)
			return
		}
		sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK, encodeDeviceGroup(g))
	}
}

func (s *Server) handleDeviceGroupSummary(w http.ResponseWriter, r *http.Request) {
	clientID, ok := s.callerClient(w, r)
	if !ok {
		return
	}
	sum, err := s.deviceGroups.Summary(r.Context(), clientID, r.PathValue("groupId"))
	if err != nil {
		writeDeviceGroupError(w, r, err)
		return
	}
	members := make([]DeviceGroupMemberResponse, 0, len(sum.Members))
	for _, m := range sum.Members {
		members = append(members, DeviceGroupMemberResponse{
			Target:     encodePollingTarget(&m.Target),
			OpenAlerts: m.OpenAlerts,
		})
	}
	sendJSONResponse(w, logging.FromContext(r.Context()), http.StatusOK, DeviceGroupSummaryResponse{
		ID:         sum.Group.ID,
		Name:       sum.Group.Name,
		Members:    members,
		OpenAlerts: DeviceGroupAlertCounts(sum.OpenAlerts),
	})
}

// saveDeviceGroup creates the group when id is blank and replaces it otherwise.
func (s *Server) saveDeviceGroup(w http.ResponseWriter, r *http.Request, clientID, id string) {
	var in DeviceGroupRequest
	if !decodeJSONStrict(w, r, &in, MaxBodySizeJSON) {
		return
	}
	g := devicegroups.Group{
		ID:          id,
		ClientID:    clientID,
		Name:        in.Name,
		Description: in.Description,
		Members:     in.Members,
	}
	var err error
	status := http.StatusOK
	if id == "" {
		g, err = s.deviceGroups.Create(r.Context(), g)
		status = http.StatusCreated
	} else {
		g, err = s.deviceGroups.Update(r.Context(), g)
	}
	if err != nil {
		writeDeviceGroupError(w, r, err)
		return
	}
	if id == "" {
		w.Header().Set("Location", deviceGroupsPath+"/"+g.ID)
	}
	sendJSONResponse(w, logging.FromContext(r.Context()), status, encodeDeviceGroup(g))
}

// ownGroup checks that a request's groupId names one of the caller's device
// groups, writing the denial and reporting false when it does not. Alert rules
// and scheduled reports are install-wide, so without this an operator could
// scope one by another client's group. An empty groupId is unscoped.
func (s *Server) ownGroup(w http.ResponseWriter, r *http.Request, groupID string) bool {
	if groupID == "" {
		return true
	}
	clientID, ok := s.callerClient(w, r)
	if !ok {
		return false
	}
	if _, err := s.deviceGroups.Get(r.Context(), clientID, groupID); err != nil {
		if errors.Is(err, devicegroups.ErrNotFound) {
			writeError(w, r, http.StatusBadRequest, ErrCodeValidation, "groupId does not name a device group")
			return false
		}
		writeDeviceGroupError(w, r, err)
		return false
	}
	return true
}

func writeDeviceGroupError(w http.ResponseWriter, r *http.Request, err error) {
	if ve, ok := errors.AsType[*devicegroups.ValidationError](err); ok {
		writeError(w, r, http.StatusBadRequest, ErrCodeValidation, ve.Msg)
		return
	}
	switch {
	case errors.Is(err, devicegroups.ErrNotFound):
		writeError(w, r, http.StatusNotFound, ErrCodeNotFound, "Device group not found")
	case errors.Is(err, devicegroups.ErrUnknownMember):
		writeError(w, r, http.StatusBadRequest, ErrCodeValidation, "A member is not one of your polling targets")
	case errors.Is(err, devicegroups.ErrNameTaken):
		writeError(w, r, http.StatusConflict, ErrCodeConflict, "A device group with this name already exists")
	case errors.Is(err, devicegroups.ErrInUse):
		writeError(w, r, http.StatusConflict, ErrCodeConflict,
			"An alert rule or scheduled report is scoped to this device group; change or delete it first")
	case errors.Is(err, devicegroups.ErrUnavailable):
		writeError(w, r, http.StatusServiceUnavailable, ErrCodeServiceUnavail, "Database not initialized")
	default:
		logging.FromContext(r.Context()).ErrorContext(r.Context(), "device group request failed", "error", err)
		writeError(w, r, http.StatusInternalServerError, ErrCodeInternal, "Device group request failed")
	}
}

func encodeDeviceGroup(g devicegroups.Group) DeviceGroupResponse {
	members := g.Members
	if members == nil {
		members = []string{}
	}
	return DeviceGroupResponse{
		ID:          g.ID,
		Name:        g.Name,
		Description: g.Description,
		Members:     members,
		CreatedAt:   formatTime(g.CreatedAt),
		UpdatedAt:   formatTime(g.UpdatedAt),
	}
}
