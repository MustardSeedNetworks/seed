package api

import (
	"context"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/engine"
	wifianomaly "github.com/MustardSeedNetworks/seed/internal/wifi/anomaly"
)

// The shared Coordinator is built over the database's anomaly store and loads
// what an earlier run persisted before any producer observes (ADR-0029 §5).
func TestInitDatabaseDependentServices_RestoresPersistedAnomalies(t *testing.T) {
	db := newTestDB(t)
	cat, err := anomaly.NewCatalog(wifianomaly.Defs()...)
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	earlier := anomaly.NewCoordinator(anomaly.NewEngine(cat), db.Anomalies())
	if err := earlier.Observe(context.Background(), anomaly.Detection{
		DefKey:  wifianomaly.DefOpenNetwork,
		Subject: anomaly.SubjectRef{Kind: anomaly.SubjectBSSID, ID: "aa:bb:cc:dd:ee:ff"},
		Source:  anomaly.SourceWiFi,
	}, time.Now()); err != nil {
		t.Fatalf("Observe: %v", err)
	}

	s := &Server{engines: engine.NewRegistry(nil), licenseDir: t.TempDir()}
	s.initDatabaseDependentServices(db)

	if s.anomalyCoord == nil {
		t.Fatal("anomaly coordinator not built")
	}
	snap := s.anomalyCoord.Engine().Snapshot()
	if len(snap) != 1 || snap[0].DefKey != wifianomaly.DefOpenNetwork {
		t.Errorf("restored anomalies = %+v, want the one persisted open-network instance", snap)
	}
}
