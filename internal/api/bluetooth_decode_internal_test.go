package api

import (
	"reflect"
	"testing"
)

func TestDecodeBTCompany(t *testing.T) {
	t.Parallel()
	cases := map[uint16]string{
		0x004C: "Apple",
		0x0075: "Samsung Electronics",
		0x00E0: "Google",
		0:      "",                 // zero ID → empty
		0x9999: "Unknown (0x9999)", // unknown → raw hex
	}
	for id, want := range cases {
		if got := decodeBTCompany(id); got != want {
			t.Errorf("decodeBTCompany(0x%04X) = %q, want %q", id, got, want)
		}
	}
}

func TestDecodeBTServices(t *testing.T) {
	t.Parallel()
	in := []string{
		"180f",                                 // short 16-bit → Battery
		"0000180d-0000-1000-8000-00805f9b34fb", // full 128-bit base → Heart Rate
		"abcd1234-0000-1000-8000-000000000000", // non-standard → passthrough
	}
	got := decodeBTServices(in)
	want := []string{"Battery", "Heart Rate", "abcd1234-0000-1000-8000-000000000000"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("decodeBTServices = %v, want %v", got, want)
	}
	if decodeBTServices(nil) != nil {
		t.Error("decodeBTServices(nil) should be nil")
	}
}

func TestDecodeBTAppearance(t *testing.T) {
	t.Parallel()
	cases := map[uint16]string{
		64:    "Phone",             // exact
		961:   "Keyboard",          // exact
		833:   "Heart Rate Sensor", // category fallback (832>>6 == 833>>6 == 13)
		0:     "",                  // zero → empty
		60000: "Unknown (60000)",   // unknown value + category → raw
	}
	for a, want := range cases {
		if got := decodeBTAppearance(a); got != want {
			t.Errorf("decodeBTAppearance(%d) = %q, want %q", a, got, want)
		}
	}
}
