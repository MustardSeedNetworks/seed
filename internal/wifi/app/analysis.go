package wifiapp

import (
	"time"

	"github.com/krisarmstrong/seed/internal/anomaly"
	"github.com/krisarmstrong/seed/internal/wifi/airspace"
	wifianomaly "github.com/krisarmstrong/seed/internal/wifi/anomaly"
)

// AnalyzeBSSes runs the Wi-Fi anomaly rules over a flat set of observed BSSes
// (e.g. captured during a site survey) and returns the resulting anomalies.
//
// It is a pure function of its inputs: it builds a transient airspace tree from
// bsses, runs the detector, and folds the detections through a fresh, single-use
// anomaly engine — no shared or persistent state with the live visibility
// engine. This lets callers that hold a point-in-time set of BSSes (the survey
// path) reuse the same rule catalog the streaming capture path uses.
//
// detector may be nil, in which case the default Wi-Fi detector is used. at
// timestamps the synthetic observations, so the returned anomalies' first/last
// seen reflect when the BSSes were measured.
func AnalyzeBSSes(
	bsses []airspace.BSSView,
	detector *wifianomaly.Detector,
	at time.Time,
) ([]anomaly.Anomaly, error) {
	if detector == nil {
		detector = wifianomaly.NewDetector()
	}
	catalog, err := wifianomaly.Catalog()
	if err != nil {
		return nil, err
	}

	engine := anomaly.NewEngine(catalog)
	tree := airspace.TreeFromBSSViews(bsses)
	for _, det := range detector.Detect(tree) {
		if obsErr := engine.Observe(det, at); obsErr != nil {
			return nil, obsErr
		}
	}
	return engine.Snapshot(), nil
}
