package management_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/settings/management"
)

func escalationUpdate(ladders any) map[string]any {
	return map[string]any{"alerts": map[string]any{"escalations": ladders}}
}

// The update arrives as decoded JSON, so the numbers are float64 here exactly
// as they are from the handler.
func ifaceDownLadder() map[string]any {
	return map[string]any{
		"rule": "iface.down",
		"stages": []any{
			map[string]any{"afterSeconds": float64(300), "channels": []any{"webhook"}},
			map[string]any{"afterSeconds": float64(900), "channels": []any{"webhook", "email"}},
		},
		"repeatSeconds": float64(1800),
	}
}

func TestUpdateStoresEscalationLaddersAndServesThemBack(t *testing.T) {
	cfg := config.DefaultConfig()
	reconfig := &countingReconfigurer{}
	svc := management.NewService(&fakeStore{cfg: cfg}, fakeEncrypter{}, reconfig)

	if err := svc.Update(escalationUpdate([]any{ifaceDownLadder()}), ""); err != nil {
		t.Fatalf("Update: %v", err)
	}
	want := []config.AlertEscalationConfig{{
		Rule: "iface.down",
		Stages: []config.AlertEscalationStage{
			{AfterSeconds: 300, Channels: []string{"webhook"}},
			{AfterSeconds: 900, Channels: []string{"webhook", "email"}},
		},
		RepeatSeconds: 1800,
	}}
	if !reflect.DeepEqual(cfg.Alerts.Escalations, want) {
		t.Fatalf("stored %+v, want %+v", cfg.Alerts.Escalations, want)
	}
	if reconfig.calls != 1 {
		t.Errorf("reconfigure called %d times, want 1: a ladder applies without a restart", reconfig.calls)
	}

	ladders, err := management.EscalationLadders(cfg.Alerts.Escalations)
	if err != nil || len(ladders) != 1 || len(ladders[0].Stages) != 2 {
		t.Fatalf("EscalationLadders = %+v, %v", ladders, err)
	}

	settings, _ := svc.Get()
	served := settings["alerts"].(map[string]any)["escalations"]
	if got := reflect.ValueOf(served).Len(); got != 1 {
		t.Fatalf("served %d ladders, want 1", got)
	}

	// An empty list turns escalation off.
	if err = svc.Update(escalationUpdate([]any{}), ""); err != nil {
		t.Fatalf("Update to clear: %v", err)
	}
	if len(cfg.Alerts.Escalations) != 0 {
		t.Fatalf("stored %+v after clearing", cfg.Alerts.Escalations)
	}
}

func TestUpdateRefusesALadderThatCouldNotRun(t *testing.T) {
	tooSoon := ifaceDownLadder()
	tooSoon["stages"] = []any{map[string]any{"afterSeconds": float64(10), "channels": []any{"webhook"}}}
	badChannel := ifaceDownLadder()
	badChannel["stages"] = []any{map[string]any{"afterSeconds": float64(300), "channels": []any{"sms"}}}
	unknownField := ifaceDownLadder()
	unknownField["escalateTo"] = "manager"

	tests := map[string]any{
		"stage sooner than a minute": []any{tooSoon},
		"unknown channel":            []any{badChannel},
		"two ladders for one rule":   []any{ifaceDownLadder(), ifaceDownLadder()},
		"unknown field":              []any{unknownField},
		"not a list":                 "iface.down",
	}
	for name, ladders := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			svc := management.NewService(&fakeStore{cfg: cfg}, fakeEncrypter{}, nil)
			err := svc.Update(escalationUpdate(ladders), "")
			if !errors.As(err, new(management.ValidationError)) {
				t.Fatalf("Update = %v, want a ValidationError", err)
			}
			if len(cfg.Alerts.Escalations) != 0 {
				t.Fatalf("stored %+v from a refused update", cfg.Alerts.Escalations)
			}
		})
	}
}
