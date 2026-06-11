package checkers_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/probe"
	"github.com/MustardSeedNetworks/seed/internal/probe/checkers"
)

func TestFileShareChecker_Kind(t *testing.T) {
	t.Parallel()
	if checkers.NewFileShareChecker().Kind() != "fileshare" {
		t.Errorf("Kind != fileshare")
	}
}

func TestFileShareChecker_Run_SMBSuccess(t *testing.T) {
	t.Parallel()
	params, _ := json.Marshal(checkers.FileShareParams{
		Protocol: "smb",
		Share:    "//nas.example.com/backup",
	})
	// modbusDialer + modbusFakeConn already exist in this package (modbus_test.go);
	// reuse them as the PingDialer seam — a successful connect needs only Close().
	c := checkers.NewFileShareChecker().WithFileShareDialer(&modbusDialer{conn: &modbusFakeConn{}})
	r := c.Run(context.Background(), probe.Probe{
		Kind:   "fileshare",
		Target: "nas.example.com",
		Params: params,
	})
	if !r.Success {
		t.Fatalf("Success = false on successful SMB dial: %s", r.Error)
	}
	var meta map[string]any
	if err := json.Unmarshal(r.Metadata, &meta); err != nil {
		t.Fatalf("metadata not JSON: %v", err)
	}
	if got := meta["protocol"]; got != "smb" {
		t.Errorf("metadata protocol = %v, want smb", got)
	}
	// addr must contain the SMB port 445.
	addr, _ := meta["addr"].(string)
	if addr == "" {
		t.Errorf("metadata addr missing or empty")
	}
	if !fileShareAddrContains(addr, "445") {
		t.Errorf("metadata addr %q should contain port 445", addr)
	}
}

func TestFileShareChecker_Run_DialError(t *testing.T) {
	t.Parallel()
	params, _ := json.Marshal(checkers.FileShareParams{Protocol: "nfs"})
	c := checkers.NewFileShareChecker().WithFileShareDialer(&modbusDialer{err: errors.New("connection refused")})
	r := c.Run(context.Background(), probe.Probe{
		Kind:   "fileshare",
		Target: "nfs.example.com",
		Params: params,
	})
	if r.Success {
		t.Error("Success = true on dial error; want false")
	}
	if r.Error == "" {
		t.Error("expected non-empty Error on dial failure")
	}
}

func TestFileShareChecker_Run_UnsupportedProtocol(t *testing.T) {
	t.Parallel()
	params, _ := json.Marshal(checkers.FileShareParams{Protocol: "ftp"})
	c := checkers.NewFileShareChecker().WithFileShareDialer(&modbusDialer{conn: &modbusFakeConn{}})
	r := c.Run(context.Background(), probe.Probe{
		Kind:   "fileshare",
		Target: "host.example.com",
		Params: params,
	})
	if r.Success {
		t.Error("Success = true on unsupported protocol; want false")
	}
	if r.Error == "" {
		t.Error("expected an error message for unsupported protocol ftp")
	}
}

// fileShareAddrContains is a simple substring check — avoids importing strings
// for a single test helper.
func fileShareAddrContains(s, sub string) bool {
	if len(sub) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
