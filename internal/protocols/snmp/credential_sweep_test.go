package snmp_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/config"
	snmppkg "github.com/MustardSeedNetworks/seed/internal/protocols/snmp"
)

func testCfg(communities []string, v3 int) *config.SNMPConfig {
	creds := make([]config.SNMPv3Credential, v3)
	for i := range creds {
		creds[i].Username = "user"
	}
	return &config.SNMPConfig{Communities: communities, V3Credentials: creds}
}

// TestSweepPrefersV3 — a device that answers both should be talked to over the
// authenticated transport, not a community string in cleartext on the wire.
func TestSweepPrefersV3(t *testing.T) {
	var order []string
	got, err := snmppkg.SweepCredentials(context.Background(), testCfg([]string{"public"}, 1), "x",
		func(*config.SNMPv3Credential) (string, error) {
			order = append(order, "v3")
			return "v3-result", nil
		},
		func(string) (string, error) {
			order = append(order, "v2c")
			return "v2c-result", nil
		},
	)
	if err != nil || got != "v3-result" {
		t.Fatalf("got %q, %v; want v3-result", got, err)
	}
	if len(order) != 1 || order[0] != "v3" {
		t.Errorf("call order %v; v2c should not have been tried", order)
	}
}

// TestSweepFallsBackToEveryCommunity — the fallback must try them all, not
// stop at the first failure.
func TestSweepFallsBackToEveryCommunity(t *testing.T) {
	tried := 0
	got, err := snmppkg.SweepCredentials(context.Background(),
		testCfg([]string{"wrong-1", "wrong-2", "right"}, 1), "x",
		func(*config.SNMPv3Credential) (string, error) {
			return "", errors.New("v3 refused")
		},
		func(community string) (string, error) {
			tried++
			if community == "right" {
				return "ok", nil
			}
			return "", errors.New("bad community")
		},
	)
	if err != nil || got != "ok" {
		t.Fatalf("got %q, %v; want ok", got, err)
	}
	if tried != 3 {
		t.Errorf("tried %d communities, want 3", tried)
	}
}

// TestSweepStopsOnCancelledContext is the behaviour the hand-written loops did
// not have. Against an unreachable host each credential burns a full timeout,
// so without this one dead host costs one timeout per configured credential.
func TestSweepStopsOnCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	attempts := 0
	_, err := snmppkg.SweepCredentials(ctx, testCfg([]string{"a", "b", "c"}, 2), "x",
		func(*config.SNMPv3Credential) (string, error) {
			attempts++
			return "", errors.New("nope")
		},
		func(string) (string, error) {
			attempts++
			return "", errors.New("nope")
		},
	)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
	if attempts != 0 {
		t.Errorf("made %d attempts after cancellation, want 0", attempts)
	}
}

// TestSweepExhaustedErrorNamesNoSecrets — the failure says what was being done,
// never which community strings were tried.
func TestSweepExhaustedErrorNamesNoSecrets(t *testing.T) {
	_, err := snmppkg.SweepCredentials(context.Background(),
		testCfg([]string{"s3cret-community"}, 0), "query LLDP neighbors",
		func(*config.SNMPv3Credential) (string, error) { return "", errors.New("no") },
		func(string) (string, error) { return "", errors.New("no") },
	)
	if !errors.Is(err, snmppkg.ErrNoCredentialSucceeded) {
		t.Fatalf("err = %v, want snmppkg.ErrNoCredentialSucceeded", err)
	}
	if got := err.Error(); strings.Contains(got, "s3cret-community") {
		t.Errorf("error message leaked a community string: %q", got)
	}
}

func TestSweepNilConfig(t *testing.T) {
	_, err := snmppkg.SweepCredentials[string](context.Background(), nil, "x", nil, nil)
	if err == nil {
		t.Error("nil config produced no error")
	}
}
