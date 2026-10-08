package app

// discovery_store.go supplies the database writers discovery and the
// vulnerability scanner persist through: devices keyed by MAC, and each scan
// pass's findings under the device's persisted id.

import (
	"context"
	"fmt"
	"strings"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/discovery/vuln"
)

// NewDeviceWriter returns discovery's database writer.
func NewDeviceWriter(db *database.DB) discovery.DBDeviceWriter {
	return deviceWriter{db: db}
}

// NewVulnStore returns the vulnerability scanner's store. It persists the
// device first, so a finding never waits on the next discovery flush.
func NewVulnStore(db *database.DB) vuln.Store {
	return vulnStore{devices: deviceWriter{db: db}, db: db}
}

type deviceWriter struct {
	db *database.DB
}

func (w deviceWriter) PersistDevices(ctx context.Context, devices []*discovery.DiscoveredDevice) error {
	for _, d := range devices {
		if err := w.db.Devices().Upsert(ctx, &database.Device{
			ID:         d.MAC,
			IPAddress:  d.IP,
			MACAddress: d.MAC,
			Hostname:   d.Hostname,
			Vendor:     d.Vendor,
			DeviceType: d.OSGuess,
			LastSeen:   d.LastSeen,
			IsActive:   true,
		}); err != nil {
			return err
		}
	}
	return nil
}

type vulnStore struct {
	devices deviceWriter
	db      *database.DB
}

func (s vulnStore) SaveScan(
	ctx context.Context,
	device *discovery.DiscoveredDevice,
	result *discovery.DeviceVulnerabilities,
) error {
	if err := s.devices.PersistDevices(ctx, []*discovery.DiscoveredDevice{device}); err != nil {
		return err
	}
	stored, err := s.db.Devices().GetByIP(ctx, device.IP)
	if err != nil {
		return fmt.Errorf("resolving device id for %s: %w", device.IP, err)
	}

	findings := make([]database.VulnerabilityFinding, 0, len(result.Vulnerabilities))
	for i := range result.Vulnerabilities {
		v := &result.Vulnerabilities[i]
		findings = append(findings, database.VulnerabilityFinding{
			CVEID: v.CVEID,
			// Providers report NVD's upper-case severities; reports and the
			// scanner's own filter work in lower case.
			Severity:          strings.ToLower(v.Severity),
			CVSSScore:         v.Score,
			Description:       v.Description,
			AffectedComponent: result.Product,
			AffectedVersion:   result.Version,
		})
	}
	return s.db.Vulnerabilities().RecordScan(ctx, stored.ID, findings, result.ScanTime)
}
