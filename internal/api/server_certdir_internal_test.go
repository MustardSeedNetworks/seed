package api

import (
	"path/filepath"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/config"
)

func TestNewServerCertDir(t *testing.T) {
	custom := filepath.Join(t.TempDir(), "certs")
	tests := []struct {
		name    string
		certDir string
		want    string
	}{
		{"unset keeps the working-directory default", "", selfSignedCertDir},
		{"set is used for the generated pair and its fingerprint", custom, custom},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := NewServer(Deps{Config: config.DefaultConfig(), ConfigPath: "seed.json", CertDir: tt.certDir})
			if err != nil {
				t.Fatalf("NewServer: %v", err)
			}
			if s.certDir != tt.want {
				t.Errorf("certDir = %q, want %q", s.certDir, tt.want)
			}
			if got, want := s.activeCertPath(), filepath.Join(tt.want, "server.crt"); got != want {
				t.Errorf("activeCertPath = %q, want %q", got, want)
			}
		})
	}
}
