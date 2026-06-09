package ipconfig_test

import (
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/network/ipconfig"
)

type fakeHW struct {
	staticErr, dhcpErr, mtuErr, refreshErr error
	staticIface, dhcpIface, mtuIface       string
	mtu                                    int
	current                                string
	refreshed                              int
}

func (h *fakeHW) ConfigureStaticIP(iface string, _ ipconfig.StaticIP) error {
	h.staticIface = iface
	return h.staticErr
}

func (h *fakeHW) ConfigureDHCP(iface string) error { h.dhcpIface = iface; return h.dhcpErr }

func (h *fakeHW) SetMTU(iface string, mtu int) error {
	h.mtuIface, h.mtu = iface, mtu
	return h.mtuErr
}
func (h *fakeHW) CurrentInterface() string { return h.current }
func (h *fakeHW) RefreshInterfaces() error { h.refreshed++; return h.refreshErr }

type fakeStore struct {
	settings    ipconfig.Settings
	persisted   string // "static" | "dhcp"
	persistErr  error
	persistedIP ipconfig.StaticIP
}

func (s *fakeStore) IPSettings() ipconfig.Settings { return s.settings }
func (s *fakeStore) PersistStatic(ip ipconfig.StaticIP) error {
	s.persisted, s.persistedIP = "static", ip
	return s.persistErr
}
func (s *fakeStore) PersistDHCP() error { s.persisted = "dhcp"; return s.persistErr }

func TestApplyStaticHappyPath(t *testing.T) {
	hw, store := &fakeHW{}, &fakeStore{}
	svc := ipconfig.NewService(hw, store)

	err := svc.Apply("eth0", ipconfig.ModeStatic, ipconfig.StaticIP{Address: "10.0.0.5"})
	if err != nil {
		t.Fatalf("apply static: %v", err)
	}
	if hw.staticIface != "eth0" || store.persisted != "static" || hw.refreshed != 1 {
		t.Fatalf(
			"unexpected sequence: iface=%q persisted=%q refreshed=%d",
			hw.staticIface,
			store.persisted,
			hw.refreshed,
		)
	}
	if store.persistedIP.Address != "10.0.0.5" {
		t.Fatalf("static ip not persisted: %+v", store.persistedIP)
	}
}

func TestApplyDHCPHappyPath(t *testing.T) {
	hw, store := &fakeHW{}, &fakeStore{}
	if err := ipconfig.NewService(hw, store).Apply("eth1", ipconfig.ModeDHCP, ipconfig.StaticIP{}); err != nil {
		t.Fatalf("apply dhcp: %v", err)
	}
	if hw.dhcpIface != "eth1" || store.persisted != "dhcp" || hw.refreshed != 1 {
		t.Fatalf("unexpected dhcp sequence")
	}
}

func TestApplyErrorMapping(t *testing.T) {
	cases := []struct {
		name string
		hw   *fakeHW
		st   *fakeStore
		mode string
		want error
	}{
		{"invalid mode", &fakeHW{}, &fakeStore{}, "bogus", ipconfig.ErrInvalidMode},
		{
			"static hw fails",
			&fakeHW{staticErr: errors.New("x")},
			&fakeStore{},
			ipconfig.ModeStatic,
			ipconfig.ErrStaticConfig,
		},
		{
			"dhcp hw fails",
			&fakeHW{dhcpErr: errors.New("x")},
			&fakeStore{},
			ipconfig.ModeDHCP,
			ipconfig.ErrDHCPConfig,
		},
		{
			"persist fails",
			&fakeHW{},
			&fakeStore{persistErr: errors.New("x")},
			ipconfig.ModeStatic,
			ipconfig.ErrSave,
		},
		{
			"refresh fails",
			&fakeHW{refreshErr: errors.New("x")},
			&fakeStore{},
			ipconfig.ModeStatic,
			ipconfig.ErrRefresh,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ipconfig.NewService(tc.hw, tc.st).Apply("eth0", tc.mode, ipconfig.StaticIP{})
			if !errors.Is(err, tc.want) {
				t.Fatalf("want %v, got %v", tc.want, err)
			}
		})
	}
}

func TestApplyDoesNotPersistWhenHardwareFails(t *testing.T) {
	hw, store := &fakeHW{staticErr: errors.New("nope")}, &fakeStore{}
	_ = ipconfig.NewService(hw, store).Apply("eth0", ipconfig.ModeStatic, ipconfig.StaticIP{})
	if store.persisted != "" {
		t.Fatalf("config must not be persisted when hardware config fails, got %q", store.persisted)
	}
}

func TestSetMTUResolvesCurrentInterface(t *testing.T) {
	hw := &fakeHW{current: "wlan0"}
	got, err := ipconfig.NewService(hw, &fakeStore{}).SetMTU("", 9000)
	if err != nil {
		t.Fatalf("set mtu: %v", err)
	}
	if got != "wlan0" || hw.mtuIface != "wlan0" || hw.mtu != 9000 {
		t.Fatalf("mtu resolution mismatch: got=%q iface=%q mtu=%d", got, hw.mtuIface, hw.mtu)
	}
}

func TestSetMTUError(t *testing.T) {
	hw := &fakeHW{mtuErr: errors.New("eio")}
	if _, err := ipconfig.NewService(hw, &fakeStore{}).SetMTU("eth0", 1500); !errors.Is(err, ipconfig.ErrSetMTU) {
		t.Fatalf("want ErrSetMTU, got %v", err)
	}
}
