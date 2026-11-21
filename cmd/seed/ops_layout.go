// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver"
	"github.com/MustardSeedNetworks/foundation/pkg/instance"
	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/seed/internal/api"
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/fsutil"
	"github.com/MustardSeedNetworks/seed/internal/paths"
	"github.com/MustardSeedNetworks/seed/internal/support"
)

const (
	workDirFlag = "workdir"
	// daemonTimeout bounds the one request status makes to the daemon.
	daemonTimeout = 5 * time.Second
	// maxVersionBody bounds the /__version response read; it is a few hundred
	// bytes.
	maxVersionBody = 1 << 16
	loopbackHost   = "127.0.0.1"
)

// layout is where one seed install keeps its files, resolved the way the
// daemon resolves them. The daemon opens its log, database and certificates
// relative to its working directory, so every operations command resolves
// them against that same directory instead of guessing at /var/log/seed or
// wherever the config happens to sit.
type layout struct {
	ConfigPath string
	WorkDir    string
	LogFile    string
	DBPath     string
	CertFile   string
	KeyFile    string
	Port       int
}

// addWorkDirFlag registers --workdir on an operations command.
func addWorkDirFlag(cmd *cobra.Command) {
	cmd.Flags().String(workDirFlag, "",
		"directory the daemon runs in, which relative paths in the config resolve against "+
			"(default: the installed service's data directory)")
}

// resolveLayout reads the install's config, without creating or changing
// anything, and resolves the paths the daemon would use. A missing config file
// is not an error: it resolves the defaults a first run would use.
func resolveLayout(state *cliState, cmd *cobra.Command) (layout, error) {
	workDir, err := cmd.Flags().GetString(workDirFlag)
	if err != nil {
		return layout{}, err
	}
	if workDir == "" {
		workDir = paths.Resolve(paths.ModeAuto).DataDir
	}
	configPath := paths.ResolveConfigPath(state.cfgFile, paths.ModeAuto)
	cfg, err := config.Load(configPath)
	if err != nil {
		return layout{}, fmt.Errorf("load config %s: %w", configPath, err)
	}

	l := layout{ConfigPath: configPath, WorkDir: workDir, Port: cfg.Server.Port}
	l.LogFile = l.resolve(firstSet(cfg.Logging.File, filepath.FromSlash(paths.DefaultLogFile)))
	l.DBPath = l.resolve(firstSet(cfg.Database.Path, filepath.FromSlash(paths.DefaultDatabaseFile)))
	l.CertFile = l.resolve(firstSet(cfg.Server.CertFile,
		filepath.Join(paths.DefaultCertDir, httpserver.DefaultCertFileName)))
	l.KeyFile = l.resolve(firstSet(cfg.Server.KeyFile,
		filepath.Join(paths.DefaultCertDir, httpserver.DefaultKeyFileName)))
	return l, nil
}

