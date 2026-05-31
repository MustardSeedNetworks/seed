package pipeline_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/alerts/pipeline"
	"github.com/krisarmstrong/seed/internal/database"
)

func silentLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }
func at() time.Time              { return time.Date(2026, 5, 31, 12, 0, 0, 0, time.UTC) }

type fakeEvents struct {
	mu      sync.Mutex
	rows    []*database.ListenerEvent
	listErr error
}

func (f *fakeEvents) List(_ context.Context, _ database.ListenerEventListOptions) ([]*database.ListenerEvent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]*database.ListenerEvent, len(f.rows))
	copy(out, f.rows)
	return out, nil
}

type fakeAlerts struct {
	mu        sync.Mutex
	created   []*database.Alert
	createErr error
}

func (f *fakeAlerts) Create(_ context.Context, alert *database.Alert) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return f.createErr
	}
	f.created = append(f.created, alert)
	return nil
}

type fakeSettings struct {
	mu     sync.Mutex
	values map[string]string
}

func newFakeSettings() *fakeSettings {
	return &fakeSettings{values: make(map[string]string)}
}

func (f *fakeSettings) GetWithDefault(_ context.Context, key, def string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.values[key]
	if !ok {
		return def, nil
	}
	return v, nil
}

func (f *fakeSettings) Set(_ context.Context, key, value string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.values[key] = value
	return nil
}

func syslogEvent(severity, source string, observed time.Time, message string) *database.ListenerEvent {
	payload, _ := json.Marshal(map[string]string{
		"facility":     "3",
		"severityName": severity,
		"message":      message,
	})
	return &database.ListenerEvent{
		Kind:        "syslog-udp",
		ClientID:    "default",
		SourceAddr:  source,
		Severity:    severity,
		ObservedAt:  observed,
		PayloadJSON: string(payload),
	}
}

func trapEvent(trapOID, source string, observed time.Time) *database.ListenerEvent {
	payload, _ := json.Marshal(map[string]string{
		"version": "v2c",
		"trapOid": trapOID,
	})
	return &database.ListenerEvent{
		Kind:        "snmp-trap-v2c",
		ClientID:    "default",
		SourceAddr:  source,
		ObservedAt:  observed,
		PayloadJSON: string(payload),
	}
}

func TestNewListenerPipeline_RejectsMissingDeps(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cfg  pipeline.ListenerConfig
	}{
		{"missing Events", pipeline.ListenerConfig{Alerts: &fakeAlerts{}, Settings: newFakeSettings()}},
		{"missing Alerts", pipeline.ListenerConfig{Events: &fakeEvents{}, Settings: newFakeSettings()}},
		{"missing Settings", pipeline.ListenerConfig{Events: &fakeEvents{}, Alerts: &fakeAlerts{}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := pipeline.NewListenerPipeline(tt.cfg); err == nil {
				t.Errorf("expected error for %s", tt.name)
			}
		})
	}
}

