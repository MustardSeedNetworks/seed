// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/MustardSeedNetworks/foundation/pkg/instance"

	"github.com/MustardSeedNetworks/seed/internal/paths"
)

// What to do instead of each refused command. Where the daemon exposes the
// same operation over HTTP the message names that route; where it does not —
// resetting the config, and the license, which has only a read endpoint — it
// says to stop the daemon rather than name a route that does not exist.
const (
	insteadOfResetConfig = "Stop seed serve before resetting its config, " +
		"or change settings through the running daemon at PUT /api/v1/settings."
	insteadOfCredentials = "Ask the running daemon instead: GET /api/v1/setup/status."
	insteadOfSetupWizard = "Complete setup through the running daemon instead: POST /api/v1/setup/complete."
	insteadOfLicense     = "Stop seed serve before changing the license: the running daemon " +
		"holds its activation state in memory and would not see the change."
)

// exitRefused is the exit status of a command that refused to touch state
// it must not: a running seed serve owns it, or the Windows system layout needs
// an elevated shell. It is distinct from the generic 1 so a script can tell
// "refused" from "the command failed".
const exitRefused = 2

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

// elevationRequiredError reports that the config this command resolved is the
// Windows service's, which only SYSTEM and Administrators can open.
type elevationRequiredError struct {
	configPath string
}

func (e *elevationRequiredError) Error() string {
	return fmt.Sprintf("%s belongs to the Seed service and needs an elevated shell: "+
		"run this command from PowerShell started with Run as administrator", e.configPath)
}

// checkElevation refuses configPath when it is the system layout's config and
// the process is not elevated. It never falls back to the user profile: that
// would configure something the service does not read.
func checkElevation(configPath, systemConfig string, elevated bool) error {
	if elevated || !sameConfigPath(configPath, systemConfig) {
		return nil
	}
	return &elevationRequiredError{configPath: configPath}
}

// untrustedDataDirError reports that the Windows system tree failed
// verification, so an elevated command must not read or write it.
type untrustedDataDirError struct {
	err error
}

func (e *untrustedDataDirError) Error() string {
	return fmt.Sprintf("refusing to use the Seed service data: %v. "+
		"Reinstall with install.ps1, which creates it with the right permissions", e.err)
}

func (e *untrustedDataDirError) Unwrap() error { return e.err }

func sameConfigPath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// checkSystemTree runs [verifyDataDir] over the system tree when configPath is
// its config, so an elevated command never works inside a directory someone
// else could have prepared. Any other config path is the caller's own.
func checkSystemTree(configPath string, system *paths.Paths) error {
	if !sameConfigPath(configPath, system.ConfigFile) {
		return nil
	}
	if err := verifyDataDir(system.Root, system.ConfigFile); err != nil {
		return &untrustedDataDirError{err: err}
	}
	return nil
}

// requireElevation is checkElevation, then checkSystemTree, against this machine.
func requireElevation(configPath string) error {
	system := paths.Resolve(paths.ModeSystem)
	if err := checkElevation(configPath, system.ConfigFile, processElevated()); err != nil {
		return err
	}
	return checkSystemTree(configPath, system)
}

// exitCodeFor maps an error returned by the root command to a process exit
// status: the status a command tagged its error with, 2 for a refused write,
// and the historical 1 for everything else.
func exitCodeFor(err error) int {
	if code, ok := codeOf(err); ok {
		return code
	}
	if _, ok := errors.AsType[*daemonRunningError](err); ok {
		return exitRefused
	}
	if _, ok := errors.AsType[*elevationRequiredError](err); ok {
		return exitRefused
	}
	if _, ok := errors.AsType[*untrustedDataDirError](err); ok {
		return exitRefused
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
	configPath := paths.ResolveConfigPath(state.cfgFile, paths.ModeAuto)
	if err := requireElevation(configPath); err != nil {
		return err
	}
	return refuseIfDaemonOwns(configPath, instead)
}

// guardConfigRead is the PreRunE for a command that only reads the resolved
// config, so a non-elevated shell gets the elevation message rather than an
// access-denied error from the ACL.
func guardConfigRead(state *cliState) error {
	return requireElevation(paths.ResolveConfigPath(state.cfgFile, paths.ModeAuto))
}

// guardLicenseCommand is the PreRunE for the license verbs that write
// activation state. The license manager resolves its own directory from the
// user's home and honors no --config flag, so the lock dir is resolved without
// one rather than pretending the flag reaches it.
func guardLicenseCommand() error {
	configPath := paths.ResolveConfigPath("", paths.ModeAuto)
	if err := requireElevation(configPath); err != nil {
		return err
	}
	return refuseIfDaemonOwns(configPath, insteadOfLicense)
}
