package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Populated by goreleaser during build
var version = "latest"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
