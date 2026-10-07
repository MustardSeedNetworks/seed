package app

// speedtest.go keeps a finished speed test for the reports' bandwidth figures
// (#2623): the reporting store reads speedtest_results back.

import (
	"context"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/speedtest"
)

// RecordSpeedtest stores res as measured on iface. A nil database stores
// nothing: the test harness and a daemon without a store still run tests.
func RecordSpeedtest(ctx context.Context, db *database.DB, iface string, res *speedtest.Result) error {
	if db == nil {
		return nil
	}
	return db.Metrics().RecordSpeedTest(ctx, &database.SpeedTestResult{
		InterfaceName:  iface,
		ServerName:     res.Server,
		ServerLocation: res.Location,
		DownloadMbps:   res.Download,
		UploadMbps:     res.Upload,
		LatencyMs:      res.Latency,
		Timestamp:      res.Timestamp.UTC(),
	})
}
