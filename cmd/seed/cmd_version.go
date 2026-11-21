package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/seed/internal/version"
)

func initVersionCmd(state *cliState) {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long: `Print Seed version information.

With --json, print the build metadata compiled into this binary under the same
keys the daemon serves at /__version (version, commit, buildTime, releaseTrain,
uiBuildHash), so a deployment check reads the binary on disk and the running
daemon the same way.`,
		Example: `  # Show the running version
  seed version

  # For scripts
  seed version --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			asJSON, err := cmd.Flags().GetBool("json")
			if err != nil {
				return err
			}
			if asJSON {
				return writeJSON(cmd.OutOrStdout(), version.Info())
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Seed %s\n", version.GetVersion())
			return err
		},
	}
	versionCmd.Flags().Bool("json", false, "print the build metadata as JSON")
	state.rootCmd.AddCommand(versionCmd)
}
