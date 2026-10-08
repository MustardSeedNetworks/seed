package app

// listeners.go builds the persistence the passive-ingress listeners write
// through (syslog, SNMP traps, flows, microbursts, VoIP), so the api wires
// listeners against their own ports and never names a repository (ADR-0020).

import (
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/listener"
	"github.com/MustardSeedNetworks/seed/internal/listener/flow"
	"github.com/MustardSeedNetworks/seed/internal/listener/microburst"
	listenersink "github.com/MustardSeedNetworks/seed/internal/listener/sink"
	"github.com/MustardSeedNetworks/seed/internal/listener/voip"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// ListenerPersistence is what each passive-ingress listener persists into.
// Events is shared: every listener publishes its events to the one
// listener_events sink.
type ListenerPersistence struct {
	Events         listener.Sink
	Flows          flow.Store
	FlowIndicators flow.IndicatorSource
	Microbursts    microburst.Store
	VoIP           voip.Store
}

// NewListenerPersistence binds the listener ports to db.
func NewListenerPersistence(db *database.DB) ListenerPersistence {
	return ListenerPersistence{
		Events:         listenersink.New(db.ListenerEvents(), logging.GetLogger(), nil),
		Flows:          db.FlowRecords(),
		FlowIndicators: db.FlowRecords(),
		Microbursts:    db.Microbursts(),
		VoIP:           db.VoIPStreams(),
	}
}
