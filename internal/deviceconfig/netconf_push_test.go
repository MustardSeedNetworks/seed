package deviceconfig_test

import (
	"bufio"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"github.com/MustardSeedNetworks/seed/internal/deviceconfig"
)

const (
	capCandidate = "urn:ietf:params:netconf:capability:candidate:1.0"
	capConfirmed = "urn:ietf:params:netconf:capability:confirmed-commit:1.1"
	capValidate  = "urn:ietf:params:netconf:capability:validate:1.1"
)

var messageID = regexp.MustCompile(`message-id="(\d+)"`)

// netconfDatastores is a NETCONF server with a running and a candidate
// datastore: edit-config replaces the candidate with candidateAfterEdit, and
// an operation named in fail answers with an rpc-error.
type netconfDatastores struct {
	capabilities       []string
	running            string
	candidateAfterEdit string
	fail               map[string]string

	mu         sync.Mutex
	operations []string
	candidate  string
	done       chan struct{}
}

func (d *netconfDatastores) serve(ch ssh.Channel) {
	defer close(d.done)
	r := bufio.NewReader(ch)
	if _, err := readUntil(r, eom); err != nil {
		return
	}
	var caps strings.Builder
	for _, c := range d.capabilities {
		caps.WriteString("<capability>" + c + "</capability>")
	}
	_, _ = fmt.Fprintf(ch, `<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"><capabilities>%s`+
		`</capabilities><session-id>9</session-id></hello>%s`, caps.String(), eom)
	d.candidate = d.running
	for {
		msg, err := readUntil(r, eom)
		if err != nil {
			return
		}
		op, reply := d.answer(msg)
		id := messageID.FindStringSubmatch(msg)[1]
		_, _ = fmt.Fprintf(ch, `<rpc-reply message-id="%s" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">%s`+
			`</rpc-reply>%s`, id, reply, eom)
		if op == "close-session" {
			return
		}
	}
}

func (d *netconfDatastores) answer(msg string) (string, string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	var op string
	for _, name := range []string{
		"lock", "unlock", "edit-config", "validate", "discard-changes", "close-session", "get-config",
	} {
		if strings.Contains(msg, "<"+name+">") || strings.Contains(msg, "<"+name+"/>") {
			op = name
			break
		}
	}
	if op == "get-config" && strings.Contains(msg, "<candidate/></source>") {
		op = "get-config candidate"
	}
	d.operations = append(d.operations, op)
	if tag, ok := d.fail[op]; ok {
		return op, `<rpc-error><error-type>protocol</error-type><error-tag>` + tag +
			`</error-tag><error-severity>error</error-severity></rpc-error>`
	}
	switch op {
	case "get-config":
		return op, "<data>" + d.running + "</data>"
	case "get-config candidate":
		return op, "<data>" + d.candidate + "</data>"
	case "edit-config":
		d.candidate = d.candidateAfterEdit
	case "discard-changes":
		d.candidate = d.running
	}
	return op, "<ok/>"
}

func (d *netconfDatastores) ops() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.operations...)
}

func startNETCONFDatastores(t *testing.T, d *netconfDatastores) deviceconfig.DryRunRequest {
	t.Helper()
	d.done = make(chan struct{})
	dev := startSSHDevice(t, deviceOpts{password: "wr1te", netconf: d.serve})
	fr := dev.request(t, "wr1te", "", dev.fingerprint)
	return deviceconfig.DryRunRequest{
		Host: fr.Host, Port: fr.Port, User: "pusher", Password: "wr1te", PinnedHostKey: dev.fingerprint,
		Platform: deviceconfig.PlatformNETCONF, Mode: deviceconfig.PushMerge,
		Candidate: `<system><host-name>core</host-name></system>`,
	}
}

// TestNETCONFDryRunLeavesTheDeviceAsItWas: the candidate is loaded under a
// lock, validated, read back and discarded, and the diff is the device's two
// datastores rendered one leaf per line.
func TestNETCONFDryRunLeavesTheDeviceAsItWas(t *testing.T) {
	t.Parallel()
	d := &netconfDatastores{
		capabilities:       []string{base10, capCandidate, capConfirmed, capValidate},
		running:            `<system><host-name>edge</host-name><ntp><server>192.0.2.1</server></ntp></system>`,
		candidateAfterEdit: `<system><host-name>core</host-name><ntp><server>192.0.2.1</server></ntp></system>`,
	}
	req := startNETCONFDatastores(t, d)

	res, err := deviceconfig.SSHFetcher{}.DryRun(fetchCtx(t), req)
	require.NoError(t, err)
	<-d.done
	require.Equal(t, []string{
		"lock", "get-config", "edit-config", "validate", "get-config candidate", "discard-changes", "unlock",
		"close-session",
	}, d.ops())
	require.Equal(t, d.running, d.candidate, "the candidate is discarded")
	require.Equal(t, 1, res.Added)
	require.Equal(t, 1, res.Removed)
	require.Contains(t, res.Diff, "-  <host-name>edge</host-name>\n")
	require.Contains(t, res.Diff, "+  <host-name>core</host-name>\n")
}

func TestNETCONFDryRunFailures(t *testing.T) {
	t.Parallel()
	full := []string{base10, capCandidate, capConfirmed}
	tests := []struct {
		name    string
		caps    []string
		fail    map[string]string
		refused string
		err     string
		ops     []string
	}{
		{
			name:    "no candidate datastore",
			caps:    []string{base10, capConfirmed},
			refused: "candidate:1.0",
		},
		{
			name:    "no confirmed commit",
			caps:    []string{base10, capCandidate},
			refused: "confirmed-commit:1.1",
		},
		{
			name: "someone else holds the candidate",
			caps: full,
			fail: map[string]string{"lock": "lock-denied"},
			err:  "lock: lock-denied",
			ops:  []string{"lock"},
		},
		{
			name: "the device rejects the candidate, which is still discarded",
			caps: full,
			fail: map[string]string{"edit-config": "invalid-value"},
			err:  "edit-config: invalid-value",
			ops:  []string{"lock", "get-config", "edit-config", "discard-changes", "unlock", "close-session"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := &netconfDatastores{capabilities: tc.caps, running: "<a/>", candidateAfterEdit: "<b/>", fail: tc.fail}
			req := startNETCONFDatastores(t, d)
			_, err := deviceconfig.SSHFetcher{}.DryRun(fetchCtx(t), req)
			if tc.refused != "" {
				refused, ok := errors.AsType[*deviceconfig.PlanRefusedError](err)
				require.True(t, ok, "%v", err)
				require.Contains(t, refused.Reason, tc.refused)
				require.Empty(t, d.ops(), "nothing is sent to a device that cannot revert")
				return
			}
			require.ErrorIs(t, err, deviceconfig.ErrCommand)
			require.ErrorContains(t, err, tc.err)
			if tc.ops[len(tc.ops)-1] == "close-session" {
				<-d.done
			}
			require.Equal(t, tc.ops, d.ops()[:len(tc.ops)])
		})
	}
}

func TestDryRunRefusesAPlatformWithoutOne(t *testing.T) {
	t.Parallel()
	for _, p := range []deviceconfig.Platform{deviceconfig.PlatformCiscoIOS, deviceconfig.PlatformCiscoNXOS} {
		_, err := deviceconfig.SSHFetcher{}.DryRun(fetchCtx(t), deviceconfig.DryRunRequest{Platform: p})
		_, ok := errors.AsType[*deviceconfig.PlanRefusedError](err)
		require.True(t, ok, "%s: %v", p, err)
	}
}
