package wifiapp_test

import (
	"errors"
	"testing"

	"github.com/krisarmstrong/seed/internal/wifi"
	wifiapp "github.com/krisarmstrong/seed/internal/wifi/app"
)

// fakeHardware is a configurable wifiapp.Hardware for the management tests.
type fakeHardware struct {
	managerAvail bool
	scannerAvail bool
	wireless     bool
	setIface     string
	scanNets     []*wifi.ScannedNetwork
	scanErr      error
	connectRes   *wifi.ConnectionResult
	connectErr   error
	disconnRes   *wifi.ConnectionResult
	disconnErr   error
}

func (f *fakeHardware) ManagerAvailable() bool { return f.managerAvail }
func (f *fakeHardware) ScannerAvailable() bool { return f.scannerAvail }
func (f *fakeHardware) IsWireless() bool       { return f.wireless }
func (f *fakeHardware) SetInterface(name string) {
	f.setIface = name
}

func (f *fakeHardware) Scan() ([]*wifi.ScannedNetwork, error) { return f.scanNets, f.scanErr }
func (f *fakeHardware) Connect(_, _ string) (*wifi.ConnectionResult, error) {
	return f.connectRes, f.connectErr
}

func (f *fakeHardware) Disconnect() (*wifi.ConnectionResult, error) {
	return f.disconnRes, f.disconnErr
}

type fakeLister struct{ names []string }

func (f fakeLister) WirelessInterfaceNames() []string { return f.names }

type fakeStore struct {
	resolved string
	saved    string
	saveErr  error
	saveCnt  int
}

func (f *fakeStore) ResolvedWiFiInterface() string { return f.resolved }
func (f *fakeStore) SaveWiFiInterface(name string) error {
	f.saved = name
	f.saveCnt++
	return f.saveErr
}

func TestSettings(t *testing.T) {
	m := wifiapp.NewManagement(
		&fakeHardware{managerAvail: true, wireless: true},
		fakeLister{names: []string{"wlan0", "wlan1"}},
		&fakeStore{resolved: "wlan0"},
	)
	got := m.Settings()
	if got.Interface != "wlan0" {
		t.Errorf("interface = %q, want wlan0", got.Interface)
	}
	if len(got.AvailableWiFi) != 2 {
		t.Errorf("availableWiFi = %v, want 2 entries", got.AvailableWiFi)
	}
	if !got.IsWireless {
		t.Error("isWireless = false, want true")
	}
}

func TestSettingsNoManager(t *testing.T) {
	m := wifiapp.NewManagement(
		&fakeHardware{managerAvail: false, wireless: true},
		fakeLister{},
		&fakeStore{resolved: "eth0"},
	)
	if m.Settings().IsWireless {
		t.Error("isWireless = true with no manager, want false")
	}
}

func TestUpdateInterfaceSetsRadioAndSaves(t *testing.T) {
	hw := &fakeHardware{managerAvail: true}
	store := &fakeStore{}
	m := wifiapp.NewManagement(hw, fakeLister{}, store)

	if err := m.UpdateInterface("wlan2"); err != nil {
		t.Fatalf("UpdateInterface: %v", err)
	}
	if hw.setIface != "wlan2" {
		t.Errorf("radio interface = %q, want wlan2", hw.setIface)
	}
	if store.saved != "wlan2" || store.saveCnt != 1 {
		t.Errorf("store saved = %q cnt=%d, want wlan2 cnt=1", store.saved, store.saveCnt)
	}
}

func TestUpdateInterfaceEmptySkipsRadio(t *testing.T) {
	hw := &fakeHardware{managerAvail: true}
	store := &fakeStore{}
	m := wifiapp.NewManagement(hw, fakeLister{}, store)

	if err := m.UpdateInterface(""); err != nil {
		t.Fatalf("UpdateInterface: %v", err)
	}
	if hw.setIface != "" {
		t.Errorf("radio interface = %q, want unchanged", hw.setIface)
	}
	if store.saved != "" || store.saveCnt != 1 {
		t.Errorf("store should still persist the empty selection once")
	}
}

