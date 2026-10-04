package orchestrator

import (
	"context"
	"fmt"

	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/iftable"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/sink"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/ifrate"
)

// ratingSink persists every observation through the embedded Sink and, for
// if_table, also rates the interface counters against the target's previous
// poll and stores the rates.
type ratingSink struct {
	*sink.Sink

	rater *ifrate.Rater
	rates ifrate.Store
}

// PublishIfTable implements [iftable.Publisher].
func (s ratingSink) PublishIfTable(ctx context.Context, obs iftable.Observation) error {
	if err := s.Sink.PublishIfTable(ctx, obs); err != nil {
		return err
	}
	rates := s.rater.Observe(snapshot(obs))
	if err := s.rates.RecordInterfaceRates(ctx, rates); err != nil {
		return fmt.Errorf("orchestrator: record interface rates: %w", err)
	}
	return nil
}

func snapshot(obs iftable.Observation) ifrate.Snapshot {
	readings := make([]ifrate.Reading, len(obs.Rows))
	for i, row := range obs.Rows {
		c := row.Counters
		readings[i] = ifrate.Reading{
			IfIndex:       row.IfIndex,
			InOctets:      c.InOctets,
			OutOctets:     c.OutOctets,
			WideOctets:    c.HCOctets,
			InErrors:      c.InErrors,
			OutErrors:     c.OutErrors,
			InDiscards:    c.InDiscards,
			OutDiscards:   c.OutDiscards,
			Discontinuity: c.Discontinuity,
		}
	}
	return ifrate.Snapshot{
		ClientID:  obs.ClientID,
		TargetID:  obs.TargetID,
		At:        obs.ObservedAt,
		SysUpTime: obs.SysUpTime,
		Readings:  readings,
	}
}
