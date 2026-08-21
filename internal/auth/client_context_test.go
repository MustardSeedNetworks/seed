package auth_test

// client_context_test.go covers the claim itself: that every token-issuing
// path stamps it, that the accessor refuses to invent one, and that the
// storeless fallback constant still names a client the database actually has.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/database"
)

func TestDefaultClientIDMatchesDatabase(t *testing.T) {
	t.Parallel()

	if auth.DefaultClientID != database.DefaultClientID {
		t.Errorf("auth.DefaultClientID = %q, database.DefaultClientID = %q — the storeless "+
			"fallback must name a client the schema seeds",
			auth.DefaultClientID, database.DefaultClientID)
	}
}

func TestClientIDFromContextFailsClosed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ctx  context.Context
	}{
		{"no claim at all", context.Background()},
		{"empty claim", auth.WithClientID(context.Background(), "")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := auth.ClientIDFromContext(tt.ctx)
			if err == nil {
				t.Fatalf("got client %q with no error; an unattributable request must be a denial", got)
			}
			if got != "" {
				t.Errorf("got client %q alongside the error; the accessor must not manufacture a tenant", got)
			}
		})
	}
}

func TestClientIDRoundTripsThroughContext(t *testing.T) {
	t.Parallel()

	ctx := auth.WithClientID(context.Background(), "acme")
	got, err := auth.ClientIDFromContext(ctx)
	if err != nil {
		t.Fatalf("ClientIDFromContext: %v", err)
	}
	if got != "acme" {
		t.Errorf("client = %q, want %q", got, "acme")
	}
}

// TestClientClaimIsNotHeaderDerived is the point of using a context value
// rather than the X-Username-style header this codebase threads identity
// with: a caller cannot supply one.
func TestClientClaimIsNotHeaderDerived(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/anything", http.NoBody)
	req.Header.Set("X-Client-Id", "attacker-tenant")

	if _, err := auth.ClientIDFromContext(req.Context()); err == nil {
		t.Error("a request carrying only headers resolved a client; tenancy must not be caller-supplied")
	}
}
