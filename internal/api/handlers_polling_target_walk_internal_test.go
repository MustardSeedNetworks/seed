package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/polling"
	snmppoller "github.com/MustardSeedNetworks/seed/internal/polling/snmp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/snmpwalkfile"
)

const pollingTargetWalkPath = pollingTargetPath + "/walk"

type plainDecrypter struct{}

func (plainDecrypter) DecryptValue(encrypted string) (string, error) { return encrypted, nil }

// newWalkTestServer serves walks from a recorded Cisco 3750 walk, with the
// builtin MIB names loaded and one target that holds a v2c credential.
func newWalkTestServer(t *testing.T) (*Server, *database.DB, *polling.Target) {
	t.Helper()
	s, db := newPollingTargetsTestServer(t)
	s.mibDB = app.NewMIBDatabase(s.persist)
	if err := s.mibDB.LoadBuiltinOIDs(); err != nil {
		t.Fatalf("load builtin OIDs: %v", err)
	}
	walk, err := snmpwalkfile.Open("../polling/snmp/snmpwalkfile/testdata/cisco-c3750-03.walk")
	if err != nil {
		t.Fatalf("open recorded walk: %v", err)
	}
	resolver, err := snmppoller.NewCredentialResolver(db.DeviceCredentials(), plainDecrypter{})
	if err != nil {
		t.Fatalf("credential resolver: %v", err)
	}
	s.snmpBrowser = snmppoller.NewBrowser(
		func(snmppoller.Target, snmppoller.ResolvedCredentials) (snmppoller.Client, error) {
			return walk.Client(), nil
		},
		resolver,
	)

	seedCredential(t, db, "cred-v2c", polling.CredentialKindV2c)
	target := seedTarget(t, db, "core-switch")
	target.CredentialsID = "cred-v2c"
	if err = db.PollingTargets().Update(t.Context(), testClientID, target); err != nil {
		t.Fatalf("attach credential: %v", err)
	}
	return s, db, target
}

func postWalk(t *testing.T, s *Server, id, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := withClaim(
		httptest.NewRequest(http.MethodPost, pollingTargetsPath+"/"+id+"/walk", bytes.NewBufferString(body)),
	)
	rec := httptest.NewRecorder()
	routed(pollingTargetWalkPath, s.handlePollingTargetWalk)(rec, req)
	return rec
}

func decodeWalk(t *testing.T, rec *httptest.ResponseRecorder) SNMPWalkResponse {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var got SNMPWalkResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return got
}

// Walking ifTable by name returns each varbind with its column name and the
// ifIndex as the instance; a MAC renders as hex and a description as text.
func TestHandlePollingTargetWalk_ResolvesNames(t *testing.T) {
	s, _, target := newWalkTestServer(t)

	got := decodeWalk(t, postWalk(t, s, target.ID, `{"oid":"ifTable"}`))

	if got.OID != "1.3.6.1.2.1.2.2" || got.TargetID != target.ID || got.Truncated {
		t.Fatalf("walk header = %+v, want ifTable root, this target, not truncated", got)
	}
	want := map[string]SNMPWalkRow{
		"1.3.6.1.2.1.2.2.1.2.1": {OID: "1.3.6.1.2.1.2.2.1.2.1", Name: "ifDescr", Instance: "1", Value: "Vlan1"},
		"1.3.6.1.2.1.2.2.1.6.1": {
			OID: "1.3.6.1.2.1.2.2.1.6.1", Name: "ifPhysAddress", Instance: "1", Value: "00:1c:b1:da:2c:c0",
		},
	}
	for _, row := range got.Rows {
		if w, ok := want[row.OID]; ok {
			if row != w {
				t.Errorf("row %s = %+v, want %+v", row.OID, row, w)
			}
			delete(want, row.OID)
		}
		if row.Name == "" {
			t.Errorf("row %s has no resolved name", row.OID)
		}
	}
	if len(want) != 0 {
		t.Errorf("rows missing from the walk: %v", want)
	}
}

