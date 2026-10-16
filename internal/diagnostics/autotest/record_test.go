package autotest_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/autotest"
)

func TestSpecValidateLabels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		spec autotest.Spec
		ok   bool
	}{
		{"empty", autotest.Spec{}, true},
		{"typical", autotest.Spec{JackLabel: "B2-3F-R301-P07", AssetTag: "PP-0042"}, true},
		{"64 characters of UTF-8", autotest.Spec{JackLabel: strings.Repeat("é", 64)}, true},
		{"65 characters", autotest.Spec{AssetTag: strings.Repeat("x", 65)}, false},
		{"control character", autotest.Spec{JackLabel: "a\nb"}, false},
		{"invalid UTF-8", autotest.Spec{JackLabel: "\xff"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := tc.spec.Validate(); (err == nil) != tc.ok {
				t.Fatalf("Validate() = %v, want ok=%v", err, tc.ok)
			}
		})
	}
}

// A run's JSON parses back into the same record.
func TestParseRecordRoundTrip(t *testing.T) {
	t.Parallel()

	stages := []autotest.Stage{
		fixed(autotest.StageLink, autotest.OutcomePass),
		fixed(autotest.StageDNS, autotest.OutcomeWarn),
	}
	rep := autotest.Run(t.Context(), "eth0", autotest.Spec{JackLabel: "J1", AssetTag: "A1"}, stages, func(float64) {})
	data, err := json.Marshal(rep)
	if err != nil {
		t.Fatal(err)
	}
	got, err := autotest.ParseRecord(data)
	if err != nil {
		t.Fatalf("ParseRecord: %v", err)
	}
	again, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if string(again) != string(data) {
		t.Fatalf("round trip changed the record:\n got %s\nwant %s", again, data)
	}
}

func TestParseRecordRefusesTrailingData(t *testing.T) {
	t.Parallel()

	rep := autotest.Run(t.Context(), "eth0", autotest.Spec{}, nil, func(float64) {})
	data, err := json.Marshal(rep)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = autotest.ParseRecord(append(data, data...)); err == nil {
		t.Fatal("ParseRecord accepted two records")
	}
}
