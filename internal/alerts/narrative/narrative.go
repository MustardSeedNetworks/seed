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
	"github.com/MustardSeedNetworks/seed/internal/timeseries/ifrate"
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

// Text is a narrative rendered in one language: the shape the inbox serves
// and every delivery channel sends.
type Text struct {
	Summary   string   `json:"summary"`
	Evidence  []string `json:"evidence"`
	NextCheck string   `json:"nextCheck"`
}

// Render renders n with t, a localizer's TWithData.
func (n Narrative) Render(t func(key string, data map[string]any) string) Text {
	evidence := make([]string, 0, len(n.Evidence))
	for _, m := range n.Evidence {
		evidence = append(evidence, t(m.Key, m.Data))
	}
	return Text{
		Summary:   t(n.Summary.Key, n.Summary.Data),
		Evidence:  evidence,
		NextCheck: t(n.NextCheck.Key, n.NextCheck.Data),
	}
}

// Cluster is what Explain reads: the cause, the alerts whose RootCauseID
// names it, and what else Seed knows about where it happened.
type Cluster struct {
	Cause   *alerts.Alert
	Effects []*alerts.Alert
	// Device is the name the operator knows the cause's device by.
	Device string
	// Counters is the cause interface's error and discard rates over
	// CountersWindow up to the alert. Only an interface-down cause reads it.
	Counters ifrate.ErrorPeaks
}

// CountersWindow is how far back an interface's error and discard rates are
// read when it goes down: several polls at the 60 to 300 s intervals targets
// are polled at, short enough to describe the link as it was just before.
const CountersWindow = 15 * time.Minute

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

// Explain returns the narrative for the cluster rooted at c.Cause. It
// reports false when the cause is itself explained by another alert (that
// alert's narrative covers it), when its rule has no narrative (operator
// rules are open-ended), or when its evidence cannot be read.
func Explain(c Cluster) (Narrative, bool) {
	if c.Cause.RootCauseID != nil {
		return Narrative{}, false
	}
	switch c.Cause.Rule {
	case alerts.RuleInterfaceDown:
		return interfaceDown(c)
	case alerts.RuleBGPFlap:
		return bgpFlap(c.Cause, c.Device)
	case alerts.RuleStorageHigh, alerts.RuleStorageCritical:
		return storage(c.Cause, c.Device)
	default:
		return Narrative{}, false
	}
}

// CountersFor reports the interface whose error and discard rates Explain
// cites for cause, so the caller reads only what a narrative will use: the
// ifIndex of an interface-down cause that heads its cluster.
func CountersFor(cause *alerts.Alert) (uint32, bool) {
	var ev alerts.InterfaceDownEvidence
	if cause.Rule != alerts.RuleInterfaceDown || cause.RootCauseID != nil || !decode(cause, &ev) {
		return 0, false
	}
	return ev.IfIndex, true
}

func interfaceDown(c Cluster) (Narrative, bool) {
	cause := c.Cause
	var ev alerts.InterfaceDownEvidence
	if !decode(cause, &ev) {
		return Narrative{}, false
	}
	where := map[string]any{"interface": ev.IfName, "device": c.Device}

	summary := "interfaceDown.summary"
	var peers []alerts.BGPPeerEvidence
	var peerEvidence []Message
	for _, effect := range c.Effects {
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
	})}, counterEvidence(ev.IfName, c.Counters)...)
	evidence = append(evidence, peerEvidence...)

	next := operStatusName(ev.IfOperStatus)
	// A link that was corrupting frames before it dropped is failing
	// physically; one that dropped clean could as well have been unplugged or
	// shut at the far end, so it keeps the general check.
	if next == "down" && (c.Counters.InErrors > 0 || c.Counters.OutErrors > 0) {
		next = "downAfterErrors"
	}

	return Narrative{
		Summary:   msg(summary, with(where, "peers", len(peers))),
		Evidence:  evidence,
		NextCheck: msg("interfaceDown.next."+next, where),
	}, true
}

// counterEvidence states what the interface's error and discard counters did
// before it went down: the peak rate of each counter that moved, or that none
// did. With no rated poll in the window there is nothing to state.
func counterEvidence(ifName string, peaks ifrate.ErrorPeaks) []Message {
	if peaks.Polls == 0 {
		return nil
	}
	minutes := int(CountersWindow / time.Minute)
	var out []Message
	for _, counter := range []struct {
		name string
		peak float64
	}{
		{"ifInErrors", peaks.InErrors},
		{"ifOutErrors", peaks.OutErrors},
		{"ifInDiscards", peaks.InDiscards},
		{"ifOutDiscards", peaks.OutDiscards},
	} {
		if counter.peak > 0 {
			out = append(out, msg("interfaceDown.evidence.counterPeak", map[string]any{
				"counter":   counter.name,
				"interface": ifName,
				"rate":      formatRate(counter.peak),
				"minutes":   minutes,
			}))
		}
	}
	if out == nil {
		out = append(out, msg("interfaceDown.evidence.countersClean", map[string]any{
			"interface": ifName,
			"minutes":   minutes,
			"polls":     peaks.Polls,
		}))
	}
	return out
}

// formatRate keeps three significant figures, so one error in a five-minute
// poll still reads as a rate and not as 0.00, without printing a large rate
// in exponent form.
func formatRate(perSecond float64) string {
	const wholeFrom = 100
	if perSecond >= wholeFrom {
		return strconv.FormatFloat(perSecond, 'f', 0, 64)
	}
	return strconv.FormatFloat(perSecond, 'g', 3, 64)
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
