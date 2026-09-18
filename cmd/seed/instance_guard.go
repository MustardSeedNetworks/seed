// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/MustardSeedNetworks/foundation/pkg/instance"

	"github.com/MustardSeedNetworks/seed/internal/paths"
)

// exitDaemonRunning is the exit status of a command that refused to write
// state a running seed serve owns. It is distinct from the generic 1 so a
// script can tell "the daemon has it" from "the command failed".
const exitDaemonRunning = 2

// daemonRunningError reports that seed serve holds the instance lock on the
// directory this command would have written to.
type daemonRunningError struct {
	dir     string
	info    instance.Info
	instead string
}

func (e *daemonRunningError) Error() string {
	where := fmt.Sprintf("seed serve is running on %s", e.dir)
	if e.info.PID != 0 {
		where += fmt.Sprintf(" (pid %d", e.info.PID)
		if e.info.Port != 0 {
			where += fmt.Sprintf(", port %d", e.info.Port)
		}
		where += ")"
	} else if e.info.Port != 0 {
		where += fmt.Sprintf(" (port %d)", e.info.Port)
	}
	return where + ". " + e.instead
}

// exitCodeFor maps an error returned by the root command to a process exit
// status. Everything but a refused write is the historical 1.
func exitCodeFor(err error) int {
	var held *daemonRunningError
	if errors.As(err, &held) {
		return exitDaemonRunning
	}
	return 1
}

// lockDir is the directory seed serve and the CLI both take the instance lock
// on. It is the directory holding the resolved config file, because that is
// the one path every one of these commands resolves identically and the one
// `--config` / SEED_CONFIG_PATH moves: two installs pointed at two configs are
// then correctly independent.
func lockDir(configPath string) string {
	return filepath.Dir(configPath)
}

// refuseIfDaemonOwns returns a *daemonRunningError when seed serve holds the
// lock on the config's directory. `instead` says what to do about it.
//
// A probe that cannot be read at all fails open with a warning on stderr —
// never stdout, which carries `credentials --json`. The case is a root
// daemon's 0600 lock record under /etc/seed read by an operator who is not
// root; a directory with no lock file in it is simply free, because Probe
// opens an existing record rather than creating one. The guard is a courtesy
// that keeps a CLI from writing under a live daemon; the durability guarantee
// is Config.Save's atomic rename, not this.
func refuseIfDaemonOwns(configPath, instead string) error {
	dir := lockDir(configPath)
	info, held, err := instance.Probe(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not check whether seed serve is running on %s: %v\n", dir, err)
		return nil
	}
	if !held {
		return nil
	}
	return &daemonRunningError{dir: dir, info: info, instead: instead}
}

// guardConfigCommand is the PreRunE for a command that resolves its config the
// standard way.
func guardConfigCommand(state *cliState, instead string) error {
	return refuseIfDaemonOwns(paths.ResolveConfigPath(state.cfgFile, paths.ModeAuto), instead)
}

// guardLicenseCommand is the PreRunE for the license verbs that write
// activation state. The license manager resolves its own directory from the
// user's home and honors no --config flag, so the lock dir is resolved without
// one rather than pretending the flag reaches it.
func guardLicenseCommand() error {
	return refuseIfDaemonOwns(paths.ResolveConfigPath("", paths.ModeAuto),
		"Stop seed serve before changing the license: the running daemon holds its activation state in memory and would not see the change.")
}
