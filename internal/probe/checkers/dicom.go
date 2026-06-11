package checkers

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/probe"
)

// defaultDICOMTimeout is the per-attempt DICOM probe timeout.
const defaultDICOMTimeout = 5 * time.Second

// defaultDICOMPort is the default TCP port for DICOM.
const defaultDICOMPort = 11112

// DICOM PS3.8 §9.1 PDU types.
const (
	dicomPDUTypeAAssociateRQ = 0x01
	dicomPDUTypeAAssociateAC = 0x02
)

// DICOM PS3.8 §9.3 item types.
const (
	dicomItemApplicationContext = 0x10
	dicomItemPresentationCtx    = 0x20
	dicomItemAbstractSyntax     = 0x30
	dicomItemTransferSyntax     = 0x40
	dicomItemUserInformation    = 0x50
	dicomSubItemMaxPDU          = 0x51
)

// dicomPDUHeaderLen is the length of the DICOM PDU header
// (1 byte type + 1 reserved + 4 byte length).
const dicomPDUHeaderLen = 6

// DICOM A-ASSOCIATE-RQ body section sizes.
const (
	dicomAETitleLen      = 16
	dicomReservedZeroLen = 32
	dicomProtocolVersion = 0x0001
	dicomMaxPDUValue     = 0x4000 // 16384, the typical max PDU length
	dicomPresentationID  = 0x01
	dicomReservedByte    = 0x00
)

// dicomBodyInitialCap is the initial capacity for the body buffer.
// Sized to fit a typical A-ASSOCIATE-RQ in one allocation.
const dicomBodyInitialCap = 256

// dicomPCBodyInitialCap is the initial capacity for the
// presentation-context sub-buffer.
const dicomPCBodyInitialCap = 64

// DICOMParams is the kind-specific params shape.
type DICOMParams struct {
	Port      int    `json:"port,omitempty"`       // default 11112
	CallingAE string `json:"calling_ae,omitempty"` // default "SEED"
	CalledAE  string `json:"called_ae,omitempty"`  // default "ANY-SCP"
	TimeoutMs int    `json:"timeout_ms,omitempty"` // default 5000
}

// DICOMChecker implements probe.Checker for Kind="dicom". Performs
// a minimal A-ASSOCIATE-RQ + reads the response PDU type. Reaching
// A-ASSOCIATE-AC (type 0x02) is the success signal.
type DICOMChecker struct {
	dialer PingDialer
}

// NewDICOMChecker returns a DICOMChecker wired to a real dialer.
func NewDICOMChecker() *DICOMChecker {
	return &DICOMChecker{dialer: realPingDialer{}}
}

// WithDICOMDialer swaps the dialer (for tests).
func (c *DICOMChecker) WithDICOMDialer(d PingDialer) *DICOMChecker {
	c.dialer = d
	return c
}

// Kind returns probe.KindDICOM.
func (c *DICOMChecker) Kind() string { return probe.KindDICOM }

// RequiredCapabilities returns nil.
func (c *DICOMChecker) RequiredCapabilities() []string { return nil }

