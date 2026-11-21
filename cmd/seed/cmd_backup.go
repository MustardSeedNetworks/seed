// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/seed/internal/fsutil"
	"github.com/MustardSeedNetworks/seed/internal/support"
	"github.com/MustardSeedNetworks/seed/internal/version"
)

const (
	insteadOfBackup = "Stop seed serve and run backup again: a database copied while the daemon is " +
		"writing to it is not a valid backup."
	insteadOfRestore = "Stop seed serve and run restore again: restoring under a live daemon " +
		"replaces files it holds open."
	archiveSuffix = ".tar.gz"
)

// errNothingToBackUp marks an install with no files yet.
var errNothingToBackUp = errors.New("nothing to back up")

func initBackupCmd(state *cliState) {
	backupCmd := &cobra.Command{
		Use:   "backup",
		Short: "Archive the install's config, database and certificates",
		Long: `Archive everything needed to rebuild this install: the config file, the
credential keyring, the database, and the TLS certificate and key.

The archive holds the install's secrets (the JWT secret in the config, the
keyring that decrypts stored SNMP credentials, the TLS private key). Keep it
as private as the install itself; it is created readable by you alone.

The license activation is not included: it is bound to this device and lives
in your home directory, so re-activate it on a rebuilt host.

backup refuses to run while seed serve is running, because a database file
copied under a writing daemon is not a valid backup.`,
		Example: `  # Write seed-backup-<timestamp>.tar.gz in the current directory
  seed backup

  # Choose the output file
  seed backup -o /var/backups/seed.tar.gz`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SilenceUsage = true
			return runBackup(cmd, state)
		},
	}
	backupCmd.Flags().StringP("output", "o", "", "output archive (default: seed-backup-<timestamp>.tar.gz)")
	addWorkDirFlag(backupCmd)
	state.rootCmd.AddCommand(backupCmd)
}

func runBackup(cmd *cobra.Command, state *cliState) error {
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}
	l, err := resolveLayout(state, cmd)
	if err != nil {
		return err
	}
	if refuseErr := l.refuseWhileRunning(insteadOfBackup); refuseErr != nil {
		return refuseErr
	}
	now := time.Now().UTC()
	if output == "" {
		output = "seed-backup-" + now.Format("20060102T150405Z") + archiveSuffix
	}
	count, err := writeBackup(l, output, now)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Backed up %d files to %s\n", count, output)
	return err
}

// writeBackup archives the install's existing files to output.
func writeBackup(l layout, output string, now time.Time) (int, error) {
	members, err := l.stateMembers()
	if err != nil {
		return 0, err
	}
	if len(members) == 0 {
		return 0, fmt.Errorf("%w: no config, database or certificate under %s or %s",
			errNothingToBackUp, filepath.Dir(l.ConfigPath), l.WorkDir)
	}
	writeErr := writeArchive(output, func(w io.Writer) error {
		return support.Backup(w, members, version.GetVersion(), now)
	})
	return len(members), writeErr
}

func initRestoreCmd(state *cliState) {
	restoreCmd := &cobra.Command{
		Use:   "restore <archive>",
		Short: "Restore the install from a seed backup",
		Long: `Restore the config, credential keyring, database and certificates from an
archive written by 'seed backup'.

restore refuses to run while seed serve is running, and refuses an archive
that is not a seed backup. The whole archive is read and checked before any
file is touched: an entry that is not one of the backup's known files, a link,
a name that climbs out of the install, a duplicate or a truncated archive is
refused and leaves the install exactly as it was.

Before replacing anything, restore writes a pre-restore backup of what is
there now, beside the archive, as <archive>.pre-restore-<timestamp>.tar.gz.
Restoring that archive undoes the restore.

Files are written with owner-only permission by the user running restore. On a
system install run it as the service user (for example
'sudo -u seed seed restore ...') so the daemon can read what it restores.`,
		Example: `  # Restore, with the daemon stopped
  seed restore seed-backup-20261121T120000Z.tar.gz

  # Put back what was there before the restore
  seed restore seed-backup-20261121T120000Z.pre-restore-20261121T121500Z.tar.gz`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return runRestore(cmd, state, args[0])
		},
	}
	addWorkDirFlag(restoreCmd)
	state.rootCmd.AddCommand(restoreCmd)
}

func runRestore(cmd *cobra.Command, state *cliState, archive string) error {
	l, err := resolveLayout(state, cmd)
	if err != nil {
		return err
	}
	if refuseErr := l.refuseWhileRunning(insteadOfRestore); refuseErr != nil {
		return refuseErr
	}

	root, base, err := fsutil.RootAt(archive)
	if err != nil {
		return fmt.Errorf("open %s: %w", archive, err)
	}
	defer func() { _ = root.Close() }()
	file, err := root.Open(base)
	if err != nil {
		return fmt.Errorf("open %s: %w", archive, err)
	}
	defer func() { _ = file.Close() }()

	out := cmd.OutOrStdout()
	preRestore := ""
	keepCopy := func() error {
		now := time.Now().UTC()
		preRestore = strings.TrimSuffix(archive, archiveSuffix) + ".pre-restore-" +
			now.Format("20060102T150405Z") + archiveSuffix
		_, copyErr := writeBackup(l, preRestore, now)
		if errors.Is(copyErr, errNothingToBackUp) {
			// A fresh host has nothing to keep; that is not a reason to refuse.
			preRestore = ""
			return nil
		}
		return copyErr
	}
	restored, err := support.Restore(file, l.restoreTargets(), keepCopy)
	if err != nil {
		if errors.Is(err, support.ErrArchive) || errors.Is(err, support.ErrUnsafeTarget) {
			return fmt.Errorf("%w; nothing was changed", err)
		}
		return err
	}
	if preRestore != "" {
		if _, printErr := fmt.Fprintf(out, "Kept the previous state in %s\n", preRestore); printErr != nil {
			return printErr
		}
	}
	_, err = fmt.Fprintf(out, "Restored %d files from %s\n", len(restored), archive)
	return err
}
