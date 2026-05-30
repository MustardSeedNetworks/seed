// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"testing"

	"github.com/krisarmstrong/seed/internal/canopy/survey"
)

// newRoamFixture returns a survey with one Active + one Passive sample
// at the top level and one per-floor Active sample, with the roam
// fields populated so the filter has something to strip.
func newRoamFixture() *survey.Survey {
	return &survey.Survey{
		ID: "s1",
		Samples: []*survey.SamplePoint{
			{
				X: 1, Y: 1,
				SampleData: &survey.ActiveSample{
					SSID:          "office",
					BSSID:         "aa:bb:cc:dd:ee:ff",
					RSSI:          -55,
					DataRate:      300,
					RoamingEvent:  true,
					PreviousBSSID: "11:22:33:44:55:66",
					RoamCount:     5,
				},
			},
			{
				X: 2, Y: 2,
				SampleData: &survey.PassiveSample{UniqueSSIDs: 7},
			},
		},
		Floors: []*survey.Floor{
			{
				ID: "f1",
				Samples: []*survey.SamplePoint{
					{
						X: 3, Y: 3,
						SampleData: &survey.ActiveSample{
							SSID:          "lab",
							RSSI:          -60,
							DataRate:      150,
							RoamingEvent:  true,
							PreviousBSSID: "deadbeefcafe",
							RoamCount:     2,
						},
					},
				},
			},
		},
	}
}

// TestFilterSurveyRoamFields_StripsTopLevelSamples — roam fields on
// the top-level Samples slice are zeroed while non-roam fields and
// non-Active samples (Passive) are preserved.
func TestFilterSurveyRoamFields_StripsTopLevelSamples(t *testing.T) {
	t.Parallel()
	in := newRoamFixture()
	got := filterSurveyRoamFields(in)

	if got == in {
		t.Fatal("filter must deep-copy; got the input pointer back")
	}
	active0, ok := got.Samples[0].SampleData.(*survey.ActiveSample)
	if !ok || active0 == nil {
		t.Fatalf("samples[0] sample data = %T, want *ActiveSample", got.Samples[0].SampleData)
	}
	if active0.RoamingEvent || active0.PreviousBSSID != "" || active0.RoamCount != 0 {
		t.Errorf("samples[0] roam fields not stripped: %+v", active0)
	}
	if active0.SSID != "office" || active0.BSSID != "aa:bb:cc:dd:ee:ff" ||
		active0.RSSI != -55 || active0.DataRate != 300 {
		t.Errorf("samples[0] non-roam fields disturbed: %+v", active0)
	}
	passive, isPassive := got.Samples[1].SampleData.(*survey.PassiveSample)
	if !isPassive || passive.UniqueSSIDs != 7 {
		t.Errorf("samples[1] passive sample disturbed: %+v", got.Samples[1].SampleData)
	}
}

// TestFilterSurveyRoamFields_StripsPerFloorSamples — same treatment
// applies to samples nested under Floors (the modern multi-floor path).
func TestFilterSurveyRoamFields_StripsPerFloorSamples(t *testing.T) {
	t.Parallel()
	got := filterSurveyRoamFields(newRoamFixture())

	sample, ok := got.Floors[0].Samples[0].SampleData.(*survey.ActiveSample)
	if !ok || sample == nil {
		t.Fatalf("floors[0].samples[0] sample data wrong type")
	}
	if sample.RoamingEvent || sample.PreviousBSSID != "" || sample.RoamCount != 0 {
		t.Errorf("floors[0].samples[0] roam fields not stripped: %+v", sample)
	}
}

// TestFilterSurveyRoamFields_DoesNotMutateInput — the filter must
// deep-copy so the surveyManager cache stays untouched.
func TestFilterSurveyRoamFields_DoesNotMutateInput(t *testing.T) {
	t.Parallel()
	in := newRoamFixture()
	_ = filterSurveyRoamFields(in)

	origActive, ok := in.Samples[0].SampleData.(*survey.ActiveSample)
	if !ok {
		t.Fatal("fixture sample type changed")
	}
	if !origActive.RoamingEvent || origActive.PreviousBSSID != "11:22:33:44:55:66" ||
		origActive.RoamCount != 5 {
		t.Errorf("filter mutated the input survey: %+v", origActive)
	}
}

// TestApplyRoamFilterIfGated_AppliesGateBasedOnLicense proves the
// dispatcher: with a Pro-trial license the survey passes through
// unmodified; without one, the filter is applied.
func TestApplyRoamFilterIfGated_AppliesGateBasedOnLicense(t *testing.T) {
	t.Parallel()
	s, mgr := apiTokenTestSetup(t)

	in := &survey.Survey{
		ID: "s1",
		Samples: []*survey.SamplePoint{
			{
				X: 1, Y: 1,
				SampleData: &survey.ActiveSample{
					SSID:          "x",
					RoamingEvent:  true,
					PreviousBSSID: "old",
					RoamCount:     3,
				},
			},
		},
	}

	// 1. No license → roam fields stripped.
	got := s.applyRoamFilterIfGated(in)
	if got == in {
		t.Fatal("free tier: dispatcher should return a filtered copy, got input pointer back")
	}
	freeActive := got.Samples[0].SampleData.(*survey.ActiveSample)
	if freeActive.RoamCount != 0 || freeActive.PreviousBSSID != "" {
		t.Errorf("free tier: roam fields not stripped: %+v", freeActive)
	}

	// 2. Start trial → roam fields preserved (dispatcher returns input
	//    as-is).
	if res := mgr.StartTrial(); !res.Success {
		t.Fatalf("StartTrial: %s", res.Message)
	}
	got2 := s.applyRoamFilterIfGated(in)
	if got2 != in {
		t.Error("pro trial: dispatcher should return the input pointer unchanged")
	}
}
