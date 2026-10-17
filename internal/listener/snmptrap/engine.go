package snmptrap

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/gosnmp/gosnmp"
)

// For an SNMPv3 inform Seed is the authoritative engine (RFC 3414 1.5.1): the
// sender discovers Seed's snmpEngineID, localizes its keys against it and keeps
// its clock inside Seed's time window. This file is that half of the receiver.

// seedEnterprise is Mustard Seed Networks' IANA Private Enterprise Number.
const seedEnterprise = 67249

// An RFC 3411 snmpEngineID: the enterprise number with its top bit set, a
// format octet, then the format's data. Seed uses the administratively
// assigned octets format with random data.
const (
	engineIDNewFormat    = 0x80000000
	engineIDHeaderLen    = 5
	engineIDFormatOctets = 5
	engineIDRandomLen    = 12
)

// maxEngineTime is snmpEngineTime's maximum (RFC 3414 2.2.1).
const maxEngineTime = 2147483647

// The usmStats objects a Report PDU names (RFC 3414 5).
const (
	oidNotInTimeWindows = ".1.3.6.1.6.3.15.1.1.2.0"
	oidUnknownEngineIDs = ".1.3.6.1.6.3.15.1.1.4.0"
)

// EngineStore persists Seed's snmpEngineID and snmpEngineBoots across
// restarts. BootSNMPEngine records one start: it stores newID on the first
// call and returns the stored ID with the incremented boots count after that.
type EngineStore interface {
	BootSNMPEngine(ctx context.Context, newID []byte) ([]byte, uint32, error)
}

// newEngineID returns an RFC 3411 snmpEngineID under Seed's enterprise
// number, in the octets format with random content.
func newEngineID() []byte {
	id := make([]byte, engineIDHeaderLen+engineIDRandomLen)
	binary.BigEndian.PutUint32(id, engineIDNewFormat|seedEnterprise)
	id[engineIDHeaderLen-1] = engineIDFormatOctets
	_, _ = rand.Read(id[engineIDHeaderLen:])
	return id
}

// bootEngine records this start in the store and sets Seed's engine clock.
func (l *Listener) bootEngine(ctx context.Context) error {
	id, boots, err := l.engineStore.BootSNMPEngine(ctx, newEngineID())
	if err != nil {
		return err
	}
	l.engineID = string(id)
	l.engineBoots = boots
	l.bootedAt = l.now()
	return nil
}

// engineTime is snmpEngineTime: seconds since this boot, at most the RFC 3414
// maximum.
func (l *Listener) engineTime() uint32 {
	secs := int64(l.now().Sub(l.bootedAt) / time.Second)
	switch {
	case secs < 0:
		return 0
	case secs > maxEngineTime:
		return maxEngineTime
	}
	return uint32(secs)
}

// inOwnWindow is RFC 3414 3.2.7 a, the authoritative engine's check of a
// message's time fields against its own clock.
func (l *Listener) inOwnWindow(boots, msgTime uint32) error {
	drift := int64(msgTime) - int64(l.engineTime())
	if l.engineBoots == maxEngineBoots || boots != l.engineBoots || drift > timeWindow || drift < -timeWindow {
		return fmt.Errorf("%w: own engine boots %d time %d, message boots %d time %d",
			errNotInTimeWindow, l.engineBoots, l.engineTime(), boots, msgTime)
	}
	return nil
}

// discovery answers a confirmed-class v3 message addressed to an engine other
// than Seed's with the usmStatsUnknownEngineIDs report that tells the sender
// Seed's engine ID, boots and time (RFC 3414 3.2.3 b and 4). It reports
// whether the message was consumed.
func (l *Listener) discovery(ctx context.Context, conn *net.UDPConn, pkt *gosnmp.SnmpPacket, src *net.UDPAddr) bool {
	if pkt.Version != gosnmp.Version3 || (pkt.PDUType != gosnmp.GetRequest && pkt.PDUType != gosnmp.InformRequest) {
		return false
	}
	usm, ok := pkt.SecurityParameters.(*gosnmp.UsmSecurityParameters)
	if !ok || l.engineID == "" || usm.AuthoritativeEngineID == l.engineID {
		return false
	}
	if pkt.MsgFlags&gosnmp.Reportable == 0 {
		return true
	}
	l.unknownEngineIDs++
	sp := &gosnmp.UsmSecurityParameters{UserName: usm.UserName, Logger: gosnmp.NewLogger(nil)}
	l.report(ctx, conn, pkt, src, gosnmp.NoAuthNoPriv, sp, oidUnknownEngineIDs, l.unknownEngineIDs)
	return true
}

// reportNotInTimeWindow sends the authenticated usmStatsNotInTimeWindows
// report RFC 3414 3.2.7 a requires, from which the sender resynchronizes.
func (l *Listener) reportNotInTimeWindow(
	ctx context.Context,
	conn *net.UDPConn,
	pkt *gosnmp.SnmpPacket,
	src *net.UDPAddr,
) {
	if pkt.MsgFlags&gosnmp.Reportable == 0 {
		return
	}
	sp, err := l.responder(pkt)
	if err != nil {
		l.logger.WarnContext(ctx, "snmptrap: time window report not sent", "source", src.String(), "error", err)
		return
	}
	l.notInTimeWindows++
	l.report(ctx, conn, pkt, src, gosnmp.AuthNoPriv, sp, oidNotInTimeWindows, l.notInTimeWindows)
}

