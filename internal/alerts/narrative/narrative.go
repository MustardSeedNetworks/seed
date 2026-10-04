// Package narrative explains a correlated cluster of alerts in plain language:
// what happened, on which device and interface, the evidence Seed holds for
// it, and the check an engineer should make next.
//
// Every narrative is a fixed rule over the alerts' typed evidence, so the same
// alerts always produce the same words and each sentence can be traced to a
// field Seed actually observed. A rule with nothing specific to say produces
// no narrative rather than a vague one: a confident wrong sentence in front of
// an engineer mid-incident is worse than none, the standard the correlation
// package already holds itself to.
//
// The package returns message keys and their data, not text. The caller
// renders them in the reader's language; the keys live under the api
// namespace's "narrative" section.
package narrative

import (
	"encoding/json"
	"maps"
	"math"
	"strconv"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/alerts/correlation"
)

// Message is one sentence: a locale key and the values it interpolates.
type Message struct {
	Key  string
	Data map[string]any
}

// Narrative explains one cluster: a cause and the alerts it explains.
type Narrative struct {
	// Summary says what happened and where.
	Summary Message
	// Evidence is what Seed observed, cause first, then each effect in the
	// order it was raised.
	Evidence []Message
	// NextCheck is the one thing to look at next.
	NextCheck Message
}

const keyPrefix = "api.narrative."

// RFC 2863 ifOperStatus values. 1 (up) never reaches here: the pipeline only
// raises RuleInterfaceDown on a transition away from it.
const (
	ifOperDown           = 2
	ifOperTesting        = 3
	ifOperDormant        = 5
	ifOperNotPresent     = 6
	ifOperLowerLayerDown = 7
)

// RFC 4273 bgpPeerState values. 6 (established) never reaches here: the
// pipeline only raises RuleBGPFlap on a transition away from it.
const (
	bgpIdle        = 1
	bgpConnect     = 2
	bgpActive      = 3
	bgpOpenSent    = 4
	bgpOpenConfirm = 5
)

// Explain returns the narrative for the cluster rooted at cause, given the
// alerts whose RootCauseID names it and the name the operator knows the
// cause's device by. It reports false when the cause is
// itself explained by another alert (that alert's narrative covers it), when
// its rule has no narrative (operator rules are open-ended), or when its
// evidence cannot be read.
func Explain(cause *alerts.Alert, effects []*alerts.Alert, device string) (Narrative, bool) {
	if cause.RootCauseID != nil {
		return Narrative{}, false
	}
	switch cause.Rule {
	case alerts.RuleInterfaceDown:
		return interfaceDown(cause, effects, device)
	case alerts.RuleBGPFlap:
		return bgpFlap(cause, device)
	case alerts.RuleStorageHigh, alerts.RuleStorageCritical:
		return storage(cause, device)
	default:
		return Narrative{}, false
	}
}

func interfaceDown(cause *alerts.Alert, effects []*alerts.Alert, device string) (Narrative, bool) {
	var ev alerts.InterfaceDownEvidence
	if !decode(cause, &ev) {
		return Narrative{}, false
	}
	where := map[string]any{"interface": ev.IfName, "device": device}

	summary := "interfaceDown.summary"
	var peers []alerts.BGPPeerEvidence
	var peerEvidence []Message
	for _, effect := range effects {
		var peer alerts.BGPPeerEvidence
		if effect.Rule != alerts.RuleBGPFlap || !decode(effect, &peer) {
			continue
		}
		peers = append(peers, peer)
		peerEvidence = append(peerEvidence, msg("interfaceDown.evidence.bgpPeer", map[string]any{
			"peer":    peer.RemoteAddr,
			"as":      peer.RemoteAS,
			"seconds": secondsBetween(cause.CreatedAt, effect.CreatedAt),
		}))
	}
	if len(peers) > 0 {
		summary = "interfaceDown.summaryWithBgp"
	}

	evidence := append([]Message{msg("interfaceDown.evidence.operStatus", map[string]any{
		"status":  operStatusName(ev.IfOperStatus),
		"ifIndex": ev.IfIndex,
		"time":    cause.CreatedAt.UTC().Format(time.RFC3339),
	})}, peerEvidence...)

	return Narrative{
		Summary:   msg(summary, with(where, "peers", len(peers))),
		Evidence:  evidence,
		NextCheck: msg("interfaceDown.next."+operStatusName(ev.IfOperStatus), where),
	}, true
}

func bgpFlap(cause *alerts.Alert, device string) (Narrative, bool) {
	var ev alerts.BGPPeerEvidence
	if !decode(cause, &ev) {
		return Narrative{}, false
	}
	where := map[string]any{"peer": ev.RemoteAddr, "as": ev.RemoteAS, "device": device}
	return Narrative{
		Summary: msg("bgpFlap.summary", with(where, "minutes",
			int(correlation.DefaultWindow/time.Minute))),
		Evidence: []Message{msg("bgpFlap.evidence.state", map[string]any{
			"state": bgpStateName(ev.State),
			"time":  cause.CreatedAt.UTC().Format(time.RFC3339),
		})},
		NextCheck: msg("bgpFlap.next."+bgpStateName(ev.State), where),
	}, true
}

func storage(cause *alerts.Alert, device string) (Narrative, bool) {
	var ev alerts.StorageEvidence
	if !decode(cause, &ev) {
		return Narrative{}, false
	}
	where := map[string]any{"filesystem": ev.Description, "device": device}
	return Narrative{
		Summary: msg("storage.summary", with(where, "percent",
			strconv.FormatFloat(ev.UsedPercent, 'f', 1, 64))),
		Evidence: []Message{msg("storage.evidence.usage", map[string]any{
			"used": ev.UsedBytes,
			"size": ev.SizeBytes,
			"time": cause.CreatedAt.UTC().Format(time.RFC3339),
		})},
		NextCheck: msg("storage.next", where),
	}, true
}

// operStatusName is the RFC 2863 enumeration label, which is also the key
// suffix of the matching next check. Engineers read these labels on the
// device's own CLI, so they stay untranslated.
func operStatusName(status int) string {
	switch status {
	case ifOperDown:
		return "down"
	case ifOperTesting:
		return "testing"
	case ifOperDormant:
		return "dormant"
	case ifOperNotPresent:
		return "notPresent"
	case ifOperLowerLayerDown:
		return "lowerLayerDown"
	default: // 4 is unknown; anything else is outside the MIB.
		return "unknown"
	}
}

// bgpStateName is the RFC 4273 enumeration label and next-check key suffix.
func bgpStateName(state int) string {
	switch state {
	case bgpIdle:
		return "idle"
	case bgpConnect:
		return "connect"
	case bgpActive:
		return "active"
	case bgpOpenSent:
		return "openSent"
	case bgpOpenConfirm:
		return "openConfirm"
	default:
		return "unknown"
	}
}

func decode(a *alerts.Alert, v any) bool {
	return json.Unmarshal([]byte(a.Metadata), v) == nil
}

func secondsBetween(from, to time.Time) int64 {
	return int64(math.Round(to.Sub(from).Seconds()))
}

func msg(key string, data map[string]any) Message {
	return Message{Key: keyPrefix + key, Data: data}
}

// with returns a copy of data with one more value, so the shared "where"
// map is never mutated between messages.
func with(data map[string]any, key string, value any) map[string]any {
	out := maps.Clone(data)
	out[key] = value
	return out
}
