package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readRepositoryFile(t *testing.T, elements ...string) string {
	t.Helper()
	path := filepath.Join(append([]string{"..", ".."}, elements...)...)
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(contents)
}

func assertHTTPSOnlyServiceDefinition(t *testing.T, contents, command string) {
	t.Helper()
	if !strings.Contains(contents, command) {
		t.Fatalf("service definition does not invoke %q", command)
	}
	for _, forbidden := range []string{
		"SEED_HTTPS_ENABLED",
		"SEED_ACME_",
		"--http",
		"--no-tls",
	} {
		if strings.Contains(contents, forbidden) {
			t.Errorf("service definition contains removed TLS control %q", forbidden)
		}
	}
}

func TestPlatformServicesUseHTTPSOnlyEntrypoint(t *testing.T) {
	// deploy/deb/seed.service is the only Linux unit: it is what the .deb and
	// .rpm install. deploy/systemd/seed.service was a byte-identical copy that
	// existed for the hand-rolled installer deleted in #2634.
	t.Run("Linux DEB", func(t *testing.T) {
		contents := readRepositoryFile(t, "deploy", "deb", "seed.service")
		assertHTTPSOnlyServiceDefinition(t, contents, "/usr/bin/seed serve")
	})
	t.Run("macOS launchd", func(t *testing.T) {
		contents := readRepositoryFile(t, "deploy", "launchd", "com.seed.plist")
		assertHTTPSOnlyServiceDefinition(t, contents, "/usr/local/seed/seed")
	})
	t.Run("Windows service", func(t *testing.T) {
		contents := readRepositoryFile(t, "cmd", "seed", "cmd_service_windows.go")
		assertHTTPSOnlyServiceDefinition(t, contents, "p.server.Start()")
	})
}

// TestServerLifecycleBindsOneRedirectingTLSListener pins the F-6 contract:
// seed binds exactly one listener, foundation's httpserver.Listen, which
// serves TLS and answers a plaintext request on the same port with a 308 to
// https and nothing else. internal/api's TestProductionListenerServesApplicationOnlyOverTLS
// asserts that behaviour on a real socket.
func TestServerLifecycleBindsOneRedirectingTLSListener(t *testing.T) {
	contents := readRepositoryFile(t, "internal", "api", "server_lifecycle.go")
	for _, required := range []string{"httpserver.Listen(", ".Serve(ln)"} {
		if !strings.Contains(contents, required) {
			t.Fatalf("server lifecycle does not contain %q", required)
		}
	}
	for _, forbidden := range []string{
		// httpserver.Listen already returns a TLS listener; ServeTLS on it
		// would wrap TLS in TLS.
		".ServeTLS(",
		"func (s *Server) startHTTP(",
		"return s.startHTTP()",
		"ListenAndServe(",
		"autocert",
		"golang.org/x/crypto/acme",
	} {
		if strings.Contains(contents, forbidden) {
			t.Errorf("server lifecycle contains %q", forbidden)
		}
	}
}

func TestReverseProxyExampleUsesVerifiedHTTPSUpstream(t *testing.T) {
	contents := readRepositoryFile(t, "internal", "api", "proxy.go")
	for _, required := range []string{
		"proxy_pass https://localhost:8443",
		"proxy_ssl_trusted_certificate",
		"proxy_ssl_verify on",
	} {
		if !strings.Contains(contents, required) {
			t.Fatalf("reverse-proxy example is missing %q", required)
		}
	}
	if strings.Contains(contents, "proxy_pass http://") {
		t.Fatal("reverse-proxy example uses a plaintext upstream")
	}
}

func TestE2ERunnerRequiresPlaintextRedirect(t *testing.T) {
	contents := readRepositoryFile(t, "scripts", "run-e2e.sh")
	for _, required := range []string{
		`plain_url="http://${base_url#https://}"`,
		`if [ "$plain_reply" != "308 0 $base_url/__version" ]; then`,
	} {
		if !strings.Contains(contents, required) {
			t.Fatalf("E2E runner is missing plaintext-listener assertion %q", required)
		}
	}
}

// TestProductionEntrypointTreeHasOneListener walks cmd/seed and internal/api:
// httpserver.Listen is called once, http.Server.Serve only on its listener,
// and no other primitive opens or wraps a TCP listener for HTTP.
func TestProductionEntrypointTreeHasOneListener(t *testing.T) {
	lifecycle := filepath.Join("..", "..", "internal", "api", "server_lifecycle.go")
	listenCalls := 0
	for _, root := range []string{filepath.Join("..", "..", "cmd", "seed"), filepath.Join("..", "..", "internal", "api")} {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			contents, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			listenCalls += checkListenerPrimitives(t, path, string(contents), path == lifecycle)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if listenCalls != 1 {
		t.Errorf("httpserver.Listen called %d times, want exactly 1", listenCalls)
	}
}

// checkListenerPrimitives reports any listener primitive in one source file
// and returns how many times it calls httpserver.Listen.
func checkListenerPrimitives(t *testing.T, path, contents string, isLifecycle bool) int {
	t.Helper()
	for _, forbidden := range []string{
		"ListenAndServe(",
		"ListenAndServeTLS(",
		"httpserver.Bind(",
		"httpserver.BindExact(",
		"httpserver.PlaintextRedirect(",
		"tls.NewListener(",
	} {
		if strings.Contains(contents, forbidden) {
			t.Errorf("%s contains listener primitive %q", path, forbidden)
		}
	}
	if !isLifecycle && strings.Contains(contents, ".Serve(") {
		t.Errorf("%s serves HTTP outside server_lifecycle.go", path)
	}
	return strings.Count(contents, "httpserver.Listen(")
}