// report sends a Report PDU naming one usmStats counter in reply to req.
func (l *Listener) report(ctx context.Context, conn *net.UDPConn, req *gosnmp.SnmpPacket, src *net.UDPAddr,
	flags gosnmp.SnmpV3MsgFlags, sp *gosnmp.UsmSecurityParameters, oid string, count uint32,
) {
	out := &gosnmp.SnmpPacket{
		Version:            gosnmp.Version3,
		MsgFlags:           flags,
		SecurityModel:      gosnmp.UserSecurityModel,
		SecurityParameters: sp,
		MsgID:              req.MsgID,
		ContextEngineID:    l.engineID,
		PDUType:            gosnmp.Report,
		RequestID:          req.RequestID,
		Variables:          []gosnmp.SnmpPDU{{Name: oid, Type: gosnmp.Counter32, Value: count}},
		Logger:             gosnmp.NewLogger(nil),
	}
	l.send(ctx, conn, out, sp, src, "report")
}

// acknowledgeV3 answers an admitted v3 inform with a Response carrying the
// same varbinds, authenticated and encrypted at the inform's level under
// Seed's engine.
func (l *Listener) acknowledgeV3(ctx context.Context, conn *net.UDPConn, pkt *gosnmp.SnmpPacket, src *net.UDPAddr) {
	sp, err := l.responder(pkt)
	if err != nil {
		l.logger.WarnContext(ctx, "snmptrap: inform response failed", "source", src.String(), "error", err)
		return
	}
	out := *pkt
	out.MsgFlags = pkt.MsgFlags &^ gosnmp.Reportable
	out.SecurityParameters = sp
	out.PDUType = gosnmp.GetResponse
	out.Error = gosnmp.NoError
	out.ErrorIndex = 0
	if err = sp.InitPacket(&out); err != nil {
		l.logger.WarnContext(ctx, "snmptrap: inform response failed", "source", src.String(), "error", err)
		return
	}
	l.send(ctx, conn, &out, sp, src, "inform response")
}

// send stamps Seed's engine fields on sp and writes out to src.
func (l *Listener) send(ctx context.Context, conn *net.UDPConn, out *gosnmp.SnmpPacket,
	sp *gosnmp.UsmSecurityParameters, src *net.UDPAddr, what string,
) {
	sp.AuthoritativeEngineID = l.engineID
	sp.AuthoritativeEngineBoots = l.engineBoots
	sp.AuthoritativeEngineTime = l.engineTime()
	msg, err := out.MarshalMsg()
	if err == nil {
		_, err = conn.WriteToUDP(msg, src)
	}
	if err != nil {
		l.logger.WarnContext(ctx, "snmptrap: "+what+" failed", "source", src.String(), "error", err)
	}
}

// responderKey identifies one credential's keys.
type responderKey struct {
	user      string
	authProto gosnmp.SnmpV3AuthProtocol
	authPass  string
	privProto gosnmp.SnmpV3PrivProtocol
	privPass  string
}

func keyOf(usm *gosnmp.UsmSecurityParameters) responderKey {
	return responderKey{
		user:      usm.UserName,
		authProto: usm.AuthenticationProtocol,
		authPass:  usm.AuthenticationPassphrase,
		privProto: usm.PrivacyProtocol,
		privPass:  usm.PrivacyPassphrase,
	}
}

// responder returns the long-lived security parameters Seed signs and
// encrypts its own messages to pkt's user with, keys localized to Seed's
// engine. They outlive one message because their salt counter must never
// repeat under one key within a boot (RFC 3826 3.1.2.1); a fresh copy per
// message would start every count from the same value.
func (l *Listener) responder(pkt *gosnmp.SnmpPacket) (*gosnmp.UsmSecurityParameters, error) {
	usm, ok := pkt.SecurityParameters.(*gosnmp.UsmSecurityParameters)
	if !ok {
		return nil, fmt.Errorf("security parameters %T", pkt.SecurityParameters)
	}
	key := keyOf(usm)
	if sp, known := l.responders[key]; known {
		return sp, nil
	}
	sp := &gosnmp.UsmSecurityParameters{
		UserName:                 key.user,
		AuthoritativeEngineID:    l.engineID,
		AuthenticationProtocol:   key.authProto,
		AuthenticationPassphrase: key.authPass,
		PrivacyProtocol:          key.privProto,
		PrivacyPassphrase:        key.privPass,
		Logger:                   gosnmp.NewLogger(nil),
	}
	if err := sp.InitSecurityKeys(); err != nil {
		return nil, fmt.Errorf("localize keys for %q: %w", key.user, err)
	}
	l.responders[key] = sp
	return sp, nil
}
