package enumerate

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/protocols/snmp"
)

type stubSNMPCreds struct {
	session *snmp.Session
	err     error
}

func (c stubSNMPCreds) SNMPSession(context.Context) (*snmp.Session, error) {
	if c.err != nil {
		return nil, c.err
	}
	copied := *c.session
	return &copied, nil
}

func communitySession() *snmp.Session {
	s := snmp.NewSession(&config.SNMPConfig{Timeout: 5 * time.Second, Retries: 2, Port: 161})
	s.Communities = []snmp.Community{{ID: "cred-1", String: "public"}}
	return s
}

// recordingQuery answers for the addresses in sysNames and records every
// address asked and the session it was asked with.
type recordingQuery struct {
	mu       sync.Mutex
	sysNames map[string]string
	asked    []string
	sessions []snmp.Session
}

func (q *recordingQuery) query(
	_ context.Context, ip string, session *snmp.Session,
) (*snmp.SystemInfo, snmp.CredentialRef, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.asked = append(q.asked, ip)
	q.sessions = append(q.sessions, *session)
	name, ok := q.sysNames[ip]
	if !ok {
		return nil, snmp.CredentialRef{}, errors.New("request timeout")
	}
	return &snmp.SystemInfo{SysName: name}, snmp.CredentialRef{ID: "cred-1", Version: snmp.VersionV2c}, nil
}

func TestSilentTargetsKeepsOnlyUnansweredAddressesInsideTheTargets(t *testing.T) {
	_, routed, err := net.ParseCIDR("10.51.0.0/24")
	if err != nil {
		t.Fatal(err)
	}
	swept := []PingResult{
		{IP: "10.51.0.1", Reachable: true}, // answered the echo
		{IP: "10.51.0.2"},                  // silent
		{IP: "10.51.0.3"},                  // silent but in the neighbour table
		{IP: "10.52.0.4"},                  // silent, outside every target
		{IP: "10.51.0.5"},                  // silent
		{IP: "10.51.0.2"},                  // swept twice (a retry)
		{IP: "not-an-ip"},
	}

	got := silentTargets(swept, []*net.IPNet{routed}, map[string]bool{"10.51.0.3": true})

	want := addrs(t, "10.51.0.2", "10.51.0.5")
	if !slices.Equal(got, want) {
		t.Errorf("silentTargets = %v, want %v", got, want)
	}
}

func TestSNMPProbeFindsTheHostThatAnswersSNMPButNotPing(t *testing.T) {
	q := &recordingQuery{sysNames: map[string]string{"10.51.0.7": "core-sw-07"}}
	p := &snmpProber{creds: stubSNMPCreds{session: communitySession()}, query: q.query, interval: time.Millisecond}
	silent := addrs(t, "10.51.0.6", "10.51.0.7", "10.51.0.8")

	found, report := p.probe(context.Background(), silent)

	if len(found) != 1 || found[0].IP != "10.51.0.7" || found[0].Hostname != "core-sw-07" ||
		found[0].State != stateSNMPOnly || found[0].MAC != "" || found[0].IsLocal {
		t.Fatalf("found = %+v, want one SNMP-only entry for 10.51.0.7 named core-sw-07", found)
	}
	if want := (SNMPProbeReport{Silent: 3, Tried: 3, Answered: 1}); report != want {
		t.Errorf("report = %+v, want %+v", report, want)
	}

	slices.Sort(q.asked)
	if want := []string{"10.51.0.6", "10.51.0.7", "10.51.0.8"}; !slices.Equal(q.asked, want) {
		t.Errorf("asked %v, want exactly the silent addresses %v", q.asked, want)
	}
	for _, s := range q.sessions {
		if s.Timeout != snmpProbeTimeout || s.Retries != 0 {
			t.Errorf("probe asked with timeout %v retries %d, want %v and 0", s.Timeout, s.Retries, snmpProbeTimeout)
		}
	}
}