func TestHandlePollingTargetWalk_LimitTruncates(t *testing.T) {
	s, _, target := newWalkTestServer(t)

	got := decodeWalk(t, postWalk(t, s, target.ID, `{"oid":"1.3.6.1.2.1.1","limit":3}`))

	if len(got.Rows) != 3 || !got.Truncated {
		t.Fatalf("got %d rows, truncated %v; want 3, true", len(got.Rows), got.Truncated)
	}
	if got.Rows[0].Name != "sysDescr" || got.Rows[0].Instance != "0" {
		t.Errorf("first row = %+v, want sysDescr.0", got.Rows[0])
	}
}

func TestHandlePollingTargetWalk_Refusals(t *testing.T) {
	tests := []struct {
		name    string
		target  func(t *testing.T, db *database.DB, walkable *polling.Target) string
		body    string
		want    int
		message string
	}{
		{
			name:   "unknown MIB name",
			target: func(_ *testing.T, _ *database.DB, w *polling.Target) string { return w.ID },
			body:   `{"oid":"noSuchObject"}`, want: http.StatusBadRequest, message: "known MIB object name",
		},
		{
			name:   "not an OID",
			target: func(_ *testing.T, _ *database.DB, w *polling.Target) string { return w.ID },
			body:   `{"oid":"1.3.6.x"}`, want: http.StatusBadRequest, message: "numeric OID",
		},
		{
			name:   "limit over the cap",
			target: func(_ *testing.T, _ *database.DB, w *polling.Target) string { return w.ID },
			body:   `{"oid":"system","limit":5001}`, want: http.StatusBadRequest, message: "between 1 and 5000",
		},
		{
			name:   "unknown field",
			target: func(_ *testing.T, _ *database.DB, w *polling.Target) string { return w.ID },
			body:   `{"oid":"system","community":"public"}`, want: http.StatusBadRequest, message: "unknown field",
		},
		{
			name:   "missing target",
			target: func(*testing.T, *database.DB, *polling.Target) string { return "no-such-target" },
			body:   `{"oid":"system"}`, want: http.StatusNotFound, message: "not found",
		},
		{
			name: "another client's target",
			target: func(t *testing.T, db *database.DB, _ *polling.Target) string {
				seedOtherClient(t, db)
				return seedTargetFor(t, db, otherClientID, "theirs").ID
			},
			body: `{"oid":"system"}`, want: http.StatusNotFound, message: "not found",
		},
		{
			name: "target without a credential",
			target: func(t *testing.T, db *database.DB, _ *polling.Target) string {
				return seedTarget(t, db, "no-credential").ID
			},
			body: `{"oid":"system"}`, want: http.StatusConflict, message: "references no credentials",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, db, walkable := newWalkTestServer(t)
			rec := postWalk(t, s, tt.target(t, db, walkable), tt.body)
			if rec.Code != tt.want || !strings.Contains(rec.Body.String(), tt.message) {
				t.Errorf(
					"status = %d body = %s; want %d containing %q",
					rec.Code,
					rec.Body.String(),
					tt.want,
					tt.message,
				)
			}
		})
	}
}

func TestHandlePollingTargetWalk_UnavailableWithoutKeyring(t *testing.T) {
	s, _, target := newWalkTestServer(t)
	s.snmpBrowser = nil

	rec := postWalk(t, s, target.ID, `{"oid":"system"}`)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503; body=%s", rec.Code, rec.Body.String())
	}
}

func TestWalkValue(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   any
		want string
	}{
		{name: "absent", in: nil, want: ""},
		{name: "text", in: []byte("Gi1/0/1"), want: "Gi1/0/1"},
		{name: "binary", in: []byte{0x00, 0x1c, 0xff}, want: "00:1c:ff"},
		{name: "object identifier", in: ".1.3.6.1.4.1.9.1.516", want: "1.3.6.1.4.1.9.1.516"},
		{name: "counter64", in: uint64(18446744073709551615), want: "18446744073709551615"},
		{name: "integer", in: -1, want: "-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := walkValue(tt.in); got != tt.want {
				t.Errorf("walkValue(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
