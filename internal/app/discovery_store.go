package app

// discovery_store.go supplies the database writers discovery and the
// vulnerability scanner persist through: devices keyed by MAC, and each scan
// pass's findings under the device's persisted id.

import (
	"context"
	"errors"
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

// PersistDevices upserts each device by MAC, so a device that comes back on a
// new address is updated in place. One failing device does not drop the rest
// of the flush; every failure is returned together.
func (w deviceWriter) PersistDevices(ctx context.Context, devices []*discovery.DiscoveredDevice) error {
	var errs []error
	for _, d := range devices {
		if _, err := w.persist(ctx, d); err != nil {
			errs = append(errs, fmt.Errorf("persisting device %s: %w", d.IP, err))
		}
	}
	return errors.Join(errs...)
}

// persist upserts one device and returns its stored id. A device without a MAC
// falls back to its address.
func (w deviceWriter) persist(ctx context.Context, d *discovery.DiscoveredDevice) (string, error) {
	rec := &database.Device{
		ID:         d.MAC,
		IPAddress:  d.IP,
		MACAddress: d.MAC,
		Hostname:   d.Hostname,
		Vendor:     d.Vendor,
		DeviceType: d.OSGuess,
		LastSeen:   d.LastSeen,
		IsActive:   true,
	}
	if err := w.db.Devices().UpsertByMAC(ctx, rec); err != nil {
		return "", err
	}
	return rec.ID, nil
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
	deviceID, err := s.devices.persist(ctx, device)
	if err != nil {
		return fmt.Errorf("persisting device %s: %w", device.IP, err)
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
	return s.db.Vulnerabilities().RecordScan(ctx, deviceID, findings, result.ScanTime)
}
