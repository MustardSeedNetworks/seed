package checkers

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/probe"
)

// defaultModbusTimeout is the per-attempt Modbus probe timeout.
const defaultModbusTimeout = 5 * time.Second

// defaultModbusPort is the default Modbus TCP port (IANA 502).
const defaultModbusPort = 502

// Modbus read function codes (Modbus Application Protocol V1.1b3 §6).
const (
	modbusFCReadCoils            = 0x01
	modbusFCReadDiscreteInputs   = 0x02
	modbusFCReadHoldingRegisters = 0x03
	modbusFCReadInputRegisters   = 0x04
)

// Register-type selectors accepted in params.register_type.
const (
	modbusRegisterHolding  = "holding"
	modbusRegisterInput    = "input"
	modbusRegisterCoil     = "coil"
	modbusRegisterDiscrete = "discrete"
)

// Modbus TCP ADU framing sizes (Modbus Messaging on TCP/IP V1.0b §3-4).
const (
	modbusRequestSize    = 12 // MBAP header (7) + PDU (5: FC + addr + qty)
	modbusPDULength      = 6  // unit ID + FC + start addr + quantity, the MBAP length field
	modbusMinResponseLen = 9  // MBAP (7) + FC (1) + byte-count/exception (1)
	modbusExceptionFlag  = 0x80
	modbusUnitIDMax      = 247 // 1-247 addressable; 0 = broadcast
	modbusMaxRegister    = 0xFFFF
)

// modbusResponseBufferSize bounds a single read of the Modbus response.
const modbusResponseBufferSize = 256

// ModbusParams is the kind-specific params shape. UnitID and
// TestRegister default to 0 (valid); RegisterType defaults to holding.
type ModbusParams struct {
	Port         int    `json:"port,omitempty"`          // default 502
	UnitID       int    `json:"unit_id,omitempty"`       // 0-247, default 0
	RegisterType string `json:"register_type,omitempty"` // holding|input|coil|discrete, default holding
	TestRegister int    `json:"test_register,omitempty"` // 0-65535, default 0
	TimeoutMs    int    `json:"timeout_ms,omitempty"`    // default 5000
}

// ModbusChecker implements probe.Checker for Kind="modbus". It opens a
// Modbus TCP connection and issues a single read of one register/coil,
// reporting success when the device returns a non-exception response.
type ModbusChecker struct {
	dialer PingDialer
}

// NewModbusChecker returns a ModbusChecker wired to a real dialer.
func NewModbusChecker() *ModbusChecker {
	return &ModbusChecker{dialer: realPingDialer{}}
}

// WithModbusDialer swaps the dialer (for tests).
func (c *ModbusChecker) WithModbusDialer(d PingDialer) *ModbusChecker {
	c.dialer = d
	return c
}

// Kind returns probe.KindMODBUS.
func (c *ModbusChecker) Kind() string { return probe.KindMODBUS }

// RequiredCapabilities returns nil; Modbus TCP needs no special hardware.
func (c *ModbusChecker) RequiredCapabilities() []string { return nil }

