package api

// POST /api/v1/polling-targets/{targetId}/walk — the MIB browser (P-E5). An
// operator names any subtree, by number or by MIB name, and gets back every
// varbind under it with the object it belongs to. The walk authenticates with
// the target's stored credentials, the same ones the poller uses.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/mibdb"
	snmppoller "github.com/MustardSeedNetworks/seed/internal/polling/snmp"
)

// ErrCodeSNMPFailed is a walk the device did not complete: no answer, a wrong
// community, or an agent error.
const ErrCodeSNMPFailed = "SNMP_FAILED"

const (
	walkDefaultLimit = 500
	walkMaxLimit     = 5000
	// walkTimeout bounds one walk. A healthy agent returns walkMaxLimit
	// varbinds in about a hundred GETBULK round trips, well inside it.
	walkTimeout = 30 * time.Second
)

var numericOID = regexp.MustCompile(`^\d+(\.\d+)*$`)

// SNMPWalkRequest names the subtree to walk. OID is numeric
// ("1.3.6.1.2.1.2.2") or a MIB name with an optional instance ("ifTable",
// "sysDescr.0"). Limit defaults to 500 rows and cannot exceed 5000.
type SNMPWalkRequest struct {
	OID   string `json:"oid"`
	Limit int    `json:"limit,omitempty"`
}

// SNMPWalkRow is one varbind. Name is the longest-matching known object and
// Instance the arcs after it; both are absent for an object the MIB database
// does not know. Value is the text of an OCTET STRING when it is printable and
// its colon-separated hex bytes when it is not, and the decimal or dotted form of anything
// else.
type SNMPWalkRow struct {
	OID      string `json:"oid"`
	Name     string `json:"name,omitempty"`
	Instance string `json:"instance,omitempty"`
	Value    string `json:"value"`
}

// SNMPWalkResponse is one walk. OID is the numeric root walked, and Truncated
// says the subtree held more rows than the limit.
type SNMPWalkResponse struct {
	TargetID  string        `json:"targetId"`
	OID       string        `json:"oid"`
	Rows      []SNMPWalkRow `json:"rows"`
	Truncated bool          `json:"truncated"`
}

func (s *Server) handlePollingTargetWalk(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	clientID, ok := s.callerClient(w, r)
	if !ok {
		return
	}
	var req SNMPWalkRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, ErrCodeValidation, "invalid JSON body: "+err.Error())
		return
	}
	limit, err := walkLimit(req.Limit)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, ErrCodeValidation, err.Error())
		return
	}
	if s.mibDB == nil || s.snmpBrowser == nil {
		writeError(w, r, http.StatusServiceUnavailable, ErrCodeServiceUnavail,
			"SNMP browsing is not available: no MIB database or credential keyring")
		return
	}
	root, err := s.mibDB.ResolveOIDName(strings.TrimSpace(req.OID))
	if err != nil || !numericOID.MatchString(root) {
		writeError(w, r, http.StatusBadRequest, ErrCodeValidation,
			"'oid' must be a numeric OID or a known MIB object name")
		return
	}
	target, err := s.pollingTargets.Get(r.Context(), clientID, r.PathValue("targetId"))
	if err != nil {
		writePollingError(w, r, err, "Failed to load target")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), walkTimeout)
	defer cancel()
	vbs, truncated, err := s.snmpBrowser.Walk(ctx, target, root, limit)
	switch {
	case errors.Is(err, snmppoller.ErrCredentialsUnresolved):
		writeError(w, r, http.StatusConflict, ErrCodeConflict, err.Error())
		return
	case err != nil:
		logger.WarnContext(r.Context(), "snmp walk failed", "target_id", target.ID, "oid", root, "error", err)
		writeError(w, r, http.StatusBadGateway, ErrCodeSNMPFailed, err.Error())
		return
	}

	rows := make([]SNMPWalkRow, 0, len(vbs))
	for _, vb := range vbs {
		row := SNMPWalkRow{OID: vb.OID, Value: walkValue(vb.Value)}
		entry, instance, lookupErr := s.mibDB.LookupNumeric(r.Context(), vb.OID)
		switch {
		case lookupErr == nil:
			row.Name, row.Instance = entry.Name, instance
		case !errors.Is(lookupErr, mibdb.ErrNotFound):
			logger.ErrorContext(r.Context(), "mib lookup failed", "error", lookupErr)
			writeError(w, r, http.StatusInternalServerError, ErrCodeInternal, "MIB lookup failed")
			return
		}
		rows = append(rows, row)
	}
	sendJSONResponse(w, logger, http.StatusOK, SNMPWalkResponse{
		TargetID: target.ID, OID: root, Rows: rows, Truncated: truncated,
	})
}

func walkLimit(requested int) (int, error) {
	switch {
	case requested == 0:
		return walkDefaultLimit, nil
	case requested < 0 || requested > walkMaxLimit:
		return 0, fmt.Errorf("'limit' must be between 1 and %d", walkMaxLimit)
	}
	return requested, nil
}

// walkValue renders a decoded varbind value. gosnmp hands back OCTET STRINGs
// as bytes, OBJECT IDENTIFIERs and IpAddresses as strings, and the integer
// types as Go integers; NoSuchObject and NoSuchInstance arrive as nil.
func walkValue(v any) string {
	switch val := v.(type) {
	case nil:
		return ""
	case []byte:
		if printable(val) {
			return string(val)
		}
		return strings.ReplaceAll(fmt.Sprintf("% x", val), " ", ":")
	case string:
		return strings.TrimPrefix(val, ".")
	}
	return fmt.Sprint(v)
}

// printable reports whether an OCTET STRING reads as text. A MAC address or a
// bitmap is binary and renders as hex instead.
func printable(b []byte) bool {
	if !utf8.Valid(b) {
		return false
	}
	for _, r := range string(b) {
		if !unicode.IsPrint(r) && !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}
