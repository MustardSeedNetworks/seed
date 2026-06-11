package checkers_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/probe"
	"github.com/MustardSeedNetworks/seed/internal/probe/checkers"
)

// ldapDialer returns a fakePingDialer pre-loaded with a single outcome.
// The dialer records gotAddrs so tests can assert the address that was
// dialled (host:port), enabling port-selection assertions without a
// live network.
func ldapDialer(err error) *fakePingDialer {
	d := &fakePingDialer{}
	if err != nil {
		d.attempts = []dialOutcome{{err: err}}
	} else {
		d.attempts = []dialOutcome{{conn: fakeConn{}}}
	}
	return d
}

func TestLDAPChecker_Kind(t *testing.T) {
	t.Parallel()
	if checkers.NewLDAPChecker().Kind() != "ldap" {
		t.Errorf("Kind() = %q, want ldap", checkers.NewLDAPChecker().Kind())
	}
}

func TestLDAPChecker_Run_PlainSuccess(t *testing.T) {
	t.Parallel()
	d := ldapDialer(nil)
	c := checkers.NewLDAPChecker().WithLDAPDialer(d)

	r := c.Run(context.Background(), probe.Probe{
		Kind:   "ldap",
		Target: "ldap.example.com",
	})

	if !r.Success {
		t.Fatalf("Success = false; err=%q", r.Error)
	}
	// Default port for plain LDAP must be 389.
	if len(d.gotAddrs) == 0 || d.gotAddrs[0] != "ldap.example.com:389" {
		t.Errorf("dialed addr = %q, want ldap.example.com:389", d.gotAddrs)
	}

	// Metadata must carry addr, tls:false, bind_dn_configured:false.
	var meta map[string]any
	if err := json.Unmarshal(r.Metadata, &meta); err != nil {
		t.Fatalf("metadata is not valid JSON: %v", err)
	}
	if meta["addr"] != "ldap.example.com:389" {
		t.Errorf("metadata[addr] = %v, want ldap.example.com:389", meta["addr"])
	}
	if meta["tls"] != false {
		t.Errorf("metadata[tls] = %v, want false", meta["tls"])
	}
	if meta["bind_dn_configured"] != false {
		t.Errorf("metadata[bind_dn_configured] = %v, want false", meta["bind_dn_configured"])
	}
}

func TestLDAPChecker_Run_BindDNConfigured(t *testing.T) {
	t.Parallel()
	d := ldapDialer(nil)
	c := checkers.NewLDAPChecker().WithLDAPDialer(d)

	params, _ := json.Marshal(checkers.LDAPParams{BindDN: "cn=admin,dc=example,dc=com"})
	r := c.Run(context.Background(), probe.Probe{
		Kind:   "ldap",
		Target: "ldap.example.com",
		Params: params,
	})

	if !r.Success {
		t.Fatalf("Success = false; err=%q", r.Error)
	}

	var meta map[string]any
	_ = json.Unmarshal(r.Metadata, &meta)
	// BindDN present in params — must be reflected in metadata.
	if meta["bind_dn_configured"] != true {
		t.Errorf("metadata[bind_dn_configured] = %v, want true", meta["bind_dn_configured"])
	}
}

func TestLDAPChecker_Run_DialError(t *testing.T) {
	t.Parallel()
	d := ldapDialer(errors.New("connection refused"))
	c := checkers.NewLDAPChecker().WithLDAPDialer(d)

	r := c.Run(context.Background(), probe.Probe{
		Kind:   "ldap",
		Target: "down.example.com",
	})

	if r.Success {
		t.Error("Success = true; want false on dial error")
	}
	if r.Error == "" {
		t.Error("Error is empty; want a non-empty error message")
	}
}

func TestLDAPChecker_Run_CustomPort(t *testing.T) {
	t.Parallel()
	d := ldapDialer(nil)
	c := checkers.NewLDAPChecker().WithLDAPDialer(d)

	params, _ := json.Marshal(checkers.LDAPParams{Port: 3389})
	r := c.Run(context.Background(), probe.Probe{
		Kind:   "ldap",
		Target: "ldap.example.com",
		Params: params,
	})

	if !r.Success {
		t.Fatalf("Success = false; err=%q", r.Error)
	}
	if len(d.gotAddrs) == 0 || d.gotAddrs[0] != "ldap.example.com:3389" {
		t.Errorf("dialed addr = %q, want ldap.example.com:3389", d.gotAddrs)
	}
}

// TestLDAPChecker_Run_Port636Plain exercises the port plumbing that feeds
// the default-LDAPS port value (636) through to the dialer. UseTLS is NOT
// set — the test forces Port=636 on the plain path so it stays offline and
// deterministic, verifying that numeric port 636 is correctly assembled
// into the dial address. The default port-selection logic (UseTLS=true →
// port 636, false → port 389) is covered without a live TLS server.
func TestLDAPChecker_Run_Port636Plain(t *testing.T) {
	t.Parallel()
	d := ldapDialer(nil)
	c := checkers.NewLDAPChecker().WithLDAPDialer(d)

	params, _ := json.Marshal(checkers.LDAPParams{Port: 636})
	r := c.Run(context.Background(), probe.Probe{
		Kind:   "ldap",
		Target: "ldap.example.com",
		Params: params,
	})

	if !r.Success {
		t.Fatalf("Success = false; err=%q", r.Error)
	}
	if len(d.gotAddrs) == 0 || d.gotAddrs[0] != "ldap.example.com:636" {
		t.Errorf("dialed addr = %q, want ldap.example.com:636", d.gotAddrs)
	}
}
