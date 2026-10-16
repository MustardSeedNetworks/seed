package outbound_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/outbound"
)

func TestUpdateValidate(t *testing.T) {
	t.Parallel()
	const vendorDefault = "https://api.meraki.com/api/v1"
	tests := []struct {
		name            string
		update          outbound.Update
		hasKey          bool
		defaultEndpoint string
		wantErr         string
	}{
		{name: "answer off", update: outbound.Update{Choice: outbound.ChoiceOff}},
		{
			name:            "on with a key",
			update:          outbound.Update{Choice: outbound.ChoiceOn, APIKey: "k"},
			defaultEndpoint: vendorDefault,
		},
		{
			name:            "on keeping the stored key",
			update:          outbound.Update{Choice: outbound.ChoiceOn},
			hasKey:          true,
			defaultEndpoint: vendorDefault,
		},
		{
			name:   "on naming the endpoint a vendor without a default needs",
			update: outbound.Update{Choice: outbound.ChoiceOn, APIKey: "k", Endpoint: "https://fw1.example.net"},
		},
		{
			name:   "off without an endpoint a vendor without a default needs",
			update: outbound.Update{Choice: outbound.ChoiceOff, APIKey: "k"},
		},
		{
			name:    "un-asking",
			update:  outbound.Update{Choice: outbound.ChoiceUnanswered},
			wantErr: `choice must be "on" or "off"`,
		},
		{
			name:            "on without a key",
			update:          outbound.Update{Choice: outbound.ChoiceOn},
			defaultEndpoint: vendorDefault,
			wantErr:         "needs an API key",
		},
		{
			name:            "on while clearing the stored key",
			update:          outbound.Update{Choice: outbound.ChoiceOn, ClearAPIKey: true},
			hasKey:          true,
			defaultEndpoint: vendorDefault,
			wantErr:         "needs an API key",
		},
		{
			name:    "on without an endpoint when the vendor has no default",
			update:  outbound.Update{Choice: outbound.ChoiceOn, APIKey: "k"},
			wantErr: "needs an endpoint",
		},
		{
			name:    "plain http",
			update:  outbound.Update{Choice: outbound.ChoiceOff, Endpoint: "http://api.meraki.com/api/v1"},
			wantErr: "endpoint: must be https",
		},
		{
			name:    "key and clear",
			update:  outbound.Update{Choice: outbound.ChoiceOff, APIKey: "k", ClearAPIKey: true},
			wantErr: "exclusive",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.update.Validate(tt.hasKey, tt.defaultEndpoint)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, outbound.ErrInvalidSettings)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
