package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/krisarmstrong/seed/internal/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print The Seed version information.`,
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Printf("The Seed %s\n", version.Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
