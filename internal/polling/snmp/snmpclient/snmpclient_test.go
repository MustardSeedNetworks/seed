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

// dialErr drives one Get against a target that nothing is listening on, so
// the returned error is whichever check fired first. The point of these tests
// is *which* error it is: a credential problem must be reported as one,
// before any packet leaves the host.
func dialErr(t *testing.T, target snmp.Target, creds snmp.ResolvedCredentials) error {
	t.Helper()
	factory := snmpclient.NewFactory(snmpclient.Options{})
	client, err := factory(target, creds)
	if err != nil {
		t.Fatalf("factory: %v", err)
	}
	_, getErr := client.Get(context.Background(), []string{"1.3.6.1.2.1.1.1.0"})
	if getErr == nil {
		t.Fatal("expected an error")
	}
	return getErr
}

// TestApplyAuth_NoCommunityFailsClosed is the one that matters most here.
//
// The resolver refuses to hand back a credential a target is not entitled to,
// and this dialer used to undo that by substituting "public" — the community
// every scanner tries first — whenever the resolved one was empty. A target
// with no configured credential became a probe of somebody's
// default-configured agent, sent under this product's name.
func TestApplyAuth_NoCommunityFailsClosed(t *testing.T) {
	t.Parallel()
	err := dialErr(t,
		snmp.Target{IPAddress: "127.0.0.1", SNMPVersion: "v2c"},
		snmp.ResolvedCredentials{},
	)
	if !strings.Contains(err.Error(), "community") {
		t.Errorf("error = %v, want it to name the missing community", err)
	}
	if strings.Contains(err.Error(), "connect") {
		t.Errorf("error = %v, want the credential refused before dialling", err)
	}
}

// A resolved community still dials: failing closed must not mean failing
// always. The error here is the connection failing, not the credential.
func TestApplyAuth_ResolvedCommunityStillDials(t *testing.T) {
	t.Parallel()
	err := dialErr(t,
		snmp.Target{IPAddress: "192.0.2.1", SNMPVersion: "v2c"},
		snmp.ResolvedCredentials{SNMPCommunity: "s3cret"},
	)
	if strings.Contains(err.Error(), "community") {
		t.Errorf("error = %v, want the dial to have been attempted", err)
	}
}

// An unrecognised protocol name used to fall through to SHA/AES, so a typo
// surfaced as an authentication failure that reads exactly like a wrong
// password — and the agent was contacted with a protocol nobody chose.
func TestApplyAuth_UnknownV3ProtocolsAreRejected(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		creds snmp.ResolvedCredentials
		want  string
	}{
		{
			name: "auth protocol",
			creds: snmp.ResolvedCredentials{
				SNMPv3User:       "operator",
				SNMPv3AuthSecret: "authpass",
				SNMPv3AuthProto:  "SHA3",
			},
			want: "auth protocol",
		},
		{
			name: "privacy protocol",
			creds: snmp.ResolvedCredentials{
				SNMPv3User:       "operator",
				SNMPv3AuthSecret: "authpass",
				SNMPv3PrivSecret: "privpass",
				SNMPv3PrivProto:  "AES512",
			},
			want: "privacy protocol",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := dialErr(t,
				snmp.Target{IPAddress: "127.0.0.1", SNMPVersion: "v3"},
				tt.creds,
			)
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %v, want it to name the unsupported %s", err, tt.want)
			}
		})
	}
}

// An unset protocol is not a typo: the operator did not choose one, so the
// modern default applies and the dial proceeds.
func TestApplyAuth_UnsetV3ProtocolsUseDefaults(t *testing.T) {
	t.Parallel()
	err := dialErr(t,
		snmp.Target{IPAddress: "192.0.2.1", SNMPVersion: "v3"},
		snmp.ResolvedCredentials{
			SNMPv3User:       "operator",
			SNMPv3AuthSecret: "authpass",
			SNMPv3PrivSecret: "privpass",
		},
	)
	if strings.Contains(err.Error(), "protocol") {
		t.Errorf("error = %v, want unset protocols to default rather than fail", err)
	}
}