// Run sends an A-ASSOCIATE-RQ to Target:Port and inspects the
// returned PDU type. A-ASSOCIATE-AC = success.
func (c *DICOMChecker) Run(ctx context.Context, p probe.Probe) probe.Result {
	params := parseDICOMParams(p.Params)

	port := params.Port
	if port == 0 {
		port = defaultDICOMPort
	}
	timeout := defaultDICOMTimeout
	if params.TimeoutMs > 0 {
		timeout = time.Duration(params.TimeoutMs) * time.Millisecond
	}
	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	addr := net.JoinHostPort(p.Target, strconv.Itoa(port))
	start := time.Now()
	conn, err := c.dialer.Dial(dialCtx, "tcp", addr)
	if err != nil {
		return probe.Result{
			ProbeID: p.ID, ClientID: p.ClientID, Kind: p.Kind,
			Timestamp: time.Now().UTC(), Success: false,
			LatencyMs: float64(time.Since(start).Milliseconds()),
			Error:     err.Error(),
		}
	}
	defer func() { _ = conn.Close() }()

	deadline := time.Now().Add(timeout)
	_ = conn.SetDeadline(deadline)

	callingAE := stringOrDefault(params.CallingAE, "SEED")
	calledAE := stringOrDefault(params.CalledAE, "ANY-SCP")

	rq, buildErr := buildDICOMAssociateRQ(callingAE, calledAE)
	if buildErr != nil {
		return probe.Result{
			ProbeID: p.ID, ClientID: p.ClientID, Kind: p.Kind,
			Timestamp: time.Now().UTC(), Success: false,
			LatencyMs: float64(time.Since(start).Milliseconds()),
			Error:     "build A-ASSOCIATE-RQ: " + buildErr.Error(),
		}
	}
	if _, writeErr := conn.Write(rq); writeErr != nil {
		return probe.Result{
			ProbeID: p.ID, ClientID: p.ClientID, Kind: p.Kind,
			Timestamp: time.Now().UTC(), Success: false,
			LatencyMs: float64(time.Since(start).Milliseconds()),
			Error:     "write A-ASSOCIATE-RQ: " + writeErr.Error(),
		}
	}

	hdr := make([]byte, dicomPDUHeaderLen)
	if _, readErr := io.ReadFull(conn, hdr); readErr != nil {
		return probe.Result{
			ProbeID: p.ID, ClientID: p.ClientID, Kind: p.Kind,
			Timestamp: time.Now().UTC(), Success: false,
			LatencyMs: float64(time.Since(start).Milliseconds()),
			Error:     "read PDU header: " + readErr.Error(),
		}
	}
	latencyMs := float64(time.Since(start).Milliseconds())

	pduType := hdr[0]
	meta, _ := json.Marshal(map[string]any{
		metaKeyAddr: addr,
		"pdu_type":  pduType,
	})

	if pduType != dicomPDUTypeAAssociateAC {
		return probe.Result{
			ProbeID: p.ID, ClientID: p.ClientID, Kind: p.Kind,
			Timestamp: time.Now().UTC(), Success: false,
			LatencyMs: latencyMs,
			Error:     fmt.Sprintf("unexpected PDU type 0x%02x (want 0x02 A-ASSOCIATE-AC)", pduType),
			Metadata:  meta,
		}
	}

	return probe.Result{
		ProbeID:   p.ID,
		ClientID:  p.ClientID,
		Kind:      p.Kind,
		Timestamp: time.Now().UTC(),
		Success:   true,
		LatencyMs: latencyMs,
		Metadata:  meta,
	}
}

// DICOM standard UIDs used by the Verification SOP A-ASSOCIATE-RQ.
const (
	dicomAppCtxVerification    = "1.2.840.10008.3.1.1.1"
	dicomAbstractSyntaxVerify  = "1.2.840.10008.1.1"
	dicomTransferSyntaxDefault = "1.2.840.10008.1.2"
)

