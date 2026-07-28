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
	t.Run("Linux DEB", func(t *testing.T) {
		contents := readRepositoryFile(t, "deploy", "deb", "seed.service")
		assertHTTPSOnlyServiceDefinition(t, contents, "/usr/bin/seed serve")
	})
	t.Run("Linux systemd", func(t *testing.T) {
		contents := readRepositoryFile(t, "deploy", "systemd", "seed.service")
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

func TestServerLifecycleHasNoPlainHTTPListener(t *testing.T) {
	contents := readRepositoryFile(t, "internal", "api", "server_lifecycle.go")
	if !strings.Contains(contents, ".ServeTLS(") {
		t.Fatal("server lifecycle does not contain the TLS listener")
	}
	for _, forbidden := range []string{
		"func (s *Server) startHTTP(",
		"return s.startHTTP()",
		".Serve(",
		"ListenAndServe(",
		"autocert",
		"golang.org/x/crypto/acme",
	} {
		if strings.Contains(contents, forbidden) {
			t.Errorf("server lifecycle contains plaintext or ACME path %q", forbidden)
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

func TestE2ERunnerRejectsPlaintextApplicationListener(t *testing.T) {
	contents := readRepositoryFile(t, "scripts", "run-e2e.sh")
	for _, required := range []string{
		`plain_url="http://${base_url#https://}"`,
		`curl -sf --max-time 2 "$plain_url/__version"`,
	} {
		if !strings.Contains(contents, required) {
			t.Fatalf("E2E runner is missing plaintext-listener assertion %q", required)
		}
	}
}

func TestProductionEntrypointTreeHasNoPlainHTTPServe(t *testing.T) {
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
			for _, forbidden := range []string{".Serve(", "ListenAndServe("} {
				if strings.Contains(string(contents), forbidden) {
					t.Errorf("%s contains plaintext HTTP serving primitive %q", path, forbidden)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
