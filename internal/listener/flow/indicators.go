package flow

import (
	"context"
	"encoding/json"
	"net/netip"

	"github.com/MustardSeedNetworks/seed/internal/indicators"
	"github.com/MustardSeedNetworks/seed/internal/listener"
)

// IndicatorSource supplies the operator's threat indicator list. The
// database implementation caches it, so the collector asks once a batch.
type IndicatorSource interface {
	FlowIndicators(ctx context.Context) (*indicators.List, error)
}

// hitKey identifies one host talking to one listed address.
type hitKey struct {
	host, listed netip.Addr
}

// checkIndicators publishes one event for each host and listed address the
// batch's flows connect. A flow is checked whether or not the store took
// it: a database fault must not hide traffic to a hostile address.
func (l *Listener) checkIndicators(ctx context.Context, batch []Record) {
	if l.indicators == nil {
		return
	}
	list, err := l.indicators.FlowIndicators(ctx)
	if err != nil {
		l.logger.WarnContext(ctx, "flow: threat indicators unavailable", "error", err)
		return
	}
	if list.Len() == 0 {
		return
	}
	hits := map[hitKey]*listener.FlowIndicatorHit{}
	var order []hitKey
	record := func(f *Record, host, listed netip.Addr) {
		indicator, ok := list.Match(listed)
		if !ok {
			return
		}
		k := hitKey{host: host, listed: listed}
		h := hits[k]
		if h == nil {
			h = &listener.FlowIndicatorHit{
				Host:      host.String(),
				Listed:    listed.String(),
				Indicator: indicator.String(),
				Exporter:  f.Exporter.String(),
				FirstSeen: f.Start,
				LastSeen:  f.End,
			}
			hits[k] = h
			order = append(order, k)
		}
		h.Flows++
		h.Bytes += f.Bytes
		h.Packets += f.Packets
		if f.Start.Before(h.FirstSeen) {
			h.FirstSeen = f.Start
		}
		if f.End.After(h.LastSeen) {
			h.LastSeen = f.End
		}
	}
	for i := range batch {
		f := &batch[i]
		record(f, f.SrcAddr, f.DstAddr)
		record(f, f.DstAddr, f.SrcAddr)
	}
	for _, k := range order {
		l.publishHit(ctx, hits[k])
	}
}

func (l *Listener) publishHit(ctx context.Context, h *listener.FlowIndicatorHit) {
	payload, err := json.Marshal(h)
	if err != nil {
		l.logger.WarnContext(ctx, "flow: encode threat indicator hit", "error", err)
		return
	}
	if pubErr := l.sink.Publish(ctx, listener.Event{
		Kind:       listener.FlowIndicatorKind,
		SourceAddr: h.Host,
		Timestamp:  l.now(),
		Payload:    payload,
	}); pubErr != nil {
		l.logger.WarnContext(ctx, "flow: publish threat indicator hit failed",
			"host", h.Host, "listed", h.Listed, "error", pubErr)
	}
}
