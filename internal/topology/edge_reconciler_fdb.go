package topology

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/polling/observation"
)

// fdbKind is the observation kind the forwarding-database pass reads.
const fdbKind = "fdb"

// fdbHighWaterKey is deliberately separate from edgeHighWaterKey.
// The neighbor-protocol mark is already far ahead on any install
// that has been polling, so sharing it would skip every fdb
// observation already on disk and leave the access layer blank
// until the next poll of every switch.
const fdbHighWaterKey = "topology.edge.fdb.high_water"

// fdbStatusLearned is dot1qTpFdbStatus "learned" (RFC 4188). Self
// and mgmt rows are the switch's own addresses, not something
// attached to the port.
const fdbStatusLearned = 3

// fdbPayload mirrors the fdb.Observation JSON shape.
type fdbPayload struct {
	Entries []struct {
		MACAddress string `json:"MACAddress"`
		BridgePort uint32 `json:"BridgePort"`
		IfIndex    uint32 `json:"IfIndex"`
		Status     int    `json:"Status"`
	} `json:"Entries"`
}

// reconcileFDB turns forwarding-database observations into edges to
// the endpoints behind access ports. Endpoints run no discovery
// protocol, so the FDB is the only evidence that attaches them to a
// switch port; the neighbor-protocol pass above can never see them.
// Returns the number of links written and the newest ObservedAt.
func (r *EdgeReconciler) reconcileFDB(ctx context.Context) (int, error) {
	since, err := r.loadHighWaterAt(ctx, fdbHighWaterKey)
	if err != nil {
		return 0, fmt.Errorf("load fdb high-water: %w", err)
	}
	observations, err := r.obs.List(ctx, observation.ListOptions{
		Kind:  fdbKind,
		Since: since,
		Limit: defaultBatchLimit,
	})
	if err != nil {
		return 0, fmt.Errorf("list fdb observations: %w", err)
	}

	var maxAt time.Time
	count := 0
	for _, obs := range observations {
		if obs.ObservedAt.After(maxAt) {
			maxAt = obs.ObservedAt
		}
		sourceNodeID, lookupErr := r.store.NodeIDForTarget(ctx, obs.ClientID, obs.TargetID)
		if lookupErr != nil {
			if !errors.Is(lookupErr, ErrTopologyNodeNotFound) {
				r.logger.WarnContext(ctx, "fdb: source node lookup failed",
					"target_id", obs.TargetID, "error", lookupErr)
			}
			continue
		}
		count += r.applyFDBObservation(ctx, obs, sourceNodeID)
	}
	if !maxAt.IsZero() {
		if saveErr := r.saveHighWaterAt(ctx, fdbHighWaterKey, maxAt); saveErr != nil {
			return count, fmt.Errorf("save fdb high-water: %w", saveErr)
		}
	}
	return count, nil
}

// applyFDBObservation writes one edge per access port. A port is an
// access port when exactly one MAC is learned on it and no neighbor
// protocol has already claimed it; anything else is an uplink or a
// trunk, where the MACs behind it say nothing about what is plugged
// in. Returns the count of edges that landed.
func (r *EdgeReconciler) applyFDBObservation(
	ctx context.Context,
	obs *observation.SNMPObservation,
	sourceNodeID string,
) int {
	var p fdbPayload
	if err := json.Unmarshal([]byte(obs.PayloadJSON), &p); err != nil {
		r.logger.WarnContext(ctx, "fdb: decode payload failed",
			"target_id", obs.TargetID, "error", err)
		return 0
	}

	// dot1qTpFdbTable is per-VLAN: one MAC on a trunk shows up once
	// per VLAN it is tagged in, so the MACs are deduplicated before
	// the port is judged, and the judgement happens before any
	// lookup — a core switch has thousands of MACs on a handful of
	// trunk ports and none of them should reach the database.
	macsByPort := map[uint32]map[string]uint32{}
	for _, e := range p.Entries {
		if e.Status != fdbStatusLearned || e.IfIndex == 0 || e.MACAddress == "" {
			continue
		}
		if macsByPort[e.IfIndex] == nil {
			macsByPort[e.IfIndex] = map[string]uint32{}
		}
		macsByPort[e.IfIndex][e.MACAddress] = e.BridgePort
	}

	claimedPorts, peers := r.neighborClaims(ctx, sourceNodeID)

	count := 0
	for _, ifIndex := range sortedPorts(macsByPort) {
		macs := macsByPort[ifIndex]
		if len(macs) != 1 {
			continue
		}
		if claimedPorts[ifIndexLabel(ifIndex)] {
			continue
		}
		for mac, bridgePort := range macs {
			count += r.linkFDBEndpoint(ctx, obs, sourceNodeID, ifIndex, bridgePort, mac, peers)
		}
	}
	return count
}

