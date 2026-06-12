package export_test

import (
	"context"
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/export"
)

// fakeSources returns a fixed device summary and a fixed set of cards.
type fakeSources struct {
	refreshErr error
	cards      map[string]any
}

func (f fakeSources) RefreshInterfaces() error { return f.refreshErr }
func (f fakeSources) DeviceMAC(string) string  { return "aa:bb:cc:dd:ee:ff" }
func (f fakeSources) IPMode() string           { return "dhcp" }
func (f fakeSources) Cards(context.Context, string) map[string]any {
	return f.cards
}

func TestBuildComposesDocument(t *testing.T) {
	src := fakeSources{cards: map[string]any{
		"link": map[string]any{"linkUp": true},
		"dns":  map[string]any{"server": "8.8.8.8"},
	}}
	svc := export.NewService(src)

	data, err := svc.Build(context.Background(), "eth0", "1.2.3")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if data.Version != "1.2.3" || data.Device.Interface != "eth0" ||
		data.Device.MAC != "aa:bb:cc:dd:ee:ff" || data.Device.IPMode != "dhcp" {
		t.Errorf("device/meta wrong: %+v", data)
	}
	if data.Timestamp == "" {
		t.Error("timestamp should be stamped")
	}
	if len(data.Cards) != 2 {
		t.Errorf("want 2 cards, got %d: %+v", len(data.Cards), data.Cards)
	}
}

func TestBuildAbortsOnRefreshError(t *testing.T) {
	wantErr := errors.New("refresh failed")
	svc := export.NewService(fakeSources{refreshErr: wantErr})
	if _, err := svc.Build(context.Background(), "eth0", "1"); !errors.Is(err, wantErr) {
		t.Errorf("Build should abort on refresh error, got %v", err)
	}
}
