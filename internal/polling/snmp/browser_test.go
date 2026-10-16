package snmp_test

import (
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/polling"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/snmpwalkfile"
)

func recordedBrowser(t *testing.T) *snmp.Browser {
	t.Helper()
	walk, err := snmpwalkfile.Open("snmpwalkfile/testdata/cisco-c3750-03.walk")
	if err != nil {
		t.Fatalf("open recorded walk: %v", err)
	}
	store := &fakeCredStore{creds: &polling.Credentials{
		ID:              "cred-1",
		Kind:            polling.CredentialKindV2c,
		SNMPCommunityCT: "enc:v1:community",
	}}
	factory := func(snmp.Target, snmp.ResolvedCredentials) (snmp.Client, error) {
		return walk.Client(), nil
	}
	return snmp.NewBrowser(factory, newResolver(t, store, fakeDecrypter{}))
}

func TestBrowserWalk(t *testing.T) {
	t.Parallel()
	target := &polling.Target{ID: "t-1", IPAddress: "192.0.2.1", CredentialsID: "cred-1"}
	tests := []struct {
		name          string
		limit         int
		wantRows      int
		wantTruncated bool
	}{
		{name: "whole system group", limit: 100, wantRows: 8},
		{name: "exactly the limit", limit: 8, wantRows: 8},
		{name: "cut at the limit", limit: 3, wantRows: 3, wantTruncated: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			vbs, truncated, err := recordedBrowser(t).Walk(t.Context(), target, "1.3.6.1.2.1.1", tt.limit)
			if err != nil {
				t.Fatalf("Walk: %v", err)
			}
			if len(vbs) != tt.wantRows || truncated != tt.wantTruncated {
				t.Fatalf(
					"Walk = %d rows, truncated %v; want %d, %v",
					len(vbs),
					truncated,
					tt.wantRows,
					tt.wantTruncated,
				)
			}
			if vbs[0].OID != "1.3.6.1.2.1.1.1.0" {
				t.Errorf("first OID = %s, want sysDescr.0", vbs[0].OID)
			}
		})
	}
}

func TestBrowserWalkRefusesUnresolvedCredentials(t *testing.T) {
	t.Parallel()
	target := &polling.Target{ID: "t-1", IPAddress: "192.0.2.1"}
	_, _, err := recordedBrowser(t).Walk(t.Context(), target, "1.3.6.1.2.1.1", 10)
	if !errors.Is(err, snmp.ErrCredentialsUnresolved) {
		t.Errorf("Walk error = %v, want ErrCredentialsUnresolved", err)
	}
}
