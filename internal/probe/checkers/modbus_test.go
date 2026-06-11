package checkers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/probe"
	"github.com/MustardSeedNetworks/seed/internal/probe/checkers"
)

// modbusFakeConn delivers a canned Modbus TCP response and records the
// request bytes the checker wrote.
type modbusFakeConn struct {
	mu       sync.Mutex
	response []byte
	written  []byte
	pos      int
}

func (m *modbusFakeConn) Read(b []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.pos >= len(m.response) {
		return 0, errors.New("modbusFakeConn: response exhausted")
	}
	n := copy(b, m.response[m.pos:])
	m.pos += n
	return n, nil
}

func (m *modbusFakeConn) Write(b []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.written = append(m.written, b...)
	return len(b), nil
}

func (*modbusFakeConn) Close() error                     { return nil }
func (*modbusFakeConn) LocalAddr() net.Addr              { return nil }
func (*modbusFakeConn) RemoteAddr() net.Addr             { return nil }
func (*modbusFakeConn) SetDeadline(time.Time) error      { return nil }
func (*modbusFakeConn) SetReadDeadline(time.Time) error  { return nil }
func (*modbusFakeConn) SetWriteDeadline(time.Time) error { return nil }

type modbusDialer struct {
	conn net.Conn
	err  error
}

func (d *modbusDialer) Dial(_ context.Context, _, _ string) (net.Conn, error) {
	return d.conn, d.err
}

// makeModbusReadResponse builds a holding-register read response with
// the given 16-bit value: MBAP header + FC 0x03 + byte count 2 + value.
func makeModbusReadResponse(value uint16) []byte {
	return []byte{
		0x00, 0x01, // transaction ID
		0x00, 0x00, // protocol ID
		0x00, 0x05, // length
		0x00,             // unit ID
		0x03,             // function code (read holding registers)
		0x02,             // byte count
		byte(value >> 8), // value hi
		byte(value),      // value lo
	}
}

// makeModbusException builds an exception response: FC with high bit set
// plus an exception code.
func makeModbusException(fc, code uint8) []byte {
	return []byte{
		0x00, 0x01,
		0x00, 0x00,
		0x00, 0x03,
		0x00,
		fc | 0x80,
		code,
	}
}

func TestModbusChecker_Kind(t *testing.T) {
	t.Parallel()
	if checkers.NewModbusChecker().Kind() != "modbus" {
		t.Errorf("Kind != modbus")
	}
}

func TestModbusChecker_Run_Success(t *testing.T) {
	t.Parallel()
	conn := &modbusFakeConn{response: makeModbusReadResponse(0x1234)}
	c := checkers.NewModbusChecker().WithModbusDialer(&modbusDialer{conn: conn})
	r := c.Run(context.Background(), probe.Probe{Kind: "modbus", Target: "plc.example.com"})
	if !r.Success {
		t.Fatalf("Success = false on valid read: %s", r.Error)
	}
	// The request is a well-formed Modbus TCP ADU: protocol ID 0, FC 0x03.
	if len(conn.written) != 12 || conn.written[2] != 0x00 || conn.written[3] != 0x00 || conn.written[7] != 0x03 {
		t.Errorf("malformed request ADU: %v", conn.written)
	}
	var meta map[string]any
	if err := json.Unmarshal(r.Metadata, &meta); err != nil {
		t.Fatalf("metadata not JSON: %v", err)
	}
	if got, ok := meta["register_value"].(float64); !ok || int(got) != 0x1234 {
		t.Errorf("register_value = %v, want 4660", meta["register_value"])
	}
}

func TestModbusChecker_Run_Exception(t *testing.T) {
	t.Parallel()
	conn := &modbusFakeConn{response: makeModbusException(0x03, 0x02)}
	c := checkers.NewModbusChecker().WithModbusDialer(&modbusDialer{conn: conn})
	r := c.Run(context.Background(), probe.Probe{Kind: "modbus", Target: "plc.example.com"})
	if r.Success {
		t.Error("Success = true on exception response; want false")
	}
	if r.Error == "" {
		t.Error("expected an exception error message")
	}
}

func TestModbusChecker_Run_DialError(t *testing.T) {
	t.Parallel()
	c := checkers.NewModbusChecker().WithModbusDialer(&modbusDialer{err: errors.New("refused")})
	r := c.Run(context.Background(), probe.Probe{Kind: "modbus", Target: "down"})
	if r.Success {
		t.Error("Success = true; dial error should fail")
	}
}

func TestModbusChecker_Run_InvalidUnitID(t *testing.T) {
	t.Parallel()
	params, _ := json.Marshal(checkers.ModbusParams{UnitID: 999})
	c := checkers.NewModbusChecker().WithModbusDialer(&modbusDialer{conn: &modbusFakeConn{}})
	r := c.Run(context.Background(), probe.Probe{Kind: "modbus", Target: "plc", Params: params})
	if r.Success {
		t.Error("Success = true with out-of-range unit_id; want validation failure")
	}
}

func TestModbusChecker_Run_RegisterTypeSelectsFunctionCode(t *testing.T) {
	t.Parallel()
	conn := &modbusFakeConn{response: makeModbusException(0x01, 0x01)}
	params, _ := json.Marshal(checkers.ModbusParams{RegisterType: "coil"})
	c := checkers.NewModbusChecker().WithModbusDialer(&modbusDialer{conn: conn})
	_ = c.Run(context.Background(), probe.Probe{Kind: "modbus", Target: "plc", Params: params})
	if len(conn.written) < 8 || conn.written[7] != 0x01 {
		t.Errorf("coil register_type should select FC 0x01; got %v", conn.written)
	}
}
