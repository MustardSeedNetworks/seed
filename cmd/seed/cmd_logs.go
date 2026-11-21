// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/seed/internal/fsutil"
	"github.com/MustardSeedNetworks/seed/internal/support"
)

const (
	defaultLogLines = 100
	// logTailWindow bounds how far from the end of the log a tail reads.
	logTailWindow = 4 << 20
	followPoll    = 500 * time.Millisecond
)

func initLogsCmd(state *cliState) {
	logsCmd := &cobra.Command{
		Use:   "logs",
		Short: "Show or follow the daemon's log file",
		Long: `Show the end of the log file seed serve writes, or follow it as it grows.

The log is the file named by logging.file in the config, or logs/seed.log
under the daemon's working directory when that is unset. Following survives
log rotation: when the file is replaced or truncated, logs reopens it.`,
		Example: `  # The last 100 lines
  seed logs

  # The last 500 lines
  seed logs --lines 500

  # Follow new lines until interrupted
  seed logs -f`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SilenceUsage = true
			return runLogs(cmd, state)
		},
	}
	logsCmd.Flags().BoolP("follow", "f", false, "keep running and print new lines as they are written")
	logsCmd.Flags().IntP("lines", "n", defaultLogLines, "number of lines to show from the end of the log")
	addWorkDirFlag(logsCmd)
	state.rootCmd.AddCommand(logsCmd)
}

func runLogs(cmd *cobra.Command, state *cliState) error {
	follow, err := cmd.Flags().GetBool("follow")
	if err != nil {
		return err
	}
	lines, err := cmd.Flags().GetInt("lines")
	if err != nil {
		return err
	}
	if lines < 0 {
		return fmt.Errorf("--lines must not be negative, got %d", lines)
	}
	l, err := resolveLayout(state, cmd)
	if err != nil {
		return err
	}

	root, base, err := fsutil.RootAt(l.LogFile)
	if err != nil {
		return noLogError(l.LogFile, err)
	}
	defer func() { _ = root.Close() }()
	file, err := root.Open(base)
	if err != nil {
		return noLogError(l.LogFile, err)
	}
	defer func() { _ = file.Close() }()

	out := cmd.OutOrStdout()
	tail, err := support.TailLines(file, lines, logTailWindow)
	if err != nil {
		return fmt.Errorf("read %s: %w", l.LogFile, err)
	}
	for _, line := range tail {
		if _, writeErr := fmt.Fprintln(out, line); writeErr != nil {
			return writeErr
		}
	}
	if !follow {
		return nil
	}

	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
	defer stop()
	return followLog(ctx, root, base, file, out, followPoll)
}

func noLogError(logFile string, err error) error {
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("no log file at %s: seed serve has not run here, or it runs in another directory "+
			"(pass --workdir)", logFile)
	}
	return fmt.Errorf("open %s: %w", logFile, err)
}

// followLog copies what is appended to file to w until ctx ends, reopening
// base under root when rotation replaces the file or truncates it. file must be
// positioned where following should begin.
func followLog(ctx context.Context, root *os.Root, base string, file *os.File, w io.Writer, every time.Duration) error {
	current := file
	reopened := false
	defer func() {
		if reopened {
			_ = current.Close()
		}
	}()
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
		if _, err := io.Copy(w, current); err != nil {
			return err
		}
		next, rotated, err := rotatedFile(root, base, current)
		if err != nil {
			return err
		}
		if !rotated {
			continue
		}
		if reopened {
			_ = current.Close()
		}
		current, reopened = next, true
	}
}

// rotatedFile returns a freshly opened base when it is no longer the file
// current reads from (rotation renamed or replaced it), or when it has shrunk
// below current's read position (truncation). rotated is false when nothing
// changed or the log is briefly absent mid-rotation.
func rotatedFile(root *os.Root, base string, current *os.File) (*os.File, bool, error) {
	onDisk, err := root.Stat(base)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	held, err := current.Stat()
	if err != nil {
		return nil, false, err
	}
	position, err := current.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, false, err
	}
	if os.SameFile(onDisk, held) && onDisk.Size() >= position {
		return nil, false, nil
	}
	next, err := root.Open(base)
	if err != nil {
		return nil, false, err
	}
	return next, true, nil
}
