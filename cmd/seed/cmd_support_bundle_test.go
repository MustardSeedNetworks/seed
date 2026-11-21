package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/support"
)

const (
	secretJWT       = "jwt-SENTINEL-6f1c2d9a"
	secretHash      = "$2a$12$SENTINELPASSWORDHASHxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	secretClient    = "client-secret-SENTINEL-77aa"
	secretNVD       = "nvd-key-SENTINEL-1234abcd"
	secretCommunity = "community-SENTINEL-snmp"
	secretKeyring   = "keyring-SENTINEL-bytes-9981"
	secretTLSKey    = "TLSKEY-SENTINEL-material"
	secretDB        = "database-SENTINEL-row"
	secretEnvKey    = "env-keyring-SENTINEL-4455"
	secretBearer    = "bearerSENTINELtokenvalue123"
)

func TestSupportBundleNeverCarriesSecrets(t *testing.T) {
	in := newOpsInstall(t, func(doc map[string]any) {
		setPath(doc, "auth.jwt_secret", secretJWT)
		setPath(doc, "auth.default_password_hash", secretHash)
		setPath(doc, "security.vulnerability_scanning.nvd_api_key", secretNVD)
		setPath(doc, "auth.sso.providers", []any{
			map[string]any{"name": "okta", "client_secret": secretClient, "enabled": true},
		})
		setPath(doc, "alerts.trap.community", secretCommunity)
	})
	t.Setenv(keyringEnv, secretEnvKey)
	in.writeFile(in.keyringPath(), secretKeyring)
	in.writeFile(in.keyFile, "-----BEGIN PRIVATE KEY-----\n"+secretTLSKey+"\n-----END PRIVATE KEY-----\n")
	in.writeFile(in.dbPath, secretDB)
	in.writeFile(in.logFile, strings.Join([]string{
		"time=2026-11-21 level=INFO msg=\"Seed starting\" port=8443",
		"time=2026-11-21 level=WARN msg=\"auth failed\" jwt=" + secretJWT,
		"time=2026-11-21 level=INFO msg=\"snmp poll\" community=" + secretCommunity + " target=10.1.1.1",
		"time=2026-11-21 level=DEBUG msg=\"request\" Authorization: Bearer " + secretBearer,
		"time=2026-11-21 level=ERROR msg=\"keyring\" value=" + secretEnvKey,
		"-----BEGIN PRIVATE KEY-----", secretTLSKey, "-----END PRIVATE KEY-----",
		"time=2026-11-21 level=INFO msg=\"done\"",
	}, "\n")+"\n")

	out := filepath.Join(t.TempDir(), "bundle.tar.gz")
	code, stdout, stderr := in.run("support-bundle", "-o", out)
	if code != 0 {
		t.Fatalf("exit = %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	files := readTarGz(t, out)

	for _, want := range []string{"manifest.json", "config.json", "daemon.json", "platform.txt", "logs/seed.log"} {
		if _, ok := files[want]; !ok {
			t.Errorf("bundle lacks %s; has %v", want, keys(files))
		}
	}
	everything := files.all()
	for name, secret := range map[string]string{
		"jwt secret": secretJWT, "password hash": secretHash, "sso client secret": secretClient,
		"nvd api key": secretNVD, "trap community": secretCommunity, "keyring file": secretKeyring,
		"tls private key": secretTLSKey, "database": secretDB, "keyring env override": secretEnvKey,
		"bearer token": secretBearer,
	} {
		if strings.Contains(everything, secret) {
			t.Errorf("the bundle carries the %s (%q)", name, secret)
		}
	}
	if !strings.Contains(files["config.json"], support.Placeholder) {
		t.Error("config.json shows no sign that anything was redacted")
	}
	if !strings.Contains(files["config.json"], `"okta"`) {
		t.Error("redaction removed the non-secret SSO provider name")
	}
	if !strings.Contains(files["logs/seed.log"], "Seed starting") ||
		!strings.Contains(files["logs/seed.log"], "10.1.1.1") {
		t.Errorf("scrubbing removed the log's ordinary content:\n%s", files["logs/seed.log"])
	}
	var manifest bundleManifest
	if err := json.Unmarshal([]byte(files["manifest.json"]), &manifest); err != nil {
		t.Fatalf("manifest.json: %v", err)
	}
	if manifest.OS != runtime.GOOS || manifest.Arch != runtime.GOARCH || !manifest.ConfigFound || !manifest.LogFound {
		t.Errorf("manifest = %+v", manifest)
	}

	if code, _, _ = in.run("support-bundle", "-o", out); code != 1 {
		t.Errorf("overwriting an existing bundle exited %d, want 1", code)
	}
}

func TestSupportBundleOnAHostWhereSeedNeverRan(t *testing.T) {
	in := newOpsInstall(t, nil)
	if err := os.Remove(in.configPath); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "bundle.tar.gz")
	code, _, stderr := in.run("support-bundle", "-o", out)
	if code != 0 {
		t.Fatalf("exit = %d, stderr %q", code, stderr)
	}
	files := readTarGz(t, out)
	if _, ok := files["config.json"]; ok {
		t.Error("a bundle invented a config for a host with none")
	}
	if _, ok := files["manifest.json"]; !ok {
		t.Error("a bundle without a config or log still owes support the manifest")
	}
}