// Run reads one register from the Modbus device at Target:Port and
// reports success on a valid, non-exception response.
func (c *ModbusChecker) Run(ctx context.Context, p probe.Probe) probe.Result {
	params := parseModbusParams(p.Params)

	port := params.Port
	if port == 0 {
		port = defaultModbusPort
	}
	if params.UnitID < 0 || params.UnitID > modbusUnitIDMax {
		return modbusFailure(p, 0, fmt.Sprintf(
			"modbus probe requires params.unit_id in range 0-%d", modbusUnitIDMax))
	}
	if params.TestRegister < 0 || params.TestRegister > modbusMaxRegister {
		return modbusFailure(p, 0, fmt.Sprintf(
			"modbus probe requires params.test_register in range 0-%d", modbusMaxRegister))
	}

	timeout := defaultModbusTimeout
	if params.TimeoutMs > 0 {
		timeout = time.Duration(params.TimeoutMs) * time.Millisecond
	}
	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	addr := net.JoinHostPort(p.Target, strconv.Itoa(port))
	start := time.Now()
	conn, err := c.dialer.Dial(dialCtx, "tcp", addr)
	if err != nil {
		return modbusFailure(p, time.Since(start), err.Error())
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	functionCode := modbusFunctionCode(params.RegisterType)
	request := buildModbusTCPRequest(
		uint8(params.UnitID),
		functionCode,
		uint16(params.TestRegister),
		1,
	)
	if _, writeErr := conn.Write(request); writeErr != nil {
		return modbusFailure(p, time.Since(start), "write request: "+writeErr.Error())
	}

	respBuf := make([]byte, modbusResponseBufferSize)
	n, readErr := conn.Read(respBuf)
	if readErr != nil {
		return modbusFailure(p, time.Since(start), "read response: "+readErr.Error())
	}
	latencyMs := float64(time.Since(start).Milliseconds())

	if n < modbusMinResponseLen {
		return modbusFailure(p, time.Since(start), "response too short")
	}
	// Exception response: function code byte has the high bit set.
	if respBuf[7]&modbusExceptionFlag != 0 {
		return modbusFailure(p, time.Since(start), "modbus exception: "+modbusExceptionString(respBuf[8]))
	}

	meta := map[string]any{
		metaKeyAddr:     addr,
		"unit_id":       params.UnitID,
		"function_code": functionCode,
	}
	// Register reads carry a 16-bit value at the data offset; surface it.
	if (functionCode == modbusFCReadHoldingRegisters || functionCode == modbusFCReadInputRegisters) && n >= 11 {
		meta["register_value"] = int(binary.BigEndian.Uint16(respBuf[9:11]))
	}
	metaJSON, _ := json.Marshal(meta)

	return probe.Result{
		ProbeID:   p.ID,
		ClientID:  p.ClientID,
		Kind:      p.Kind,
		Timestamp: time.Now().UTC(),
		Success:   true,
		LatencyMs: latencyMs,
		Metadata:  metaJSON,
	}
}

// modbusFailure builds a failed Result with the measured latency so far.
func modbusFailure(p probe.Probe, elapsed time.Duration, msg string) probe.Result {
	return probe.Result{
		ProbeID:   p.ID,
		ClientID:  p.ClientID,
		Kind:      p.Kind,
		Timestamp: time.Now().UTC(),
		Success:   false,
		LatencyMs: float64(elapsed.Milliseconds()),
		Error:     msg,
	}
}

// modbusFunctionCode maps a register-type selector to its read function
// code, defaulting to holding registers for any unrecognized value.
func modbusFunctionCode(registerType string) uint8 {
	switch strings.ToLower(registerType) {
	case modbusRegisterCoil:
		return modbusFCReadCoils
	case modbusRegisterDiscrete:
		return modbusFCReadDiscreteInputs
	case modbusRegisterInput:
		return modbusFCReadInputRegisters
	default:
		return modbusFCReadHoldingRegisters
	}
}

// buildModbusTCPRequest assembles a Modbus TCP/IP ADU: a 7-byte MBAP
// header (transaction ID, protocol ID 0, length, unit ID) followed by
// the read PDU (function code, start address, quantity).
func buildModbusTCPRequest(unitID, functionCode uint8, startAddr, quantity uint16) []byte {
	request := make([]byte, modbusRequestSize)
	binary.BigEndian.PutUint16(request[0:2], 1) // transaction ID (arbitrary)
	binary.BigEndian.PutUint16(request[2:4], 0) // protocol ID (0 = Modbus)
	binary.BigEndian.PutUint16(request[4:6], modbusPDULength)
	request[6] = unitID
	request[7] = functionCode
	binary.BigEndian.PutUint16(request[8:10], startAddr)
	binary.BigEndian.PutUint16(request[10:12], quantity)
	return request
}

// modbusExceptionString renders a Modbus exception code (Modbus
// Application Protocol V1.1b3 §7) as a human-readable reason.
func modbusExceptionString(code uint8) string {
	exceptions := map[uint8]string{
		0x01: "Illegal Function",
		0x02: "Illegal Data Address",
		0x03: "Illegal Data Value",
		0x04: "Server Device Failure",
		0x05: "Acknowledge",
		0x06: "Server Device Busy",
		0x08: "Memory Parity Error",
		0x0A: "Gateway Path Unavailable",
		0x0B: "Gateway Target Device Failed to Respond",
	}
	if msg, ok := exceptions[code]; ok {
		return msg
	}
	return fmt.Sprintf("Unknown exception (0x%02X)", code)
}

func parseModbusParams(raw json.RawMessage) ModbusParams {
	if len(raw) == 0 {
		return ModbusParams{}
	}
	var p ModbusParams
	_ = json.Unmarshal(raw, &p)
	return p
}
