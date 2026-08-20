// Package snmpclient is the production gosnmp-backed implementation
// of [snmp.Client]. It dials a fresh UDP socket per Get/Walk
// (SNMPv2c is connectionless; the cost is sub-millisecond) so the
// [snmp.Client] interface stays Close()-less and each collector
// remains independent.
//
// SNMPv2c and SNMPv3 (auth + privacy combinations) are supported.
// SNMPv1 is not — every modern enterprise device speaks v2c at
// minimum, and v1's missing GETBULK makes large-table walks 5-10x
// slower without buying any real-world compatibility.
package snmpclient

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/seed/internal/polling/snmp"
)

// Default network parameters. Real production may override via
// FactoryOptions; tests usually accept these.
const (
	defaultPort           uint16        = 161
	defaultTimeout        time.Duration = 5 * time.Second
	defaultRetries        int           = 2
	defaultMaxRepetitions uint32        = 50
)

// Options tunes the gosnmp dial. Pass to NewFactory; zero-valued
// fields fall back to the package defaults.
type Options struct {
	// Port overrides the default SNMP port (161). Useful when an
	// agent is reachable only through a NAT/relay on a non-standard
	// port.
	Port uint16

	// Timeout per request. ctx-derived deadlines also clamp this:
	// the effective timeout is min(Options.Timeout, ctx-remaining).
	Timeout time.Duration

	// Retries before giving up on a single request.
	Retries int

	// MaxRepetitions for GETBULK during Walk. Larger reduces
	// round trips on big tables (ifTable, fdb) but risks ICMP
	// fragmentation on poorly-MTU-tuned WAN paths. 50 is a safe
	// default; pin to 25 on lossy links.
	MaxRepetitions uint32
}

// NewFactory returns an [snmp.ClientFactory] that dials gosnmp.
// Pass zero Options to use defaults.
func NewFactory(opts Options) snmp.ClientFactory {
	if opts.Port == 0 {
		opts.Port = defaultPort
	}
	if opts.Timeout <= 0 {
		opts.Timeout = defaultTimeout
	}
	if opts.Retries < 0 {
		opts.Retries = defaultRetries
	}
	if opts.MaxRepetitions == 0 {
		opts.MaxRepetitions = defaultMaxRepetitions
	}
	return func(target snmp.Target, creds snmp.ResolvedCredentials) (snmp.Client, error) {
		if target.IPAddress == "" {
			return nil, errors.New("snmpclient: target IPAddress required")
		}
		return &client{target: target, creds: creds, opts: opts}, nil
	}
}

// client implements snmp.Client. Stateless — each Get/Walk opens
// its own UDP socket via gosnmp.Connect, defers Close.
type client struct {
	target snmp.Target
	creds  snmp.ResolvedCredentials
	opts   Options
}

// Get fetches the named OIDs. Missing OIDs return varbinds with a
// nil Value rather than failing the whole call (parity with the
// existing collector code).
func (c *client) Get(ctx context.Context, oids []string) ([]snmp.Varbind, error) {
	g, err := c.dial(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = g.Conn.Close() }()

	pkt, err := g.Get(oids)
	if err != nil {
		return nil, fmt.Errorf("snmpclient: Get(%s): %w", c.target.IPAddress, err)
	}
	return toVarbinds(pkt.Variables), nil
}

// Walk traverses prefix using GETBULK (SNMPv2c+) and returns every
// varbind reached before the subtree boundary or ctx cancellation.
func (c *client) Walk(ctx context.Context, prefix string) ([]snmp.Varbind, error) {
	g, err := c.dial(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = g.Conn.Close() }()

	prefix = strings.TrimPrefix(prefix, ".")
	var out []snmp.Varbind
	cb := func(pdu gosnmp.SnmpPDU) error {
		// Honor ctx between PDUs — gosnmp itself doesn't accept ctx
		// so this is the cancellation seam for long walks.
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		out = append(out, toVarbind(pdu))
		return nil
	}
	if walkErr := g.BulkWalk(prefix, cb); walkErr != nil {
		return nil, fmt.Errorf("snmpclient: BulkWalk(%s, %s): %w",
			c.target.IPAddress, prefix, walkErr)
	}
	return out, nil
}

// dial constructs and connects a gosnmp session sized for ctx.
func (c *client) dial(ctx context.Context) (*gosnmp.GoSNMP, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	timeout := c.opts.Timeout
	if dl, ok := ctx.Deadline(); ok {
		remaining := time.Until(dl)
		if remaining < timeout && remaining > 0 {
			timeout = remaining
		}
	}

	g := &gosnmp.GoSNMP{
		Target:         c.target.IPAddress,
		Port:           c.opts.Port,
		Timeout:        timeout,
		Retries:        c.opts.Retries,
		MaxRepetitions: c.opts.MaxRepetitions,
	}
	if applyErr := applyAuth(g, c.target.SNMPVersion, c.creds); applyErr != nil {
		return nil, applyErr
	}
	if connErr := g.Connect(); connErr != nil {
		return nil, fmt.Errorf("snmpclient: connect %s:%d: %w",
			c.target.IPAddress, c.opts.Port, connErr)
	}
	return g, nil
}

