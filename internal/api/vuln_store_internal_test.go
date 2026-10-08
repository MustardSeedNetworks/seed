package api

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/database/dbtest"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/discovery/vuln"
	"github.com/MustardSeedNetworks/seed/internal/reporting"
	"github.com/MustardSeedNetworks/seed/internal/reporting/store"
)

// cveFixture is a local CVE database with one critical kernel CVE, matched by
// a device whose vendor is "Linux" and whose OS guess is "Linux 5.4".
const cveFixture = `{
  "version": "1",
  "cves": [{
    "id": "CVE-2099-0001",
    "description": "Kernel use-after-free in the fixture driver",
    "severity": "CRITICAL",
    "cvssScore": 9.8,
    "vendor": "linux",
    "product": "linux-kernel",
    "versions": ["5.4"]
  }]
}`

func newStoredVulnScanner(t *testing.T, db *database.DB) *vuln.VulnerabilityScanner {
	t.Helper()
	cvePath := filepath.Join(t.TempDir(), "cves.json")
	require.NoError(t, os.WriteFile(cvePath, []byte(cveFixture), 0o600))
	scanner, err := vuln.NewVulnerabilityScanner(&vuln.VulnerabilityScannerConfig{
		Enabled:           true,
		CVEDatabase:       "local",
		LocalDBPath:       cvePath,
		SeverityThreshold: "low",
	})
	require.NoError(t, err)
	scanner.SetStore(app.NewVulnStore(db))
	return scanner
}

// TestVulnFindingsSurviveRestartIntoReports is D-SEED-77's acceptance (#2628):
// findings from a scan reach report aggregation and the export after the
// daemon's database is closed and reopened.
func TestVulnFindingsSurviveRestartIntoReports(t *testing.T) {
	ctx := context.Background()
	dbPath := dbtest.Path(t)

	db, err := database.Open(dbPath)
	require.NoError(t, err)
	scanner := newStoredVulnScanner(t, db)
	device := &discovery.DiscoveredDevice{
		IP: "10.20.30.40", MAC: "02:00:00:00:00:01", Vendor: "Linux",
		OSGuess: "Linux 5.4", LastSeen: time.Now(),
	}
	result, err := scanner.ScanDevice(ctx, device)
	require.NoError(t, err)
	require.Len(t, result.Vulnerabilities, 1)
	require.NoError(t, db.Close())

	db, err = database.Open(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	data, err := reporting.NewAggregatorService(config.DefaultConfig(), store.NewMetricsRepo(db)).
		Aggregate(ctx, reporting.PeriodWeekly, "", "")
	require.NoError(t, err)
	require.Equal(t, 1, data.VulnCount.Critical)
	require.Equal(t, 1, data.VulnCount.Total)
	require.Len(t, data.TopIssues, 1)
	require.Equal(t, "Kernel use-after-free in the fixture driver", data.TopIssues[0].Description)

	exported, err := store.NewExportRepo(db).ExportVulnerabilities(ctx)
	require.NoError(t, err)
	require.Len(t, exported, 1)
	require.Equal(t, "CVE-2099-0001", exported[0]["cve_id"])
	ip, ok := exported[0]["device_ip"].(*string)
	require.True(t, ok)
	require.Equal(t, device.IP, *ip)
}

// TestVulnRescanResolvesAndReopens covers the pass-to-pass lifecycle: a CVE a
// later pass no longer reports leaves the report, and returns when reported
// again.
func TestVulnRescanResolvesAndReopens(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(dbtest.Path(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	scanner := newStoredVulnScanner(t, db)
	metrics := store.NewMetricsRepo(db)
	openCount := func() int {
		counts, countErr := metrics.VulnerabilitySeverityCounts(ctx, time.Now().Add(-time.Hour))
		require.NoError(t, countErr)
		return counts["critical"]
	}
	device := &discovery.DiscoveredDevice{
		IP: "10.20.30.41", MAC: "02:00:00:00:00:02", Vendor: "Linux", OSGuess: "Linux 5.4",
	}

	_, err = scanner.ScanDevice(ctx, device)
	require.NoError(t, err)
	require.Equal(t, 1, openCount())

	device.OSGuess = "Windows 10" // no fixture CVE matches this pass
	_, err = scanner.ScanDevice(ctx, device)
	require.NoError(t, err)
	require.Equal(t, 0, openCount())

	device.OSGuess = "Linux 5.4"
	_, err = scanner.ScanDevice(ctx, device)
	require.NoError(t, err)
	require.Equal(t, 1, openCount())
}

// TestVulnFindingsFollowRenumberedDevice: a device rescanned at a new address
// keeps one finding under its MAC-keyed row, reported at the new address
// (seed#3210).
func TestVulnFindingsFollowRenumberedDevice(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(dbtest.Path(t))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	scanner := newStoredVulnScanner(t, db)
	device := &discovery.DiscoveredDevice{
		IP: "10.20.30.42", MAC: "02:00:00:00:00:03", Vendor: "Linux", OSGuess: "Linux 5.4",
	}
	_, err = scanner.ScanDevice(ctx, device)
	require.NoError(t, err)

	device.IP = "10.20.30.43"
	_, err = scanner.ScanDevice(ctx, device)
	require.NoError(t, err)

	exported, err := store.NewExportRepo(db).ExportVulnerabilities(ctx)
	require.NoError(t, err)
	require.Len(t, exported, 1)
	ip, ok := exported[0]["device_ip"].(*string)
	require.True(t, ok)
	require.Equal(t, device.IP, *ip)
}
