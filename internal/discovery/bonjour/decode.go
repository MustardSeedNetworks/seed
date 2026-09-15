package bonjour

import (
	"net/netip"
	"slices"
	"strings"

	"golang.org/x/net/dns/dnsmessage"
)

// dnssdMetaQuery is the service-type enumeration name of RFC 6763 §9. A PTR
// query for it returns the service types a responder has, which is what makes
// a browse possible without a hard-coded list.
const dnssdMetaQuery = "_services._dns-sd._udp.local."

// collector accumulates records across packets and assembles services. A
// responder is free to answer a type PTR query with the instance PTR alone
// and send SRV, TXT and A later, in separate packets or in the Additionals of
// an unrelated one — observed against macOS's mDNSResponder, which answered
// `_airplay._tcp.local` with the instance PTR and nothing else. So records are
// harvested by name from every section of every packet and joined at the end,
// never assumed to arrive together.
type collector struct {
	lim limits

	types map[string]struct{}
	// instances is keyed by the fully qualified instance name.
	instances map[string]*instanceRecords
	// addrs maps a host name to its A/AAAA records.
	addrs map[string][]netip.Addr

	responses int
	truncated bool
}

type instanceRecords struct {
	name    string
	srvHost string
	srvPort uint16
	txt     map[string]string
	sources map[netip.Addr]struct{}
	// order preserves first-seen ordering so output is stable without a sort
	// on a field the wire does not define an order for.
	seq int
}

func newCollector(lim limits) *collector {
	return &collector{
		lim:       lim,
		types:     make(map[string]struct{}),
		instances: make(map[string]*instanceRecords),
		addrs:     make(map[string][]netip.Addr),
	}
}

// observe decodes one mDNS packet and folds its records in. A packet that
// does not unpack, or that is a query rather than a response, is ignored:
// the segment is untrusted and a malformed frame is not an error condition
// for the browse.
func (c *collector) observe(data []byte, src netip.Addr) {
	var msg dnsmessage.Message
	if err := msg.Unpack(data); err != nil {
		return
	}
	if !msg.Response {
		return
	}
	c.responses++

	for _, section := range [][]dnsmessage.Resource{msg.Answers, msg.Authorities, msg.Additionals} {
		for i := range section {
			c.observeRecord(&section[i], src)
		}
	}
}

func (c *collector) observeRecord(r *dnsmessage.Resource, src netip.Addr) {
	name := trimRoot(r.Header.Name.String())
	switch body := r.Body.(type) {
	case *dnsmessage.PTRResource:
		c.observePTR(name, trimRoot(body.PTR.String()), src)
	case *dnsmessage.SRVResource:
		if inst, ok := c.instance(name, src); ok {
			inst.srvHost = trimRoot(body.Target.String())
			inst.srvPort = body.Port
		}
	case *dnsmessage.TXTResource:
		if inst, ok := c.instance(name, src); ok {
			c.mergeTXT(inst, body.TXT)
		}
	case *dnsmessage.AResource:
		c.observeAddr(name, netip.AddrFrom4(body.A))
	case *dnsmessage.AAAAResource:
		c.observeAddr(name, netip.AddrFrom16(body.AAAA).Unmap())
	}
}

func (c *collector) observePTR(owner, target string, src netip.Addr) {
	if owner == trimRoot(dnssdMetaQuery) {
		// The meta-query's answers are service types, not instances.
		if len(c.types) >= c.lim.maxServiceTypes {
			c.truncated = true
			return
		}
		c.types[target] = struct{}{}
		return
	}
	// A PTR owned by a service type names an instance of it. Anything else —
	// reverse-lookup PTRs in particular — is not DNS-SD and is dropped.
	if !isServiceType(owner) {
		return
	}
	c.types[owner] = struct{}{}
	c.instance(target, src)
}

