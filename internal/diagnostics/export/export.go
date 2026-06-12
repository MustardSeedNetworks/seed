// Package export is the diagnostic-export use-case (ADR-0020, WS-A10): the
// /api/v1/export endpoint's application service. It coordinates the per-card
// diagnostic sources into one export document, owning the assembly logic the
// transport layer used to carry as a fan-out of *Server methods. The card data
// is gathered through the consumer-defined Sources port, satisfied by an adapter
// in the api layer over the live diagnostic services.
package export

import (
	"context"
	"time"
)

// DeviceInfo is the export's device summary.
type DeviceInfo struct {
	Interface string `json:"interface"`
	MAC       string `json:"mac,omitempty"`
	IPMode    string `json:"ipMode"`
}

// Data is the full diagnostic export document.
type Data struct {
	Version   string         `json:"version"`
	Timestamp string         `json:"timestamp"`
	Device    DeviceInfo     `json:"device"`
	Cards     map[string]any `json:"cards"`
}

// Sources provides the export's device info and the assembled per-card
// diagnostic data. Cards returns the present cards keyed by name (absent cards
// are simply omitted); the per-card gathering/shaping lives in the adapter.
type Sources interface {
	RefreshInterfaces() error
	DeviceMAC(iface string) string
	IPMode() string
	Cards(ctx context.Context, iface string) map[string]any
}

// Service is the diagnostic-export use-case.
type Service struct {
	src Sources
}

// NewService builds the use-case over its Sources port.
func NewService(src Sources) *Service { return &Service{src: src} }

// Build assembles the export document for iface, stamping it with ver. It first
// refreshes the interface set (an error there aborts the export), then composes
// the device summary and the gathered cards.
func (s *Service) Build(ctx context.Context, iface, ver string) (Data, error) {
	if err := s.src.RefreshInterfaces(); err != nil {
		return Data{}, err
	}
	return Data{
		Version:   ver,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Device: DeviceInfo{
			Interface: iface,
			MAC:       s.src.DeviceMAC(iface),
			IPMode:    s.src.IPMode(),
		},
		Cards: s.src.Cards(ctx, iface),
	}, nil
}
