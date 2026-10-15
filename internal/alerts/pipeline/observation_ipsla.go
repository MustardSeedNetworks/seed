package pipeline

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/polling/observation"
)

// The rttMonLatestRttOperSense values the rules tell apart. Every value other
// than these two is a run that did not reach its target.
const (
	slaSenseOK            = 1
	slaSenseOverThreshold = 3
)

// ipslaObsPayload mirrors the subset of the ip_sla Observation the rules read.
type ipslaObsPayload struct {
	Operations []struct {
		Index       int    `json:"Index"`
		Tag         string `json:"Tag"`
		Target      string `json:"Target"`
		ThresholdMs int    `json:"ThresholdMs"`
		Latest      *struct {
			Sense     int    `json:"Sense"`
			RTTMs     uint32 `json:"RTTMs"`
			RunUptime uint32 `json:"RunUptime"`
		} `json:"Latest"`
	} `json:"Operations"`
}

// slaRun is the latest run of one operation as a previous poll saw it.
type slaRun struct {
	runUptime uint32
	sense     int
}

func (r slaRun) completed() bool {
	return r.sense == slaSenseOK || r.sense == slaSenseOverThreshold
}

// decodeIPSLA reads an ip_sla observation into each operation's latest run,
// keyed by rttMonCtrlAdminIndex. An operation that has never run is absent.
func decodeIPSLA(obs *observation.SNMPObservation) (ipslaObsPayload, map[int]slaRun, bool) {
	var pay ipslaObsPayload
	if err := json.Unmarshal([]byte(obs.PayloadJSON), &pay); err != nil {
		return pay, nil, false
	}
	runs := make(map[int]slaRun, len(pay.Operations))
	for _, op := range pay.Operations {
		if op.Latest != nil {
			runs[op.Index] = slaRun{runUptime: op.Latest.RunUptime, sense: op.Latest.Sense}
		}
	}
	return pay, runs, true
}

func (p *ObservationPipeline) lookupIPSLA(key string) (map[int]slaRun, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	runs, ok := p.slaState[key]
	return runs, ok
}

func (p *ObservationPipeline) recordIPSLA(key string, runs map[int]slaRun) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.slaState[key] = runs
}

func (p *ObservationPipeline) primeIPSLA(obs *observation.SNMPObservation) {
	if _, runs, ok := decodeIPSLA(obs); ok {
		p.recordIPSLA("ipsla/"+obs.TargetID, runs)
	}
}

// evaluateIPSLA compares each operation's latest run with the one the
// previous poll saw. Only a new run (rttMonLatestRttOperTime moved) is judged,
// and only its change of state alerts: a run failing after one that completed
// raises RuleIPSLAFailed, and a completed run over the device's threshold
// after one within it raises RuleIPSLAOverThreshold. An operation that stays
// down raises nothing more, and the first run seed sees sets the baseline.
func (p *ObservationPipeline) evaluateIPSLA(
	ctx context.Context, obs *observation.SNMPObservation,
) int {
	pay, cur, ok := decodeIPSLA(obs)
	if !ok {
		return 0
	}
	key := "ipsla/" + obs.TargetID
	prev, seen := p.lookupIPSLA(key)
	p.recordIPSLA(key, cur)
	if !seen {
		return 0
	}

	count := 0
	for _, op := range pay.Operations {
		run, ran := cur[op.Index]
		last, known := prev[op.Index]
		if !ran || !known || run.runUptime == last.runUptime {
			continue
		}
		name := op.Tag
		if name == "" {
			name = fmt.Sprintf("operation %d", op.Index)
		}
		var rule string
		var alert *alerts.Alert
		switch {
		case last.completed() && !run.completed():
			rule = alerts.RuleIPSLAFailed
			alert = &alerts.Alert{
				Type:     alerts.TypeConnectivity,
				Severity: alerts.SeverityError,
				Title:    fmt.Sprintf("IP SLA %s on %s failed", name, obs.TargetID),
				Message: fmt.Sprintf("The run toward %s did not complete (rttMonLatestRttOperSense %d)",
					op.Target, run.sense),
			}
		case last.sense == slaSenseOK && run.sense == slaSenseOverThreshold:
			rule = alerts.RuleIPSLAOverThreshold
			alert = &alerts.Alert{
				Type:     alerts.TypePerformance,
				Severity: alerts.SeverityWarning,
				Title:    fmt.Sprintf("IP SLA %s on %s over threshold", name, obs.TargetID),
				Message: fmt.Sprintf("The run toward %s took %d ms, over its %d ms threshold",
					op.Target, op.Latest.RTTMs, op.ThresholdMs),
			}
		default:
			continue
		}
		alert.Source = obs.TargetID
		alert.Metadata = evidenceJSON(alerts.IPSLAEvidence{
			Index: op.Index, Tag: op.Tag, Target: op.Target,
			Sense: run.sense, RTTMs: op.Latest.RTTMs, ThresholdMs: op.ThresholdMs,
		})
		if p.fire(ctx, rule, fmt.Sprintf("%s/%d", key, op.Index), alert) {
			count++
		}
	}
	return count
}
