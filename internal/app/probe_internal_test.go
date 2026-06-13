package app

import (
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
)

// TestDBProbeToModel asserts the composition-root adapter translates a probe
// database row into the probe package's domain type, passing the JSON columns
// through as raw message bytes (the translation that moved out of internal/probe
// when its persistence port was narrowed to domain types, WS-B).
func TestDBProbeToModel(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC()
	row := &database.Probe{
		ID:              "p-1",
		ClientID:        "tenant-x",
		Kind:            "dns",
		DisplayName:     "internal",
		Target:          "internal.example.com",
		ParamsJSON:      `{"server":"10.0.0.5"}`,
		IntervalSeconds: 60,
		Enabled:         true,
		WarningJSON:     `{"latency_ms":50}`,
		CriticalJSON:    `{"latency_ms":200}`,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	got := dbProbeToModel(row)

	if got.ID != "p-1" || got.ClientID != "tenant-x" || got.Kind != "dns" {
		t.Errorf("identity fields mistranslated: %+v", got)
	}
	if got.DisplayName != "internal" || got.Target != "internal.example.com" {
		t.Errorf("descriptor fields mistranslated: %+v", got)
	}
	if got.IntervalSeconds != 60 || !got.Enabled {
		t.Errorf("schedule fields mistranslated: %+v", got)
	}
	if string(got.Params) != `{"server":"10.0.0.5"}` {
		t.Errorf("Params = %q, want round-trip", string(got.Params))
	}
	if string(got.Warning) != `{"latency_ms":50}` {
		t.Errorf("Warning = %q, want round-trip", string(got.Warning))
	}
	if string(got.Critical) != `{"latency_ms":200}` {
		t.Errorf("Critical = %q, want round-trip", string(got.Critical))
	}
	if !got.CreatedAt.Equal(now) || !got.UpdatedAt.Equal(now) {
		t.Errorf("timestamps mistranslated: created=%v updated=%v", got.CreatedAt, got.UpdatedAt)
	}
}