// linkFDBEndpoint resolves one learned MAC to a node and writes the
// edge. Returns 1 when a link landed.
func (r *EdgeReconciler) linkFDBEndpoint(
	ctx context.Context,
	obs *observation.SNMPObservation,
	sourceNodeID string,
	ifIndex, bridgePort uint32,
	mac string,
	peers map[string]bool,
) int {
	remoteNodeID, remoteIface, lookupErr := r.store.NodeForMAC(ctx, obs.ClientID, mac)
	if lookupErr != nil {
		// The MAC belongs to a device seed has never polled. V1.0
		// draws edges between known nodes only, same as the
		// neighbor-protocol pass.
		return 0
	}
	if remoteNodeID == sourceNodeID || peers[remoteNodeID] {
		// The switch's own MAC, or a peer a neighbor protocol has
		// already described in more detail than the FDB can.
		return 0
	}
	evidence, _ := json.Marshal(map[string]string{
		"kind":          fdbKind,
		"source_target": obs.TargetID,
		"remote_mac":    mac,
		"bridge_port":   strconv.FormatUint(uint64(bridgePort), 10),
	})
	link := &Link{
		ID:              linkIDFor(sourceNodeID, "", remoteNodeID, ""),
		SourceNodeID:    sourceNodeID,
		TargetNodeID:    remoteNodeID,
		SourceInterface: ifIndexLabel(ifIndex),
		TargetInterface: remoteIface,
		LinkType:        fdbKind,
		Status:          "up",
		LastSeen:        obs.ObservedAt,
		EvidenceJSON:    string(evidence),
	}
	if upsertErr := r.store.UpsertLink(ctx, link); upsertErr != nil {
		r.logger.WarnContext(ctx, "fdb: link upsert failed",
			"link_id", link.ID, "error", upsertErr)
		return 0
	}
	return 1
}

// neighborClaims returns the local ports of nodeID that a neighbor
// protocol has already described, and the nodes on the far end of
// those edges. Both are how an uplink whose FDB happens to hold a
// single MAC is told apart from an access port.
func (r *EdgeReconciler) neighborClaims(
	ctx context.Context, nodeID string,
) (map[string]bool, map[string]bool) {
	ports, peers := map[string]bool{}, map[string]bool{}
	links, err := r.store.ListLinks(ctx, nodeID)
	if err != nil {
		r.logger.WarnContext(ctx, "fdb: list existing links failed",
			"node_id", nodeID, "error", err)
		return ports, peers
	}
	for _, l := range links {
		if l.LinkType == fdbKind {
			continue
		}
		switch nodeID {
		case l.SourceNodeID:
			ports[l.SourceInterface] = true
			peers[l.TargetNodeID] = true
		case l.TargetNodeID:
			ports[l.TargetInterface] = true
			peers[l.SourceNodeID] = true
		}
	}
	return ports, peers
}

func sortedPorts(byPort map[uint32]map[string]uint32) []uint32 {
	out := make([]uint32, 0, len(byPort))
	for p := range byPort {
		out = append(out, p)
	}
	slices.Sort(out)
	return out
}
