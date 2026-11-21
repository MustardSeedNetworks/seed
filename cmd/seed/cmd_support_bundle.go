// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/seed/internal/fsutil"
	"github.com/MustardSeedNetworks/seed/internal/license"
	"github.com/MustardSeedNetworks/seed/internal/support"
	"github.com/MustardSeedNetworks/seed/internal/version"
)

const (
	// bundleLogBytes is how much of the log a bundle carries. A bundle is
	// mailed, and the last few megabytes hold the failure.
	bundleLogBytes = 4 << 20
	// archiveFileMode is the permission an output archive is created with:
	// the operator only, because a bundle describes the host and a backup
	// holds its keys.
	archiveFileMode = 0o600
	keyringEnv      = "SEED_CREDENTIAL_KEY"
)

func initSupportBundleCmd(state *cliState) {
	bundleCmd := &cobra.Command{
		Use:   "support-bundle",
		Short: "Collect redacted diagnostics for support",
		Long: `Collect one archive of diagnostics: the daemon's /__version, the config, the end
of the log, OS information and the platform report.

Credentials are removed before anything is written: every config value under a
key that names a credential (password, token, secret, community, API key,
license, hash) is replaced, and the log tail is scrubbed of those same values
plus anything shaped like a bearer token, JWT, key=value credential or PEM
private key. TLS private keys, the credential keyring and the database are
never read.

Read the bundle before sending it. It is redacted, not anonymised: addresses,
interface names and device names are still in it.`,
		Example: `  # Write seed-support-<timestamp>.tar.gz in the current directory
  seed support-bundle

  # Choose the output file
  seed support-bundle -o /tmp/seed-support.tar.gz`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SilenceUsage = true
			return runSupportBundle(cmd, state)
		},
	}
	bundleCmd.Flags().StringP("output", "o", "", "output archive (default: seed-support-<timestamp>.tar.gz)")
	addWorkDirFlag(bundleCmd)
	state.rootCmd.AddCommand(bundleCmd)
}

func runSupportBundle(cmd *cobra.Command, state *cliState) error {
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}
	l, err := resolveLayout(state, cmd)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if output == "" {
		output = "seed-support-" + now.Format("20060102T150405Z") + ".tar.gz"
	}

	files, err := collectBundle(cmd, l, now)
	if err != nil {
		return err
	}
	if writeErr := writeArchive(output, func(w io.Writer) error {
		return support.WriteBundle(w, files, now)
	}); writeErr != nil {
		return writeErr
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(),
		"Support bundle written to %s (%d files). Read it before you send it: it is redacted, not anonymised.\n",
		output, len(files))
	return err
}

// bundleManifest is the bundle's first file: what ran, on what, when, and
// where this install keeps its files.
type bundleManifest struct {
	CollectedAt string            `json:"collectedAt"`
	Build       version.BuildInfo `json:"build"`
	OS          string            `json:"os"`
	Arch        string            `json:"arch"`
	CPUs        int               `json:"cpus"`
	GoVersion   string            `json:"goVersion"`
	OSRelease   string            `json:"osRelease,omitempty"`
	ConfigPath  string            `json:"configPath"`
	WorkDir     string            `json:"workDir"`
	LogFile     string            `json:"logFile"`
	ConfigFound bool              `json:"configFound"`
	LogFound    bool              `json:"logFound"`
}

func collectBundle(cmd *cobra.Command, l layout, now time.Time) ([]support.File, error) {
	manifest := bundleManifest{
		CollectedAt: now.Format(time.RFC3339),
		Build:       version.Info(),
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		CPUs:        runtime.NumCPU(),
		GoVersion:   runtime.Version(),
		OSRelease:   osRelease(),
		ConfigPath:  l.ConfigPath,
		WorkDir:     l.WorkDir,
		LogFile:     l.LogFile,
	}
	secrets := ambientSecrets()

	var files []support.File
	configDoc, found, err := redactedConfig(l.ConfigPath)
	if err != nil {
		return nil, err
	}
	manifest.ConfigFound = found
	if found {
		files = append(files, support.File{Name: "config.json", Data: configDoc.data})
		secrets = append(secrets, configDoc.secrets...)
	}

	daemon, err := l.queryDaemon(cmd.Context())
	if err != nil {
		daemon = daemonStatus{Error: err.Error()}
	}
	daemonDoc, err := json.MarshalIndent(daemon, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("render daemon status: %w", err)
	}
	files = append(files, support.File{Name: "daemon.json", Data: append(daemonDoc, '\n')})

	var platform bytes.Buffer
	runPlatform(&platform)
	files = append(files, support.File{Name: "platform.txt", Data: platform.Bytes()})

	logTail, logFound, err := readLogTail(l.LogFile)
	if err != nil {
		return nil, err
	}
	manifest.LogFound = logFound
	if logFound {
		files = append(files, support.File{
			Name: "logs/seed.log",
			Data: []byte(support.ScrubText(string(logTail), secrets)),
		})
	}

	manifestDoc, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("render manifest: %w", err)
	}
	return append([]support.File{{Name: "manifest.json", Data: append(manifestDoc, '\n')}}, files...), nil
}