// instance returns the record set for a fully qualified instance name,
// creating it on first sight. The second result is false when the name is not
// an instance of a service type, or when the instance bound is reached.
func (c *collector) instance(name string, src netip.Addr) (*instanceRecords, bool) {
	if _, _, ok := splitInstance(name); !ok {
		return nil, false
	}
	inst, seen := c.instances[name]
	if !seen {
		if len(c.instances) >= c.lim.maxInstances {
			c.truncated = true
			return nil, false
		}
		inst = &instanceRecords{
			name:    name,
			txt:     make(map[string]string),
			sources: make(map[netip.Addr]struct{}),
			seq:     len(c.instances),
		}
		c.instances[name] = inst
	}
	if src.IsValid() {
		inst.sources[src] = struct{}{}
	}
	return inst, true
}

func (c *collector) mergeTXT(inst *instanceRecords, strs []string) {
	for _, s := range strs {
		if len(inst.txt) >= c.lim.maxTXTPairs {
			c.truncated = true
			return
		}
		key, value, found := strings.Cut(s, "=")
		if key == "" {
			continue
		}
		if !found {
			// RFC 6763 §6.4: a key with no '=' is present with no value, and
			// is not the same as a key with an empty value. Both land as "";
			// seed shows presence, not that distinction.
			value = ""
		}
		if len(value) > c.lim.maxTXTValueBytes {
			value = value[:c.lim.maxTXTValueBytes]
			c.truncated = true
		}
		inst.txt[key] = value
	}
}

func (c *collector) observeAddr(host string, addr netip.Addr) {
	if !addr.IsValid() || addr.IsUnspecified() {
		return
	}
	existing := c.addrs[host]
	if len(existing) >= c.lim.maxAddressesPerSvc {
		c.truncated = true
		return
	}
	if slices.Contains(existing, addr) {
		return
	}
	c.addrs[host] = append(existing, addr)
}

// isServiceType reports whether a name is a DNS-SD service type such as
// "_airplay._tcp.local": two underscore labels over a domain.
func isServiceType(name string) bool {
	// _airplay, _tcp, local — the shortest a service type can be.
	const serviceTypeLabels = 3
	labels := splitLabels(name)
	if len(labels) < serviceTypeLabels {
		return false
	}
	return strings.HasPrefix(labels[0], "_") && (labels[1] == "_tcp" || labels[1] == "_udp")
}

// splitInstance splits "Living Room._airplay._tcp.local" into its instance
// name and service type. It reports false for a name that is not an instance.
func splitInstance(name string) (string, string, bool) {
	// Living Room, _airplay, _tcp, local — an instance is a service type
	// with one more label in front of it.
	const instanceLabels = 4
	labels := splitLabels(name)
	if len(labels) < instanceLabels {
		return "", "", false
	}
	if !strings.HasPrefix(labels[1], "_") || (labels[2] != "_tcp" && labels[2] != "_udp") {
		return "", "", false
	}
	return labels[0], labels[1] + "." + labels[2], true
}

// splitLabels splits a name on its label separators.
//
// It splits on every dot, with no escape handling, because the parser never
// hands one over: golang.org/x/net/dns/dnsmessage rejects the whole packet
// with "Name: invalid dns name" when any label contains a literal dot, and
// emits names verbatim otherwise — a backslash in a label comes back as a
// backslash, not as an escape. Measured against the library on 2026-09-15;
// an unescaping pass here would be code no input can reach.
//
// The cost of that library behaviour is real and is not seed's to fix here:
// a Bonjour instance named "Office No. 5" makes its announcement undecodable,
// so the service is invisible to this browse and to the name resolution in
// internal/discovery/resolve, which parses with the same library. Filed
// separately rather than answered by standing up a second mDNS stack.
func splitLabels(name string) []string {
	if name == "" {
		return nil
	}
	return strings.Split(name, ".")
}

func trimRoot(name string) string {
	return strings.TrimSuffix(name, ".")
}
