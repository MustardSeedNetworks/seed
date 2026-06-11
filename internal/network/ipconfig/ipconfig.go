// Package ipconfig holds the network IP-configuration application (use-case)
// layer (ADR-0020). It owns the IP-configuration and MTU orchestration that
// previously lived in the api.Server network handlers — the multi-step
// "configure the interface, persist the config, refresh" sequence — behind
// narrow consumer-defined ports over the netif manager and the config store.
// Handlers keep transport concerns: request decode, input validation,
// interface-from-query resolution, and localized error mapping. The adapters
// satisfying the ports live in the composition root (internal/app).
package ipconfig

import "errors"

// IP modes.
const (
	ModeDHCP   = "dhcp"
	ModeStatic = "static"
)

// Sentinel errors, mapped by handlers to the pre-strangle HTTP responses.
var (
	ErrInvalidMode  = errors.New("invalid ip mode")
	ErrStaticConfig = errors.New("static ip configuration failed")
	ErrDHCPConfig   = errors.New("dhcp configuration failed")
	ErrSave         = errors.New("failed to save configuration")
	ErrRefresh      = errors.New("failed to refresh interfaces")
	ErrSetMTU       = errors.New("failed to set mtu")
)

// StaticIP is the use-case static-IP model.
type StaticIP struct {
	Address string
	Netmask string
	Gateway string
	DNS     []string
}

// Settings is the read model for the IP-settings GET.
type Settings struct {
	Mode    string
	Address string
	Netmask string
	Gateway string
	DNS     []string
}

// Hardware is the netif surface the use-case drives, defined at the consumer
// (ADR-0020) and satisfied by an adapter over *netif.Manager in internal/app.
type Hardware interface {
	ConfigureStaticIP(iface string, ip StaticIP) error
	ConfigureDHCP(iface string) error
	SetMTU(iface string, mtu int) error
	CurrentInterface() string
	RefreshInterfaces() error
}

// ConfigStore reads and persists the IP block. The adapter owns the config
// locking and on-disk save.
type ConfigStore interface {
	IPSettings() Settings
	PersistStatic(ip StaticIP) error
	PersistDHCP() error
}

// Service is the IP-configuration + MTU use-case.
type Service struct {
	hw    Hardware
	store ConfigStore
}

// NewService builds the use-case over its narrow dependencies.
func NewService(hw Hardware, store ConfigStore) *Service {
	return &Service{hw: hw, store: store}
}

// Settings returns the current IP configuration.
func (s *Service) Settings() Settings {
	return s.store.IPSettings()
}

// Apply configures the interface for the requested mode, persists the resulting
// config, then refreshes the interface table. Hardware configuration runs first
// so the persisted config only changes after the interface accepts it; each step
// maps to its own sentinel for faithful handler error mapping.
func (s *Service) Apply(iface, mode string, ip StaticIP) error {
	switch mode {
	case ModeStatic:
		if err := s.hw.ConfigureStaticIP(iface, ip); err != nil {
			return ErrStaticConfig
		}
		if err := s.store.PersistStatic(ip); err != nil {
			return ErrSave
		}
	case ModeDHCP:
		if err := s.hw.ConfigureDHCP(iface); err != nil {
			return ErrDHCPConfig
		}
		if err := s.store.PersistDHCP(); err != nil {
			return ErrSave
		}
	default:
		return ErrInvalidMode
	}

	if err := s.hw.RefreshInterfaces(); err != nil {
		return ErrRefresh
	}
	return nil
}

// SetMTU sets the interface MTU, defaulting to the current interface when iface
// is empty, and refreshes interfaces best-effort. It returns the resolved
// interface name so the handler can echo it.
func (s *Service) SetMTU(iface string, mtu int) (string, error) {
	if iface == "" {
		iface = s.hw.CurrentInterface()
	}
	if err := s.hw.SetMTU(iface, mtu); err != nil {
		return iface, ErrSetMTU
	}
	// Refresh is best-effort: the MTU change already succeeded.
	_ = s.hw.RefreshInterfaces()
	return iface, nil
}
