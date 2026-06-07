package checkers_test

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/probe"
	"github.com/MustardSeedNetworks/seed/internal/probe/checkers"
)

// dicomFakeConn delivers canned response bytes (the PDU header
// the test wants to assert on).
type dicomFakeConn struct {
	mu       sync.Mutex
	response []byte
	written  []byte
	pos      int
}

func (d *dicomFakeConn) Read(b []byte) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.pos >= len(d.response) {
		return 0, errors.New("dicomFakeConn: response exhausted")
	}
	n := copy(b, d.response[d.pos:])
	d.pos += n
	return n, nil
}

func (d *dicomFakeConn) Write(b []byte) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.written = append(d.written, b...)
	return len(b), nil
}

func (*dicomFakeConn) Close() error                     { return nil }
func (*dicomFakeConn) LocalAddr() net.Addr              { return nil }
func (*dicomFakeConn) RemoteAddr() net.Addr             { return nil }
func (*dicomFakeConn) SetDeadline(time.Time) error      { return nil }
func (*dicomFakeConn) SetReadDeadline(time.Time) error  { return nil }
func (*dicomFakeConn) SetWriteDeadline(time.Time) error { return nil }

type dicomDialer struct {
	conn net.Conn
	err  error
}

func (d *dicomDialer) Dial(_ context.Context, _, _ string) (net.Conn, error) {
	return d.conn, d.err
}

// makeAcceptPDU returns a 6-byte PDU header with type 0x02
// (A-ASSOCIATE-AC) and length 0. The DICOMChecker only reads
// the header so this is enough to satisfy a success path.
func makeAcceptPDU() []byte {
	return []byte{0x02, 0x00, 0x00, 0x00, 0x00, 0x00}
}

// makeRejectPDU returns a header with type 0x03 (A-ASSOCIATE-RJ).
func makeRejectPDU() []byte {
	return []byte{0x03, 0x00, 0x00, 0x00, 0x00, 0x00}
}

func TestDICOMChecker_Kind(t *testing.T) {
	t.Parallel()
	if checkers.NewDICOMChecker().Kind() != "dicom" {
		t.Errorf("Kind != dicom")
	}
}

func TestDICOMChecker_Run_Accept(t *testing.T) {
	t.Parallel()
	conn := &dicomFakeConn{response: makeAcceptPDU()}
	c := checkers.NewDICOMChecker().WithDICOMDialer(&dicomDialer{conn: conn})
	r := c.Run(context.Background(), probe.Probe{Kind: "dicom", Target: "modality.example.com"})
	if !r.Success {
		t.Errorf("Success = false on A-ASSOCIATE-AC: %s", r.Error)
	}
	if len(conn.written) < 6 || conn.written[0] != 0x01 {
		t.Errorf("written PDU type should be 0x01 (A-ASSOCIATE-RQ); got %v", conn.written[:1])
	}
}

func TestDICOMChecker_Run_Reject(t *testing.T) {
	t.Parallel()
	conn := &dicomFakeConn{response: makeRejectPDU()}
	c := checkers.NewDICOMChecker().WithDICOMDialer(&dicomDialer{conn: conn})
	r := c.Run(context.Background(), probe.Probe{Kind: "dicom", Target: "modality.example.com"})
	if r.Success {
		t.Error("Success = true on A-ASSOCIATE-RJ; want false")
	}
}

func TestDICOMChecker_Run_DialError(t *testing.T) {
	t.Parallel()
	c := checkers.NewDICOMChecker().WithDICOMDialer(&dicomDialer{err: errors.New("refused")})
	r := c.Run(context.Background(), probe.Probe{Kind: "dicom", Target: "down"})
	if r.Success {
		t.Error("Success = true; dial error should fail")
	}
}
