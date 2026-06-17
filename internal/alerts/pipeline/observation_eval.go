package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/polling/observation"
)

func (p *ObservationPipeline) scanKind(
	ctx context.Context, kind string, since time.Time,
) (int, time.Time, error) {
	observations, err := p.obs.List(ctx, observation.ListOptions{
		Kind: kind, Since: since, Limit: defaultBatch,
	})
	if err != nil {
		return 0, time.Time{}, err
	}
	var maxAt time.Time
	count := 0
	for _, obs := range observations {
		if obs.ObservedAt.After(maxAt) {
			maxAt = obs.ObservedAt
		}
		switch kind {
		case "if_table":
			count += p.evaluateIfTable(ctx, obs)
		case "bgp4_mib":
			count += p.evaluateBGP(ctx, obs)
		case "host_resources":
			count += p.evaluateHostResources(ctx, obs)
		}
	}
	return count, maxAt, nil
}

// ifTableObsPayload mirrors the iftable Observation. Subset of
// fields needed for the delta check.
type ifTableObsPayload struct {
	Rows []struct {
		IfIndex uint32 `json:"IfIndex"`
		IfName  string `json:"IfName"`
		IfAdmin int    `json:"IfAdmin"`
		IfOper  int    `json:"IfOper"`
	} `json:"Rows"`
}

const ifOperUp = 1

// evaluateIfTable detects oper status transitions. Fires "interface
// down" alerts when admin=up AND oper changes from up to anything
// else. Up→up and any change from initial unknown state do NOT fire
// (we need a "previous up" to define a transition).
func (p *ObservationPipeline) evaluateIfTable(
	ctx context.Context, obs *observation.SNMPObservation,
) int {
	var pay ifTableObsPayload
	if err := json.Unmarshal([]byte(obs.PayloadJSON), &pay); err != nil {
		return 0
	}
	count := 0
	for _, row := range pay.Rows {
		key := fmt.Sprintf("if/%s/%d", obs.TargetID, row.IfIndex)
		prev := p.lookupIface(key)
		p.recordIface(key, row.IfOper)

		// Transition: previously up, now not. Admin-down doesn't
		// alert (operator intentionally disabled the port).
		if prev != ifOperUp || row.IfOper == ifOperUp || row.IfAdmin != ifOperUp {
			continue
		}
		alert := &alerts.Alert{
			Type:     alerts.TypeConnectivity,
			Severity: alerts.SeverityWarning,
			Title:    fmt.Sprintf("Interface %s down on %s", row.IfName, obs.TargetID),
			Message: fmt.Sprintf("ifOperStatus transitioned from up to %d (ifIndex=%d, admin=up)",
				row.IfOper, row.IfIndex),
			Source:   obs.TargetID,
			Metadata: obs.PayloadJSON,
		}
		if p.fire(ctx, "iface.down", key, alert) {
			count++
		}
	}
	return count
}

// bgpObsPayload mirrors the bgp4 Observation peers subset.
type bgpObsPayload struct {
	Peers []struct {
		RemoteAddr string `json:"RemoteAddr"`
		State      int    `json:"State"`
		RemoteAS   uint32 `json:"RemoteAS"`
	} `json:"Peers"`
}

const bgpStateEstablished = 6

// evaluateBGP detects peers leaving the Established state.
func (p *ObservationPipeline) evaluateBGP(
	ctx context.Context, obs *observation.SNMPObservation,
) int {
	var pay bgpObsPayload
	if err := json.Unmarshal([]byte(obs.PayloadJSON), &pay); err != nil {
		return 0
	}
	count := 0
	for _, peer := range pay.Peers {
		key := fmt.Sprintf("bgp/%s/%s", obs.TargetID, peer.RemoteAddr)
		prev := p.lookupBGP(key)
		p.recordBGP(key, peer.State)

		// Transition: was Established, now isn't.
		if prev != bgpStateEstablished || peer.State == bgpStateEstablished {
			continue
		}
		alert := &alerts.Alert{
			Type:     alerts.TypeConnectivity,
			Severity: alerts.SeverityError,
			Title:    fmt.Sprintf("BGP peer %s left Established on %s", peer.RemoteAddr, obs.TargetID),
			Message: fmt.Sprintf("Peer state transitioned from 6 (Established) to %d, AS%d",
				peer.State, peer.RemoteAS),
			Source:   obs.TargetID,
			Metadata: obs.PayloadJSON,
		}
		if p.fire(ctx, "bgp.flap", key, alert) {
			count++
		}
	}
	return count
}

