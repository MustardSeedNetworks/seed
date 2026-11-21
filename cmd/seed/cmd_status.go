// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/spf13/cobra"
)

func initStatusCmd(state *cliState) {
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Report whether the seed daemon is running",
		Long: `Report whether seed serve is running on this install and whether it answers.

The daemon holds a single-instance lock on the config directory; status reads
that lock for the process and port, then asks the daemon's unauthenticated
/__version for its build, so a hung daemon is told apart from a stopped one.

Exit status:
  0  running and answering
  1  could not tell, or running but not answering
  2  not running`,
		Example: `  # Is it up?
  seed status

  # Machine-readable, for a monitor or deploy script
  seed status --json

  # Branch on the exit status
  seed status >/dev/null || echo "seed is not healthy"`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SilenceUsage = true
			return runStatus(cmd, state)
		},
	}
	statusCmd.Flags().Bool("json", false, "print the status as JSON")
	addWorkDirFlag(statusCmd)
	state.rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, state *cliState) error {
	asJSON, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}
	l, err := resolveLayout(state, cmd)
	if err != nil {
		return err
	}
	status, err := l.queryDaemon(cmd.Context())
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	var printErr error
	if asJSON {
		printErr = writeJSON(out, status)
	} else {
		printErr = printStatus(out, status, l)
	}
	if printErr != nil {
		return printErr
	}
	switch {
	case !status.Running:
		return reportedExit(exitNotRunning)
	case status.Error != "":
		return withExitCode(exitFailure, errors.New(status.Error))
	default:
		return nil
	}
}

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func printStatus(w io.Writer, status daemonStatus, l layout) error {
	if !status.Running {
		_, err := fmt.Fprintf(w, "Status: NOT RUNNING\nNo seed serve holds %s\n", l.lockDir())
		return err
	}
	state := "RUNNING"
	if status.Error != "" {
		state = "NOT ANSWERING"
	}
	if _, err := fmt.Fprintf(w, "Status: %s\n", state); err != nil {
		return err
	}
	rows := []struct{ label, value string }{
		{"PID", itoaIfSet(status.PID)},
		{"Port", itoaIfSet(status.Port)},
		{"Version", status.Version},
		{"Commit", status.Commit},
		{"Built", status.BuildTime},
		{"UI build", status.UIBuildHash},
		{"TLS fingerprint", status.TLSFingerprint},
	}
	for _, row := range rows {
		if row.value == "" {
			continue
		}
		if _, err := fmt.Fprintf(w, "%s: %s\n", row.label, row.value); err != nil {
			return err
		}
	}
	return nil
}

func itoaIfSet(n int) string {
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}
