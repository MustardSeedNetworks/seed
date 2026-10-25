package api_test

import (
	"testing"

	"github.com/MustardSeedNetworks/foundation/pkg/passkey"

	"github.com/MustardSeedNetworks/seed/internal/api"
	"github.com/MustardSeedNetworks/seed/internal/config"
)

func TestPasskeyConfigUsesTrustedHTTPSOrigin(t *testing.T) {
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
			got := api.ExportPasskeyConfigFromServer(cfg, tt.listenerPort)
			if got.RPID != tt.wantRPID {
				t.Fatalf("RPID = %q, want %q", got.RPID, tt.wantRPID)
			}
			if got.Origin != tt.wantOrigin {
				t.Fatalf("Origin = %q, want %q", got.Origin, tt.wantOrigin)
			}
			if _, err := passkey.New(got); err != nil {
				t.Fatalf("passkey.New() error = %v", err)
			}
		})
	}
}