func firstSet(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func (l layout) resolve(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(l.WorkDir, p)
}

func (l layout) lockDir() string { return lockDir(l.ConfigPath) }

func (l layout) credentialKeyPath() string {
	return filepath.Join(filepath.Dir(l.ConfigPath), config.KeyringFileName)
}

// probe reports whether seed serve holds this install's instance lock. Unlike
// the write-refusing guards, which fail open, a command about to read or
// replace the install's state fails closed: a probe that cannot be read is an
// error, because "I could not tell" must not read as "nothing is running".
func (l layout) probe() (instance.Info, bool, error) {
	info, held, err := instance.Probe(l.lockDir())
	if err != nil {
		return instance.Info{}, false, fmt.Errorf("check whether seed serve is running on %s: %w", l.lockDir(), err)
	}
	return info, held, nil
}

// refuseWhileRunning returns a *daemonRunningError when seed serve holds the
// install, and an error when that cannot be determined.
func (l layout) refuseWhileRunning(instead string) error {
	info, held, err := l.probe()
	if err != nil {
		return err
	}
	if held {
		return &daemonRunningError{dir: l.lockDir(), info: info, instead: instead}
	}
	return nil
}

// stateMembers lists the install's files that exist, as backup entries.
func (l layout) stateMembers() ([]support.Member, error) {
	candidates := []support.Member{
		{Entry: support.EntryConfig, Path: l.ConfigPath},
		{Entry: support.EntryKeyring, Path: l.credentialKeyPath()},
		{Entry: support.EntryDatabase, Path: l.DBPath},
		{Entry: support.EntryDatabaseWAL, Path: l.DBPath + "-wal"},
		{Entry: support.EntryDatabaseSHM, Path: l.DBPath + "-shm"},
		{Entry: support.EntryCertificate, Path: l.CertFile},
		{Entry: support.EntryKey, Path: l.KeyFile},
	}
	var members []support.Member
	for _, candidate := range candidates {
		exists, err := fileExists(candidate.Path)
		if err != nil {
			return nil, err
		}
		if exists {
			members = append(members, candidate)
		}
	}
	return members, nil
}

// restoreTargets maps every archive entry to where it lands in this install.
func (l layout) restoreTargets() []support.Target {
	return []support.Target{
		{Entry: support.EntryConfig, Path: l.ConfigPath},
		{Entry: support.EntryKeyring, Path: l.credentialKeyPath()},
		{Entry: support.EntryDatabase, Path: l.DBPath},
		{Entry: support.EntryDatabaseWAL, Path: l.DBPath + "-wal", RemoveIfAbsent: true},
		{Entry: support.EntryDatabaseSHM, Path: l.DBPath + "-shm", RemoveIfAbsent: true},
		{Entry: support.EntryCertificate, Path: l.CertFile},
		{Entry: support.EntryKey, Path: l.KeyFile},
	}
}

// fileExists reports whether p names something, resolving it through an
// [os.Root] on its directory like every other operator-supplied path here.
func fileExists(p string) (bool, error) {
	root, base, err := fsutil.RootAt(p)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer func() { _ = root.Close() }()
	if _, statErr := root.Stat(base); statErr != nil {
		if errors.Is(statErr, os.ErrNotExist) {
			return false, nil
		}
		return false, statErr
	}
	return true, nil
}

// daemonStatus is what `status` reports and the support bundle records about
// the daemon: whether it holds the instance lock, and what its unauthenticated
// /__version said.
type daemonStatus struct {
	Running        bool   `json:"running"`
	PID            int    `json:"pid,omitempty"`
	Port           int    `json:"port,omitempty"`
	Version        string `json:"version,omitempty"`
	Commit         string `json:"commit,omitempty"`
	BuildTime      string `json:"buildTime,omitempty"`
	ReleaseTrain   string `json:"releaseTrain,omitempty"`
	UIBuildHash    string `json:"uiBuildHash,omitempty"`
	TLSFingerprint string `json:"tlsFingerprint,omitempty"`
	Error          string `json:"error,omitempty"`
}

// queryDaemon asks the install's daemon what it is. The instance lock is the
// authority on whether one is running; /__version is how the daemon proves it
// is serving. A held lock whose daemon does not answer is reported as running
// with the error, because the process exists and is not healthy.
func (l layout) queryDaemon(ctx context.Context) (daemonStatus, error) {
	info, held, err := l.probe()
	if err != nil {
		return daemonStatus{}, err
	}
	if !held {
		return daemonStatus{}, nil
	}
	status := daemonStatus{Running: true, PID: info.PID, Port: info.Port}
	if status.Port == 0 {
		status.Port = l.Port
	}
	l.askDaemon(ctx, &status)
	return status, nil
}

// askDaemon fills status from the daemon's /__version, or records why it
// could not.
func (l layout) askDaemon(ctx context.Context, status *daemonStatus) {
	build, err := l.fetchVersion(ctx, status.Port)
	if err != nil {
		status.Error = err.Error()
		return
	}
	status.Version = build.Version
	status.Commit = build.Commit
	status.BuildTime = build.BuildTime
	status.ReleaseTrain = build.ReleaseTrain
	status.UIBuildHash = build.UIBuildHash
	status.TLSFingerprint = build.TLSFingerprint
}

func (l layout) fetchVersion(ctx context.Context, port int) (api.BuildVersionResponse, error) {
	var build api.BuildVersionResponse
	tlsConfig, err := l.trustDaemonCert()
	if err != nil {
		return build, err
	}
	client := &http.Client{
		Timeout:   daemonTimeout,
		Transport: &http.Transport{TLSClientConfig: tlsConfig},
	}
	defer client.CloseIdleConnections()

	target := "https://" + net.JoinHostPort(loopbackHost, strconv.Itoa(port)) + "/__version"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, http.NoBody)
	if err != nil {
		return build, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return build, fmt.Errorf("the daemon did not answer on port %d: %w", port, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return build, fmt.Errorf("the daemon answered /__version with %s", resp.Status)
	}
	if decodeErr := json.NewDecoder(io.LimitReader(resp.Body, maxVersionBody)).Decode(&build); decodeErr != nil {
		return build, fmt.Errorf("the daemon's /__version was not readable: %w", decodeErr)
	}
	return build, nil
}

// trustDaemonCert trusts the daemon's own certificate, read from the path the
// daemon serves it from, so the request is verified rather than sent with
// verification off. The daemon's default certificate is self-signed.
func (l layout) trustDaemonCert() (*tls.Config, error) {
	root, base, err := fsutil.RootAt(l.CertFile)
	if err != nil {
		return nil, fmt.Errorf("read the daemon certificate %s: %w", l.CertFile, err)
	}
	defer func() { _ = root.Close() }()
	data, err := root.ReadFile(base)
	if err != nil {
		return nil, fmt.Errorf("read the daemon certificate %s: %w", l.CertFile, err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("%s holds no PEM certificate", l.CertFile)
	}
	leaf, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse the daemon certificate %s: %w", l.CertFile, err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(leaf)
	return &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: pool, ServerName: nameFor(leaf)}, nil
}

// nameFor picks the name to verify the loopback connection against: none when
// the certificate covers the loopback address, otherwise its first DNS name.
func nameFor(leaf *x509.Certificate) string {
	loopback := net.ParseIP(loopbackHost)
	for _, ip := range leaf.IPAddresses {
		if ip.Equal(loopback) {
			return ""
		}
	}
	if len(leaf.DNSNames) > 0 {
		return leaf.DNSNames[0]
	}
	return ""
}
