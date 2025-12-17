package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print The Seed version information.`,
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Printf("The Seed %s\n", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
