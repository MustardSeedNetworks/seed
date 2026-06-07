// Package fdp implements the fdp SNMP Collector for Foundry /
// Brocade / Ruckus ICX Discovery Protocol neighbors. FDP is wire-
// compatible with CDP: Foundry copied Cisco's cdpCacheTable schema
// column-for-column under their own enterprise OID prefix
// (1.3.6.1.4.1.1991.1.1.3.2.2.1).
//
// Implementation is a thin wrapper that constructs a [cdp.Collector]
// with WithName("fdp") + WithTablePrefix(TablePrefix). Any new CDP
// column the upstream collector learns to parse is automatically
// available here.
package fdp

import (
	"time"

	"github.com/MustardSeedNetworks/seed/internal/polling/snmp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/cdp"
)

// Name is the collector key used in polling_targets.collector_chain.
const Name = "fdp"

// TablePrefix is the Foundry FDP cache-table root. Wire-compatible
// with cdpCacheTable column numbering.
const TablePrefix = "1.3.6.1.4.1.1991.1.1.3.2.2.1"

// New returns a CDP-shaped Collector configured for FDP — same
// schema, different OID prefix, different chain name. The returned
// *cdp.Collector reports Name()=="fdp" so polling_targets chains
// referencing "fdp" resolve to this implementation.
func New(factory snmp.ClientFactory, publisher cdp.Publisher, now func() time.Time) *cdp.Collector {
	return cdp.New(factory, publisher, now,
		cdp.WithName(Name),
		cdp.WithTablePrefix(TablePrefix),
	)
}
