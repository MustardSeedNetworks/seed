package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/krisarmstrong/seed/internal/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print The Seed version information.`,
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Fprintf(os.Stdout, "The Seed %s\n", version.GetVersion())
	},
}

func initVersionCmd() {
	rootCmd.AddCommand(versionCmd)
}
