package api

import (
	"net"
	"path/filepath"
	"testing"

	"github.com/krisarmstrong/seed/internal/database"
)

// freeAddr returns a free UDP loopback address as a "host:port"
// string. Used to seed env vars for listener wire-up tests so the
// integration doesn't collide with real :514 / :162.
func freeAddr(t *testing.T) string {
	t.Helper()
	c, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("freeAddr: %v", err)
	}
	addr := c.LocalAddr().String()
	_ = c.Close()
	return addr
}

func newTestDB(t *testing.T) *database.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "seed.db")
	db, err := database.Open(path)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestInitListeners_NoEnvVarsRegistersNoListeners(t *testing.T) {
	t.Setenv("SEED_SYSLOG_BIND", "")
	t.Setenv("SEED_SNMP_TRAP_BIND", "")

	services := NewServiceContainer()
	before := len(services.Engines.Engines())
	initListeners(services, newTestDB(t))
	after := len(services.Engines.Engines())

	if after != before {
		t.Errorf("engines after init = %d, want %d (no listeners enabled)", after, before)
	}
}

func TestInitListeners_SyslogEnvVarRegistersListener(t *testing.T) {
	t.Setenv("SEED_SYSLOG_BIND", freeAddr(t))
	t.Setenv("SEED_SNMP_TRAP_BIND", "")

	services := NewServiceContainer()
	initListeners(services, newTestDB(t))

	names := make(map[string]bool)
	for _, e := range services.Engines.Engines() {
		names[e.Name()] = true
	}
	if !names["syslog-udp"] {
		t.Errorf("syslog-udp not registered; got engines = %v", names)
	}
	if names["snmp-trap-v2c"] {
		t.Errorf("snmp-trap-v2c should NOT be registered when env unset")
	}
}

func TestInitListeners_SnmpTrapEnvVarRegistersListener(t *testing.T) {
	t.Setenv("SEED_SYSLOG_BIND", "")
	t.Setenv("SEED_SNMP_TRAP_BIND", freeAddr(t))

	services := NewServiceContainer()
	initListeners(services, newTestDB(t))

	names := make(map[string]bool)
	for _, e := range services.Engines.Engines() {
		names[e.Name()] = true
	}
	if !names["snmp-trap-v2c"] {
		t.Errorf("snmp-trap-v2c not registered; got engines = %v", names)
	}
}

func TestInitListeners_BothEnvVarsRegistersBoth(t *testing.T) {
	t.Setenv("SEED_SYSLOG_BIND", freeAddr(t))
	t.Setenv("SEED_SNMP_TRAP_BIND", freeAddr(t))

	services := NewServiceContainer()
	initListeners(services, newTestDB(t))

	names := make(map[string]bool)
	for _, e := range services.Engines.Engines() {
		names[e.Name()] = true
	}
	if !names["syslog-udp"] || !names["snmp-trap-v2c"] {
		t.Errorf("expected both listeners registered, got %v", names)
	}
}
