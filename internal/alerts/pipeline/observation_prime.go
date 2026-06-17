package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/MustardSeedNetworks/seed/internal/polling/observation"
)

// primeFromHistory reads recent observations and walks them through
// the same payload-decode logic the scan loop uses, but stops short
// of firing alerts. The point is to populate ifaceState / bgpState /
// storageState so the first real scan after restart compares against
// real previous values instead of zero.
func (p *ObservationPipeline) primeFromHistory(ctx context.Context) error {
	for _, kind := range observationKinds() {
		observations, err := p.obs.List(ctx, observation.ListOptions{
			Kind:  kind,
			Limit: p.replayDepth,
		})
		if err != nil {
			return fmt.Errorf("replay %s: %w", kind, err)
		}
		// Walk oldest-first so the most recent observation overwrites
		// older state — matches what the scan loop would have done.
		for _, obs := range slices.Backward(observations) {
			p.primeOne(kind, obs)
		}
	}
	return nil
}

// primeOne dispatches to the per-kind primer. Unknown kinds are
// silently skipped — they were filtered out by observationKinds()
// at the caller, but we double-check here for safety.
func (p *ObservationPipeline) primeOne(kind string, obs *observation.SNMPObservation) {
	switch kind {
	case "if_table":
		p.primeIfTable(obs)
	case "bgp4_mib":
		p.primeBGP(obs)
	case "host_resources":
		p.primeHostResources(obs)
	}
}

func (p *ObservationPipeline) primeIfTable(obs *observation.SNMPObservation) {
	var pay ifTableObsPayload
	if err := json.Unmarshal([]byte(obs.PayloadJSON), &pay); err != nil {
		return
	}
	for _, row := range pay.Rows {
		key := fmt.Sprintf("if/%s/%d", obs.TargetID, row.IfIndex)
		p.recordIface(key, row.IfOper)
	}
}

func (p *ObservationPipeline) primeBGP(obs *observation.SNMPObservation) {
	var pay bgpObsPayload
	if err := json.Unmarshal([]byte(obs.PayloadJSON), &pay); err != nil {
		return
	}
	for _, peer := range pay.Peers {
		key := fmt.Sprintf("bgp/%s/%s", obs.TargetID, peer.RemoteAddr)
		p.recordBGP(key, peer.State)
	}
}

func (p *ObservationPipeline) primeHostResources(obs *observation.SNMPObservation) {
	var pay hostResObsPayload
	if err := json.Unmarshal([]byte(obs.PayloadJSON), &pay); err != nil {
		return
	}
	for _, st := range pay.Storage {
		if st.SizeBytes == 0 {
			continue
		}
		pct := percentMultiplier * float64(st.UsedBytes) / float64(st.SizeBytes)
		key := fmt.Sprintf("storage/%s/%d", obs.TargetID, st.Index)
		p.recordStorage(key, pct)
	}
}