// applyAuth picks SNMPv2c or SNMPv3 based on Target.SNMPVersion +
// credentials.
//
// Everything here fails closed. The resolver already refuses to hand back a
// credential a target is not entitled to, and this is the dialer that would
// otherwise undo that: substituting a default community, or quietly picking a
// protocol the operator did not configure, sends packets the operator never
// authorised. A missing or unrecognised credential is an error before the
// socket opens, not a guess on the wire.
func applyAuth(g *gosnmp.GoSNMP, version string, creds snmp.ResolvedCredentials) error {
	switch strings.ToLower(strings.TrimSpace(version)) {
	case "", "v2c", "2c", "v2":
		if creds.SNMPCommunity == "" {
			// "public" is the community every scanner tries first. Falling
			// back to it turns a target with no configured credential into a
			// probe of somebody's default-configured agent.
			return errors.New("snmpclient: no SNMP community resolved for target")
		}
		g.Version = gosnmp.Version2c
		g.Community = creds.SNMPCommunity
		return nil
	case "v3", "3":
		if creds.SNMPv3User == "" {
			return errors.New("snmpclient: SNMPv3User required for SNMPv3 target")
		}
		g.Version = gosnmp.Version3
		g.SecurityModel = gosnmp.UserSecurityModel
		usm := &gosnmp.UsmSecurityParameters{
			UserName: creds.SNMPv3User,
		}
		flags := gosnmp.NoAuthNoPriv
		if creds.SNMPv3AuthSecret != "" {
			authProto, authErr := authProtocol(creds.SNMPv3AuthProto)
			if authErr != nil {
				return authErr
			}
			usm.AuthenticationProtocol = authProto
			usm.AuthenticationPassphrase = creds.SNMPv3AuthSecret
			flags = gosnmp.AuthNoPriv
		}
		if creds.SNMPv3PrivSecret != "" {
			if creds.SNMPv3AuthSecret == "" {
				return errors.New("snmpclient: SNMPv3 priv requires auth")
			}
			privProto, privErr := privProtocol(creds.SNMPv3PrivProto)
			if privErr != nil {
				return privErr
			}
			usm.PrivacyProtocol = privProto
			usm.PrivacyPassphrase = creds.SNMPv3PrivSecret
			flags = gosnmp.AuthPriv
		}
		g.MsgFlags = flags
		g.SecurityParameters = usm
		return nil
	default:
		return fmt.Errorf("snmpclient: unsupported SNMP version %q", version)
	}
}

// authProtocol maps a configured authentication protocol name.
//
// An empty name means the operator did not choose one, so SHA is the default
// they get. A name we do not recognise is different: it is a typo or a
// protocol this build cannot speak, and defaulting it produced an
// authentication failure that reads exactly like a wrong password. Say which
// problem it is instead.
func authProtocol(name string) (gosnmp.SnmpV3AuthProtocol, error) {
	switch strings.ToUpper(strings.TrimSpace(name)) {
	case "":
		return gosnmp.SHA, nil
	case "MD5":
		return gosnmp.MD5, nil
	case "SHA", "SHA1":
		return gosnmp.SHA, nil
	case "SHA224":
		return gosnmp.SHA224, nil
	case "SHA256":
		return gosnmp.SHA256, nil
	case "SHA384":
		return gosnmp.SHA384, nil
	case "SHA512":
		return gosnmp.SHA512, nil
	default:
		return 0, fmt.Errorf("snmpclient: unsupported SNMPv3 auth protocol %q", name)
	}
}

// privProtocol maps a configured privacy protocol name, on the same terms as
// authProtocol: unset means AES, unrecognised is an error rather than AES.
func privProtocol(name string) (gosnmp.SnmpV3PrivProtocol, error) {
	switch strings.ToUpper(strings.TrimSpace(name)) {
	case "":
		return gosnmp.AES, nil
	case "DES":
		return gosnmp.DES, nil
	case "AES", "AES128":
		return gosnmp.AES, nil
	case "AES192":
		return gosnmp.AES192, nil
	case "AES256":
		return gosnmp.AES256, nil
	default:
		return 0, fmt.Errorf("snmpclient: unsupported SNMPv3 privacy protocol %q", name)
	}
}

// toVarbinds maps a gosnmp result Variables slice.
func toVarbinds(vars []gosnmp.SnmpPDU) []snmp.Varbind {
	out := make([]snmp.Varbind, 0, len(vars))
	for _, v := range vars {
		out = append(out, toVarbind(v))
	}
	return out
}

// toVarbind unwraps gosnmp's SnmpPDU into the package's flat
// (OID, value) shape. gosnmp returns ObjectIdentifier values as
// string already; OCTET STRINGs as []byte; counters/gauges as
// numeric — every collector's tolerant decoders handle this fan-out.
func toVarbind(p gosnmp.SnmpPDU) snmp.Varbind {
	return snmp.Varbind{
		OID:   strings.TrimPrefix(p.Name, "."),
		Value: p.Value,
	}
}
