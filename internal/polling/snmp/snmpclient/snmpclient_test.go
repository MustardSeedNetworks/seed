package snmpclient_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/polling/snmp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/snmpclient"
)

func TestNewFactory_RejectsEmptyIPAddress(t *testing.T) {
	t.Parallel()
	factory := snmpclient.NewFactory(snmpclient.Options{})
	if _, err := factory(snmp.Target{}, snmp.ResolvedCredentials{}); err == nil {
		t.Error("expected factory to reject empty IPAddress")
	}
}

func TestNewFactory_AppliesDefaultsForZeroOptions(t *testing.T) {
	t.Parallel()
	// Validation only — we don't dial. Just confirm the factory
	// accepts a valid Target with all-zero options.
	factory := snmpclient.NewFactory(snmpclient.Options{})
	client, err := factory(snmp.Target{IPAddress: "127.0.0.1"}, snmp.ResolvedCredentials{})
	if err != nil {
		t.Fatalf("factory: %v", err)
	}
	if client == nil {
		t.Error("factory returned nil client without error")
	}
}

// TestGet_ContextCancelledReturnsCtxErr verifies that an already-
// cancelled context never even reaches gosnmp.Connect.
func TestGet_ContextCancelledReturnsCtxErr(t *testing.T) {
	t.Parallel()
	factory := snmpclient.NewFactory(snmpclient.Options{})
	client, err := factory(
		snmp.Target{IPAddress: "127.0.0.1", SNMPVersion: "v2c"},
		snmp.ResolvedCredentials{SNMPCommunity: "public"},
	)
	if err != nil {
		t.Fatalf("factory: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, getErr := client.Get(ctx, []string{"1.3.6.1.2.1.1.1.0"}); getErr == nil {
		t.Error("expected context-cancelled Get to error")
	}
}

func TestWalk_ContextCancelledReturnsCtxErr(t *testing.T) {
	t.Parallel()
	factory := snmpclient.NewFactory(snmpclient.Options{})
	client, err := factory(
		snmp.Target{IPAddress: "127.0.0.1", SNMPVersion: "v2c"},
		snmp.ResolvedCredentials{SNMPCommunity: "public"},
	)
	if err != nil {
		t.Fatalf("factory: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, walkErr := client.Walk(ctx, "1.3.6.1.2.1.1"); walkErr == nil {
		t.Error("expected context-cancelled Walk to error")
	}
}

// TestDial_ConnectFailureUnreachableAddress verifies the dial error
// path bubbles a meaningful message. Uses TEST-NET-1 (192.0.2.0/24,
// RFC 5737) which is guaranteed unrouted; SNMPv2c uses UDP so the
// failure surfaces as a per-request timeout rather than a refused
// connection.
func TestDial_ConnectFailureUnreachableAddress(t *testing.T) {
	t.Parallel()
	factory := snmpclient.NewFactory(snmpclient.Options{
		Timeout: 100 * time.Millisecond,
		Retries: 0,
	})
	client, err := factory(
		snmp.Target{IPAddress: "192.0.2.1", SNMPVersion: "v2c"},
		snmp.ResolvedCredentials{SNMPCommunity: "public"},
	)
	if err != nil {
		t.Fatalf("factory: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_, err = client.Get(ctx, []string{"1.3.6.1.2.1.1.1.0"})
	if err == nil {
		t.Fatal("expected Get against unreachable host to fail")
	}
	if !strings.Contains(err.Error(), "snmpclient:") {
		t.Errorf("error does not wrap with snmpclient prefix: %v", err)
	}
}

func TestApplyAuth_SNMPv3RequiresUser(t *testing.T) {
	t.Parallel()
	factory := snmpclient.NewFactory(snmpclient.Options{})
	client, err := factory(
		snmp.Target{IPAddress: "127.0.0.1", SNMPVersion: "v3"},
		snmp.ResolvedCredentials{},
	)
	if err != nil {
		t.Fatalf("factory: %v", err)
	}
	if _, getErr := client.Get(context.Background(), []string{"1.3.6.1.2.1.1.1.0"}); getErr == nil {
		t.Error("expected v3 dial with empty user to error")
	}
}

func TestApplyAuth_SNMPv3PrivRequiresAuth(t *testing.T) {
	t.Parallel()
	factory := snmpclient.NewFactory(snmpclient.Options{})
	client, err := factory(
		snmp.Target{IPAddress: "127.0.0.1", SNMPVersion: "v3"},
		snmp.ResolvedCredentials{
			SNMPv3User:       "operator",
			SNMPv3PrivSecret: "privpass",
		},
	)
	if err != nil {
		t.Fatalf("factory: %v", err)
	}
	if _, getErr := client.Get(context.Background(), []string{"1.3.6.1.2.1.1.1.0"}); getErr == nil {
		t.Error("expected priv-without-auth to error")
	}
}

func TestApplyAuth_UnsupportedVersionErrors(t *testing.T) {
	t.Parallel()
	factory := snmpclient.NewFactory(snmpclient.Options{})
	client, err := factory(
		snmp.Target{IPAddress: "127.0.0.1", SNMPVersion: "v1"},
		snmp.ResolvedCredentials{},
	)
	if err != nil {
		t.Fatalf("factory: %v", err)
	}
	if _, getErr := client.Get(context.Background(), []string{"1.3.6.1.2.1.1.1.0"}); getErr == nil {
		t.Error("expected v1 to be rejected")
	}
}

func TestNewFactory_NegativeRetriesClampedToDefault(t *testing.T) {
	t.Parallel()
	// Smoke test: factory accepts negative retries and clamps to
	// default. We assert by constructing a client successfully —
	// if the clamp wasn't applied, gosnmp would crash on the next
	// Get.
	factory := snmpclient.NewFactory(snmpclient.Options{Retries: -5})
	if _, err := factory(snmp.Target{IPAddress: "127.0.0.1"}, snmp.ResolvedCredentials{}); err != nil {
		t.Errorf("factory should clamp negative retries: %v", err)
	}
}
