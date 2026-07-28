package api_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/api"
	"github.com/MustardSeedNetworks/seed/internal/config"
)

func TestProductionListenerServesApplicationOnlyOverTLS(t *testing.T) {
	held := holdFallbackBasePort(t)
	defer func() { _ = held.Close() }()

	cfg := config.DefaultConfig()
	cfg.Server.Port = held.Addr().(*net.TCPAddr).Port
	server := api.NewTestServerWithConfig(cfg)
	defer server.Close()
	t.Chdir(t.TempDir())
	if _, _, err := server.EnsureSelfSignedCert(); err != nil {
		t.Fatal(err)
	}
	transport := trustedTransport(t)
	t.Cleanup(transport.CloseIdleConnections)

	serveErr := make(chan error, 1)
	initialized := make(chan struct{})
	go func() {
		serveErr <- server.StartHTTPSForTest(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, "tls-only")
		}), initialized)
	}()
	<-initialized
	var listenerErr error
	var shutdownOnce sync.Once
	shutdown := func() {
		shutdownOnce.Do(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := server.ShutdownHTTPSForTest(ctx); err != nil {
				listenerErr = err
				return
			}
			listenerErr = <-serveErr
		})
	}
	t.Cleanup(shutdown)

	deadline := time.Now().Add(30 * time.Second)
	client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
	response, actualPort := waitForTLSListener(t, client, cfg.Server.Port, deadline)
	body, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || string(body) != "tls-only" {
		t.Fatalf("HTTPS response = %d %q", response.StatusCode, body)
	}
	if !server.WebAuthnEnabledForTest() {
		t.Fatal("bound HTTPS listener did not initialize WebAuthn")
	}

	assertNoPlaintextApplication(t, actualPort)

	shutdown()
	if listenerErr == nil || !strings.Contains(listenerErr.Error(), http.ErrServerClosed.Error()) {
		t.Fatalf("listener exit = %v, want %v", listenerErr, http.ErrServerClosed)
	}
}

func holdFallbackBasePort(t *testing.T) net.Listener {
	t.Helper()
	for range 20 {
		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			t.Fatal(err)
		}
		if listener.Addr().(*net.TCPAddr).Port <= 65526 {
			return listener
		}
		_ = listener.Close()
	}
	t.Fatal("could not allocate a port with a complete +1..+9 fallback range")
	return nil
}

func trustedTransport(t *testing.T) *http.Transport {
	t.Helper()
	certPEM, err := os.ReadFile("certs/server.crt")
	if err != nil {
		t.Fatalf("generated certificate was not ready: %v", err)
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(certPEM) {
		t.Fatal("generated certificate did not parse")
	}
	return &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: roots}}
}

func waitForTLSListener(
	t *testing.T,
	client *http.Client,
	requestedPort int,
	deadline time.Time,
) (*http.Response, int) {
	t.Helper()
	var lastErr error
	for time.Now().Before(deadline) {
		for candidate := requestedPort + 1; candidate <= requestedPort+9; candidate++ {
			response, err := client.Get("https://localhost:" + strconv.Itoa(candidate))
			if err == nil {
				return response, candidate
			}
			lastErr = err
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("HTTPS fallback listener did not become ready: %v", lastErr)
	return nil, 0
}

func assertNoPlaintextApplication(t *testing.T, port int) {
	t.Helper()
	response, err := (&http.Client{Timeout: 5 * time.Second}).Get(
		"http://127.0.0.1:" + strconv.Itoa(port),
	)
	if err != nil {
		return
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode == http.StatusOK || strings.Contains(string(body), "tls-only") {
		t.Fatalf("application responded over plaintext HTTP: %d %q", response.StatusCode, body)
	}
}
