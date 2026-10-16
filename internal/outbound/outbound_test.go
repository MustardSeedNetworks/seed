package outbound_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/outbound"
)

func TestValidateEndpoint(t *testing.T) {
	t.Parallel()
	tests := []struct {
		raw     string
		wantErr string
	}{
		{raw: "https://api.meraki.com/api/v1"},
		{raw: "https://10.0.0.5:8443/eox"},
		{raw: "http://api.meraki.com/api/v1", wantErr: "must be https"},
		{raw: "api.meraki.com/api/v1", wantErr: "must be https"},
		{raw: "https:///api/v1", wantErr: "must name a host"},
		{raw: "https://user:pw@api.meraki.com", wantErr: "must not carry credentials"},
		{raw: "https://api.meraki.com/api/v1?key=x", wantErr: "must not carry a query or fragment"},
		{raw: "https://api.meraki.com/api/v1#frag", wantErr: "must not carry a query or fragment"},
		{raw: "https://[::1", wantErr: "not a URL"},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			t.Parallel()
			err := outbound.ValidateEndpoint(tt.raw)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestChoiceValidAnswer(t *testing.T) {
	t.Parallel()
	assert.True(t, outbound.ChoiceOn.ValidAnswer())
	assert.True(t, outbound.ChoiceOff.ValidAnswer())
	assert.False(t, outbound.ChoiceUnanswered.ValidAnswer(), "the question cannot be un-asked")
	assert.False(t, outbound.Choice("maybe").ValidAnswer())
}

// TestNewClientFollowsNoRedirect: a redirect would carry the request's secret
// to a host the operator never configured.
func TestNewClientFollowsNoRedirect(t *testing.T) {
	t.Parallel()
	var followed bool
	elsewhere := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { followed = true }))
	t.Cleanup(elsewhere.Close)
	origin := httptest.NewServer(http.RedirectHandler(elsewhere.URL, http.StatusFound))
	t.Cleanup(origin.Close)

	resp, err := outbound.NewClient(time.Second).Get(origin.URL)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, http.StatusFound, resp.StatusCode)
	assert.False(t, followed)
}