func TestScanOnce_SyslogSevereErrorEmitsAlert(t *testing.T) {
	t.Parallel()
	events := &fakeEvents{rows: []*database.ListenerEvent{
		syslogEvent("error", "10.0.0.1:514", at(), "SSH login failed for user admin"),
	}}
	alerts := &fakeAlerts{}
	p, _ := pipeline.NewListenerPipeline(pipeline.ListenerConfig{
		Events: events, Alerts: alerts, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	if err := p.ScanOnce(context.Background()); err != nil {
		t.Fatalf("ScanOnce: %v", err)
	}
	if len(alerts.created) != 1 {
		t.Fatalf("alerts = %d, want 1", len(alerts.created))
	}
	if alerts.created[0].Severity != database.AlertSeverityError {
		t.Errorf("severity = %q, want error", alerts.created[0].Severity)
	}
}

func TestScanOnce_SyslogInfoDoesNotAlert(t *testing.T) {
	t.Parallel()
	events := &fakeEvents{rows: []*database.ListenerEvent{
		syslogEvent("informational", "10.0.0.1:514", at(), "nightly cron run"),
	}}
	alerts := &fakeAlerts{}
	p, _ := pipeline.NewListenerPipeline(pipeline.ListenerConfig{
		Events: events, Alerts: alerts, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	_ = p.ScanOnce(context.Background())
	if len(alerts.created) != 0 {
		t.Errorf("informational syslog should not alert, got %d alerts", len(alerts.created))
	}
}

func TestScanOnce_TrapLinkDownEmitsConnectivityAlert(t *testing.T) {
	t.Parallel()
	events := &fakeEvents{rows: []*database.ListenerEvent{
		trapEvent("1.3.6.1.6.3.1.1.5.3", "10.0.0.5:0", at()),
	}}
	alerts := &fakeAlerts{}
	p, _ := pipeline.NewListenerPipeline(pipeline.ListenerConfig{
		Events: events, Alerts: alerts, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	_ = p.ScanOnce(context.Background())
	if len(alerts.created) != 1 {
		t.Fatalf("alerts = %d, want 1", len(alerts.created))
	}
	a := alerts.created[0]
	if a.Type != database.AlertTypeConnectivity || a.Severity != database.AlertSeverityWarning {
		t.Errorf("type/severity = %q/%q", a.Type, a.Severity)
	}
}

func TestScanOnce_TrapAuthFailureEmitsSecurityAlert(t *testing.T) {
	t.Parallel()
	events := &fakeEvents{rows: []*database.ListenerEvent{
		trapEvent("1.3.6.1.6.3.1.1.5.5", "10.0.0.5:0", at()),
	}}
	alerts := &fakeAlerts{}
	p, _ := pipeline.NewListenerPipeline(pipeline.ListenerConfig{
		Events: events, Alerts: alerts, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	_ = p.ScanOnce(context.Background())
	if len(alerts.created) != 1 {
		t.Fatalf("alerts = %d, want 1", len(alerts.created))
	}
	a := alerts.created[0]
	if a.Type != database.AlertTypeSecurity || a.Severity != database.AlertSeverityError {
		t.Errorf("type/severity = %q/%q", a.Type, a.Severity)
	}
}

func TestScanOnce_SuppressionBlocksRepeats(t *testing.T) {
	t.Parallel()
	// Two identical linkDown traps in the same scan -> one alert.
	events := &fakeEvents{rows: []*database.ListenerEvent{
		trapEvent("1.3.6.1.6.3.1.1.5.3", "10.0.0.5:0", at()),
		trapEvent("1.3.6.1.6.3.1.1.5.3", "10.0.0.5:0", at().Add(time.Second)),
	}}
	alerts := &fakeAlerts{}
	p, _ := pipeline.NewListenerPipeline(pipeline.ListenerConfig{
		Events: events, Alerts: alerts, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at, Suppression: time.Minute,
	})
	_ = p.ScanOnce(context.Background())
	if len(alerts.created) != 1 {
		t.Errorf("suppression should collapse to 1 alert, got %d", len(alerts.created))
	}
}

func TestScanOnce_DifferentSourcesNotSuppressed(t *testing.T) {
	t.Parallel()
	events := &fakeEvents{rows: []*database.ListenerEvent{
		trapEvent("1.3.6.1.6.3.1.1.5.3", "10.0.0.5:0", at()),
		trapEvent("1.3.6.1.6.3.1.1.5.3", "10.0.0.6:0", at()),
	}}
	alerts := &fakeAlerts{}
	p, _ := pipeline.NewListenerPipeline(pipeline.ListenerConfig{
		Events: events, Alerts: alerts, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at, Suppression: time.Minute,
	})
	_ = p.ScanOnce(context.Background())
	if len(alerts.created) != 2 {
		t.Errorf("alerts from different sources must not be suppressed, got %d", len(alerts.created))
	}
}

func TestScanOnce_PersistsHighWater(t *testing.T) {
	t.Parallel()
	older := at().Add(-time.Hour)
	newer := at()
	events := &fakeEvents{rows: []*database.ListenerEvent{
		syslogEvent("error", "10.0.0.1:514", older, "old"),
		syslogEvent("error", "10.0.0.2:514", newer, "new"),
	}}
	settings := newFakeSettings()
	p, _ := pipeline.NewListenerPipeline(pipeline.ListenerConfig{
		Events: events, Alerts: &fakeAlerts{}, Settings: settings,
		Logger: silentLogger(), Now: at,
	})
	_ = p.ScanOnce(context.Background())

	hw, _ := settings.GetWithDefault(context.Background(), "alerts.listener.high_water", "")
	parsed, perr := time.Parse(time.RFC3339Nano, hw)
	if perr != nil {
		t.Fatalf("parse high-water: %v", perr)
	}
	if !parsed.Equal(newer) {
		t.Errorf("high-water = %v, want %v", parsed, newer)
	}
}

func TestScanOnce_ListErrorPropagates(t *testing.T) {
	t.Parallel()
	p, _ := pipeline.NewListenerPipeline(pipeline.ListenerConfig{
		Events:   &fakeEvents{listErr: errors.New("db down")},
		Alerts:   &fakeAlerts{},
		Settings: newFakeSettings(),
		Logger:   silentLogger(),
		Now:      at,
	})
	if err := p.ScanOnce(context.Background()); err == nil {
		t.Error("expected list error to propagate")
	}
}

func TestScanOnce_CreateErrorContinuesBatch(t *testing.T) {
	t.Parallel()
	events := &fakeEvents{rows: []*database.ListenerEvent{
		syslogEvent("error", "10.0.0.1:514", at(), "boom"),
	}}
	alerts := &fakeAlerts{createErr: errors.New("constraint")}
	p, _ := pipeline.NewListenerPipeline(pipeline.ListenerConfig{
		Events: events, Alerts: alerts, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
	})
	if err := p.ScanOnce(context.Background()); err != nil {
		t.Fatalf("create error should not abort scan: %v", err)
	}
}

func TestScanOnce_CustomRulesOverrideDefaults(t *testing.T) {
	t.Parallel()
	hits := 0
	custom := []pipeline.Rule{
		{
			ID:    "test.everything",
			Match: func(_ *database.ListenerEvent) bool { return true },
			Build: func(evt *database.ListenerEvent) *database.Alert {
				hits++
				return &database.Alert{
					Type: "test", Severity: "info",
					Title: "all", Message: "all", Source: evt.SourceAddr,
				}
			},
		},
	}
	events := &fakeEvents{rows: []*database.ListenerEvent{
		// An informational syslog would NOT trip the defaults but
		// will trip our match-all custom rule.
		syslogEvent("informational", "10.0.0.1:514", at(), "noise"),
	}}
	p, _ := pipeline.NewListenerPipeline(pipeline.ListenerConfig{
		Events: events, Alerts: &fakeAlerts{}, Settings: newFakeSettings(),
		Logger: silentLogger(), Now: at,
		Rules: custom,
	})
	_ = p.ScanOnce(context.Background())
	if hits != 1 {
		t.Errorf("custom rule should fire once, got %d", hits)
	}
}

func TestListenerPipeline_EngineName(t *testing.T) {
	t.Parallel()
	p, _ := pipeline.NewListenerPipeline(pipeline.ListenerConfig{
		Events:   &fakeEvents{},
		Alerts:   &fakeAlerts{},
		Settings: newFakeSettings(),
		Logger:   silentLogger(),
		Now:      at,
	})
	if p.Name() != pipeline.ListenerPipelineName {
		t.Errorf("Name() = %q, want %q", p.Name(), pipeline.ListenerPipelineName)
	}
}

func TestListenerStartStop_Idempotent(t *testing.T) {
	t.Parallel()
	p, _ := pipeline.NewListenerPipeline(pipeline.ListenerConfig{
		Events:   &fakeEvents{},
		Alerts:   &fakeAlerts{},
		Settings: newFakeSettings(),
		Logger:   silentLogger(),
		Now:      at,
		Interval: 500 * time.Millisecond,
	})
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := p.Start(context.Background()); err != nil {
		t.Fatalf("second Start: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := p.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}
