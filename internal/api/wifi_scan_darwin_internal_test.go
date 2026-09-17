//go:build darwin

package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/wifi"
	"github.com/MustardSeedNetworks/seed/internal/wifi/troubleshooting"
)

type withheldRadio struct{ troubleshooting.Hardware }

func (withheldRadio) ManagerAvailable() bool { return true }
func (withheldRadio) ScannerAvailable() bool { return true }
func (withheldRadio) IsWireless() bool       { return true }
func (withheldRadio) Scan() ([]*wifi.ScannedNetwork, error) {
	return nil, errors.Join(wifi.ErrDetailsWithheld, errors.New("no helper agent connected"))
}

func TestWiFiScanHandlersPreserveWithheldDetails(t *testing.T) {
	t.Parallel()
	s := &Server{wifiManagement: troubleshooting.NewManagement(withheldRadio{}, nil, nil)}
	handlers := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"scan", s.handleWiFiScan},
		{"channel graph", s.handleWiFiChannelGraph},
	}
	for _, tt := range handlers {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			response := httptest.NewRecorder()
			tt.handler(response, httptest.NewRequest(http.MethodGet, "/?interface=en0", nil))
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d", response.Code)
			}
			var body struct {
				Available   bool   `json:"available"`
				Error       string `json:"error"`
				Remediation string `json:"remediation"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			reason, remediation := wifi.WithheldExplanation()
			if !body.Available || body.Error != reason || body.Remediation != remediation || body.Error == "" ||
				body.Remediation == "" {
				t.Fatalf("response = %s", response.Body.String())
			}
		})
	}
}
