package discovery

import (
	"context"
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/protocols/snmp"
)

var errNoCredentials = errors.New("no SNMP credentials")

type noSNMPCredentials struct{}

func (noSNMPCredentials) SNMPSession(context.Context) (*snmp.Session, error) {
	return nil, errNoCredentials
}

// The route-table learning source reads SNMPData.Routing, so a default install
// has to walk IP-FORWARD-MIB or that source never runs (seed#2833). The walk
// is bounded at snmp.MaxRouteRows, which is what makes the default safe.
func TestDefaultProfilerWalksTheRouteTable(t *testing.T) {
	withoutSelection := DefaultProfilerConfig()
	withoutSelection.SNMPMIBs = nil

	for name, cfg := range map[string]*ProfilerConfig{
		"default config":       DefaultProfilerConfig(),
		"no MIB selection set": withoutSelection,
	} {
		t.Run(name, func(t *testing.T) {
			p := NewDeviceProfiler(cfg, noSNMPCredentials{})
			if p.snmpCollector == nil || !p.snmpCollector.mibConfig.Routing {
				t.Fatal("the profiler's SNMP collector does not walk the route table")
			}
		})
	}
}
