package api_test

import (
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/api"
	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/config"
)

func TestWebAuthnConfigUsesTrustedHTTPSOrigin(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name         string
		listenerPort int
		origin       string
		wantRPID     string
		wantOrigin   string
	}{
		{
			name:         "local listener uses actual fallback port",
			listenerPort: 8445,
			wantRPID:     "localhost",
			wantOrigin:   "https://localhost:8445",
		},
		{
			name:         "configured origin remains exact",
			listenerPort: 8445,
			origin:       "https://seed.example.com:8443",
			wantRPID:     "seed.example.com",
			wantOrigin:   "https://seed.example.com:8443",
		},
		{
			name:         "reverse proxy keeps exact public origin",
			listenerPort: 8445,
			origin:       "https://seed.example.com",
			wantRPID:     "seed.example.com",
			wantOrigin:   "https://seed.example.com",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := config.DefaultConfig()
			cfg.Server.PublicOrigin = tt.origin
			got := api.ExportWebAuthnConfigFromServer(cfg, tt.listenerPort)
			if got.RPID != tt.wantRPID {
				t.Fatalf("RPID = %q, want %q", got.RPID, tt.wantRPID)
			}
			if len(got.RPOrigins) != 1 || !slices.Contains(got.RPOrigins, tt.wantOrigin) {
				t.Fatalf("RPOrigins = %v, want only %q", got.RPOrigins, tt.wantOrigin)
			}
			if _, err := auth.NewWebAuthnManager(got); err != nil {
				t.Fatalf("NewWebAuthnManager() error = %v", err)
			}
		})
	}
}
