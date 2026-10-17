package delivery

import (
	"context"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/alerts/narrative"
)

// The SNMP trap channel (#3037): one SNMPv2-Trap-PDU per alert to the
// operator's trap receiver, the seedAlertRaised notification of
// docs/mibs/SEED-ALERT-MIB.mib under Mustard Seed Networks' IANA Private
// Enterprise Number, 67249.
//
// The trap is SNMPv2c, authenticated by community. An SNMPv3 trap would be
// sent by an authoritative engine with a persisted engine ID and boots
// counter, which Seed does not have.

// The OIDs of SEED-ALERT-MIB. The objects are accessible-for-notify scalars,
// sent with the scalar instance suffix .0.
const (
	oidSeedAlertMIB = "1.3.6.1.4.1.67249.1.1.1"
	// oidSeedAlertRaised is seedAlertNotifications.1.
	oidSeedAlertRaised = oidSeedAlertMIB + ".0.1"
	oidSeedAlertObject = oidSeedAlertMIB + ".1."
	// oidSysUpTime and oidSnmpTrapOID open every SNMPv2 notification
	// (RFC 3416 §4.2.6).
	oidSysUpTime   = "1.3.6.1.2.1.1.3.0"
	oidSnmpTrapOID = "1.3.6.1.6.3.1.1.4.1.0"
)

// The seedAlertObjects arcs, in the notification's OBJECTS order.
const (
	trapObjID = iota + 1
	trapObjSeverity
	trapObjType
	trapObjRule
	trapObjSource
	trapObjDevice
	trapObjCause
	trapObjTitle
	trapObjMessage
	trapObjSummary
	trapObjNextCheck
	trapObjEvidence
)

// The SeedSeverity enumeration.
const (
	trapSeverityCritical = 1
	trapSeverityError    = 2
	trapSeverityWarning  = 3
	trapSeverityInfo     = 4
)

const (
	// portSNMPTrap is the port IANA assigns to snmptrap.
	portSNMPTrap = 162
	// maxAdminString is SnmpAdminString's size limit (RFC 3411).
	maxAdminString = 255
	// timeTick is the TimeTicks unit, a hundredth of a second.
	timeTick = 10 * time.Millisecond
)

// TrapConfig wires an SNMP trap Notifier. Host and Community are required.
type TrapConfig struct {
	Options

	// Host is the trap receiver's name or address.
	Host string
	// Port defaults to 162.
	Port uint16
	// Community is the SNMPv2c community the receiver admits.
	Community string
	// Timeout bounds one attempt.
	Timeout time.Duration
}

// ValidateTrap rejects a configuration that could never deliver, by the rule
// NewTrap applies, so the settings service refuses it at the API.
func ValidateTrap(cfg TrapConfig) error {
	if cfg.Host == "" {
		return fmt.Errorf("%w: trap receiver host is required", ErrInvalidConfig)
	}
	if strings.ContainsAny(cfg.Host, " /@") || (strings.Contains(cfg.Host, ":") && net.ParseIP(cfg.Host) == nil) {
		return fmt.Errorf("%w: trap receiver host %q must be a bare host name or address, without scheme or port",
			ErrInvalidConfig, cfg.Host)
	}
	if cfg.Community == "" {
		return fmt.Errorf("%w: trap receiver community is required", ErrInvalidConfig)
	}
	return nil
}

// trapSender sends one notification per alert to one receiver.
type trapSender struct {
	host      string
	port      uint16
	community string
	timeout   time.Duration
	// started is when this sender was built, the epoch its sysUpTime counts
	// from: re-pointing the receiver re-initialises the sender, which is the
	// event RFC 3418 has sysUpTime count from.
	started time.Time
	now     func() time.Time
}