// buildDICOMAssociateRQ constructs a minimal-but-valid A-ASSOCIATE-RQ
// PDU. The structure mirrors what the legacy
// internal/api/health_checks_dicom.go builds — just enough to elicit
// either an accept or reject response. Real DICOM SCP servers
// generally accept this when the AE titles are well-known.
//
// Format (DICOM PS3.8 §9.3.2 simplified):
//
//	Header:        type + reserved + 4-byte length
//	PDU body:      version (2B) + reserved (2B) + calledAE (padded) +
//	               callingAE (padded) + reserved (zero) +
//	               application context item + presentation context +
//	               user information.
//
// V1.0 ships the minimal viable form; real-world SCPs differ in
// strictness. Operators with stricter SCPs use kind="tcp" against
// the DICOM port as a fallback.
func buildDICOMAssociateRQ(calling, called string) ([]byte, error) {
	body := make([]byte, 0, dicomBodyInitialCap)

	// Protocol version (2B) + 2 reserved
	body = binary.BigEndian.AppendUint16(body, dicomProtocolVersion)
	body = append(body, dicomReservedByte, dicomReservedByte)

	// Called AE then Calling AE, space-padded to fixed width.
	body = append(body, padAE(called)...)
	body = append(body, padAE(calling)...)

	// 32 reserved bytes (zero).
	body = append(body, make([]byte, dicomReservedZeroLen)...)

	// Application context item (Verification SOP).
	appCtxBytes, err := appendItem(nil, dicomItemApplicationContext, []byte(dicomAppCtxVerification))
	if err != nil {
		return nil, err
	}
	body = append(body, appCtxBytes...)

	// Presentation context body: PC id (1B), 3 reserved, abstract
	// syntax sub-item, transfer syntax sub-item.
	pcBody := make([]byte, 0, dicomPCBodyInitialCap)
	pcBody = append(pcBody, dicomPresentationID, dicomReservedByte, dicomReservedByte, dicomReservedByte)
	pcBody, err = appendItem(pcBody, dicomItemAbstractSyntax, []byte(dicomAbstractSyntaxVerify))
	if err != nil {
		return nil, err
	}
	pcBody, err = appendItem(pcBody, dicomItemTransferSyntax, []byte(dicomTransferSyntaxDefault))
	if err != nil {
		return nil, err
	}
	body, err = appendItem(body, dicomItemPresentationCtx, pcBody)
	if err != nil {
		return nil, err
	}

	// User Information item: max PDU length sub-item only.
	const userInfoInitialCap = 8
	userInfo := make([]byte, 0, userInfoInitialCap)
	userInfo = append(userInfo, dicomSubItemMaxPDU, dicomReservedByte)
	const maxPDULengthFieldBytes = 4
	maxPDUBytes := make([]byte, maxPDULengthFieldBytes)
	binary.BigEndian.PutUint32(maxPDUBytes, dicomMaxPDUValue)
	maxPDUBytesLen, err := safeUint16(len(maxPDUBytes))
	if err != nil {
		return nil, err
	}
	userInfo = binary.BigEndian.AppendUint16(userInfo, maxPDUBytesLen)
	userInfo = append(userInfo, maxPDUBytes...)
	body, err = appendItem(body, dicomItemUserInformation, userInfo)
	if err != nil {
		return nil, err
	}

	bodyLen, err := safeUint32(len(body))
	if err != nil {
		return nil, err
	}

	pdu := make([]byte, 0, dicomPDUHeaderLen+len(body))
	pdu = append(pdu, dicomPDUTypeAAssociateRQ, dicomReservedByte)
	pdu = binary.BigEndian.AppendUint32(pdu, bodyLen)
	pdu = append(pdu, body...)
	return pdu, nil
}

// appendItem appends a DICOM (type, length, value) item to buf and
// returns the result. Errors when value's length exceeds the
// uint16 length field's capacity.
func appendItem(buf []byte, itemType byte, value []byte) ([]byte, error) {
	valueLen, err := safeUint16(len(value))
	if err != nil {
		return nil, err
	}
	buf = append(buf, itemType, dicomReservedByte)
	buf = binary.BigEndian.AppendUint16(buf, valueLen)
	buf = append(buf, value...)
	return buf, nil
}

// safeUint16 returns n as uint16, or an error if it exceeds the
// maximum representable value.
func safeUint16(n int) (uint16, error) {
	if n < 0 || n > 0xFFFF {
		return 0, fmt.Errorf("length %d exceeds uint16 capacity", n)
	}
	return uint16(n), nil
}

// safeUint32 returns n as uint32, or an error if it exceeds the
// maximum representable value.
func safeUint32(n int) (uint32, error) {
	if n < 0 || n > 0xFFFFFFFF {
		return 0, fmt.Errorf("length %d exceeds uint32 capacity", n)
	}
	return uint32(n), nil
}

// padAE returns a 16-byte buffer with the AE title left-aligned and
// space-padded.
func padAE(ae string) []byte {
	const aeLen = 16
	out := make([]byte, aeLen)
	for i := range aeLen {
		if i < len(ae) {
			out[i] = ae[i]
		} else {
			out[i] = ' '
		}
	}
	return out
}

func stringOrDefault(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func parseDICOMParams(raw json.RawMessage) DICOMParams {
	if len(raw) == 0 {
		return DICOMParams{}
	}
	var p DICOMParams
	_ = json.Unmarshal(raw, &p)
	return p
}
