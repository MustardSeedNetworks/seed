package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/foundation/pkg/instance"

	"github.com/MustardSeedNetworks/seed/internal/api"
	"github.com/MustardSeedNetworks/seed/internal/config"
)

// opsInstall is a throwaway seed install: a config directory, a working
// directory for the daemon's relative paths, and the files a daemon leaves.
type opsInstall struct {
	t          *testing.T
	configPath string
	workDir    string
	logFile    string
	dbPath     string
	certFile   string
	keyFile    string
}

// newOpsInstall writes a config with absolute paths into a temp directory and
// isolates every path resolver from the developer's own seed install.
func newOpsInstall(t *testing.T, mutate func(doc map[string]any)) *opsInstall {
	t.Helper()
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "xdg-config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "xdg-data"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "xdg-state"))
	t.Setenv("SEED_CONFIG_PATH", "")
	t.Setenv(keyringEnv, "")

	in := &opsInstall{
		t:          t,
		configPath: filepath.Join(root, "etc", "seed.json"),
		workDir:    filepath.Join(root, "work"),
		logFile:    filepath.Join(root, "work", "logs", "seed.log"),
		dbPath:     filepath.Join(root, "work", "data", "seed.db"),
		certFile:   filepath.Join(root, "work", "certs", "server.crt"),
		keyFile:    filepath.Join(root, "work", "certs", "server.key"),
	}
	raw, err := json.Marshal(config.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err = json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	setPath(doc, "logging.file", in.logFile)
	setPath(doc, "database.path", in.dbPath)
	setPath(doc, "server.cert_file", in.certFile)
	setPath(doc, "server.key_file", in.keyFile)
	if mutate != nil {
		mutate(doc)
	}
	in.writeConfig(doc)
	return in
}

func setPath(doc map[string]any, dotted string, value any) {
	parts := strings.Split(dotted, ".")
	node := doc
	for _, part := range parts[:len(parts)-1] {
		next, ok := node[part].(map[string]any)
		if !ok {
			next = map[string]any{}
			node[part] = next
		}
		node = next
	}
	node[parts[len(parts)-1]] = value
}

func (in *opsInstall) writeConfig(doc map[string]any) {
	in.t.Helper()
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		in.t.Fatal(err)
	}
	in.writeFile(in.configPath, string(raw))
}

func (in *opsInstall) writeFile(path, content string) {
	in.t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		in.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		in.t.Fatal(err)
	}
}

func (in *opsInstall) read(path string) (string, bool) {
	in.t.Helper()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", false
	}
	if err != nil {
		in.t.Fatal(err)
	}
	return string(data), true
}

func (in *opsInstall) keyringPath() string {
	return filepath.Join(filepath.Dir(in.configPath), config.KeyringFileName)
}

func (in *opsInstall) run(args ...string) (int, string, string) {
	in.t.Helper()
	full := append([]string{"--config", in.configPath, "--workdir", in.workDir}, args...)
	var out, errOut bytes.Buffer
	code := runCLI(full, &out, &errOut)
	return code, out.String(), errOut.String()
}

// holdLock stands in for a running seed serve: it takes the instance lock the
// daemon takes and publishes the port.
func (in *opsInstall) holdLock(port int) {
	in.t.Helper()
	lock, err := instance.Acquire(filepath.Dir(in.configPath))
	if err != nil {
		in.t.Fatalf("take instance lock: %v", err)
	}
	if port != 0 {
		if err = lock.SetPort(port); err != nil {
			in.t.Fatal(err)
		}
	}
	in.t.Cleanup(func() { _ = lock.Release() })
}

// serveVersion runs a TLS server that answers /__version the way the daemon
// does, writes its certificate where the daemon would, and returns its port.
func (in *opsInstall) serveVersion() int {
	in.t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/__version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.BuildVersionResponse{
			Version: "9.9.9-test", Commit: "abc1234", BuildTime: "2026-11-21T00:00:00Z",
			ReleaseTrain: "2026.11", UIBuildHash: "uihash", TLSFingerprint: "AA:BB",
		})
	})
	srv := httptest.NewTLSServer(mux)
	in.t.Cleanup(srv.Close)
	in.writeFile(in.certFile, string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})))
	_, port, err := net.SplitHostPort(srv.Listener.Addr().String())
	if err != nil {
		in.t.Fatal(err)
	}
	n, err := strconv.Atoi(port)
	if err != nil {
		in.t.Fatal(err)
	}
	return n
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()
	return l.Addr().(*net.TCPAddr).Port
}

func canBreakLock() bool { return runtime.GOOS != "windows" && os.Geteuid() != 0 }

type bundleContents map[string]string

func readTarGz(t *testing.T, path string) bundleContents {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	files := bundleContents{}
	tr := tar.NewReader(gz)
	for {
		header, nextErr := tr.Next()
		if errors.Is(nextErr, io.EOF) {
			return files
		}
		if nextErr != nil {
			t.Fatal(nextErr)
		}
		data, readErr := io.ReadAll(tr)
		if readErr != nil {
			t.Fatal(readErr)
		}
		files[header.Name] = string(data)
	}
}

func (c bundleContents) all() string {
	var sb strings.Builder
	for name, content := range c {
		sb.WriteString(name)
		sb.WriteString("\n")
		sb.WriteString(content)
		sb.WriteString("\n")
	}
	return sb.String()
}

// The values the bundle must never carry. Each one is planted in the config,
// in the log, or in a file the bundle must not read.

func keys(m bundleContents) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}