type redactedDoc struct {
	data    []byte
	secrets []string
}

// redactedConfig reads the config file as a document and redacts it. A missing
// file is reported as not found rather than failing the bundle: a host where
// seed never ran is exactly the host support gets asked about.
func redactedConfig(configPath string) (redactedDoc, bool, error) {
	root, base, err := fsutil.RootAt(configPath)
	if errors.Is(err, os.ErrNotExist) {
		return redactedDoc{}, false, nil
	}
	if err != nil {
		return redactedDoc{}, false, fmt.Errorf("open %s: %w", configPath, err)
	}
	defer func() { _ = root.Close() }()
	raw, err := root.ReadFile(base)
	if errors.Is(err, os.ErrNotExist) {
		return redactedDoc{}, false, nil
	}
	if err != nil {
		return redactedDoc{}, false, fmt.Errorf("read %s: %w", configPath, err)
	}
	data, secrets, err := support.RedactJSON(raw)
	if err != nil {
		return redactedDoc{}, false, fmt.Errorf("%s: %w", configPath, err)
	}
	return redactedDoc{data: data, secrets: secrets}, true, nil
}

func readLogTail(logFile string) ([]byte, bool, error) {
	root, base, err := fsutil.RootAt(logFile)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("open %s: %w", logFile, err)
	}
	defer func() { _ = root.Close() }()
	file, err := root.Open(base)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("open %s: %w", logFile, err)
	}
	defer func() { _ = file.Close() }()
	data, err := support.TailBytes(file, bundleLogBytes)
	if err != nil {
		return nil, false, fmt.Errorf("read %s: %w", logFile, err)
	}
	return data, true, nil
}

// ambientSecrets are credentials that live outside the config file but can
// still show up in a log line: the credential-key environment override, and
// the activated license key. The license is best effort: a host that never
// activated one has none to hide.
func ambientSecrets() []string {
	var secrets []string
	if value := os.Getenv(keyringEnv); value != "" {
		secrets = append(secrets, value)
	}
	if mgr, err := license.NewManager(); err == nil {
		if activation := mgr.GetState(); activation != nil && activation.LicenseKey != "" {
			secrets = append(secrets, activation.LicenseKey)
		}
	}
	return secrets
}

// osRelease names the Linux distribution, the first thing support asks for.
func osRelease() string {
	if runtime.GOOS != "linux" {
		return ""
	}
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return ""
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		if value, ok := strings.CutPrefix(line, "PRETTY_NAME="); ok {
			return strings.Trim(value, `"`)
		}
	}
	return ""
}

// writeArchive creates output exclusively, with owner-only permission, and
// removes it again if the write fails: an operator must never mail a
// truncated bundle believing it complete.
func writeArchive(output string, write func(io.Writer) error) error {
	root, base, err := fsutil.RootAt(output)
	if err != nil {
		return fmt.Errorf("create %s: %w", output, err)
	}
	defer func() { _ = root.Close() }()
	file, err := root.OpenFile(base, os.O_WRONLY|os.O_CREATE|os.O_EXCL, archiveFileMode)
	if err != nil {
		return fmt.Errorf("create %s: %w", output, err)
	}
	writeErr := write(file)
	closeErr := file.Close()
	if failed := errors.Join(writeErr, closeErr); failed != nil {
		_ = root.Remove(base)
		return fmt.Errorf("write %s: %w", output, failed)
	}
	return nil
}
