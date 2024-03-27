package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Populated by goreleaser during build
var version = "latest"

func NewVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version)
		},
	}
	return cmd
}