// hostResObsPayload mirrors the host_resources Observation storage
// subset. SizeBytes / UsedBytes come from the collector already
// pre-multiplied by allocation_units.
type hostResObsPayload struct {
	Storage []struct {
		Index       uint32 `json:"Index"`
		Description string `json:"Description"`
		SizeBytes   uint64 `json:"SizeBytes"`
		UsedBytes   uint64 `json:"UsedBytes"`
	} `json:"Storage"`
}

// evaluateHostResources fires when a filesystem crosses 85% (warning)
// or 95% (critical). The delta state tracks the last-seen %, so we
// fire only on the upward crossing (not every poll while above the
// threshold).
func (p *ObservationPipeline) evaluateHostResources(
	ctx context.Context, obs *observation.SNMPObservation,
) int {
	var pay hostResObsPayload
	if err := json.Unmarshal([]byte(obs.PayloadJSON), &pay); err != nil {
		return 0
	}
	count := 0
	for _, st := range pay.Storage {
		if st.SizeBytes == 0 {
			continue
		}
		pct := percentMultiplier * float64(st.UsedBytes) / float64(st.SizeBytes)
		key := fmt.Sprintf("storage/%s/%d", obs.TargetID, st.Index)
		prev := p.lookupStorage(key)
		p.recordStorage(key, pct)

		// Upward crossings: prev below threshold and now at-or-above.
		// Two thresholds means two possible alerts per observation.
		if prev < storageFullPct && pct >= storageFullPct {
			alert := &alerts.Alert{
				Type:     alerts.TypeSystem,
				Severity: alerts.SeverityCritical,
				Title:    fmt.Sprintf("Filesystem %s critical on %s", st.Description, obs.TargetID),
				Message:  fmt.Sprintf("Usage crossed %.0f%%: %.1f%% of %d bytes", storageFullPct, pct, st.SizeBytes),
				Source:   obs.TargetID,
				Metadata: obs.PayloadJSON,
			}
			if p.fire(ctx, "storage.critical", key, alert) {
				count++
			}
		} else if prev < storageHighPct && pct >= storageHighPct {
			alert := &alerts.Alert{
				Type:     alerts.TypeSystem,
				Severity: alerts.SeverityWarning,
				Title:    fmt.Sprintf("Filesystem %s high on %s", st.Description, obs.TargetID),
				Message:  fmt.Sprintf("Usage crossed %.0f%%: %.1f%% of %d bytes", storageHighPct, pct, st.SizeBytes),
				Source:   obs.TargetID,
				Metadata: obs.PayloadJSON,
			}
			if p.fire(ctx, "storage.high", key, alert) {
				count++
			}
		}
	}
	return count
}

// fire writes an alert if the (ruleID, entityKey) fingerprint isn't
// suppressed. Returns true when the alert was actually written.
func (p *ObservationPipeline) fire(
	ctx context.Context, ruleID, entityKey string, alert *alerts.Alert,
) bool {
	now := p.now()
	fingerprint := fingerprintFor(ruleID, entityKey, alert.Source)
	suppressed, suppErr := p.suppress.IsSuppressed(ctx, fingerprint, now)
	if suppErr != nil {
		p.logger.WarnContext(ctx, "suppression check failed",
			"rule", ruleID, "key", entityKey, "error", suppErr)
	}
	if suppressed {
		return false
	}
	if err := p.alerts.Create(ctx, alert); err != nil {
		p.logger.WarnContext(ctx, "alert create failed",
			"rule", ruleID, "key", entityKey, "error", err)
		return false
	}
	if markErr := p.suppress.Mark(ctx, fingerprint, ruleID, entityKey, now.Add(p.suppression)); markErr != nil {
		p.logger.WarnContext(ctx, "suppression mark failed",
			"rule", ruleID, "key", entityKey, "error", markErr)
	}
	return true
}
