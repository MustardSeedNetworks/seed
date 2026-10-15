package pipeline

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/polling/observation"
)

// stpBridgeID mirrors the stp collector's BridgeID.
type stpBridgeID struct {
	Priority uint16 `json:"Priority"`
	Address  string `json:"Address"`
}

func (b stpBridgeID) String() string { return fmt.Sprintf("%d/%s", b.Priority, b.Address) }

// stpObsPayload mirrors the subset of the stp Observation the rules read.
type stpObsPayload struct {
	Root            stpBridgeID `json:"Root"`
	RootPort        int         `json:"RootPort"`
	TopologyChanges uint32      `json:"TopologyChanges"`
	Ports           []struct {
		Port    int `json:"Port"`
		IfIndex int `json:"IfIndex"`
		State   int `json:"State"`
	} `json:"Ports"`
}

// stpSnapshot is one target's previous spanning-tree view.
type stpSnapshot struct {
	root       stpBridgeID
	topChanges uint32
	portState  map[int]int
}

// decodeSTP reads an stp observation. It reports false when the payload is
// unreadable or the device runs no spanning tree, which has no root.
func decodeSTP(obs *observation.SNMPObservation) (stpObsPayload, stpSnapshot, bool) {
	var pay stpObsPayload
	if err := json.Unmarshal([]byte(obs.PayloadJSON), &pay); err != nil || pay.Root.Address == "" {
		return pay, stpSnapshot{}, false
	}
	snap := stpSnapshot{
		root:       pay.Root,
		topChanges: pay.TopologyChanges,
		portState:  make(map[int]int, len(pay.Ports)),
	}
	for _, port := range pay.Ports {
		snap.portState[port.Port] = port.State
	}
	return pay, snap, true
}

func (p *ObservationPipeline) lookupSTP(key string) (stpSnapshot, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	snap, ok := p.stpState[key]
	return snap, ok
}

func (p *ObservationPipeline) recordSTP(key string, snap stpSnapshot) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stpState[key] = snap
}

func (p *ObservationPipeline) primeSTP(obs *observation.SNMPObservation) {
	if _, snap, ok := decodeSTP(obs); ok {
		p.recordSTP("stp/"+obs.TargetID, snap)
	}
}

// evaluateSTP compares a switch's spanning-tree view with its previous poll.
// A new root bridge raises RuleSTPRootChange and nothing else: the
// topology-change counter always rises with it, and reporting both would
// bury the cause under its symptom. Otherwise a rise in dot1dStpTopChanges
// raises RuleSTPTopologyChange with the local ports that changed state. A
// fall means the agent restarted and its counter began again, so it is
// recorded without alerting.
func (p *ObservationPipeline) evaluateSTP(
	ctx context.Context, obs *observation.SNMPObservation,
) int {
	pay, cur, ok := decodeSTP(obs)
	if !ok {
		return 0
	}
	key := "stp/" + obs.TargetID
	prev, seen := p.lookupSTP(key)
	p.recordSTP(key, cur)
	if !seen {
		return 0
	}

	if cur.root != prev.root {
		alert := &alerts.Alert{
			Type:     alerts.TypeConnectivity,
			Severity: alerts.SeverityError,
			Title:    fmt.Sprintf("Spanning-tree root bridge changed on %s", obs.TargetID),
			Message:  fmt.Sprintf("Root bridge moved from %s to %s (root port %d)", prev.root, cur.root, pay.RootPort),
			Source:   obs.TargetID,
			Metadata: evidenceJSON(alerts.STPRootChangeEvidence{
				PreviousRoot: prev.root.String(), Root: cur.root.String(), RootPort: pay.RootPort,
			}),
		}
		if p.fire(ctx, alerts.RuleSTPRootChange, key, alert) {
			return 1
		}
		return 0
	}

	if cur.topChanges <= prev.topChanges {
		return 0
	}
	transitions := []alerts.STPPortTransition{}
	for _, port := range pay.Ports {
		if from, known := prev.portState[port.Port]; known && from != port.State {
			transitions = append(transitions, alerts.STPPortTransition{
				Port: port.Port, IfIndex: port.IfIndex, From: from, To: port.State,
			})
		}
	}
	changes := cur.topChanges - prev.topChanges
	alert := &alerts.Alert{
		Type:     alerts.TypeConnectivity,
		Severity: alerts.SeverityWarning,
		Title:    fmt.Sprintf("Spanning-tree topology change on %s", obs.TargetID),
		Message: fmt.Sprintf("dot1dStpTopChanges rose by %d to %d; %d local port(s) changed state",
			changes, cur.topChanges, len(transitions)),
		Source: obs.TargetID,
		Metadata: evidenceJSON(alerts.STPTopologyChangeEvidence{
			Changes: changes, TopologyChanges: cur.topChanges, Ports: transitions,
		}),
	}
	if p.fire(ctx, alerts.RuleSTPTopologyChange, key, alert) {
		return 1
	}
	return 0
}
