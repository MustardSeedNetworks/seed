package arp_test

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/polling/snmp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/arp"
)

const physicalPrefix = "1.3.6.1.2.1.4.35.1"

// tableClient answers each table walk separately, which is what a real agent
// does and what the single-response fake cannot express.
type tableClient struct {
	physical    []snmp.Varbind
	media       []snmp.Varbind
	physicalErr error
	mediaErr    error
	walked      []string
}

func (c *tableClient) Get(_ context.Context, _ []string) ([]snmp.Varbind, error) {
	return nil, errors.New("get not used by arp")
}

func (c *tableClient) Walk(_ context.Context, prefix string) ([]snmp.Varbind, error) {
	c.walked = append(c.walked, prefix)
	if strings.HasPrefix(prefix, physicalPrefix) {
		return c.physical, c.physicalErr
	}
	return c.media, c.mediaErr
}

func factoryForTable(c *tableClient) snmp.ClientFactory {
	return func(_ snmp.Target, _ snmp.ResolvedCredentials) (snmp.Client, error) { return c, nil }
}

// physOID builds an ipNetToPhysicalTable OID: column, ifIndex, address type,
// address length, then the address one octet at a time. The explicit length is
// what makes an IPv6 address expressible in an index at all.
func physOID(t *testing.T, col string, ifIndex uint32, ip string) string {
	t.Helper()

	address, err := netip.ParseAddr(ip)
	if err != nil {
		t.Fatalf("parse %q: %v", ip, err)
	}
	raw := address.AsSlice()

	addrType := 2
	if address.Is4() {
		addrType = 1
	}

	octets := make([]string, len(raw))
	for i, b := range raw {
		octets[i] = strconv.FormatUint(uint64(b), 10)
	}
	return fmt.Sprintf("%s.%s.%d.%d.%d.%s",
		physicalPrefix, col, ifIndex, addrType, len(raw), strings.Join(octets, "."))
}

func collectWith(t *testing.T, client *tableClient) arp.Observation {
	t.Helper()

	pub := &fakePublisher{}
	collector := arp.New(factoryForTable(client), pub, at)
	target := snmp.Target{ID: "t1", ClientID: "c1"}
	if err := collector.Collect(t.Context(), target, snmp.ResolvedCredentials{}); err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(pub.got) != 1 {
		t.Fatalf("published %d observations, want 1", len(pub.got))
	}
	return pub.got[0]
}

// The point of the issue: ipNetToMediaTable's index ends in four dotted octets
// and has nowhere to put a longer address, so an IPv6 neighbour was simply not
// expressible before.
func TestIPv6NeighbourIsDiscovered(t *testing.T) {
	t.Parallel()

	client := &tableClient{physical: []snmp.Varbind{
		{OID: physOID(t, "4", 2, "2001:db8::1"), Value: []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}},
		{OID: physOID(t, "6", 2, "2001:db8::1"), Value: 3},
		{OID: physOID(t, "7", 2, "2001:db8::1"), Value: 1},
	}}

	obs := collectWith(t, client)

	if len(obs.Entries) != 1 {
		t.Fatalf("entries = %+v, want 1", obs.Entries)
	}
	entry := obs.Entries[0]
	if entry.IPAddress != "2001:db8::1" {
		t.Errorf("address = %q, want 2001:db8::1", entry.IPAddress)
	}
	if entry.MACAddress != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("mac = %q", entry.MACAddress)
	}
	if entry.MediaType != arp.MediaTypeDynamic {
		t.Errorf("type = %d, want dynamic", entry.MediaType)
	}
	if entry.State != arp.StateReachable {
		t.Errorf("state = %d, want reachable", entry.State)
	}
}

// A device implementing both tables answers the same question twice. Reporting
// the binding twice would double every IPv4 neighbour on a modern router.
func TestBindingInBothTablesIsReportedOnce(t *testing.T) {
	t.Parallel()

	client := &tableClient{
		physical: []snmp.Varbind{
			{OID: physOID(t, "4", 1, "192.0.2.10"), Value: []byte{1, 2, 3, 4, 5, 6}},
			{OID: physOID(t, "6", 1, "192.0.2.10"), Value: 3},
			{OID: physOID(t, "7", 1, "192.0.2.10"), Value: 1},
		},
		media: []snmp.Varbind{
			{OID: arpOID("2", 1, "192.0.2.10"), Value: []byte{1, 2, 3, 4, 5, 6}},
			{OID: arpOID("4", 1, "192.0.2.10"), Value: 3},
		},
	}

	obs := collectWith(t, client)

	if len(obs.Entries) != 1 {
		t.Fatalf("entries = %+v, want 1 -- the binding appears in both tables", obs.Entries)
	}
	// The modern table wins, so the state it carries survives.
	if obs.Entries[0].State != arp.StateReachable {
		t.Errorf("state = %d; the ipNetToPhysicalTable row should have won", obs.Entries[0].State)
	}
}

