package enumerate

// snmpsweep.go finds the hosts of a target network that the ping sweep cannot:
// a routed device that filters ICMP but answers SNMP (seed#2449). Off-link,
// the ARP table holds only the next hop, so a host that drops the echo request
// is invisible to everything else the sweep does.
//
// The probe is bounded three ways, because it puts the operator's stored
// credentials on the wire towards addresses nobody has seen answer anything:
// it only asks addresses inside the swept target networks, it asks at most
// snmpProbeMaxHosts of them per sweep, and it starts no more than one every
// snmpProbeInterval.

import (
	"context"
	"net"
	"net/netip"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/protocols/snmp"
)

const (
	// snmpProbeMaxHosts caps the silent addresses one sweep asks: one /24's
	// worth, the same budget the ping sweep spends per target network.
	snmpProbeMaxHosts = DefaultMaxHostsPerSubnet
	// snmpProbeInterval spaces probe starts, holding the sweep to 50
	// requests a second per credential however many addresses are silent.
	snmpProbeInterval = 20 * time.Millisecond
	// snmpProbeWorkers bounds the probes in flight while one waits out its
	// timeout.
	snmpProbeWorkers = 16
	// snmpProbeTimeout is how long one credential waits for a reply. An
	// address that answered no echo is usually empty, and the configured
	// polling timeout (5 s, with retries) would hold a sweep for minutes.
	snmpProbeTimeout = time.Second
)

// stateSNMPOnly marks an entry found only by the SNMP probe, the counterpart
// of PING_ONLY.
const stateSNMPOnly = "SNMP_ONLY"

// SNMPProbeReport says what the last sweep's SNMP probe of silent addresses
// did, so an operator can tell "nothing answered" from "nothing was asked".
type SNMPProbeReport struct {
	// Silent is how many target-network addresses answered no echo.
	Silent int `json:"silent"`
	// Tried is how many of them were asked; fewer than Silent when the
	// per-sweep cap cut the list short.
	Tried int `json:"tried"`
	// Answered is how many replied with their system group.
	Answered int `json:"answered"`
	// Skipped names why nothing was asked, empty when the probe ran.
	Skipped string `json:"skipped,omitempty"`
}

// systemQuery is snmp.GetSystemInfo, held as a field so tests can answer for
// a device.
type systemQuery func(context.Context, string, *snmp.Session) (*snmp.SystemInfo, snmp.CredentialRef, error)

// snmpProber asks silent addresses for their SNMP system group with the
// vault's credentials.
type snmpProber struct {
	creds    discovery.SNMPCredentialProvider
	query    systemQuery
	interval time.Duration
}

// probe asks each address, at most snmpProbeMaxHosts of them, and returns an
// entry for every one that answered.
func (p *snmpProber) probe(ctx context.Context, silent []netip.Addr) ([]*ARPEntry, SNMPProbeReport) {
	report := SNMPProbeReport{Silent: len(silent)}
	if len(silent) == 0 {
		return nil, report
	}
	session, err := p.creds.SNMPSession(ctx)
	if err != nil {
		report.Skipped = "credentials unresolved: " + err.Error()
		return nil, report
	}
	if len(session.Communities) == 0 && len(session.V3Credentials) == 0 {
		report.Skipped = "the credential vault holds no SNMP credential"
		return nil, report
	}
	session.Timeout = snmpProbeTimeout
	session.Retries = 0

	if len(silent) > snmpProbeMaxHosts {
		silent = silent[:snmpProbeMaxHosts]
	}

	var (
		mu    sync.Mutex
		found []*ARPEntry
		wg    sync.WaitGroup
	)
	slots := make(chan struct{}, snmpProbeWorkers)
	pace := time.NewTicker(p.interval)
	defer pace.Stop()

	for i, addr := range silent {
		if i > 0 {
			select {
			case <-ctx.Done():
			case <-pace.C:
			}
		}
		select {
		case <-ctx.Done():
		case slots <- struct{}{}:
		}
		if ctx.Err() != nil {
			break
		}
		report.Tried++
		wg.Go(func() {
			defer func() { <-slots }()
			info, _, queryErr := p.query(ctx, addr.String(), session)
			if queryErr != nil {
				return
			}
			mu.Lock()
			defer mu.Unlock()
			found = append(found, &ARPEntry{
				IP:       addr.String(),
				Hostname: info.SysName,
				State:    stateSNMPOnly,
				LastSeen: time.Now(),
			})
		})
	}
	wg.Wait()

	report.Answered = len(found)
	logging.GetLogger().InfoContext(ctx, "Probed silent target-network addresses over SNMP",
		"silent", report.Silent, "tried", report.Tried, "answered", report.Answered)
	return found, report
}

// silentTargets returns the swept target-network addresses that answered no
// echo and are not already known from the neighbour table, in sweep order.
func silentTargets(swept []PingResult, targets []*net.IPNet, known map[string]bool) []netip.Addr {
	prefixes := make([]netip.Prefix, 0, len(targets))
	for _, target := range targets {
		if prefix, ok := ipv4Prefix(target); ok {
			prefixes = append(prefixes, prefix)
		}
	}

	var silent []netip.Addr
	seen := make(map[netip.Addr]bool)
	for _, result := range swept {
		if result.Reachable || known[result.IP] {
			continue
		}
		addr, err := netip.ParseAddr(result.IP)
		if err != nil || seen[addr] || !containsAddr(prefixes, addr) {
			continue
		}
		seen[addr] = true
		silent = append(silent, addr)
	}
	return silent
}

func containsAddr(prefixes []netip.Prefix, addr netip.Addr) bool {
	for _, prefix := range prefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}
