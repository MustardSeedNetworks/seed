package orchestrator

import (
	"context"
	"fmt"

	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/iftable"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/ipsla"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/sink"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/ifrate"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/sla"
)

// ratingSink persists every observation through the embedded Sink. For
// if_table it also rates the interface counters against the target's
// previous poll and stores the rates; for ip_sla it stores each operation's
// new results.
type ratingSink struct {
	*sink.Sink

	rater   *ifrate.Rater
	tracker *sla.Tracker
	metrics MetricsStore
}

// PublishIfTable implements [iftable.Publisher].
func (s ratingSink) PublishIfTable(ctx context.Context, obs iftable.Observation) error {
	if err := s.Sink.PublishIfTable(ctx, obs); err != nil {
		return err
	}
	rates := s.rater.Observe(snapshot(obs))
	if err := s.metrics.RecordInterfaceRates(ctx, rates); err != nil {
		return fmt.Errorf("orchestrator: record interface rates: %w", err)
	}
	return nil
}

// PublishIPSLA implements [ipsla.Publisher].
func (s ratingSink) PublishIPSLA(ctx context.Context, obs ipsla.Observation) error {
	if err := s.Sink.PublishIPSLA(ctx, obs); err != nil {
		return err
	}
	if err := s.metrics.RecordSLAResults(ctx, s.tracker.Observe(obs)); err != nil {
		return fmt.Errorf("orchestrator: record ip-sla results: %w", err)
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
			EtherLike:     etherLikeReading(c.EtherLike),
			SpeedBps:      row.SpeedBps,
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

// etherLikeReading renames the collector's dot3StatsTable counters to the
// rate metrics they feed.
func etherLikeReading(counters map[string]uint64) map[string]uint64 {
	if counters == nil {
		return nil
	}
	out := make(map[string]uint64, len(counters))
	for name, value := range counters {
		if metric := etherLikeMetric(name); metric != "" {
			out[metric] = value
		}
	}
	return out
}

func etherLikeMetric(name string) string {
	switch name {
	case iftable.Dot3AlignmentErrors:
		return ifrate.MetricDot3AlignmentErrors
	case iftable.Dot3FCSErrors:
		return ifrate.MetricDot3FCSErrors
	case iftable.Dot3SingleCollisionFrames:
		return ifrate.MetricDot3SingleCollisionFrames
	case iftable.Dot3MultipleCollisionFrames:
		return ifrate.MetricDot3MultipleCollisionFrames
	case iftable.Dot3SQETestErrors:
		return ifrate.MetricDot3SQETestErrors
	case iftable.Dot3DeferredTransmissions:
		return ifrate.MetricDot3DeferredTransmissions
	case iftable.Dot3LateCollisions:
		return ifrate.MetricDot3LateCollisions
	case iftable.Dot3ExcessiveCollisions:
		return ifrate.MetricDot3ExcessiveCollisions
	case iftable.Dot3InternalMacTransmitErrors:
		return ifrate.MetricDot3InternalMacTransmitErrors
	case iftable.Dot3CarrierSenseErrors:
		return ifrate.MetricDot3CarrierSenseErrors
	case iftable.Dot3FrameTooLongs:
		return ifrate.MetricDot3FrameTooLongs
	case iftable.Dot3InternalMacReceiveErrors:
		return ifrate.MetricDot3InternalMacReceiveErrors
	case iftable.Dot3SymbolErrors:
		return ifrate.MetricDot3SymbolErrors
	default:
		return ""
	}
}