// An agent that implements ipNetToPhysicalTable properly should not be charged
// for a second walk of a table that adds nothing.
func TestLegacyTableIsNotWalkedWhenTheModernOneAnswers(t *testing.T) {
	t.Parallel()

	client := &tableClient{physical: []snmp.Varbind{
		{OID: physOID(t, "4", 1, "192.0.2.10"), Value: []byte{1, 2, 3, 4, 5, 6}},
	}}

	collectWith(t, client)

	if len(client.walked) != 1 {
		t.Errorf("walked %v, want only the ipNetToPhysicalTable", client.walked)
	}
}

// An agent too old for RFC 4293 still has to work.
func TestLegacyOnlyAgentStillWorks(t *testing.T) {
	t.Parallel()

	client := &tableClient{
		physicalErr: errors.New("no such object"),
		media: []snmp.Varbind{
			{OID: arpOID("2", 1, "192.0.2.10"), Value: []byte{1, 2, 3, 4, 5, 6}},
			{OID: arpOID("4", 1, "192.0.2.10"), Value: 3},
		},
	}

	obs := collectWith(t, client)

	if len(obs.Entries) != 1 || obs.Entries[0].IPAddress != "192.0.2.10" {
		t.Fatalf("entries = %+v, want the one legacy binding", obs.Entries)
	}
	// The legacy table has no state column, so zero is honest here.
	if obs.Entries[0].State != 0 {
		t.Errorf("state = %d, want 0 -- ipNetToMediaTable does not report one", obs.Entries[0].State)
	}
}

// Some agents put IPv6 in the new table and leave IPv4 in the old one. Trusting
// the new table alone would silently drop every IPv4 neighbour on those.
func TestSplitAgentGetsBothFamilies(t *testing.T) {
	t.Parallel()

	client := &tableClient{
		physical: []snmp.Varbind{
			{OID: physOID(t, "4", 1, "2001:db8::5"), Value: []byte{1, 1, 1, 1, 1, 1}},
		},
		media: []snmp.Varbind{
			{OID: arpOID("2", 1, "192.0.2.10"), Value: []byte{2, 2, 2, 2, 2, 2}},
			{OID: arpOID("4", 1, "192.0.2.10"), Value: 3},
		},
	}

	obs := collectWith(t, client)

	if len(obs.Entries) != 2 {
		t.Fatalf("entries = %+v, want both families", obs.Entries)
	}
	var sawV4, sawV6 bool
	for _, entry := range obs.Entries {
		address, err := netip.ParseAddr(entry.IPAddress)
		if err != nil {
			t.Fatalf("unparseable address %q", entry.IPAddress)
		}
		sawV4 = sawV4 || address.Is4()
		sawV6 = sawV6 || address.Is6()
	}
	if !sawV4 || !sawV6 {
		t.Errorf("entries = %+v, want one of each family", obs.Entries)
	}
}

// Both tables failing is a real failure; only one is not.
func TestBothTablesFailingIsAnError(t *testing.T) {
	t.Parallel()

	client := &tableClient{
		physicalErr: errors.New("no such object"),
		mediaErr:    errors.New("timeout"),
	}
	collector := arp.New(factoryForTable(client), &fakePublisher{}, at)

	err := collector.Collect(t.Context(), snmp.Target{ID: "t1"}, snmp.ResolvedCredentials{})
	if err == nil {
		t.Fatal("both walks failed but Collect reported success")
	}
}

// A row whose declared length disagrees with its address family indexes
// nothing. Guessing which field to believe would invent a binding.
func TestIndexWithMismatchedLengthIsSkipped(t *testing.T) {
	t.Parallel()

	client := &tableClient{physical: []snmp.Varbind{
		// Declares IPv6 but supplies four octets.
		{OID: physicalPrefix + ".4.1.2.4.192.0.2.10", Value: []byte{1, 2, 3, 4, 5, 6}},
		// Declares IPv4 but supplies a length of 16.
		{OID: physicalPrefix + ".4.1.1.16.192.0.2.11", Value: []byte{1, 2, 3, 4, 5, 6}},
		// An InetAddressType that cannot be a neighbour address at all.
		{OID: physicalPrefix + ".4.1.16.4.192.0.2.12", Value: []byte{1, 2, 3, 4, 5, 6}},
	}}

	obs := collectWith(t, client)

	if len(obs.Entries) != 0 {
		t.Errorf("entries = %+v, want none -- every index was malformed", obs.Entries)
	}
}

// A "local" neighbour is the device's own address; RFC 4293 added the value
// and the existing constants keep their numbers, so it must not be mistaken
// for one of them.
func TestLocalTypeIsDistinctFromStatic(t *testing.T) {
	t.Parallel()

	client := &tableClient{physical: []snmp.Varbind{
		{OID: physOID(t, "4", 1, "2001:db8::1"), Value: []byte{1, 2, 3, 4, 5, 6}},
		{OID: physOID(t, "6", 1, "2001:db8::1"), Value: arp.MediaTypeLocal},
	}}

	obs := collectWith(t, client)

	if len(obs.Entries) != 1 || obs.Entries[0].MediaType != arp.MediaTypeLocal {
		t.Fatalf("entries = %+v, want a local binding", obs.Entries)
	}
	if obs.Entries[0].MediaType == arp.MediaTypeStatic {
		t.Error("local was collapsed into static")
	}
}