func TestScanResolvesAndSucceeds(t *testing.T) {
	nets := []*wifi.ScannedNetwork{{SSID: "office"}}
	m := wifiapp.NewManagement(
		&fakeHardware{managerAvail: true, scannerAvail: true, wireless: true, scanNets: nets},
		fakeLister{},
		&fakeStore{resolved: "wlan0"},
	)
	got := m.Scan("")
	if got.Interface != "wlan0" {
		t.Errorf("interface = %q, want resolved wlan0", got.Interface)
	}
	if !got.Available || got.Error != "" {
		t.Errorf("available=%v error=%q, want available no error", got.Available, got.Error)
	}
	if len(got.Networks) != 1 {
		t.Errorf("networks = %v, want 1", got.Networks)
	}
}

func TestScanDegrades(t *testing.T) {
	tests := []struct {
		name      string
		hw        *fakeHardware
		wantAvail bool
		wantErr   string
	}{
		{
			name:    "no scanner",
			hw:      &fakeHardware{scannerAvail: false},
			wantErr: "WiFi scanner not initialized",
		},
		{
			name:    "no wireless adapter",
			hw:      &fakeHardware{scannerAvail: true, managerAvail: true, wireless: false},
			wantErr: "No wireless adapter available. Connect a WiFi adapter to scan networks.",
		},
		{
			name: "scan failure",
			hw: &fakeHardware{
				scannerAvail: true,
				managerAvail: true,
				wireless:     true,
				scanErr:      errors.New("boom"),
			},
			wantAvail: true,
			wantErr:   "Wi-Fi scan failed. Check permissions and interface availability.",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := wifiapp.NewManagement(tc.hw, fakeLister{}, &fakeStore{resolved: "wlan0"})
			got := m.Scan("wlan9")
			if got.Interface != "wlan9" {
				t.Errorf("interface = %q, want requested wlan9", got.Interface)
			}
			if got.Available != tc.wantAvail {
				t.Errorf("available = %v, want %v", got.Available, tc.wantAvail)
			}
			if got.Error != tc.wantErr {
				t.Errorf("error = %q, want %q", got.Error, tc.wantErr)
			}
			if got.Networks == nil {
				t.Error("networks is nil, want non-nil empty slice")
			}
		})
	}
}

func TestStatus(t *testing.T) {
	tests := []struct {
		name        string
		hw          *fakeHardware
		adapters    []string
		wantStatus  string
		wantCanScan bool
	}{
		{name: "unavailable", hw: &fakeHardware{}, adapters: nil, wantStatus: "unavailable"},
		{
			name:       "available not selected",
			hw:         &fakeHardware{managerAvail: true, wireless: false},
			adapters:   []string{"wlan0"},
			wantStatus: "available",
		},
		{
			name:        "ready",
			hw:          &fakeHardware{managerAvail: true, wireless: true},
			adapters:    []string{"wlan0"},
			wantStatus:  "ready",
			wantCanScan: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := wifiapp.NewManagement(tc.hw, fakeLister{names: tc.adapters}, &fakeStore{resolved: "wlan0"})
			got := m.Status("")
			if got.Status != tc.wantStatus {
				t.Errorf("status = %q, want %q", got.Status, tc.wantStatus)
			}
			if got.CanScan != tc.wantCanScan {
				t.Errorf("canScan = %v, want %v", got.CanScan, tc.wantCanScan)
			}
			if got.CurrentInterface != "wlan0" {
				t.Errorf("currentInterface = %q, want resolved wlan0", got.CurrentInterface)
			}
		})
	}
}

func TestConnect(t *testing.T) {
	res := &wifi.ConnectionResult{Success: true}
	m := wifiapp.NewManagement(
		&fakeHardware{managerAvail: true, connectRes: res},
		fakeLister{}, &fakeStore{},
	)
	got, err := m.Connect("ssid", "pw")
	if err != nil || got != res {
		t.Fatalf("Connect = %v, %v; want result, nil", got, err)
	}
}

func TestConnectNoRadio(t *testing.T) {
	m := wifiapp.NewManagement(&fakeHardware{managerAvail: false}, fakeLister{}, &fakeStore{})
	if _, err := m.Connect("ssid", ""); !errors.Is(err, wifiapp.ErrRadioUnavailable) {
		t.Fatalf("err = %v, want ErrRadioUnavailable", err)
	}
}

func TestDisconnectNoRadio(t *testing.T) {
	m := wifiapp.NewManagement(&fakeHardware{managerAvail: false}, fakeLister{}, &fakeStore{})
	if _, err := m.Disconnect(); !errors.Is(err, wifiapp.ErrRadioUnavailable) {
		t.Fatalf("err = %v, want ErrRadioUnavailable", err)
	}
}