func TestSNMPProbeAsksAtMostTheCap(t *testing.T) {
	q := &recordingQuery{}
	p := &snmpProber{creds: stubSNMPCreds{session: communitySession()}, query: q.query, interval: time.Microsecond}
	silent := make([]netip.Addr, 0, snmpProbeMaxHosts+40)
	for i := range snmpProbeMaxHosts + 40 {
		silent = append(silent, netip.MustParseAddr(fmt.Sprintf("10.60.%d.%d", i/250, i%250+1)))
	}

	_, report := p.probe(context.Background(), silent)

	if report.Silent != snmpProbeMaxHosts+40 || report.Tried != snmpProbeMaxHosts {
		t.Errorf("report = %+v, want %d silent and %d tried", report, snmpProbeMaxHosts+40, snmpProbeMaxHosts)
	}
	if len(q.asked) != snmpProbeMaxHosts {
		t.Errorf("asked %d addresses, want the cap %d", len(q.asked), snmpProbeMaxHosts)
	}
}

func TestSNMPProbeHoldsItsRate(t *testing.T) {
	const interval = 10 * time.Millisecond
	q := &recordingQuery{}
	p := &snmpProber{creds: stubSNMPCreds{session: communitySession()}, query: q.query, interval: interval}
	silent := addrs(t, "10.51.0.1", "10.51.0.2", "10.51.0.3", "10.51.0.4", "10.51.0.5", "10.51.0.6")

	start := time.Now()
	p.probe(context.Background(), silent)

	if elapsed, floor := time.Since(start), time.Duration(len(silent)-1)*interval; elapsed < floor {
		t.Errorf("%d probes took %v, want at least %v at one per %v", len(silent), elapsed, floor, interval)
	}
}

func TestSNMPProbeAsksNothingWithoutCredentials(t *testing.T) {
	empty := snmp.NewSession(&config.SNMPConfig{})
	tests := []struct {
		name  string
		creds stubSNMPCreds
		want  string
	}{
		{
			"vault unreadable",
			stubSNMPCreds{err: errors.New("2 clients exist")},
			"credentials unresolved: 2 clients exist",
		},
		{"vault empty", stubSNMPCreds{session: empty}, "the credential vault holds no SNMP credential"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &recordingQuery{}
			p := &snmpProber{creds: tt.creds, query: q.query, interval: time.Millisecond}

			found, report := p.probe(context.Background(), addrs(t, "10.51.0.2"))

			if len(found) != 0 || len(q.asked) != 0 {
				t.Errorf("found %v after asking %v, want nothing asked", found, q.asked)
			}
			if want := (SNMPProbeReport{Silent: 1, Skipped: tt.want}); report != want {
				t.Errorf("report = %+v, want %+v", report, want)
			}
		})
	}
}

func TestSNMPProbeStopsWhenTheSweepIsCancelled(t *testing.T) {
	q := &recordingQuery{}
	p := &snmpProber{creds: stubSNMPCreds{session: communitySession()}, query: q.query, interval: time.Hour}
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(20*time.Millisecond, cancel)

	_, report := p.probe(ctx, addrs(t, "10.51.0.1", "10.51.0.2", "10.51.0.3"))

	if report.Tried != 1 {
		t.Errorf("tried %d, want only the first address before the cancel", report.Tried)
	}
}

func TestMergeARPResultsRecordsSNMPOnlyHostsAsFoundBySNMP(t *testing.T) {
	scanner := NewARPScanner("lo", nil)
	scanner.entries = map[string]*ARPEntry{
		"10.51.0.7": {IP: "10.51.0.7", Hostname: "core-sw-07", State: stateSNMPOnly},
		"10.51.0.9": {IP: "10.51.0.9", State: "PING_ONLY"},
	}
	d := &DeviceDiscovery{devices: map[string]*DiscoveredDevice{}, arpScanner: scanner}

	d.mergeARPResults()

	snmpHost := d.devices["ip:10.51.0.7"]
	if snmpHost == nil || snmpHost.Hostname != "core-sw-07" ||
		!slices.Equal(snmpHost.DiscoveryMethod, []Method{MethodSNMP}) {
		t.Errorf("SNMP-only host merged as %+v, want hostname core-sw-07 found by snmp", snmpHost)
	}
	if pingHost := d.devices["ip:10.51.0.9"]; pingHost == nil ||
		!slices.Equal(pingHost.DiscoveryMethod, []Method{MethodPING}) {
		t.Errorf("ping-only host merged as %+v, want found by ping", pingHost)
	}
}
