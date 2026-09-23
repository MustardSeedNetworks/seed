package auth_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/auth"
)

func TestBreachCheckRequiresExplicitOptIn(t *testing.T) {
	for _, value := range []string{"unset", "", "0", "false", "true", "invalid", " 1 ", "1"} {
		t.Run(value, func(t *testing.T) { checkBreachOptIn(t, value) })
	}
}

func checkBreachOptIn(t *testing.T, value string) {
	t.Helper()
	want := int32(0)
	if value == "1" {
		want = 1
	}
	if got := breachRequestCount(t, value); got != want {
		t.Fatalf("requests=%d, want %d", got, want)
	}
}

func breachRequestCount(t *testing.T, value string) int32 {
	t.Helper()
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	t.Cleanup(auth.SetHIBPEndpointForTest(server.URL + "/"))
	configureBreachCheck(t, value)
	breached, count, err := auth.CheckPasswordBreached(t.Context(), "offline-test-passphrase")
	if err != nil || breached || count != 0 {
		t.Fatalf("unexpected result: %v, %d, %v", breached, count, err)
	}
	return requests.Load()
}

func configureBreachCheck(t *testing.T, value string) {
	t.Helper()
	t.Setenv("SEED_ENABLE_HIBP", value)
	if value == "unset" {
		if err := os.Unsetenv("SEED_ENABLE_HIBP"); err != nil {
			t.Fatal(err)
		}
	}
}
