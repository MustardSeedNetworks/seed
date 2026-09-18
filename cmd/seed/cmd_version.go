package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/MustardSeedNetworks/seed/internal/version"
)

func initVersionCmd(state *cliState) {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  `Print Seed version information.`,
		Example: `  # Show the running version
  seed version`,
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Fprintf(os.Stdout, "Seed %s\n", version.GetVersion())
		},
	}
	state.rootCmd.AddCommand(versionCmd)
}