// NewTrap validates cfg and builds an SNMP trap Notifier. It makes no
// connection.
func NewTrap(cfg TrapConfig) (*Notifier, error) {
	if err := ValidateTrap(cfg); err != nil {
		return nil, err
	}
	port := cfg.Port
	if port == 0 {
		port = portSNMPTrap
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	n := newNotifier(alerts.ChannelTrap, "snmp://"+net.JoinHostPort(cfg.Host, strconv.Itoa(int(port))), cfg.Options)
	n.transport = &trapSender{
		host:      cfg.Host,
		port:      port,
		community: cfg.Community,
		timeout:   timeout,
		started:   n.now(),
		now:       n.now,
	}
	return n, nil
}

// send makes one delivery. A trap is unacknowledged, so only resolving the
// receiver or handing the datagram to the kernel can fail, and neither is
// permanent.
func (s *trapSender) send(ctx context.Context, alert *alerts.Alert, story *narrative.Text) (bool, error) {
	client := &gosnmp.GoSNMP{
		Context:   ctx,
		Target:    s.host,
		Port:      s.port,
		Transport: "udp",
		Community: s.community,
		Version:   gosnmp.Version2c,
		Timeout:   s.timeout,
	}
	addr := net.JoinHostPort(s.host, strconv.Itoa(int(s.port)))
	if err := client.Connect(); err != nil {
		return false, fmt.Errorf("connect to %s: %w", addr, err)
	}
	defer func() { _ = client.Conn.Close() }()
	vars := trapVariables(alert, story, s.now().Sub(s.started))
	if _, err := client.SendTrap(gosnmp.SnmpTrap{Variables: vars}); err != nil {
		return false, fmt.Errorf("send to %s: %w", addr, err)
	}
	return false, nil
}

// trapVariables renders alert as the seedAlertRaised varbind list: sysUpTime,
// snmpTrapOID, then every object of the notification in OBJECTS order. An
// object the alert has no value for is sent empty or zero rather than left
// out, so a receiver can read each one by position as well as by name.
func trapVariables(alert *alerts.Alert, story *narrative.Text, uptime time.Duration) []gosnmp.SnmpPDU {
	text := func(obj int, value string) gosnmp.SnmpPDU {
		return gosnmp.SnmpPDU{
			Name:  oidSeedAlertObject + strconv.Itoa(obj) + ".0",
			Type:  gosnmp.OctetString,
			Value: string(truncateUTF8([]byte(value), maxAdminString)),
		}
	}
	gauge := func(obj int, value int64) gosnmp.SnmpPDU {
		return gosnmp.SnmpPDU{
			Name:  oidSeedAlertObject + strconv.Itoa(obj) + ".0",
			Type:  gosnmp.Counter64,
			Value: uint64(max(value, 0)),
		}
	}

	var device string
	if alert.DeviceID != nil {
		device = *alert.DeviceID
	}
	var cause int64
	if alert.RootCauseID != nil {
		cause = *alert.RootCauseID
	}
	var summary, nextCheck, evidence string
	if story != nil {
		summary, nextCheck, evidence = story.Summary, story.NextCheck, strings.Join(story.Evidence, "; ")
	}
	return []gosnmp.SnmpPDU{
		{Name: oidSysUpTime, Type: gosnmp.TimeTicks, Value: timeTicks(uptime)},
		{Name: oidSnmpTrapOID, Type: gosnmp.ObjectIdentifier, Value: oidSeedAlertRaised},
		gauge(trapObjID, alert.ID),
		{
			Name:  oidSeedAlertObject + strconv.Itoa(trapObjSeverity) + ".0",
			Type:  gosnmp.Integer,
			Value: trapSeverity(alert.Severity),
		},
		text(trapObjType, alert.Type),
		text(trapObjRule, alert.Rule),
		text(trapObjSource, alert.Source),
		text(trapObjDevice, device),
		gauge(trapObjCause, cause),
		text(trapObjTitle, alert.Title),
		text(trapObjMessage, alert.Message),
		text(trapObjSummary, summary),
		text(trapObjNextCheck, nextCheck),
		text(trapObjEvidence, evidence),
	}
}

// trapSeverity maps an alert severity onto SeedSeverity. An unknown
// severity is sent as warning, the judgement syslogSeverity makes.
func trapSeverity(severity string) int {
	switch severity {
	case alerts.SeverityCritical:
		return trapSeverityCritical
	case alerts.SeverityError:
		return trapSeverityError
	case alerts.SeverityInfo:
		return trapSeverityInfo
	default:
		return trapSeverityWarning
	}
}

// timeTicks is d in hundredths of a second, wrapped modulo 2^32 as sysUpTime
// does after 497 days. A negative d, from an injected clock, is 0.
func timeTicks(d time.Duration) uint32 {
	ticks := int64(d/timeTick) % (math.MaxUint32 + 1)
	if ticks < 0 {
		return 0
	}
	return uint32(ticks)
}
