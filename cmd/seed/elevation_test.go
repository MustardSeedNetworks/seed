package main

import (
	"errors"
	"strings"
	"testing"
)

func TestCheckElevation(t *testing.T) {
	const system = "/system/config/seed.json"
	tests := []struct {
		name       string
		configPath string
		elevated   bool
		wantErr    bool
	}{
		{"system config, elevated", system, true, false},
		{"system config, not elevated", system, false, true},
		{"unclean spelling of the system config", "/system/config/../config/seed.json", false, true},
		{"explicit config elsewhere needs no elevation", "/home/alice/seed.json", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkElevation(tt.configPath, system, tt.elevated)
			if (err != nil) != tt.wantErr {
				t.Fatalf("checkElevation error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && exitCodeFor(err) != exitWriteRefused {
				t.Errorf("exit code = %d, want %d", exitCodeFor(err), exitWriteRefused)
			}
		})
	}
}

func TestElevationErrorNamesThePathAndTheFix(t *testing.T) {
	err := checkElevation("/system/config/seed.json", "/system/config/seed.json", false)
	if _, ok := errors.AsType[*elevationRequiredError](err); !ok {
		t.Fatalf("got %T, want *elevationRequiredError", err)
	}
	for _, want := range []string{"/system/config/seed.json", "elevated"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q lacks %q", err, want)
		}
	}
}
